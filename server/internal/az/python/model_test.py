#!/usr/bin/env python3

from __future__ import annotations

import copy
import gc
import gzip
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

import numpy as np
import torch
from torch.nn import functional as F

from batching import collate_positions
from benchmark import benchmark as run_model_benchmark
import learner as learner_module
from learner import Example, ReplayWindow, load_replay, losses, resolve_training_steps, train
from model import DEBUG_CONFIG, LARGE_CONFIG, MAIN_CONFIG, HexConv2d, TerraMysticaNet, checkpoint_manifest, load_checkpoint, publish_checkpoint, resolve_device, resolve_model_config, save_checkpoint
from schema import (
    ACTION_FEATURE_NAMES,
    AXIAL_NEIGHBORS,
    BASE_ACTION_KINDS,
    BASE_SPECIALS,
    GLOBAL_FEATURE_NAMES,
    MANIFEST,
    SPATIAL_FEATURE_NAMES,
    encode_actions,
    encode_position,
    validate_position_schema,
)


def _fixture(binary: str, seed: int = 20260810, first_id: str = "p0", second_id: str = "p1") -> dict:
    output = subprocess.check_output([binary, str(seed), first_id, second_id], text=True)
    return json.loads(output)


def _action_samples() -> list[dict]:
    first, second = {"Q": 0, "R": 0}, {"Q": 1, "R": 0}
    actions = [
        {"kind": 0, "hexes": [first], "terrain": 3, "build": True, "use_skip": True},
        {"kind": 1, "hexes": [first], "building": 2},
        {"kind": 2}, {"kind": 3},
        {"kind": 5, "track": 3, "amount": 2},
        {"kind": 6, "power": 2},
        {"kind": 8, "card": -1},
        {"kind": 9, "hexes": [first]},
        {"kind": 10, "hexes": [first], "terrain": 1},
        {"kind": 11, "amount": 1, "amount_2": 2},
        {"kind": 12, "amount": 1},
        {"kind": 13, "favor_tile": 11},
        {"kind": 14, "hexes": [first], "terrain": 0},
        {"kind": 15, "hexes": [first]},
        {"kind": 16},
        {"kind": 20, "amount": 2},
        {"kind": 21, "track": 1},
        {"kind": 26, "faction": 1},
        {"kind": 30, "card": 4},
        {"kind": 31, "town_tile": 2, "hexes": [first]},
        {"kind": 32, "tracks": [0, 2]},
        {"kind": 33, "amount": 2},
        {"kind": 34, "conversion": "worker_to_coin", "amount": 3},
        {"kind": 35, "amount": 2},
        {"kind": 36, "hexes": [first, second]},
    ]
    specials = [
        {"kind": 7, "special": 0, "track": 0},
        {"kind": 7, "special": 1, "hexes": [first]},
        {"kind": 7, "special": 3, "hexes": [first]},
        {"kind": 7, "special": 4},
        {"kind": 7, "special": 5, "hexes": [first], "build": True},
        {"kind": 7, "special": 6, "hexes": [first], "build": True},
        {"kind": 7, "special": 7, "track": 1},
        {"kind": 7, "special": 8, "hexes": [first], "terrain": 2, "build": True, "use_skip": True},
        {"kind": 7, "special": 9, "track": 2},
        {"kind": 7, "special": 10, "hexes": [first]},
    ]
    return actions + specials


def _tensor_batch(encoded_items):
    spatial = torch.tensor(np.stack([item.spatial for item in encoded_items]))
    global_features = torch.tensor(np.stack([item.global_features for item in encoded_items]))
    action_features = torch.tensor(np.stack([item.action_features for item in encoded_items]))
    action_hex_indices = torch.tensor(np.stack([item.action_hex_indices for item in encoded_items]))
    action_mask = torch.ones(action_features.shape[:2], dtype=torch.bool)
    return spatial, global_features, action_features, action_hex_indices, action_mask


def _accelerator_device() -> torch.device | None:
    if torch.cuda.is_available():
        return torch.device("cuda")
    if torch.backends.mps.is_available():
        return torch.device("mps")
    return None


