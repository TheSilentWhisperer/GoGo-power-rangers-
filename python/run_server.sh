#!/bin/bash

# Script to run the Python gRPC server with optimizations
# to prevent slowdown when running in background or screen is locked
# 
# Usage:
#   ./run_server.sh              # Run detached in background (recommended)
#   ./run_server.sh --foreground # Run in foreground for debugging
#   ./run_server.sh --no-training # Run without training service
#   ./run_server.sh --foreground --no-training # Foreground without training

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Parse arguments
FOREGROUND=false
TRAINING_FLAG=""
for arg in "$@"; do
    if [[ "$arg" == "--foreground" ]]; then
        FOREGROUND=true
    elif [[ "$arg" == "--no-training" ]]; then
        TRAINING_FLAG="--no-training"
    fi
done

echo "Starting Python gRPC server with performance optimizations..."
if [[ -n "$TRAINING_FLAG" ]]; then
    echo "⚠️  Training service DISABLED"
fi

# Set environment variables for optimal performance
export CUDA_LAUNCH_BLOCKING=0  # Allow async CUDA operations
export CUDA_DEVICE_ORDER=PCI_BUS_ID
export OMP_NUM_THREADS=1  # Prevent OpenMP threading issues
export OPENBLAS_NUM_THREADS=1
export MKL_NUM_THREADS=1
export PYTHONUNBUFFERED=1  # Disable Python output buffering

# Function to run with systemd-inhibit if available (prevents sleep/idle)
run_with_inhibit() {
    if command -v systemd-inhibit &> /dev/null; then
        echo "Using systemd-inhibit to prevent system sleep during training..."
        systemd-inhibit --what=idle:sleep --why="MCTS model training in progress" python3 -u main.py $TRAINING_FLAG "$@"
    else
        echo "systemd-inhibit not available, running normally..."
        python3 -u main.py $TRAINING_FLAG "$@"
    fi
}

if [ "$FOREGROUND" = true ]; then
    # Foreground mode for debugging
    echo "[FOREGROUND MODE] Running in foreground..."
    run_with_inhibit
else
    # Background mode (default) - detach from terminal completely
    echo "[BACKGROUND MODE] Running detached in background..."
    echo "Check progress with: tail -f training.log"
    
    # Run detached with nohup
    if command -v systemd-inhibit &> /dev/null; then
        nohup bash -c "source venv/bin/activate && systemd-inhibit --what=idle:sleep --why=\"MCTS model training in progress\" python3 -u main.py $TRAINING_FLAG" > training.log 2>&1 &
    else
        nohup bash -c "source venv/bin/activate && python3 -u main.py $TRAINING_FLAG" > training.log 2>&1 &
    fi
    
    sleep 2
    if ps aux | grep -q "[p]ython3.*main.py"; then
        echo "✓ Server started successfully in background"
    else
        echo "✗ Server failed to start (check training.log for errors)"
        cat training.log | tail -20
        exit 1
    fi
fi
