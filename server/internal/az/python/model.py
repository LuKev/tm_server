"""Masked axial hex-CNN with ragged legal-action policy and scalar value heads."""

from __future__ import annotations

from dataclasses import asdict, dataclass
import hashlib
import json
import os
from pathlib import Path
import tempfile
from typing import Any

import torch
from torch import Tensor, nn
from torch.nn import functional as F

from schema import MANIFEST, SchemaManifest


@dataclass(frozen=True)
class ModelConfig:
    channels: int
    residual_blocks: int


DEBUG_CONFIG = ModelConfig(channels=64, residual_blocks=4)
MAIN_CONFIG = ModelConfig(channels=128, residual_blocks=10)
LARGE_CONFIG = ModelConfig(channels=256, residual_blocks=20)
MODEL_CONFIGS = {
    "debug": DEBUG_CONFIG,
    "main": MAIN_CONFIG,
    "large": LARGE_CONFIG,
}


def model_config(name: str) -> ModelConfig:
    try:
        return MODEL_CONFIGS[name]
    except KeyError as error:
        raise ValueError(f"unknown model config {name!r}") from error


def checkpoint_model_config(path: str | Path) -> ModelConfig:
    payload = torch.load(path, map_location="cpu", weights_only=False)
    raw = ((payload.get("manifest") or {}).get("model") or {})
    if set(raw) != {"channels", "residual_blocks"}:
        raise ValueError(f"checkpoint is missing a valid model config: {raw!r}")
    return ModelConfig(channels=int(raw["channels"]), residual_blocks=int(raw["residual_blocks"]))


def resolve_model_config(name: str, checkpoint: str | Path | None = None) -> tuple[str, ModelConfig]:
    stored = checkpoint_model_config(checkpoint) if checkpoint else None
    if name == "auto":
        selected = stored or DEBUG_CONFIG
    else:
        selected = model_config(name)
        if stored is not None and stored != selected:
            raise ValueError(f"checkpoint model config {stored!r} does not match requested {selected!r}")
    for known_name, known_config in MODEL_CONFIGS.items():
        if selected == known_config:
            return known_name, selected
    return f"{selected.residual_blocks}x{selected.channels}", selected


def resolve_device(name: str) -> torch.device:
    if name == "auto":
        if torch.cuda.is_available():
            return torch.device("cuda")
        if torch.backends.mps.is_available():
            return torch.device("mps")
        return torch.device("cpu")
    if name not in ("cpu", "cuda", "mps"):
        raise ValueError(f"unsupported device {name!r}")
    device = torch.device(name)
    if device.type == "cuda" and not torch.cuda.is_available():
        raise ValueError("CUDA was requested but is unavailable")
    if device.type == "mps" and not torch.backends.mps.is_available():
        raise ValueError("MPS was requested but is unavailable")
    return device


class HexConv2d(nn.Module):
    """A learned center plus the six axial neighbors on the fixed q/r grid."""

    def __init__(self, in_channels: int, out_channels: int, bias: bool = True) -> None:
        super().__init__()
        self.weight = nn.Parameter(torch.empty(out_channels, in_channels, 3, 3))
        self.bias = nn.Parameter(torch.empty(out_channels)) if bias else None
        kernel_mask = torch.zeros(1, 1, 3, 3)
        for row, column in ((1, 1), (1, 2), (0, 2), (0, 1), (1, 0), (2, 0), (2, 1)):
            kernel_mask[0, 0, row, column] = 1
        self.register_buffer("kernel_mask", kernel_mask)
        nn.init.kaiming_normal_(self.weight, nonlinearity="relu")
        if self.bias is not None:
            nn.init.zeros_(self.bias)

    def forward(self, value: Tensor) -> Tensor:
        return F.conv2d(value, self.weight * self.kernel_mask, self.bias, padding=1)


