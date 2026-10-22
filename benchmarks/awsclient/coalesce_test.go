package awsclient

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/require"

	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
	ddbservice "github.com/Viridian-Inc/cloudmock/services/dynamodb"
)

type clientFunc func(*http.Request) (*http.Response, error)

func (f clientFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

type observedBody struct {
	reader io.Reader
	reads  int
	closes int
	err    error
}

func (b *observedBody) Read(p []byte) (int, error) {
	b.reads++
	return b.reader.Read(p)
}

func (b *observedBody) Close() error { b.closes++; return b.err }

func jsonRequest(t *testing.T, body io.ReadCloser, size int64) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "http://localhost/", body)
	require.NoError(t, err)
	req.ContentLength = size
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	return req
}

func TestCoalescingClientPreservesRequest(t *testing.T) {
	payload := []byte(`{"value":"spaces + & < > \u0000 日本語"}`)
	body := &observedBody{reader: bytes.NewReader(payload)}
	req := jsonRequest(t, body, int64(len(payload)))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 signed-request")
	req.Header.Set("X-Amz-Date", "20261021T120000Z")
	headers := req.Header.Clone()
	var replayCalls int
	req.GetBody = func() (io.ReadCloser, error) {
		replayCalls++
		return io.NopCloser(bytes.NewReader(payload)), nil
	}

	wantResponse := &http.Response{StatusCode: http.StatusOK}
	client := &coalescingClient{base: clientFunc(func(got *http.Request) (*http.Response, error) {
		require.NotSame(t, req, got)
		require.Same(t, ctx, got.Context())
		require.Equal(t, headers, got.Header)
		require.Equal(t, req.ContentLength, got.ContentLength)
		data, err := io.ReadAll(got.Body)
		require.NoError(t, err)
		require.Equal(t, payload, data)
		require.NoError(t, got.Body.Close())
		replayed, err := got.GetBody()
		require.NoError(t, err)
		require.NoError(t, replayed.Close())
		got.Header.Set("Authorization", "changed by downstream")
		return wantResponse, nil
	})}
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Same(t, wantResponse, resp)
	require.Same(t, body, req.Body)
	require.Equal(t, headers, req.Header)
	require.Equal(t, 1, body.closes)
	require.Equal(t, 1, replayCalls)
}

func TestCoalescingClientPassesThroughIneligibleRequests(t *testing.T) {
	for _, name := range []string{"nil", "no_body", "unknown", "zero", "large", "chunked", "xml", "expect_continue"} {
		t.Run(name, func(t *testing.T) {
			body := &observedBody{reader: strings.NewReader("{}")}
			req := jsonRequest(t, body, 2)
			switch name {
			case "nil":
				req.Body = nil
			case "no_body":
				req.Body = http.NoBody
			case "unknown":
				req.ContentLength = -1
			case "zero":
				req.ContentLength = 0
			case "large":
				req.ContentLength = coalesceLimit + 1
			case "chunked":
				req.TransferEncoding = []string{"chunked"}
			case "xml":
				req.Header.Set("Content-Type", "application/xml")
			case "expect_continue":
				req.Header.Set("Expect", "100-continue")
			}
			called := false
			client := &coalescingClient{base: clientFunc(func(got *http.Request) (*http.Response, error) {
				called = true
				require.Same(t, req, got)
				require.Zero(t, body.reads)
				require.Zero(t, body.closes)
				return &http.Response{StatusCode: http.StatusOK}, nil
			})}
			_, err := client.Do(req)
			require.NoError(t, err)
			require.True(t, called)
		})
	}
}

type failingReader struct{ err error }

func (r failingReader) Read(p []byte) (int, error) { return copy(p, "bad"), r.err }

func TestCoalescingClientBodyFailures(t *testing.T) {
	readErr := errors.New("read failed")
	closeErr := errors.New("close failed")
	for _, tc := range []struct {
		name   string
		reader io.Reader
		length int64
		close  error
		want   error
	}{
		{"short", strings.NewReader("{}"), 3, nil, nil},
		{"long", strings.NewReader("{}x"), 2, nil, nil},
		{"read", failingReader{readErr}, 3, nil, readErr},
		{"close", strings.NewReader("{}"), 2, closeErr, closeErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &observedBody{reader: tc.reader, err: tc.close}
			req := jsonRequest(t, body, tc.length)
			client := &coalescingClient{base: clientFunc(func(*http.Request) (*http.Response, error) {
				t.Error("invalid body reached HTTP client")
				return nil, nil
			})}
			resp, err := client.Do(req)
			require.Nil(t, resp)
			require.Error(t, err)
			if tc.want != nil {
				require.ErrorIs(t, err, tc.want)
			}
			require.Equal(t, 1, body.closes)
		})
	}
}

