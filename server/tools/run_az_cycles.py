"""Sequential AlphaZero cycle transactions; resume replays only unpublished stages.

Each committed directory contains its own logs and SHA256 inventory. An attempt
is never training input until its directory is atomically published. Recovery
does not salvage individual games/updates: it quarantines incomplete attempts
and reruns that stage from the immutable cycle parent and seeds. CPU numerical
determinism depends on the invoked tools; MPS is not promised bitwise replay.
"""
import argparse
import fcntl
import hashlib
import json
import os
import re
from pathlib import Path
import signal
import subprocess
import sys
import uuid


def digest(path):
    with Path(path).open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def atomic_json(path, value):
    path = Path(path)
    temporary = path.with_suffix(path.suffix + ".tmp")
    with temporary.open("w") as stream:
        json.dump(value, stream, sort_keys=True, separators=(",", ":"))
        stream.flush()
        os.fsync(stream.fileno())
    temporary.replace(path)


def inventory(directory):
    result = {}
    for path in sorted(directory.rglob("*")):
        if path.is_symlink():
            raise ValueError(f"symlink in stage: {path}")
        if path.is_file() and path != directory / "stage.json":
            result[str(path.relative_to(directory))] = digest(path)
    return result


def verify(directory, identity):
    manifest = json.loads((directory / "stage.json").read_text())
    if manifest["identity"] != identity or manifest["files"] != inventory(directory):
        raise ValueError(f"stage identity/artifact mismatch: {directory}")


def record_stage(root, relative):
    path = root / "committed-stages.json"
    journal = json.loads(path.read_text()) if path.exists() else {}
    journal[relative] = digest(root / relative / "stage.json")
    atomic_json(path, journal)


def verify_journal(root):
    path = root / "committed-stages.json"
    if not path.exists():
        raise ValueError("missing committed-stage journal")
    for relative, expected in json.loads(path.read_text()).items():
        directory = root / relative
        if digest(directory / "stage.json") != expected:
            raise ValueError(f"committed stage manifest changed: {relative}")
        verify(directory, json.loads((directory / "stage.json").read_text())["identity"])
    latest = root / "checkpoints/latest.json"
    if latest.exists():
        reference = json.loads(latest.read_text())
        target = (latest.parent / reference["path"]).resolve()
        if not target.is_relative_to(latest.parent.resolve()) or digest(target) != reference["checkpoint_sha256"]:
            raise ValueError("derived latest checkpoint reference is corrupt")
    progress = root / "progress.json"
    if progress.exists():
        value = json.loads(progress.read_text())
        cycle = value["completed_cycle"]
        if value["next_cycle"] != cycle + 1 or digest(checkpoint(root / f"checkpoints/cycle-{cycle}")) != value["checkpoint_sha256"]:
            raise ValueError("derived progress is corrupt")


def stage(root, relative, identity, command, failpoint=lambda point: None):
    """Publish one whole stage; command receives an isolated output directory."""
    destination = root / relative
    if destination.exists():
        verify(destination, identity)
        record_stage(root, relative)
        return destination
    attempts = root / "attempts"
    attempts.mkdir(exist_ok=True)
    quarantine = root / "quarantine"
    quarantine.mkdir(exist_ok=True)
    key = relative.replace("/", "-")
    for old in attempts.glob(key + "-*"):
        old.rename(quarantine / old.name)
    attempt = attempts / (key + "-" + uuid.uuid4().hex)
    attempt.mkdir()
    with (attempt / "stdout.json").open("wb") as stdout, (attempt / "stderr.log").open("wb") as stderr:
        process = subprocess.Popen(command(attempt), stdout=stdout, stderr=stderr, start_new_session=True)
        def terminate(signum, frame):
            try:
                os.killpg(process.pid, signal.SIGTERM)
            except ProcessLookupError:
                pass
            raise KeyboardInterrupt("stage interrupted; resume quarantines this attempt")
        previous = {sig: signal.signal(sig, terminate) for sig in (signal.SIGTERM, signal.SIGINT)}
        try:
            code = process.wait()
        except KeyboardInterrupt:
            # Reap outside the signal handler: wait() holds a non-reentrant
            # lock when the handler interrupts it.
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                process.wait()
            raise
        finally:
            for sig, handler in previous.items():
                signal.signal(sig, handler)
    atomic_json(attempt / "status.json", {"exit_code": code})
    print((attempt / "stdout.json").read_text(), end="", flush=True)
    print((attempt / "stderr.log").read_text(), end="", file=sys.stderr, flush=True)
    if code:
        raise RuntimeError(f"stage {relative} failed ({code}); attempt retained: {attempt}")
    failpoint("after_command")
    atomic_json(attempt / "stage.json", {"identity": identity, "files": inventory(attempt)})
    failpoint("before_publish")
    destination.parent.mkdir(parents=True, exist_ok=True)
    attempt.rename(destination)
    failpoint("after_publish")
    record_stage(root, relative)
    return destination


