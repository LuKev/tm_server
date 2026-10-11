"""Bounded optimizer stability screen, never a model-strength comparison."""

from __future__ import annotations

import argparse
import gc
import json
import math
import os
from pathlib import Path
import tempfile
import time
from typing import Any

import torch

from learner import DIAGNOSTIC_SPLIT, ReplayWindow, load_replay, select_replay_paths, train
from model import MODEL_CONFIGS, TerraMysticaNet, model_fingerprint, resolve_device


def stability_probe(
    replay: ReplayWindow, *, model_config: str = "main", device: str = "mps",
    learning_rates: tuple[float, ...] = (1e-4, 3e-4, 1e-3), steps: int = 20,
    batch_size: int = 32, seed: int = 731, model_seed: int = 937,
) -> dict[str, Any]:
    if not 1 <= steps <= 100 or not 1 <= batch_size <= 128 or not 1 <= len(learning_rates) <= 8:
        raise ValueError("screen is bounded to 100 steps, batch 128 and eight learning rates")
    if not all(math.isfinite(rate) and rate > 0 for rate in learning_rates):
        raise ValueError("learning rates must be finite and positive")
    resolved_device = resolve_device(device)
    results = []
    for learning_rate in learning_rates:
        torch.manual_seed(model_seed)
        model = TerraMysticaNet(MODEL_CONFIGS[model_config])
        initial_model_id = "random-" + model_fingerprint(model)
        started = time.perf_counter()
        result: dict[str, Any] = {"learning_rate": learning_rate, "initial_model_id": initial_model_id}
        try:
            metrics, optimizer_state, learner_state = train(
                model, replay, steps=steps, batch_size=batch_size, learning_rate=learning_rate,
                weight_decay=1e-4, seed=seed, device=resolved_device, optimizer_name="sgd",
                momentum=.9, collect_telemetry=True,
            )
            result.update(status="finite", diagnostics=metrics,
                          final_model_id="model-" + model_fingerprint(model),
                          completed_steps=learner_state["completed_steps"])
            del optimizer_state, learner_state
        except FloatingPointError as error:
            result.update(status="non_finite", error=str(error))
        result["elapsed_seconds"] = time.perf_counter() - started
        results.append(result)
        del model
        gc.collect()
    return {
        "purpose": "bounded development optimizer stability screen; not optimality or strength evidence",
        "model_config": model_config, "device": str(resolved_device), "dtype": "float32",
        "torch_version": str(torch.__version__), "torch_threads": torch.get_num_threads(),
        "optimizer": {"name": "sgd", "momentum": .9, "weight_decay": 1e-4},
        "seed": seed, "model_seed": model_seed, "steps": steps, "batch_size": batch_size,
        "training_plies": replay.training_ply_count, "heldout_plies": replay.ply_count - replay.training_ply_count,
        "examples_per_training_ply": steps * batch_size / replay.training_ply_count,
        "replay_identity": replay.identity(), "diagnostic_split": DIAGNOSTIC_SPLIT if replay.holdout else "disabled",
        "runs": results,
    }


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", action="append", required=True)
    parser.add_argument("--engine-commit", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--model-config", choices=MODEL_CONFIGS, default="main")
    parser.add_argument("--device", default="mps")
    parser.add_argument("--learning-rates", type=float, nargs="+", default=[1e-4, 3e-4, 1e-3])
    parser.add_argument("--steps", type=int, default=20)
    parser.add_argument("--batch-size", type=int, default=32)
    parser.add_argument("--seed", type=int, default=731)
    parser.add_argument("--model-seed", type=int, default=937)
    parser.add_argument("--torch-threads", type=int, default=1)
    args = parser.parse_args()
    if args.torch_threads < 1:
        parser.error("--torch-threads must be positive")
    output = Path(args.output)
    if output.exists():
        parser.error("refusing to overwrite a completed probe")
    torch.set_num_threads(args.torch_threads)
    paths = select_replay_paths(args.input, 10000)
    replay = load_replay(paths, expected_engine_commit=args.engine_commit, holdout=True)
    report = stability_probe(
        replay, model_config=args.model_config, device=args.device,
        learning_rates=tuple(args.learning_rates), steps=args.steps, batch_size=args.batch_size,
        seed=args.seed, model_seed=args.model_seed,
    )
    report.update(engine_commit=args.engine_commit, replay_shards=[path.name for path in paths])
    output.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile("w", dir=output.parent, prefix=".probe-", delete=False) as temporary:
        temporary_path = Path(temporary.name)
        json.dump(report, temporary, sort_keys=True, indent=2, allow_nan=False)
        temporary.flush()
        os.fsync(temporary.fileno())
    try:
        os.link(temporary_path, output)
    finally:
        temporary_path.unlink(missing_ok=True)
    print(json.dumps({"report": str(output), "statuses": [run["status"] for run in report["runs"]]}))


if __name__ == "__main__":
    main()
