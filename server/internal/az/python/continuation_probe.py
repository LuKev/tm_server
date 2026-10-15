"""Controlled fixed-replay SGD continuation; diagnostic only, never publishes models."""

from __future__ import annotations

import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import random
import tempfile
import time

import torch
from torch.nn import functional as F

from learner import (ReplayWindow, _training_batch, load_replay, resolve_latest,
                     select_replay_paths, train, validate_parent_provenance)
from model import (TerraMysticaNet, load_checkpoint, model_fingerprint,
                   resolve_device, resolve_model_config)


def fixed_examples(replay: ReplayWindow, heldout: bool, count: int, seed: int):
    refs = [(game, ply) for game, plies in enumerate(
        replay.heldout_plies if heldout else replay.training_plies) for ply in plies]
    selected = random.Random(seed).sample(refs, min(count, len(refs)))
    return [replay._example(game, ply) for game, ply in selected]


def calibration(model, examples, device):
    """Uniform without-replacement fixed sample, separate from stratified learner diagnostics."""
    if not examples:
        return None
    model.eval()
    values, targets, policy_losses, target_entropies, uniform_losses = [], [], [], [], []
    with torch.inference_mode():
        for start in range(0, len(examples), 32):
            batch, target_policy, target_value = _training_batch(examples[start:start + 32], device)
            logits, value = model(*batch.model_inputs())
            values.extend(value.cpu().tolist())
            targets.extend(target_value.cpu().tolist())
            policy_losses.extend((-(target_policy * F.log_softmax(logits, -1)).sum(-1)).cpu().tolist())
            target_entropies.extend((-(target_policy * target_policy.clamp_min(1e-30).log()).sum(-1)).cpu().tolist())
            uniform_losses.extend(batch.action_mask.sum(-1).float().log().cpu().tolist())
    v, z = torch.tensor(values), torch.tensor(targets)
    return {
        "positions": len(examples), "prediction_mean": float(v.mean()),
        "target_mean": float(z.mean()), "prediction_std": float(v.std(unbiased=False)),
        "value_mse": float(((v - z) ** 2).mean()),
        "diagnostic_target_mean_constant_mse": float(((z - z.mean()) ** 2).mean()),
        "zero_constant_mse": float((z ** 2).mean()),
        "policy_loss": sum(policy_losses) / len(policy_losses),
        "target_policy_entropy": sum(target_entropies) / len(target_entropies),
        "target_to_prediction_policy_kl": (sum(policy_losses) - sum(target_entropies)) / len(policy_losses),
        "uniform_legal_policy_loss": sum(uniform_losses) / len(uniform_losses),
        "target_to_uniform_policy_kl": (sum(uniform_losses) - sum(target_entropies)) / len(uniform_losses),
        "saturation_fraction": float((v.abs() >= .95).float().mean()),
        "by_target": {str(target): {"count": int((z == target).sum()),
                       "prediction_mean": float(v[z == target].mean())}
                      for target in (-1, 0, 1) if (z == target).any()},
    }


