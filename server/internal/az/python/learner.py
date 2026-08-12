"""AlphaZero policy/value learner over immutable canonical trajectory shards."""

from __future__ import annotations

import argparse
from collections import OrderedDict
from dataclasses import dataclass
import gzip
import hashlib
import json
import math
from pathlib import Path
import random
import re
import time
from typing import Any

import numpy as np
import torch
from torch import Tensor
from torch.nn import functional as F

from batching import TensorBatch, collate_positions
from model import MODEL_CONFIGS, TerraMysticaNet, load_checkpoint, model_fingerprint, publish_checkpoint, resolve_device, resolve_model_config
from schema import ACTION_SCHEMA_VERSION, RULES_VERSION, STATE_SCHEMA_VERSION, EncodedPosition, encode_position, validate_position_schema

TRAJECTORY_FORMAT_VERSION = 2
_SHARD_NAME = re.compile(r"trajectory-([0-9a-f]{64})\.json\.gz")


@dataclass(frozen=True)
class Example:
    encoded: EncodedPosition
    policy: np.ndarray
    value: float


@dataclass(frozen=True)
class ShardGame:
    path: Path
    matchup: tuple[int, int]
    ply_count: int


class ReplayWindow:
    """Samples uniformly while retaining at most a bounded number of shard games."""

    def __init__(
        self,
        games: list[list[Example] | ShardGame],
        matchups: list[tuple[int, int]],
        *,
        cache_games: int = 8,
    ) -> None:
        if not games or len(games) != len(matchups):
            raise ValueError("replay window requires non-empty games and matchups")
        if cache_games <= 0:
            raise ValueError("replay cache must retain at least one game")
        self.games = games
        self.cache_games = cache_games
        self._cache: OrderedDict[int, dict[str, Any]] = OrderedDict()
        self.materialization_loads = 0
        self.materialization_cache_hits = 0
        self.by_matchup: dict[tuple[int, int], list[int]] = {}
        for index, matchup in enumerate(matchups):
            self.by_matchup.setdefault(matchup, []).append(index)
        self.matchups = sorted(self.by_matchup)

    def sample(self, rng: random.Random, count: int) -> list[Example]:
        output = []
        for _ in range(count):
            matchup = rng.choice(self.matchups)
            game_index = rng.choice(self.by_matchup[matchup])
            game = self.games[game_index]
            ply_index = rng.randrange(len(game) if isinstance(game, list) else game.ply_count)
            output.append(self._example(game_index, ply_index))
        return output

    def diagnostic(self, count: int) -> list[Example]:
        output = []
        for matchup in self.matchups:
            for game_index in self.by_matchup[matchup]:
                output.append(self._example(game_index, 0))
                if len(output) >= count:
                    return output
        return output

    def _example(self, game_index: int, ply_index: int) -> Example:
        game = self.games[game_index]
        if isinstance(game, list):
            return game[ply_index]
        trajectory = self._trajectory(game_index, game)
        return _example_from_validated_trajectory(trajectory, ply_index)

    def _trajectory(self, game_index: int, game: ShardGame) -> dict[str, Any]:
        cached = self._cache.get(game_index)
        if cached is not None:
            self.materialization_cache_hits += 1
            self._cache.move_to_end(game_index)
            return cached
        trajectory = _read_trajectory(game.path)
        manifest = trajectory.get("manifest") or {}
        matchup = tuple(int(value) for value in (manifest.get("factions") or []))
        if matchup != game.matchup or len(trajectory.get("steps") or []) != game.ply_count:
            raise ValueError(f"trajectory metadata changed after replay validation: {game.path}")
        self.materialization_loads += 1
        self._cache[game_index] = trajectory
        self._cache.move_to_end(game_index)
        while len(self._cache) > self.cache_games:
            self._cache.popitem(last=False)
        return trajectory

    @property
    def resident_game_count(self) -> int:
        return len(self._cache)

    @property
    def game_count(self) -> int:
        return len(self.games)

    @property
    def ply_count(self) -> int:
        return sum(len(game) if isinstance(game, list) else game.ply_count for game in self.games)