func TestCoalescingClientLimitAndDownstreamError(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), coalesceLimit)
	body := &observedBody{reader: bytes.NewReader(payload)}
	req := jsonRequest(t, body, int64(len(payload)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	downstreamErr := errors.New("network failed")
	client := &coalescingClient{base: clientFunc(func(got *http.Request) (*http.Response, error) {
		require.Nil(t, got.GetBody, "do not introduce new redirect/retry replay behavior")
		data, err := io.ReadAll(got.Body)
		require.NoError(t, err)
		require.Equal(t, payload, data)
		require.NoError(t, got.Body.Close())
		return nil, downstreamErr
	})}
	_, err := client.Do(req)
	require.ErrorIs(t, err, downstreamErr)
	require.Equal(t, 1, body.closes)
}

type blockingBody struct {
	started chan struct{}
	closed  chan struct{}
	closes  atomic.Int32
}

func (b *blockingBody) Read([]byte) (int, error) {
	close(b.started)
	<-b.closed
	return 0, errors.New("body closed")
}

func (b *blockingBody) Close() error {
	if b.closes.Add(1) == 1 {
		close(b.closed)
	}
	return nil
}

func TestCoalescingClientCancellation(t *testing.T) {
	for _, when := range []string{"before_read", "during_read"} {
		t.Run(when, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			body := &blockingBody{started: make(chan struct{}), closed: make(chan struct{})}
			req := jsonRequest(t, body, 2).WithContext(ctx)
			client := &coalescingClient{base: clientFunc(func(*http.Request) (*http.Response, error) {
				t.Error("canceled request reached HTTP client")
				return nil, nil
			})}
			if when == "before_read" {
				cancel()
			}
			done := make(chan error, 1)
			go func() { _, err := client.Do(req); done <- err }()
			if when == "during_read" {
				select {
				case <-body.started:
					cancel()
				case <-time.After(2 * time.Second):
					t.Fatal("body read did not start")
				}
			}
			select {
			case err := <-done:
				require.ErrorIs(t, err, context.Canceled)
			case <-time.After(2 * time.Second):
				t.Fatal("cancellation did not finish")
			}
			require.Equal(t, int32(1), body.closes.Load())
		})
	}
}

type countedConn struct {
	net.Conn
	writes atomic.Int32
}

func (c *countedConn) Write(p []byte) (int, error) {
	c.writes.Add(1)
	return c.Conn.Write(p)
}

func TestCoalescingClientSDKUsesOneWireWrite(t *testing.T) {
	for _, coalesce := range []bool{false, true} {
		t.Run(fmt.Sprint(coalesce), func(t *testing.T) {
			clientConn, serverConn := net.Pipe()
			conn := &countedConn{Conn: clientConn}
			t.Cleanup(func() { clientConn.Close(); serverConn.Close() })
			transport := &http.Transport{DialContext: func(context.Context, string, string) (net.Conn, error) { return conn, nil }}
			t.Cleanup(transport.CloseIdleConnections)
			var httpClient aws.HTTPClient = &http.Client{Transport: transport}
			if coalesce {
				httpClient = &coalescingClient{base: httpClient}
			}
			wire := make(chan []byte, 1)
			serverErr := make(chan error, 1)
			go func() {
				req, err := http.ReadRequest(bufio.NewReader(serverConn))
				if err != nil {
					serverErr <- err
					return
				}
				body, err := io.ReadAll(req.Body)
				req.Body.Close()
				if err != nil {
					serverErr <- err
					return
				}
				wire <- body
				_, err = fmt.Fprintf(serverConn, "HTTP/1.1 200 OK\r\nContent-Type: application/x-amz-json-1.0\r\nContent-Length: 2\r\nX-Amz-Crc32: %d\r\n\r\n{}", crc32.ChecksumIEEE([]byte("{}")))
				serverErr <- err
			}()
			client := dynamodb.New(dynamodb.Options{
				Region: "us-east-1", BaseEndpoint: aws.String("http://localhost"), HTTPClient: httpClient,
				Credentials: credentials.NewStaticCredentialsProvider("test", "test", ""), RetryMaxAttempts: 1,
			})
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, err := client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String("wire-table"), Item: map[string]types.AttributeValue{
				"id": &types.AttributeValueMemberS{Value: "key + & 日本語"},
			}})
			require.NoError(t, err)
			require.NoError(t, <-serverErr)
			require.JSONEq(t, `{"TableName":"wire-table","Item":{"id":{"S":"key + & 日本語"}}}`, string(<-wire))
			wantWrites := int32(2)
			if coalesce {
				wantWrites = 1
			}
			require.Equal(t, wantWrites, conn.writes.Load())
		})
	}
}

