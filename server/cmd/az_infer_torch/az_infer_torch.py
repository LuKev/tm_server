#!/usr/bin/env python3
"""Serve a PyTorch policy/value checkpoint for Go MCTS."""

import argparse
import base64
import importlib.util
import json
import os
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Optional

for entry in list(sys.path):
    path = Path(entry)
    if (path / "cmd" / "az_infer_torch").exists() and (path / "internal").exists():
        sys.path.remove(entry)
stdlib_cmd = Path(os.__file__).resolve().parent / "cmd.py"
if stdlib_cmd.exists():
    spec = importlib.util.spec_from_file_location("cmd", stdlib_cmd)
    cmd_module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(cmd_module)
    sys.modules["cmd"] = cmd_module

try:
    import torch
    from torch import nn
except ModuleNotFoundError as exc:
    print("PyTorch is required for az_infer_torch. Install torch in the Bazel Python runtime.", file=sys.stderr)
    raise SystemExit(2) from exc


class PolicyValueNet(nn.Module):
    def __init__(self, input_size: int, action_count: int, hidden_size: int):
        super().__init__()
        self.body = nn.Sequential(
            nn.Linear(input_size, hidden_size),
            nn.ReLU(),
            nn.Linear(hidden_size, hidden_size),
            nn.ReLU(),
        )
        self.policy = nn.Linear(hidden_size, action_count)
        self.value = nn.Sequential(nn.Linear(hidden_size, 1), nn.Tanh())

    def forward(self, features):
        hidden = self.body(features)
        return self.policy(hidden), self.value(hidden)


class HexPolicyValueNet(nn.Module):
    def __init__(self, input_size: int, action_count: int, hidden_size: int, observation_shape):
        super().__init__()
        if len(observation_shape) != 3:
            raise ValueError(f"hex architecture requires [global, hex, per_hex] shape, got {observation_shape}")
        self.input_size = input_size
        self.global_size = int(observation_shape[0])
        self.hex_count = int(observation_shape[1])
        self.per_hex_size = int(observation_shape[2])
        if self.global_size + self.hex_count * self.per_hex_size > input_size:
            raise ValueError("observation shape exceeds input size")
        self.hex_encoder = nn.Sequential(
            nn.Linear(self.per_hex_size, hidden_size),
            nn.ReLU(),
            nn.Linear(hidden_size, hidden_size),
            nn.ReLU(),
        )
        self.global_encoder = nn.Sequential(
            nn.Linear(self.global_size, hidden_size),
            nn.ReLU(),
        )
        self.body = nn.Sequential(
            nn.Linear(hidden_size * 3, hidden_size),
            nn.ReLU(),
            nn.Linear(hidden_size, hidden_size),
            nn.ReLU(),
        )
        self.policy = nn.Linear(hidden_size, action_count)
        self.value = nn.Sequential(nn.Linear(hidden_size, 1), nn.Tanh())

    def forward(self, features):
        global_features = features[:, : self.global_size]
        board_start = self.global_size
        board_end = board_start + self.hex_count * self.per_hex_size
        board = features[:, board_start:board_end].reshape(-1, self.hex_count, self.per_hex_size)
        encoded_hexes = self.hex_encoder(board)
        mean_pool = encoded_hexes.mean(dim=1)
        max_pool = encoded_hexes.max(dim=1).values
        global_hidden = self.global_encoder(global_features)
        hidden = self.body(torch.cat([global_hidden, mean_pool, max_pool], dim=1))
        return self.policy(hidden), self.value(hidden)


def build_model(architecture: str, input_size: int, action_count: int, hidden_size: int, observation_shape):
    if architecture == "hex":
        return HexPolicyValueNet(input_size, action_count, hidden_size, observation_shape)
    if architecture == "mlp":
        return PolicyValueNet(input_size, action_count, hidden_size)
    raise ValueError(f"unknown architecture: {architecture}")


