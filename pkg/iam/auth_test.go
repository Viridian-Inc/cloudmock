package iam

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractAccessKeyID(t *testing.T) {
	t.Run("valid SigV4 header extracts key ID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		req.Header.Set("Authorization",
			"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20230101/us-east-1/s3/aws4_request, SignedHeaders=host;x-amz-date, Signature=abc123")

		keyID, err := ExtractAccessKeyID(req)
		require.NoError(t, err)
		assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", keyID)
	})

	t.Run("empty Authorization header returns error", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://example.com", nil)

		_, err := ExtractAccessKeyID(req)
		assert.Error(t, err)
	})
}
