#!/usr/bin/env python3
"""Train a small policy/value network from az_export samples.

This is intentionally a compact baseline trainer. It consumes the sparse JSONL
format emitted by //cmd/az_export:az_export and writes a PyTorch checkpoint with
the action vocabulary and input size in its metadata.
"""

import argparse
import importlib.util
import json
import os
import sys
from pathlib import Path

# Bazel runfiles put the Go repo root on sys.path. This repo has a top-level
# cmd/ directory, which can shadow Python's stdlib cmd module when torch imports
# pdb. Remove only that runfiles root before importing third-party packages.
for entry in list(sys.path):
	path = Path(entry)
	if (path / "cmd" / "az_train_torch").exists() and (path / "internal").exists():
		sys.path.remove(entry)
stdlib_cmd = Path(os.__file__).resolve().parent / "cmd.py"
if stdlib_cmd.exists():
	spec = importlib.util.spec_from_file_location("cmd", stdlib_cmd)
	cmd_module = importlib.util.module_from_spec(spec)
	assert spec.loader is not None
	spec.loader.exec_module(cmd_module)
	sys.modules["cmd"] = cmd_module

try:
    import torch
    from torch import nn
    from torch.utils.data import DataLoader, Dataset
except ModuleNotFoundError as exc:
    print("PyTorch is required for az_train_torch. Install torch in the Bazel Python runtime.", file=sys.stderr)
    raise SystemExit(2) from exc


class JsonlDataset(Dataset):
    def __init__(self, path: Path, input_size: int, action_count: int):
        self.rows = []
        self.input_size = input_size
        self.action_count = action_count
        with path.open("r", encoding="utf-8") as handle:
            for line in handle:
                if line.strip():
                    self.rows.append(json.loads(line))

    def __len__(self):
        return len(self.rows)

    def __getitem__(self, index):
        row = self.rows[index]
        features = list(row["encoding"])
        if len(features) < self.input_size:
            features.extend([0.0] * (self.input_size - len(features)))
        else:
            features = features[: self.input_size]
        policy = torch.zeros(self.action_count, dtype=torch.float32)
        for target in row["policyTargets"]:
            policy[int(target["actionIndex"])] = float(target["probability"])
        legal = torch.zeros(self.action_count, dtype=torch.float32)
        for action_index in row["legalActionIndices"]:
            legal[int(action_index)] = 1.0
        return {
            "features": torch.tensor(features, dtype=torch.float32),
            "policy": policy,
            "legal": legal,
            "value": torch.tensor([float(row["value"])], dtype=torch.float32),
        }


class PolicyValueNet(nn.Module):
    def __init__(self, input_size: int, action_count: int, hidden_size: int):
        super().__init__()
        self.body = nn.Sequential(
            nn.Linear(input_size, hidden_size),
            nn.ReLU(),
            nn.Linear(hidden_size, hidden_size),
            nn.ReLU(),
        )
        self.policy = nn.Linear(hidden_size, action_count)
        self.value = nn.Sequential(nn.Linear(hidden_size, 1), nn.Tanh())

    def forward(self, features):
        hidden = self.body(features)
        return self.policy(hidden), self.value(hidden)


