package iam

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWildcardMatch(t *testing.T) {
	tests := []struct {
		pattern  string
		value    string
		expected bool
	}{
		{"*", "anything", true},
		{"s3:*", "s3:GetObject", true},
		{"s3:Get*", "s3:GetObject", true},
		{"s3:Get*", "s3:PutObject", false},
		{"s3:GetObject", "s3:GetObject", true},
		{"s3:GetObject", "s3:PutObject", false},
		{"arn:aws:s3:::my-bucket/*", "arn:aws:s3:::my-bucket/key", true},
		{"arn:aws:s3:::my-bucket/*", "arn:aws:s3:::other-bucket/key", false},
		{"arn:aws:s3:::*", "arn:aws:s3:::my-bucket", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"|"+tt.value, func(t *testing.T) {
			result := WildcardMatch(tt.pattern, tt.value)
			assert.Equal(t, tt.expected, result, "WildcardMatch(%q, %q)", tt.pattern, tt.value)
		})
	}
}
