#!/usr/bin/env python3
"""Point pinned upstream test fixtures at CloudMock without changing assertions."""

import argparse
import json
from pathlib import Path
import xml.etree.ElementTree as ET


PYTHON_TESTS = {
    "test_lambda_handler_happy_path",
    "test_lambda_handler_notfound_path",
}
GO_PACKAGE = "github.com/vvatanabe/dynamomq"
GO_SUBTESTS = {
    "TestDynamoMQClientSendMessage": {
        "should_succeed_when_id_is_not_duplicated",
        "should_delay_add_to_queue_when_delay_is_set",
    },
    "TestDynamoMQClientReceiveMessage": {
        "should_return_EmptyQueueError_when_queue_is_empty",
        "should_return_message_when_exists_ready_message",
        "should_return_message_when_exists_message_expired_visibility_timeout",
        "should_return_EmptyQueueError_when_exists_message_but_visibility_timeout_is_not_expired",
    },
    "TestDynamoMQClientDeleteMessage": {
        "should_not_return_error_when_not_existing_id",
        "should_succeed_when_id_is_found",
    },
}
GO_TESTS = set(GO_SUBTESTS)
GO_LEAF_TESTS = {f"{parent}/{child}" for parent, children in GO_SUBTESTS.items() for child in children}
GO_CASES = GO_TESTS | GO_LEAF_TESTS


def replace_once(source, old, new):
    if source.count(old) != 1:
        raise ValueError(f"upstream fixture changed: expected exactly one {old!r}")
    return source.replace(old, new, 1)


def prepare_python(checkout):
    path = checkout / "tests/unit/mock_test.py"
    source = path.read_text()
    # Keep the upstream table setup, handler invocation, and all assertions.
    # Boto3 uses the required AWS_ENDPOINT_URL supplied by the workflow.
    source = replace_once(source, "import moto\n", "")
    source = replace_once(source, "@moto.mock_dynamodb\n", "")
    for name in PYTHON_TESTS:
        if source.count(f"def {name}(self):") != 1:
            raise ValueError(f"upstream Python test is missing: {name}")
    path.write_text(source)


def prepare_dynamomq(checkout):
    path = checkout / "client_test.go"
    source = path.read_text()
    source = replace_once(source, '\t"reflect"\n', '\t"reflect"\n\t"os"\n')
    replacement = '''	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	if endpoint == "" {
		t.Fatal("DYNAMODB_ENDPOINT is required for CloudMock compatibility")
	}
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion("us-east-1"))
	if err != nil {
		t.Fatal(err)
	}
	client = dynamodb.NewFromConfig(cfg, func(options *dynamodb.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.RetryMaxAttempts = 1
	})
	clean = func() {
		_, err := client.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{TableName: aws.String(tableName)})
		if err != nil {
			t.Errorf("delete compatibility table: %v", err)
		}
	}'''
    source = replace_once(source, "\tclient, clean = dynamotest.NewDynamoDB(t)", replacement)
    # PrepTable only operates on the supplied client; the replaced constructor
    # was the part that launched an unrelated DynamoDB Local Docker container.
    if source.count("dynamotest.PrepTable(t, client,") != 1:
        raise ValueError("upstream DynamoMQ table setup changed")
    for name in GO_TESTS:
        if source.count(f"func {name}(t *testing.T)") != 1:
            raise ValueError(f"upstream DynamoMQ test is missing: {name}")
    path.write_text(source)


def verify_python(report):
    cases = ET.parse(report).getroot().findall(".//testcase")
    passed = set()
    for case in cases:
        if case.find("failure") is not None or case.find("error") is not None or case.find("skipped") is not None:
            raise ValueError(f"Python compatibility test did not pass: {case.get('name')}")
        passed.add(case.get("name"))
    if len(cases) != len(PYTHON_TESTS) or passed != PYTHON_TESTS:
        raise ValueError(f"expected both upstream Python tests to run; passed={sorted(passed)}")
    print("Verified both upstream Python tests passed against CloudMock")