def checkpoint(directory):
    reference = json.loads((directory / "latest.json").read_text())
    path = directory / reference["path"]
    if path.parent != directory or digest(path) != reference["checkpoint_sha256"]:
        raise ValueError(f"invalid checkpoint reference: {directory}")
    return path


def tool_identity(executable, *, require_python=False):
    """Bazel Python launchers are stable even when project imports change."""
    executable = Path(executable).resolve()
    sources = executable.parent / (executable.name + ".runfiles") / "tm_server"
    project_python = {
        str(path.relative_to(sources)): digest(path)
        for path in sorted(sources.rglob("*.py")) if path.is_file()
    }
    if require_python and not project_python:
        raise ValueError(f"missing project Python runfiles: {executable}")
    return {"executable": digest(executable), "project_python": project_python}


def runtime_pins(directory):
    # Bazel verifies downloaded runtime/wheel content against these pinned
    # definitions. This is reproducibility validation, not a hostile-cache audit.
    return {name: digest(directory / name) for name in ("WORKSPACE", "az_requirements_lock.txt")}


def adopt(root, cycle, candidate):
    """Monotone adoption, including interruption between latest and progress."""
    adopted = 0
    if (root / "progress.json").exists():
        adopted = json.loads((root / "progress.json").read_text())["completed_cycle"]
    latest = root / "checkpoints/latest.json"
    if latest.exists():
        reference = json.loads(latest.read_text())
        adopted = max(adopted, int(Path(reference["path"]).parent.name.removeprefix("cycle-")))
    if cycle < adopted:
        return
    reference = json.loads((candidate.parent / "latest.json").read_text())
    reference["path"] = str(candidate.relative_to(latest.parent))
    atomic_json(latest, reference)
    atomic_json(root / "progress.json", {"completed_cycle": cycle, "next_cycle": cycle + 1, "checkpoint_sha256": digest(candidate)})


