# GoGo Power Rangers! - Budget AlphaGo on an Old Laptop

An AlphaGo-inspired neural network MCTS implementation optimized for single-laptop training and inference. This project combines a Python gRPC server for neural network evaluation with a Go game engine and interactive UI powered by Ebiten.

## Overview

**GoGo Power Rangers** implements the core AlphaGo/AlphaZero training pipeline:
- **Self-play generation**: MCTS with neural network guidance (400 simulations/move during training)
- **Experience replay**: 800K capacity FIFO buffer with ~3.6M total positions cycled
- **Neural network training**: Gogo81 architecture (96 channels, 15 residual blocks, ~1.5M parameters)
- **Evaluation**: Interactive play and tournament evaluation with configurable simulation budgets (200/800/3200 simulations)

**Performance**: Trained to reasonable play strength (~3-4 kyu estimate) on an RTX 5080 in ~39 hours.

## Requirements

### System Requirements
- **Linux** (required for Unix Domain Socket communication between Go and Python)
  - macOS: Not supported (UDS socket path handling differs)
  - Windows: Not supported
- **Python 3.9+**
- **Go 1.20+**
- **CUDA 11.8+** (optional, uses CPU fallback if not available)

### Hardware
- **Minimum**: 8GB RAM, any discrete GPU (or CPU-only, ~10x slower)
- **Recommended**: 16GB+ RAM, RTX 3060 or better

## Installation

### 1. Clone the Repository
```bash
git clone https://github.com/your-username/GoGo-power-rangers.git
cd GoGo\ power\ rangers\!
```

### 2. Set Up Python Environment

```bash
# Create virtual environment
python3 -m venv .venv
source .venv/bin/activate

# Install Python dependencies
pip install --upgrade pip
pip install -r requirements.txt
```

Generate Python protobuf bindings:
```bash
cd python
python -m grpc_tools.grpc_python_codegen -I../proto --python_out=gen --grpc_python_out=gen ../proto/remote_trainer.proto
cd ..
```

### 3. Set Up Go Build

```bash
cd go

# Download Go dependencies
go mod download
go mod tidy

# Build the game binary
go build -o bin/play ./cmd/play

cd ..
```

### 4. Download Pretrained Model (Optional)

The system will train from scratch if no model exists. To use a pretrained model:

```bash
# Place the pretrained weights at:
python/saves/gogo81.pt
```

## Quick Start

### Terminal 1: Start the Python gRPC Server

```bash
cd python
source ../.venv/bin/activate
./run_server.sh
```

For training mode (default):
```bash
./run_server.sh
```

For inference-only mode (no training updates):
```bash
./run_server.sh --no-training
```

For debugging:
```bash
./run_server.sh --foreground
```

**Expected output:**
```
Starting Python gRPC server with performance optimizations...
[INFO] Loading pretrained model from saves/gogo81.pt
[INFO] Training service ENABLED
[INFO] Server started on unix:///tmp/position_evaluator.sock
```

### Terminal 2: Launch Self-Play Games

```bash
cd go
./bin/play
```

A window appears showing an interactive 9×9 Go board. **Both Black and White are controlled by MCTS agents** - you observe the game, no interaction needed. Press Space to pause/resume.

```bash
# With 5000 simulations per move (stronger but slower, argmax deterministic)
./bin/play -eval

# Tournament mode: 3×3 round-robin between 200/800/3200 simulation agents
./bin/play -tournament
```

## Project Structure

```
.
├── python/                           # Python server (gRPC, training, inference)
│   ├── main.py                       # Server entry point
│   ├── run_server.sh                 # Shell script with optimizations
│   ├── models/
│   │   └── gogo81.py                 # Neural network architecture
│   ├── services/
│   │   ├── inference.py              # Batch inference & queue management
│   │   └── training.py               # Training loop with replay buffer
│   ├── gen/proto/                    # Generated protobuf bindings
│   ├── saves/                        # Model checkpoints
│   └── logs/                         # Training logs (JSON)
│
├── go/                               # Go game engine & UI
│   ├── cmd/play/
│   │   └── main.go                   # Game entry point
│   ├── internal/
│   │   ├── agents/
│   │   │   ├── mcts.go               # MCTS algorithm (PuCT)
│   │   │   └── evaluator.go          # gRPC client for NN evaluation
│   │   ├── environment/
│   │   │   ├── board.go              # 9×9 board state & rules
│   │   │   └── scoring.go            # Chinese scoring
│   │   ├── ui/
│   │   │   ├── app.go                # Ebiten game loop
│   │   │   ├── tournament.go         # 3×3 tournament infrastructure
│   │   │   └── rendering.go          # Board visualization
│   │   └── utils/
│   │       └── helpers.go
│   ├── gen/proto/                    # Generated protobuf code
│   ├── go.mod & go.sum               # Go dependencies
│   └── bin/play                      # Generated binary
│
├── proto/
│   └── remote_trainer.proto          # gRPC service definitions
│
├── report.tex                        # Master's report (LaTeX)
├── report.pdf                        # Compiled report
├── references.bib                    # Bibliography
└── plot_training_logs.py             # Training visualization script
```

