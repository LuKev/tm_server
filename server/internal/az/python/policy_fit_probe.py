"""Disposable, fixed-self-play policy fitting diagnostic; never publishes weights."""

from __future__ import annotations

import argparse
import hashlib
import json
import math
import os
from pathlib import Path
import random
import tempfile
import time

import torch
from torch.nn import functional as F

from learner import DIAGNOSTIC_SPLIT, _training_batch, alpha_zero_loss, load_replay, select_replay_paths
from model import MODEL_CONFIGS, TerraMysticaNet, model_fingerprint, resolve_device


def digest(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def fixed_refs(replay, count, seed, *, heldout=False):
    """Uniform sample without replacement, not action/phase/target balancing."""
    plies = replay.heldout_plies if heldout else replay.training_plies
    population = [(game, ply) for game, indices in enumerate(plies) for ply in indices]
    if count < 1 or not population:
        raise ValueError("fixed subset requires positive count and nonempty eligible population")
    return random.Random(seed).sample(population, min(count, len(population)))


def minibatches(count, steps, batch_size, seed):
    rng = random.Random(seed)
    return [[rng.randrange(count) for _ in range(batch_size)] for _ in range(steps)]


def per_position_metrics(logits, values, target, outcomes, mask):
    log_probability = F.log_softmax(logits, dim=-1)
    probability = log_probability.exp()
    entropy = -(target * target.clamp_min(1e-30).log()).sum(-1)
    cross_entropy = -(target * log_probability).sum(-1)
    legal_count = mask.sum(-1)
    selected_target = target.gather(1, logits.argmax(-1, keepdim=True)).squeeze(1)
    return {
        "policy_cross_entropy": cross_entropy,
        "target_entropy": entropy,
        "prediction_entropy": -(probability * log_probability).sum(-1),
        "target_to_prediction_kl": cross_entropy - entropy,
        "target_to_uniform_kl": legal_count.float().log() - entropy,
        "top_visit_agreement": (selected_target == target.max(-1).values).float(),
        "value_mse": (values - outcomes).square(),
        "legal_logit_spread": logits.masked_fill(~mask, -torch.inf).max(-1).values
        - logits.masked_fill(~mask, torch.inf).min(-1).values,
        "legal_actions": legal_count.float(),
    }


def measure(model, examples, device, batch_size=32):
    model.eval()
    rows = {}
    with torch.no_grad():
        for start in range(0, len(examples), batch_size):
            batch, policy, outcomes = _training_batch(examples[start:start + batch_size], device)
            logits, values = model(*batch.model_inputs())
            for name, value in per_position_metrics(logits, values, policy, outcomes, batch.action_mask).items():
                if not torch.isfinite(value).all():
                    raise FloatingPointError("non-finite diagnostic metric: " + name)
                rows.setdefault(name, []).append(value.cpu())
    rows = {name: torch.cat(values) for name, values in rows.items()}
    result = {}
    for name, selected in (("all", torch.ones(len(examples), dtype=torch.bool)),
                           ("multiple_legal_actions", rows["legal_actions"] > 1)):
        count = int(selected.sum())
        result[name] = {"count": count, **{key: float(value[selected].mean()) if count else None
                                           for key, value in rows.items()}}
    return result


def objective(model, batch, policy, outcomes, arm):
    joint, policy_loss, _ = alpha_zero_loss(model, batch, policy, outcomes)
    if arm == "joint":
        return joint
    if arm == "policy_only":
        return policy_loss
    raise ValueError("unknown diagnostic arm")


def shared_gradients(model, examples, device):
    """Inspect the two objectives on shared board trunk; does not update weights."""
    model.eval()
    batch, policy, outcomes = _training_batch(examples, device)
    _, policy_loss, value_loss = alpha_zero_loss(model, batch, policy, outcomes)
    parameters = [parameter for name, parameter in model.named_parameters()
                  if name.startswith(("stem.", "stem_norm.", "global_broadcast.", "blocks."))]
    policy_grads = torch.autograd.grad(policy_loss, parameters, retain_graph=True)
    value_grads = torch.autograd.grad(value_loss, parameters)
    policy_flat = torch.cat([gradient.flatten() for gradient in policy_grads])
    value_flat = torch.cat([gradient.flatten() for gradient in value_grads])
    policy_norm, value_norm = policy_flat.norm(), value_flat.norm()
    if not torch.isfinite(policy_norm) or not torch.isfinite(value_norm):
        raise FloatingPointError("non-finite shared gradient")
    denominator = float(policy_norm * value_norm)
    return {"policy_norm": float(policy_norm), "value_norm": float(value_norm),
            "cosine": float(torch.dot(policy_flat, value_flat) / (policy_norm * value_norm)) if denominator > 0 else None}


def run_arm(model, training, heldout, stream, *, arm, device, learning_rate):
    model.to(device)
    optimizer = torch.optim.SGD(model.parameters(), lr=learning_rate, momentum=.9, weight_decay=1e-4)
    started = time.perf_counter()
    result = {"arm": arm, "initial_fingerprint": model_fingerprint(model), "snapshots": [],
              "max_gradient_norm": 0., "status": "finite"}

    def snapshot(step):
        result["snapshots"].append({"step": step, "training": measure(model, training, device),
                                    "heldout": measure(model, heldout, device)})

    snapshot(0)
    gradient_examples = [training[index] for index in stream[0]]
    result["initial_shared_gradients"] = shared_gradients(model, gradient_examples, device)
    for step, indices in enumerate(stream, 1):
        model.train()
        batch, policy, outcomes = _training_batch([training[index] for index in indices], device)
        loss = objective(model, batch, policy, outcomes, arm)
        if not torch.isfinite(loss):
            raise FloatingPointError("non-finite diagnostic loss")
        optimizer.zero_grad(set_to_none=True)
        loss.backward()
        norm = torch.linalg.vector_norm(torch.stack([parameter.grad.detach().norm()
                                                    for parameter in model.parameters() if parameter.grad is not None]))
        if not torch.isfinite(norm):
            raise FloatingPointError("non-finite diagnostic gradient")
        result["max_gradient_norm"] = max(result["max_gradient_norm"], float(norm))
        optimizer.step()
        if step == 100 or step == len(stream):
            snapshot(step)
    if not all(torch.isfinite(parameter).all() for parameter in model.parameters()):
        raise FloatingPointError("non-finite diagnostic weights")
    result.update(final_fingerprint=model_fingerprint(model), elapsed_seconds=time.perf_counter() - started,
                  completed_steps=len(stream), final_shared_gradients=shared_gradients(model, gradient_examples, device))
    return result


def probe(replay, *, model_seed=44, sample_seed=731, batch_seed=732, model_config="main",
          device="mps", steps=400, learning_rate=1e-4, training_count=256, heldout_count=128, batch_size=32):
    if not replay.holdout:
        raise ValueError("canonical heldout split is required")
    if not (1 <= steps <= 2000 and 1 <= batch_size <= 128 and 1 <= training_count <= 1024
            and 1 <= heldout_count <= 512 and math.isfinite(learning_rate) and 0 < learning_rate <= .1):
        raise ValueError("probe exceeds bounded diagnostic configuration")
    device = resolve_device(device)
    train_refs = fixed_refs(replay, training_count, sample_seed)
    test_refs = fixed_refs(replay, heldout_count, sample_seed + 1, heldout=True)
    training = [replay._example(*ref) for ref in train_refs]
    heldout = [replay._example(*ref) for ref in test_refs]
    stream = minibatches(len(training), steps, batch_size, batch_seed)
    identities = {}
    for name, refs in (("training", train_refs), ("heldout", test_refs)):
        identities[name] = [{"shard": replay.games[game].path.name, "ply": ply,
                             "state_id": replay.games[game].state_ids[ply]} for game, ply in refs]
    runs = []
    for arm in ("joint", "policy_only"):
        torch.manual_seed(model_seed)
        model = TerraMysticaNet(MODEL_CONFIGS[model_config])
        runs.append(run_arm(model, training, heldout, stream, arm=arm, device=device, learning_rate=learning_rate))
        del model
    if runs[0]["initial_fingerprint"] != runs[1]["initial_fingerprint"]:
        raise RuntimeError("arms did not share initialization")
    return {
        "purpose": "disposable fixed-self-play fitting diagnostic; not strength evidence; no saved weights",
        "model_config": model_config, "model_seed": model_seed, "sample_seed": sample_seed, "batch_seed": batch_seed,
        "device": str(device), "torch_version": str(torch.__version__), "torch_threads": torch.get_num_threads(),
        "dtype": "float32", "steps": steps, "batch_size": batch_size,
        "optimizer": {"name": "sgd", "learning_rate": learning_rate, "momentum": .9, "weight_decay": 1e-4},
        "split": DIAGNOSTIC_SPLIT, "sampling": "uniform-without-replacement subsets; uniform-with-replacement minibatches",
        "eligible_training_positions": replay.training_ply_count,
        "eligible_heldout_positions": replay.ply_count - replay.training_ply_count,
        "sample_identities": identities, "sample_sha256": digest(identities),
        "minibatch_sha256": digest(stream), "replay_identity": replay.identity(), "runs": runs,
    }


def write_exclusive(path, report):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary_path = None
    try:
        with tempfile.NamedTemporaryFile("w", dir=path.parent, prefix=".policy-fit-", delete=False) as temporary:
            temporary_path = Path(temporary.name)
            json.dump(report, temporary, sort_keys=True, indent=2, allow_nan=False)
            temporary.flush()
            os.fsync(temporary.fileno())
        os.link(temporary_path, path)
    finally:
        if temporary_path is not None:
            temporary_path.unlink(missing_ok=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", action="append", required=True)
    parser.add_argument("--engine-commit", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--device", default="mps")
    parser.add_argument("--model-config", choices=MODEL_CONFIGS, default="main")
    for name, default in (("model-seed", 44), ("sample-seed", 731), ("batch-seed", 732),
                          ("steps", 400), ("batch-size", 32), ("training-count", 256), ("heldout-count", 128)):
        parser.add_argument("--" + name, type=int, default=default)
    parser.add_argument("--learning-rate", type=float, default=1e-4)
    args = parser.parse_args()
    if Path(args.output).exists():
        parser.error("refusing to overwrite completed probe")
    torch.set_num_threads(1)
    def source_hashes():
        return {name: hashlib.sha256(Path(__file__).with_name(name).read_bytes()).hexdigest()
                for name in ("policy_fit_probe.py", "learner.py", "model.py", "batching.py", "schema.py")}
    source_before = source_hashes()
    paths = select_replay_paths(args.input, 10000)
    replay = load_replay(paths, expected_engine_commit=args.engine_commit, holdout=True)
    options = {key: value for key, value in vars(args).items() if key not in ("input", "engine_commit", "output")}
    report = probe(replay, **options)
    if source_before != source_hashes():
        raise RuntimeError("diagnostic sources changed during run; refusing report publication")
    report.update(engine_commit=args.engine_commit,
                  replay_shards=[{"path": str(path), "sha256": hashlib.sha256(path.read_bytes()).hexdigest()} for path in paths],
                  source_sha256=source_before)
    write_exclusive(args.output, report)
    print(json.dumps({"report": args.output, "statuses": [run["status"] for run in report["runs"]]}))


if __name__ == "__main__":
    main()
