"""Fault injection tests for stage publication, without expensive game runs."""
from pathlib import Path
import json
import shutil
import sys
import tempfile
import unittest
from unittest import mock
from types import SimpleNamespace

import run_az_cycles as runner

from run_az_cycles import adopt, atomic_json, checkpoint, digest, stage, verify_journal, tool_identity, runtime_pins


class RecoveryTest(unittest.TestCase):
    def seed_config(self, seed=None, learner_seed_base=None):
        args = SimpleNamespace(engine_commit="test", cycles=2, games=4, simulations=8,
                               train_steps=3, eval_cases=1, model_config="debug",
                               selfplay="actor", inference="inference", learner="learner", evaluator="eval")
        env = {} if seed is None else {"TM_AZ_MODEL_SEED": seed}
        if learner_seed_base is not None:
            env["TM_AZ_LEARNER_SEED_BASE"] = learner_seed_base
        with mock.patch.dict(runner.os.environ, env, clear=True), mock.patch.object(runner, "tool_identity", return_value={}), mock.patch.object(runner, "runtime_pins", return_value={}):
            return args, runner.configuration(args)

    def test_model_seed_configuration_and_resume_are_immutable(self):
        args, default = self.seed_config()
        self.assertEqual(default["model_seed"], "0")
        self.assertEqual(default["learner_seed_base"], "0")
        _, config = self.seed_config("0017")
        self.assertEqual(config["model_seed"], "17")
        for invalid in ("-1", str(2**63), "1.5"):
            with self.assertRaises(ValueError):
                self.seed_config(invalid)
        for invalid in ("-1", str(2**63 - 2), "1.5"):
            with self.assertRaises(ValueError):
                self.seed_config("17", invalid)
        self.assertEqual(self.seed_config("17", str(2**63 - 3))[1]["learner_seed_base"], str(2**63 - 3))
        with tempfile.TemporaryDirectory() as temporary, mock.patch.object(runner, "execute_cycles"):
            args.run_dir, args.mode = temporary, "new"
            with mock.patch.object(runner, "configuration", return_value=config):
                runner.run(args)
                args.mode = "resume"
                runner.run(args)
            with mock.patch.object(runner, "configuration", return_value=default), self.assertRaisesRegex(ValueError, "immutable manifest"):
                runner.run(args)
            _, changed_sampling = self.seed_config("17", "100")
            with mock.patch.object(runner, "configuration", return_value=changed_sampling), self.assertRaisesRegex(ValueError, "immutable manifest"):
                runner.run(args)
            # Older manifests stay bound to their original runner revision;
            # do not silently migrate them by injecting a default seed.
            old = dict(config)
            del old["model_seed"]
            atomic_json(Path(temporary) / "run-manifest.json", old)
            with mock.patch.object(runner, "configuration", return_value=config), self.assertRaisesRegex(ValueError, "immutable manifest"):
                runner.run(args)

    def test_model_seed_reaches_actor_learner_and_initial_parent(self):
        args, config = self.seed_config("17", "100")
        commands = []
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            candidate = root / "weights.pt"
            candidate.write_bytes(b"fixed checkpoint")
            def capture(root, relative, identity, command):
                output = root / relative
                output.mkdir(parents=True)
                commands.append(command(output))
                return output
            with mock.patch.object(runner, "stage", side_effect=capture), mock.patch.object(runner, "checkpoint", return_value=candidate), mock.patch.object(runner, "adopt"):
                runner.execute_cycles(args, root, config)
            self.assertEqual(json.loads((root / "cycle-1-parent.json").read_text())["parent_sha256"], "random-seed-17")
            self.assertEqual(json.loads((root / "cycle-2-parent.json").read_text())["parent_sha256"], digest(candidate))
            seeded = [cmd for cmd in commands if cmd[0] in ("actor", "learner")]
            self.assertEqual(len(seeded), 4)
            for command in seeded:
                self.assertEqual(command[command.index("--model-seed") + 1], "17")
            learners = [cmd for cmd in seeded if cmd[0] == "learner"]
            self.assertEqual([cmd[cmd.index("--seed") + 1] for cmd in learners], ["101", "102"])
            actors = [cmd for cmd in seeded if cmd[0] == "actor"]
            self.assertEqual(actors[0][actors[0].index("--checkpoint") + 1], "")
            self.assertEqual(actors[1][actors[1].index("--checkpoint") + 1], str(candidate))

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
