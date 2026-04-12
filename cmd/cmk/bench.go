package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// benchOp describes a single benchmark operation.
type benchOp struct {
	Name        string
	Service     string // for Authorization header credential scope
	Target      string // X-Amz-Target (empty for query protocol)
	ContentType string
	Body        string
	Query       string // for query-protocol services (?Action=...)
}

var benchOps = []benchOp{
	{Name: "DynamoDB GetItem", Service: "dynamodb", Target: "DynamoDB_20120810.GetItem",
		ContentType: "application/x-amz-json-1.0",
		Body:        `{"TableName":"bench","Key":{"pk":{"S":"key-1"}}}`},
	{Name: "DynamoDB PutItem", Service: "dynamodb", Target: "DynamoDB_20120810.PutItem",
		ContentType: "application/x-amz-json-1.0",
		Body:        `{"TableName":"bench","Item":{"pk":{"S":"k"},"d":{"S":"v"}}}`},
	{Name: "SQS SendMessage", Service: "sqs", Query: "Action=SendMessage&QueueUrl=http://localhost:PORT/000000000000/bench-queue&MessageBody=hello"},
	{Name: "S3 PutObject (1KB)", Service: "s3", Body: strings.Repeat("x", 1024)},
	{Name: "S3 GetObject (1KB)", Service: "s3"},
	{Name: "SNS Publish", Service: "sns", Query: "Action=Publish&TopicArn=arn:aws:sns:us-east-1:000000000000:bench-topic&Message=hello"},
	{Name: "STS GetCallerIdentity", Service: "sts", Query: "Action=GetCallerIdentity&Version=2011-06-15"},
	{Name: "KMS Encrypt", Service: "kms", Target: "TrentService.Encrypt",
		ContentType: "application/x-amz-json-1.0",
		Body:        `{"KeyId":"alias/bench-key","Plaintext":"aGVsbG8="}`},
}

// benchResult holds the result for one operation against one target.
type benchResult struct {
	Op       string
	Target   string
	ReqSec   float64
	AvgMs    float64
	Errors   int64
	Requests int64
}

func cmdBench(args []string) {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	concurrency := fs.Int("c", 50, "concurrent connections per operation")
	duration := fs.Duration("d", 10*time.Second, "duration per operation")
	skipLS := fs.Bool("skip-localstack", false, "skip LocalStack even if available")
	skipMoto := fs.Bool("skip-moto", false, "skip Moto even if available")
	fs.Parse(args)

	fmt.Println()
	fmt.Println("CloudMock Benchmark")
	fmt.Println("===================")
	fmt.Printf("Concurrency: %d connections\n", *concurrency)
	fmt.Printf("Duration:    %s per operation\n", *duration)
	fmt.Println()

	// --- Detect targets ---
	type benchTarget struct {
		Name     string
		Endpoint string
		cleanup  func()
	}
	var targets []benchTarget

	// 1. Start CloudMock
	cmPort := findFreePort()
	cmEndpoint := fmt.Sprintf("http://localhost:%d", cmPort)
	cmProc := startCloudMock(cmPort)
	if cmProc == nil {
		fmt.Fprintf(os.Stderr, "Failed to start CloudMock\n")
		os.Exit(1)
	}
	targets = append(targets, benchTarget{
		Name:     "CloudMock",
		Endpoint: cmEndpoint,
		cleanup:  func() { cmProc.Process.Signal(os.Interrupt); cmProc.Wait() },
	})
	fmt.Printf("  CloudMock:   %s (pid %d)\n", cmEndpoint, cmProc.Process.Pid)

	// 2. Detect LocalStack
	if !*skipLS {
		if lsPort, lsProc := startLocalStack(); lsProc != nil {
			lsEndpoint := fmt.Sprintf("http://localhost:%d", lsPort)
			targets = append(targets, benchTarget{
				Name:     "LocalStack",
				Endpoint: lsEndpoint,
				cleanup:  func() { exec.Command("docker", "stop", "cmk-bench-ls").Run() },
			})
			fmt.Printf("  LocalStack:  %s (container cmk-bench-ls)\n", lsEndpoint)
			_ = lsProc
		} else {
			fmt.Println("  LocalStack:  not found (docker pull localstack/localstack to compare)")
		}
	}

	// 3. Detect Moto
	if !*skipMoto {
		if motoPort, motoProc := startMoto(); motoProc != nil {
			motoEndpoint := fmt.Sprintf("http://localhost:%d", motoPort)
			targets = append(targets, benchTarget{
				Name:     "Moto",
				Endpoint: motoEndpoint,
				cleanup:  func() { motoProc.Process.Signal(os.Interrupt); motoProc.Wait() },
			})
			fmt.Printf("  Moto:        %s (pid %d)\n", motoEndpoint, motoProc.Process.Pid)
		} else {
			fmt.Println("  Moto:        not found (pip install moto[server] to compare)")
		}
	}

	fmt.Println()

	// --- Seed data on each target ---
	for _, t := range targets {
		seedTarget(t.Endpoint)
	}

	// --- Run benchmarks ---
	var results []benchResult

	for i, op := range benchOps {
		fmt.Printf("[%d/%d] %s ", i+1, len(benchOps), op.Name)
		for _, t := range targets {
			r := runBenchOp(op, t.Endpoint, *concurrency, *duration)
			r.Target = t.Name
			results = append(results, r)
			fmt.Printf(" %s:%.0f", t.Name[:2], r.ReqSec)
		}
		fmt.Println()
	}

	// --- Cleanup ---
	for _, t := range targets {
		t.cleanup()
	}

	// --- Print results ---
	fmt.Println()
	targetNames := make([]string, len(targets))
	for i, t := range targets {
		targetNames[i] = t.Name
	}
	printMarkdownTable(results, targetNames)
}

