"""Fault injection tests for stage publication, without expensive game runs."""
from pathlib import Path
import json
import shutil
import sys
import tempfile
import unittest

from run_az_cycles import adopt, atomic_json, checkpoint, digest, stage, verify_journal, tool_identity, runtime_pins


class RecoveryTest(unittest.TestCase):
    def test_every_stage_boundary(self):
        for role in ("actor", "learner", "evaluation"):
            for boundary in ("after_command", "before_publish", "after_publish"):
                with self.subTest(role=role, boundary=boundary), tempfile.TemporaryDirectory() as temporary:
                    root = Path(temporary)
                    identity = {"cycle": 1, "parent_sha256": "frozen"}
                    def command(output):
                        return [sys.executable, "-c", "from pathlib import Path; import sys; Path(sys.argv[1]).write_text('deterministic output')", str(output / "artifact")]
                    def interrupt(point):
                        if point == boundary:
                            raise RuntimeError("injected interruption")
                    with self.assertRaisesRegex(RuntimeError, "injected"):
                        stage(root, role, identity, command, interrupt)
                    output = stage(root, role, identity, command)
                    self.assertEqual((output / "artifact").read_text(), "deterministic output")
                    self.assertEqual(len(list(root.glob(role))), 1)
                    self.assertEqual(len(list((root / "quarantine").iterdir())), int(boundary != "after_publish"))
                    def must_not_repeat(output):
                        self.fail("published stage was repeated")
                    stage(root, role, identity, must_not_repeat)
                    restored = root / "relocated"
                    restored.mkdir()
                    shutil.copytree(output, restored / role)
                    stage(restored, role, identity, must_not_repeat)
                    (restored / role / "artifact").write_text("corrupt")
                    with self.assertRaisesRegex(ValueError, "mismatch"):
                        stage(restored, role, identity, must_not_repeat)

    def test_partial_failed_process_is_quarantined(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            def failed(output):
                return [sys.executable, "-c", "from pathlib import Path; import sys; Path(sys.argv[1]).write_text('partial'); sys.exit(7)", str(output / "artifact")]
            with self.assertRaisesRegex(RuntimeError, "failed"):
                stage(root, "actor", {}, failed)
            self.assertFalse((root / "actor").exists())
            stage(root, "actor", {}, lambda output: [sys.executable, "-c", "pass"])
            self.assertFalse((root / "actor/artifact").exists())
            self.assertEqual(len(list((root / "quarantine").iterdir())), 1)

    def test_checkpoint_reference_hash_and_local_path(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            snapshot = root / "checkpoint.pt"
            snapshot.write_bytes(b"weights")
            atomic_json(root / "latest.json", {"path": snapshot.name, "checkpoint_sha256": digest(snapshot)})
            self.assertEqual(checkpoint(root), snapshot)
            snapshot.write_bytes(b"changed")
            with self.assertRaisesRegex(ValueError, "invalid checkpoint"):
                checkpoint(root)

    def test_missing_committed_output_is_not_regenerated(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            output = stage(root, "actor", {}, lambda output: [sys.executable, "-c", "pass"])
            verify_journal(root)
            shutil.rmtree(output)
            with self.assertRaises(FileNotFoundError):
                verify_journal(root)

    def test_missing_journal_rejected(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            with self.assertRaisesRegex(ValueError, "missing committed"):
                verify_journal(root)

    def test_imported_source_is_part_of_tool_identity(self):
        with tempfile.TemporaryDirectory() as temporary:
            executable = Path(temporary) / "learner"
            executable.write_text("unchanged launcher")
            module = Path(temporary) / "learner.runfiles/tm_server/internal/az/python/model.py"
            module.parent.mkdir(parents=True)
            module.write_text("version = 1")
            before = tool_identity(executable)
            module.write_text("version = 2")
            after = tool_identity(executable)
            self.assertEqual(before["executable"], after["executable"])
            self.assertNotEqual(before, after)
            alias = Path(temporary) / "alias"
            alias.symlink_to(executable)
            self.assertEqual(after, tool_identity(alias, require_python=True))
            module.unlink()
            with self.assertRaisesRegex(ValueError, "missing project Python"):
                tool_identity(alias, require_python=True)

    def test_runtime_dependency_pins_are_hashed(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / "WORKSPACE").write_text("python runtime 1")
            (root / "az_requirements_lock.txt").write_text("torch pin 1")
            before = runtime_pins(root)
            (root / "az_requirements_lock.txt").write_text("torch pin 2")
            self.assertNotEqual(runtime_pins(root), before)

    def test_adoption_never_rolls_back_during_resume(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            candidates = []
            for cycle in (1, 2):
                candidate = root / f"checkpoints/cycle-{cycle}/checkpoint.pt"
                candidate.parent.mkdir(parents=True)
                candidate.write_text(f"weights-{cycle}")
                atomic_json(candidate.parent / "latest.json", {"path": candidate.name, "checkpoint_sha256": digest(candidate)})
                candidates.append(candidate)
                adopt(root, cycle, candidate)
            expected = (root / "checkpoints/latest.json").read_bytes()
            # Simulate latest for cycle2 committed, progress still on cycle1.
            atomic_json(root / "progress.json", {"completed_cycle": 1, "next_cycle": 2, "checkpoint_sha256": digest(candidates[0])})
            adopt(root, 1, candidates[0])
            self.assertEqual((root / "checkpoints/latest.json").read_bytes(), expected)
            adopt(root, 2, candidates[1])
            self.assertEqual(json.loads((root / "progress.json").read_text())["completed_cycle"], 2)


if __name__ == "__main__":
    unittest.main()
