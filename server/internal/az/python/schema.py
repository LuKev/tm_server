"""Versioned Terra Mystica v0 tensor materialization.

The Go engine persists canonical, player-relative JSON.  This module turns that
raw record and its complete legal SearchAction list into the fixed tensors used
by the first network.  It deliberately has no learned action vocabulary.
"""

from __future__ import annotations

from dataclasses import dataclass
import hashlib
import math
from typing import Any, Iterable

import numpy as np

FLOAT32_MAX = float(np.finfo(np.float32).max)

RULES_VERSION = 1
STATE_SCHEMA_VERSION = 1
ACTION_SCHEMA_VERSION = 1

GRID_HEIGHT = 9
GRID_WIDTH = 17
GRID_Q_OFFSET = 4
AXIAL_NEIGHBORS = ((1, 0), (1, -1), (0, -1), (-1, 0), (-1, 1), (0, 1))
BRIDGE_DIRECTIONS = ((1, -2), (2, -1), (1, 1), (-1, 2), (-2, 1), (-1, -1))

BASE_ACTION_KINDS = (0, 1, 2, 3, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 20, 21, 26, 30, 31, 32, 33, 34, 35, 36)
BASE_SPECIALS = (0, 1, 3, 4, 5, 6, 7, 8, 9, 10)
BASE_FACTIONS = tuple(range(1, 15))
BASE_TERRAINS = tuple(range(8))
BASE_BUILDINGS = tuple(range(5))
BASE_CARDS = tuple(range(9))
BASE_FAVORS = tuple(range(12))
BASE_TOWNS = (0, 1, 2, 4, 5)
POWER_ACTIONS = tuple(range(6))
CULT_TRACKS = tuple(range(4))
SCORING_TILES = tuple(range(9))
CONVERSIONS = (
    "power_to_coin",
    "power_to_worker",
    "power_to_priest",
    "priest_to_worker",
    "worker_to_coin",
    "alchemists_vp_to_coin",
    "alchemists_coin_to_vp",
)

SPATIAL_FEATURE_NAMES = (
    "valid",
    *(f"terrain_{value}" for value in BASE_TERRAINS),
    *(f"building_{value}" for value in BASE_BUILDINGS),
    "building_self",
    "building_opponent",
    "town_member",
    "town_marker_self",
    "town_marker_opponent",
    *(f"bridge_self_direction_{index}" for index in range(len(BRIDGE_DIRECTIONS))),
    *(f"bridge_opponent_direction_{index}" for index in range(len(BRIDGE_DIRECTIONS))),
    "pending_town_immediate_self",
    "pending_town_delayed_self",
    "pending_town_immediate_opponent",
    "pending_town_delayed_opponent",
    "pending_halflings_transformed",
    "skip_used_self",
    "skip_used_opponent",
)


@dataclass(frozen=True)
class EncodedPosition:
    spatial: np.ndarray
    global_features: np.ndarray
    action_features: np.ndarray
    action_hex_indices: np.ndarray


@dataclass(frozen=True)
class SchemaManifest:
    rules_version: int = RULES_VERSION
    state_schema_version: int = STATE_SCHEMA_VERSION
    action_schema_version: int = ACTION_SCHEMA_VERSION
    spatial_features: int = len(SPATIAL_FEATURE_NAMES)
    global_features: int = 0
    action_features: int = 0
    grid_height: int = GRID_HEIGHT
    grid_width: int = GRID_WIDTH
    grid_q_offset: int = GRID_Q_OFFSET
    coordinate_convention: str = "axial_qr:row=r,column=q+grid_q_offset"
    axial_neighbors: tuple[tuple[int, int], ...] = AXIAL_NEIGHBORS
    bridge_directions: tuple[tuple[int, int], ...] = BRIDGE_DIRECTIONS
    spatial_schema_sha256: str = ""
    global_schema_sha256: str = ""
    action_schema_sha256: str = ""

    def as_dict(self) -> dict[str, int]:
        return dict(self.__dict__)


class _Features:
    def __init__(self, collect: bool = True) -> None:
        self.collect = collect
        self.names: list[str] = []
        self.values: list[float] = []

    def scalar(self, name: str, value: Any, scale: float = 1.0) -> None:
        try:
            scalar_value = float(value or 0) / scale
        except (OverflowError, TypeError, ValueError) as error:
            raise ValueError(f"invalid scalar feature {name}: {value}") from error
        if not math.isfinite(scalar_value) or abs(scalar_value) > FLOAT32_MAX:
            raise ValueError(f"non-finite scalar feature {name}: {value}")
        if not self.collect:
            return
        self.names.append(f"{name}:scalar/{scale:g}")
        self.values.append(scalar_value)

    def flag(self, name: str, value: Any) -> None:
        if not self.collect:
            return
        self.names.append(f"{name}:flag")
        self.values.append(1.0 if value else 0.0)

    def one_hot(self, prefix: str, value: Any, domain: Iterable[Any]) -> None:
        if not self.collect:
            return
        for candidate in domain:
            self.names.append(f"{prefix}:one_hot={candidate}")
            self.values.append(1.0 if value == candidate else 0.0)


