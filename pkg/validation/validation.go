package validation

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

// ValidationError is returned when input fails validation.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s: %s", e.Field, e.Message)
}

// Required checks that a string field is non-empty.
func Required(field, value string) *ValidationError {
	if value == "" {
		return &ValidationError{Field: field, Message: "is required"}
	}
	return nil
}

// MinLength checks minimum string length.
func MinLength(field, value string, min int) *ValidationError {
	if len(value) < min {
		return &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at least %d characters long", min),
		}
	}
	return nil
}

// MaxLength checks maximum string length.
func MaxLength(field, value string, max int) *ValidationError {
	if len(value) > max {
		return &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at most %d characters long", max),
		}
	}
	return nil
}

// Pattern checks a string matches a regex pattern.
func Pattern(field, value, pattern, description string) *ValidationError {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return &ValidationError{Field: field, Message: fmt.Sprintf("invalid pattern: %s", err)}
	}
	if !re.MatchString(value) {
		return &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must match %s", description),
		}
	}
	return nil
}

// OneOf checks a string is one of the allowed values.
func OneOf(field, value string, allowed []string) *ValidationError {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return &ValidationError{
		Field:   field,
		Message: fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")),
	}
}

// InRange checks an integer is within bounds (inclusive).
func InRange(field string, value, min, max int) *ValidationError {
	if value < min || value > max {
		return &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be between %d and %d", min, max),
		}
	}
	return nil
}

// arnPattern matches arn:partition:service:region:account:resource (region and account may be empty).
var arnPattern = regexp.MustCompile(`^arn:[a-zA-Z0-9\-]+:[a-zA-Z0-9\-]+:[a-zA-Z0-9\-]*:[0-9]*:.+$`)

// ARN validates an ARN format (arn:partition:service:region:account:resource).
func ARN(field, value string) *ValidationError {
	if !arnPattern.MatchString(value) {
		return &ValidationError{
			Field:   field,
			Message: "must be a valid ARN (arn:partition:service:region:account:resource)",
		}
	}
	return nil
}

// namePattern matches typical AWS resource names: alphanumeric, hyphens, underscores, 1-128 chars.
var namePattern = regexp.MustCompile(`^[a-zA-Z0-9_\-]{1,128}$`)

// Name validates a typical AWS resource name (alphanumeric, hyphens, underscores, 1-128 chars).
func Name(field, value string) *ValidationError {
	if !namePattern.MatchString(value) {
		return &ValidationError{
			Field:   field,
			Message: "must be 1-128 characters of alphanumeric, hyphens, or underscores",
		}
	}
	return nil
}

// CIDR validates a CIDR block format.
func CIDR(field, value string) *ValidationError {
	_, _, err := net.ParseCIDR(value)
	if err != nil {
		return &ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be a valid CIDR block (e.g. 10.0.0.0/16): %s", err),
		}
	}
	return nil
}

// Validate runs multiple checks and returns the first error, or nil.
func Validate(checks ...*ValidationError) *ValidationError {
	for _, check := range checks {
		if check != nil {
			return check
		}
	}
	return nil
}
