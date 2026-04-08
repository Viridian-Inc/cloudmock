package service

import (
	"fmt"
	"io"
	"net/http"
)

// MaxRequestBodySize is the maximum allowed request body size (10 MB).
const MaxRequestBodySize = 10 * 1024 * 1024

// ParseRequestBody reads and returns the full body from an HTTP request.
func ParseRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	defer r.Body.Close()
	reader := io.LimitReader(r.Body, MaxRequestBodySize+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > MaxRequestBodySize {
		return nil, fmt.Errorf("request body too large (max %d bytes)", MaxRequestBodySize)
	}
	return data, nil
}