def mode_agreement(model, examples, device):
    batch, _, _ = _training_batch(examples[:32], device)
    with torch.no_grad():
        model.train()
        train_policy, train_value = model(*batch.model_inputs())
        model.eval()
        eval_policy, eval_value = model(*batch.model_inputs())
    return {"logit_max_abs_difference": float((train_policy - eval_policy).abs().max()),
            "value_max_abs_difference": float((train_value - eval_value).abs().max())}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", action="append", required=True)
    parser.add_argument("--initial-latest", required=True)
    parser.add_argument("--baseline-latest", required=True)
    parser.add_argument("--engine-commit", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--device", default="mps")
    parser.add_argument("--seed", type=int, default=22004)
    parser.add_argument("--model-seed", type=int, default=22)
    parser.add_argument("--steps", type=int, default=100)
    parser.add_argument("--chunk-steps", type=int, default=10)
    parser.add_argument("--diagnostic-positions", type=int, default=512)
    parser.add_argument("--learning-rates", type=float, nargs="+", default=[.001, .0003, .0001])
    args = parser.parse_args()
    if not 1 <= args.steps <= 100 or not 1 <= args.chunk_steps <= 100 or not 1 <= args.diagnostic_positions <= 1024:
        parser.error("bounded to 100 updates and 1024 positions per diagnostic population")
    if not 1 <= len(args.learning_rates) <= 3 or not all(0 < rate <= .001 for rate in args.learning_rates):
        parser.error("bounded to three positive learning rates at most .001")
    output = Path(args.output)
    if output.exists():
        parser.error("refusing to overwrite probe report")
    torch.set_num_threads(1)
    device = resolve_device(args.device)
    initial = resolve_latest(args.initial_latest)
    baseline_path = resolve_latest(args.baseline_latest)
    _, config = resolve_model_config("auto", initial)
    baseline = TerraMysticaNet(config)
    baseline_payload = load_checkpoint(baseline_path, baseline)
    validate_parent_provenance(baseline_payload, args.engine_commit)
    paths = select_replay_paths(args.input, 10000)
    replay = load_replay(paths, expected_engine_commit=args.engine_commit, holdout=True)
    populations = {"training": fixed_examples(replay, False, args.diagnostic_positions, 54721),
                   "heldout": fixed_examples(replay, True, args.diagnostic_positions, 54722)}
    baseline.to(device)
    baseline_calibration = {name: calibration(baseline, examples, device)
                            for name, examples in populations.items()}
    baseline.cpu()
    runs = []
    previous_states = {}
    for rate in args.learning_rates:
        torch.manual_seed(args.model_seed)
        model = TerraMysticaNet(config)
        parent = load_checkpoint(initial, model)
        validate_parent_provenance(parent, args.engine_commit)
        initial_model_id = "model-" + model_fingerprint(model)
        if initial_model_id != parent["model_id"]:
            raise ValueError("initial model fingerprint mismatch")
        if parent["optimizer_config"] != {"name": "sgd", "learning_rate": .001,
                                          "weight_decay": .0001, "momentum": .9}:
            raise ValueError("probe requires the original SGD configuration")
        optimizer = copy.deepcopy(parent["optimizer_state"])
        for group in optimizer["param_groups"]:
            group["lr"] = rate  # Deliberate LR-only intervention; preserve momentum tensors.
        model.to(device)
        checkpoints = [{"step": 0, **{name: calibration(model, examples, device)
                                      for name, examples in populations.items()}}]
        agreement = mode_agreement(model, populations["training"], device)
        started = time.perf_counter()
        state = None
        for start in range(0, args.steps, args.chunk_steps):
            count = min(args.chunk_steps, args.steps - start)
            metrics, optimizer, state = train(
                model, replay, steps=count, batch_size=32, learning_rate=rate,
                weight_decay=.0001, seed=args.seed, device=device, optimizer_name="sgd",
                momentum=.9, optimizer_state=optimizer, learner_state=state, collect_telemetry=True)
            checkpoints.append({"step": start + count, "learner": metrics,
                                **{name: calibration(model, examples, device)
                                   for name, examples in populations.items()}})
        model.cpu()
        differences = [float((value - baseline.state_dict()[key]).abs().max())
                       for key, value in model.state_dict().items()]
        duplicate_difference = (max(float((value - previous_states[rate][key]).abs().max())
                                    for key, value in model.state_dict().items())
                                if rate in previous_states else None)
        if args.learning_rates.count(rate) > 1:
            previous_states[rate] = {key: value.detach().clone() for key, value in model.state_dict().items()}
        runs.append({"learning_rate": rate, "elapsed_seconds": time.perf_counter() - started,
                     "initial_model_id": initial_model_id,
                     "mode_agreement": agreement, "checkpoints": checkpoints,
                     "final_model_id": "model-" + model_fingerprint(model),
                     "same_rate_previous_max_parameter_difference": duplicate_difference,
                     "original_final_max_parameter_difference": max(differences),
                     "original_final_allclose_rtol1em4_atol1em5": all(
                         torch.allclose(value, baseline.state_dict()[key], rtol=1e-4, atol=1e-5)
                         for key, value in model.state_dict().items()),
                     "completed_steps": state["completed_steps"],
                     "sampling_rng_sha256": hashlib.sha256(repr(state["sampling_rng"]).encode()).hexdigest()})
    report = {"purpose": "fixed self-play replay LR-only continuation; not strength evidence",
              "intervention": "preserve initial weights and SGD momentum; change group LR only",
              "engine_commit": args.engine_commit, "device": str(device), "torch_version": str(torch.__version__),
              "seed": args.seed, "model_seed": args.model_seed, "batch_size": 32, "steps": args.steps,
              "chunk_steps": args.chunk_steps,
              "initial_checkpoint_sha256": hashlib.sha256(initial.read_bytes()).hexdigest(),
              "baseline_model_id": baseline_payload["model_id"],
              "baseline_calibration": baseline_calibration,
              "baseline_learner_contract": baseline_payload["learner_state"]["contract"],
              "baseline_sampling_rng_sha256": hashlib.sha256(repr(
                  baseline_payload["learner_state"]["sampling_rng"]).encode()).hexdigest(),
              "replay_identity": replay.identity(), "replay_shards": [path.name for path in paths],
              "training_positions": replay.training_ply_count, "heldout_positions": replay.ply_count - replay.training_ply_count,
              "diagnostics": "fixed uniform samples without replacement, RNG independent of optimizer; heldout excluded from optimizer",
              "source_sha256": {name: hashlib.sha256(Path(__file__).with_name(name).read_bytes()).hexdigest()
                                for name in ("continuation_probe.py", "learner.py", "model.py", "schema.py", "batching.py")},
              "runs": runs}
    output.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile("w", dir=output.parent, delete=False) as temporary:
        json.dump(report, temporary, sort_keys=True, indent=2, allow_nan=False)
        temporary.flush()
        os.fsync(temporary.fileno())
    try:
        os.link(temporary.name, output)
    finally:
        Path(temporary.name).unlink(missing_ok=True)
    print(json.dumps({"report": str(output), "runs": len(runs)}))


if __name__ == "__main__":
    main()
