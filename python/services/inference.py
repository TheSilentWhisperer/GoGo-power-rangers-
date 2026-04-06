import gen.proto.remote_trainer_pb2 as remote_trainer_pb2
import gen.proto.remote_trainer_pb2_grpc as remote_trainer_pb2_grpc
from google.protobuf import empty_pb2
import torch 
import torch.nn.functional as F
import queue
import threading
import time
import os

# GPU Performance Optimizations
torch.backends.cudnn.benchmark = True  # Auto-tune cuDNN for best performance
torch.backends.cudnn.deterministic = False  # Faster but non-deterministic
os.environ['PYTORCH_CUDA_ALLOC_CONF'] = 'expandable_segments:True'
os.environ['CUDA_LAUNCH_BLOCKING'] = '0'  # Enable async GPU operations

class InferenceClient(remote_trainer_pb2_grpc.PositionEvaluatorServicer):
    def __init__(self, model, device, batch_size):
        super().__init__()
        
        self.model = model
        self.device = device
        self.batch_size = batch_size
        self.to_evaluate = []
        self.evaluated = []  # Simple list of (request_id, value, priors) tuples
        self.model_forward_count = 0
        self.batch_start_time = None  # Track when batch started
        self.batch_timeout = 0.05  # Flush batch if stuck for 500ms (100x buffer over 5ms eval time)
        
        # Start background thread to periodically flush partial batches
        self.should_stop = False
        self.flush_thread = threading.Thread(target=self._batch_flush_monitor, daemon=True)
        self.flush_thread.start()
        
        # Disable torch.compile - TorchInductor worker processes hang in concurrent gRPC context
        # Using standard forward pass for gRPC server stability
        # try:
        #     if device.type == 'cuda':
        #         import sys
        #         if sys.version_info >= (3, 11):
        #             original_forward = self.model.forward
        #             self.model.forward = torch.compile(
        #                 original_forward, 
        #                 mode="reduce-overhead", 
        #                 fullgraph=False
        #             )
        #             print("[INFO] Model forward pass compiled for faster inference")
        #         else:
        #             print("[INFO] torch.compile requires Python 3.11+, using standard forward pass")
        # except Exception as e:
        #     print(f"[WARNING] torch.compile failed: {e}. Using standard forward pass.")
        print("[INFO] torch.compile disabled for gRPC server stability")

    def _batch_flush_monitor(self):
        """Background thread that periodically checks and flushes partial batches."""
        while not self.should_stop:
            try:
                time.sleep(0.1)  # Check every 100ms
                
                # Check if batch has timed out
                if self.to_evaluate and self.batch_start_time:
                    batch_age = time.time() - self.batch_start_time
                    if batch_age >= self.batch_timeout:
                        self._process_batch()
                        self.batch_start_time = None
            except Exception as e:
                pass  # Silently handle monitor errors

    def ResetServer(self, request, context): 
        self.to_evaluate.clear()
        self.evaluated.clear()
        self.batch_start_time = None
        return empty_pb2.Empty()

    def get_input(self, request):
        history_length, height, width = request.history_length, request.height, request.width
        input_board = torch.tensor(request.flattened_board_history, dtype=torch.int8).view(1, history_length, height, width).to(self.device).float() # Shape: (1, history_length, height, width), convert to float for model
        pass_count = torch.tensor([request.enemy_passed], dtype=torch.float32).unsqueeze(-1).to(self.device) # Shape: (1, 1)
        legal_mask = torch.tensor(request.legal_actions_mask, dtype=torch.float32).view(1, height*width + 2).to(self.device) # Shape: (1, height*width + 2)
        # `request.black_to_play` is 1 if Black is to play, -1 if White is to play.
        # The board planes in `flattened_board_history` are provided from the
        # current player's perspective (1 for current player's stones,
        # -1 for opponent stones). We pass the player color so the model can
        # condition on the absolute color when needed.
        player_color = torch.tensor([request.black_to_play], dtype=torch.float32).unsqueeze(-1).to(self.device) # Shape: (1, 1)
        return {
            "request_id": request.request_id,
            "board": input_board,
            "player_color": player_color,
            "pass_count": pass_count,
            "legal_mask": legal_mask
        }

    def get_inputs(self, input_list):
        request_ids = [inp["request_id"] for inp in input_list]
        boards = torch.cat([inp["board"] for inp in input_list], dim=0)  # Shape: (batch_size, history_length, height, width)
        player_colors = torch.cat([inp["player_color"] for inp in input_list], dim=0)  # Shape: (batch_size, 1)
        pass_counts = torch.cat([inp["pass_count"] for inp in input_list], dim=0)  # Shape: (batch_size, 1)
        legal_masks = torch.cat([inp["legal_mask"] for inp in input_list], dim=0)  # Shape: (batch_size, height*width + 2) 

        return {
            "request_ids": request_ids,
            "boards": boards,
            "player_colors": player_colors,
            "pass_counts": pass_counts,
            "legal_masks": legal_masks
        }

    def SubmitEvaluation(self, request, context):
        """Queue position for batch evaluation."""
        try:
            # Track when batch starts
            if len(self.to_evaluate) == 0:
                self.batch_start_time = time.time()
            
            self.to_evaluate.append(request)
            
            # Check if batch has been waiting too long (safety timeout to prevent getting stuck)
            batch_age = time.time() - self.batch_start_time if self.batch_start_time else 0
            batch_timeout_exceeded = batch_age >= self.batch_timeout and len(self.to_evaluate) > 0
            
            # Process batch when: full OR explicitly flushed OR timeout exceeded
            if len(self.to_evaluate) >= self.batch_size or request.flush or batch_timeout_exceeded:
                self._process_batch()
                self.batch_start_time = None  # Reset timer after processing
            
        except Exception as e:
            print(f"Error in SubmitEvaluation: {e}")
            import traceback
            traceback.print_exc()
        
        return empty_pb2.Empty()
    
    def _process_batch(self):
        """Process all queued requests."""
        if not self.to_evaluate:
            return
        
        try:
            batch_requests = self.to_evaluate[:self.batch_size]
            del self.to_evaluate[:self.batch_size]
            
            t_start = time.time()
            
            # Get batch inputs
            history_length = batch_requests[0].history_length
            height = int(batch_requests[0].height)
            width = int(batch_requests[0].width)
            
            request_ids = [req.request_id for req in batch_requests]
            input_boards = torch.cat([
                torch.tensor(req.flattened_board_history, dtype=torch.float32).view(1, history_length, height, width) 
                for req in batch_requests
            ], dim=0).to(self.device)
            
            pass_counts = torch.tensor([req.enemy_passed for req in batch_requests], dtype=torch.float32).unsqueeze(-1).to(self.device)
            legal_masks = torch.cat([
                torch.tensor(req.legal_actions_mask, dtype=torch.float32).view(1, height*width + 2) 
                for req in batch_requests
            ], dim=0).to(self.device)
            
            player_colors = torch.tensor([req.black_to_play for req in batch_requests], dtype=torch.float32).unsqueeze(-1).to(self.device)
            
            batch_inputs = {
                "boards": input_boards,
                "player_colors": player_colors,
                "pass_counts": pass_counts,
                "legal_masks": legal_masks,
                "request_ids": request_ids
            }
            
            # Run inference
            self.model.eval()
            with torch.no_grad():
                _, values, priors_logits = self.model(batch_inputs)
            
            t_forward = time.time()
            
            # Process results
            values_cpu = values.cpu()
            masks_gpu = batch_inputs["legal_masks"]
            masked_logits = priors_logits.masked_fill(masks_gpu == 0, float('-1e9'))
            probs_gpu = F.softmax(masked_logits, dim=1)
            probs_cpu = probs_gpu.cpu()
            
            # Store results
            for i, rid in enumerate(request_ids):
                val = values_cpu[i].item() if hasattr(values_cpu[i], 'item') else float(values_cpu[i])
                probs = probs_cpu[i].detach().cpu().numpy()
                self.evaluated.append((rid, val, probs))
            
            t_end = time.time()
            
            # Cleanup
            del batch_inputs, values, priors_logits, masked_logits, probs_gpu, values_cpu, probs_cpu
            torch.cuda.empty_cache()
            
        except Exception as e:
            print(f"Error in _process_batch: {e}")
            import traceback
            traceback.print_exc()

    def EvaluatePosition(self, request, context):
        try:
            input_data = self.get_inputs([self.get_input(request)])
            self.model.eval()  # Set model to evaluation mode
            with torch.no_grad():  # Disable gradient calculation for inference
                _, value, priors_logits = self.model(input_data)  # Run inference on the single input
            # Move to CPU immediately before cleanup
            value_cpu = value.cpu()
            logits = priors_logits.squeeze(0).cpu()  # Move to CPU immediately
            mask_cpu = input_data["legal_masks"][0].cpu()  # Move to CPU immediately and save for later
            masked_logits = logits.masked_fill(mask_cpu == 0, float('-1e9'))
            priors = F.softmax(masked_logits, dim=0)
            
            # Clean up GPU tensors
            del input_data, value, priors_logits, logits, masked_logits
            torch.cuda.empty_cache()  # Clear GPU cache to prevent fragmentation
            

            # Debug info for synchronous evaluation: only print on anomalies
            try:
                legal_sel = (mask_cpu != 0)
                legal_count = int(legal_sel.sum().item())
                legal_mass = float(priors[legal_sel].sum().item())
                illegal_mass = float(priors[~legal_sel].sum().item())
            except Exception:
                legal_count = None
                legal_mass = None
                illegal_mass = None

            anomaly = False
            try:
                if legal_count == 0:
                    anomaly = True
                elif illegal_mass is not None and illegal_mass > 1e-6:
                    anomaly = True
                elif legal_mass is not None and legal_mass < 1.0 - 1e-6:
                    anomaly = True
            except Exception:
                anomaly = False

            response = remote_trainer_pb2.EvaluationResponseData()
            response.request_id = request.request_id
            response.value = value_cpu.item()
            response.priors.extend(priors.detach().cpu().numpy().tolist())
            del mask_cpu, priors  # Clean up CPU tensors
            return response
        except Exception as e:
            print(f"Error in EvaluatePosition: {e}")
            import traceback
            traceback.print_exc()
            return empty_pb2.Empty()

    def RetrieveEvaluation(self, request, context):
        """Retrieve the next evaluated position from the queue (FIFO)."""
        
        # Check if pending batch has timed out while waiting for more requests
        if self.to_evaluate and self.batch_start_time:
            batch_age = time.time() - self.batch_start_time
            if batch_age >= self.batch_timeout:
                self._process_batch()
                self.batch_start_time = None
        
        if self.evaluated:
            request_id, value, priors = self.evaluated.pop(0)
            response = remote_trainer_pb2.EvaluationResponse()
            response.data.request_id = request_id
            response.data.value = float(value)
            response.data.priors.extend(priors.tolist())
            return response
        
        return empty_pb2.Empty()