func TestCoalescingClientSDKRetryPreservesBody(t *testing.T) {
	t.Setenv("AWS_DEFAULTS_MODE", "legacy")
	bodies := make(chan []byte, 2)
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Error(err)
		}
		bodies <- body
		w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, `{"__type":"InternalServerError","message":"retry"}`)
			return
		}
		w.Header().Set("X-Amz-Crc32", fmt.Sprint(crc32.ChecksumIEEE([]byte("{}"))))
		io.WriteString(w, "{}")
	}))
	defer server.Close()
	cfg, err := NewConfig(server.URL)
	require.NoError(t, err)
	cfg.Retryer = func() aws.Retryer {
		return retry.NewStandard(func(o *retry.StandardOptions) {
			o.MaxAttempts = 2
			o.Backoff = retry.BackoffDelayerFunc(func(int, error) (time.Duration, error) { return 0, nil })
		})
	}
	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) { o.BaseEndpoint = aws.String(server.URL) })
	_, err = client.PutItem(context.Background(), &dynamodb.PutItemInput{TableName: aws.String("retry-table"), Item: map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: "same bytes"}}})
	require.NoError(t, err)
	require.Equal(t, int32(2), attempts.Load())
	require.Equal(t, <-bodies, <-bodies)
}

func TestCoalescingClientSDKWrites1000Items(t *testing.T) {
	t.Setenv("AWS_DEFAULTS_MODE", "legacy")
	svc := ddbservice.New("123456789012", "us-east-1")
	t.Cleanup(svc.Close)
	registry := routing.NewRegistry()
	registry.Register(svc)
	server := httptest.NewServer(gateway.TestModeHandler(&service.CallerIdentity{AccountID: "123456789012", IsRoot: true}, "us-east-1", "123456789012", registry))
	defer server.Close()
	cfg, err := NewConfig(server.URL)
	require.NoError(t, err)
	_, wrapped := cfg.HTTPClient.(*coalescingClient)
	require.True(t, wrapped)
	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) { o.BaseEndpoint = aws.String(server.URL); o.RetryMaxAttempts = 1 })
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	table := "coalesced-items"
	_, err = client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: &table, BillingMode: types.BillingModePayPerRequest,
		AttributeDefinitions: []types.AttributeDefinition{{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS}},
		KeySchema:            []types.KeySchemaElement{{AttributeName: aws.String("id"), KeyType: types.KeyTypeHash}},
	})
	require.NoError(t, err)
	for i := 0; i < 1000; i++ {
		_, err = client.PutItem(ctx, &dynamodb.PutItemInput{TableName: &table, Item: map[string]types.AttributeValue{
			"id":   &types.AttributeValueMemberS{Value: fmt.Sprintf("key-%d", i)},
			"data": &types.AttributeValueMemberS{Value: fmt.Sprintf("value-%d + & 日本語", i)},
		}})
		require.NoError(t, err)
	}
	for i := 0; i < 1000; i++ {
		out, err := client.GetItem(ctx, &dynamodb.GetItemInput{TableName: &table, Key: map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: fmt.Sprintf("key-%d", i)}}})
		require.NoError(t, err)
		require.Equal(t, &types.AttributeValueMemberS{Value: fmt.Sprintf("value-%d + & 日本語", i)}, out.Item["data"], "item %d changed", i)
	}
}