def _verify_manifest(
    trajectory: dict[str, Any], path: Path, expected_engine_commit: str
) -> tuple[int, int]:
    manifest = trajectory.get("manifest") or {}
    expected = {
        "format_version": TRAJECTORY_FORMAT_VERSION,
        "rules_version": RULES_VERSION,
        "state_version": STATE_SCHEMA_VERSION,
        "action_version": ACTION_SCHEMA_VERSION,
    }
    actual = {key: manifest.get(key) for key in expected}
    if actual != expected:
        raise ValueError(f"trajectory schema mismatch in {path}: got {actual}, want {expected}")
    if manifest.get("engine_commit") != expected_engine_commit:
        raise ValueError(
            f"trajectory engine commit mismatch in {path}: "
            f"got {manifest.get('engine_commit')!r}, want {expected_engine_commit!r}"
        )
    if not manifest.get("model_id"):
        raise ValueError(f"trajectory in {path} is missing model identity")
    search_seed = manifest.get("search_seed")
    if not isinstance(search_seed, int) or isinstance(search_seed, bool) or search_seed == 0:
        raise ValueError(f"trajectory in {path} is missing an exact search seed")
    factions = manifest.get("factions") or []
    if len(factions) != 2:
        raise ValueError(f"trajectory in {path} does not contain two factions")
    return int(factions[0]), int(factions[1])


def _read_trajectory(path: str | Path) -> dict[str, Any]:
    shard_path = Path(path)
    match = _SHARD_NAME.fullmatch(shard_path.name)
    compressed = shard_path.read_bytes()
    digest = hashlib.sha256(compressed).hexdigest()
    if match is None or match.group(1) != digest:
        raise ValueError(f"trajectory filename hash mismatch for {shard_path}")
    return json.loads(gzip.decompress(compressed))


def _example_from_validated_trajectory(trajectory: dict[str, Any], step_index: int) -> Example:
    step = trajectory["steps"][step_index]
    visits = np.asarray(step["visits"], dtype=np.float32)
    return Example(
        encode_position(step["state"], step["actions"]),
        visits / visits.sum(),
        float(step["outcome"]),
    )


def _validate_shard(path: str | Path, expected_engine_commit: str) -> ShardGame:
    raw_path = Path(path).resolve()
    trajectory = _read_trajectory(raw_path)
    matchup = _verify_manifest(trajectory, raw_path, expected_engine_commit)
    manifest = trajectory["manifest"]
    checkpoint_sha256 = str(manifest.get("checkpoint_sha256") or "")
    if str(manifest.get("model_id", "")).startswith("model-") and not re.fullmatch(r"[0-9a-f]{64}", checkpoint_sha256):
        raise ValueError(f"trained trajectory in {raw_path} is missing checkpoint identity")
    steps = trajectory.get("steps") or []
    if not trajectory.get("completed") or int(trajectory.get("ply_count", -1)) != len(steps) or not steps:
        raise ValueError(f"trajectory in {raw_path} is incomplete")
    final_vp = trajectory.get("final_vp") or []
    terminal = trajectory.get("terminal_state") or {}
    terminal_bytes = json.dumps(terminal, separators=(",", ":"), ensure_ascii=False).encode()
    terminal_hash = hashlib.sha256(terminal_bytes).hexdigest()[:32]
    if len(final_vp) != 2 or terminal_hash != trajectory.get("terminal_state_hash"):
        raise ValueError(f"invalid terminal result in {raw_path}")
    if int((terminal.get("state") or {}).get("phase", -1)) != 5:
        raise ValueError(f"terminal state in {raw_path} is not terminal")
    for step_index, step in enumerate(steps):
        actions = step.get("actions") or []
        validate_position_schema(step.get("state"), actions)
        state_bytes = json.dumps(step.get("state"), separators=(",", ":"), ensure_ascii=False).encode()
        state_hash = hashlib.sha256(state_bytes).hexdigest()[:32]
        if state_hash != step.get("state_hash"):
            raise ValueError(f"state hash mismatch in {raw_path} step {step_index}")
        visits = np.asarray(step.get("visits") or [], dtype=np.float32)
        if (
            len(actions) == 0
            or visits.shape != (len(actions),)
            or not np.isfinite(visits).all()
            or (visits < 0).any()
            or visits.sum() <= 0
        ):
            raise ValueError(f"invalid policy target in {raw_path} step {step_index}")
        chosen = int(step.get("chosen_index", -1))
        if chosen < 0 or chosen >= len(actions) or visits[chosen] <= 0:
            raise ValueError(f"invalid chosen action in {raw_path} step {step_index}")
        if len({json.dumps(action, sort_keys=True, separators=(",", ":")) for action in actions}) != len(actions):
            raise ValueError(f"duplicate legal action in {raw_path} step {step_index}")
        value = float(step.get("outcome"))
        if value not in (-1, 0, 1):
            raise ValueError(f"invalid value target in {raw_path} step {step_index}")
        seat = int(step.get("decision_seat", -1))
        if seat not in (0, 1):
            raise ValueError(f"invalid decision seat in {raw_path} step {step_index}")
        faction_state = (step.get("state") or {}).get("faction_state") or []
        if len(faction_state) != 2 or int(faction_state[0].get("type", -1)) != matchup[seat]:
            raise ValueError(f"decision seat does not match canonical self faction in {raw_path} step {step_index}")
        margin = int(final_vp[seat]) - int(final_vp[1 - seat])
        expected_value = 1 if margin > 0 else -1 if margin < 0 else 0
        if (
            int(step.get("final_vp")) != int(final_vp[seat])
            or int(step.get("vp_margin")) != margin
            or value != expected_value
        ):
            raise ValueError(f"inconsistent result target in {raw_path} step {step_index}")
        expected_next = terminal_hash if step_index + 1 == len(steps) else steps[step_index + 1].get("state_hash")
        if step.get("next_state_hash") != expected_next:
            raise ValueError(f"broken state hash chain in {raw_path} step {step_index}")
    return ShardGame(path=raw_path, matchup=matchup, ply_count=len(steps))


