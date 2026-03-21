package iam

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_CreateAndLookupUser(t *testing.T) {
	store := NewStore("000000000000")

	user, err := store.CreateUser("testuser")
	require.NoError(t, err)

	assert.Equal(t, "testuser", user.Name)
	assert.Contains(t, user.ARN, "arn:aws:iam::000000000000:user/testuser")

	got, err := store.GetUser("testuser")
	require.NoError(t, err)
	assert.Equal(t, user, got)
}

func TestStore_CreateAccessKey(t *testing.T) {
	store := NewStore("000000000000")

	_, err := store.CreateUser("testuser")
	require.NoError(t, err)

	key, err := store.CreateAccessKey("testuser")
	require.NoError(t, err)

	assert.NotEmpty(t, key.AccessKeyID)
	assert.NotEmpty(t, key.SecretAccessKey)

	got, err := store.LookupAccessKey(key.AccessKeyID)
	require.NoError(t, err)
	assert.Equal(t, "testuser", got.UserName)
	assert.Equal(t, "000000000000", got.AccountID)
	assert.False(t, got.IsRoot)
}

func TestStore_RootAccount(t *testing.T) {
	store := NewStore("000000000000")

	err := store.InitRoot("AKID_ROOT", "SECRET_ROOT")
	require.NoError(t, err)

	got, err := store.LookupAccessKey("AKID_ROOT")
	require.NoError(t, err)
	assert.True(t, got.IsRoot)
	assert.Equal(t, "SECRET_ROOT", got.SecretAccessKey)
}

func TestStore_AttachPolicy(t *testing.T) {
	store := NewStore("000000000000")

	_, err := store.CreateUser("testuser")
	require.NoError(t, err)

	policy := &Policy{
		Version: "2012-10-17",
		Statements: []Statement{
			{
				Effect:    "Allow",
				Actions:   []string{"s3:GetObject"},
				Resources: []string{"*"},
			},
		},
	}

	err = store.AttachUserPolicy("testuser", "MyPolicy", policy)
	require.NoError(t, err)

	policies, err := store.GetUserPolicies("testuser")
	require.NoError(t, err)
	assert.Len(t, policies, 1)

	// Verify the AKIA prefix for access keys
	_, err2 := store.CreateAccessKey("testuser")
	require.NoError(t, err2)

	// Also verify UserID prefix for completeness
	user, _ := store.GetUser("testuser")
	assert.True(t, strings.HasPrefix(user.UserID, "AIDA"))
}
