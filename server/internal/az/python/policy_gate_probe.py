"""CPU-only initial-policy context sensitivity audit; no training or saved weights."""

import argparse
import hashlib
import json
from pathlib import Path

import torch

from batching import collate_positions
from learner import load_replay
from model import MODEL_CONFIGS, TerraMysticaNet, model_fingerprint
from policy_fit_probe import digest, write_exclusive


def pair_metrics(preactivation, first_weight, output_weight, action_width, context_start):
    gates = preactivation > 0
    left, right = torch.triu_indices(len(gates), len(gates), offset=1)
    if not len(left):
        return None
    difference = (gates[left].float() - gates[right].float()) * output_weight.reshape(1, -1)
    context_jacobian = difference @ first_weight[:, context_start:]
    # Action inputs vary independently; identical gates need not cancel their ranking signal.
    action_jacobian = (gates.float() * output_weight.reshape(1, -1)) @ first_weight[:, :action_width]
    return {
        "pairs": len(left),
        "identical_gate_fraction": float((gates[left] == gates[right]).all(-1).float().mean()),
        "different_gate_fraction": float((gates[left] != gates[right]).float().mean()),
        "context_pair_jacobian_rms": float(context_jacobian.square().mean().sqrt()),
        "action_jacobian_rms": float(action_jacobian.square().mean().sqrt()),
    }


def aggregate(rows):
    if not rows:
        return {"positions": 0, "pairs": 0, "position_equal": {}, "pair_weighted": {}}
    pairs = sum(row["pairs"] for row in rows)
    names = [name for name in rows[0] if name != "pairs"]
    return {"positions": len(rows), "pairs": pairs,
            "position_equal": {name: sum(row[name] for row in rows) / len(rows) for name in names},
            "pair_weighted": {name: sum(row[name] * row["pairs"] for row in rows) / pairs for name in names}}


def audit(model, examples):
    model.eval()
    rows = {"baseline": [], "zero_context": []}
    maximum_error = 0.
    nonspatial_positions = 0
    with torch.no_grad():
        for example in examples:
            batch = collate_positions([example.encoded])
            captured = []
            hook = model.policy[0].register_forward_pre_hook(lambda module, args: captured.append(args[0]))
            try:
                actual, _ = model(*batch.model_inputs())
            finally:
                hook.remove()
            inputs = captured[0][0]
            manual = model.policy(inputs).squeeze(-1)
            maximum_error = max(maximum_error, float((actual[0] - manual).abs().max()))
            selected = (batch.action_hex_indices[0] < 0).all(-1)
            nonspatial_positions += int(selected.any())
            if int(selected.sum()) < 2:
                continue
            width = batch.action_features.shape[-1]
            context_start = width + 2 * model.config.channels
            for name in rows:
                features = inputs[selected].clone()
                if name == "zero_context":
                    features[:, context_start:] = 0
                preactivation = model.policy[0](features)
                rows[name].append(pair_metrics(preactivation, model.policy[0].weight, model.policy[2].weight,
                                               width, context_start))
    if maximum_error > 1e-5:
        raise ValueError("manual versus actual policy logits differ")
    return {"sampled_positions": len(examples), "positions_with_nonspatial_actions": nonspatial_positions,
            "manual_forward_max_abs_error": maximum_error, **{name: aggregate(values) for name, values in rows.items()}}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--fit-report", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()
    if Path(args.output).exists():
        parser.error("refusing to overwrite gate audit")
    torch.set_num_threads(1)
    report_bytes = Path(args.fit_report).read_bytes()
    report = json.loads(report_bytes)
    def source_hashes():
        return {name: hashlib.sha256(Path(__file__).with_name(name).read_bytes()).hexdigest()
                for name in ("policy_gate_probe.py", *report["source_sha256"])}
    source_before = source_hashes()
    if any(source_before[name] != sha for name, sha in report["source_sha256"].items()):
        raise ValueError("fit-report source identity differs")
    paths = [Path(shard["path"]) for shard in report["replay_shards"]]
    for path, shard in zip(paths, report["replay_shards"]):
        if hashlib.sha256(path.read_bytes()).hexdigest() != shard["sha256"]:
            raise ValueError("replay hash changed")
    replay = load_replay(paths, expected_engine_commit=report["engine_commit"], holdout=True)
    if replay.identity() != report["replay_identity"] or digest(report["sample_identities"]) != report["sample_sha256"]:
        raise ValueError("replay/sample identity differs")
    by_name = {game.path.name: index for index, game in enumerate(replay.games)}
    examples = []
    for ref in report["sample_identities"]["training"]:
        game, ply = by_name[ref["shard"]], ref["ply"]
        if replay.games[game].state_ids[ply] != ref["state_id"] or ply not in replay.training_plies[game]:
            raise ValueError("sample state/split identity differs")
        examples.append(replay._example(game, ply))
    torch.manual_seed(report["model_seed"])
    model = TerraMysticaNet(MODEL_CONFIGS[report["model_config"]])
    fingerprint = model_fingerprint(model)
    if any(run["initial_fingerprint"] != fingerprint for run in report["runs"]):
        raise ValueError("initial model identity differs")
    result = audit(model, examples)
    if source_hashes() != source_before or model_fingerprint(model) != fingerprint:
        raise ValueError("source/model changed during audit")
    result.update(purpose="initial-policy structural diagnostic, not strength evidence; no saved weights",
                  fit_report_sha256=hashlib.sha256(report_bytes).hexdigest(), model_fingerprint=fingerprint,
                  sample_sha256=report["sample_sha256"], source_sha256=source_before,
                  context="shared global-action and pooled-board vectors; zeroed with bias preserved",
                  jacobian="context: pair-logit contrast RMS across pairs and coordinates; action: individual-logit RMS across actions and coordinates",
                  torch_version=str(torch.__version__), device="cpu")
    write_exclusive(args.output, result)
    print(json.dumps({"report": args.output}))


if __name__ == "__main__":
    main()
