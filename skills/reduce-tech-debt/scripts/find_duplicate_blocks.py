#!/usr/bin/env python3
"""Find repeated normalized code blocks across source files.

This is a heuristic detector for clone candidates. It reports repeated windows of
N normalized lines so results should be reviewed before refactoring.
"""

from __future__ import annotations

import argparse
import hashlib
import os
import re
from collections import defaultdict
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable

DEFAULT_EXTENSIONS = {
    ".py",
    ".js",
    ".jsx",
    ".ts",
    ".tsx",
    ".go",
    ".java",
    ".rb",
    ".php",
    ".rs",
    ".cs",
}

DEFAULT_EXCLUDES = {
    ".git",
    "node_modules",
    "dist",
    "build",
    "coverage",
    "vendor",
    "__pycache__",
}


@dataclass(frozen=True)
class WindowHit:
    path: str
    start: int


def iter_files(root: Path, extensions: set[str], excludes: set[str]) -> Iterable[Path]:
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if d not in excludes]
        for filename in filenames:
            path = Path(dirpath) / filename
            if path.suffix.lower() in extensions:
                yield path


def normalize_line(line: str) -> str:
    # Remove inline comments for common comment prefixes and normalize spacing.
    line = re.sub(r"//.*$", "", line)
    line = re.sub(r"#.*$", "", line)
    line = re.sub(r"\s+", "", line)
    return line.strip()


def compute_windows(path: Path, window_size: int, min_chars: int) -> list[tuple[str, int]]:
    try:
        lines = path.read_text(encoding="utf-8", errors="ignore").splitlines()
    except OSError:
        return []

    normalized = [normalize_line(line) for line in lines]
    windows: list[tuple[str, int]] = []

    for idx in range(0, len(normalized) - window_size + 1):
        block = normalized[idx : idx + window_size]
        if any(not line for line in block):
            continue
        joined = "\n".join(block)
        if len(joined) < min_chars:
            continue
        digest = hashlib.sha1(joined.encode("utf-8")).hexdigest()
        windows.append((digest, idx + 1))

    return windows


def parse_csv_set(raw: str) -> set[str]:
    return {item.strip() for item in raw.split(",") if item.strip()}


def main() -> int:
    parser = argparse.ArgumentParser(description="Find duplicate code blocks")
    parser.add_argument("--root", default=".", help="Project root to scan")
    parser.add_argument("--window", type=int, default=8, help="Normalized lines per block")
    parser.add_argument("--min-chars", type=int, default=120, help="Minimum normalized chars per block")
    parser.add_argument(
        "--extensions",
        default=",".join(sorted(DEFAULT_EXTENSIONS)),
        help="Comma-separated file extensions to scan",
    )
    parser.add_argument(
        "--exclude-dirs",
        default=",".join(sorted(DEFAULT_EXCLUDES)),
        help="Comma-separated directory names to exclude",
    )
    parser.add_argument("--top", type=int, default=50, help="Max duplicate groups to print")
    args = parser.parse_args()

    root = Path(args.root).resolve()
    extensions = parse_csv_set(args.extensions)
    excludes = parse_csv_set(args.exclude_dirs)

    groups: dict[str, list[WindowHit]] = defaultdict(list)

    for path in iter_files(root, extensions, excludes):
        for digest, start_line in compute_windows(path, args.window, args.min_chars):
            rel = str(path.relative_to(root))
            groups[digest].append(WindowHit(path=rel, start=start_line))

    duplicates = [hits for hits in groups.values() if len(hits) > 1]
    duplicates.sort(key=len, reverse=True)

    print(f"Scanned root: {root}")
    print(f"Duplicate groups: {len(duplicates)}")

    if not duplicates:
        return 0

    shown = 0
    for hits in duplicates:
        if shown >= args.top:
            break
        unique_locations = {(h.path, h.start) for h in hits}
        if len(unique_locations) < 2:
            continue
        print(f"\nGroup size: {len(unique_locations)}")
        for path, start in sorted(unique_locations):
            print(f"- {path}:{start}")
        shown += 1

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