class ResidualHexBlock(nn.Module):
    def __init__(self, channels: int, global_features: int) -> None:
        super().__init__()
        self.conv1 = HexConv2d(channels, channels, bias=False)
        self.norm1 = nn.GroupNorm(8, channels)
        self.conv2 = HexConv2d(channels, channels, bias=False)
        self.norm2 = nn.GroupNorm(8, channels)
        self.film = nn.Linear(global_features, 2 * channels)

    def forward(self, value: Tensor, global_features: Tensor, valid_mask: Tensor) -> Tensor:
        residual = value
        value = F.relu(self.norm1(self.conv1(value * valid_mask)))
        value = self.norm2(self.conv2(value * valid_mask))
        scale, bias = self.film(global_features).chunk(2, dim=-1)
        value = value * (1 + scale[:, :, None, None]) + bias[:, :, None, None]
        return F.relu(value + residual) * valid_mask


class TerraMysticaNet(nn.Module):
    def __init__(self, config: ModelConfig = DEBUG_CONFIG, manifest: SchemaManifest = MANIFEST) -> None:
        super().__init__()
        self.config = config
        self.manifest = manifest
        channels = config.channels
        self.stem = HexConv2d(manifest.spatial_features, channels, bias=False)
        self.stem_norm = nn.GroupNorm(8, channels)
        self.global_broadcast = nn.Linear(manifest.global_features, channels)
        self.blocks = nn.ModuleList(
            ResidualHexBlock(channels, manifest.global_features) for _ in range(config.residual_blocks)
        )
        self.action_global = nn.Sequential(
            nn.Linear(manifest.global_features, channels), nn.ReLU(),
        )
        policy_input = manifest.action_features + 3 * channels
        self.policy = nn.Sequential(
            nn.Linear(policy_input, channels), nn.ReLU(), nn.Linear(channels, 1),
        )
        self.value = nn.Sequential(
            nn.Linear(channels + manifest.global_features, channels),
            nn.ReLU(),
            nn.Linear(channels, 1),
            nn.Tanh(),
        )

    def trunk(self, spatial: Tensor, global_features: Tensor) -> tuple[Tensor, Tensor]:
        valid_mask = spatial[:, :1]
        value = spatial * valid_mask
        value = self.stem_norm(self.stem(value))
        value = F.relu(value + self.global_broadcast(global_features)[:, :, None, None]) * valid_mask
        for block in self.blocks:
            value = block(value, global_features, valid_mask)
        return value, valid_mask

    def forward(
        self,
        spatial: Tensor,
        global_features: Tensor,
        action_features: Tensor,
        action_hex_indices: Tensor,
        action_mask: Tensor,
    ) -> tuple[Tensor, Tensor]:
        trunk, valid_mask = self.trunk(spatial, global_features)
        batch, channels, height, width = trunk.shape
        flat = trunk.reshape(batch, channels, height * width).transpose(1, 2)
        gathered: list[Tensor] = []
        for reference in range(2):
            indices = action_hex_indices[:, :, reference]
            safe_indices = indices.clamp_min(0)
            cells = torch.gather(flat, 1, safe_indices[:, :, None].expand(-1, -1, channels))
            gathered.append(cells * (indices >= 0)[:, :, None])
        global_action = self.action_global(global_features)[:, None, :].expand(-1, action_features.shape[1], -1)
        policy_input = torch.cat((action_features, gathered[0], gathered[1], global_action), dim=-1)
        logits = self.policy(policy_input).squeeze(-1)
        logits = logits.masked_fill(~action_mask, torch.finfo(logits.dtype).min)

        masked_sum = (trunk * valid_mask).sum(dim=(2, 3))
        valid_count = valid_mask.sum(dim=(2, 3)).clamp_min(1)
        pooled = masked_sum / valid_count
        value = self.value(torch.cat((pooled, global_features), dim=-1)).squeeze(-1)
        return logits, value


def checkpoint_manifest(model: TerraMysticaNet) -> dict[str, Any]:
    return {
        "schema": model.manifest.as_dict(),
        "model": asdict(model.config),
    }