class InferenceService:
    def __init__(self, checkpoint_path: Path, vocab_path: Optional[Path]):
        checkpoint = torch.load(checkpoint_path, map_location="cpu")
        self.input_size = int(checkpoint["input_size"])
        self.action_count = int(checkpoint["action_count"])
        self.hidden_size = int(checkpoint["hidden_size"])
        self.architecture = str(checkpoint.get("architecture", "mlp"))
        self.observation_schema = str(checkpoint.get("observation_schema", ""))
        self.observation_shape = list(checkpoint.get("observation_shape", []))
        self.action_vocab = checkpoint.get("action_vocab")
        if self.action_vocab is None:
            if vocab_path is None:
                manifest = checkpoint.get("manifest", {})
                raw_vocab = manifest.get("vocabPath")
                vocab_path = Path(raw_vocab) if raw_vocab else None
            if vocab_path is None:
                raise ValueError("checkpoint has no action_vocab; pass --vocab")
            self.action_vocab = json.loads(vocab_path.read_text(encoding="utf-8"))
        self.index_by_action = {action_id: index for index, action_id in enumerate(self.action_vocab)}
        self.net = build_model(self.architecture, self.input_size, self.action_count, self.hidden_size, self.observation_shape)
        self.net.load_state_dict(checkpoint["state_dict"])
        self.net.eval()

    def health(self) -> dict:
        return {
            "ok": True,
            "inputSize": self.input_size,
            "actionCount": self.action_count,
            "architecture": self.architecture,
            "observationSchema": self.observation_schema,
            "observationShape": self.observation_shape,
        }

    def evaluate(self, request: dict) -> dict:
        return self.evaluate_many([request])[0]

    def evaluate_many(self, requests: list[dict]) -> list[dict]:
        if not requests:
            return []
        batch_features = []
        known_by_request = []
        legal_by_request = []
        for request in requests:
            features = list(request.get("encoding", []))
            if len(features) < self.input_size:
                features.extend([0.0] * (self.input_size - len(features)))
            else:
                features = features[: self.input_size]
            legal_actions = list(request.get("legalActions", []))
            known = [(action_id, self.index_by_action[action_id]) for action_id in legal_actions if action_id in self.index_by_action]
            batch_features.append(features)
            legal_by_request.append(legal_actions)
            known_by_request.append(known)
        responses = []
        with torch.no_grad():
            logits_batch, values = self.net(torch.tensor(batch_features, dtype=torch.float32))
        for row, legal_actions, known in zip(range(len(requests)), legal_by_request, known_by_request):
            logits = logits_batch[row]
            priors = {}
            if known:
                indices = torch.tensor([index for _, index in known], dtype=torch.long)
                probs = torch.softmax(logits.index_select(0, indices), dim=0)
                for (action_id, _), prob in zip(known, probs):
                    priors[action_id] = float(prob)
            else:
                uniform = 1.0 / max(1, len(legal_actions))
                priors = {action_id: uniform for action_id in legal_actions}
            responses.append({"priors": priors, "value": float(values[row][0])})
        return responses

    def evaluate_binary(self, request: dict) -> dict:
        return self.evaluate_many_binary(request)[0]

    def evaluate_many_binary(self, request: dict) -> list[dict]:
        count = int(request.get("count", 0))
        input_size = int(request.get("inputSize", 0))
        if count <= 0:
            return []
        if input_size <= 0:
            raise ValueError("binary request requires positive inputSize")
        raw = base64.b64decode(request.get("features", ""))
        expected = count * input_size * 4
        if len(raw) != expected:
            raise ValueError(f"binary features length {len(raw)} != expected {expected}")
        features = torch.frombuffer(bytearray(raw), dtype=torch.float32).reshape(count, input_size)
        if input_size < self.input_size:
            pad = torch.zeros((count, self.input_size - input_size), dtype=torch.float32)
            features = torch.cat([features, pad], dim=1)
        elif input_size > self.input_size:
            features = features[:, : self.input_size]
        legal_by_request = list(request.get("legalActions", []))
        if len(legal_by_request) != count:
            raise ValueError("legalActions count does not match feature count")
        known_by_request = []
        for legal_actions in legal_by_request:
            known_by_request.append([(action_id, self.index_by_action[action_id]) for action_id in legal_actions if action_id in self.index_by_action])
        responses = []
        with torch.no_grad():
            logits_batch, values = self.net(features)
        for row, legal_actions, known in zip(range(count), legal_by_request, known_by_request):
            logits = logits_batch[row]
            priors = {}
            if known:
                indices = torch.tensor([index for _, index in known], dtype=torch.long)
                probs = torch.softmax(logits.index_select(0, indices), dim=0)
                for (action_id, _), prob in zip(known, probs):
                    priors[action_id] = float(prob)
            else:
                uniform = 1.0 / max(1, len(legal_actions))
                priors = {action_id: uniform for action_id in legal_actions}
            responses.append({"priors": priors, "value": float(values[row][0])})
        return responses

def make_handler(service: InferenceService, access_log: bool):
    class Handler(BaseHTTPRequestHandler):
        def do_GET(self):
            if self.path != "/healthz":
                self.send_response(404)
                self.end_headers()
                return
            raw = json.dumps(service.health()).encode("utf-8")
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(raw)))
            self.end_headers()
            self.wfile.write(raw)

        def do_POST(self):
            if self.path not in ("/evaluate", "/evaluate_batch", "/evaluate_binary", "/evaluate_batch_binary"):
                self.send_response(404)
                self.end_headers()
                return
            length = int(self.headers.get("Content-Length", "0"))
            try:
                request = json.loads(self.rfile.read(length))
                if self.path == "/evaluate_batch":
                    requests = request.get("requests", request)
                    if not isinstance(requests, list):
                        raise ValueError("batch request must be a list or {requests: [...]}")
                    response = {"responses": service.evaluate_many(requests)}
                elif self.path == "/evaluate_batch_binary":
                    response = {"responses": service.evaluate_many_binary(request)}
                elif self.path == "/evaluate_binary":
                    response = service.evaluate_binary(request)
                else:
                    response = service.evaluate(request)
                raw = json.dumps(response).encode("utf-8")
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(raw)))
                self.end_headers()
                self.wfile.write(raw)
            except Exception as exc:
                raw = json.dumps({"error": str(exc)}).encode("utf-8")
                self.send_response(500)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(raw)))
                self.end_headers()
                self.wfile.write(raw)

        def log_message(self, fmt, *args):
            if access_log:
                print(fmt % args, file=sys.stderr)

    return Handler


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--checkpoint", required=True)
    parser.add_argument("--vocab", default="")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=9097)
    parser.add_argument("--access-log", action="store_true", help="write HTTP access logs to stderr")
    parser.add_argument("--torch-threads", type=int, default=1, help="PyTorch intra-op CPU threads")
    parser.add_argument("--torch-interop-threads", type=int, default=1, help="PyTorch inter-op CPU threads")
    args = parser.parse_args()
    if args.torch_threads > 0:
        torch.set_num_threads(args.torch_threads)
    if args.torch_interop_threads > 0:
        torch.set_num_interop_threads(args.torch_interop_threads)
    service = InferenceService(Path(args.checkpoint), Path(args.vocab) if args.vocab else None)
    server = ThreadingHTTPServer((args.host, args.port), make_handler(service, args.access_log))
    print(f"serving torch evaluator on http://{args.host}:{args.port}/evaluate", file=sys.stderr)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