## Architecture

### Communication: Unix Domain Socket (UDS)

The Go client and Python server communicate via **Unix Domain Socket** at `/tmp/position_evaluator.sock`:

```
Go Game Engine (MCTS)
    ↓
gRPC over UDS
    ↓
Python Server (Neural Network Inference + Training)
```

**Why UDS?**
- Lower overhead than TCP for local communication
- No port binding conflicts
- **Linux-only** (this is why Windows/macOS are not supported)

### Server Components

#### Inference Service
- **Batch queue**: Accumulates position evaluation requests (up to 20)
- **Timeout**: Evaluates after 50ms even if buffer not full
- **GPU optimizations**: TF32 enabled for faster inference
- Each request returns: policy logits + value estimate

#### Training Service
- **Replay buffer**: FIFO with 800K position capacity
- **Training loop**: Continuous updates when buffer has ≥100K positions
- **Adam optimizer**: Learning rate 3e-4, cosine annealing with warmup
- **Dual loss**: Policy cross-entropy + MSE value loss

### MCTS Algorithm (PuCT)

```
Per move during search:
  for i in 1..num_simulations:
    1. Select path from root using UCB scores
    2. Expand leaf node with NN policy guidance
    3. Backup value estimate up the tree

Move selection: argmax visit count (temperature sampling during training)
```

**Configuration:**
- Training: 400 simulations/move, temperature=1.0, resign at -0.9
- Evaluation: 200/800/3200 simulations/move (configurable), argmax, no resign

## Usage Modes

### 1. Self-Play Training
```bash
go/bin/play
```
- Both Black and White controlled by MCTS agents (400 simulations/move)
- Uses temperature sampling for exploration and Dirichlet noise
- Self-play games automatically recorded and fed to training pipeline
- Press Space to pause/resume observation
- Continuous training occurs in the Python server

### 2. Evaluation Mode (Stronger Self-Play)
```bash
go/bin/play -eval
```
- Both agents use 5000 simulations/move with argmax (deterministic, no noise)
- No training updates submitted (inference-only mode)
- Stronger play quality for observation (no exploration)

### 3. Tournament Mode
```bash
go/bin/play -tournament
```
Runs 3×3 round-robin tournament (no UI, console output):
- **Agents**: 200 sims, 800 sims, 3200 sims per move per agent
- **Setup**: Each pair plays 10 self-play games + 20 cross-play games (different colors)
- **Output**: Win rates by agent strength

Example results from training:
```
200 vs 200:     80% (self), 100%/0% (vs 800), 90%/0% (vs 3200)
800 vs 800:     100% (self), 100%/0% (vs 200), 100%/0% (vs 3200)
3200 vs 3200:   90% (self), 100%/0% (vs 200), 90%/0% (vs 800)
```
**Note**: Black winrate significantly higher across all matchups (color asymmetry in training).

## Training Pipeline During Gameplay

When running `./bin/play` or `./bin/play -eval`, the Python server **continuously**:
1. Receives board positions and move visit counts from the UI
2. Stores them in the replay buffer
3. Trains the neural network in the background when buffer ≥ 100K positions

To monitor training in real-time:
```bash
tail -f python/logs/$(ls -t python/logs | head -1)/training_steps.json
```

To visualize all training progress:
```bash
python plot_training_logs.py  # Generates training_logs.png
```

## Command-Line Flags

### Python Server
```bash
./run_server.sh                    # Default: training enabled, background
./run_server.sh --foreground       # Run in foreground (debugging)
./run_server.sh --no-training      # Disable training, inference-only
./run_server.sh --foreground --no-training
```

**Environment Variables:**
```bash
CUDA_LAUNCH_BLOCKING=0             # Async CUDA (default)
OMP_NUM_THREADS=1                  # Prevent OpenMP issues
PYTHONUNBUFFERED=1                # Unbuffered output
```

### Go  Game
```bash
./bin/play                         # Self-play training (400 sims, training enabled)
./bin/play -eval                   # Evaluation mode (5000 sims, no training)
./bin/play -tournament             # Tournament 3×3 matrix
```

