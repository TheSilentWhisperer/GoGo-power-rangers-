import gen.proto.remote_trainer_pb2 as remote_trainer_pb2
import gen.proto.remote_trainer_pb2_grpc as remote_trainer_pb2_grpc
from google.protobuf import empty_pb2
import torch 
import queue
import random
import torch.nn.functional as F
from copy import deepcopy
import os
import sys
from tqdm import tqdm
import time
import traceback
from services.logger import get_logger

# GPU Performance Optimizations
torch.backends.cudnn.benchmark = True  # Auto-tune cuDNN for best performance
torch.backends.cudnn.deterministic = False  # Faster but non-deterministic
# Disable JIT profiling to prevent hanging/deadlocks
torch._C._jit_set_profiling_mode(False)
torch._C._jit_set_profiling_executor(False)
os.environ['PYTORCH_CUDA_ALLOC_CONF'] = 'expandable_segments:True'
os.environ['CUDA_LAUNCH_BLOCKING'] = '0'  # Enable async GPU operations

# Make stderr non-blocking so tqdm doesn't freeze on screensaver
# This prevents terminal blocking when display is locked
try:
    import fcntl
    flags = fcntl.fcntl(sys.stderr.fileno(), fcntl.F_GETFL)
    fcntl.fcntl(sys.stderr.fileno(), fcntl.F_SETFL, flags | os.O_NONBLOCK)
except Exception:
    pass  # If non-blocking fails, fall back to normal behavior


class ReplayBuffer:
    def __init__(self, capacity):
        self.capacity = capacity
        self.buffer = []
        self.position = 0
        self.save_path = "saves/replay_buffer.pt"

    def push(self, experience):
        # Aggressively quantize experience to save memory:
        # - Boards, player colors, pass counts, and value are stored as small integer dtypes (int8 / uint8)
        # - Legal masks are stored as uint8
        # - Policy distributions are stored as float16
        try:
            inp = experience.get("input", {})
            if isinstance(inp, dict):
                for k, v in list(inp.items()):
                    try:
                        if hasattr(v, "cpu"):
                            t = v.cpu()
                            # Boards are expected to contain small integers (-1,0,1): store as int8
                            if k == "board" or k == "boards":
                                if t.is_floating_point():
                                    # round to nearest integer then clamp to int8 range
                                    tq = torch.clamp(t.round(), -128, 127).to(dtype=torch.int8)
                                else:
                                    tq = t.to(dtype=torch.int8)
                                inp[k] = tq
                                continue

                            # legal_mask is binary: store as uint8
                            if "legal_mask" in k or k == "legal_masks":
                                try:
                                    inp[k] = t.to(dtype=torch.uint8)
                                except Exception:
                                    inp[k] = t
                                continue

                            # player colors and pass counts are small integers: store as int8
                            if k in ("player_color", "player_colors", "pass_count", "pass_counts"):
                                if t.is_floating_point():
                                    tq = torch.clamp(t.round(), -128, 127).to(dtype=torch.int8)
                                else:
                                    tq = t.to(dtype=torch.int8)
                                inp[k] = tq
                                continue

                            # Fallback: keep on CPU but don't keep unnecessary precision
                            if t.dtype in (torch.float32, torch.float64):
                                inp[k] = t.to(dtype=torch.float16)
                            else:
                                inp[k] = t
                    except Exception:
                        pass

            # sanitize target tensors
            tgt = experience.get("target", {})
            if isinstance(tgt, dict):
                # policy tensors can be large: store as CPU float16
                p = tgt.get("policy")
                if hasattr(p, "cpu"):
                    try:
                        p_cpu = p.cpu().to(dtype=torch.float16)
                        tgt["policy"] = p_cpu
                    except Exception:
                        pass

                # value is in {-1,0,1}: store as Python int (small) or int8 tensor
                val = tgt.get("value")
                try:
                    if hasattr(val, "item"):
                        vnum = int(round(val.item()))
                    else:
                        vnum = int(round(float(val)))
                    # store as small python int to keep serialization compact
                    tgt["value"] = vnum
                except Exception:
                    pass
        except Exception:
            # fall back to storing the original experience if sanitization fails
            pass

        if len(self.buffer) < self.capacity:
            self.buffer.append(experience)
        else:
            self.buffer[self.position] = experience
        self.position = (self.position + 1) % self.capacity

    def sample(self, batch_size):
        # Use random.choices for O(k) sampling instead of random.sample O(n)
        # This is much faster for large buffers with replacement (fine for training)
        return random.choices(self.buffer, k=batch_size)

    def save(self, save_path=None):
        if save_path is None:
            save_path = self.save_path
        os.makedirs(os.path.dirname(save_path), exist_ok=True)
        # Save atomically to avoid creating a truncated file if the process is killed.
        tmp_path = save_path + ".tmp"
        torch.save((self.buffer, self.position), tmp_path)
        os.replace(tmp_path, save_path)

    def load(self):
        if os.path.exists(self.save_path):
            # Force loading to CPU to avoid GPU memory usage and reduce surprises
            try:
                self.buffer, self.position = torch.load(self.save_path, map_location="cpu")
            except Exception:
                # If load fails, don't raise — reset buffer and let training continue
                self.buffer, self.position = [], 0
        else:
            self.buffer, self.position = [], 0

    def __len__(self):
        return len(self.buffer)

