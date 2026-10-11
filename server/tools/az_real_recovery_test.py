"""Real CPU actor interruption, archived relocation, and exact model replay drill."""
import json
import os
from pathlib import Path
import signal
import shutil
import subprocess
import sys
import tarfile
import time
import unittest

import torch


class RecoveryDrill(unittest.TestCase):
    def test_real_interruption_and_restore(self):
        root = Path(os.environ.get("TEST_UNDECLARED_OUTPUTS_DIR", os.environ["TEST_TMPDIR"]))
        root.mkdir(parents=True, exist_ok=True)
        reference, interrupted, restored = (root / name for name in ("reference", "interrupted", "restored"))
        env = {**os.environ, "TM_AZ_INFERENCE_DEVICE": "cpu", "TM_AZ_LEARNER_DEVICE": "cpu",
               "TM_AZ_ACTOR_GAMES_PER_BATCH": "1", "TM_AZ_INFERENCE_MAX_WAIT": "0s",
               "TM_AZ_TORCH_THREADS": "1", "TM_AZ_REPLAY_CACHE_GAMES": "1",
               "TM_AZ_EVAL_BASELINES": "uniform,previous"}

        def command(mode, directory):
            return [*TOOLS[:1], "--mode", mode, *TOOLS[1:], str(directory), "test", "2", "4", "1", "3", "1", "debug"]

        def await_partial(process, directory):
            deadline = time.monotonic() + 120
            while not list(directory.glob("attempts/replay-cycle-2-*/trajectory-*.json.gz")):
                self.assertIsNone(process.poll(), "actor finished before partial-output interruption")
                self.assertLess(time.monotonic(), deadline, "actor did not publish first partial shard")
                time.sleep(0.02)

        # Both continuations must share the *same* immutable parent package.
        # Independently trained cycle-1 files have different provenance timings
        # and hence different byte hashes, even when their weights are equal.
        with (root / "reference.log").open("w") as log:
            process = subprocess.Popen(command("new", reference), env=env, stdout=log, stderr=log)
            try:
                await_partial(process, reference)
                shutil.copytree(reference, interrupted, ignore=shutil.ignore_patterns("attempts", "quarantine"))
                self.assertFalse((interrupted / "replay/cycle-2").exists(), "snapshot missed its cycle-1 boundary")
                self.assertEqual(process.wait(timeout=120), 0)
            finally:
                if process.poll() is None:
                    process.terminate()
                    process.wait(timeout=30)
        with (root / "interruption.log").open("w") as log:
            process = subprocess.Popen(command("resume", interrupted), env=env, stdout=log, stderr=log)
            try:
                await_partial(process, interrupted)
                process.send_signal(signal.SIGTERM)
                self.assertNotEqual(process.wait(timeout=30), 0)
            finally:
                if process.poll() is None:
                    process.kill()
                    process.wait(timeout=5)
        self.assertFalse((interrupted / "replay/cycle-2").exists())
        self.assertTrue((interrupted / "checkpoints/cycle-1/latest.json").is_file())
        archive = root / "recovery-package.tar.gz"
        with tarfile.open(archive, "w:gz") as package:
            package.add(interrupted, arcname="restored")
        with tarfile.open(archive, "r:gz") as package:
            package.extractall(root, filter="data")
        subprocess.run(command("resume", restored), env=env, check=True)
        self.assertTrue(list((restored / "quarantine").iterdir()))
        names = lambda directory: sorted(path.name for path in (directory / "replay").rglob("trajectory-*.json.gz"))
        self.assertEqual(names(reference), names(restored))
        latest = lambda directory: json.loads((directory / "checkpoints/latest.json").read_text())
        self.assertEqual(latest(reference)["model_id"], latest(restored)["model_id"])
        def checkpoint_state(directory):
            path = directory / "checkpoints" / latest(directory)["path"]
            return torch.load(path, map_location="cpu", weights_only=True)

        def identical(left, right):
            if isinstance(left, torch.Tensor):
                torch.testing.assert_close(left, right, rtol=0, atol=0)
            elif isinstance(left, dict):
                self.assertEqual(left.keys(), right.keys())
                for key in left:
                    identical(left[key], right[key])
            elif isinstance(left, (list, tuple)):
                self.assertEqual(len(left), len(right))
                for a, b in zip(left, right):
                    identical(a, b)
            else:
                self.assertEqual(left, right)

        expected, actual = checkpoint_state(reference), checkpoint_state(restored)
        for key in ("optimizer_state", "optimizer_config", "learner_state"):
            identical(expected[key], actual[key])
        before = {str(p.relative_to(restored)): p.read_bytes() for p in restored.rglob("*") if p.is_file()}
        subprocess.run(command("resume", restored), env=env, check=True)
        after = {str(p.relative_to(restored)): p.read_bytes() for p in restored.rglob("*") if p.is_file()}
        self.assertEqual(before, after)
        (root / "drill-result.json").write_text(json.dumps({
            "cpu_model_identical": True, "trajectory_ids_identical": True,
            "optimizer_rng_cursor_identical": True,
            "relocated_archive_resumed": True, "repeated_resume_unchanged": True,
            "model_id": latest(restored)["model_id"], "games": len(names(restored)),
        }, sort_keys=True) + "\n")


if __name__ == "__main__":
    TOOLS = [str(Path(arg).resolve()) for arg in sys.argv[1:6]]
    unittest.main(argv=[sys.argv[0]])
