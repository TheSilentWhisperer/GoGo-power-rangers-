import grpc
from concurrent import futures
import os
from services.inference import InferenceClient
from models.gogo81 import Gogo81
import torch

import gen.proto.remote_trainer_pb2_grpc as remote_trainer_pb2_grpc

UDS_PATH = "/tmp/position_evaluator.sock"

# Remove existing socket file if it exists
if os.path.exists(UDS_PATH):
    os.remove(UDS_PATH)

# Create server
server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))

model = Gogo81()
device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
batch_size = 32  # Adjust based on your needs and hardware capabilities

remote_trainer_pb2_grpc.add_PositionEvaluatorServicer_to_server(InferenceClient(model, device, batch_size), server)

server.add_insecure_port(f'unix://{UDS_PATH}')

print(f"Starting gRPC server on UDS {UDS_PATH} ...")
server.start()

try:
    server.wait_for_termination()
except KeyboardInterrupt:
    print("Shutting down server...")