class HexPolicyValueNet(nn.Module):
    def __init__(self, input_size: int, action_count: int, hidden_size: int, observation_shape):
        super().__init__()
        if len(observation_shape) != 3:
            raise ValueError(f"hex architecture requires [global, hex, per_hex] shape, got {observation_shape}")
        self.input_size = input_size
        self.global_size = int(observation_shape[0])
        self.hex_count = int(observation_shape[1])
        self.per_hex_size = int(observation_shape[2])
        if self.global_size + self.hex_count * self.per_hex_size > input_size:
            raise ValueError("observation shape exceeds input size")
        self.hex_encoder = nn.Sequential(
            nn.Linear(self.per_hex_size, hidden_size),
            nn.ReLU(),
            nn.Linear(hidden_size, hidden_size),
            nn.ReLU(),
        )
        self.global_encoder = nn.Sequential(
            nn.Linear(self.global_size, hidden_size),
            nn.ReLU(),
        )
        self.body = nn.Sequential(
            nn.Linear(hidden_size * 3, hidden_size),
            nn.ReLU(),
            nn.Linear(hidden_size, hidden_size),
            nn.ReLU(),
        )
        self.policy = nn.Linear(hidden_size, action_count)
        self.value = nn.Sequential(nn.Linear(hidden_size, 1), nn.Tanh())

    def forward(self, features):
        global_features = features[:, : self.global_size]
        board_start = self.global_size
        board_end = board_start + self.hex_count * self.per_hex_size
        board = features[:, board_start:board_end].reshape(-1, self.hex_count, self.per_hex_size)
        encoded_hexes = self.hex_encoder(board)
        mean_pool = encoded_hexes.mean(dim=1)
        max_pool = encoded_hexes.max(dim=1).values
        global_hidden = self.global_encoder(global_features)
        hidden = self.body(torch.cat([global_hidden, mean_pool, max_pool], dim=1))
        return self.policy(hidden), self.value(hidden)


def build_model(architecture: str, input_size: int, action_count: int, hidden_size: int, observation_shape):
    if architecture == "hex":
        return HexPolicyValueNet(input_size, action_count, hidden_size, observation_shape)
    if architecture == "mlp":
        return PolicyValueNet(input_size, action_count, hidden_size)
    raise ValueError(f"unknown architecture: {architecture}")


def policy_loss(logits, policy_targets, legal_mask):
    masked = logits.masked_fill(legal_mask <= 0, -1e9)
    log_probs = torch.log_softmax(masked, dim=1)
    return -(policy_targets * log_probs).sum(dim=1).mean()


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--samples", required=True)
    parser.add_argument("--manifest", required=True)
    parser.add_argument("--vocab", default="")
    parser.add_argument("--output", required=True)
    parser.add_argument("--epochs", type=int, default=5)
    parser.add_argument("--batch_size", type=int, default=64)
    parser.add_argument("--hidden_size", type=int, default=128)
    parser.add_argument("--lr", type=float, default=1e-3)
    parser.add_argument("--architecture", choices=["hex", "mlp"], default="hex")
    args = parser.parse_args()

    manifest = json.loads(Path(args.manifest).read_text(encoding="utf-8"))
    vocab_path = Path(args.vocab) if args.vocab else Path(manifest["vocabPath"])
    action_vocab = json.loads(vocab_path.read_text(encoding="utf-8"))
    dataset = JsonlDataset(Path(args.samples), int(manifest["encodingSize"]), int(manifest["actionCount"]))
    if len(dataset) == 0:
        raise SystemExit("empty dataset")
    loader = DataLoader(dataset, batch_size=args.batch_size, shuffle=True)
    observation_shape = manifest.get("observationShape", [])
    architecture = args.architecture
    if architecture == "hex" and len(observation_shape) != 3:
        architecture = "mlp"
    net = build_model(architecture, dataset.input_size, dataset.action_count, args.hidden_size, observation_shape)
    optimizer = torch.optim.Adam(net.parameters(), lr=args.lr)
    for epoch in range(args.epochs):
        total = 0.0
        for batch in loader:
            logits, value = net(batch["features"])
            loss = policy_loss(logits, batch["policy"], batch["legal"]) + nn.functional.mse_loss(value, batch["value"])
            optimizer.zero_grad()
            loss.backward()
            optimizer.step()
            total += float(loss.detach())
        print(json.dumps({"epoch": epoch + 1, "loss": total / max(1, len(loader))}), file=sys.stderr)
    torch.save(
        {
            "state_dict": net.state_dict(),
            "input_size": dataset.input_size,
            "action_count": dataset.action_count,
            "hidden_size": args.hidden_size,
            "architecture": architecture,
            "observation_schema": manifest.get("observationSchema", ""),
            "observation_shape": manifest.get("observationShape", []),
            "manifest": manifest,
            "action_vocab": action_vocab,
        },
        args.output,
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
