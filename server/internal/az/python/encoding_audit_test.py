"""Paired semantic records for the R3 information-preservation audit."""

import copy
import unittest

import numpy as np

from schema import (
    ACTION_SCHEMA_VERSION, RULES_VERSION, STATE_SCHEMA_VERSION,
    MANIFEST, encode_actions, encode_global, encode_spatial,
)


def record(state):
    return {"rules_version": RULES_VERSION, "state_version": STATE_SCHEMA_VERSION,
            "action_version": ACTION_SCHEMA_VERSION, "state": state,
            "faction_state": [{}, {}]}


def globals_for(state):
    return encode_global(record(state))[0]


class EncodingAuditTest(unittest.TestCase):
    def test_pending_owners_are_not_presence_bits(self):
        # During an opponent reaction, the owner of a deferred main-action
        # continuation need not be the canonical decision player (self).
        for field, key in (
            ("pendingFavorTileSelection", "PlayerID"),
            ("pendingHalflingsSpades", "PlayerID"),
            ("pendingDarklingsPriestOrdination", "PlayerID"),
            ("pendingCultistsCultSelection", "PlayerID"),
            ("pendingTownCultTopChoice", "PlayerID"),
            ("pendingChaosMagiciansDoubleTurn", "playerId"),
        ):
            with self.subTest(field=field):
                self.assertFalse(np.array_equal(globals_for({field: {key: "self"}}),
                                                globals_for({field: {key: "opponent"}})))
        self.assertFalse(np.array_equal(globals_for({"pendingFreeActionsPlayerId": "self"}),
                                        globals_for({"pendingFreeActionsPlayerId": "opponent"})))

    def test_order_source_and_event_links_survive_equal_aggregates(self):
        first = {"Amount": 1, "VPCost": 0, "FromPlayerID": "opponent", "eventId": 41,
                 "sourceHex": {"Q": 0, "R": 1}}
        second = {"Amount": 3, "VPCost": 2, "FromPlayerID": "opponent", "eventId": 42,
                  "sourceHex": {"Q": 1, "R": 1}}
        state = {"pendingLeechOffers": {"self": [first, second]},
                 "pendingCultistsLeech": {"41": {"PlayerID": "opponent", "OffersCreated": 1}}}
        baseline = globals_for(state)
        reordered = copy.deepcopy(state)
        reordered["pendingLeechOffers"]["self"].reverse()
        self.assertFalse(np.array_equal(baseline, globals_for(reordered)))
        linked = copy.deepcopy(state)
        linked["pendingCultistsLeech"] = {"42": state["pendingCultistsLeech"]["41"]}
        self.assertFalse(np.array_equal(baseline, globals_for(linked)))
        moved = copy.deepcopy(state)
        moved["pendingLeechOffers"]["self"][0]["sourceHex"]["Q"] = 2
        self.assertFalse(np.array_equal(baseline, globals_for(moved)))
        renamed = copy.deepcopy(state)
        renamed["pendingLeechOffers"]["self"][0]["eventId"] = 99
        renamed["pendingLeechOffers"]["self"][1]["eventId"] = 98
        renamed["pendingCultistsLeech"] = {"99": state["pendingCultistsLeech"]["41"]}
        np.testing.assert_array_equal(baseline, globals_for(renamed))

    def test_town_partition_survives_same_union_and_count(self):
        hexes = [{"Q": i, "R": 0} for i in range(8)]
        def state(groups):
            return {"pendingTownFormations": {"self": [
                {"Hexes": group, "CanBeDelayed": False} for group in groups]}}
        first = state([hexes[:4], hexes[4:]])
        second = state([hexes[:3] + hexes[4:5], hexes[3:4] + hexes[5:]])
        # Old union planes plus aggregate count/hex count were identical.
        self.assertFalse(np.array_equal(encode_spatial(first), encode_spatial(second)))

    def test_overflow_is_rejected_not_truncated(self):
        with self.assertRaisesRegex(ValueError, "leech queue"):
            globals_for({"pendingLeechOffers": {"self": [{"Amount": 1}] * 3}})
        with self.assertRaisesRegex(ValueError, "pending towns"):
            encode_spatial({"pendingTownFormations": {"self": [{"Hexes": []}] * 6}})

    def test_compound_chaos_not_silently_count_encoded(self):
        with self.assertRaisesRegex(ValueError, "compound Chaos"):
            encode_actions([{"kind": 7, "special": 4, "subactions": [{"kind": 2}, {"kind": 3}]}])
        encode_actions([{"kind": 7, "special": 4}])

    def test_schema_fingerprints_visible(self):
        # Kept in test output for independent golden update review.
        print("representation audit schema:", MANIFEST.as_dict())


if __name__ == "__main__":
    unittest.main()