class RepresentationTest(unittest.TestCase):
    fixture_binary: str
    inference_binary: str

    @classmethod
    def setUpClass(cls) -> None:
        torch.set_num_threads(1)
        torch.manual_seed(7)

    def test_versioned_golden_schema_covers_every_base_action(self) -> None:
        actions = _action_samples()
        record = _fixture(self.fixture_binary)["state"]
        for action in actions:
            validate_position_schema(record, [action])
        matrix, gathers, names = encode_actions(actions)
        self.assertEqual(set(BASE_ACTION_KINDS), {action["kind"] for action in actions})
        self.assertEqual(set(BASE_SPECIALS), {action.get("special") for action in actions if action["kind"] == 7})
        self.assertEqual(matrix.shape, (len(actions), MANIFEST.action_features))
        self.assertEqual(gathers.shape, (len(actions), 2))
        self.assertEqual(len({row.tobytes() for row in matrix}), len(actions))
        self.assertEqual(names, ACTION_FEATURE_NAMES)
        self.assertEqual(
            hashlib.sha256("\n".join(SPATIAL_FEATURE_NAMES).encode()).hexdigest(),
            "62fea181998d7c0742a9774329dcbb3b47cd64bd7ef0e2b1b9f8c6233a7099cd",
        )
        self.assertEqual(
            hashlib.sha256("\n".join(GLOBAL_FEATURE_NAMES).encode()).hexdigest(),
            "8f7030c42eff327c38a4229000139407f310a65e9afcf5c80077a0f82f49b25e",
        )
        self.assertEqual(
            hashlib.sha256("\n".join(ACTION_FEATURE_NAMES).encode()).hexdigest(),
            "af60817febe08613a91087f0c7660f8d8de9283bdeacadd87069d2574d46e146",
        )

    def test_each_typed_action_argument_changes_features_or_gathers(self) -> None:
        first, second = {"Q": 0, "R": 4}, {"Q": 1, "R": 4}
        pairs = [
            ({"kind": 0, "hexes": [first], "terrain": 1}, {"kind": 0, "hexes": [first], "terrain": 2}),
            ({"kind": 0, "hexes": [first]}, {"kind": 0, "hexes": [first], "build": True}),
            ({"kind": 0, "hexes": [first]}, {"kind": 0, "hexes": [first], "use_skip": True}),
            ({"kind": 0, "hexes": [first]}, {"kind": 0, "hexes": [second]}),
            ({"kind": 1, "hexes": [first], "building": 1}, {"kind": 1, "hexes": [first], "building": 2}),
            ({"kind": 5, "track": 0, "amount": 2}, {"kind": 5, "track": 1, "amount": 2}),
            ({"kind": 6, "power": 0}, {"kind": 6, "power": 1}),
            ({"kind": 7, "special": 0}, {"kind": 7, "special": 1}),
            ({"kind": 8, "card": 0}, {"kind": 8, "card": 1}),
            ({"kind": 11, "amount": 1, "amount_2": 0}, {"kind": 11, "amount": 1, "amount_2": 1}),
            ({"kind": 13, "favor_tile": 0}, {"kind": 13, "favor_tile": 1}),
            ({"kind": 26, "faction": 1}, {"kind": 26, "faction": 2}),
            ({"kind": 31, "town_tile": 0, "hexes": [first]}, {"kind": 31, "town_tile": 1, "hexes": [first]}),
            ({"kind": 32, "tracks": [0]}, {"kind": 32, "tracks": [1]}),
            ({"kind": 34, "conversion": "power_to_coin", "amount": 1}, {"kind": 34, "conversion": "worker_to_coin", "amount": 1}),
            ({"kind": 34, "conversion": "worker_to_coin", "amount": 1}, {"kind": 34, "conversion": "worker_to_coin", "amount": 2}),
            ({"kind": 36, "hexes": [first, second]}, {"kind": 36, "hexes": [second, first]}),
        ]
        for left, right in pairs:
            features, gathers, _ = encode_actions([left, right])
            self.assertTrue(
                not np.array_equal(features[0], features[1]) or not np.array_equal(gathers[0], gathers[1]),
                (left, right),
            )
        _, gathers, _ = encode_actions([
            {"kind": 0, "hexes": [{"Q": 0, "R": 4}]},
            {"kind": 0, "hexes": [{"Q": 1, "R": 4}]},
        ])
        np.testing.assert_array_equal(gathers, np.asarray([[72, -1], [73, -1]]))

    def test_go_canonical_record_encodes_and_is_player_id_relative(self) -> None:
        ordinary = _fixture(self.fixture_binary)
        renamed = _fixture(self.fixture_binary, first_id="alice", second_id="bob")
        self.assertEqual(ordinary, renamed)
        encoded = encode_position(ordinary["state"], ordinary["actions"])
        renamed_encoded = encode_position(renamed["state"], renamed["actions"])
        np.testing.assert_array_equal(encoded.spatial, renamed_encoded.spatial)
        np.testing.assert_array_equal(encoded.global_features, renamed_encoded.global_features)
        np.testing.assert_array_equal(encoded.action_features, renamed_encoded.action_features)
        np.testing.assert_array_equal(encoded.action_hex_indices, renamed_encoded.action_hex_indices)
        self.assertEqual(encoded.spatial.shape, (MANIFEST.spatial_features, MANIFEST.grid_height, MANIFEST.grid_width))
        self.assertEqual(encoded.global_features.shape, (MANIFEST.global_features,))
        self.assertGreater(encoded.action_features.shape[0], 0)
        self.assertEqual(float(encoded.spatial[0].sum()), 113.0)

    def test_hex_kernel_and_mask_exclude_invalid_cells(self) -> None:
        convolution = HexConv2d(1, 1, bias=False)
        self.assertEqual(int(convolution.kernel_mask.sum()), 7)
        self.assertEqual(AXIAL_NEIGHBORS, ((1, 0), (1, -1), (0, -1), (-1, 0), (-1, 1), (0, 1)))
        nonzero = {tuple(index.tolist()) for index in torch.nonzero(convolution.kernel_mask[0, 0])}
        self.assertEqual(nonzero, {(1, 1), (1, 2), (0, 2), (0, 1), (1, 0), (2, 0), (2, 1)})
        payload = _fixture(self.fixture_binary)
        encoded = encode_position(payload["state"], payload["actions"])
        batch = _tensor_batch([encoded])
        model = TerraMysticaNet(DEBUG_CONFIG).eval()
        with torch.no_grad():
            expected = model(*batch)
            changed_spatial = batch[0].clone()
            invalid = (batch[0][:, :1] == 0).expand_as(changed_spatial[:, 1:])
            changed_spatial[:, 1:] = torch.where(
                invalid, torch.full_like(changed_spatial[:, 1:], 1000), changed_spatial[:, 1:]
            )
            actual = model(changed_spatial, *batch[1:])
        torch.testing.assert_close(actual[0], expected[0])
        torch.testing.assert_close(actual[1], expected[1])

    def test_bridge_pairing_and_representative_state_families_are_encoded(self) -> None:
        payload = _fixture(self.fixture_binary)
        record = payload["state"]
        baseline = encode_position(record, payload["actions"])

        first_pairing = copy.deepcopy(record)
        second_pairing = copy.deepcopy(record)
        first_pairing["state"]["map"]["bridges"] = {
            "0,4|1,2": "self",
            "2,3|3,1": "self",
        }
        second_pairing["state"]["map"]["bridges"] = {
            "0,4|2,3": "self",
            "1,2|3,1": "self",
        }
        first_spatial = encode_position(first_pairing, payload["actions"]).spatial
        second_spatial = encode_position(second_pairing, payload["actions"]).spatial
        self.assertEqual(float(first_spatial.sum()), float(second_spatial.sum()))
        self.assertFalse(np.array_equal(first_spatial, second_spatial))

        mutated = copy.deepcopy(record)
        state = mutated["state"]
        map_hex = next(value for value in state["map"]["hexes"].values() if value["Terrain"] != 7)
        map_hex["Building"] = {"type": 0, "faction": 7, "playerId": "self", "powerValue": 1}
        state["cultTracks"]["playerPositions"]["self"]["0"] = 4
        state["bonusCards"]["available"]["0"] = 7
        state["pendingSpades"]["self"] = 2
        state["pendingSpadeBuildAllowed"]["self"] = True
        state["pendingTownFormations"]["self"] = [{
            "PlayerID": "self", "Hexes": [{"Q": 0, "R": 4}], "CanBeDelayed": False,
        }]
        encoded = encode_position(mutated, payload["actions"])
        self.assertFalse(np.array_equal(encoded.spatial, baseline.spatial))
        self.assertFalse(np.array_equal(encoded.global_features, baseline.global_features))
        names = {name: index for index, name in enumerate(GLOBAL_FEATURE_NAMES)}
        self.assertEqual(encoded.global_features[names["self_cult_0:scalar/10"]], 0.4)
        self.assertGreater(encoded.global_features[names["bonus_coins_0:scalar/12"]], 0)
        self.assertGreater(encoded.global_features[names["pending_spades_self:scalar/3"]], 0)
        pending_plane = SPATIAL_FEATURE_NAMES.index("pending_town_immediate_self")
        self.assertEqual(float(encoded.spatial[pending_plane].sum()), 1.0)

        wrong_version = copy.deepcopy(record)
        wrong_version["state_version"] += 1
        with self.assertRaisesRegex(ValueError, "state schema mismatch"):
            encode_position(wrong_version, payload["actions"])

    def test_planned_network_parameter_budgets(self) -> None:
        debug_parameters = sum(parameter.numel() for parameter in TerraMysticaNet(DEBUG_CONFIG).parameters())
        main_parameters = sum(parameter.numel() for parameter in TerraMysticaNet(MAIN_CONFIG).parameters())
        large_parameters = sum(parameter.numel() for parameter in TerraMysticaNet(LARGE_CONFIG).parameters())
        self.assertGreaterEqual(debug_parameters, 500_000)
        self.assertLessEqual(debug_parameters, 1_000_000)
        self.assertGreaterEqual(main_parameters, 3_000_000)
        self.assertLessEqual(main_parameters, 5_000_000)
        self.assertGreaterEqual(large_parameters, 25_000_000)
        self.assertLessEqual(large_parameters, 40_000_000)

    def test_training_work_can_be_bound_to_replay_plies(self) -> None:
        self.assertEqual(resolve_training_steps(
            replay_plies=318, batch_size=32, steps=None, examples_per_replay_ply=0.25,
        ), 3)
        self.assertEqual(resolve_training_steps(
            replay_plies=318, batch_size=32, steps=None, examples_per_replay_ply=1,
        ), 10)
        self.assertEqual(resolve_training_steps(
            replay_plies=318, batch_size=32, steps=None, examples_per_replay_ply=None,
        ), 100)
        with self.assertRaisesRegex(ValueError, "mutually exclusive"):
            resolve_training_steps(
                replay_plies=318, batch_size=32, steps=2, examples_per_replay_ply=1,
            )
        for steps, ratio in ((0, None), (None, 0), (None, float("inf"))):
            with self.assertRaises(ValueError):
                resolve_training_steps(
                    replay_plies=318, batch_size=32, steps=steps,
                    examples_per_replay_ply=ratio,
                )

    def test_model_benchmark_is_bound_to_checkpoint_identity_and_shape(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            torch.manual_seed(41)
            published = publish_checkpoint(directory, TerraMysticaNet(DEBUG_CONFIG), {"test": True})
            result = run_model_benchmark(
                config_name="auto", device_name="cpu", mode="inference",
                batch_size=1, action_count=3, warmup=0, iterations=1, seed=0,
                checkpoint=published["path"],
            )
            self.assertEqual(result["model_config"], "debug")
            self.assertEqual(result["model_id"], published["model_id"])
            self.assertEqual(result["checkpoint_sha256"], published["checkpoint_sha256"])
            with self.assertRaisesRegex(ValueError, "does not match requested"):
                run_model_benchmark(
                    config_name="large", device_name="cpu", mode="inference",
                    batch_size=1, action_count=3, warmup=0, iterations=1, seed=0,
                    checkpoint=published["path"],
                )

    def test_large_network_runs_exact_ragged_forward_and_backward(self) -> None:
        payload = _fixture(self.fixture_binary)
        first = encode_position(payload["state"], payload["actions"][:3])
        second = encode_position(payload["state"], payload["actions"][:1])
        batch = collate_positions([first, second])
        model = TerraMysticaNet(LARGE_CONFIG)
        logits, values = model(*batch.model_inputs())
        self.assertEqual(logits.shape, (2, 3))
        self.assertEqual(values.shape, (2,))
        self.assertTrue(torch.isfinite(logits[0]).all())
        self.assertTrue((logits[1, 1:] < -1e20).all())
        self.assertTrue(torch.isfinite(values).all())
        loss = F.cross_entropy(logits, torch.tensor([0, 0])) + values.square().mean()
        loss.backward()
        self.assertTrue(all(parameter.grad is None or torch.isfinite(parameter.grad).all() for parameter in model.parameters()))

    def test_explicit_device_resolution(self) -> None:
        self.assertEqual(resolve_device("cpu"), torch.device("cpu"))
        with self.assertRaisesRegex(ValueError, "unsupported device"):
            resolve_device("tpu")

    def test_accelerator_forward_matches_cpu_when_available(self) -> None:
        device = _accelerator_device()
        if device is None:
            self.skipTest("no CUDA or MPS accelerator")
        payload = _fixture(self.fixture_binary)
        encoded = encode_position(payload["state"], payload["actions"][:5])
        torch.manual_seed(19)
        cpu_model = TerraMysticaNet(DEBUG_CONFIG).eval()
        accelerated_model = TerraMysticaNet(DEBUG_CONFIG).eval().to(device)
        accelerated_model.load_state_dict(cpu_model.state_dict())
        with torch.inference_mode():
            cpu_logits, cpu_values = cpu_model(*collate_positions([encoded]).model_inputs())
            accelerated_logits, accelerated_values = accelerated_model(
                *collate_positions([encoded], device).model_inputs()
            )
        torch.testing.assert_close(accelerated_logits.cpu(), cpu_logits, rtol=2e-4, atol=2e-4)
        torch.testing.assert_close(accelerated_values.cpu(), cpu_values, rtol=2e-4, atol=2e-4)
        accelerated_model.train()
        optimizer = torch.optim.AdamW(accelerated_model.parameters(), lr=1e-3)
        logits, values = accelerated_model(*collate_positions([encoded], device).model_inputs())
        loss = F.cross_entropy(logits, torch.tensor([0], device=device)) + values.square().mean()
        optimizer.zero_grad(set_to_none=True)
        loss.backward()
        optimizer.step()
        self.assertTrue(all(torch.isfinite(parameter).all() for parameter in accelerated_model.parameters()))

    def test_large_accelerator_checkpoint_lifecycle(self) -> None:
        device = _accelerator_device()
        if device is None:
            self.skipTest("no CUDA or MPS accelerator")
        payload = _fixture(self.fixture_binary)
        actions = payload["actions"][:3]
        encoded = encode_position(payload["state"], actions)
        replay = ReplayWindow(
            [[Example(encoded, np.asarray([1.0, 0.0, 0.0], dtype=np.float32), 0.5)]],
            [(7, 13)],
        )
        with tempfile.TemporaryDirectory() as directory:
            model = TerraMysticaNet(LARGE_CONFIG)
            metrics = train(
                model, replay, steps=1, batch_size=1, learning_rate=1e-4,
                weight_decay=1e-4, seed=31, device=device,
            )
            self.assertTrue(np.isfinite(list(metrics["after"].values())).all())
            published = publish_checkpoint(directory, model, {"device": str(device), "test": True})
            checkpoint = Path(published["path"])
            self.assertEqual(resolve_model_config("auto", checkpoint), ("large", LARGE_CONFIG))
            del model
            gc.collect()
            if device.type == "cuda":
                torch.cuda.empty_cache()
            elif device.type == "mps":
                torch.mps.empty_cache()
            result = subprocess.run(
                [
                    self.inference_binary, "--checkpoint", str(checkpoint),
                    "--model-config", "auto", "--device", str(device), "--torch-threads", "1",
                ],
                input=json.dumps({"type": "metadata"}) + "\n" + json.dumps({
                    "type": "evaluate", "requests": [{"state": payload["state"], "actions": actions}],
                }) + "\n",
                capture_output=True, text=True, timeout=60,
            )
        self.assertEqual(result.returncode, 0, result.stderr)
        metadata_response, evaluation_response = [json.loads(line) for line in result.stdout.splitlines()]
        metadata = metadata_response["metadata"]
        self.assertEqual(metadata["model_config"], "large")
        self.assertEqual(metadata["device"], str(device))
        self.assertEqual(metadata["model_id"], published["model_id"])
        self.assertNotIn("error", evaluation_response)
        self.assertEqual(len(evaluation_response["results"][0]["policy_logits"]), len(actions))
        self.assertTrue(-1 <= evaluation_response["results"][0]["value"] <= 1)

    def test_ragged_legal_action_mask(self) -> None:
        payload = _fixture(self.fixture_binary)
        encoded = encode_position(payload["state"], payload["actions"])
        spatial, global_features, action_features, action_hex_indices, _ = _tensor_batch([encoded, encoded])
        action_mask = torch.ones(action_features.shape[:2], dtype=torch.bool)
        action_mask[1, len(payload["actions"]) // 2 :] = False
        model = TerraMysticaNet(DEBUG_CONFIG).eval()
        logits, values = model(spatial, global_features, action_features, action_hex_indices, action_mask)
        self.assertEqual(logits.shape, action_mask.shape)
        self.assertEqual(values.shape, (2,))
        self.assertTrue(torch.isfinite(logits[0]).all())
        self.assertTrue((logits[1, ~action_mask[1]] < -1e20).all())

    def test_shared_collator_pads_ragged_actions(self) -> None:
        payload = _fixture(self.fixture_binary)
        first = encode_position(payload["state"], payload["actions"])
        second = encode_position(payload["state"], payload["actions"][:3])
        batch = collate_positions([first, second])
        self.assertEqual(batch.action_mask.shape, (2, len(payload["actions"])))
        self.assertTrue(batch.action_mask[0].all())
        self.assertEqual(int(batch.action_mask[1].sum()), 3)
        model = TerraMysticaNet(DEBUG_CONFIG).eval()
        logits, values = model(*batch.model_inputs())
        self.assertTrue(torch.isfinite(logits[0]).all())
        self.assertTrue((logits[1, 3:] < -1e20).all())
        self.assertTrue(torch.isfinite(values).all())

    def test_long_lived_inference_protocol_batches_real_records(self) -> None:
        payload = _fixture(self.fixture_binary)
        process = subprocess.Popen(
            [self.inference_binary, "--model-config", "debug", "--seed", "17"],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )
        assert process.stdin is not None and process.stdout is not None
        process.stdin.write(json.dumps({"type": "metadata"}) + "\n")
        process.stdin.flush()
        metadata = json.loads(process.stdout.readline())["metadata"]
        self.assertEqual(metadata["model_config"], "debug")
        self.assertEqual(metadata["device"], "cpu")
        self.assertEqual(metadata["dtype"], "float32")
        self.assertGreater(metadata["parameters"], 0)
        self.assertEqual(metadata["torch_threads"], 1)
        request = {"type": "evaluate", "requests": [payload, payload]}
        process.stdin.write(json.dumps(request) + "\n")
        process.stdin.flush()
        response = json.loads(process.stdout.readline())
        process.stdin.close()
        self.assertEqual(process.wait(timeout=20), 0, process.stderr.read() if process.stderr else "")
        self.assertNotIn("error", response)
        self.assertEqual(len(response["results"]), 2)
        for result in response["results"]:
            self.assertEqual(len(result["policy_logits"]), len(payload["actions"]))
            self.assertTrue(np.isfinite(result["policy_logits"]).all())
            self.assertTrue(-1 <= result["value"] <= 1)

    def test_learner_consumes_hashed_raw_shard_and_reduces_policy_loss(self) -> None:
        payload = _fixture(self.fixture_binary)
        actions = payload["actions"][:4]
        state_bytes = json.dumps(payload["state"], separators=(",", ":"), ensure_ascii=False).encode()
        state_hash = hashlib.sha256(state_bytes).hexdigest()[:32]
        terminal_state = copy.deepcopy(payload["state"])
        terminal_state["state"]["phase"] = 5
        terminal_bytes = json.dumps(terminal_state, separators=(",", ":"), ensure_ascii=False).encode()
        terminal_hash = hashlib.sha256(terminal_bytes).hexdigest()[:32]
        trajectory = {
            "manifest": {
                "format_version": 1,
                "rules_version": 1,
                "state_version": 1,
                "action_version": 1,
                "engine_commit": "test",
                "model_id": "random",
                "seed": 3,
                "factions": [7, 13],
            },
            "steps": [
                {
                    "state": payload["state"],
                    "state_hash": state_hash,
                    "actions": actions,
                    "visits": [40, 0, 0, 0],
                    "chosen_index": 0,
                    "decision_seat": 0,
                    "next_state_hash": terminal_hash,
                    "outcome": 1,
                    "final_vp": 100,
                    "vp_margin": 10,
                }
            ],
            "final_vp": [100, 90],
            "ply_count": 1,
            "completed": True,
            "terminal_state": terminal_state,
            "terminal_state_hash": terminal_hash,
        }
        compressed = gzip.compress(json.dumps(trajectory, separators=(",", ":")).encode(), mtime=0)
        digest = hashlib.sha256(compressed).hexdigest()
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / f"trajectory-{digest}.json.gz"
            path.write_bytes(compressed)
            with mock.patch.object(learner_module, "encode_position", wraps=encode_position) as encode_mock:
                lazy_probe = load_replay([path], expected_engine_commit="test", cache_games=1)
                self.assertEqual(encode_mock.call_count, 0)
                lazy_probe._example(0, 0)
                self.assertEqual(encode_mock.call_count, 1)
            replay = load_replay([path], expected_engine_commit="test", cache_games=1)
            self.assertEqual(replay.game_count, 1)
            self.assertEqual(replay.ply_count, 1)
            self.assertEqual(replay.resident_game_count, 0)
            mutations = [
                ("state schema mismatch", lambda value: value["steps"][0]["state"].__setitem__("state_version", 999)),
                ("state schema mismatch", lambda value: value["steps"][0]["state"].__setitem__("state_version", True)),
                ("invalid scalar feature self_coins", lambda value: value["steps"][0]["state"]["state"]["players"]["self"]["resources"].__setitem__("coins", "bad")),
                ("non-finite scalar feature amount", lambda value: value["steps"][0]["actions"].__setitem__(0, {"kind": 34, "conversion": "worker_to_coin", "amount": 10**40})),
                ("unsupported v0 action kind", lambda value: value["steps"][0]["actions"].__setitem__(0, {"kind": 999})),
                ("invalid terrain", lambda value: value["steps"][0]["actions"].__setitem__(0, {"kind": 0, "hexes": [{"Q": 0, "R": 0}], "terrain": 999})),
                ("invalid hex coordinate", lambda value: value["steps"][0]["actions"].__setitem__(0, {"kind": 0, "hexes": [{"Q": 99, "R": 99}], "terrain": 1})),
                ("state hash mismatch", lambda value: value["steps"][0].__setitem__("state_hash", "0" * 32)),
                ("broken state hash chain", lambda value: value["steps"][0].__setitem__("next_state_hash", "0" * 32)),
                ("invalid policy target", lambda value: value["steps"][0].__setitem__("visits", [40, -1, 0, 0])),
                ("decision seat", lambda value: value["steps"][0].__setitem__("decision_seat", 1)),
                ("inconsistent result", lambda value: value["steps"][0].__setitem__("outcome", 0)),
                ("engine commit", lambda value: value["manifest"].__setitem__("engine_commit", "")),
                ("engine commit mismatch", lambda value: value["manifest"].__setitem__("engine_commit", "other")),
                ("model identity", lambda value: value["manifest"].__setitem__("model_id", "")),
                ("checkpoint identity", lambda value: value["manifest"].__setitem__("model_id", "model-" + "a" * 64)),
            ]
            for expected, mutate in mutations:
                corrupted = copy.deepcopy(trajectory)
                mutate(corrupted)
                corrupt_bytes = gzip.compress(json.dumps(corrupted, separators=(",", ":")).encode(), mtime=0)
                corrupt_hash = hashlib.sha256(corrupt_bytes).hexdigest()
                corrupt_path = Path(directory) / f"trajectory-{corrupt_hash}.json.gz"
                corrupt_path.write_bytes(corrupt_bytes)
                with self.assertRaisesRegex(ValueError, expected):
                    load_replay([corrupt_path], expected_engine_commit="test")
            second_trajectory = copy.deepcopy(trajectory)
            second_trajectory["manifest"]["seed"] = 4
            second_bytes = gzip.compress(json.dumps(second_trajectory, separators=(",", ":")).encode(), mtime=0)
            second_hash = hashlib.sha256(second_bytes).hexdigest()
            second_path = Path(directory) / f"trajectory-{second_hash}.json.gz"
            second_path.write_bytes(second_bytes)
            bounded = load_replay(
                [path, second_path], expected_engine_commit="test", cache_games=1
            )
            self.assertEqual(bounded.resident_game_count, 0)
            bounded._example(0, 0)
            bounded._example(1, 0)
            self.assertEqual(bounded.resident_game_count, 1)
            bounded._example(0, 0)
            self.assertEqual(bounded.materialization_loads, 3)
            model = TerraMysticaNet(DEBUG_CONFIG)
            metrics = train(
                model,
                replay,
                steps=50,
                batch_size=2,
                learning_rate=0.003,
                weight_decay=1e-4,
                seed=5,
            )
            self.assertLessEqual(replay.resident_game_count, 1)
            self.assertGreater(replay.materialization_loads, 0)
            self.assertGreater(replay.materialization_cache_hits, 0)
            with torch.no_grad():
                next(model.parameters()).fill_(float("nan"))
            with self.assertRaisesRegex(FloatingPointError, "non-finite diagnostic"):
                losses(model, replay.diagnostic(1))
        self.assertTrue(np.isfinite(list(metrics["after"].values())).all())
        self.assertLess(metrics["after"]["policy"], metrics["before"]["policy"] * 0.1, metrics)
        self.assertLess(metrics["after"]["value"], metrics["before"]["value"], metrics)

    def test_inference_rejects_mislabeled_checkpoint_identity(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            checkpoint = Path(directory) / "bad.pt"
            save_checkpoint(checkpoint, TerraMysticaNet(DEBUG_CONFIG), model_id="model-not-the-weights")
            result = subprocess.run(
                [self.inference_binary, "--checkpoint", str(checkpoint)],
                input='{"type":"metadata"}\n',
                capture_output=True,
                text=True,
                timeout=20,
            )
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("model identity mismatch", result.stderr)

    def test_debug_network_overfits_tiny_real_encoded_dataset(self) -> None:
        items = []
        for seed in (101, 202, 303, 404):
            payload = _fixture(self.fixture_binary, seed=seed)
            items.append(encode_position(payload["state"], payload["actions"]))
        batch = _tensor_batch(items)
        model = TerraMysticaNet(DEBUG_CONFIG)
        optimizer = torch.optim.AdamW(model.parameters(), lr=0.003, weight_decay=1e-4)
        policy_targets = torch.tensor([0, 1, 2, 3])
        value_targets = torch.tensor([-1.0, -0.35, 0.35, 1.0])
        for _ in range(300):
            logits, values = model(*batch)
            loss = F.cross_entropy(logits, policy_targets) + 2 * F.mse_loss(values, value_targets)
            optimizer.zero_grad(set_to_none=True)
            loss.backward()
            optimizer.step()
        model.eval()
        with torch.no_grad():
            logits, values = model(*batch)
            policy_loss = F.cross_entropy(logits, policy_targets).item()
            value_error = torch.mean(torch.abs(values - value_targets)).item()
        self.assertLess(policy_loss, 0.02)
        self.assertLess(value_error, 0.04, values.tolist())

    def test_checkpoint_rejects_schema_mismatch(self) -> None:
        model = TerraMysticaNet(DEBUG_CONFIG)
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "model.pt"
            save_checkpoint(path, model, model_id="debug-test")
            with self.assertRaisesRegex(FileExistsError, "immutable checkpoint"):
                save_checkpoint(path, model, model_id="replacement")
            loaded = TerraMysticaNet(DEBUG_CONFIG)
            payload = load_checkpoint(path, loaded)
            self.assertEqual(payload["model_id"], "debug-test")
            self.assertEqual(resolve_model_config("auto", path), ("debug", DEBUG_CONFIG))
            with self.assertRaisesRegex(ValueError, "does not match requested"):
                resolve_model_config("main", path)
            mutations = [
                ("rules_version", 2),
                ("grid_q_offset", 5),
                ("spatial_schema_sha256", "same-dimension-but-reordered"),
                ("action_schema_sha256", "same-dimension-but-reordered"),
            ]
            for field, value in mutations:
                wrong = copy.deepcopy(checkpoint_manifest(loaded))
                wrong["schema"][field] = value
                with self.assertRaisesRegex(ValueError, "manifest mismatch"):
                    load_checkpoint(path, loaded, expected_manifest=wrong)


if __name__ == "__main__":
    fixture_argument = sys.argv[1]
    inference_argument = sys.argv[2]
    sys.argv = [sys.argv[0]]
    RepresentationTest.fixture_binary = fixture_argument
    RepresentationTest.inference_binary = inference_argument
    unittest.main()
