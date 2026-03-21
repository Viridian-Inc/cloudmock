package gateway

import "net/http"

// CORSMiddleware adds CORS headers to all responses.
// Enabled when CLOUDMOCK_CORS=true (default in dev).
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, HEAD, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers",
				"Content-Type, Authorization, X-Amz-Target, X-Amz-Date, "+
					"X-Amz-Security-Token, X-Amz-Content-Sha256, X-Amz-User-Agent, "+
					"x-api-key, amz-sdk-invocation-id, amz-sdk-request")
			w.Header().Set("Access-Control-Expose-Headers",
				"x-amzn-RequestId, x-amz-request-id, x-amz-id-2, ETag, x-amz-version-id")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