def load_replay(
    paths: list[str | Path], *, expected_engine_commit: str, cache_games: int = 8
) -> ReplayWindow:
    games: list[ShardGame] = []
    matchups: list[tuple[int, int]] = []
    for path in sorted(Path(value).resolve() for value in paths):
        game = _validate_shard(path, expected_engine_commit)
        games.append(game)
        matchups.append(game.matchup)
    return ReplayWindow(games, matchups, cache_games=cache_games)


def _training_batch(examples: list[Example], device: torch.device | str) -> tuple[TensorBatch, Tensor, Tensor]:
    batch = collate_positions([example.encoded for example in examples], device)
    policy = torch.zeros(batch.action_mask.shape, dtype=torch.float32, device=device)
    for index, example in enumerate(examples):
        policy[index, : len(example.policy)] = torch.from_numpy(example.policy).to(device)
    values = torch.tensor([example.value for example in examples], dtype=torch.float32, device=device)
    return batch, policy, values


def alpha_zero_loss(
    model: TerraMysticaNet, batch: TensorBatch, target_policy: Tensor, target_value: Tensor
) -> tuple[Tensor, Tensor, Tensor]:
    logits, values = model(*batch.model_inputs())
    policy_loss = -(target_policy * F.log_softmax(logits, dim=-1)).sum(dim=-1).mean()
    value_loss = F.mse_loss(values, target_value)
    return policy_loss + value_loss, policy_loss, value_loss


def resolve_training_steps(
    *, replay_plies: int, batch_size: int, steps: int | None,
    examples_per_replay_ply: float | None,
) -> int:
    if replay_plies <= 0 or batch_size <= 0:
        raise ValueError("replay plies and batch size must be positive")
    if steps is not None and examples_per_replay_ply is not None:
        raise ValueError("steps and examples per replay ply are mutually exclusive")
    if steps is not None:
        if steps <= 0:
            raise ValueError("steps must be positive when specified")
        return steps
    if examples_per_replay_ply is not None:
        if not math.isfinite(examples_per_replay_ply) or examples_per_replay_ply <= 0:
            raise ValueError("examples per replay ply must be finite and positive when specified")
        requested_examples = math.ceil(examples_per_replay_ply * replay_plies)
        return max(1, math.ceil(requested_examples / batch_size))
    return 100


def losses(model: TerraMysticaNet, examples: list[Example], device: torch.device | str = "cpu") -> dict[str, float]:
    model.eval()
    batch, target_policy, target_value = _training_batch(examples, device)
    with torch.no_grad():
        total, policy_loss, value_loss = alpha_zero_loss(model, batch, target_policy, target_value)
    if not all(torch.isfinite(value) for value in (total, policy_loss, value_loss)):
        raise FloatingPointError("non-finite diagnostic AlphaZero loss")
    return {"policy": float(policy_loss), "value": float(value_loss), "total": float(total)}


