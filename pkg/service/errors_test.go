package service

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAWSError_XMLMarshal(t *testing.T) {
	err := NewAWSError("NoSuchBucket", "The specified bucket does not exist", http.StatusNotFound)

	data, marshalErr := xml.Marshal(err)
	require.NoError(t, marshalErr)

	xmlStr := string(data)
	assert.Contains(t, xmlStr, "<Code>NoSuchBucket</Code>")
	assert.Contains(t, xmlStr, "<Message>The specified bucket does not exist</Message>")
}

func TestAWSError_JSONMarshal(t *testing.T) {
	err := NewAWSError("NoSuchBucket", "The specified bucket does not exist", http.StatusNotFound)

	data, marshalErr := json.Marshal(err)
	require.NoError(t, marshalErr)

	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result))

	assert.Equal(t, "NoSuchBucket", result["__type"])
	assert.Equal(t, "The specified bucket does not exist", result["Message"])
}

func TestAWSError_Error(t *testing.T) {
	err := NewAWSError("NoSuchBucket", "The specified bucket does not exist", http.StatusNotFound)
	assert.Equal(t, "NoSuchBucket: The specified bucket does not exist", err.Error())
}

func TestAWSError_StatusCode(t *testing.T) {
	err := NewAWSError("NoSuchBucket", "The specified bucket does not exist", http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, err.StatusCode())
}

func TestErrNotFound(t *testing.T) {
	err := ErrNotFound("resource", "my-resource")
	assert.Equal(t, "NoSuchKey", err.Code)
	assert.Equal(t, http.StatusNotFound, err.StatusCode())
}

func TestErrAlreadyExists(t *testing.T) {
	err := ErrAlreadyExists("resource", "my-resource")
	assert.Equal(t, "AlreadyExists", err.Code)
	assert.Equal(t, http.StatusConflict, err.StatusCode())
}

func TestErrAccessDenied(t *testing.T) {
	err := ErrAccessDenied("s3:GetObject")
	assert.Equal(t, "AccessDenied", err.Code)
	assert.Equal(t, http.StatusForbidden, err.StatusCode())
}

func TestErrValidation(t *testing.T) {
	err := ErrValidation("field must not be empty")
	assert.Equal(t, "ValidationError", err.Code)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode())
}
