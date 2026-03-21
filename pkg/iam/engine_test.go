package iam

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_AllowAll(t *testing.T) {
	engine := NewEngine()
	engine.AddPolicy("user1", &Policy{
		Version: "2012-10-17",
		Statements: []Statement{
			{
				Effect:    "Allow",
				Actions:   []string{"*"},
				Resources: []string{"*"},
			},
		},
	})

	result := engine.Evaluate(&EvalRequest{
		Principal: "user1",
		Action:    "s3:GetObject",
		Resource:  "arn:aws:s3:::any-bucket/any-key",
	})

	require.NotNil(t, result)
	assert.Equal(t, Allow, result.Decision)
}

func TestEngine_ExplicitDeny(t *testing.T) {
	engine := NewEngine()
	engine.AddPolicy("user1", &Policy{
		Version: "2012-10-17",
		Statements: []Statement{
			{
				Effect:    "Allow",
				Actions:   []string{"s3:*"},
				Resources: []string{"*"},
			},
			{
				Effect:    "Deny",
				Actions:   []string{"s3:DeleteBucket"},
				Resources: []string{"*"},
			},
		},
	})

	result := engine.Evaluate(&EvalRequest{
		Principal: "user1",
		Action:    "s3:DeleteBucket",
		Resource:  "arn:aws:s3:::my-bucket",
	})

	require.NotNil(t, result)
	assert.Equal(t, Deny, result.Decision)
	assert.Equal(t, "explicit deny", result.Reason)
}

func TestEngine_ImplicitDeny(t *testing.T) {
	engine := NewEngine()
	engine.AddPolicy("user1", &Policy{
		Version: "2012-10-17",
		Statements: []Statement{
			{
				Effect:    "Allow",
				Actions:   []string{"s3:GetObject"},
				Resources: []string{"arn:aws:s3:::my-bucket/*"},
			},
		},
	})

	result := engine.Evaluate(&EvalRequest{
		Principal: "user1",
		Action:    "s3:PutObject",
		Resource:  "arn:aws:s3:::my-bucket/key.txt",
	})

	require.NotNil(t, result)
	assert.Equal(t, Deny, result.Decision)
	assert.Equal(t, "implicit deny", result.Reason)
}

func TestEngine_ResourceScoping(t *testing.T) {
	engine := NewEngine()
	engine.AddPolicy("user1", &Policy{
		Version: "2012-10-17",
		Statements: []Statement{
			{
				Effect:    "Allow",
				Actions:   []string{"s3:GetObject"},
				Resources: []string{"arn:aws:s3:::my-bucket/*"},
			},
		},
	})

	// Allowed resource
	result := engine.Evaluate(&EvalRequest{
		Principal: "user1",
		Action:    "s3:GetObject",
		Resource:  "arn:aws:s3:::my-bucket/key.txt",
	})
	require.NotNil(t, result)
	assert.Equal(t, Allow, result.Decision)

	// Denied resource (different bucket)
	result = engine.Evaluate(&EvalRequest{
		Principal: "user1",
		Action:    "s3:GetObject",
		Resource:  "arn:aws:s3:::other-bucket/key.txt",
	})
	require.NotNil(t, result)
	assert.Equal(t, Deny, result.Decision)
}

func TestEngine_NoPolicies(t *testing.T) {
	engine := NewEngine()

	result := engine.Evaluate(&EvalRequest{
		Principal: "user1",
		Action:    "s3:GetObject",
		Resource:  "arn:aws:s3:::any-bucket/any-key",
	})

	require.NotNil(t, result)
	assert.Equal(t, Deny, result.Decision)
	assert.Equal(t, "implicit deny", result.Reason)
}

func TestEngine_RootBypass(t *testing.T) {
	engine := NewEngine()
	// No policies attached to root user.

	result := engine.Evaluate(&EvalRequest{
		Principal: "root",
		Action:    "s3:DeleteBucket",
		Resource:  "arn:aws:s3:::any-bucket",
		IsRoot:    true,
	})

	require.NotNil(t, result)
	assert.Equal(t, Allow, result.Decision)
}
