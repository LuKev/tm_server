#!/usr/bin/env python3
"""Fetch Snellman S67-S69 (G1-G7) logs and expected final scores for replay fixtures."""

from __future__ import annotations

import argparse
import json
import pathlib
import time
import urllib.parse
import urllib.request
from typing import Any

SNELLMAN_VIEW_GAME_URL = "https://terra.snellman.net/app/view-game/"
SEASONS = (67, 68, 69)
GAMES = range(1, 8)


def fetch_game_state(game_id: str) -> dict[str, Any]:
    data = {
        "game": game_id,
        "preview": "",
        "preview-faction": "",
        "csrf-token": "invalid",
        "cache-token": str(time.time()),
    }
    encoded = urllib.parse.urlencode(data).encode("utf-8")
    req = urllib.request.Request(
        SNELLMAN_VIEW_GAME_URL,
        data=encoded,
        method="POST",
        headers={"Content-Type": "application/x-www-form-urlencoded"},
    )
    with urllib.request.urlopen(req, timeout=60) as response:
        return json.loads(response.read().decode("utf-8"))


def ledger_to_tab_text(ledger: list[dict[str, Any]]) -> str:
    lines: list[str] = []
    for record in ledger:
        if "comment" in record:
            lines.append(str(record["comment"]))
            continue

        row: list[str] = [str(record.get("faction", ""))]

        for key, unit in (("VP", "VP"), ("C", "C"), ("W", "W"), ("P", "P"), ("PW", "PW"), ("CULT", "")):
            field = record.get(key, {})
            delta = field.get("delta", "")
            if isinstance(delta, int):
                delta_text = f"+{delta}" if delta > 0 else ("" if delta == 0 else str(delta))
            else:
                delta_text = str(delta or "")
            value = str(field.get("value", ""))
            value_with_unit = f"{value} {unit}".strip()
            row.append(delta_text)
            row.append(value_with_unit)

        # Keep leech as compact values for parity with Snellman row shape.
        leech = record.get("leech", {}) or {}
        leech_text = " ".join(str(v) for _, v in sorted(leech.items()))
        row.append(leech_text)
        row.append(str(record.get("commands", "")))
        lines.append("\t".join(row))

    return "\n".join(lines) + "\n"


def main() -> None:
    parser = argparse.ArgumentParser(description="Fetch Snellman batch fixtures")
    parser.add_argument(
        "--output-dir",
        default="server/internal/replay/testdata/snellman_batch",
        help="Directory where fixture files and manifest are written",
    )
    args = parser.parse_args()

    output_dir = pathlib.Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    games_manifest: list[dict[str, Any]] = []

    for season in SEASONS:
        for game_num in GAMES:
            game_id = f"4pLeague_S{season}_D1L1_G{game_num}"
            state = fetch_game_state(game_id)
            ledger = state.get("ledger", [])
            text = ledger_to_tab_text(ledger)

            expected_scores = {
                faction.lower(): int(payload["VP"])
                for faction, payload in sorted(state.get("factions", {}).items())
            }

            file_name = f"{game_id}.txt"
            (output_dir / file_name).write_text(text, encoding="utf-8")

            games_manifest.append(
                {
                    "game_id": game_id,
                    "log_file": file_name,
                    "expected_total_vp": expected_scores,
                }
            )

    manifest = {
        "source": "https://terra.snellman.net/app/view-game/",
        "seasons": list(SEASONS),
        "games": games_manifest,
    }
    (output_dir / "manifest.json").write_text(
        json.dumps(manifest, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )

    print(f"Wrote {len(games_manifest)} games to {output_dir}")


if __name__ == "__main__":
    main()