def _value(value: Any, name: str, default: Any = None) -> Any:
    if not isinstance(value, dict):
        return default
    return value.get(name, default)


def _role(player_id: Any) -> int:
    if player_id == "self":
        return 0
    if player_id == "opponent":
        return 1
    return -1


def _coord(value: Any) -> tuple[int, int] | None:
    if not isinstance(value, dict):
        return None
    q = value.get("Q")
    r = value.get("R")
    if not isinstance(q, int) or isinstance(q, bool) or not isinstance(r, int) or isinstance(r, bool):
        return None
    return q, r


def _grid_index(q: int, r: int) -> int:
    column = q + GRID_Q_OFFSET
    if not (0 <= r < GRID_HEIGHT and 0 <= column < GRID_WIDTH):
        return -1
    return r * GRID_WIDTH + column


def _put_plane(spatial: np.ndarray | None, plane: int, coordinate: tuple[int, int] | None, value: float = 1.0) -> None:
    if spatial is None or coordinate is None:
        return
    index = _grid_index(*coordinate)
    if index >= 0:
        spatial[plane, index // GRID_WIDTH, index % GRID_WIDTH] = value


def encode_spatial(state: dict[str, Any], *, validate_only: bool = False) -> np.ndarray | None:
    spatial = None if validate_only else np.zeros((len(SPATIAL_FEATURE_NAMES), GRID_HEIGHT, GRID_WIDTH), dtype=np.float32)
    plane = {name: index for index, name in enumerate(SPATIAL_FEATURE_NAMES)}
    game_map = state.get("map") or {}
    hexes = game_map.get("hexes") or {}
    for key, map_hex in hexes.items():
        q_text, r_text = key.split(",", 1)
        coordinate = (int(q_text), int(r_text))
        _put_plane(spatial, plane["valid"], coordinate)
        terrain = int(map_hex.get("Terrain", -1))
        if terrain in BASE_TERRAINS:
            _put_plane(spatial, plane[f"terrain_{terrain}"], coordinate)
        building = map_hex.get("Building")
        if building:
            building_type = int(building.get("type", -1))
            if building_type in BASE_BUILDINGS:
                _put_plane(spatial, plane[f"building_{building_type}"], coordinate)
            owner = _role(building.get("playerId"))
            if owner >= 0:
                _put_plane(spatial, plane[("building_self", "building_opponent")[owner]], coordinate)
        if map_hex.get("PartOfTown", False):
            _put_plane(spatial, plane["town_member"], coordinate)
        if map_hex.get("HasTownTile", False):
            owner = _role(map_hex.get("TownTileOwnerPlayerID"))
            if owner >= 0:
                _put_plane(spatial, plane[("town_marker_self", "town_marker_opponent")[owner]], coordinate)

    bridge_direction_index = {direction: index for index, direction in enumerate(BRIDGE_DIRECTIONS)}
    for key, owner_id in (game_map.get("bridges") or {}).items():
        owner = _role(owner_id)
        if owner < 0:
            continue
        endpoints = []
        for endpoint in key.split("|", 1):
            q_text, r_text = endpoint.split(",", 1)
            endpoints.append((int(q_text), int(r_text)))
        if len(endpoints) != 2:
            continue
        first, second = endpoints
        forward = (second[0] - first[0], second[1] - first[1])
        reverse = (-forward[0], -forward[1])
        if forward not in bridge_direction_index or reverse not in bridge_direction_index:
            raise ValueError(f"invalid bridge direction {forward} in {key}")
        role_name = ("self", "opponent")[owner]
        _put_plane(spatial, plane[f"bridge_{role_name}_direction_{bridge_direction_index[forward]}"], first)
        _put_plane(spatial, plane[f"bridge_{role_name}_direction_{bridge_direction_index[reverse]}"], second)

    for role_name, formations in (state.get("pendingTownFormations") or {}).items():
        owner = _role(role_name)
        if owner < 0:
            continue
        for formation in formations or []:
            timing = "delayed" if formation.get("CanBeDelayed") else "immediate"
            plane_name = f"pending_town_{timing}_{role_name}"
            for hex_value in formation.get("Hexes", []) or []:
                _put_plane(spatial, plane[plane_name], _coord(hex_value))
            _put_plane(
                spatial,
                plane[plane_name],
                _coord(formation.get("SkippedRiverHex")),
            )

    halflings = state.get("pendingHalflingsSpades")
    for hex_value in _value(halflings, "TransformedHexes", []) or []:
        _put_plane(spatial, plane["pending_halflings_transformed"], _coord(hex_value))
    for role_name, used_hexes in (state.get("skipAbilityUsedThisAction") or {}).items():
        owner = _role(role_name)
        if owner < 0:
            continue
        for hex_value in used_hexes or []:
            _put_plane(spatial, plane[("skip_used_self", "skip_used_opponent")[owner]], _coord(hex_value))
    return spatial


def _sequence_roles(features: _Features, prefix: str, sequence: list[Any], length: int) -> None:
    for index in range(length):
        value = _role(sequence[index]) if index < len(sequence) else -1
        features.one_hot(f"{prefix}_{index}", value, (-1, 0, 1))


def _player_features(
    features: _Features,
    state: dict[str, Any],
    role_name: str,
    faction_state: dict[str, Any],
    structure_counts: dict[str, list[int]],
) -> None:
    prefix = role_name
    player = (state.get("players") or {}).get(role_name) or {}
    faction_type = int(faction_state.get("type") or 0)
    features.one_hot(f"{prefix}_faction", faction_type, BASE_FACTIONS)
    resources = player.get("resources") or {}
    power = resources.get("power") or {}
    features.scalar(f"{prefix}_vp", player.get("victoryPoints"), 100)
    features.scalar(f"{prefix}_coins", resources.get("coins"), 30)
    features.scalar(f"{prefix}_workers", resources.get("workers"), 20)
    features.scalar(f"{prefix}_priests", resources.get("priests"), 7)
    features.scalar(f"{prefix}_power_1", power.get("powerI"), 20)
    features.scalar(f"{prefix}_power_2", power.get("powerII"), 20)
    features.scalar(f"{prefix}_power_3", power.get("powerIII"), 20)
    features.scalar(f"{prefix}_shipping", player.get("shippingLevel"), 5)
    features.scalar(f"{prefix}_digging", player.get("diggingLevel"), 2)
    features.scalar(f"{prefix}_bridges_built", player.get("bridgesBuilt"), 3)
    features.scalar(f"{prefix}_keys", player.get("keys"), 4)
    features.scalar(f"{prefix}_towns", player.get("townsFormed"), 8)
    features.flag(f"{prefix}_passed", player.get("hasPassed"))
    features.flag(f"{prefix}_stronghold_ability", player.get("hasStrongholdAbility"))
    for building, maximum in enumerate((8, 4, 3, 1, 1)):
        count = structure_counts[role_name][building]
        features.scalar(f"{prefix}_building_supply_{building}", maximum - count, maximum)
    cults = player.get("cults") or {}
    authoritative_cults = (((state.get("cultTracks") or {}).get("playerPositions") or {}).get(role_name) or {})
    for track in CULT_TRACKS:
        features.scalar(f"{prefix}_cult_{track}", authoritative_cults.get(str(track)), 10)
        features.scalar(f"{prefix}_legacy_player_cult_{track}", cults.get(str(track)), 10)
    player_card_map = (state.get("bonusCards") or {}).get("playerCards") or {}
    card = player_card_map.get(role_name, -1)
    features.one_hot(f"{prefix}_bonus_card", card, BASE_CARDS)
    features.flag(f"{prefix}_bonus_card_selected", role_name in player_card_map)
    features.flag(
        f"{prefix}_bonus_player_has_card",
        ((state.get("bonusCards") or {}).get("playerHasCard") or {}).get(role_name),
    )
    favor_tiles = ((state.get("favorTiles") or {}).get("playerTiles") or {}).get(role_name, []) or []
    for tile in BASE_FAVORS:
        features.scalar(f"{prefix}_favor_{tile}", favor_tiles.count(tile), 3)
    town_tiles = player.get("townTiles") or []
    for tile in BASE_TOWNS:
        features.scalar(f"{prefix}_town_tile_{tile}", town_tiles.count(tile), 2)
    used_specials = player.get("specialActionsUsed") or {}
    for special in BASE_SPECIALS:
        features.flag(f"{prefix}_special_used_{special}", used_specials.get(str(special), False))
    features.scalar(f"{prefix}_setup_dwellings", (state.get("setupPlacedDwellings") or {}).get(role_name), 3)
    features.scalar(f"{prefix}_priests_sent", ((state.get("scoringTiles") or {}).get("priestsSent") or {}).get(role_name), 7)
    for key, scale in (("terraform_cost_one", 3), ("flight_range", 4), ("shipping_level", 5)):
        features.scalar(f"{prefix}_faction_{key}", faction_state.get(key), scale)
    features.flag(f"{prefix}_faction_has_stronghold", faction_state.get("has_stronghold"))

    cult_state = state.get("cultTracks") or {}
    claimed = ((cult_state.get("bonusPositionsClaimed") or {}).get(role_name) or {})
    for track in CULT_TRACKS:
        by_position = claimed.get(str(track)) or {}
        for position in (3, 5, 7, 10):
            features.flag(f"{prefix}_cult_claimed_{track}_{position}", by_position.get(str(position), False))
        priests = ((cult_state.get("priestsOnActionSpaces") or {}).get(role_name) or {}).get(str(track), 0)
        features.scalar(f"{prefix}_cult_priests_{track}", priests, 7)


def _pending_features(features: _Features, state: dict[str, Any]) -> None:
    offers = state.get("pendingLeechOffers") or {}
    pending_cultists = state.get("pendingCultistsLeech") or {}
    for role_name in ("self", "opponent"):
        role_offers = offers.get(role_name) or []
        features.scalar(f"pending_leech_{role_name}_count", len(role_offers), 4)
        features.scalar(f"pending_leech_{role_name}_power", sum(item.get("Amount", 0) or 0 for item in role_offers), 12)
        features.scalar(f"pending_leech_{role_name}_vp", sum(item.get("VPCost", 0) or 0 for item in role_offers), 10)
        for source in (0, 1):
            features.scalar(
                f"pending_leech_{role_name}_source_{source}",
                sum(_role(item.get("FromPlayerID")) == source for item in role_offers),
                4,
            )
        features.scalar(
            f"pending_leech_{role_name}_cultists_linked",
            sum(str(item.get("eventId", -1)) in pending_cultists for item in role_offers),
            4,
        )

    formations_by_player = state.get("pendingTownFormations") or {}
    for role_name in ("self", "opponent"):
        formations = formations_by_player.get(role_name) or []
        features.scalar(f"pending_town_{role_name}_count", len(formations), 4)
        features.scalar(f"pending_town_{role_name}_hexes", sum(len(item.get("Hexes", []) or []) for item in formations), 20)
        features.scalar(f"pending_town_{role_name}_delayed", sum(bool(item.get("CanBeDelayed")) for item in formations), 4)
        features.scalar(f"pending_spades_{role_name}", (state.get("pendingSpades") or {}).get(role_name), 3)
        features.flag(f"pending_spade_build_{role_name}", (state.get("pendingSpadeBuildAllowed") or {}).get(role_name))
        features.scalar(f"pending_cult_spades_{role_name}", (state.get("pendingCultRewardSpades") or {}).get(role_name), 3)
        features.scalar(f"skip_used_{role_name}_count", len((state.get("skipAbilityUsedThisAction") or {}).get(role_name) or []), 8)

    favor = state.get("pendingFavorTileSelection")
    features.flag("pending_favor", bool(favor))
    features.scalar("pending_favor_count", _value(favor, "Count"), 2)
    selected = _value(favor, "SelectedTiles", []) or []
    for tile in BASE_FAVORS:
        features.flag(f"pending_favor_selected_{tile}", tile in selected)

    halflings = state.get("pendingHalflingsSpades")
    features.flag("pending_halflings", bool(halflings))
    features.scalar("pending_halflings_spades", _value(halflings, "SpadesRemaining"), 3)
    features.scalar("pending_halflings_hexes", len(_value(halflings, "TransformedHexes", []) or []), 3)
    features.flag("pending_darklings", bool(state.get("pendingDarklingsPriestOrdination")))
    features.flag("pending_cultists", bool(state.get("pendingCultistsCultSelection")))

    cult_top = state.get("pendingTownCultTopChoice")
    features.flag("pending_town_cult_top", bool(cult_top))
    features.scalar("pending_town_cult_advance", _value(cult_top, "AdvanceAmount"), 2)
    features.scalar("pending_town_cult_max", _value(cult_top, "MaxSelections"), 4)
    candidates = _value(cult_top, "CandidateTracks", []) or []
    for track in CULT_TRACKS:
        features.flag(f"pending_town_cult_track_{track}", track in candidates)

    chaos = state.get("pendingChaosMagiciansDoubleTurn")
    features.flag("pending_chaos", bool(chaos))
    features.scalar("pending_chaos_actions", _value(chaos, "actionsRemaining"), 2)
    features.flag("pending_free_actions", bool(state.get("pendingFreeActionsPlayerId")))

    features.scalar("pending_cultists_leech_count", len(pending_cultists), 4)
    for owner in (0, 1):
        features.scalar(
            f"pending_cultists_leech_owner_{owner}",
            sum(_role(item.get("PlayerID")) == owner for item in pending_cultists.values()),
            4,
        )
    for field_name in ("OffersCreated", "ResolvedCount", "AcceptedCount", "DeclinedCount"):
        features.scalar(
            f"pending_cultists_leech_{field_name}",
            sum(item.get(field_name, 0) or 0 for item in pending_cultists.values()),
            4,
        )


def encode_global(
    record: dict[str, Any],
    structure_counts: dict[str, list[int]] | None = None,
    *,
    validate_only: bool = False,
) -> tuple[np.ndarray, tuple[str, ...]] | None:
    if int(record.get("rules_version", -1)) != RULES_VERSION:
        raise ValueError(f"rules schema mismatch: {record.get('rules_version')} != {RULES_VERSION}")
    if int(record.get("action_version", -1)) != ACTION_SCHEMA_VERSION:
        raise ValueError(f"action schema mismatch: {record.get('action_version')} != {ACTION_SCHEMA_VERSION}")
    if int(record.get("state_version", -1)) != STATE_SCHEMA_VERSION:
        raise ValueError(f"state schema mismatch: {record.get('state_version')} != {STATE_SCHEMA_VERSION}")
    state = record.get("state") or {}
    features = _Features(collect=not validate_only)
    features.scalar("rules_version", RULES_VERSION)
    features.scalar("state_schema_version", STATE_SCHEMA_VERSION)
    features.scalar("action_schema_version", ACTION_SCHEMA_VERSION)
    features.flag("map_base", (state.get("map") or {}).get("id") == "base")
    features.flag("setup_snellman", state.get("setupMode") == "snellman")
    features.flag("turn_order_pass", state.get("turnOrderPolicy") == "pass_order")
    features.one_hot("phase", int(state.get("phase", -1)), range(6))
    features.one_hot("round", int(state.get("round", -1)), range(7))
    features.one_hot("setup_subphase", state.get("setupSubphase"), ("none", "dwellings", "bonus_cards", "complete"))
    current_id = None
    turn_order = state.get("turnOrder") or []
    current_index = int(state.get("currentPlayerIndex", -1))
    if 0 <= current_index < len(turn_order):
        current_id = turn_order[current_index]
    features.one_hot("current_player", _role(current_id), (-1, 0, 1))
    _sequence_roles(features, "turn_order", turn_order, 2)
    _sequence_roles(features, "pass_order", state.get("passOrder") or [], 2)
    _sequence_roles(features, "setup_dwelling_order", state.get("setupDwellingOrder") or [], 5)
    _sequence_roles(features, "setup_bonus_order", state.get("setupBonusOrder") or [], 2)
    features.scalar("setup_dwelling_index", state.get("setupDwellingIndex"), 5)
    features.scalar("setup_bonus_index", state.get("setupBonusIndex"), 2)

    if structure_counts is None:
        structure_counts = {"self": [0] * 5, "opponent": [0] * 5}
        for map_hex in ((state.get("map") or {}).get("hexes") or {}).values():
            building = map_hex.get("Building")
            owner = _value(building, "playerId")
            building_type = int(_value(building, "type", -1))
            if owner in structure_counts and building_type in BASE_BUILDINGS:
                structure_counts[owner][building_type] += 1
    faction_state = record.get("faction_state") or [{}, {}]
    while len(faction_state) < 2:
        faction_state.append({})
    _player_features(features, state, "self", faction_state[0] or {}, structure_counts)
    _player_features(features, state, "opponent", faction_state[1] or {}, structure_counts)

    scoring_tiles = (state.get("scoringTiles") or {}).get("tiles") or []
    for index in range(6):
        tile = scoring_tiles[index].get("type", -1) if index < len(scoring_tiles) else -1
        features.one_hot(f"scoring_round_{index + 1}", tile, SCORING_TILES)
    used_power = (state.get("powerActions") or {}).get("UsedActions") or {}
    for action in POWER_ACTIONS:
        features.flag(f"power_action_used_{action}", used_power.get(str(action), False))
    bonus_cards = state.get("bonusCards") or {}
    available_cards = bonus_cards.get("available") or {}
    for card in BASE_CARDS:
        features.flag(f"bonus_available_{card}", str(card) in available_cards)
        features.scalar(f"bonus_coins_{card}", available_cards.get(str(card)), 12)
    available_favors = (state.get("favorTiles") or {}).get("available") or {}
    for tile in BASE_FAVORS:
        features.scalar(f"favor_available_{tile}", available_favors.get(str(tile)), 3)
    available_towns = (state.get("townTiles") or {}).get("available") or {}
    for tile in BASE_TOWNS:
        features.scalar(f"town_available_{tile}", available_towns.get(str(tile)), 2)

    cult_state = state.get("cultTracks") or {}
    occupied = cult_state.get("position10Occupied") or {}
    priests_on_track = cult_state.get("priestsOnTrack") or {}
    for track in CULT_TRACKS:
        features.one_hot(f"cult_top_owner_{track}", _role(occupied.get(str(track))), (-1, 0, 1))
        by_spot = priests_on_track.get(str(track)) or {}
        for spot in (1, 2, 3):
            occupants = by_spot.get(str(spot)) or []
            for owner in (0, 1):
                features.scalar(f"cult_track_{track}_spot_{spot}_owner_{owner}", sum(_role(item) == owner for item in occupants), 3)
    _pending_features(features, state)
    if validate_only:
        return None
    return np.asarray(features.values, dtype=np.float32), tuple(features.names)


def _required_domain(action: dict[str, Any], field: str, domain: tuple[Any, ...], default: Any = 0) -> Any:
    value = action.get(field, default)
    if domain and all(isinstance(candidate, int) for candidate in domain):
        if not isinstance(value, int) or isinstance(value, bool):
            raise ValueError(f"action kind {action.get('kind')} has invalid {field} {value}")
    if value not in domain:
        raise ValueError(f"action kind {action.get('kind')} has invalid {field} {value}")
    return value


def _validate_action_hexes(action: dict[str, Any], count: int) -> None:
    hexes = action.get("hexes") or []
    if len(hexes) != count:
        raise ValueError(f"action kind {action.get('kind')} requires {count} hexes")
    for value in hexes:
        coordinate = _coord(value)
        if coordinate is None or _grid_index(*coordinate) < 0:
            raise ValueError(f"action kind {action.get('kind')} has invalid hex coordinate {value}")


def _validate_action(action: dict[str, Any]) -> None:
    if not isinstance(action, dict):
        raise ValueError("legal action is not an object")
    kind = action.get("kind", -1)
    if not isinstance(kind, int) or isinstance(kind, bool) or kind not in BASE_ACTION_KINDS:
        raise ValueError(f"unsupported v0 action kind {kind}")
    allowed = {"kind"}
    hex_count = 0
    if kind == 0:
        allowed.update(("hexes", "terrain", "build", "use_skip"))
        hex_count = 1
        _required_domain(action, "terrain", BASE_TERRAINS)
    elif kind == 1:
        allowed.update(("hexes", "building"))
        hex_count = 1
        _required_domain(action, "building", BASE_BUILDINGS)
    elif kind == 5:
        allowed.update(("track", "amount"))
        _required_domain(action, "track", CULT_TRACKS)
    elif kind == 6:
        allowed.add("power")
        power = _required_domain(action, "power", POWER_ACTIONS)
        if power == 0:
            allowed.add("hexes")
            hex_count = 2
        elif power in (4, 5):
            allowed.update(("hexes", "build", "use_skip"))
            hex_count = 1
    elif kind == 7:
        allowed.add("special")
        special = _required_domain(action, "special", BASE_SPECIALS)
        if special in (0, 7, 9):
            allowed.add("track")
            _required_domain(action, "track", CULT_TRACKS)
        elif special in (1, 3, 5, 6, 10):
            allowed.add("hexes")
            hex_count = 1
            if special in (5, 6):
                allowed.add("build")
        elif special == 8:
            allowed.update(("hexes", "terrain", "build", "use_skip"))
            hex_count = 1
            _required_domain(action, "terrain", BASE_TERRAINS)
        elif special == 4:
            allowed.add("subactions")
            subactions = action.get("subactions") or []
            if len(subactions) not in (0, 2):
                raise ValueError("Chaos Magicians special requires zero or two subactions")
            for subaction in subactions:
                _validate_action(subaction)
    elif kind == 8:
        allowed.add("card")
        _required_domain(action, "card", (-1, *BASE_CARDS), -1)
    elif kind == 9:
        allowed.add("hexes")
        hex_count = 1
    elif kind in (10, 14):
        allowed.update(("hexes", "terrain"))
        hex_count = 1
        _required_domain(action, "terrain", BASE_TERRAINS)
    elif kind == 11:
        allowed.update(("amount", "amount_2"))
    elif kind == 12:
        allowed.add("amount")
    elif kind == 13:
        allowed.add("favor_tile")
        _required_domain(action, "favor_tile", BASE_FAVORS)
    elif kind == 15:
        allowed.add("hexes")
        hex_count = 1
    elif kind == 20:
        allowed.add("amount")
    elif kind == 21:
        allowed.add("track")
        _required_domain(action, "track", CULT_TRACKS)
    elif kind == 26:
        allowed.add("faction")
        _required_domain(action, "faction", BASE_FACTIONS)
    elif kind == 30:
        allowed.add("card")
        _required_domain(action, "card", BASE_CARDS)
    elif kind == 31:
        allowed.update(("town_tile", "hexes"))
        hex_count = 1
        _required_domain(action, "town_tile", BASE_TOWNS)
    elif kind == 32:
        allowed.add("tracks")
        tracks = action.get("tracks") or []
        if any(not isinstance(track, int) or isinstance(track, bool) or track not in CULT_TRACKS for track in tracks) or tracks != sorted(set(tracks)):
            raise ValueError(f"action kind {kind} has invalid tracks {tracks}")
    elif kind == 33:
        allowed.add("amount")
    elif kind == 34:
        allowed.update(("conversion", "amount"))
        _required_domain(action, "conversion", CONVERSIONS, None)
    elif kind == 35:
        allowed.add("amount")
    elif kind == 36:
        allowed.add("hexes")
        hex_count = 2

    if hex_count:
        _validate_action_hexes(action, hex_count)
    for field in ("build", "use_skip"):
        if field in action and not isinstance(action[field], bool):
            raise ValueError(f"action kind {kind} has non-boolean {field}")
    if "amount" in allowed:
        amount = action.get("amount", 0)
        minimum = 0 if kind in (11, 12, 20) else 1
        if not isinstance(amount, int) or isinstance(amount, bool) or amount < minimum:
            raise ValueError(f"action kind {kind} has invalid amount {amount}")
    if "amount_2" in action and (not isinstance(action["amount_2"], int) or isinstance(action["amount_2"], bool) or action["amount_2"] < 0):
        raise ValueError(f"action kind {kind} has invalid amount_2 {action.get('amount_2')}")
    extras = set(action) - allowed
    if extras:
        raise ValueError(f"action kind {kind} has unsupported fields {sorted(extras)}")


def _validate_position_envelope(record: dict[str, Any], actions: list[dict[str, Any]]) -> None:
    if not isinstance(record, dict):
        raise ValueError("canonical state record is not an object")
    versions = (
        ("rules_version", RULES_VERSION, "rules schema"),
        ("state_version", STATE_SCHEMA_VERSION, "state schema"),
        ("action_version", ACTION_SCHEMA_VERSION, "action schema"),
    )
    for field, expected, label in versions:
        if type(record.get(field)) is not int or record.get(field) != expected:
            raise ValueError(f"{label} mismatch: {record.get(field)} != {expected}")
    state = record.get("state")
    faction_state = record.get("faction_state")
    if not isinstance(state, dict) or not isinstance(faction_state, list) or len(faction_state) != 2:
        raise ValueError("canonical state record has invalid state/faction_state")
    if not isinstance(actions, list) or not actions:
        raise ValueError("canonical position has no legal actions")
    for key in ((state.get("map") or {}).get("hexes") or {}):
        try:
            q_text, r_text = key.split(",", 1)
            coordinate = (int(q_text), int(r_text))
        except (AttributeError, TypeError, ValueError) as error:
            raise ValueError(f"invalid map hex coordinate {key}") from error
        if _grid_index(*coordinate) < 0:
            raise ValueError(f"map hex coordinate outside encoder grid {key}")
    for action in actions:
        _validate_action(action)


def _action_feature_vector(
    action: dict[str, Any], *, collect: bool = True
) -> tuple[np.ndarray | None, tuple[str, ...], tuple[int, int]]:
    features = _Features(collect=collect)
    kind = int(action.get("kind", -1))
    if kind not in BASE_ACTION_KINDS:
        raise ValueError(f"unsupported v0 action kind {kind}")
    features.one_hot("kind", kind, BASE_ACTION_KINDS)
    features.one_hot("special", int(action.get("special", 0)) if kind == 7 else -1, BASE_SPECIALS)
    features.one_hot("power", int(action.get("power", 0)) if kind == 6 else -1, POWER_ACTIONS)
    features.one_hot("conversion", action.get("conversion") if kind == 34 else None, CONVERSIONS)
    terrain_kinds = kind in (0, 10, 14) or (kind == 7 and int(action.get("special", 0)) == 8)
    features.one_hot("terrain", int(action.get("terrain", 0)) if terrain_kinds else -1, BASE_TERRAINS)
    features.one_hot("building", int(action.get("building", 0)) if kind == 1 else -1, BASE_BUILDINGS)
    track_kinds = kind in (5, 21) or (kind == 7 and int(action.get("special", 0)) in (0, 7, 9))
    features.one_hot("track", int(action.get("track", 0)) if track_kinds else -1, CULT_TRACKS)
    selected_tracks = action.get("tracks") or []
    for track in CULT_TRACKS:
        features.flag(f"tracks_{track}", track in selected_tracks)
    card_kinds = kind in (8, 30)
    features.one_hot("card", int(action.get("card", 0)) if card_kinds else -1, BASE_CARDS)
    features.one_hot("favor", int(action.get("favor_tile", 0)) if kind == 13 else -1, BASE_FAVORS)
    features.one_hot("town", int(action.get("town_tile", 0)) if kind == 31 else -1, BASE_TOWNS)
    features.one_hot("faction", int(action.get("faction", 0)) if kind == 26 else -1, BASE_FACTIONS)
    features.scalar("amount", action.get("amount"), 20)
    features.scalar("amount_2", action.get("amount_2"), 20)
    features.flag("build", action.get("build"))
    features.flag("use_skip", action.get("use_skip"))
    features.scalar("subaction_count", len(action.get("subactions") or []), 2)
    coordinates = [_coord(value) for value in (action.get("hexes") or [])[:2]]
    while len(coordinates) < 2:
        coordinates.append(None)
    gather: list[int] = []
    for index, coordinate in enumerate(coordinates):
        features.flag(f"hex_{index}_present", coordinate is not None)
        features.scalar(f"hex_{index}_q", coordinate[0] if coordinate else 0, 12)
        features.scalar(f"hex_{index}_r", coordinate[1] if coordinate else 0, 8)
        gather.append(_grid_index(*coordinate) if coordinate else -1)
    vector = np.asarray(features.values, dtype=np.float32) if collect else None
    return vector, tuple(features.names), (gather[0], gather[1])


def validate_position_schema(record: dict[str, Any], actions: list[dict[str, Any]]) -> None:
    _validate_position_envelope(record, actions)
    state = record["state"]
    # Traverse every feature conversion without collecting tensors. Shard
    # acceptance therefore remains deterministic while replay examples stay
    # lazy and bounded.
    encode_spatial(state, validate_only=True)
    encode_global(record, validate_only=True)
    for action in actions:
        _action_feature_vector(action, collect=False)


def encode_actions(actions: list[dict[str, Any]]) -> tuple[np.ndarray, np.ndarray, tuple[str, ...]]:
    vectors: list[np.ndarray] = []
    gathers: list[tuple[int, int]] = []
    names: tuple[str, ...] | None = None
    for action in actions:
        vector, action_names, gather = _action_feature_vector(action)
        assert vector is not None
        if names is not None and action_names != names:
            raise AssertionError("action feature schema changed between candidates")
        names = action_names
        vectors.append(vector)
        gathers.append(gather)
    if names is None:
        _, names, _ = _action_feature_vector({"kind": BASE_ACTION_KINDS[0]})
    width = len(names)
    matrix = np.stack(vectors) if vectors else np.zeros((0, width), dtype=np.float32)
    return matrix, np.asarray(gathers, dtype=np.int64).reshape((-1, 2)), names


def encode_position(record: dict[str, Any], actions: list[dict[str, Any]]) -> EncodedPosition:
    _validate_position_envelope(record, actions)
    spatial = encode_spatial(record.get("state") or {})
    global_features, _ = encode_global(record)
    action_features, action_hex_indices, _ = encode_actions(actions)
    return EncodedPosition(spatial, global_features, action_features, action_hex_indices)


_EMPTY_RECORD = {
    "rules_version": RULES_VERSION,
    "state_version": STATE_SCHEMA_VERSION,
    "action_version": ACTION_SCHEMA_VERSION,
    "state": {},
    "faction_state": [{}, {}],
}
_EMPTY_GLOBAL, GLOBAL_FEATURE_NAMES = encode_global(_EMPTY_RECORD)
_, _, ACTION_FEATURE_NAMES = encode_actions([])
def _schema_digest(names: Iterable[str]) -> str:
    return hashlib.sha256("\n".join(names).encode()).hexdigest()


MANIFEST = SchemaManifest(
    global_features=len(GLOBAL_FEATURE_NAMES),
    action_features=len(ACTION_FEATURE_NAMES),
    spatial_schema_sha256=_schema_digest(SPATIAL_FEATURE_NAMES),
    global_schema_sha256=_schema_digest(GLOBAL_FEATURE_NAMES),
    action_schema_sha256=_schema_digest(ACTION_FEATURE_NAMES),
)
