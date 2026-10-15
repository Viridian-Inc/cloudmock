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
GO_TESTS = {
    "TestDynamoMQClientSendMessage",
    "TestDynamoMQClientReceiveMessage",
    "TestDynamoMQClientDeleteMessage",
}


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


def verify_dynamomq(report):
    passed = set()
    for line in report.read_text().splitlines():
        event = json.loads(line)
        if event.get("Action") in {"fail", "skip"}:
            raise ValueError(f"DynamoMQ compatibility test did not pass: {event.get('Test', event.get('Package'))}")
        if event.get("Action") == "pass" and event.get("Test") in GO_TESTS:
            passed.add(event["Test"])
    if passed != GO_TESTS:
        raise ValueError(f"expected all three upstream DynamoMQ tests to run; passed={sorted(passed)}")
    print("Verified all three upstream DynamoMQ tests passed against CloudMock")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("operation", choices=["prepare-python", "prepare-dynamomq", "verify-python", "verify-dynamomq"])
    parser.add_argument("path", type=Path)
    args = parser.parse_args()
    operations = {
        "prepare-python": prepare_python,
        "prepare-dynamomq": prepare_dynamomq,
        "verify-python": verify_python,
        "verify-dynamomq": verify_dynamomq,
    }
    try:
        operations[args.operation](args.path)
    except (OSError, ValueError, ET.ParseError) as error:
        parser.exit(1, f"Compatibility fixture error: {error}\n")


if __name__ == "__main__":
    main()
