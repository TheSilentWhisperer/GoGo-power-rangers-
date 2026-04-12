import json
import os
from datetime import datetime
from pathlib import Path


class MetricsLogger:
    """Logs training metrics (loss, win rate, validation) to JSON files organized by timestamp."""
    
    def __init__(self, log_dir: str = "logs"):
        self.log_dir = Path(log_dir)
        self.log_dir.mkdir(exist_ok=True)
        
        # Create timestamped session directory
        self.session_id = datetime.now().strftime("%Y%m%d_%H%M%S")
        self.session_dir = self.log_dir / self.session_id
        self.session_dir.mkdir(exist_ok=True)
        
        # Metrics storage
        self.phase_metrics = {}  # Per-phase aggregates
        self.training_steps = []  # Per-step training loss
        self.validation_data = []  # Validation results
        
        print(f"[LOGGER] Session logs will be saved to {self.session_dir}")
    
    def log_training_step(self, step: int, value_loss: float, policy_loss: float, total_loss: float, lr: float):
        """Log a single training step."""
        self.training_steps.append({
            "step": step,
            "value_loss": float(value_loss),
            "policy_loss": float(policy_loss),
            "total_loss": float(total_loss),
            "learning_rate": float(lr)
        })
    
    def log_phase_end(self, phase: int, total_steps: int, avg_value_loss: float, avg_policy_loss: float, 
                      avg_total_loss: float, experiences_count: int, duration_sec: float):
        """Log metrics at end of training phase."""
        self.phase_metrics[phase] = {
            "phase": phase,
            "total_steps": total_steps,
            "avg_value_loss": float(avg_value_loss),
            "avg_policy_loss": float(avg_policy_loss),
            "avg_total_loss": float(avg_total_loss),
            "experiences_count": experiences_count,
            "duration_sec": float(duration_sec)
        }
        
        print(f"[LOGGER] Phase {phase}: value_loss={avg_value_loss:.4f}, policy_loss={avg_policy_loss:.4f}, " +
              f"total_loss={avg_total_loss:.4f}, duration={duration_sec:.1f}s")
    
    def log_validation_game(self, game_num: int, current_model_won: bool, score_diff: float, 
                           move_count: int, notes: str = ""):
        """Log result of a validation game (current model vs previous best)."""
        self.validation_data.append({
            "game": game_num,
            "current_model_won": bool(current_model_won),
            "score_diff": float(score_diff),
            "move_count": int(move_count),
            "notes": notes
        })
    
    def log_validation_summary(self, phase: int, win_rate: float, avg_score_diff: float, total_games: int):
        """Log validation summary for a phase."""
        validation_summary = {
            "phase": phase,
            "win_rate": float(win_rate),
            "win_rate_pct": f"{win_rate*100:.1f}%",
            "avg_score_diff": float(avg_score_diff),
            "total_games": total_games,
            "timestamp": datetime.now().isoformat()
        }
        
        # Append to validation summary file
        summary_path = self.session_dir / "validation_summary.jsonl"
        with open(summary_path, "a") as f:
            f.write(json.dumps(validation_summary) + "\n")
        
        print(f"[LOGGER] Validation phase {phase}: win_rate={win_rate*100:.1f}% ({int(win_rate*total_games)}/{total_games}), " +
              f"avg_score_diff={avg_score_diff:.2f}")
    
    def save_session(self):
        """Save all collected metrics to files."""
        # Save training steps
        if self.training_steps:
            steps_file = self.session_dir / "training_steps.json"
            with open(steps_file, "w") as f:
                json.dump(self.training_steps, f, indent=2)
            print(f"[LOGGER] Saved {len(self.training_steps)} training steps to {steps_file}")
        
        # Save phase summary
        if self.phase_metrics:
            phases_file = self.session_dir / "phase_summary.json"
            with open(phases_file, "w") as f:
                json.dump(list(self.phase_metrics.values()), f, indent=2)
            print(f"[LOGGER] Saved {len(self.phase_metrics)} phase summaries to {phases_file}")
        
        # Save validation games (already appended incrementally to validation_summary.jsonl)
        print(f"[LOGGER] Session saved to {self.session_dir}")
    
    def print_session_summary(self):
        """Print summary statistics for the session."""
        if not self.phase_metrics:
            return
        
        phases = sorted(self.phase_metrics.values(), key=lambda x: x["phase"])
        print("\n" + "="*80)
        print(f"SESSION SUMMARY ({self.session_id})")
        print("="*80)
        print(f"{'Phase':<8} {'Avg Value Loss':<18} {'Avg Policy Loss':<18} {'Avg Total Loss':<18} {'Exp':<8} {'Time(s)':<10}")
        print("-"*80)
        for m in phases:
            print(f"{m['phase']:<8} {m['avg_value_loss']:<18.4f} {m['avg_policy_loss']:<18.4f} " +
                  f"{m['avg_total_loss']:<18.4f} {m['experiences_count']:<8} {m['duration_sec']:<10.1f}")
        print("="*80 + "\n")


# Global logger instance
_logger = None


def get_logger() -> MetricsLogger:
    """Get or create the global metrics logger."""
    global _logger
    if _logger is None:
        _logger = MetricsLogger()
    return _logger


def init_logger(log_dir: str = "logs") -> MetricsLogger:
    """Initialize a new logger."""
    global _logger
    _logger = MetricsLogger(log_dir)
    return _logger