// runBenchOp sends requests for the given operation at the given concurrency.
func runBenchOp(op benchOp, endpoint string, conns int, dur time.Duration) benchResult {
	var totalOps atomic.Int64
	var totalErrors atomic.Int64
	deadline := time.Now().Add(dur)

	port := strings.TrimPrefix(endpoint, "http://localhost:")

	var wg sync.WaitGroup
	for i := 0; i < conns; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{Timeout: 5 * time.Second}

			for time.Now().Before(deadline) {
				var req *http.Request
				var err error

				if op.Target != "" {
					// JSON protocol (X-Amz-Target)
					ct := op.ContentType
					if ct == "" {
						ct = "application/x-amz-json-1.1"
					}
					req, err = http.NewRequest("POST", endpoint+"/", strings.NewReader(op.Body))
					if err != nil {
						totalErrors.Add(1)
						continue
					}
					req.Header.Set("X-Amz-Target", op.Target)
					req.Header.Set("Content-Type", ct)
				} else if op.Query != "" {
					// Query protocol
					q := strings.ReplaceAll(op.Query, "PORT", port)
					req, err = http.NewRequest("POST", endpoint+"/?"+q, nil)
					if err != nil {
						totalErrors.Add(1)
						continue
					}
					req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				} else if op.Service == "s3" && strings.Contains(op.Name, "Put") {
					req, err = http.NewRequest("PUT", endpoint+"/bench-bucket/bench-key", strings.NewReader(op.Body))
					if err != nil {
						totalErrors.Add(1)
						continue
					}
				} else if op.Service == "s3" && strings.Contains(op.Name, "Get") {
					req, err = http.NewRequest("GET", endpoint+"/bench-bucket/bench-key", nil)
					if err != nil {
						totalErrors.Add(1)
						continue
					}
				} else {
					continue
				}

				req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=test/20260410/us-east-1/"+op.Service+"/aws4_request")

				resp, err := client.Do(req)
				if err != nil {
					totalErrors.Add(1)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				if resp.StatusCode >= 400 {
					totalErrors.Add(1)
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
	avgMs := 0.0
	if ops > 0 {
		avgMs = (float64(conns) / rps) * 1000
	}

	return benchResult{
		Op:       op.Name,
		ReqSec:   rps,
		AvgMs:    avgMs,
		Errors:   errs,
		Requests: ops,
	}
}

// printMarkdownTable prints a copy-pasteable markdown comparison table.
func printMarkdownTable(results []benchResult, targets []string) {
	// Build header.
	header := "| Operation |"
	sep := "|---|"
	for _, t := range targets {
		header += fmt.Sprintf(" %s |", t)
		sep += "---|"
	}

	// Add comparison columns if multiple targets.
	if len(targets) > 1 {
		for _, t := range targets[1:] {
			header += fmt.Sprintf(" vs %s |", t)
			sep += "---|"
		}
	}

	fmt.Println("## Benchmark Results")
	fmt.Println()
	fmt.Printf("Tested on %s with %s. Copy this table into issues or Slack.\n\n", time.Now().Format("2006-01-02"), hostInfo())
	fmt.Println(header)
	fmt.Println(sep)

	// Group results by operation.
	type opRow struct {
		name    string
		results map[string]benchResult
	}
	ops := make([]opRow, 0)
	seen := make(map[string]int)
	for _, r := range results {
		idx, ok := seen[r.Op]
		if !ok {
			idx = len(ops)
			seen[r.Op] = idx
			ops = append(ops, opRow{name: r.Op, results: make(map[string]benchResult)})
		}
		ops[idx].results[r.Target] = r
	}

	for _, op := range ops {
		row := fmt.Sprintf("| **%s** |", op.name)
		var cmRPS float64
		for _, t := range targets {
			r, ok := op.results[t]
			if !ok {
				row += " - |"
				continue
			}
			row += fmt.Sprintf(" **%s** |", formatRPS(r.ReqSec))
			if t == "CloudMock" {
				cmRPS = r.ReqSec
			}
		}
		if len(targets) > 1 && cmRPS > 0 {
			for _, t := range targets[1:] {
				r, ok := op.results[t]
				if !ok || r.ReqSec == 0 {
					row += " - |"
					continue
				}
				ratio := cmRPS / r.ReqSec
				row += fmt.Sprintf(" %.0fx |", ratio)
			}
		}
		fmt.Println(row)
	}

	// Geometric mean.
	if len(targets) > 1 {
		row := "| **Geometric mean** |"
		gmeans := make(map[string]float64)
		for _, t := range targets {
			product := 1.0
			count := 0
			for _, op := range ops {
				if r, ok := op.results[t]; ok && r.ReqSec > 0 {
					product *= r.ReqSec
					count++
				}
			}
			if count > 0 {
				gm := math.Pow(product, 1.0/float64(count))
				gmeans[t] = gm
				row += fmt.Sprintf(" **%s** |", formatRPS(gm))
			} else {
				row += " - |"
			}
		}
		if cmGM, ok := gmeans["CloudMock"]; ok {
			for _, t := range targets[1:] {
				if otherGM, ok := gmeans[t]; ok && otherGM > 0 {
					row += fmt.Sprintf(" **%.0fx** |", cmGM/otherGM)
				} else {
					row += " - |"
				}
			}
		}
		fmt.Println(row)
	}

	fmt.Println()
}

func formatRPS(rps float64) string {
	if rps >= 1000 {
		return fmt.Sprintf("%.0f", rps)
	}
	return fmt.Sprintf("%.1f", rps)
}

func hostInfo() string {
	host, _ := os.Hostname()
	return host
}

// --- Target lifecycle ---

func startCloudMock(port int) *exec.Cmd {
	// Find the cloudmock binary.
	bin := findBinary("cloudmock")
	if bin == "" {
		// Try building from source.
		bin = "go"
	}

	var cmd *exec.Cmd
	if bin == "go" {
		cmd = exec.Command("go", "run", "./cmd/gateway")
	} else {
		cmd = exec.Command(bin)
	}

	cmd.Env = append(os.Environ(),
		"CLOUDMOCK_TEST_MODE=true",
		fmt.Sprintf("CLOUDMOCK_PORT=%d", port),
	)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return nil
	}

	if !waitForHealth(fmt.Sprintf("http://localhost:%d/_cloudmock/health", port), 15*time.Second) {
		cmd.Process.Signal(os.Interrupt)
		cmd.Wait()
		return nil
	}
	return cmd
}

func startLocalStack() (int, *exec.Cmd) {
	if _, err := exec.LookPath("docker"); err != nil {
		return 0, nil
	}

	// Check if localstack image exists.
	out, err := exec.Command("docker", "image", "inspect", "localstack/localstack").CombinedOutput()
	if err != nil || !strings.Contains(string(out), "Id") {
		return 0, nil
	}

	port := findFreePort()

	// Stop any existing container with the same name.
	exec.Command("docker", "rm", "-f", "cmk-bench-ls").Run()

	cmd := exec.Command("docker", "run", "--rm", "--name", "cmk-bench-ls",
		"-p", fmt.Sprintf("%d:4566", port),
		"-e", "SERVICES=dynamodb,s3,sqs,sns,sts,kms",
		"localstack/localstack",
	)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return 0, nil
	}

	if !waitForHealth(fmt.Sprintf("http://localhost:%d/_localstack/health", port), 30*time.Second) {
		exec.Command("docker", "stop", "cmk-bench-ls").Run()
		return 0, nil
	}
	return port, cmd
}

func startMoto() (int, *exec.Cmd) {
	// Try moto_server first, then python -m moto.server.
	var cmd *exec.Cmd
	port := findFreePort()

	if _, err := exec.LookPath("moto_server"); err == nil {
		cmd = exec.Command("moto_server", "-p", fmt.Sprintf("%d", port))
	} else {
		python := findPython()
		if python == "" {
			return 0, nil
		}
		// Check if moto is installed.
		check := exec.Command(python, "-c", "import moto.server")
		if err := check.Run(); err != nil {
			return 0, nil
		}
		cmd = exec.Command(python, "-m", "moto.server", "-p", fmt.Sprintf("%d", port))
	}

	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return 0, nil
	}

	if !waitForHealth(fmt.Sprintf("http://localhost:%d/moto-api/", port), 15*time.Second) {
		cmd.Process.Signal(os.Interrupt)
		cmd.Wait()
		return 0, nil
	}
	return port, cmd
}