def model_fingerprint(model: TerraMysticaNet) -> str:
    digest = hashlib.sha256(json.dumps(checkpoint_manifest(model), sort_keys=True).encode())
    for name, value in sorted(model.state_dict().items()):
        array = value.detach().cpu().contiguous().numpy()
        digest.update(name.encode())
        digest.update(str(array.dtype).encode())
        digest.update(str(array.shape).encode())
        digest.update(array.tobytes())
    return digest.hexdigest()


def _write_immutable(path: Path, payload: dict[str, Any]) -> str:
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.exists():
        raise FileExistsError(f"refusing to replace immutable checkpoint {path}")
    with tempfile.NamedTemporaryFile(prefix=".checkpoint-", suffix=".tmp", dir=path.parent, delete=False) as temporary:
        temporary_path = Path(temporary.name)
    try:
        torch.save(payload, temporary_path)
        with temporary_path.open("rb") as saved:
            os.fsync(saved.fileno())
            file_hash = hashlib.sha256(saved.read()).hexdigest()
        os.link(temporary_path, path)
        path.chmod(0o444)
        return file_hash
    finally:
        temporary_path.unlink(missing_ok=True)


def save_checkpoint(path: str | Path, model: TerraMysticaNet, **extra: Any) -> None:
    payload = {
        "manifest": checkpoint_manifest(model),
        "model_state": model.state_dict(),
        **extra,
    }
    _write_immutable(Path(path), payload)


def publish_checkpoint(
    directory: str | Path,
    model: TerraMysticaNet,
    provenance: dict[str, Any],
    *,
    optimizer_state: dict[str, Any] | None = None,
    optimizer_config: dict[str, Any] | None = None,
) -> dict[str, Any]:
    output = Path(directory)
    output.mkdir(parents=True, exist_ok=True)
    model_id = "model-" + model_fingerprint(model)
    payload = {
        "manifest": checkpoint_manifest(model),
        "model_state": model.state_dict(),
        "model_id": model_id,
        "provenance": provenance,
    }
    if optimizer_state is not None or optimizer_config is not None:
        if optimizer_state is None or optimizer_config is None:
            raise ValueError("optimizer state and config must be published together")
        payload["optimizer_state"] = optimizer_state
        payload["optimizer_config"] = optimizer_config
    with tempfile.NamedTemporaryFile(prefix=".publish-", suffix=".pt", dir=output, delete=False) as temporary:
        temporary_path = Path(temporary.name)
    temporary_path.unlink()
    try:
        torch.save(payload, temporary_path)
        checkpoint_sha256 = hashlib.sha256(temporary_path.read_bytes()).hexdigest()
        target = output / f"checkpoint-{checkpoint_sha256}.pt"
        try:
            os.link(temporary_path, target)
        except FileExistsError:
            if target.read_bytes() != temporary_path.read_bytes():
                raise FileExistsError(f"checkpoint hash collision at {target}")
        target.chmod(0o444)
    finally:
        temporary_path.unlink(missing_ok=True)

    latest = {"path": target.name, "checkpoint_sha256": checkpoint_sha256, "model_id": model_id}
    with tempfile.NamedTemporaryFile("w", prefix=".latest-", suffix=".tmp", dir=output, delete=False) as temporary:
        json.dump(latest, temporary, sort_keys=True)
        temporary.flush()
        os.fsync(temporary.fileno())
        latest_temporary = Path(temporary.name)
    latest_temporary.replace(output / "latest.json")
    return {**latest, "path": str(target)}


def load_checkpoint(
    path: str | Path,
    model: TerraMysticaNet,
    *,
    expected_manifest: dict[str, Any] | None = None,
) -> dict[str, Any]:
    payload = torch.load(path, map_location="cpu", weights_only=False)
    expected = expected_manifest or checkpoint_manifest(model)
    actual = payload.get("manifest")
    if actual != expected:
        raise ValueError(f"checkpoint manifest mismatch: got {actual!r}, want {expected!r}")
    model.load_state_dict(payload["model_state"], strict=True)
    return payload
