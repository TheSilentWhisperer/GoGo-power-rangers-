#!/bin/bash
# GPU Performance Optimization Script for RTX 4060
# Run with: sudo bash GPU_PERFORMANCE_SETUP.sh
#
# Safety notes:
# - RTX 4060 thermal limit: 76-82°C (currently at 67°C - safe to max out)
# - Max GPU clock: 2450 MHz
# - These settings are safe for your hardware

set -e

echo "=========================================="
echo "GPU Performance Optimization"
echo "=========================================="

# Check if running as sudo
if [ "$EUID" -ne 0 ]; then 
    echo "ERROR: This script must be run with sudo"
    exit 1
fi

# 1. Enable GPU Persistent Mode (keeps GPU in active state)
echo ""
echo "1/3: Enabling GPU persistent mode..."
nvidia-smi -pm 1
echo "✓ GPU persistent mode enabled"

# 2. Lock GPU to maximum clock (RTX 4060 max is 2450 MHz)
echo ""
echo "2/3: Locking GPU to maximum performance clocks..."
nvidia-smi -i 0 -lgc 2450
echo "✓ GPU clocks locked to 2450 MHz"

# 3. Set CPU to performance mode (disabled frequency scaling)
echo ""
echo "3/3: Setting CPU to performance mode..."
echo performance | tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor > /dev/null
echo "✓ CPU frequency scaling disabled (locked to max)"

# Verification
echo ""
echo "=========================================="
echo "Verification"
echo "=========================================="
echo ""
echo "GPU Status:"
nvidia-smi --query-gpu=name,driver_version,compute_cap --format=csv,noheader
nvidia-smi --query-gpu=index,memory.total,temperature.gpu,power.limit --format=csv,noheader,nounits

echo ""
echo "CPU Frequency:"
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor

echo ""
echo "=========================================="
echo "✓ All optimizations applied successfully!"
echo "=========================================="
echo ""
echo "These settings will persist until reboot."
echo "To revert CPU frequency scaling, run:"
echo "  echo powersave | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor"
