import unittest

import torch

from policy_gate_probe import aggregate, pair_metrics


class GateProbeTest(unittest.TestCase):
    def test_identical_gates_cancel_context_but_not_action_derivative(self):
        result = pair_metrics(torch.ones(2, 2), torch.tensor([[2., 3.], [4., 5.]]),
                              torch.tensor([[1., 2.]]), 1, 1)
        self.assertEqual(result["identical_gate_fraction"], 1.)
        self.assertEqual(result["context_pair_jacobian_rms"], 0.)
        self.assertEqual(result["action_jacobian_rms"], 10.)

    def test_distinct_gates_match_autograd_and_zero_context_preserves_bias(self):
        weight = torch.tensor([[2., 3.], [4., 5.]])
        output = torch.tensor([[1., 2.]])
        actions = torch.tensor([[-1.], [1.]])
        bias = torch.tensor([.1, -.2])
        context = torch.tensor([.5], requires_grad=True)
        preactivation = actions @ weight[:, :1].T + context @ weight[:, 1:].T + bias
        logits = preactivation.relu() @ output.T
        jacobian = torch.autograd.grad(logits[0] - logits[1], context)[0]
        result = pair_metrics(preactivation, weight, output, 1, 1)
        self.assertEqual(result["identical_gate_fraction"], 0.)
        self.assertAlmostEqual(result["context_pair_jacobian_rms"], float(jacobian.abs()), places=6)
        zero_preactivation = actions @ weight[:, :1].T + bias
        torch.testing.assert_close(zero_preactivation, torch.tensor([[-1.9, -4.2], [2.1, 3.8]]))

    def test_aggregation_and_one_action(self):
        self.assertIsNone(pair_metrics(torch.ones(1, 2), torch.ones(2, 2), torch.ones(1, 2), 1, 1))
        result = aggregate([{"pairs": 1, "metric": 2.}, {"pairs": 3, "metric": 4.}])
        self.assertEqual(result["position_equal"]["metric"], 3.)
        self.assertEqual(result["pair_weighted"]["metric"], 3.5)


if __name__ == "__main__":
    unittest.main()
