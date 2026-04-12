#!/usr/bin/env python3
"""
Plot training logs from the latest training session.
"""

import json
import matplotlib.pyplot as plt
import numpy as np
from pathlib import Path

# Load the latest logs
log_dir = Path("python/logs/20260407_231235")
training_steps_file = log_dir / "training_steps.json"
phase_summary_file = log_dir / "phase_summary.json"

print(f"Loading training steps from {training_steps_file}...")
with open(training_steps_file, 'r') as f:
    training_steps = json.load(f)

print(f"Loading phase summary from {phase_summary_file}...")
with open(phase_summary_file, 'r') as f:
    phase_summary = json.load(f)

# Extract data for plotting
steps = [s["step"] for s in training_steps]
value_losses = [s["value_loss"] for s in training_steps]
policy_losses = [s["policy_loss"] for s in training_steps]
total_losses = [s["total_loss"] for s in training_steps]
learning_rates = [s["learning_rate"] for s in training_steps]

phases = [p["phase"] for p in phase_summary]
phase_avg_value_losses = [p["avg_value_loss"] for p in phase_summary]
phase_avg_policy_losses = [p["avg_policy_loss"] for p in phase_summary]
phase_avg_total_losses = [p["avg_total_loss"] for p in phase_summary]
phase_durations = [p["duration_sec"] for p in phase_summary]

# Create a figure with multiple subplots
fig, axes = plt.subplots(2, 2, figsize=(14, 10))
fig.suptitle('Training Progress Over Time', fontsize=16, fontweight='bold')

# Plot 1: Total Loss over steps
ax = axes[0, 0]
ax.plot(steps, total_losses, linewidth=1.5, color='#1f77b4', alpha=0.8)
ax.set_xlabel('Training Step')
ax.set_ylabel('Total Loss')
ax.set_title('Total Loss per Step')
ax.grid(True, alpha=0.3)

# Plot 2: Value and Policy Loss over steps
ax = axes[0, 1]
ax.plot(steps, value_losses, linewidth=1.5, label='Value Loss', color='#ff7f0e', alpha=0.8)
ax.plot(steps, policy_losses, linewidth=1.5, label='Policy Loss', color='#2ca02c', alpha=0.8)
ax.set_xlabel('Training Step')
ax.set_ylabel('Loss')
ax.set_title('Value vs Policy Loss per Step')
ax.legend()
ax.grid(True, alpha=0.3)

# Plot 3: Learning Rate Schedule
ax = axes[1, 0]
ax.plot(steps, learning_rates, linewidth=1.5, color='#d62728', alpha=0.8)
ax.set_xlabel('Training Step')
ax.set_ylabel('Learning Rate')
ax.set_title('Learning Rate Schedule')
ax.grid(True, alpha=0.3)
# Use scientific notation for y-axis
ax.ticklabel_format(style='scientific', axis='y', scilimits=(0,0))

# Plot 4: Phase-wise average losses
ax = axes[1, 1]
x_pos = np.arange(len(phases))
width = 0.25
ax.bar(x_pos - width, phase_avg_value_losses, width, label='Avg Value Loss', alpha=0.8, color='#ff7f0e')
ax.bar(x_pos, phase_avg_policy_losses, width, label='Avg Policy Loss', alpha=0.8, color='#2ca02c')
ax.bar(x_pos + width, phase_avg_total_losses, width, label='Avg Total Loss', alpha=0.8, color='#1f77b4')
ax.set_xlabel('Training Phase')
ax.set_ylabel('Average Loss')
ax.set_title('Average Losses per Phase')
ax.set_xticks(x_pos)
ax.set_xticklabels([f'P{p}' for p in phases])
ax.legend()
ax.grid(True, alpha=0.3, axis='y')

plt.tight_layout()
plt.savefig('training_logs.png', dpi=300, bbox_inches='tight')
print(f"✓ Saved training plot to training_logs.png")

# Print summary statistics
print(f"\nTraining Summary:")
print(f"  Total training steps: {len(training_steps)}")
print(f"  Total training phases: {len(phase_summary)}")
print(f"  Final total loss: {total_losses[-1]:.4f}")
print(f"  Best total loss: {min(total_losses):.4f} (step {steps[np.argmin(total_losses)]})")
print(f"  Total training time: {sum(phase_durations):.1f} seconds ({sum(phase_durations)/60:.1f} minutes)")
