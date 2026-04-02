package harness

import (
	"fmt"
	"strings"
)

func CheckField(resp map[string]any, field string, expected any) Finding {
	actual, ok := resp[field]
	if !ok {
		return Finding{Field: field, Expected: fmt.Sprintf("%v", expected), Actual: "<missing>", Grade: GradeFail}
	}
	if fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected) {
		return Finding{Field: field, Expected: fmt.Sprintf("%v", expected), Actual: fmt.Sprintf("%v", actual), Grade: GradeFail}
	}
	return Finding{Field: field, Expected: fmt.Sprintf("%v", expected), Actual: fmt.Sprintf("%v", actual), Grade: GradePass}
}

func CheckFieldExists(resp map[string]any, field string) Finding {
	if _, ok := resp[field]; !ok {
		return Finding{Field: field, Expected: "<present>", Actual: "<missing>", Grade: GradePartial}
	}
	return Finding{Field: field, Expected: "<present>", Actual: "<present>", Grade: GradePass}
}

func CheckNotNil(val any, name string) Finding {
	if val == nil {
		return Finding{Field: name, Expected: "<not nil>", Actual: "<nil>", Grade: GradeFail}
	}
	return Finding{Field: name, Expected: "<not nil>", Actual: "<not nil>", Grade: GradePass}
}

func CheckAWSError(err error, expectedCode string) Finding {
	if err == nil {
		return Finding{Field: "error", Expected: expectedCode, Actual: "<nil>", Grade: GradeFail}
	}
	if strings.Contains(err.Error(), expectedCode) {
		return Finding{Field: "error", Expected: expectedCode, Actual: err.Error(), Grade: GradePass}
	}
	return Finding{Field: "error", Expected: expectedCode, Actual: err.Error(), Grade: GradeFail}
}