## Advanced: Running in Background

Use `tmux` to keep the server running while you do other work:

```bash
# Start Python server in new tmux session (detached)
tmux new-session -d -s gogo -c /path/to/python
tmux send-keys -t gogo "bash run_server.sh" Enter

# Check server logs
tmux attach -t gogo

# Detach without killing: Ctrl+B then D

# Kill when done
tmux kill-session -t gogo
```

Or use `systemd-inhibit` to prevent system sleep:
```bash
systemd-inhibit --what=idle:sleep --why="MCTS training" bash run_server.sh
```

## Monitoring Training

View training logs from the latest session:

```bash
cd python
ls -t logs | head -1  # Find latest session
cat logs/[latest-session]/training_steps.json | jq '.[] | .total_loss' | tail -20
```

Generate training visualization:
```bash
python ../plot_training_logs.py
```
Output: `training_logs.png` (loss curves, learning rate schedule, phase-wise summary)

## Configuration

### Model Architecture (Gogo81)
Edit `python/models/gogo81.py`:
```python
Gogo81(model_channels=96, num_res_blocks=15, input_planes=4)
```
- **Channels**: Feature map width (96 = ~1.5M parameters)
- **Blocks**: Residual block depth (15 = shallow, good balance)
- **Input planes**: Board history (4 = last 4 moves)

### Training Hyperparameters
Edit `python/services/training.py`:
```python
BATCH_SIZE = 128
LEARNING_RATE = 3e-4
REPLAY_BUFFER_SIZE = 800_000
MIN_BUFFER_SIZE = 100_000  # Start training when buffer has this many positions
```

### MCTS Parameters
Edit `go/internal/agents/mcts.go`:
```go
const SimulationsTraining = 400
const SimulationsEval = 2000
const ResignThreshold = -0.9
const Temperature = 1.0  // During training; 0 during eval
```

## Troubleshooting

### "Error: unix:///tmp/position_evaluator.sock: connection refused"
- Python server not running: `cd python && ./run_server.sh`
- Wrong OS: This project requires Linux (UDS not supported on Windows/macOS)

### "CUDA out of memory"
- Reduce batch size in `python/services/inference.py`: `batch_size = 10`
- Or run on CPU (slower): server will auto-fallback if CUDA unavailable

### "Slow training feed rate"
- Increase self-play simulations (trades off: stronger opponents, slower generation)
- Check GPU usage: `nvidia-smi -l 1`
- Verify no other GPU processes: `nvidia-smi`

### Server hangs during inference
- Restart server: `kill` the Python process and `./run_server.sh` again
- Known issue with torch.compile disabled (causing hangs in gRPC context)

## Performance Characteristics

**Training Speed**: ~28 positions/second (400 simulations/move)
- Breakdown: 25 sec MCTS search + 3.5 sec NN inference per 100 positions
- With RTX 5080: ~3.6 hours to generate 360k self-play positions (~38.6 hours total with training)

**Inference Speed**: ~200 positions/second (batch size 20, 400 sims)
- Bottleneck: MCTS simulation cost, not NN evaluation

**Memory**: ~4GB peak (model + replay buffer)

## Known Limitations

### Black/White Color Asymmetry
The trained model exhibits severe **White color bias**, achieving ~0% White winrate across all simulation budgets. Possible causes:
1. **Architectural**: Color encoding as scalar (not embedded in initial layers)
2. **Game-theoretic**: Komi 6.5 insufficient for White compensation in training
3. **Feedback-loop**: Pessimistic value estimates → weak White play → reinforced pessimism

This is an area for future investigation and improvement.

## References

Key papers implemented:
- AlphaGo: [Mastering the game of Go with deep neural networks and tree search](https://www.nature.com/articles/nature16961) (Silver et al., 2016)
- AlphaGo Zero: [Mastering the game of Go without human knowledge](https://www.nature.com/articles/nature24270) (Silver et al., 2017)
- AlphaZero: [Mastering Chess and Shogi by Self-Play with a General Reinforcement Learning Algorithm](https://arxiv.org/abs/1712.01724) (Silver et al., 2018)

See [references.bib](references.bib) for full bibliography.

## Master's Project Report

For detailed technical implementation, see [report.pdf](report.pdf):
- Environment Simulation (9×9 Go, Chinese scoring)
- Parallel MCTS with PuCT algorithm
- Training pipeline architecture
- Results and tournament analysis
- Code architecture and communication

## License

This project is provided as-is for educational purposes.

## Authors

- **Alyx Liao**
- **Raphaël Giraud**

Master IASD, 2026