def train(
    model: TerraMysticaNet,
    replay: ReplayWindow,
    *,
    steps: int,
    batch_size: int,
    learning_rate: float,
    weight_decay: float,
    seed: int,
    device: torch.device | str = "cpu",
    optimizer_name: str = "sgd",
    momentum: float | None = None,
    optimizer_state: dict[str, Any] | None = None,
) -> tuple[dict[str, dict[str, float]], dict[str, Any]]:
    effective_momentum = 0.9 if optimizer_name == "sgd" and momentum is None else momentum
    if (
        steps <= 0
        or batch_size <= 0
        or learning_rate <= 0
        or weight_decay < 0
        or optimizer_name not in ("sgd", "adamw")
        or (optimizer_name == "sgd" and (effective_momentum is None or not 0 <= effective_momentum < 1))
        or (optimizer_name == "adamw" and momentum is not None)
    ):
        raise ValueError("invalid learner hyperparameters")
    rng = random.Random(seed)
    diagnostic = replay.diagnostic(max(1, min(32, batch_size)))
    model.to(device)
    before = losses(model, diagnostic, device)
    if optimizer_name == "sgd":
        optimizer = torch.optim.SGD(
            model.parameters(), lr=learning_rate, momentum=effective_momentum,
            weight_decay=weight_decay,
        )
    else:
        optimizer = torch.optim.AdamW(
            model.parameters(), lr=learning_rate, weight_decay=weight_decay,
        )
    if optimizer_state is not None:
        optimizer.load_state_dict(optimizer_state)
        for state in optimizer.state.values():
            for key, value in state.items():
                if isinstance(value, Tensor):
                    state[key] = value.to(device)
    model.train()
    for _ in range(steps):
        examples = replay.sample(rng, batch_size)
        batch, target_policy, target_value = _training_batch(examples, device)
        loss, _, _ = alpha_zero_loss(model, batch, target_policy, target_value)
        if not torch.isfinite(loss):
            raise FloatingPointError("non-finite AlphaZero loss")
        optimizer.zero_grad(set_to_none=True)
        loss.backward()
        optimizer.step()
    if not all(torch.isfinite(parameter).all() for parameter in model.parameters()):
        raise FloatingPointError("non-finite model parameters after training")
    after = losses(model, diagnostic, device)
    return {"before": before, "after": after}, optimizer.state_dict()


def optimizer_config(
    name: str, learning_rate: float, weight_decay: float, momentum: float | None
) -> dict[str, Any]:
    return {
        "name": name,
        "learning_rate": learning_rate,
        "weight_decay": weight_decay,
        "momentum": (0.9 if name == "sgd" and momentum is None else momentum),
    }