def read_dynamomq_results(report, exit_code):
    """Require complete evidence from the exact pinned suite, even on failure."""
    if exit_code not in {0, 1}:
        raise ValueError(f"unexpected go test exit status: {exit_code}")
    started = set()
    results = {}
    package_started = False
    package_result = None
    for line in report.read_text().splitlines():
        event = json.loads(line)
        if not isinstance(event, dict) or event.get("Package") != GO_PACKAGE:
            raise ValueError("unexpected DynamoMQ package event")
        action, name = event.get("Action"), event.get("Test")
        if package_result is not None:
            raise ValueError("DynamoMQ events continue after package completion")
        if action == "skip" or action not in {"start", "run", "pause", "cont", "output", "pass", "fail"}:
            raise ValueError(f"unexpected DynamoMQ event: {action} {name}")
        if name is not None and name not in GO_CASES:
            raise ValueError(f"unexpected DynamoMQ test: {name}")
        if action == "start":
            if package_started or name is not None:
                raise ValueError("duplicate or invalid DynamoMQ package start")
            package_started = True
        elif not package_started:
            raise ValueError("missing DynamoMQ package start")
        elif action == "run":
            if name is None or name in started:
                raise ValueError(f"duplicate or invalid DynamoMQ test start: {name}")
            started.add(name)
        elif action in {"pass", "fail"}:
            if name is None:
                package_result = action
            else:
                if name not in started or name in results:
                    raise ValueError(f"duplicate or unstarted DynamoMQ test result: {name}")
                results[name] = action
    if started != GO_CASES or set(results) != GO_CASES:
        missing = sorted(GO_CASES - set(results))
        raise ValueError(f"expected all three DynamoMQ tests and eight subtests to finish; missing={missing}")
    expected_package_result = "fail" if "fail" in results.values() else "pass"
    if package_result != expected_package_result or exit_code != (1 if package_result == "fail" else 0):
        raise ValueError("DynamoMQ package result or go test exit status does not match test evidence")
    for parent in GO_TESTS:
        if any(results[f"{parent}/{child}"] == "fail" for child in GO_SUBTESTS[parent]) and results[parent] != "fail":
            raise ValueError(f"DynamoMQ parent passed with a failing subtest: {parent}")
    return results


def verify_dynamomq(report):
    results = read_dynamomq_results(report, 0)
    if "fail" in results.values():
        raise ValueError("DynamoMQ compatibility tests did not all pass")
    print("Verified all three upstream DynamoMQ tests and eight subtests passed against CloudMock")


def compare_dynamomq(report, baseline, exit_code, baseline_exit_code):
    prior = read_dynamomq_results(baseline, baseline_exit_code)
    current = read_dynamomq_results(report, exit_code)
    if not any(prior[name] == "pass" for name in GO_LEAF_TESTS):
        raise ValueError("DynamoMQ baseline has no passing subtests; cannot check regressions")
    for label, results, status in [("Baseline", prior, baseline_exit_code), ("Current", current, exit_code)]:
        count = sum(results[name] == "pass" for name in GO_LEAF_TESTS)
        print(f"{label} DynamoMQ: {count}/{len(GO_LEAF_TESTS)} subtests passed; go test exit {status}")
    for name in sorted(prior):
        if prior[name] == "fail":
            print(f"Baseline failure: {name} (current: {current[name]})")
    regressions = [name for name in sorted(prior) if prior[name] == "pass" and current[name] != "pass"]
    if regressions:
        raise ValueError("DynamoMQ compatibility regressions: " + "; ".join(f"{name}: pass -> {current[name]}" for name in regressions))
    print("Verified no previously passing DynamoMQ test or subtest regressed")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("operation", choices=["prepare-python", "prepare-dynamomq", "verify-python", "verify-dynamomq", "compare-dynamomq"])
    parser.add_argument("path", type=Path)
    parser.add_argument("--baseline", type=Path)
    parser.add_argument("--exit-code", type=int)
    parser.add_argument("--baseline-exit-code", type=int)
    args = parser.parse_args()
    operations = {
        "prepare-python": prepare_python,
        "prepare-dynamomq": prepare_dynamomq,
        "verify-python": verify_python,
        "verify-dynamomq": verify_dynamomq,
    }
    try:
        if args.operation == "compare-dynamomq":
            if args.baseline is None or args.exit_code is None or args.baseline_exit_code is None:
                parser.error("compare-dynamomq requires --baseline, --exit-code, and --baseline-exit-code")
            compare_dynamomq(args.path, args.baseline, args.exit_code, args.baseline_exit_code)
        else:
            operations[args.operation](args.path)
    except (OSError, ValueError, ET.ParseError) as error:
        parser.exit(1, f"Compatibility fixture error: {error}\n")


if __name__ == "__main__":
    main()
