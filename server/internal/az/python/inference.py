"""Long-lived JSON-lines inference service for the local correctness loop."""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import sys
from typing import Any

import torch

from batching import collate_positions
from model import MODEL_CONFIGS, TerraMysticaNet, load_checkpoint, model_fingerprint, resolve_device, resolve_model_config
from schema import encode_position


def serve(model: TerraMysticaNet, metadata: dict[str, Any], device: torch.device) -> None:
    model.eval()
    for line in sys.stdin:
        try:
            request = json.loads(line)
            if request.get("type") == "metadata":
                print(json.dumps({"metadata": metadata}, separators=(",", ":")), flush=True)
                continue
            if request.get("type") != "evaluate":
                raise ValueError("request type must be 'metadata' or 'evaluate'")
            records = request.get("requests") or []
            if not records:
                raise ValueError("evaluate request must contain positions")
            encoded = [encode_position(item["state"], item["actions"]) for item in records]
            batch = collate_positions(encoded, device)
            with torch.inference_mode():
                logits, values = model(*batch.model_inputs())
            results = []
            for index, item in enumerate(encoded):
                count = item.action_features.shape[0]
                results.append(
                    {
                        "policy_logits": logits[index, :count].detach().cpu().tolist(),
                        "value": float(values[index].detach().cpu()),
                    }
                )
            response = {"results": results}
        except Exception as error:  # Keep the process alive and make protocol errors explicit.
            response = {"error": f"{type(error).__name__}: {error}"}
        print(json.dumps(response, separators=(",", ":")), flush=True)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--model-config", choices=("auto", *MODEL_CONFIGS), default="auto")
    parser.add_argument("--checkpoint")
    parser.add_argument("--seed", type=int, default=0)
    parser.add_argument("--device", default="cpu", help="cpu, cuda, mps, or auto")
    parser.add_argument("--torch-threads", type=int, default=1, help="CPU threads; zero keeps the PyTorch default")
    args = parser.parse_args()
    if args.torch_threads < 0:
        parser.error("--torch-threads must be non-negative")
    if args.torch_threads:
        torch.set_num_threads(args.torch_threads)
    torch.manual_seed(args.seed)
    device = resolve_device(args.device)
    config_name, config = resolve_model_config(args.model_config, args.checkpoint)
    model = TerraMysticaNet(config).to(device)
    metadata = {
        "model_id": "random-" + model_fingerprint(model),
        "checkpoint_sha256": "",
        "model_config": config_name,
        "device": str(device),
        "dtype": "float32",
        "parameters": sum(parameter.numel() for parameter in model.parameters()),
        "torch_threads": torch.get_num_threads(),
    }
    if args.checkpoint:
        payload = load_checkpoint(args.checkpoint, model)
        model_id = payload.get("model_id")
        if not model_id:
            raise ValueError("checkpoint is missing model_id")
        expected_model_id = "model-" + model_fingerprint(model)
        if model_id != expected_model_id:
            raise ValueError(f"checkpoint model identity mismatch: {model_id} != {expected_model_id}")
        metadata = {
            "model_id": str(model_id),
            "checkpoint_sha256": hashlib.sha256(Path(args.checkpoint).read_bytes()).hexdigest(),
            "model_config": config_name,
            "device": str(device),
            "dtype": "float32",
            "parameters": sum(parameter.numel() for parameter in model.parameters()),
            "torch_threads": torch.get_num_threads(),
        }
    serve(model, metadata, device)


if __name__ == "__main__":
    main()