def resolve_latest(path: str | Path) -> Path:
    latest_path = Path(path)
    latest = json.loads(latest_path.read_text())
    target = latest_path.parent / str(latest.get("path", ""))
    expected = str(latest.get("checkpoint_sha256", ""))
    if not expected or not target.is_file() or hashlib.sha256(target.read_bytes()).hexdigest() != expected:
        raise ValueError(f"invalid latest checkpoint reference {latest_path}")
    return target


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", action="append", required=True, help="trajectory shard or directory")
    parser.add_argument("--output-dir", required=True)
    parser.add_argument("--initial-checkpoint")
    parser.add_argument("--initial-latest")
    parser.add_argument("--model-config", choices=("auto", *MODEL_CONFIGS), default="auto")
    parser.add_argument("--engine-commit", required=True)
    parser.add_argument("--window-shards", type=int, default=10000)
    parser.add_argument("--replay-cache-games", type=int, default=8)
    parser.add_argument("--steps", type=int)
    parser.add_argument("--examples-per-replay-ply", type=float)
    parser.add_argument("--batch-size", type=int, default=32)
    parser.add_argument("--learning-rate", type=float, default=1e-4)
    parser.add_argument("--weight-decay", type=float, default=1e-4)
    parser.add_argument("--optimizer", choices=("sgd", "adamw"), default="sgd")
    parser.add_argument("--momentum", type=float)
    parser.add_argument("--reset-optimizer", action="store_true")
    parser.add_argument("--seed", type=int, default=0)
    parser.add_argument("--model-seed", type=int, default=0)
    parser.add_argument("--device", default="cpu", help="cpu, cuda, mps, or auto")
    parser.add_argument("--torch-threads", type=int, default=1, help="CPU threads; zero keeps the PyTorch default")
    args = parser.parse_args()

    if args.initial_checkpoint and args.initial_latest:
        parser.error("--initial-checkpoint and --initial-latest are mutually exclusive")
    initial_checkpoint = Path(args.initial_checkpoint) if args.initial_checkpoint else None
    if args.initial_latest:
        initial_checkpoint = resolve_latest(args.initial_latest)

    paths: list[Path] = []
    for value in args.input:
        path = Path(value)
        paths.extend(sorted(path.glob("trajectory-*.json.gz")) if path.is_dir() else [path])
    if args.window_shards <= 0:
        parser.error("--window-shards must be positive")
    if args.replay_cache_games <= 0:
        parser.error("--replay-cache-games must be positive")
    paths = sorted(set(paths), key=lambda path: (path.stat().st_mtime_ns, str(path)))[-args.window_shards :]
    if args.torch_threads < 0:
        parser.error("--torch-threads must be non-negative")
    if args.torch_threads:
        torch.set_num_threads(args.torch_threads)
    torch.manual_seed(args.model_seed)
    device = resolve_device(args.device)
    config_name, config = resolve_model_config(args.model_config, initial_checkpoint)
    model = TerraMysticaNet(config)
    parent_checkpoint_sha256 = ""
    parent_optimizer_state = None
    requested_optimizer_config = optimizer_config(
        args.optimizer, args.learning_rate, args.weight_decay, args.momentum
    )
    if initial_checkpoint:
        parent_payload = load_checkpoint(initial_checkpoint, model)
        parent_model_id = str(parent_payload.get("model_id") or "")
        expected_parent_model_id = "model-" + model_fingerprint(model)
        if parent_model_id != expected_parent_model_id:
            raise ValueError(f"parent checkpoint model identity mismatch: {parent_model_id} != {expected_parent_model_id}")
        parent_checkpoint_sha256 = hashlib.sha256(initial_checkpoint.read_bytes()).hexdigest()
        stored_optimizer_config = parent_payload.get("optimizer_config")
        stored_optimizer_state = parent_payload.get("optimizer_state")
        if args.reset_optimizer:
            parent_optimizer_state = None
        elif stored_optimizer_config != requested_optimizer_config or stored_optimizer_state is None:
            raise ValueError(
                "parent checkpoint optimizer is missing or incompatible; "
                "use --reset-optimizer for an intentional recorded reset"
            )
        else:
            parent_optimizer_state = stored_optimizer_state
    else:
        parent_model_id = "random-" + model_fingerprint(model)
        if args.reset_optimizer:
            parser.error("--reset-optimizer requires an initial checkpoint")
    replay_started = time.perf_counter()
    replay = load_replay(
        paths, expected_engine_commit=args.engine_commit, cache_games=args.replay_cache_games
    )
    replay_load_seconds = time.perf_counter() - replay_started
    try:
        training_steps = resolve_training_steps(
            replay_plies=replay.ply_count,
            batch_size=args.batch_size,
            steps=args.steps,
            examples_per_replay_ply=args.examples_per_replay_ply,
        )
    except ValueError as error:
        parser.error(str(error))
    training_started = time.perf_counter()
    metrics, trained_optimizer_state = train(
        model,
        replay,
        steps=training_steps,
        batch_size=args.batch_size,
        learning_rate=args.learning_rate,
        weight_decay=args.weight_decay,
        seed=args.seed,
        device=device,
        optimizer_name=args.optimizer,
        momentum=args.momentum,
        optimizer_state=parent_optimizer_state,
    )
    training_seconds = time.perf_counter() - training_started
    training_examples = training_steps * args.batch_size
    training_throughput = {
        "examples": training_examples,
        "seconds": training_seconds,
        "examples_per_second": training_examples / training_seconds if training_seconds > 0 else 0,
    }
    provenance = {
        "engine_commit": args.engine_commit,
        "learner_seed": args.seed,
        "model_seed": args.model_seed,
        "model_config": config_name,
        "device": str(device),
        "dtype": "float32",
        "torch_threads": torch.get_num_threads(),
        "parent_model_id": parent_model_id,
        "parent_checkpoint_sha256": parent_checkpoint_sha256,
        "optimizer_reset": bool(initial_checkpoint and args.reset_optimizer),
        "replay_shards": [path.name.removeprefix("trajectory-").removesuffix(".json.gz") for path in paths],
        "replay": {
            "games": replay.game_count,
            "plies": replay.ply_count,
            "cache_games": replay.cache_games,
            "resident_games": replay.resident_game_count,
            "materialization_loads": replay.materialization_loads,
            "materialization_cache_hits": replay.materialization_cache_hits,
            "validation_seconds": replay_load_seconds,
        },
        "training": {
            "steps": training_steps,
            "batch_size": args.batch_size,
            "requested_examples_per_replay_ply": args.examples_per_replay_ply,
            "actual_examples_per_replay_ply": training_examples / replay.ply_count,
            "learning_rate": args.learning_rate,
            "weight_decay": args.weight_decay,
            "optimizer": requested_optimizer_config["name"],
            "momentum": requested_optimizer_config["momentum"],
            **training_throughput,
            **metrics,
        },
    }
    published = publish_checkpoint(
        args.output_dir,
        model,
        provenance,
        optimizer_state=trained_optimizer_state,
        optimizer_config=requested_optimizer_config,
    )
    print(json.dumps({
        "checkpoint": published["path"], "model_id": published["model_id"],
        "steps": training_steps,
        "actual_examples_per_replay_ply": training_examples / replay.ply_count,
        "throughput": training_throughput, **metrics,
    }, sort_keys=True))


if __name__ == "__main__":
    main()
