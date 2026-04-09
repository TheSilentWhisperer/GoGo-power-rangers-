import grpc
from concurrent import futures
import os
import signal
import sys
import time

# Disable output buffering so tqdm and print statements show immediately
import io
sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8', write_through=True)
sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding='utf-8', write_through=True)

from services.inference import InferenceClient
from services.training import TrainingClient
from models.gogo81 import Gogo81
import torch

import gen.proto.remote_trainer_pb2_grpc as remote_trainer_pb2_grpc

# Parse command-line arguments
ENABLE_TRAINING = "--no-training" not in sys.argv
if "--no-training" in sys.argv:
    sys.argv.remove("--no-training")

UDS_PATH = "/tmp/position_evaluator.sock"

# Remove existing socket file if it exists
if os.path.exists(UDS_PATH):
    os.remove(UDS_PATH)

# Create server with single worker since Go sends training data synchronously
# Go will now block waiting for the response, so we only need one thread
server = grpc.server(futures.ThreadPoolExecutor(max_workers=1))

# Enable GPU optimizations
if torch.cuda.is_available():
    torch.backends.cuda.matmul.allow_tf32 = True
    torch.backends.cudnn.allow_tf32 = True
    torch.backends.cudnn.benchmark = True

model = Gogo81()
device = torch.device("cuda" if torch.cuda.is_available() else "cpu")

# Move model to device
model = model.to(device)

# Load pretrained model checkpoint if it exists
model_save_path = "saves/gogo81.pt"
if os.path.exists(model_save_path):
    print(f"[INFO] Loading pretrained model from {model_save_path}")
    model.load_state_dict(torch.load(model_save_path, map_location=device))
else:
    print(f"[WARNING] No pretrained model found at {model_save_path}, using random initialization")

# Disable torch.compile - TorchInductor workers hang in gRPC server context
# Using standard forward pass for stability
# try:
#     model = torch.compile(model, mode="reduce-overhead")
# except Exception as e:
#     print(f"Error during model compilation: {e}")
#     import traceback
#     traceback.print_exc()
print("[INFO] torch.compile disabled - using standard forward pass for stability")

batch_size = 20  # Process 60 evaluations per batch for GPU efficiency

inference_client = InferenceClient(model, device, batch_size)

remote_trainer_pb2_grpc.add_PositionEvaluatorServicer_to_server(inference_client, server)

# Conditionally add training service
training_client = None
if ENABLE_TRAINING:
    training_client = TrainingClient(inference_client)
    remote_trainer_pb2_grpc.add_NetTrainerServicer_to_server(training_client, server)
    print("[INFO] Training service ENABLED")
else:
    print("[INFO] Training service DISABLED (--no-training flag set)")
server.add_insecure_port(f'unix://{UDS_PATH}')

server.start()
print(f"Server started on {UDS_PATH}")

def signal_handler(signum, frame):
    """Handle SIGINT and SIGTERM signals for graceful shutdown"""
    print(f"\nReceived signal {signum}, shutting down...")
    if training_client:
        training_client.save_logs()
    server.stop(0)  # Stop immediately
    sys.exit(0)

signal.signal(signal.SIGINT, signal_handler)
signal.signal(signal.SIGTERM, signal_handler)

try:
    # Use wait_for_termination with a timeout so Ctrl+C can interrupt
    while True:
        server.wait_for_termination(timeout=1)
except KeyboardInterrupt:
    print("\nKeyboardInterrupt caught")
    if training_client:
        training_client.save_logs()
    server.stop(0)
except Exception as e:
    print(f"Error: {e}")
    if training_client:
        training_client.save_logs()
    server.stop(0)