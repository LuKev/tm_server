from types import SimpleNamespace
import unittest

from policy_subset_probe import nested_indices


class SubsetProbeTest(unittest.TestCase):
    def test_uniform_nested_selection_is_repeatable_and_excludes_single_action(self):
        examples = [SimpleNamespace(policy=[1.])] + [SimpleNamespace(policy=[.5, .5]) for _ in range(12)]
        one, eight = nested_indices(examples)
        self.assertEqual([one, eight], nested_indices(examples))
        self.assertEqual(one, eight[:1])
        self.assertEqual(len(set(eight)), 8)
        self.assertNotIn(0, eight)

    def test_insufficient_informative_examples_rejected(self):
        with self.assertRaises(ValueError):
            nested_indices([SimpleNamespace(policy=[.5, .5]) for _ in range(7)])


if __name__ == "__main__":
    unittest.main()
