#!/usr/bin/env python3
"""Extract the ordered action vocabulary embedded in a torch checkpoint."""

import argparse
import importlib.util
import json
import os
import sys
from pathlib import Path

for entry in list(sys.path):
    path = Path(entry)
    if (path / "cmd" / "az_checkpoint_vocab").exists() and (path / "internal").exists():
        sys.path.remove(entry)
stdlib_cmd = Path(os.__file__).resolve().parent / "cmd.py"
if stdlib_cmd.exists():
    spec = importlib.util.spec_from_file_location("cmd", stdlib_cmd)
    cmd_module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(cmd_module)
    sys.modules["cmd"] = cmd_module

import torch


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--checkpoint", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()

    checkpoint = torch.load(args.checkpoint, map_location="cpu")
    vocab = checkpoint.get("action_vocab")
    if not isinstance(vocab, list) or not vocab or not all(isinstance(item, str) and item for item in vocab):
        raise SystemExit("checkpoint has no valid action_vocab")
    if len(vocab) != len(set(vocab)):
        raise SystemExit("checkpoint action_vocab contains duplicates")
    Path(args.output).write_text(json.dumps(vocab, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"checkpoint": args.checkpoint, "actionCount": len(vocab), "output": args.output}))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
