import unittest

import torch

from model import ModelConfig, TerraMysticaNet, model_fingerprint
from policy_interaction_probe import BilinearPolicy, core_fingerprint, replace_head, scale_shared_context
from schema import MANIFEST


class InteractionProbeTest(unittest.TestCase):
    def test_context_scaling_preserves_weights_other_inputs_and_callers_tensor(self):
        model = TerraMysticaNet(ModelConfig(8, 1))
        fingerprint = model_fingerprint(model)
        start = MANIFEST.action_features + 16
        inputs = torch.randn(2, MANIFEST.action_features + 32)
        original = inputs.clone()
        expected = inputs.clone()
        expected[:, start:] *= .1
        expected_output = model.policy(expected)
        handle = scale_shared_context(model)
        torch.testing.assert_close(model.policy(inputs), expected_output)
        torch.testing.assert_close(inputs, original, rtol=0, atol=0)
        self.assertEqual(model_fingerprint(model), fingerprint)
        handle.remove()
        inputs.requires_grad_()
        before = model.policy(inputs)
        before_gradient = torch.autograd.grad(before.sum(), inputs)[0]
        handle = scale_shared_context(model, 1.)
        after = model.policy(inputs)
        after_gradient = torch.autograd.grad(after.sum(), inputs)[0]
        torch.testing.assert_close(before, after, rtol=0, atol=0)
        torch.testing.assert_close(before_gradient, after_gradient, rtol=0, atol=0)
        handle.remove()

    def test_context_scaling_does_not_change_value_output(self):
        model = TerraMysticaNet(ModelConfig(8, 1))
        spatial = torch.zeros(1, MANIFEST.spatial_features, MANIFEST.grid_height, MANIFEST.grid_width)
        spatial[:, 0] = 1
        inputs = (spatial, torch.zeros(1, MANIFEST.global_features),
                  torch.zeros(1, 2, MANIFEST.action_features),
                  torch.full((1, 2, 2), -1), torch.tensor([[True, False]]))
        logits_before, before = model(*inputs)
        scale_shared_context(model)
        logits_after, after = model(*inputs)
        torch.testing.assert_close(before, after, rtol=0, atol=0)
        self.assertEqual(float(logits_after[0, 1]), float(logits_before[0, 1]))
        self.assertEqual(float(logits_after[0, 1]), torch.finfo(torch.float32).min)

    def test_context_changes_relative_action_preference(self):
        head = BilinearPolicy(1, 1)
        with torch.no_grad():
            head.action.weight.fill_(1.)
            head.action.bias.zero_()
            head.context.weight.zero_()
            head.context.weight[0, -1] = 1.
            head.context.bias.zero_()
        positive = torch.tensor([[1., 0., 0., 0., 1.], [-1., 0., 0., 0., 1.]])
        negative = positive.clone()
        negative[:, -1] = -1.
        self.assertGreater(float(head(positive)[0] - head(positive)[1]), 0.)
        self.assertLess(float(head(negative)[0] - head(negative)[1]), 0.)
        positive.requires_grad_()
        derivative = torch.autograd.grad(head(positive).sum(), positive)[0]
        self.assertEqual(float(derivative[0, -1]), 1.)
        self.assertEqual(float(derivative[1, -1]), -1.)

    def test_replacement_preserves_core_and_is_reproducible(self):
        torch.manual_seed(44)
        model = TerraMysticaNet(ModelConfig(8, 1))
        before = core_fingerprint(model)
        self.assertEqual(replace_head(model, 144), before)
        weights = {key: value.clone() for key, value in model.policy.state_dict().items()}
        replace_head(model, 144)
        self.assertEqual(core_fingerprint(model), before)
        for key, value in model.policy.state_dict().items():
            torch.testing.assert_close(weights[key], value, rtol=0, atol=0)

    def test_production_forward_keeps_legal_mask(self):
        model = TerraMysticaNet(ModelConfig(8, 1))
        replace_head(model, 144)
        spatial = torch.zeros(1, MANIFEST.spatial_features, MANIFEST.grid_height, MANIFEST.grid_width)
        spatial[:, 0] = 1
        logits, value = model(spatial, torch.zeros(1, MANIFEST.global_features),
                              torch.zeros(1, 2, MANIFEST.action_features),
                              torch.full((1, 2, 2), -1), torch.tensor([[True, False]]))
        self.assertTrue(torch.isfinite(logits[0, 0]))
        self.assertEqual(float(logits[0, 1]), torch.finfo(torch.float32).min)
        self.assertTrue(torch.isfinite(value).all())


if __name__ == "__main__":
    unittest.main()
