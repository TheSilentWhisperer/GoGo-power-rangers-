# GoGo Power Rangers! - Budget AlphaGo on an Old Laptop

An AlphaGo-inspired MCTS + neural network implementation trained on a single laptop. This project addresses the "implementation gap" left by AlphaGo/AlphaZero papers by providing a complete, documented reference implementation combining Python (training/inference) and Go (game engine/UI) via gRPC over Unix Domain Sockets.

## Overview

- **Self-play training**: MCTS with PuCT (neural network-guided tree search)
- **Dual-process architecture**: Go client ↔ Python server via gRPC (400 simulations/move, batch inference)
- **Training**: Gogo81 neural network (~1.5M params) on 800K FIFO replay buffer with 12:1 data augmentation
- **Results**: ~3-4 kyu play strength achieved on RTX 5080 in ~39 hours

See [report.pdf](report.pdf) for detailed technical implementation and findings.

## Requirements

- **Linux** only (UDS socket required; Windows/macOS not supported)
- **Python 3.9+**, **Go 1.20+**, **CUDA 11.8+** (optional, falls back to CPU)
- **Hardware**: 8GB RAM minimum; RTX 3060+ recommended for reasonable training speed

## Installation

```bash
# 1. Clone repo
git clone https://github.com/your-username/GoGo-power-rangers.git
cd GoGo\ power\ rangers\!

# 2. Python environment
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt

# Generate protobuf bindings
cd python && python -m grpc_tools.grpc_python_codegen \
  -I../proto --python_out=gen --grpc_python_out=gen ../proto/remote_trainer.proto && cd ..

# 3. Build Go binary
cd go && go mod download && go build -o bin/play ./cmd/play && cd ..

# Optional: Place pretrained model at python/saves/gogo81.pt
# (Otherwise trains from scratch)
```

## Quick Start

**Terminal 1**: Start Python server
```bash
cd python && source ../.venv/bin/activate && ./run_server.sh
```
Flags: `--foreground` (debug), `--no-training` (inference-only)

**Terminal 2**: Launch self-play (both Black/White automated)
```bash
cd go && ./bin/play
```
- Space to pause/resume
- `-eval`: Stronger play (5000 sims, deterministic)
- `-tournament`: Run 3×3 tournament (200/800/3200 sims agents)

## Architecture

**Communication**: Go client (game/MCTS) ↔ Python server (training/inference) via gRPC over Unix Domain Socket (`/tmp/position_evaluator.sock`). This enables low-latency batched neural network evaluation with queue-based timeout-adaptive batching (50ms window, batch size 20).

**MCTS**: PuCT algorithm with value guidance from neural network. Selection via UCB($s$, $a$) = $Q$ + $C \cdot P \cdot \sqrt{N_s} / (1 + N_{s,a})$. No virtual loss; relies on natural visit-count penalization for parallelism.

**Training**: Gogo81 (~1.5M params) trained on replay buffer via experience accumulation → training phases (12× replay ratio, 512-batch Adam, cosine annealing). Aggressive quantization (int8/uint8 boards, float16 policy) enables 800K buffer in ~4GB.

## Monitoring & Configuration

**Real-time training**:
```bash
tail -f python/logs/$(ls -t python/logs | head -1)/training_steps.json
python plot_training_logs.py  # Generates training_logs.png
```

**Hyperparameters** (edit before running):
- `go/internal/agents/mcts.go`: `SimulationsTraining`, `ResignThreshold`, `Temperature`
- `python/services/training.py`: `BATCH_SIZE`, `LEARNING_RATE`, `REPLAY_BUFFER_SIZE`
- `python/models/gogo81.py`: `model_channels=96`, `num_res_blocks=15`

**Running in background** (with tmux):
```bash
tmux new-session -d -s gogo -c /path/to/python "bash run_server.sh"
tmux attach -t gogo        # View logs
tmux kill-session -t gogo  # Cleanup
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "unix:///tmp/position_evaluator.sock: connection refused" | Start Python server: `./run_server.sh` |
| Not Linux OS | This project requires Linux (UDS not supported on macOS/Windows) |
| CUDA out of memory | Reduce batch size in `python/services/inference.py` |
| Slow training | Check GPU: `nvidia-smi`. Verify no competing processes. |
| Server hangs | Restart Python server process. |

## Known Limitations

**Black/White Color Asymmetry**: Trained model shows ~0% White winrate across all simulation budgets. Likely causes:
- Architectural: Color encoding as scalar (not in initial layers)
- Game theory: Komi 6.5 insufficient for White compensation
- Feedback loop: Pessimistic value estimates → weak White play → reinforced pessimism

See [report.pdf](report.pdf) §Results for analysis.
