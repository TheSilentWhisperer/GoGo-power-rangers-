"""
Utility script to analyze and visualize training logs.
Run: python scripts/analyze_logs.py
"""

import json
import os
from pathlib import Path
from datetime import datetime
import statistics


def analyze_session(session_dir):
    """Analyze a single training session."""
    session_dir = Path(session_dir)
    if not session_dir.exists():
        print(f"Session directory not found: {session_dir}")
        return
    
    print(f"\n{'='*80}")
    print(f"SESSION ANALYSIS: {session_dir.name}")
    print(f"{'='*80}\n")
    
    # Load and display phase summary
    phase_file = session_dir / "phase_summary.json"
    if phase_file.exists():
        with open(phase_file) as f:
            phases = json.load(f)
        
        print(f"{'='*80}")
        print("PHASE SUMMARY")
        print(f"{'='*80}")
        print(f"{'Phase':<8} {'Avg Val Loss':<16} {'Avg Policy Loss':<16} {'Avg Total Loss':<16} {'Exp':<8} {'Time(s)':<10}")
        print(f"{'-'*80}")
        
        for p in phases:
            print(f"{p['phase']:<8} {p['avg_value_loss']:<16.6f} {p['avg_policy_loss']:<16.6f} " +
                  f"{p['avg_total_loss']:<16.6f} {p['experiences_count']:<8} {p['duration_sec']:<10.1f}")
        
        # Compute trends
        if len(phases) > 1:
            print(f"\n{'='*80}")
            print("TRAINING TRENDS")
            print(f"{'='*80}")
            
            first_phase = phases[0]
            last_phase = phases[-1]
            
            val_loss_change = ((last_phase['avg_value_loss'] - first_phase['avg_value_loss']) 
                              / first_phase['avg_value_loss'] * 100)
            policy_loss_change = ((last_phase['avg_policy_loss'] - first_phase['avg_policy_loss']) 
                                 / first_phase['avg_policy_loss'] * 100)
            total_loss_change = ((last_phase['avg_total_loss'] - first_phase['avg_total_loss']) 
                                / first_phase['avg_total_loss'] * 100)
            
            print(f"Phase 1 → Phase {len(phases)}:")
            print(f"  Value Loss:  {first_phase['avg_value_loss']:.6f} → {last_phase['avg_value_loss']:.6f}  ({val_loss_change:+.1f}%)")
            print(f"  Policy Loss: {first_phase['avg_policy_loss']:.6f} → {last_phase['avg_policy_loss']:.6f}  ({policy_loss_change:+.1f}%)")
            print(f"  Total Loss:  {first_phase['avg_total_loss']:.6f} → {last_phase['avg_total_loss']:.6f}  ({total_loss_change:+.1f}%)")
    
    # Load and display validation summary
    validation_file = session_dir / "validation_summary.jsonl"
    if validation_file.exists():
        with open(validation_file) as f:
            validations = [json.loads(line) for line in f if line.strip()]
        
        if validations:
            print(f"\n{'='*80}")
            print("VALIDATION SUMMARY")
            print(f"{'='*80}")
            print(f"{'Phase':<8} {'Win Rate':<15} {'Score Diff':<15} {'Total Games':<12}")
            print(f"{'-'*80}")
            
            for v in validations:
                print(f"{v['phase']:<8} {v['win_rate_pct']:<15} {v['avg_score_diff']:<15.2f} {v['total_games']:<12}")
    
    # Training steps statistics
    steps_file = session_dir / "training_steps.json"
    if steps_file.exists():
        with open(steps_file) as f:
            steps = json.load(f)
        
        if steps:
            print(f"\n{'='*80}")
            print("TRAINING STEP STATISTICS")
            print(f"{'='*80}")
            
            value_losses = [s['value_loss'] for s in steps]
            policy_losses = [s['policy_loss'] for s in steps]
            total_losses = [s['total_loss'] for s in steps]
            
            print(f"\nValue Loss:  min={min(value_losses):.6f}, max={max(value_losses):.6f}, " +
                  f"avg={statistics.mean(value_losses):.6f}, std={statistics.stdev(value_losses) if len(value_losses) > 1 else 0:.6f}")
            print(f"Policy Loss: min={min(policy_losses):.6f}, max={max(policy_losses):.6f}, " +
                  f"avg={statistics.mean(policy_losses):.6f}, std={statistics.stdev(policy_losses) if len(policy_losses) > 1 else 0:.6f}")
            print(f"Total Loss:  min={min(total_losses):.6f}, max={max(total_losses):.6f}, " +
                  f"avg={statistics.mean(total_losses):.6f}, std={statistics.stdev(total_losses) if len(total_losses) > 1 else 0:.6f}")
            print(f"\nTotal training steps: {len(steps)}")


def list_sessions():
    """List all available training sessions."""
    logs_dir = Path("logs")
    if not logs_dir.exists():
        print("No logs directory found")
        return
    
    sessions = sorted([d for d in logs_dir.iterdir() if d.is_dir()], reverse=True)
    
    if not sessions:
        print("No training sessions found")
        return
    
    print(f"\nAvailable sessions:\n")
    for i, session in enumerate(sessions, 1):
        phase_file = session / "phase_summary.json"
        num_phases = 0
        if phase_file.exists():
            with open(phase_file) as f:
                num_phases = len(json.load(f))
        
        print(f"{i}. {session.name} ({num_phases} phases)")


if __name__ == "__main__":
    import sys
    
    if len(sys.argv) > 1:
        session_dir = sys.argv[1]
        analyze_session(session_dir)
    else:
        list_sessions()
        
        # Analyze the most recent session
        logs_dir = Path("logs")
        if logs_dir.exists():
            sessions = sorted([d for d in logs_dir.iterdir() if d.is_dir()], reverse=True)
            if sessions:
                print(f"\nAnalyzing most recent session...")
                analyze_session(sessions[0])
