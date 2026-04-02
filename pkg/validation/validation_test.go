package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Required ---

func TestRequired_Pass(t *testing.T) {
	assert.Nil(t, Required("name", "something"))
}

func TestRequired_Fail(t *testing.T) {
	err := Required("name", "")
	require.NotNil(t, err)
	assert.Equal(t, "name", err.Field)
}

// --- MinLength ---

func TestMinLength_Pass(t *testing.T) {
	assert.Nil(t, MinLength("name", "hello", 3))
	assert.Nil(t, MinLength("name", "abc", 3)) // exact boundary
}

func TestMinLength_Fail(t *testing.T) {
	err := MinLength("name", "ab", 3)
	require.NotNil(t, err)
	assert.Equal(t, "name", err.Field)
}

// --- MaxLength ---

func TestMaxLength_Pass(t *testing.T) {
	assert.Nil(t, MaxLength("name", "hello", 10))
	assert.Nil(t, MaxLength("name", "hello", 5)) // exact boundary
}

func TestMaxLength_Fail(t *testing.T) {
	err := MaxLength("name", "toolong", 4)
	require.NotNil(t, err)
	assert.Equal(t, "name", err.Field)
}

// --- Pattern ---

func TestPattern_Pass(t *testing.T) {
	assert.Nil(t, Pattern("email", "user@example.com", `^[^@]+@[^@]+\.[^@]+$`, "email address"))
}

func TestPattern_Fail(t *testing.T) {
	err := Pattern("email", "not-an-email", `^[^@]+@[^@]+\.[^@]+$`, "email address")
	require.NotNil(t, err)
	assert.Equal(t, "email", err.Field)
	assert.Contains(t, err.Message, "email address")
}

// --- OneOf ---

func TestOneOf_Pass(t *testing.T) {
	assert.Nil(t, OneOf("status", "active", []string{"active", "inactive", "deleted"}))
}

func TestOneOf_Fail(t *testing.T) {
	err := OneOf("status", "unknown", []string{"active", "inactive"})
	require.NotNil(t, err)
	assert.Equal(t, "status", err.Field)
	assert.Contains(t, err.Message, "active")
}

// --- InRange ---

func TestInRange_Pass(t *testing.T) {
	assert.Nil(t, InRange("count", 5, 1, 10))
	assert.Nil(t, InRange("count", 1, 1, 10))  // lower bound
	assert.Nil(t, InRange("count", 10, 1, 10)) // upper bound
}

func TestInRange_Fail_TooLow(t *testing.T) {
	err := InRange("count", 0, 1, 10)
	require.NotNil(t, err)
	assert.Equal(t, "count", err.Field)
}

func TestInRange_Fail_TooHigh(t *testing.T) {
	err := InRange("count", 11, 1, 10)
	require.NotNil(t, err)
	assert.Equal(t, "count", err.Field)
}

// --- ARN ---

func TestARN_Pass(t *testing.T) {
	cases := []string{
		"arn:aws:s3:::my-bucket",
		"arn:aws:iam::123456789012:role/MyRole",
		"arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
		"arn:aws-cn:s3:::my-bucket",
	}
	for _, c := range cases {
		assert.Nil(t, ARN("arn", c), "expected valid ARN: %s", c)
	}
}

func TestARN_Fail(t *testing.T) {
	cases := []string{
		"",
		"not-an-arn",
		"arn:aws",
		"arn:aws:s3",
	}
	for _, c := range cases {
		err := ARN("arn", c)
		assert.NotNil(t, err, "expected invalid ARN to fail: %s", c)
	}
}

// --- Name ---

func TestName_Pass(t *testing.T) {
	assert.Nil(t, Name("name", "my-resource"))
	assert.Nil(t, Name("name", "resource_name"))
	assert.Nil(t, Name("name", "Resource123"))
	assert.Nil(t, Name("name", "a")) // 1 char
}

func TestName_Fail(t *testing.T) {
	// Empty
	err := Name("name", "")
	assert.NotNil(t, err)

	// Too long (129 chars)
	long := make([]byte, 129)
	for i := range long {
		long[i] = 'a'
	}
	err = Name("name", string(long))
	assert.NotNil(t, err)

	// Invalid characters
	err = Name("name", "has spaces")
	assert.NotNil(t, err)

	err = Name("name", "has@special")
	assert.NotNil(t, err)
}

// --- CIDR ---

func TestCIDR_Pass(t *testing.T) {
	assert.Nil(t, CIDR("cidr", "10.0.0.0/8"))
	assert.Nil(t, CIDR("cidr", "192.168.1.0/24"))
	assert.Nil(t, CIDR("cidr", "0.0.0.0/0"))
}

func TestCIDR_Fail(t *testing.T) {
	cases := []string{
		"",
		"10.0.0.0",
		"not-a-cidr",
		"999.999.999.999/24",
	}
	for _, c := range cases {
		err := CIDR("cidr", c)
		assert.NotNil(t, err, "expected invalid CIDR to fail: %s", c)
	}
}

// --- ValidationError.Error() ---

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{Field: "myField", Message: "is required"}
	assert.Equal(t, "validation error: myField: is required", err.Error())
}

// --- Validate ---

func TestValidate_MultipleChecks(t *testing.T) {
	// First error wins.
	err := Validate(
		Required("a", ""),        // fails
		Required("b", ""),        // also fails, but second
		Required("c", "present"), // passes
	)
	require.NotNil(t, err)
	assert.Equal(t, "a", err.Field)
}

func TestValidate_AllPass(t *testing.T) {
	err := Validate(
		Required("a", "value"),
		MinLength("b", "hello", 3),
		MaxLength("c", "hi", 10),
	)
	assert.Nil(t, err)
}
