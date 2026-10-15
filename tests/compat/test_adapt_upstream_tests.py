import contextlib
import io
import json
from pathlib import Path
import tempfile
import unittest

import adapt_upstream_tests as adapter


class TestExecutionEvidence(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.report = Path(self.temp.name) / "result"

    def test_python_requires_both_tests_to_pass(self):
        for omitted, skipped in [(False, False), (True, False), (False, True)]:
            with self.subTest(omitted=omitted, skipped=skipped):
                names = sorted(adapter.PYTHON_TESTS)
                if omitted:
                    names.pop()
                child = "<skipped/>" if skipped else ""
                cases = "".join(f'<testcase name="{name}">{child}</testcase>' for name in names)
                self.report.write_text(f"<testsuites><testsuite>{cases}</testsuite></testsuites>")
                with contextlib.redirect_stdout(io.StringIO()):
                    if omitted or skipped:
                        with self.assertRaises(ValueError):
                            adapter.verify_python(self.report)
                    else:
                        adapter.verify_python(self.report)

    def go_events(self, failures=()):
        failed = set(failures)
        failed.update(name.split("/")[0] for name in failures)
        events = [{"Action": "start"}]
        for name in sorted(adapter.GO_CASES):
            events.append({"Action": "run", "Test": name})
            events.append({"Action": "fail" if name in failed else "pass", "Test": name})
        events.append({"Action": "fail" if failed else "pass"})
        return [dict(event, Package=adapter.GO_PACKAGE) for event in events]

    def write_go_events(self, path, events):
        path.write_text("\n".join(json.dumps(event) for event in events))

    def test_go_requires_complete_test_tree_and_exit_status(self):
        complete = self.go_events()
        bad_event = dict(complete[2], Action="skip")
        unexpected = dict(complete[2], Test="TestUnselected")
        cases = {
            "complete": (complete, 0),
            "zero tests": ([complete[0], complete[-1]], 0),
            "missing subtest": (complete[:3] + complete[5:], 0),
            "missing package end": (complete[:-1], 0),
            "duplicate result": (complete[:3] + [complete[2]] + complete[3:], 0),
            "skip": (complete[:2] + [bad_event] + complete[3:], 0),
            "unexpected test": (complete[:2] + [unexpected] + complete[3:], 0),
            "wrong package": ([dict(complete[0], Package="another/module")] + complete[1:], 0),
            "build failure": ([complete[0], dict(complete[-1], Action="fail")], 1),
            "wrong exit status": (complete, 1),
            "abnormal exit status": (complete, 2),
            "events after completion": (complete + [complete[2]], 0),
        }
        for label, (events, status) in cases.items():
            with self.subTest(label=label):
                self.write_go_events(self.report, events)
                if label == "complete":
                    result = adapter.read_dynamomq_results(self.report, status)
                    self.assertEqual(set(result), adapter.GO_CASES)
                else:
                    with self.assertRaises(ValueError):
                        adapter.read_dynamomq_results(self.report, status)

    def test_go_baseline_preserves_existing_failures_without_hiding_regressions(self):
        prior_failure = "TestDynamoMQClientReceiveMessage/should_return_message_when_exists_ready_message"
        other_receive = "TestDynamoMQClientReceiveMessage/should_return_EmptyQueueError_when_queue_is_empty"
        baseline = Path(self.temp.name) / "baseline"
        self.write_go_events(baseline, self.go_events({prior_failure}))
        for label, failures, expected_error in [
            ("same existing failure", {prior_failure}, False),
            ("fixed existing failure", set(), False),
            ("new failure under failing parent", {prior_failure, other_receive}, True),
            ("same pass count but different failure", {other_receive}, True),
            ("current endpoint unavailable", adapter.GO_LEAF_TESTS, True),
        ]:
            with self.subTest(label=label):
                self.write_go_events(self.report, self.go_events(failures))
                output = io.StringIO()
                with contextlib.redirect_stdout(output):
                    if expected_error:
                        with self.assertRaisesRegex(ValueError, "pass -> fail"):
                            adapter.compare_dynamomq(self.report, baseline, int(bool(failures)), 1)
                    else:
                        adapter.compare_dynamomq(self.report, baseline, int(bool(failures)), 1)
                self.assertIn(f"Baseline failure: {prior_failure}", output.getvalue())

    def test_go_baseline_rejects_no_passing_subtests(self):
        self.write_go_events(self.report, self.go_events(adapter.GO_LEAF_TESTS))
        with self.assertRaisesRegex(ValueError, "no passing subtests"):
            adapter.compare_dynamomq(self.report, self.report, 1, 1)

    def test_go_rejects_truncated_json(self):
        self.report.write_text('{"Action": "run"')
        with self.assertRaises(ValueError):
            adapter.read_dynamomq_results(self.report, 1)

    def test_unexpected_fixture_fails_before_writing(self):
        path = Path(self.temp.name) / "tests/unit/mock_test.py"
        path.parent.mkdir(parents=True)
        source = "import moto\n# The expected mock decorator has changed.\n"
        path.write_text(source)
        with self.assertRaises(ValueError):
            adapter.prepare_python(Path(self.temp.name))
        self.assertEqual(path.read_text(), source)


if __name__ == "__main__":
    unittest.main()
