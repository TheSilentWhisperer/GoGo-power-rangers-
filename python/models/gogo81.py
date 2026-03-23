import torch.nn as nn
import torch.nn.functional as F
import torch

class MaskedSoftmax(nn.Module):
    def __init__(self, dim):
        super(MaskedSoftmax, self).__init__()
        self.dim = dim

    def forward(self, input, mask):
        # Apply the mask to the input
        masked_input = input.masked_fill(mask == 0, float('-inf'))
        # Compute softmax on the masked input
        return F.softmax(masked_input, dim=self.dim)

class ResidualBlock(nn.Module):
    def __init__(self, in_channels, out_channels):
        super(ResidualBlock, self).__init__()

        self.conv1 = nn.Conv2d(in_channels, out_channels, kernel_size=3, stride=1, padding=1)
        self.bn1 = nn.BatchNorm2d(out_channels)

        self.conv2 = nn.Conv2d(out_channels, out_channels, kernel_size=3, stride=1, padding=1)
        self.bn2 = nn.BatchNorm2d(out_channels)

        self.skip_connection = nn.Conv2d(in_channels, out_channels, kernel_size=1, stride=1) if in_channels != out_channels else None

    def forward(self, x):
        residual = x if self.skip_connection is None else self.skip_connection(x)

        x = self.conv1(x)
        x = self.bn1(x)
        x = F.relu(x)

        x = self.conv2(x)
        x = self.bn2(x)
        x += residual  # Skip connection
        x = F.relu(x)

        return x

class Gogo81(nn.Module):
    def __init__(self):
        super(Gogo81, self).__init__()

        self.encoder = nn.Sequential(
            ResidualBlock(4, 4),
            ResidualBlock(4, 4),
            ResidualBlock(4, 4),
            ResidualBlock(4, 1)
        )

        self.value_head = nn.Sequential(
            # board_encoding (81) + player_color (1) + pass_count (1)
            nn.Linear(81 + 1 + 1, 128),
            nn.ReLU(),  
            nn.Linear(128, 1),
            nn.Tanh()  # Output value in range [-1, 1]
        )

        self.policy_head = nn.Sequential(
            # board_encoding (81) + player_color (1) + pass_count (1)
            nn.Linear(81 + 1 + 1, 128),
            nn.ReLU(),
            nn.Linear(128, 83)  # 81 for board positions + 1 for pass + 1 for resign
        )   



    def forward(self, input):
        # input["board"] shape: (batch_size, 4, 9, 9)
        # input["player_color"] shape: (batch_size, 1)
        # input["pass_count"] shape: (batch_size, 1)

        board_encoding = self.encoder(input["board"])  # Shape: (batch_size, 1, 9, 9)
        board_encoding = board_encoding.view(board_encoding.size(0), -1)  # Shape: (batch_size, 81)
        player_color = input["player_color"]  # Shape: (batch_size, 1)
        pass_count = input["pass_count"]  # Shape: (batch_size, 1)
        combined_input = torch.cat([board_encoding, player_color, pass_count], dim=1)  # Shape: (batch_size, 83)

        values = self.value_head(combined_input)  # Shape: (batch_size, 1)
        policy = self.policy_head(combined_input)  # Shape: (batch_size, 82)
        priors = MaskedSoftmax(dim=1)(policy, input["legal_masks"])  # Shape: (batch_size, 82)

        return input["request_ids"], values, priors


