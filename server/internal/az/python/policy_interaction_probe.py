"""Disposable policy interaction ablations on frozen self-play; no published weights."""

import argparse
import hashlib
import json
import math
from pathlib import Path

import torch
from torch import nn

from learner import load_replay
from model import MODEL_CONFIGS, TerraMysticaNet, model_fingerprint, resolve_device
from policy_fit_probe import digest, minibatches, run_arm, write_exclusive


class BilinearPolicy(nn.Module):
    def __init__(self, action_features, channels):
        super().__init__()
        self.action_features = action_features
        self.scale = math.sqrt(channels)
        self.action = nn.Linear(action_features, channels)
        self.context = nn.Linear(4 * channels, channels)

    def forward(self, inputs):
        action = self.action(inputs[..., :self.action_features])
        context = self.context(inputs[..., self.action_features:])
        return (action * context).sum(-1, keepdim=True) / self.scale


def core_fingerprint(model):
    result = hashlib.sha256()
    for name, tensor in sorted(model.state_dict().items()):
        if not name.startswith("policy."):
            result.update(name.encode())
            result.update(tensor.detach().cpu().contiguous().numpy().tobytes())
    return result.hexdigest()


def replace_head(model, seed):
    before = core_fingerprint(model)
    torch.manual_seed(seed)
    model.policy = BilinearPolicy(model.manifest.action_features, model.config.channels)
    if core_fingerprint(model) != before:
        raise RuntimeError("head replacement changed non-policy weights")
    return before


def scale_shared_context(model, factor=.1):
    """Input-only intervention: preserve every weight and caller-owned tensor."""
    start = model.manifest.action_features + 2 * model.config.channels
    def scaled_input(module, args):
        inputs = args[0].clone()
        inputs[..., start:] *= factor
        return (inputs,)
    return model.policy.register_forward_pre_hook(scaled_input)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--fit-report", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--device", default="mps")
    parser.add_argument("--variant", choices=("bilinear", "context-scaled"), default="bilinear")
    args = parser.parse_args()
    if Path(args.output).exists():
        parser.error("refusing to overwrite interaction probe")
    torch.set_num_threads(1)
    raw = Path(args.fit_report).read_bytes()
    baseline = json.loads(raw)
    expected_optimizer = ({"name": "sgd", "learning_rate": .003, "momentum": .9, "weight_decay": .0001}
                          if args.variant == "bilinear" else
                          {"name": "adamw", "learning_rate": .001, "betas": [.9, .999], "eps": 1e-8, "weight_decay": .0001})
    if baseline["optimizer"] != expected_optimizer:
        raise ValueError("baseline optimizer does not match frozen variant protocol")
    def source_hashes():
        return {name: hashlib.sha256(Path(__file__).with_name(name).read_bytes()).hexdigest()
                for name in ("policy_interaction_probe.py", *baseline["source_sha256"])}
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
    if baseline["steps"] != 400 or baseline["batch_size"] != 32:
        raise ValueError("expected frozen 400-step batch32 baseline")
    stream = minibatches(len(examples["training"]), 400, 32, baseline["batch_seed"])
    if digest(stream) != baseline["minibatch_sha256"]:
        raise ValueError("baseline minibatch identity differs")
    torch.manual_seed(baseline["model_seed"])
    model = TerraMysticaNet(MODEL_CONFIGS[baseline["model_config"]])
    original_fingerprint = model_fingerprint(model)
    if any(run["initial_fingerprint"] != original_fingerprint for run in baseline["runs"]):
        raise ValueError("baseline initialization differs")
    original_head_parameters = sum(parameter.numel() for parameter in model.policy.parameters())
    variant_metadata = {"variant": args.variant}
    if args.variant == "bilinear":
        head_seed = baseline["model_seed"] + 100
        initial_core = replace_head(model, head_seed)
        variant_metadata.update(head_seed=head_seed, head="dot(linear(action),linear(two gathered cells,global context,pooled board))/sqrt(channels)")
    else:
        initial_core = core_fingerprint(model)
        scale_shared_context(model)
        if model_fingerprint(model) != original_fingerprint:
            raise RuntimeError("context scaling changed initial weights")
        variant_metadata.update(context_scale=.1, head="original MLP; scale only shared global-action and pooled-board inputs")
    head_parameters = sum(parameter.numel() for parameter in model.policy.parameters())
    run = run_arm(model, examples["training"], examples["heldout"], stream, arm="joint",
                  device=resolve_device(args.device), learning_rate=expected_optimizer["learning_rate"],
                  optimizer_name=expected_optimizer["name"])
    if source_hashes() != sources:
        raise ValueError("sources changed during interaction probe")
    result = {"purpose": "disposable interaction-head fitting ablation; not strength evidence; no saved weights",
              "baseline_report_sha256": hashlib.sha256(raw).hexdigest(), "source_sha256": sources,
              "replay_shards": baseline["replay_shards"], "sample_sha256": baseline["sample_sha256"],
              "minibatch_sha256": digest(stream), "baseline_initial_fingerprint": original_fingerprint,
              "initial_non_policy_fingerprint": initial_core, **variant_metadata,
              "original_head_parameters": original_head_parameters, "head_parameters": head_parameters,
              "total_parameters": sum(parameter.numel() for parameter in model.parameters()),
              "model_seed": baseline["model_seed"], "model_config": baseline["model_config"],
              "steps": 400, "batch_size": 32, "optimizer": expected_optimizer,
              "device": args.device, "torch_version": str(torch.__version__), "torch_threads": 1,
              "engine_commit": baseline["engine_commit"], "dtype": "float32",
              "runs": [run]}
    write_exclusive(args.output, result)
    print(json.dumps({"report": args.output, "status": run["status"]}))


if __name__ == "__main__":
    main()
