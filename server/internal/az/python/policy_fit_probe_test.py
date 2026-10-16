import copy
import math
from pathlib import Path
import tempfile
import unittest

import numpy as np
import torch

from learner import Example, ReplayWindow, _training_batch
from model import ModelConfig, TerraMysticaNet, model_fingerprint
from policy_fit_probe import fixed_refs, minibatches, objective, per_position_metrics, run_arm, shared_gradients, write_exclusive
from schema import EncodedPosition, MANIFEST


def examples():
    spatial = np.zeros((MANIFEST.spatial_features, MANIFEST.grid_height, MANIFEST.grid_width), np.float32)
    spatial[0] = 1
    actions = np.zeros((2, MANIFEST.action_features), np.float32)
    actions[1, 0] = 1
    encoded = EncodedPosition(spatial, np.zeros(MANIFEST.global_features, np.float32), actions,
                              np.full((2, 2), -1, np.int64))
    return [Example(encoded, np.array([1., 0.], np.float32), 1.),
            Example(encoded, np.array([.5, .5], np.float32), -1.)]


class PolicyProbeTest(unittest.TestCase):
    def setUp(self):
        torch.set_num_threads(1)
        torch.manual_seed(17)

    def test_metrics_mask_ties_and_exact_values(self):
        logits = torch.tensor([[0., 0., torch.finfo(torch.float32).min], [2., torch.finfo(torch.float32).min, torch.finfo(torch.float32).min]])
        target = torch.tensor([[.5, .5, 0.], [1., 0., 0.]])
        metrics = per_position_metrics(logits, torch.tensor([0., .5]), target, torch.tensor([1., -.5]),
                                       torch.tensor([[True, True, False], [True, False, False]]))
        torch.testing.assert_close(metrics["target_to_prediction_kl"], torch.zeros(2))
        torch.testing.assert_close(metrics["top_visit_agreement"], torch.ones(2))
        torch.testing.assert_close(metrics["value_mse"], torch.ones(2))
        torch.testing.assert_close(metrics["legal_logit_spread"], torch.zeros(2))
        self.assertAlmostEqual(float(metrics["target_entropy"][0]), math.log(2), places=6)

    def test_policy_only_has_no_value_head_gradient_and_readonly_gradient_probe(self):
        model = TerraMysticaNet(ModelConfig(8, 1))
        batch, policy, outcomes = _training_batch(examples(), "cpu")
        objective(model, batch, policy, outcomes, "policy_only").backward()
        self.assertTrue(all(parameter.grad is None for parameter in model.value.parameters()))
        fingerprint = model_fingerprint(model)
        gradients = shared_gradients(model, examples(), "cpu")
        self.assertEqual(fingerprint, model_fingerprint(model))
        self.assertTrue(math.isfinite(gradients["policy_norm"]))
        model.zero_grad(set_to_none=True)
        objective(model, batch, policy, outcomes, "joint").backward()
        self.assertGreater(sum(float(parameter.grad.abs().sum()) for parameter in model.value.parameters()), 0)

    def test_sampling_and_cpu_run_repeatability(self):
        replay = ReplayWindow([examples(), examples()], [(1, 2), (1, 2)])
        refs = fixed_refs(replay, 3, 731)
        self.assertEqual(refs, fixed_refs(replay, 3, 731))
        self.assertEqual(len(set(refs)), 3)
        self.assertEqual(len(fixed_refs(replay, 100, 731)), 4)
        stream = minibatches(2, 2, 2, 731)
        self.assertEqual(stream, minibatches(2, 2, 2, 731))
        model = TerraMysticaNet(ModelConfig(8, 1))
        duplicate = copy.deepcopy(model)
        kwargs = dict(arm="joint", device="cpu", learning_rate=1e-4)
        first = run_arm(model, examples(), examples(), stream, **kwargs)
        second = run_arm(duplicate, examples(), examples(), stream, **kwargs)
        self.assertEqual(first["initial_fingerprint"], second["initial_fingerprint"])
        self.assertEqual(first["final_fingerprint"], second["final_fingerprint"])
        self.assertEqual(first["snapshots"], second["snapshots"])

    def test_exclusive_report(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "result.json"
            write_exclusive(path, {"result": 1})
            with self.assertRaises(FileExistsError):
                write_exclusive(path, {"result": 2})
            self.assertIn('"result": 1', path.read_text())
            self.assertEqual(list(Path(directory).iterdir()), [path])


if __name__ == "__main__":
    unittest.main()
