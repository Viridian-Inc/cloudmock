// guntest sends raw HTTP/1.1 requests over TCP or Unix sockets.
// Measures true server throughput without Go HTTP client overhead.
//
// Usage:
//
//	guntest -addr localhost:4566 -c 200 -d 15s              # TCP
//	guntest -unix /tmp/cloudmock.sock -c 200 -d 15s         # Unix socket
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var getItemReq = "POST / HTTP/1.1\r\nHost: localhost\r\nAuthorization: AWS4-HMAC-SHA256 Credential=test/20260408/us-east-1/dynamodb/aws4_request\r\nX-Amz-Target: DynamoDB_20120810.GetItem\r\nContent-Type: application/x-amz-json-1.0\r\nContent-Length: 48\r\nConnection: keep-alive\r\n\r\n" +
	`{"TableName":"gun","Key":{"pk":{"S":"key-50"}}}`

var putItemReq = "POST / HTTP/1.1\r\nHost: localhost\r\nAuthorization: AWS4-HMAC-SHA256 Credential=test/20260408/us-east-1/dynamodb/aws4_request\r\nX-Amz-Target: DynamoDB_20120810.PutItem\r\nContent-Type: application/x-amz-json-1.0\r\nContent-Length: 64\r\nConnection: keep-alive\r\n\r\n" +
	`{"TableName":"gun","Item":{"pk":{"S":"k"},"d":{"S":"v"}}}`

func sqsSendReq(addr string) string {
	body := "Action=SendMessage&QueueUrl=http://" + addr + "/123456789012/gun-queue&MessageBody=hello"
	return "POST / HTTP/1.1\r\nHost: localhost\r\nAuthorization: AWS4-HMAC-SHA256 Credential=test/20260408/us-east-1/sqs/aws4_request\r\nContent-Type: application/x-www-form-urlencoded\r\nContent-Length: " + strconv.Itoa(len(body)) + "\r\nConnection: keep-alive\r\n\r\n" + body
}

func main() {
	addr := flag.String("addr", "localhost:4566", "TCP address")
	unixPath := flag.String("unix", "", "Unix socket path (overrides -addr)")
	conns := flag.Int("c", 200, "connections")
	duration := flag.Duration("d", 15*time.Second, "duration")
	op := flag.String("op", "GetItem", "GetItem, PutItem, or SQSSend")
	flag.Parse()

	var reqBytes []byte
	switch *op {
	case "GetItem":
		reqBytes = []byte(getItemReq)
	case "PutItem":
		reqBytes = []byte(putItemReq)
	case "SQSSend":
		reqBytes = []byte(sqsSendReq(*addr))
	default:
		fmt.Fprintf(os.Stderr, "unknown op: %s\n", *op)
		os.Exit(1)
	}

	var totalOps atomic.Int64
	var totalErrors atomic.Int64
	deadline := time.Now().Add(*duration)

	dial := func() (net.Conn, error) {
		if *unixPath != "" {
			return net.Dial("unix", *unixPath)
		}
		return net.Dial("tcp", *addr)
	}

	var wg sync.WaitGroup
	for i := 0; i < *conns; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := dial()
			if err != nil {
				totalErrors.Add(1)
				return
			}
			defer conn.Close()

			reader := bufio.NewReaderSize(conn, 32768)

			for time.Now().Before(deadline) {
				_, err := conn.Write(reqBytes)
				if err != nil {
					totalErrors.Add(1)
					return
				}

				// Read HTTP response
				contentLen := -1
				chunked := false

				// Status line
				statusLine, err := reader.ReadString('\n')
				if err != nil {
					totalErrors.Add(1)
					return
				}
				_ = statusLine

				// Headers
				for {
					hdr, err := reader.ReadString('\n')
					if err != nil {
						totalErrors.Add(1)
						return
					}
					if hdr == "\r\n" || hdr == "\n" {
						break
					}
					lower := strings.ToLower(hdr)
					if strings.HasPrefix(lower, "content-length:") {
						val := strings.TrimSpace(hdr[15:])
						contentLen, _ = strconv.Atoi(val)
					}
					if strings.Contains(lower, "transfer-encoding: chunked") {
						chunked = true
					}
				}

				// Body
				if contentLen > 0 {
					_, err = io.CopyN(io.Discard, reader, int64(contentLen))
					if err != nil {
						totalErrors.Add(1)
						return
					}
				} else if chunked {
					// Read chunked encoding
					for {
						sizeLine, err := reader.ReadString('\n')
						if err != nil {
							totalErrors.Add(1)
							return
						}
						size, _ := strconv.ParseInt(strings.TrimSpace(sizeLine), 16, 64)
						if size == 0 {
							reader.ReadString('\n') // trailing CRLF
							break
						}
						io.CopyN(io.Discard, reader, size+2) // chunk data + CRLF
					}
				}

				totalOps.Add(1)
			}
		}()
	}

	start := time.Now()
	wg.Wait()
	elapsed := time.Since(start)

	ops := totalOps.Load()
	errs := totalErrors.Load()
	rps := float64(ops) / elapsed.Seconds()

	mode := "TCP " + *addr
	if *unixPath != "" {
		mode = "Unix " + *unixPath
	}

	fmt.Printf("  Mode:         %s\n", mode)
	fmt.Printf("  Connections:  %d\n", *conns)
	fmt.Printf("  Duration:     %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("  Requests:     %d\n", ops)
	fmt.Printf("  Errors:       %d\n", errs)
	fmt.Printf("  Requests/sec: %.0f\n", rps)
	fmt.Printf("  Avg latency:  %.3fms\n", float64(*conns)/rps*1000)
}
