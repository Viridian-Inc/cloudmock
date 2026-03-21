package service

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
)

// AWSError represents an AWS-compatible error with XML and JSON marshaling support.
type AWSError struct {
	XMLName    xml.Name `xml:"Error"       json:"-"`
	Code       string   `xml:"Code"        json:"-"`
	Message    string   `xml:"Message"     json:"-"`
	statusCode int
}

// NewAWSError creates a new AWSError with the given code, message, and HTTP status code.
func NewAWSError(code, message string, statusCode int) *AWSError {
	return &AWSError{
		Code:       code,
		Message:    message,
		statusCode: statusCode,
	}
}

// Error implements the error interface.
func (e *AWSError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// StatusCode returns the HTTP status code for this error.
func (e *AWSError) StatusCode() int {
	return e.statusCode
}

// MarshalJSON outputs the AWS JSON error format with __type and Message fields.
func (e *AWSError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type    string `json:"__type"`
		Message string `json:"Message"`
	}{
		Type:    e.Code,
		Message: e.Message,
	})
}

// ErrNotFound returns a 404 NoSuchKey error for a named resource.
func ErrNotFound(resourceType, name string) *AWSError {
	return NewAWSError("NoSuchKey", fmt.Sprintf("%s not found: %s", resourceType, name), http.StatusNotFound)
}

// ErrAlreadyExists returns a 409 AlreadyExists error for a named resource.
func ErrAlreadyExists(resourceType, name string) *AWSError {
	return NewAWSError("AlreadyExists", fmt.Sprintf("%s already exists: %s", resourceType, name), http.StatusConflict)
}

// ErrAccessDenied returns a 403 AccessDenied error for an action.
func ErrAccessDenied(action string) *AWSError {
	return NewAWSError("AccessDenied", fmt.Sprintf("Access denied for action: %s", action), http.StatusForbidden)
}

// ErrValidation returns a 400 ValidationError with the given message.
func ErrValidation(message string) *AWSError {
	return NewAWSError("ValidationError", message, http.StatusBadRequest)
}
