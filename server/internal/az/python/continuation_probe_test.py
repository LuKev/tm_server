import copy
import unittest

import numpy as np
import torch

from continuation_probe import calibration, fixed_examples, mode_agreement
from learner import Example, ReplayWindow, losses, train
from model import ModelConfig, TerraMysticaNet
from schema import EncodedPosition, MANIFEST


class ProbeTest(unittest.TestCase):
    def test_calibration_and_diagnostics_do_not_change_segmented_training(self):
        torch.set_num_threads(1)
        torch.manual_seed(17)
        spatial = np.zeros((MANIFEST.spatial_features, MANIFEST.grid_height, MANIFEST.grid_width), np.float32)
        spatial[0] = 1
        encoded = EncodedPosition(spatial, np.zeros(MANIFEST.global_features, np.float32),
                                  np.zeros((2, MANIFEST.action_features), np.float32),
                                  np.full((2, 2), -1, np.int64))
        examples = [Example(encoded, np.array([1., 0.], np.float32), -1.),
                    Example(encoded, np.array([0., 1.], np.float32), 1.)]
        replay = ReplayWindow([examples], [(1, 2)])
        model = TerraMysticaNet(ModelConfig(8, 1))
        split = copy.deepcopy(model)
        self.assertEqual(calibration(model, [], "cpu"), None)
        report = calibration(model, examples, "cpu")
        self.assertEqual(report["target_mean"], 0.)
        self.assertEqual(report["diagnostic_target_mean_constant_mse"], 1.)
        self.assertAlmostEqual(report["value_mse"], losses(model, examples)["value"], places=6)
        self.assertAlmostEqual(report["policy_loss"], losses(model, examples)["policy"], places=6)
        self.assertEqual(mode_agreement(model, examples, "cpu"),
                         {"logit_max_abs_difference": 0., "value_max_abs_difference": 0.})
        self.assertEqual(len(fixed_examples(replay, False, 100, 19)), 2)
        kwargs = dict(batch_size=2, learning_rate=.001, weight_decay=.0001, seed=19,
                      optimizer_name="sgd", momentum=.9)
        _, _, full_state = train(model, replay, steps=4, **kwargs)
        _, optimizer, state = train(split, replay, steps=2, **kwargs)
        calibration(split, examples, "cpu")
        mode_agreement(split, examples, "cpu")
        _, _, split_state = train(split, replay, steps=2, optimizer_state=optimizer, learner_state=state, **kwargs)
        self.assertEqual(full_state["sampling_rng"], split_state["sampling_rng"])
        for key, value in model.state_dict().items():
            torch.testing.assert_close(value, split.state_dict()[key], rtol=0, atol=0)


if __name__ == "__main__":
    unittest.main()
