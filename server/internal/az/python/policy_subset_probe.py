"""Disposable nested small-subset memorization diagnostic using self-play only."""

import argparse
import hashlib
import json
from pathlib import Path
import random

import torch

from learner import load_replay
from model import MODEL_CONFIGS, TerraMysticaNet, model_fingerprint, resolve_device
from policy_fit_probe import digest, measure, minibatches, run_arm, write_exclusive


def nested_indices(examples, seed=731):
    indices = [index for index, example in enumerate(examples) if len(example.policy) > 1]
    if len(indices) < 8:
        raise ValueError("nested subset probe requires at least eight multi-action examples")
    random.Random(seed).shuffle(indices)
    return [indices[:1], indices[:8]]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--fit-report", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--device", default="mps")
    parser.add_argument("--steps", type=int, choices=(12, 13, 400), default=400)
    args = parser.parse_args()
    if Path(args.output).exists():
        parser.error("refusing to overwrite subset probe")
    torch.set_num_threads(1)
    raw = Path(args.fit_report).read_bytes()
    baseline = json.loads(raw)
    optimizer = {"name": "adamw", "learning_rate": .001, "betas": [.9, .999], "eps": 1e-8, "weight_decay": .0001}
    if baseline["optimizer"] != optimizer or baseline["steps"] != 400 or baseline["batch_size"] != 32:
        raise ValueError("expected frozen AdamW .001, 400-step, batch32 baseline")
    def source_hashes():
        return {name: hashlib.sha256(Path(__file__).with_name(name).read_bytes()).hexdigest()
                for name in ("policy_subset_probe.py", *baseline["source_sha256"])}
    sources = source_hashes()
    if any(sources[name] != sha for name, sha in baseline["source_sha256"].items()):
        raise ValueError("baseline source identity differs")
    paths = [Path(item["path"]) for item in baseline["replay_shards"]]
    for path, item in zip(paths, baseline["replay_shards"]):
        if hashlib.sha256(path.read_bytes()).hexdigest() != item["sha256"]:
            raise ValueError("baseline replay hash differs")
    replay = load_replay(paths, expected_engine_commit=baseline["engine_commit"], holdout=True)
    if replay.identity() != baseline["replay_identity"] or digest(baseline["sample_identities"]) != baseline["sample_sha256"]:
        raise ValueError("baseline replay/sample identity differs")
    by_name = {game.path.name: index for index, game in enumerate(replay.games)}
    examples = {}
    for split, refs in baseline["sample_identities"].items():
        examples[split] = []
        eligible = replay.training_plies if split == "training" else replay.heldout_plies
        for ref in refs:
            game, ply = by_name[ref["shard"]], ref["ply"]
            if replay.games[game].state_ids[ply] != ref["state_id"] or ply not in eligible[game]:
                raise ValueError("baseline sample state/split identity differs")
            examples[split].append(replay._example(game, ply))
    runs = []
    device = resolve_device(args.device)
    for indices in nested_indices(examples["training"]):
        training = [examples["training"][index] for index in indices]
        identities = [baseline["sample_identities"]["training"][index] for index in indices]
        torch.manual_seed(baseline["model_seed"])
        model = TerraMysticaNet(MODEL_CONFIGS[baseline["model_config"]])
        if any(run["initial_fingerprint"] != model_fingerprint(model) for run in baseline["runs"]):
            raise ValueError("baseline initialization differs")
        model.to(device)
        before = [measure(model, [example], device)["all"] for example in training]
        stream = minibatches(len(training), args.steps, 32, 732)
        run = run_arm(model, training, examples["heldout"], stream, arm="joint", device=device,
                      learning_rate=.001, optimizer_name="adamw")
        after = [measure(model, [example], device)["all"] for example in training]
        run.update(subset_size=len(training), sample_identities=identities, sample_sha256=digest(identities),
                   minibatch_sha256=digest(stream), per_position=[
                       {"identity": identity, "legal_actions": len(example.policy),
                        "target_entropy": initial["target_entropy"],
                        "initial_kl": initial["target_to_prediction_kl"], "final_kl": final["target_to_prediction_kl"]}
                       for identity, example, initial, final in zip(identities, training, before, after)])
        runs.append(run)
        del model
    if source_hashes() != sources:
        raise ValueError("sources changed during subset diagnostic")
    write_exclusive(args.output, {"purpose": "disposable memorization diagnostic; not strength evidence; no saved weights",
                    "baseline_report_sha256": hashlib.sha256(raw).hexdigest(), "source_sha256": sources,
                    "replay_shards": baseline["replay_shards"], "baseline_sample_sha256": baseline["sample_sha256"],
                    "engine_commit": baseline["engine_commit"], "model_seed": baseline["model_seed"],
                    "model_config": baseline["model_config"], "subset_seed": 731, "batch_seed": 732,
                    "sampling": "uniform shuffled multi-action frozen-training indices; nested prefixes 1 and 8",
                    "minibatches": "uniform sampling with replacement; size1 repeats its sole position 32 times per update",
                    "steps": args.steps, "batch_size": 32, "optimizer": optimizer, "device": str(device),
                    "dtype": "float32", "torch_version": str(torch.__version__), "torch_threads": 1, "runs": runs})
    print(json.dumps({"report": args.output, "statuses": [run["status"] for run in runs]}))


if __name__ == "__main__":
    main()
