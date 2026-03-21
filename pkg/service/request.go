package service

import (
	"io"
	"net/http"
)

// ParseRequestBody reads and returns the full body from an HTTP request.
func ParseRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}