class TrainingClient(remote_trainer_pb2_grpc.NetTrainerServicer):
    def __init__(self, inference_client):
        super().__init__()

        # Hyperparameters tuned for the larger model (default ~1.5M params)
        self.replay_buffer_capacity = 800_000
        self.min_replay_size = 32_768  # Start training after accumulating this many new experiences
        self.replay_ratio = 12
        self.batch_size = 512  # Increased from 256 with GPU optimizations
        self.base_lr = 3e-4
        self.weight_decay = 1e-4
        self.model_save_path = "saves/gogo81.pt"
        self.use_gradient_accumulation = False  # Set to True to increase effective batch size

        self.new_experience_count = 0
        self.total_experiences_played = 0  # Track total games across all training sessions
        self.trained_times = 0
        self.total_training_steps = 0  # Track total steps for LR schedule
        self.last_save_buffer_size = 0  # Track buffer size at last save for 10% checkpoints
        self.experience_pbar = None  # Progress bar for experience accumulation

        # Optimizer with weight decay; enable mixed precision when possible
        self.device = inference_client.device
        print(f"[TRAINING] Using device: {self.device}")
        self.optimizer = torch.optim.Adam(
            inference_client.model.parameters(), lr=self.base_lr, weight_decay=self.weight_decay
        )
        self.scaler = torch.amp.GradScaler('cuda') if self.device.type == 'cuda' else None

        self.inference_client = inference_client
        self.replay_buffer = ReplayBuffer(capacity=self.replay_buffer_capacity)

        self.replay_buffer.load()  # Load replay buffer from disk if it exists
        if os.path.exists(self.model_save_path):
            self.inference_client.model.load_state_dict(torch.load(self.model_save_path, map_location=self.inference_client.device))  # Load model weights from disk if they exist
        else:
            pass

        # Create progress bar for experience accumulation
        self.experience_pbar = tqdm(
            total=self.min_replay_size,
            desc="Accumulating experiences",
            unit="exp",
            file=sys.stderr,
            disable=False,
            position=0,
            leave=True
        )

    def _update_learning_rate(self):
        """Update optimizer learning rate with warmup and cosine decay schedule."""
        import math
        warmup_steps = 100
        total_steps = 100000  # Total steps for cosine decay
        min_lr = self.base_lr * 0.01  # Minimum 1% of base lr to prevent stalling
        
        if self.total_training_steps < warmup_steps:
            # Linear warmup
            lr = self.base_lr * (self.total_training_steps / warmup_steps)
        else:
            # Cosine decay after warmup
            progress = (self.total_training_steps - warmup_steps) / (total_steps - warmup_steps)
            progress = min(progress, 1.0)  # Clamp to [0, 1]
            lr = self.base_lr * 0.5 * (1 + math.cos(math.pi * progress))
            lr = max(lr, min_lr)  # Ensure LR doesn't drop below minimum
        
        for param_group in self.optimizer.param_groups:
            param_group['lr'] = lr

    def push_experiences(self, request):
        for sample in request.data:
            input = self.inference_client.get_input(sample.inputs)
            # `request.value` is the game outcome from Black's perspective
            # (1 == Black wins, -1 == White wins).
            # We need to convert it to the current position's player perspective
            black_to_play = sample.inputs.black_to_play  # 1 if Black to move, -1 if White to move
            value = request.value
            # If it's White's turn, flip the value from Black's perspective to White's perspective
            if black_to_play == -1:
                value = -value
            policy = torch.tensor(sample.policy_target, device=self.inference_client.device, dtype=torch.float32)  # Shape: (height*width + 2,)
            policy_sum = torch.sum(policy)
            if policy_sum > 0:
                policy /= policy_sum  # Normalize policy to get probabilities
            target = {
                "value": value,
                "policy": policy
            }
            base_input = deepcopy(input)
            base_target = deepcopy(target)
            #now augment the experience with symmetries of the board
            for k in range(4):  # Rotate 4 times (90 degrees each)
                rotated_input = deepcopy(base_input) # Create a deep copy to avoid modifying the original input
                rotated_target = deepcopy(base_target)  # Create a deep copy to avoid modifying the original target

                # Defensive check: ensure deepcopy didn't keep linked sub-objects
                def _find_shared(a, b, path=None):
                    if path is None:
                        path = []
                    shared = []
                    # Different types -> cannot compare structure
                    if type(a) is not type(b):
                        return shared
                    # Torch tensor: check underlying storage pointer
                    if isinstance(a, torch.Tensor):
                        try:
                            if a.data_ptr() == b.data_ptr():
                                shared.append(list(path))
                        except Exception:
                            if id(a) == id(b):
                                shared.append(list(path))
                        return shared
                    # dict: recurse keys present in both
                    if isinstance(a, dict):
                        for key in a:
                            if key in b:
                                shared.extend(_find_shared(a[key], b[key], path + [key]))
                        return shared
                    # list/tuple: recurse element-wise
                    if isinstance(a, (list, tuple)):
                        for i, (ai, bi) in enumerate(zip(a, b)):
                            shared.extend(_find_shared(ai, bi, path + [i]))
                        return shared
                    # Fallback for general objects: compare identity only for mutable/non-scalar types
                    try:
                        # ignore common immutable scalar types which may share interned ids (int, float, str, bool, bytes)
                        if not isinstance(a, (int, float, str, bool, bytes)):
                            if id(a) == id(b):
                                shared.append(list(path))
                    except Exception:
                        pass
                    return shared

                def _set_by_path(obj, path, newval):
                    cur = obj
                    for p in path[:-1]:
                        cur = cur[p]
                    last = path[-1]
                    cur[last] = newval

                shared_paths = _find_shared(base_input, rotated_input)
                if shared_paths:
                    for path in shared_paths:
                        # navigate to the item in rotated_input
                        cur = rotated_input
                        try:
                            for p in path[:-1]:
                                cur = cur[p]
                            key = path[-1]
                            item = cur[key]
                            if isinstance(item, torch.Tensor):
                                cur[key] = item.clone()
                            else:
                                # If not tensor, try to deepcopy that sub-object
                                cur[key] = deepcopy(item)
                        except Exception:
                            pass

                rotated_input["board"] = torch.rot90(rotated_input["board"], k, [2, 3])  # Rotate board
                rotated_input["legal_mask"][:, 2:] = torch.rot90(rotated_input["legal_mask"][:, 2:].squeeze().reshape(9, 9), k, [0, 1]).reshape(81).unsqueeze(0)  # Rotate legal mask (excluding pass and resign)
                rotated_target["policy"][2:] = torch.rot90(rotated_target["policy"][2:].reshape(9, 9), k, [0, 1]).reshape(-1)  # Rotate policy target (excluding pass and resign)
                self.replay_buffer.push({"input": rotated_input, "target": rotated_target})

                # Also add the horizontally flipped version of the board
                # Also add the horizontally flipped version of the board
                flipped_input = deepcopy(rotated_input)
                flipped_target = deepcopy(rotated_target)
                flipped_input["board"] = torch.flip(flipped_input["board"], [3])  # Flip horizontally
                flipped_input["legal_mask"][:, 2:] = torch.flip(flipped_input["legal_mask"][:, 2:].squeeze().reshape(9, 9), [1]).reshape(81).unsqueeze(0)  # Flip legal mask horizontally
                flipped_target["policy"][2:] = torch.flip(flipped_target["policy"][2:].reshape(9, 9), [1]).reshape(-1)  # Flip policy target horizontally 
                self.replay_buffer.push({"input": flipped_input, "target": flipped_target})

    def post_game_train(self):

        if self.new_experience_count < self.min_replay_size:
            tqdm.write(f"[TRAINING] Not enough experiences yet: {self.new_experience_count}/{self.min_replay_size}")
            return
        
        # Close and clear progress bar since we're starting training
        if self.experience_pbar is not None:
            self.experience_pbar.close()
            self.experience_pbar = None
        
        if len(self.replay_buffer) < self.batch_size:
            tqdm.write(f"[TRAINING] Replay buffer too small: {len(self.replay_buffer)}/{self.batch_size}")
            return

        total_samples = int(self.replay_ratio * self.new_experience_count)
        num_steps = (total_samples + self.batch_size - 1) // self.batch_size
        tqdm.write(f"[TRAINING] Starting training: {self.new_experience_count} new experiences, {total_samples} total samples, {num_steps} steps")
        # Use minimal tqdm config to prevent deadlocks: ncols=0 skips terminal width detection
        pbar = tqdm(range(0, total_samples, self.batch_size), total=num_steps, desc="Training", file=sys.stderr, ncols=0, delay=0.5)
        
        # Track metrics for this phase
        phase_losses = {"value": [], "policy": [], "total": []}
        phase_start_time = time.time()
        logger = get_logger()
        
        for step_idx, _ in enumerate(pbar):
            try:
                batch = self.replay_buffer.sample(self.batch_size)
                batch_inputs = self.inference_client.get_inputs([experience["input"] for experience in batch])
                # Ensure inputs are on the correct device and dtype (model expects float32)
                for k in ("boards", "player_colors", "pass_counts", "legal_masks"):
                    if k in batch_inputs:
                        try:
                            batch_inputs[k] = batch_inputs[k].to(self.inference_client.device).to(dtype=torch.float32)
                        except Exception:
                            pass
                batch_targets = {
                    "values": torch.tensor([experience["target"]["value"] for experience in batch], dtype=torch.float32, device=self.inference_client.device),
                    # Ensure policy targets are on the correct device
                    # stored policies may be float16 on CPU; convert to model dtype on device
                    "policies": torch.stack([experience["target"]["policy"] for experience in batch]).to(self.inference_client.device).to(dtype=torch.float32)
                }

                # Apply label smoothing to policy targets to reduce overconfidence
                # Smooth toward uniform distribution with weight 0.01, then mask illegal moves
                n_actions = batch_targets["policies"].shape[1]
                uniform_policy = torch.ones(n_actions, device=self.inference_client.device) / n_actions
                batch_targets["policies"] = 0.99 * batch_targets["policies"] + 0.01 * uniform_policy
                
                # Mask out illegal moves and renormalize to ensure no mass on illegal positions
                batch_targets["policies"] = batch_targets["policies"] * batch_inputs["legal_masks"]
                batch_targets["policies"] = batch_targets["policies"] / (batch_targets["policies"].sum(dim=1, keepdim=True) + 1e-8)

                # --- Validation: ensure legal masks match policy targets ---
                masks = batch_inputs.get("legal_masks")
                policies = batch_targets.get("policies")
                if masks is None or policies is None:
                    raise RuntimeError("Missing masks or policies in training batch")
                if masks.dim() != 2 or policies.dim() != 2:
                    raise RuntimeError(f"Unexpected mask/policy dims: mask.dim={masks.dim()}, policies.dim={policies.dim()}")
                if masks.shape[0] != policies.shape[0]:
                    raise RuntimeError(f"Batch size mismatch between masks and policies: {masks.shape[0]} vs {policies.shape[0]}")
                if masks.shape[1] != policies.shape[1]:
                    raise RuntimeError(f"Action-space size mismatch between masks and policies: mask_len={masks.shape[1]}, policy_len={policies.shape[1]}")

                # For each sample, ensure policy mass is only on legal moves and that legal moves receive some mass
                for i in range(policies.shape[0]):
                    mask_i = masks[i]
                    pol_i = policies[i]
                    # boolean selector for legal moves
                    legal_sel = (mask_i != 0)
                    legal_count = int(legal_sel.sum().item())
                    if legal_count == 0:
                        raise RuntimeError(f"Sample {i}: no legal moves in mask (possible bug in game logic)")

                    legal_mass = float(pol_i[legal_sel].sum().item())
                    illegal_mass = float(pol_i[~legal_sel].sum().item())
                    total_mass = float(pol_i.sum().item())

                    # allow small numerical tolerance
                    if legal_mass <= 0.0 and total_mass > 0.0:
                        raise RuntimeError(f"Sample {i}: policy assigns mass only to illegal moves (legal_mass={legal_mass}, total={total_mass})")
                    if illegal_mass > 1e-6:
                        raise RuntimeError(f"Sample {i}: policy assigns non-negligible mass to illegal moves (illegal_mass={illegal_mass})")

                self.optimizer.zero_grad()
                self.inference_client.model.train()  # Set model to training mode

                # Update learning rate based on schedule
                self._update_learning_rate()

                # Mixed precision when using CUDA
                if self.scaler is not None:
                    with torch.amp.autocast('cuda', dtype=torch.float16):
                        _, predicted_values, predicted_policies = self.inference_client.model(batch_inputs)
                        value_loss = F.mse_loss(predicted_values.squeeze(), batch_targets["values"])

                        # Use smaller mask value that fits in float16 range
                        masked_logits = predicted_policies.masked_fill(batch_inputs["legal_masks"] == 0, float("-1e4"))
                        log_probs = F.log_softmax(masked_logits, dim=1)
                        policy_loss = -torch.sum(batch_targets["policies"] * log_probs) / float(self.batch_size)

                        # Weighted combination: 0.75 * value + 1.0 * policy
                        loss = 0.75 * value_loss + 1.0 * policy_loss

                    # Scaled backward
                    self.scaler.scale(loss).backward()
                    # Unscale before clipping
                    self.scaler.unscale_(self.optimizer)
                    torch.nn.utils.clip_grad_norm_(self.inference_client.model.parameters(), max_norm=1.5)
                    self.scaler.step(self.optimizer)
                    self.scaler.update()
                else:
                    _, predicted_values, predicted_policies = self.inference_client.model(batch_inputs)
                    value_loss = F.mse_loss(predicted_values.squeeze(), batch_targets["values"])

                    # Use smaller mask value that fits in float16 range if needed
                    masked_logits = predicted_policies.masked_fill(batch_inputs["legal_masks"] == 0, float("-1e4"))
                    log_probs = F.log_softmax(masked_logits, dim=1)
                    policy_loss = -torch.sum(batch_targets["policies"] * log_probs) / float(self.batch_size)

                    # Weighted combination: 0.75 * value + 1.0 * policy
                    loss = 0.75 * value_loss + 1.0 * policy_loss
                    loss.backward()
                    torch.nn.utils.clip_grad_norm_(self.inference_client.model.parameters(), max_norm=1.5)
                    self.optimizer.step()

                self.total_training_steps += 1
                
                # Track losses for phase aggregation
                v_loss = value_loss.item()
                p_loss = policy_loss.item()
                t_loss = loss.item()
                phase_losses["value"].append(v_loss)
                phase_losses["policy"].append(p_loss)
                phase_losses["total"].append(t_loss)
                
                # Log training step
                current_lr = self.optimizer.param_groups[0]['lr']
                logger.log_training_step(self.total_training_steps, v_loss, p_loss, t_loss, current_lr)
                
                pbar.set_postfix({"value_loss": f"{v_loss:.4f}", "policy_loss": f"{p_loss:.4f}", "total_loss": f"{t_loss:.4f}"})
                
                # Periodic GPU cache clearing every 50 steps to prevent fragmentation without constant stalling
                if self.total_training_steps % 50 == 0:
                    torch.cuda.empty_cache()
            except Exception as e:
                tqdm.write(f"[TRAINING ERROR] Step {step_idx}: {type(e).__name__}: {e}")
                import traceback
                traceback.print_exc()
                raise

        finished = self.batch_size * (total_samples // self.batch_size)
        self.total_experiences_played += self.new_experience_count
        self.new_experience_count = 0  # Reset new experience count after training
        self.trained_times += 1
        tqdm.write(f"[TRAINING] Completed training phase {self.trained_times}, {len(phase_losses['value'])} batches processed")
        
        # Log phase end with aggregated metrics
        phase_duration = time.time() - phase_start_time
        avg_value_loss = sum(phase_losses["value"]) / len(phase_losses["value"]) if phase_losses["value"] else 0
        avg_policy_loss = sum(phase_losses["policy"]) / len(phase_losses["policy"]) if phase_losses["policy"] else 0
        avg_total_loss = sum(phase_losses["total"]) / len(phase_losses["total"]) if phase_losses["total"] else 0
        
        logger.log_phase_end(
            phase=self.trained_times,
            total_steps=num_steps,
            avg_value_loss=avg_value_loss,
            avg_policy_loss=avg_policy_loss,
            avg_total_loss=avg_total_loss,
            experiences_count=self.total_experiences_played,
            duration_sec=phase_duration
        )

        # Save replay buffer and model after each training session
        # Save to both generic path (for inference) and timestamped path (for checkpoints)
        checkpoint_model_path = f"saves/gogo81_{self.total_experiences_played}.pt"
        checkpoint_buffer_path = f"saves/buffer_{self.total_experiences_played}.pt"
        
        self.replay_buffer.save()  # Generic path
        self.replay_buffer.save(checkpoint_buffer_path)  # Timestamped checkpoint
        torch.save(self.inference_client.model.state_dict(), self.model_save_path)  # Generic path
        torch.save(self.inference_client.model.state_dict(), checkpoint_model_path)  # Timestamped checkpoint
        
        # Save training logs after each phase
        self.save_logs()
        
        # Recreate progress bar for next accumulation phase
        self.experience_pbar = tqdm(
            total=self.min_replay_size,
            desc="Accumulating experiences",
            unit="exp",
            file=sys.stderr,
            disable=False,
            position=0,
            leave=True
        )

    def AppendDataset(self, request, context):
        try:
            tqdm.write(f"[APPEND_DATASET] Starting, current new_experience_count: {self.new_experience_count}")
            self.push_experiences(request)
            nb_new_experiences = len(request.data) * 8
            self.new_experience_count += nb_new_experiences
            tqdm.write(f"[APPEND_DATASET] Added {nb_new_experiences} experiences, total: {self.new_experience_count}/{self.min_replay_size}")
            
            # Update progress bar
            if self.experience_pbar is not None:
                self.experience_pbar.update(nb_new_experiences)
            
            tqdm.write(f"[APPEND_DATASET] Calling post_game_train()")
            self.post_game_train()
            tqdm.write(f"[APPEND_DATASET] Finished post_game_train()")

        except Exception as e:
            print(f"[APPEND_DATASET ERROR] {type(e).__name__}: {e}")
            traceback.print_exc()

        return empty_pb2.Empty()
    
    def save_logs(self):
        """Save all training logs and metrics to disk."""
        logger = get_logger()
        logger.save_session()
        logger.print_session_summary()