// seedTarget creates the required resources for benchmarks.
func seedTarget(endpoint string) {
	client := &http.Client{Timeout: 10 * time.Second}
	auth := "AWS4-HMAC-SHA256 Credential=test/20260410/us-east-1/"

	// DynamoDB: create table
	post(client, endpoint, auth+"dynamodb/aws4_request",
		"DynamoDB_20120810.CreateTable", "application/x-amz-json-1.0",
		`{"TableName":"bench","AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],"BillingMode":"PAY_PER_REQUEST"}`)

	// DynamoDB: seed item
	post(client, endpoint, auth+"dynamodb/aws4_request",
		"DynamoDB_20120810.PutItem", "application/x-amz-json-1.0",
		`{"TableName":"bench","Item":{"pk":{"S":"key-1"},"d":{"S":"value-1"}}}`)

	// SQS: create queue
	httpPost(client, endpoint+"/?Action=CreateQueue&QueueName=bench-queue", auth+"sqs/aws4_request", "")

	// S3: create bucket + seed object
	req, _ := http.NewRequest("PUT", endpoint+"/bench-bucket", nil)
	req.Header.Set("Authorization", auth+"s3/aws4_request")
	client.Do(req)

	req, _ = http.NewRequest("PUT", endpoint+"/bench-bucket/bench-key", strings.NewReader(strings.Repeat("x", 1024)))
	req.Header.Set("Authorization", auth+"s3/aws4_request")
	client.Do(req)

	// SNS: create topic
	httpPost(client, endpoint+"/?Action=CreateTopic&Name=bench-topic", auth+"sns/aws4_request", "")

	// KMS: create key
	post(client, endpoint, auth+"kms/aws4_request",
		"TrentService.CreateKey", "application/x-amz-json-1.0",
		`{"Description":"bench"}`)
}

func post(client *http.Client, endpoint, auth, target, ct, body string) {
	req, _ := http.NewRequest("POST", endpoint+"/", strings.NewReader(body))
	req.Header.Set("Authorization", auth)
	req.Header.Set("X-Amz-Target", target)
	req.Header.Set("Content-Type", ct)
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func httpPost(client *http.Client, url, auth, body string) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, _ := http.NewRequest("POST", url, bodyReader)
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

// --- Utilities ---

func findFreePort() int {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 14566 // fallback
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func waitForHealth(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return true
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func findBinary(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return ""
}

func findPython() string {
	for _, name := range []string{"python3", "python"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

// Ensure bytes import is used (for potential future use).
var _ = bytes.NewReader