def configuration(args):
    defaults = {
        "INFERENCE_DEVICE": "cpu", "TORCH_THREADS": "1", "ACTOR_GAMES_PER_BATCH": "32",
        "INFERENCE_MAX_BATCH": "32", "INFERENCE_MAX_WAIT": "1ms", "ROOT_SEARCH_MODE": "concurrent",
        "REPLAY_CACHE_GAMES": "8", "REPLAY_WINDOW_SHARDS": "10000", "ARENA_GAMES_PER_BATCH": "8",
        "EXAMPLES_PER_REPLAY_PLY": "", "OPTIMIZER": "sgd", "LEARNING_RATE": "1e-4",
        "WEIGHT_DECAY": "1e-4", "MOMENTUM": "", "BATCH_SIZE": "32",
        "SETUP_SEED_BASE": "1000000", "SEARCH_SEED_BASE": "2000000", "EVAL_EVERY": "1",
        "MODEL_SEED": "0", "LEARNER_SEED_BASE": "0",
        "EVAL_BASELINES": "uniform,random,heuristic,policy,previous,anchor",
    }
    settings = {key.lower(): os.environ.get("TM_AZ_" + key, value) for key, value in defaults.items()}
    settings["learner_device"] = os.environ.get("TM_AZ_LEARNER_DEVICE", settings["inference_device"])
    # Match the actor's Go int64 CLI as well as PyTorch's manual_seed range.
    model_seed = int(settings["model_seed"])
    if not 0 <= model_seed <= 2**63 - 1:
        raise ValueError("model_seed must be between 0 and 2**63 - 1")
    settings["model_seed"] = str(model_seed)
    learner_seed_base = int(settings["learner_seed_base"])
    if not 0 <= learner_seed_base <= 2**63 - 1 - args.cycles:
        raise ValueError("learner_seed_base plus cycles must fit a non-negative int64")
    settings["learner_seed_base"] = str(learner_seed_base)
    for key in ("actor_games_per_batch", "inference_max_batch", "replay_cache_games", "replay_window_shards", "arena_games_per_batch", "batch_size", "eval_every"):
        if int(settings[key]) <= 0:
            raise ValueError(f"{key} must be positive")
    for key in ("setup_seed_base", "search_seed_base", "torch_threads"):
        if int(settings[key]) < 0:
            raise ValueError(f"{key} must be non-negative")
    if not re.fullmatch(r"[0-9a-f]{40}", args.engine_commit) and args.engine_commit not in ("test", "smoke"):
        raise ValueError("engine_commit must be a full immutable commit SHA (test/smoke reserved for tests)")
    if not settings["eval_baselines"] or set(settings["eval_baselines"].split(",")) - {"uniform", "random", "heuristic", "policy", "previous", "anchor"}:
        raise ValueError("invalid TM_AZ_EVAL_BASELINES")
    if settings["optimizer"] not in ("sgd", "adamw"):
        raise ValueError("TM_AZ_OPTIMIZER must be sgd or adamw")
    if settings["optimizer"] == "adamw" and settings["momentum"]:
        raise ValueError("TM_AZ_MOMENTUM requires sgd")
    if settings["optimizer"] == "sgd":
        settings["momentum"] = settings["momentum"] or "0.9"
    settings.update({key: getattr(args, key) for key in (
        "engine_commit", "cycles", "games", "simulations", "train_steps", "eval_cases", "model_config")})
    settings["format_version"] = 2
    settings["tools"] = {key: tool_identity(getattr(args, key), require_python=key in ("inference", "learner")) for key in ("selfplay", "inference", "learner", "evaluator")}
    settings["runner_sha256"] = digest(__file__)
    settings["runtime_pins"] = runtime_pins(Path(__file__).parent.parent)
    return settings


def run(args):
    config = configuration(args)
    root = Path(args.run_dir).resolve()
    if args.mode == "new" and root.exists() and any(root.iterdir()):
        raise ValueError(f"new run requires empty directory: {root}")
    if args.mode == "resume" and not (root / "run-manifest.json").is_file():
        raise ValueError("resume requires an existing run manifest")
    root.mkdir(parents=True, exist_ok=True)
    with (root / ".lock").open("a") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        manifest = root / "run-manifest.json"
        if args.mode == "new":
            if manifest.exists():
                raise ValueError("run already initialized")
            atomic_json(root / "committed-stages.json", {})
            atomic_json(manifest, config)
        elif json.loads(manifest.read_text()) != config:
            raise ValueError("resume configuration/tool identity differs from immutable manifest")
        verify_journal(root)
        execute_cycles(args, root, config)


