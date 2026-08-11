"""Reproducible local throughput probe for the exact Terra Mystica network."""

from __future__ import annotations

import argparse
import hashlib
import json
import math
from pathlib import Path
import time

import torch
from torch.nn import functional as F

from model import MODEL_CONFIGS, TerraMysticaNet, load_checkpoint, model_fingerprint, resolve_device, resolve_model_config
from schema import MANIFEST


def _synchronize(device: torch.device) -> None:
    if device.type == "cuda":
        torch.cuda.synchronize(device)
    elif device.type == "mps":
        torch.mps.synchronize()


def _percentile(values: list[float], probability: float) -> float:
    ordered = sorted(values)
    return ordered[max(0, math.ceil(probability * len(ordered)) - 1)]


def _inputs(batch_size: int, action_count: int, device: torch.device) -> tuple[torch.Tensor, ...]:
    spatial = torch.randn(
        batch_size,
        MANIFEST.spatial_features,
        MANIFEST.grid_height,
        MANIFEST.grid_width,
        device=device,
    )
    spatial[:, 0].fill_(1)
    global_features = torch.randn(batch_size, MANIFEST.global_features, device=device)
    action_features = torch.randn(batch_size, action_count, MANIFEST.action_features, device=device)
    action_hex_indices = torch.randint(
        0,
        MANIFEST.grid_height * MANIFEST.grid_width,
        (batch_size, action_count, 2),
        device=device,
    )
    action_mask = torch.ones(batch_size, action_count, dtype=torch.bool, device=device)
    return spatial, global_features, action_features, action_hex_indices, action_mask


def benchmark(
    *,
    config_name: str,
    device_name: str,
    mode: str,
    batch_size: int,
    action_count: int,
    warmup: int,
    iterations: int,
    seed: int,
    checkpoint: str | None = None,
) -> dict[str, object]:
    if min(batch_size, action_count, iterations) <= 0 or warmup < 0:
        raise ValueError("batch size, actions, and iterations must be positive; warmup must be non-negative")
    torch.manual_seed(seed)
    device = resolve_device(device_name)
    resolved_name, config = resolve_model_config(config_name, checkpoint)
    model = TerraMysticaNet(config).to(device)
    checkpoint_sha256 = ""
    model_id = ""
    if checkpoint:
        payload = load_checkpoint(checkpoint, model)
        model_id = str(payload.get("model_id") or "")
        expected_model_id = "model-" + model_fingerprint(model)
        if model_id != expected_model_id:
            raise ValueError(f"checkpoint model identity mismatch: {model_id} != {expected_model_id}")
        checkpoint_sha256 = hashlib.sha256(Path(checkpoint).read_bytes()).hexdigest()
    inputs = _inputs(batch_size, action_count, device)
    optimizer = torch.optim.AdamW(model.parameters()) if mode == "training" else None
    policy_target = torch.arange(batch_size, device=device) % action_count
    value_target = torch.linspace(-1, 1, batch_size, device=device)

    def step() -> None:
        if optimizer is None:
            with torch.inference_mode():
                model(*inputs)
            return
        logits, values = model(*inputs)
        loss = F.cross_entropy(logits, policy_target) + F.mse_loss(values, value_target)
        optimizer.zero_grad(set_to_none=True)
        loss.backward()
        optimizer.step()

    model.train(mode == "training")
    for _ in range(warmup):
        step()
    _synchronize(device)
    durations_ms: list[float] = []
    for _ in range(iterations):
        started = time.perf_counter()
        step()
        _synchronize(device)
        durations_ms.append((time.perf_counter() - started) * 1000)

    median_ms = _percentile(durations_ms, 0.5)
    parameters = sum(parameter.numel() for parameter in model.parameters())
    return {
        "format_version": 1,
        "model_config": resolved_name,
        "channels": model.config.channels,
        "residual_blocks": model.config.residual_blocks,
        "parameters": parameters,
        "parameter_bytes_fp32": parameters * 4,
        "device": str(device),
        "dtype": "float32",
        "torch_threads": torch.get_num_threads(),
        "seed": seed,
        "model_id": model_id,
        "checkpoint_sha256": checkpoint_sha256,
        "mode": mode,
        "batch_size": batch_size,
        "legal_actions_per_position": action_count,
        "warmup": warmup,
        "iterations": iterations,
        "median_ms": median_ms,
        "p95_ms": _percentile(durations_ms, 0.95),
        "positions_per_second_at_median": batch_size * 1000 / median_ms,
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--model-config", choices=("auto", *MODEL_CONFIGS), default="debug")
    parser.add_argument("--checkpoint")
    parser.add_argument("--device", default="cpu", help="cpu, cuda, mps, or auto")
    parser.add_argument("--mode", choices=("inference", "training"), default="inference")
    parser.add_argument("--batch-size", type=int, default=8)
    parser.add_argument("--actions", type=int, default=128)
    parser.add_argument("--warmup", type=int, default=5)
    parser.add_argument("--iterations", type=int, default=20)
    parser.add_argument("--seed", type=int, default=0)
    parser.add_argument("--torch-threads", type=int, default=0, help="zero keeps the PyTorch default")
    args = parser.parse_args()
    if args.torch_threads < 0:
        parser.error("--torch-threads must be non-negative")
    if args.torch_threads:
        torch.set_num_threads(args.torch_threads)
    print(json.dumps(benchmark(
        config_name=args.model_config,
        device_name=args.device,
        mode=args.mode,
        batch_size=args.batch_size,
        action_count=args.actions,
        warmup=args.warmup,
        iterations=args.iterations,
        seed=args.seed,
        checkpoint=args.checkpoint,
    ), sort_keys=True))


if __name__ == "__main__":
    main()
