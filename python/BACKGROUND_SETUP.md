# Running Python Server in Background

## Quick Start (using tmux)

```bash
# Start in a new tmux session (detached)
tmux new-session -d -s gogo -c /home/alyx/Code/ENS/MCTS/GoGo\ power\ rangers\!/python
tmux send-keys -t gogo "bash run_server.sh" Enter

# Check the session
tmux attach -t gogo

# Detach (don't close): Ctrl+B then D

# Kill the session when done
tmux kill-session -t gogo
```

## How This Prevents Slowdown

1. **nice -n -10**: Gives the process higher CPU priority so it's not preempted when you're doing other things
2. **tmux**: Keeps the process running independently of your terminal/login session. The process continues even if you lock the screen or close the window.
3. **Environment Variables**: 
   - `CUDA_LAUNCH_BLOCKING=0`: Allows async GPU operations (better performance)
   - `OMP_NUM_THREADS=1`: Prevents threading confusion
   - Prevents Python from getting throttled by thread scheduling

## System-Level Optimizations (Optional, Requires Root)

For more aggressive optimization, temporarily disable CPU frequency scaling:

```bash
# Check current governor (should be "powersave" or "schedutil" on battery)
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor

# Switch to performance mode (requires sudo)
sudo sh -c 'for cpu in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do echo "performance" > "$cpu"; done'

# Verify
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor

# Switch back to power saving when done
sudo sh -c 'for cpu in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do echo "powersave" > "$cpu"; done'
```

## Prevent System Sleep

```bash
# Disable screen lock and sleep temporarily
gsettings set org.gnome.desktop.session idle-delay 0  # Disables screensaver

# Re-enable after training
gsettings set org.gnome.desktop.session idle-delay 300  # 5 minutes
```

Or use:
```bash
# Keep system awake while process runs
systemd-inhibit --what=idle:sleep --why="Training model" bash run_server.sh
```

## Monitoring Background Process

```bash
# Check if process is running
tmux list-sessions

# Check resource usage
ps aux | grep python | grep main

# Monitor in real-time
watch -n 1 'ps aux | grep python | grep main'

# Check CUDA usage if applicable
nvidia-smi -l 1  # Refresh every 1 second
```
