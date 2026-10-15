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

    def test_go_requires_three_pass_events(self):
        for action, names in [
            ("pass", adapter.GO_TESTS),
            ("pass", set()),
            ("pass", {"TestDynamoMQClientSendMessage"}),
            ("skip", adapter.GO_TESTS),
            ("fail", adapter.GO_TESTS),
        ]:
            with self.subTest(action=action, names=names):
                events = [{"Action": action, "Test": name} for name in names]
                # A successful package with zero matching tests is insufficient.
                events.append({"Action": "pass", "Package": "github.com/vvatanabe/dynamomq"})
                self.report.write_text("\n".join(json.dumps(event) for event in events))
                with contextlib.redirect_stdout(io.StringIO()):
                    if action != "pass" or names != adapter.GO_TESTS:
                        with self.assertRaises(ValueError):
                            adapter.verify_dynamomq(self.report)
                    else:
                        adapter.verify_dynamomq(self.report)

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
