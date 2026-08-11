"""Shared ragged-action batching for inference and training."""

from __future__ import annotations

from dataclasses import dataclass

import numpy as np
import torch
from torch import Tensor

from schema import EncodedPosition, MANIFEST


@dataclass(frozen=True)
class TensorBatch:
    spatial: Tensor
    global_features: Tensor
    action_features: Tensor
    action_hex_indices: Tensor
    action_mask: Tensor

    def model_inputs(self) -> tuple[Tensor, Tensor, Tensor, Tensor, Tensor]:
        return (
            self.spatial,
            self.global_features,
            self.action_features,
            self.action_hex_indices,
            self.action_mask,
        )


def collate_positions(encoded: list[EncodedPosition], device: torch.device | str = "cpu") -> TensorBatch:
    if not encoded:
        raise ValueError("cannot collate an empty position batch")
    maximum_actions = max(item.action_features.shape[0] for item in encoded)
    if maximum_actions == 0:
        raise ValueError("every non-terminal position must have a legal action")

    batch_size = len(encoded)
    action_features = np.zeros(
        (batch_size, maximum_actions, MANIFEST.action_features), dtype=np.float32
    )
    action_hex_indices = np.full((batch_size, maximum_actions, 2), -1, dtype=np.int64)
    action_mask = np.zeros((batch_size, maximum_actions), dtype=np.bool_)
    for index, item in enumerate(encoded):
        count = item.action_features.shape[0]
        action_features[index, :count] = item.action_features
        action_hex_indices[index, :count] = item.action_hex_indices
        action_mask[index, :count] = True

    return TensorBatch(
        spatial=torch.from_numpy(np.stack([item.spatial for item in encoded])).to(device),
        global_features=torch.from_numpy(np.stack([item.global_features for item in encoded])).to(device),
        action_features=torch.from_numpy(action_features).to(device),
        action_hex_indices=torch.from_numpy(action_hex_indices).to(device),
        action_mask=torch.from_numpy(action_mask).to(device),
    )
