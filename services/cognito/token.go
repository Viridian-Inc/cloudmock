package cognito

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// buildToken creates a plausible-looking JWT-like token with a base64-encoded JSON payload.
// It uses the format: header.payload.signature (where signature is a fixed stub).
// This is not a real JWT but is sufficient for mock/test purposes.
func buildToken(claims map[string]interface{}) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))

	payload, err := json.Marshal(claims)
	if err != nil {
		payload = []byte("{}")
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)

	// Stub signature — not a real HMAC/RSA signature.
	sig := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("mock-sig-%s", encodedPayload[:8])))

	return fmt.Sprintf("%s.%s.%s", header, encodedPayload, sig)
}
