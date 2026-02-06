#!/usr/bin/env python3
"""Find candidate unused exports in JS/TS code.

Heuristic scanner:
- Collects named exports declared in .js/.jsx/.ts/.tsx files.
- Marks an export as used when referenced in any other source file.

Results are candidates only and should be validated before deletion.
"""

from __future__ import annotations

import argparse
import os
import re
from collections import defaultdict
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable

DEFAULT_EXTENSIONS = {".js", ".jsx", ".ts", ".tsx"}
DEFAULT_EXCLUDES = {".git", "node_modules", "dist", "build", "coverage"}

EXPORT_PATTERNS = [
    re.compile(r"\bexport\s+(?:const|let|var|function|class|type|interface|enum)\s+([A-Za-z_][A-Za-z0-9_]*)"),
    re.compile(r"\bexport\s*\{\s*([^}]+)\s*\}"),
]

TOKEN_PATTERN = re.compile(r"\b([A-Za-z_][A-Za-z0-9_]*)\b")


@dataclass(frozen=True)
class ExportDecl:
    symbol: str
    path: str
    line: int


def iter_files(root: Path, extensions: set[str], excludes: set[str]) -> Iterable[Path]:
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if d not in excludes]
        for filename in filenames:
            path = Path(dirpath) / filename
            if path.suffix.lower() in extensions:
                yield path


def parse_csv_set(raw: str) -> set[str]:
    return {item.strip() for item in raw.split(",") if item.strip()}


def parse_named_exports(spec: str) -> list[str]:
    names: list[str] = []
    for part in spec.split(","):
        cleaned = part.strip()
        if not cleaned:
            continue
        # Handle aliases: foo as bar
        alias_parts = re.split(r"\s+as\s+", cleaned)
        names.append(alias_parts[-1].strip())
    return [name for name in names if re.match(r"^[A-Za-z_][A-Za-z0-9_]*$", name)]


def collect_exports(root: Path, files: list[Path]) -> list[ExportDecl]:
    exports: list[ExportDecl] = []
    for path in files:
        rel = str(path.relative_to(root))
        try:
            lines = path.read_text(encoding="utf-8", errors="ignore").splitlines()
        except OSError:
            continue

        for i, line in enumerate(lines, start=1):
            for pattern in EXPORT_PATTERNS:
                match = pattern.search(line)
                if not match:
                    continue
                if match.lastindex == 1 and "{" not in line:
                    symbol = match.group(1)
                    if symbol != "default":
                        exports.append(ExportDecl(symbol=symbol, path=rel, line=i))
                else:
                    for symbol in parse_named_exports(match.group(1)):
                        exports.append(ExportDecl(symbol=symbol, path=rel, line=i))
    return exports


def collect_token_usage(root: Path, files: list[Path]) -> dict[str, set[str]]:
    usage: dict[str, set[str]] = defaultdict(set)
    for path in files:
        rel = str(path.relative_to(root))
        try:
            content = path.read_text(encoding="utf-8", errors="ignore")
        except OSError:
            continue
        for token in TOKEN_PATTERN.findall(content):
            usage[token].add(rel)
    return usage


def main() -> int:
    parser = argparse.ArgumentParser(description="Find candidate unused TS/JS exports")
    parser.add_argument("--root", default=".", help="Project root")
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
    args = parser.parse_args()

    root = Path(args.root).resolve()
    extensions = parse_csv_set(args.extensions)
    excludes = parse_csv_set(args.exclude_dirs)

    files = list(iter_files(root, extensions, excludes))
    exports = collect_exports(root, files)
    usage = collect_token_usage(root, files)

    candidates: list[ExportDecl] = []
    for decl in exports:
        token_users = usage.get(decl.symbol, set())
        # Ignore self-reference in declaring file.
        external_users = token_users - {decl.path}
        if not external_users:
            candidates.append(decl)

    print(f"Scanned root: {root}")
    print(f"Export declarations: {len(exports)}")
    print(f"Candidate unused exports: {len(candidates)}")

    for decl in sorted(candidates, key=lambda d: (d.path, d.line, d.symbol)):
        print(f"- {decl.path}:{decl.line} export {decl.symbol}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
