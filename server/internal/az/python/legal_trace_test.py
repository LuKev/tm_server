"""Materialize real Go legal trajectories; do not synthesize game state here."""

import json
import subprocess
import sys
import unittest

import numpy as np

from schema import encode_position, GLOBAL_FEATURE_NAMES, MANIFEST


class LegalTraceTest(unittest.TestCase):
    def test_natural_games_materialize_pending_boundaries(self):
        result = subprocess.run([self.fixture_binary, "--traces"],
                                capture_output=True, text=True, timeout=180)
        self.assertEqual(result.returncode, 0, result.stderr)
        games = [json.loads(line) for line in result.stdout.splitlines()]
        self.assertEqual(len(games), 14)
        factions, observed = set(), set()
        names = {name: index for index, name in enumerate(GLOBAL_FEATURE_NAMES)}
        for game in games:
            factions.update(game["factions"])
            self.assertTrue(game["terminal"])
            self.assertLess(len(game["trace"]), 1000)
            rounds = set()
            for sample in game["samples"]:
                with self.subTest(seed=game["seed"], ply=sample["ply"]):
                    record, actions = sample["state"], sample["actions"]
                    state = record["state"]
                    rounds.add(state["round"])
                    self.assertIn(game["trace"][sample["ply"]], actions)
                    encoded = encode_position(record, actions)
                    self.assertEqual(encoded.global_features.shape, (MANIFEST.global_features,))
                    for array in (encoded.spatial, encoded.global_features, encoded.action_features):
                        self.assertTrue(np.isfinite(array).all())
                    # Distinct legal choices must not collapse to identical
                    # action tensors AND referenced-cell indices.
                    representations = {
                        (features.tobytes(), indices.tobytes())
                        for features, indices in zip(encoded.action_features, encoded.action_hex_indices)
                    }
                    self.assertEqual(len(representations), len(actions))
                    for field, flag in (
                        ("pendingHalflingsSpades", "pending_halflings"),
                        ("pendingChaosMagiciansDoubleTurn", "pending_chaos"),
                        ("pendingFavorTileSelection", "pending_favor"),
                        ("pendingDarklingsPriestOrdination", "pending_darklings"),
                        ("pendingCultistsCultSelection", "pending_cultists"),
                    ):
                        if state.get(field):
                            observed.add(field)
                            self.assertEqual(encoded.global_features[names[flag + ":flag"]], 1)
                            owner_key = "playerId" if "Chaos" in field else "PlayerID"
                            owner = 0 if state[field][owner_key] == "self" else 1
                            self.assertEqual(encoded.global_features[names[f"{field}_owner:one_hot={owner}"]], 1)
                    if any(state.get("pendingLeechOffers", {}).values()):
                        observed.add("pendingLeechOffers")
                        for role, offers in state["pendingLeechOffers"].items():
                            for index, offer in enumerate(offers or []):
                                prefix = f"leech_{role}_{index}"
                                self.assertEqual(encoded.global_features[names[prefix + "_present:flag"]], 1)
                                self.assertAlmostEqual(encoded.global_features[names[prefix + "_amount:scalar/12"]], offer["Amount"] / 12)
                    if any(state.get("pendingTownFormations", {}).values()):
                        observed.add("pendingTownFormations")
                        for role, formations in state["pendingTownFormations"].items():
                            self.assertEqual(encoded.global_features[names[f"pending_town_{role}_count:scalar/4"]], len(formations or []) / 4)
            self.assertTrue(set(range(1, 7)).issubset(rounds), rounds)
        self.assertEqual(factions, set(range(1, 15)))
        print("Observed legal pending boundaries:", sorted(observed))
        self.assertTrue({"pendingLeechOffers", "pendingFavorTileSelection",
                         "pendingHalflingsSpades", "pendingChaosMagiciansDoubleTurn",
                         "pendingTownFormations"}.issubset(observed), observed)


if __name__ == "__main__":
    LegalTraceTest.fixture_binary = sys.argv.pop(1)
    unittest.main()