def execute_cycles(args, root, c):
    common = ["--model-config", c["model_config"], "--inference-device", c["inference_device"],
              "--inference-torch-threads", c["torch_threads"], "--engine-commit", c["engine_commit"]]
    parent = None
    anchor = None
    replay = []
    for cycle in range(1, c["cycles"] + 1):
        identity = {"cycle": cycle, "parent_sha256": digest(parent) if parent else f"random-seed-{c['model_seed']}"}
        pin = root / f"cycle-{cycle}-parent.json"
        if pin.exists() and json.loads(pin.read_text()) != identity:
            raise ValueError(f"cycle parent changed: {cycle}")
        if not pin.exists():
            atomic_json(pin, identity)
        actor = stage(root, f"replay/cycle-{cycle}", identity, lambda output: [
            args.selfplay, "--inference", args.inference, "--checkpoint", str(parent) if parent else "",
            *common, "--inference-max-batch", c["inference_max_batch"], "--inference-max-wait", c["inference_max_wait"],
            "--root-search-mode", c["root_search_mode"], "--output", str(output), "--games", str(c["games"]),
            "--games-per-batch", c["actor_games_per_batch"], "--simulations", str(c["simulations"]),
            "--seed", str(int(c["setup_seed_base"]) + cycle * c["games"]), "--search-seed", str(int(c["search_seed_base"]) + cycle * c["games"]), "--model-seed", c["model_seed"]])
        replay.extend(["--input", str(actor)])
        work = ["--examples-per-replay-ply", c["examples_per_replay_ply"]] if c["examples_per_replay_ply"] else ["--steps", str(c["train_steps"])]
        trained = stage(root, f"checkpoints/cycle-{cycle}", identity, lambda output: [
            args.learner, *replay, "--initial-checkpoint", str(parent) if parent else "",
            "--model-config", c["model_config"], "--device", c["learner_device"], "--torch-threads", c["torch_threads"],
            "--output-dir", str(output), "--engine-commit", c["engine_commit"], *work,
            "--batch-size", c["batch_size"], "--window-shards", c["replay_window_shards"], "--replay-cache-games", c["replay_cache_games"],
            "--optimizer", c["optimizer"], "--learning-rate", c["learning_rate"], "--weight-decay", c["weight_decay"],
            *(["--momentum", c["momentum"]] if c["momentum"] else []), "--model-seed", c["model_seed"], "--seed", str(int(c["learner_seed_base"]) + cycle)])
        candidate = checkpoint(trained)
        opponents = {"uniform": ["--baseline-uniform"], "random": ["--baseline-mode", "random"],
                     "heuristic": ["--baseline-mode", "heuristic"],
                     "policy": ["--baseline-checkpoint", str(candidate), "--baseline-mode", "policy"]}
        if parent:
            opponents["previous"] = ["--baseline-checkpoint", str(parent)]
        if anchor and anchor != parent:
            opponents["anchor"] = ["--baseline-checkpoint", str(anchor)]
        evaluate = cycle == 1 or cycle == c["cycles"] or cycle % int(c["eval_every"]) == 0
        for name in c["eval_baselines"].split(",") if evaluate else []:
            if name not in opponents:
                continue
            opponent = opponents[name]
            stage(root, f"evaluation/cycle-{cycle}-{name}", {**identity, "candidate_sha256": digest(candidate), "opponent": name}, lambda output: [
                args.evaluator, "--suite", "development", "--inference", args.inference, "--candidate-checkpoint", str(candidate), *opponent,
                *common, "--output", str(output), "--games-per-batch", c["arena_games_per_batch"],
                "--cases", str(c["eval_cases"]), "--simulations", str(c["simulations"])])
        parent = candidate
        anchor = anchor or candidate
        adopt(root, cycle, candidate)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--mode", choices=("new", "resume"), required=True)
    for name in ("selfplay", "inference", "learner", "evaluator", "run_dir", "engine_commit"):
        parser.add_argument(name)
    for name, default in (("cycles", 1), ("games", 8), ("simulations", 8), ("train_steps", 100), ("eval_cases", 8)):
        parser.add_argument(name, type=int, nargs="?", default=default)
    parser.add_argument("model_config", nargs="?", default="main")
    args = parser.parse_args()
    if min(args.cycles, args.games, args.simulations, args.train_steps, args.eval_cases) <= 0:
        parser.error("cycle, game, simulation, step and evaluation counts must be positive")
    try:
        run(args)
    except (ValueError, RuntimeError, OSError, KeyboardInterrupt) as error:
        parser.exit(1, str(error) + "\n")


if __name__ == "__main__":
    main()
