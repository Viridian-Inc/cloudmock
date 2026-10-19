package gateway_test

import (
	"bytes"
	"context"
	"errors"
	"hash/crc32"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go/logging"

	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	ddbservice "github.com/Viridian-Inc/cloudmock/services/dynamodb"
)

// Observe the bytes as the SDK consumes them, retaining the original body's
// Close method so the real SDK checksum and connection cleanup still run.
type checksumObservedBody struct {
	io.ReadCloser
	bytes *bytes.Buffer
}

func (b *checksumObservedBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.bytes.Write(p[:n])
	return n, err
}

type checksumExchange struct {
	action string
	status int
	header http.Header
	body   bytes.Buffer
}

type checksumTransport struct {
	base      http.RoundTripper
	exchanges []*checksumExchange
}

func (tr *checksumTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := tr.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	exchange := &checksumExchange{
		action: req.Header.Get("X-Amz-Target"),
		status: resp.StatusCode,
		header: resp.Header.Clone(),
	}
	tr.exchanges = append(tr.exchanges, exchange)
	resp.Body = &checksumObservedBody{ReadCloser: resp.Body, bytes: &exchange.body}
	return resp, nil
}

func newChecksumEndpoint(t *testing.T, mode string, registry *routing.Registry) string {
	t.Helper()
	if mode != "fast_test_mode" {
		var handler http.Handler
		if mode == "normal" {
			handler = newTestGatewayWithRegistry(t, "none", registry)
		} else {
			handler = gateway.TestModeHandler(testModeIdentity(), "us-east-1", "123456789012", registry)
		}
		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)
		return server.URL
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := gateway.FastTestModeServer(testModeIdentity(), "us-east-1", "123456789012", registry)
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() {
		if err := server.Shutdown(); err != nil {
			t.Errorf("shutdown fast gateway: %v", err)
		}
		if err := <-done; err != nil {
			t.Errorf("serve fast gateway: %v", err)
		}
	})
	return "http://" + listener.Addr().String()
}

func TestDynamoDBResponseChecksums(t *testing.T) {
	for _, mode := range []string{"normal", "net_http_test_mode", "fast_test_mode"} {
		t.Run(mode, func(t *testing.T) {
			svc := ddbservice.New("123456789012", "us-east-1")
			t.Cleanup(svc.Close)
			registry := routing.NewRegistry()
			registry.Register(svc)
			endpoint := newChecksumEndpoint(t, mode, registry)
			baseTransport := http.DefaultTransport.(*http.Transport).Clone()
			t.Cleanup(baseTransport.CloseIdleConnections)
			transport := &checksumTransport{base: baseTransport}
			var sdkLogs bytes.Buffer
			client := dynamodb.New(dynamodb.Options{
				Region:           "us-east-1",
				BaseEndpoint:     aws.String(endpoint),
				Credentials:      credentials.NewStaticCredentialsProvider("test", "test", ""),
				HTTPClient:       &http.Client{Transport: transport},
				Logger:           logging.NewStandardLogger(&sdkLogs),
				RetryMaxAttempts: 1,
			})
			ctx := context.Background()
			table := aws.String("checksum-responses")
			_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
				TableName: table,
				KeySchema: []types.KeySchemaElement{{AttributeName: aws.String("id"), KeyType: types.KeyTypeHash}},
				AttributeDefinitions: []types.AttributeDefinition{{
					AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS,
				}},
				BillingMode: types.BillingModePayPerRequest,
			})
			if err != nil {
				t.Fatalf("CreateTable: %v", err)
			}
			key := map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: "key"}}
			for _, value := range []string{"plain ASCII", "caf\u00e9 \u96ea \U0001f331\n\"quoted\""} {
				_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
					TableName: table,
					Item: map[string]types.AttributeValue{
						"id": key["id"], "data": &types.AttributeValueMemberS{Value: value},
					},
				})
				if err != nil {
					t.Fatalf("PutItem(%q): %v", value, err)
				}
				out, err := client.GetItem(ctx, &dynamodb.GetItemInput{TableName: table, Key: key})
				if err != nil {
					t.Fatalf("GetItem(%q): %v", value, err)
				}
				got, ok := out.Item["data"].(*types.AttributeValueMemberS)
				if !ok || got.Value != value {
					t.Fatalf("GetItem data = %#v, want %q", out.Item["data"], value)
				}
			}
			if _, err = client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: table, Key: key}); err != nil {
				t.Fatalf("DeleteItem: %v", err)
			}
			out, err := client.GetItem(ctx, &dynamodb.GetItemInput{TableName: table, Key: key})
			if err != nil || len(out.Item) != 0 {
				t.Fatalf("GetItem after DeleteItem: output=%#v, error=%v", out, err)
			}
			if _, err = client.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: table}); err != nil {
				t.Fatalf("DeleteTable: %v", err)
			}
			_, err = client.GetItem(ctx, &dynamodb.GetItemInput{TableName: table, Key: key})
			var missing *types.ResourceNotFoundException
			if !errors.As(err, &missing) {
				t.Fatalf("GetItem after DeleteTable: expected ResourceNotFoundException, got %v", err)
			}

			// Eight successful responses cover both the shared {} response and
			// dynamically serialized table/item responses, followed by one error.
			if len(transport.exchanges) != 9 {
				t.Fatalf("observed %d HTTP responses, want 9", len(transport.exchanges))
			}
			for i, exchange := range transport.exchanges[:8] {
				if exchange.status != http.StatusOK {
					t.Errorf("response %d %s: status=%d, want 200", i, exchange.action, exchange.status)
				}
				want := strconv.FormatUint(uint64(crc32.ChecksumIEEE(exchange.body.Bytes())), 10)
				if got := exchange.header.Get("X-Amz-Crc32"); got != want {
					t.Errorf("response %d %s: X-Amz-Crc32=%q, want %q for wire body %q", i, exchange.action, got, want, exchange.body.String())
				}
			}
			if logs := sdkLogs.String(); logs != "" {
				t.Errorf("successful SDK calls emitted warnings:\n%s", logs)
			}
		})
	}
}
