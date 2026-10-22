package awsclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// Only buffer small, known-length JSON requests. Larger requests and streams
// keep their existing transport behavior and memory use.
const coalesceLimit = 64 * 1024

// coalescingClient keeps the benchmark's SDK requests intact while allowing
// net/http to send small JSON headers and bodies in one buffered write. Smithy's
// safe reader hides the underlying bytes.Reader, which otherwise makes net/http
// flush headers separately even though the entire JSON body is already in memory.
type coalescingClient struct {
	base aws.HTTPClient
}

func (c *coalescingClient) Do(req *http.Request) (*http.Response, error) {
	if req.Body == nil || req.Body == http.NoBody || req.ContentLength <= 0 ||
		req.ContentLength > coalesceLimit || len(req.TransferEncoding) != 0 ||
		!isJSONContentType(req.Header.Get("Content-Type")) {
		return c.base.Do(req)
	}

	// Like net/http, cancellation must close a body whose Read is blocked.
	// Coordinate normal completion and cancellation so the original closes once.
	var once sync.Once
	var closeErr error
	closeBody := func() {
		once.Do(func() { closeErr = req.Body.Close() })
	}
	stop := context.AfterFunc(req.Context(), closeBody)
	defer stop()
	defer closeBody()
	if err := req.Context().Err(); err != nil {
		return nil, err
	}

	// Read one extra byte to reject an incorrect declared length. This remains
	// bounded by 64 KiB + 1 even if a misbehaving reader has more data available.
	body, err := io.ReadAll(io.LimitReader(req.Body, req.ContentLength+1))
	closeBody()
	if contextErr := req.Context().Err(); contextErr != nil {
		return nil, contextErr
	}
	if err != nil {
		return nil, fmt.Errorf("read benchmark request body: %w", err)
	}
	if int64(len(body)) != req.ContentLength {
		return nil, fmt.Errorf("benchmark request ContentLength=%d does not match body length %d", req.ContentLength, len(body))
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close benchmark request body: %w", closeErr)
	}

	clone := req.Clone(req.Context())
	clone.Body = io.NopCloser(bytes.NewReader(body))
	// Keep GetBody unchanged: SDK retries still build their own request body,
	// and the underlying transport retains its original replay policy.
	return c.base.Do(clone)
}

func isJSONContentType(value string) bool {
	mediaType, _, _ := strings.Cut(value, ";")
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "application/json", "application/x-amz-json-1.0", "application/x-amz-json-1.1":
		return true
	default:
		return false
	}
}
