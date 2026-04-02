package harness

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckField_Present(t *testing.T) {
	resp := map[string]any{"BucketName": "my-bucket", "CreationDate": "2026-01-01"}
	f := CheckField(resp, "BucketName", "my-bucket")
	assert.Equal(t, GradePass, f.Grade)
}

func TestCheckField_WrongValue(t *testing.T) {
	resp := map[string]any{"BucketName": "other-bucket"}
	f := CheckField(resp, "BucketName", "my-bucket")
	assert.Equal(t, GradeFail, f.Grade)
}

func TestCheckField_Missing(t *testing.T) {
	resp := map[string]any{}
	f := CheckField(resp, "BucketName", "my-bucket")
	assert.Equal(t, GradeFail, f.Grade)
}

func TestCheckFieldExists(t *testing.T) {
	resp := map[string]any{"RequestId": "abc-123"}
	f := CheckFieldExists(resp, "RequestId")
	assert.Equal(t, GradePass, f.Grade)

	f2 := CheckFieldExists(resp, "Missing")
	assert.Equal(t, GradePartial, f2.Grade)
}

func TestCheckNotNil(t *testing.T) {
	f := CheckNotNil("some value", "Response")
	assert.Equal(t, GradePass, f.Grade)

	f2 := CheckNotNil(nil, "Response")
	assert.Equal(t, GradeFail, f2.Grade)
}

func TestCheckAWSError(t *testing.T) {
	f := CheckAWSError(fmt.Errorf("ResourceNotFoundException: Table not found"), "ResourceNotFoundException")
	assert.Equal(t, GradePass, f.Grade)

	f2 := CheckAWSError(fmt.Errorf("InternalServerError"), "ResourceNotFoundException")
	assert.Equal(t, GradeFail, f2.Grade)

	f3 := CheckAWSError(nil, "ResourceNotFoundException")
	assert.Equal(t, GradeFail, f3.Grade)
}
