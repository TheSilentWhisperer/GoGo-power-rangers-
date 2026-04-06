import torch
import torch.nn as nn
import torch.nn.functional as F

class MaskedSoftmax(nn.Module):
    def __init__(self, dim):
        super(MaskedSoftmax, self).__init__()
        self.dim = dim

    def forward(self, input, mask):
        masked_input = input.masked_fill(mask == 0, float('-inf'))
        return F.softmax(masked_input, dim=self.dim)

class ResidualBlock(nn.Module):
    def __init__(self, channels):
        super(ResidualBlock, self).__init__()
        self.conv1 = nn.Conv2d(channels, channels, kernel_size=3, padding=1)
        self.ln1 = nn.GroupNorm(1, channels)  # GroupNorm with 1 group = LayerNorm
        self.conv2 = nn.Conv2d(channels, channels, kernel_size=3, padding=1)
        self.ln2 = nn.GroupNorm(1, channels)
        # He initialization for Conv2d with ReLU
        nn.init.kaiming_normal_(self.conv1.weight, mode='fan_out', nonlinearity='relu')
        nn.init.kaiming_normal_(self.conv2.weight, mode='fan_out', nonlinearity='relu')

    def forward(self, x):
        residual = x
        out = F.relu(self.ln1(self.conv1(x)))
        out = self.ln2(self.conv2(out))
        out += residual
        out = F.relu(out)
        return out


class Gogo81(nn.Module):
    def __init__(self, model_channels: int = 96, num_res_blocks: int = 15, input_planes: int = 4):
        super(Gogo81, self).__init__()
        # Encoder: increased capacity for 9x9 with moderate depth
        self.model_channels = model_channels
        self.num_res_blocks = num_res_blocks

        # Create input projection at init so optimizer sees all params.
        # Default `input_planes` matches the repo's `NewGame(..., 4, ...)` history length.
        self.input_conv = nn.Conv2d(input_planes, self.model_channels, kernel_size=3, padding=1)
        self.ln_input = nn.GroupNorm(1, self.model_channels)  # LayerNorm for input

        # Residual trunk
        self.res_blocks = nn.Sequential(*[ResidualBlock(self.model_channels) for _ in range(self.num_res_blocks)])

        # Policy head (maps trunk features -> per-loc logits + pass/resign)
        # Increased from 2 to 16 channels to better preserve spatial structure
        self.policy_conv = nn.Conv2d(self.model_channels, 16, kernel_size=1)
        self.ln_policy = nn.GroupNorm(1, 16)  # LayerNorm for policy
        self.policy_fc = nn.Linear(16*9*9 + 1 + 1, 83)

        # Value head
        self.value_conv = nn.Conv2d(self.model_channels, 1, kernel_size=1)
        self.ln_value = nn.GroupNorm(1, 1)  # LayerNorm for value
        self.value_fc1 = nn.Linear(9*9 + 1 + 1, 64)
        self.value_fc2 = nn.Linear(64, 1)

        # He initialization for hidden layers
        nn.init.kaiming_normal_(self.input_conv.weight, mode='fan_out', nonlinearity='relu')
        nn.init.kaiming_normal_(self.policy_conv.weight, mode='fan_out', nonlinearity='relu')
        nn.init.kaiming_normal_(self.value_conv.weight, mode='fan_out', nonlinearity='relu')
        nn.init.kaiming_normal_(self.value_fc1.weight, mode='fan_out', nonlinearity='relu')
        # Initialize output layers with small weights for stability (softmax/tanh outputs)
        nn.init.uniform_(self.policy_fc.weight, -0.01, 0.01)
        nn.init.uniform_(self.value_fc2.weight, -0.01, 0.01)

    def forward(self, batch_inputs):
        B = batch_inputs["boards"].shape[0]
        x = batch_inputs["boards"]

        # Create lazy input projection if needed (handles variable history length)
        if self.input_conv is None or self.input_conv.in_channels != x.shape[1]:
            in_ch = x.shape[1]
            self.input_conv = nn.Conv2d(in_ch, self.model_channels, kernel_size=3, padding=1)
            self.ln_input = nn.GroupNorm(1, self.model_channels)
            # Move newly created modules to the same device as the input
            device = x.device
            self.input_conv.to(device)
            self.ln_input.to(device)

        # Encoder
        x = F.relu(self.ln_input(self.input_conv(x)))
        x = self.res_blocks(x)

        # Policy head
        p = F.relu(self.ln_policy(self.policy_conv(x)))
        p = p.view(B, -1)
        p = torch.cat([p, batch_inputs["player_colors"], batch_inputs["pass_counts"]], dim=1)
        # Raw policy logits; softmax + masking is handled by the inference/training code.
        policy_logits = self.policy_fc(p)

        # Value head
        v = F.relu(self.ln_value(self.value_conv(x)))
        v = v.view(B, -1)
        v = torch.cat([v, batch_inputs["player_colors"], batch_inputs["pass_counts"]], dim=1)
        v = F.relu(self.value_fc1(v))
        value = torch.tanh(self.value_fc2(v))

        return batch_inputs["request_ids"], value, policy_logits