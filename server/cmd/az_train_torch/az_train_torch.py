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
import random
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
    from torch.utils.data import DataLoader, IterableDataset
except ModuleNotFoundError as exc:
    print("PyTorch is required for az_train_torch. Install torch in the Bazel Python runtime.", file=sys.stderr)
    raise SystemExit(2) from exc


class JsonlDataset(IterableDataset):
    def __init__(self, path: Path, input_size: int, action_count: int, shuffle_buffer: int = 0, seed: int = 1):
        self.path = path
        self.input_size = input_size
        self.action_count = action_count
        self.shuffle_buffer = shuffle_buffer
        self.seed = seed
        self.iteration = 0

    def __iter__(self):
        if self.shuffle_buffer <= 1:
            with self.path.open("r", encoding="utf-8") as handle:
                for line in handle:
                    if line.strip():
                        yield self.row_to_sample(json.loads(line))
            return
        iteration = self.iteration
        self.iteration += 1
        rng = random.Random(self.seed + iteration)
        buffer = []
        with self.path.open("r", encoding="utf-8") as handle:
            for line in handle:
                if not line.strip():
                    continue
                buffer.append(line)
                if len(buffer) >= self.shuffle_buffer:
                    index = rng.randrange(len(buffer))
                    selected = buffer[index]
                    buffer[index] = buffer[-1]
                    buffer.pop()
                    yield self.row_to_sample(json.loads(selected))
        rng.shuffle(buffer)
        for line in buffer:
            yield self.row_to_sample(json.loads(line))

    def row_to_sample(self, row):
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


def action_key(action):
    return json.dumps(action, sort_keys=True, separators=(",", ":"))


def load_init_checkpoint(
    net,
    checkpoint,
    expected,
    observation_shape,
    action_vocab,
    allow_action_mismatch,
    new_action_logit,
):
    strict_expected = dict(expected)
    if allow_action_mismatch:
        strict_expected.pop("action_count", None)
    for key, value in strict_expected.items():
        if checkpoint.get(key) != value:
            raise SystemExit(f"init checkpoint {key}={checkpoint.get(key)!r} does not match expected {value!r}")
    if checkpoint.get("observation_shape", []) != observation_shape:
        raise SystemExit(
            f"init checkpoint observation_shape={checkpoint.get('observation_shape', [])!r} "
            f"does not match expected {observation_shape!r}"
        )
    if checkpoint.get("action_count") == expected["action_count"]:
        net.load_state_dict(checkpoint["state_dict"])
        return {"mode": "strict", "matched_policy_actions": expected["action_count"]}
    if not allow_action_mismatch:
        raise SystemExit(
            f"init checkpoint action_count={checkpoint.get('action_count')} "
            f"does not match expected {expected['action_count']}"
        )

    current_state = net.state_dict()
    source_state = checkpoint["state_dict"]
    copied_tensors = 0
    for name, tensor in source_state.items():
        if name.startswith("policy."):
            continue
        if name in current_state and current_state[name].shape == tensor.shape:
            current_state[name] = tensor
            copied_tensors += 1

    if "policy.weight" in current_state and "policy.bias" in current_state:
        current_state["policy.weight"].zero_()
        current_state["policy.bias"].fill_(new_action_logit)

    old_vocab = checkpoint.get("action_vocab", [])
    old_index = {action_key(action): index for index, action in enumerate(old_vocab)}
    matched_policy_actions = 0
    if "policy.weight" in current_state and "policy.weight" in source_state:
        for new_index, action in enumerate(action_vocab):
            old_action_index = old_index.get(action_key(action))
            if old_action_index is None:
                continue
            current_state["policy.weight"][new_index] = source_state["policy.weight"][old_action_index]
            current_state["policy.bias"][new_index] = source_state["policy.bias"][old_action_index]
            matched_policy_actions += 1

    net.load_state_dict(current_state)
    return {
        "mode": "transfer_action_mismatch",
        "source_action_count": checkpoint.get("action_count"),
        "target_action_count": expected["action_count"],
        "matched_policy_actions": matched_policy_actions,
        "copied_non_policy_tensors": copied_tensors,
        "new_action_logit": new_action_logit,
    }


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
    parser.add_argument("--init_checkpoint", default="", help="optional compatible checkpoint to initialize from")
    parser.add_argument(
        "--init_allow_action_mismatch",
        action="store_true",
        help="transfer compatible layers and matching policy rows when init checkpoint action_count differs",
    )
    parser.add_argument(
        "--init_new_action_logit",
        type=float,
        default=-8.0,
        help="initial policy bias for unmatched actions when transferring across action vocabularies",
    )
    parser.add_argument("--shuffle_buffer", type=int, default=0, help="bounded streaming shuffle buffer; 0 disables")
    parser.add_argument("--seed", type=int, default=1, help="random seed for bounded shuffle")
    args = parser.parse_args()

    manifest = json.loads(Path(args.manifest).read_text(encoding="utf-8"))
    vocab_path = Path(args.vocab) if args.vocab else Path(manifest["vocabPath"])
    action_vocab = json.loads(vocab_path.read_text(encoding="utf-8"))
    dataset = JsonlDataset(
        Path(args.samples),
        int(manifest["encodingSize"]),
        int(manifest["actionCount"]),
        shuffle_buffer=args.shuffle_buffer,
        seed=args.seed,
    )
    sample_count = int(manifest.get("sampleCount", 0))
    if sample_count == 0:
        raise SystemExit("empty dataset")
    loader = DataLoader(dataset, batch_size=args.batch_size)
    observation_shape = manifest.get("observationShape", [])
    architecture = args.architecture
    if architecture == "hex" and len(observation_shape) != 3:
        architecture = "mlp"
    net = build_model(architecture, dataset.input_size, dataset.action_count, args.hidden_size, observation_shape)
    init_checkpoint_path = Path(args.init_checkpoint) if args.init_checkpoint else None
    init_checkpoint = None
    init_report = {"mode": "none"}
    if init_checkpoint_path is not None:
        init_checkpoint = torch.load(init_checkpoint_path, map_location="cpu")
        expected = {
            "input_size": dataset.input_size,
            "action_count": dataset.action_count,
            "hidden_size": args.hidden_size,
            "architecture": architecture,
        }
        init_report = load_init_checkpoint(
            net,
            init_checkpoint,
            expected,
            observation_shape,
            action_vocab,
            args.init_allow_action_mismatch,
            args.init_new_action_logit,
        )
        print(json.dumps({"initCheckpoint": str(init_checkpoint_path), **init_report}), file=sys.stderr)
    optimizer = torch.optim.Adam(net.parameters(), lr=args.lr)
    for epoch in range(args.epochs):
        total = 0.0
        batches = 0
        for batch in loader:
            logits, value = net(batch["features"])
            loss = policy_loss(logits, batch["policy"], batch["legal"]) + nn.functional.mse_loss(value, batch["value"])
            optimizer.zero_grad()
            loss.backward()
            optimizer.step()
            total += float(loss.detach())
            batches += 1
        print(json.dumps({"epoch": epoch + 1, "loss": total / max(1, batches), "batches": batches}), file=sys.stderr)
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
            "init_checkpoint": str(init_checkpoint_path) if init_checkpoint_path is not None else "",
            "init_report": init_report,
            "shuffle_buffer": args.shuffle_buffer,
            "seed": args.seed,
        },
        args.output,
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
