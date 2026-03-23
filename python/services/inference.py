import gen.proto.remote_trainer_pb2 as remote_trainer_pb2
import gen.proto.remote_trainer_pb2_grpc as remote_trainer_pb2_grpc
from google.protobuf import empty_pb2
import torch 
import queue
import traceback

class InferenceClient(remote_trainer_pb2_grpc.PositionEvaluatorServicer):
    def __init__(self, model, device, batch_size):
        super().__init__()
        self.model = model
        self.device = device
        self.batch_size = batch_size
        self.to_evaluate = []
        self.evaluated = []

        self.model.to(self.device)

    def ResetServer(self, request, context): 
        self.to_evaluate.clear()
        self.evaluated.clear()
        return empty_pb2.Empty()

    def get_inputs(self):
        history_length, height, width = self.to_evaluate[0].history_length, self.to_evaluate[0].height, self.to_evaluate[0].width
        request_ids = [req.request_id for req in self.to_evaluate] # Shape: (batch_size,)
        input_boards = torch.cat([torch.tensor(req.flattened_board_history, dtype=torch.float32).view(1, history_length, height, width) for req in self.to_evaluate], dim=0).to(self.device) # Shape: (batch_size, history_length, height, width)
        pass_counts = torch.tensor([req.enemy_passed for req in self.to_evaluate], dtype=torch.float32).unsqueeze(-1).to(self.device) # Shape: (batch_size, 1)
        legal_masks = torch.cat([torch.tensor(req.legal_actions_mask, dtype=torch.float32).view(1, height*width + 2) for req in self.to_evaluate], dim=0).to(self.device) # Shape: (batch_size, height*width + 2)
        player_colors = torch.tensor([req.black_to_play for req in self.to_evaluate], dtype=torch.float32).unsqueeze(-1).to(self.device) # Shape: (batch_size, 1)
        return {
            "request_ids": request_ids,
            "board": input_boards,
            "player_color": player_colors,
            "pass_count": pass_counts,
            "legal_masks": legal_masks
        }

    def EvaluatePosition(self, request, context):
        try:
            self.to_evaluate.append(request)
            if len(self.to_evaluate) >= self.batch_size:  # Process in batches
                batch_inputs = self.get_inputs()
                self.model.eval()  # Set model to evaluation mode
                with torch.no_grad():  # Disable gradient calculation for inference
                    request_ids, values, priors = self.model(batch_inputs)  # Run inference on the batch
                for i in range(len(request_ids)):
                    self.evaluated.append((request_ids[i], values[i], priors[i]))  # Store the evaluated results
                self.to_evaluate.clear()  # Clear the queue after processing
        except Exception as e:
            traceback.print_exc()
        return empty_pb2.Empty()

    def RetrieveEvaluation(self, request, context):
        if self.evaluated:
            request_id, value, priors = self.evaluated.pop()  # Get the next evaluated result
            response = remote_trainer_pb2.EvaluationResponse()
            response.data.request_id = request_id
            response.data.value = value
            response.data.priors.extend(priors.detach().cpu().numpy().tolist())
            return response

        return empty_pb2.Empty()