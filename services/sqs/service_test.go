package sqs_test

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	sqssvc "github.com/neureaux/cloudmock/services/sqs"
)

// newSQSGateway builds a full gateway stack with the SQS service registered and IAM disabled.
func newSQSGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(sqssvc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// sqsReq builds a form-encoded POST request targeting the SQS service.
func sqsReq(t *testing.T, action string, extra url.Values) *http.Request {
	t.Helper()

	form := url.Values{}
	form.Set("Action", action)
	form.Set("Version", "2012-11-05")
	for k, vs := range extra {
		for _, v := range vs {
			form.Add(k, v)
		}
	}

	body := strings.NewReader(form.Encode())
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/sqs/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// mustCreateQueue is a test helper that creates a queue and returns its URL.
func mustCreateQueue(t *testing.T, handler http.Handler, name string) string {
	t.Helper()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sqsReq(t, "CreateQueue", url.Values{"QueueName": {name}}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateQueue %s: expected 200, got %d\nbody: %s", name, w.Code, w.Body.String())
	}

	var resp struct {
		Result struct {
			QueueUrl string `xml:"QueueUrl"`
		} `xml:"CreateQueueResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("CreateQueue: unmarshal response: %v\nbody: %s", err, w.Body.String())
	}
	if resp.Result.QueueUrl == "" {
		t.Fatalf("CreateQueue: QueueUrl is empty\nbody: %s", w.Body.String())
	}
	return resp.Result.QueueUrl
}

// ---- Test 1: CreateQueue + ListQueues ----

func TestSQS_CreateQueueAndListQueues(t *testing.T) {
	handler := newSQSGateway(t)

	// Create two queues.
	url1 := mustCreateQueue(t, handler, "test-queue-1")
	url2 := mustCreateQueue(t, handler, "test-queue-2")

	if !strings.Contains(url1, "test-queue-1") {
		t.Errorf("CreateQueue: expected URL to contain queue name, got %s", url1)
	}
	if !strings.Contains(url2, "test-queue-2") {
		t.Errorf("CreateQueue: expected URL to contain queue name, got %s", url2)
	}

	// ListQueues — all.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sqsReq(t, "ListQueues", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListQueues: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "test-queue-1") {
		t.Errorf("ListQueues: expected test-queue-1 in response\nbody: %s", body)
	}
	if !strings.Contains(body, "test-queue-2") {
		t.Errorf("ListQueues: expected test-queue-2 in response\nbody: %s", body)
	}

	// ListQueues with prefix filter.
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, sqsReq(t, "ListQueues", url.Values{"QueueNamePrefix": {"test-queue-1"}}))
	if wf.Code != http.StatusOK {
		t.Fatalf("ListQueues prefix: expected 200, got %d\nbody: %s", wf.Code, wf.Body.String())
	}
	bodyf := wf.Body.String()
	if !strings.Contains(bodyf, "test-queue-1") {
		t.Errorf("ListQueues prefix: expected test-queue-1 in response\nbody: %s", bodyf)
	}
	if strings.Contains(bodyf, "test-queue-2") {
		t.Errorf("ListQueues prefix: should NOT contain test-queue-2\nbody: %s", bodyf)
	}
}

// ---- Test 2: SendMessage + ReceiveMessage round-trip ----

func TestSQS_SendAndReceiveMessage(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "round-trip-queue")

	// SendMessage.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, sqsReq(t, "SendMessage", url.Values{
		"QueueUrl":    {queueURL},
		"MessageBody": {"hello world"},
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("SendMessage: expected 200, got %d\nbody: %s", ws.Code, ws.Body.String())
	}
	sendBody := ws.Body.String()
	for _, want := range []string{"MessageId", "MD5OfMessageBody", "SendMessageResponse"} {
		if !strings.Contains(sendBody, want) {
			t.Errorf("SendMessage: expected %q in response\nbody: %s", want, sendBody)
		}
	}

	// ReceiveMessage.
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsReq(t, "ReceiveMessage", url.Values{
		"QueueUrl":            {queueURL},
		"MaxNumberOfMessages": {"1"},
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("ReceiveMessage: expected 200, got %d\nbody: %s", wr.Code, wr.Body.String())
	}
	recvBody := wr.Body.String()
	for _, want := range []string{"hello world", "ReceiptHandle", "MessageId", "MD5OfBody"} {
		if !strings.Contains(recvBody, want) {
			t.Errorf("ReceiveMessage: expected %q in response\nbody: %s", want, recvBody)
		}
	}
}

// ---- Test 3: DeleteMessage — receive, delete, receive again returns empty ----

func TestSQS_DeleteMessage(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "delete-test-queue")

	// Send a message.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, sqsReq(t, "SendMessage", url.Values{
		"QueueUrl":    {queueURL},
		"MessageBody": {"to be deleted"},
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("SendMessage: %d %s", ws.Code, ws.Body.String())
	}

	// Receive it (with a long visibility timeout to keep it inflight).
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsReq(t, "ReceiveMessage", url.Values{
		"QueueUrl":          {queueURL},
		"VisibilityTimeout": {"300"},
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("ReceiveMessage: %d %s", wr.Code, wr.Body.String())
	}

	// Extract ReceiptHandle from response.
	var recvResp struct {
		Result struct {
			Messages []struct {
				ReceiptHandle string `xml:"ReceiptHandle"`
			} `xml:"Message"`
		} `xml:"ReceiveMessageResult"`
	}
	if err := xml.Unmarshal(wr.Body.Bytes(), &recvResp); err != nil {
		t.Fatalf("ReceiveMessage: unmarshal: %v", err)
	}
	if len(recvResp.Result.Messages) == 0 {
		t.Fatal("ReceiveMessage: expected 1 message, got 0")
	}
	receiptHandle := recvResp.Result.Messages[0].ReceiptHandle

	// Delete the message.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, sqsReq(t, "DeleteMessage", url.Values{
		"QueueUrl":      {queueURL},
		"ReceiptHandle": {receiptHandle},
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DeleteMessage: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	// Receive again — should be empty.
	wr2 := httptest.NewRecorder()
	handler.ServeHTTP(wr2, sqsReq(t, "ReceiveMessage", url.Values{
		"QueueUrl":            {queueURL},
		"MaxNumberOfMessages": {"10"},
	}))
	if wr2.Code != http.StatusOK {
		t.Fatalf("ReceiveMessage 2: %d %s", wr2.Code, wr2.Body.String())
	}
	var recvResp2 struct {
		Result struct {
			Messages []struct{} `xml:"Message"`
		} `xml:"ReceiveMessageResult"`
	}
	if err := xml.Unmarshal(wr2.Body.Bytes(), &recvResp2); err != nil {
		t.Fatalf("ReceiveMessage 2: unmarshal: %v", err)
	}
	if len(recvResp2.Result.Messages) != 0 {
		t.Errorf("ReceiveMessage 2: expected empty, got %d messages", len(recvResp2.Result.Messages))
	}
}

// ---- Test 4: DeleteQueue ----

func TestSQS_DeleteQueue(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "deletable-queue")

	// Verify it's in the list.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, sqsReq(t, "ListQueues", nil))
	if !strings.Contains(wl.Body.String(), "deletable-queue") {
		t.Fatal("ListQueues: queue not found before deletion")
	}

	// DeleteQueue.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, sqsReq(t, "DeleteQueue", url.Values{"QueueUrl": {queueURL}}))
	if wd.Code != http.StatusOK {
		t.Fatalf("DeleteQueue: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	// Verify it's gone.
	wl2 := httptest.NewRecorder()
	handler.ServeHTTP(wl2, sqsReq(t, "ListQueues", nil))
	if strings.Contains(wl2.Body.String(), "deletable-queue") {
		t.Error("ListQueues: queue should not appear after deletion")
	}

	// Attempt to send to deleted queue — should get error.
	we := httptest.NewRecorder()
	handler.ServeHTTP(we, sqsReq(t, "SendMessage", url.Values{
		"QueueUrl":    {queueURL},
		"MessageBody": {"ghost"},
	}))
	if we.Code == http.StatusOK {
		t.Error("SendMessage to deleted queue: expected error, got 200")
	}
}

// ---- Test 5: GetQueueAttributes ----

func TestSQS_GetQueueAttributes(t *testing.T) {
	handler := newSQSGateway(t)

	// Create queue with a custom VisibilityTimeout attribute.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, sqsReq(t, "CreateQueue", url.Values{
		"QueueName":         {"attr-queue"},
		"Attribute.1.Name":  {"VisibilityTimeout"},
		"Attribute.1.Value": {"60"},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateQueue with attrs: %d %s", wc.Code, wc.Body.String())
	}
	var createResp struct {
		Result struct {
			QueueUrl string `xml:"QueueUrl"`
		} `xml:"CreateQueueResult"`
	}
	_ = xml.Unmarshal(wc.Body.Bytes(), &createResp)
	queueURL := createResp.Result.QueueUrl

	// GetQueueAttributes — request All.
	wa := httptest.NewRecorder()
	handler.ServeHTTP(wa, sqsReq(t, "GetQueueAttributes", url.Values{
		"QueueUrl":        {queueURL},
		"AttributeName.1": {"All"},
	}))
	if wa.Code != http.StatusOK {
		t.Fatalf("GetQueueAttributes: expected 200, got %d\nbody: %s", wa.Code, wa.Body.String())
	}
	body := wa.Body.String()
	for _, want := range []string{
		"GetQueueAttributesResponse",
		"VisibilityTimeout",
		"ApproximateNumberOfMessages",
		"QueueArn",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("GetQueueAttributes: expected %q in response\nbody: %s", want, body)
		}
	}
}

// ---- Test 6: FIFO queue (name ends .fifo) ----

func TestSQS_FIFOQueue(t *testing.T) {
	handler := newSQSGateway(t)

	queueURL := mustCreateQueue(t, handler, "my-queue.fifo")

	// GetQueueAttributes to verify FifoQueue=true.
	wa := httptest.NewRecorder()
	handler.ServeHTTP(wa, sqsReq(t, "GetQueueAttributes", url.Values{
		"QueueUrl":        {queueURL},
		"AttributeName.1": {"All"},
	}))
	if wa.Code != http.StatusOK {
		t.Fatalf("GetQueueAttributes FIFO: %d %s", wa.Code, wa.Body.String())
	}
	if !strings.Contains(wa.Body.String(), "FifoQueue") {
		t.Errorf("GetQueueAttributes FIFO: expected FifoQueue attribute\nbody: %s", wa.Body.String())
	}
	if !strings.Contains(wa.Body.String(), "true") {
		t.Errorf("GetQueueAttributes FIFO: expected FifoQueue=true\nbody: %s", wa.Body.String())
	}

	// Send a message with a deduplication ID.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, sqsReq(t, "SendMessage", url.Values{
		"QueueUrl":                {queueURL},
		"MessageBody":             {"fifo message"},
		"MessageGroupId":          {"group-1"},
		"MessageDeduplicationId":  {"dedup-1"},
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("SendMessage FIFO: %d %s", ws.Code, ws.Body.String())
	}

	// Receive it.
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsReq(t, "ReceiveMessage", url.Values{
		"QueueUrl": {queueURL},
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("ReceiveMessage FIFO: %d %s", wr.Code, wr.Body.String())
	}
	if !strings.Contains(wr.Body.String(), "fifo message") {
		t.Errorf("ReceiveMessage FIFO: expected message body\nbody: %s", wr.Body.String())
	}
}

// ---- Test 7: PurgeQueue ----

func TestSQS_PurgeQueue(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "purge-queue")

	// Send several messages.
	for i := 0; i < 3; i++ {
		ws := httptest.NewRecorder()
		handler.ServeHTTP(ws, sqsReq(t, "SendMessage", url.Values{
			"QueueUrl":    {queueURL},
			"MessageBody": {"msg"},
		}))
		if ws.Code != http.StatusOK {
			t.Fatalf("SendMessage %d: %d %s", i, ws.Code, ws.Body.String())
		}
	}

	// Verify we can receive messages before purge.
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsReq(t, "ReceiveMessage", url.Values{
		"QueueUrl":            {queueURL},
		"MaxNumberOfMessages": {"10"},
		"VisibilityTimeout":   {"0"},
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("ReceiveMessage before purge: %d %s", wr.Code, wr.Body.String())
	}
	var prePurge struct {
		Result struct {
			Messages []struct{} `xml:"Message"`
		} `xml:"ReceiveMessageResult"`
	}
	_ = xml.Unmarshal(wr.Body.Bytes(), &prePurge)
	if len(prePurge.Result.Messages) == 0 {
		t.Fatal("ReceiveMessage before purge: expected at least 1 message")
	}

	// PurgeQueue.
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, sqsReq(t, "PurgeQueue", url.Values{"QueueUrl": {queueURL}}))
	if wp.Code != http.StatusOK {
		t.Fatalf("PurgeQueue: expected 200, got %d\nbody: %s", wp.Code, wp.Body.String())
	}
	if !strings.Contains(wp.Body.String(), "PurgeQueueResponse") {
		t.Errorf("PurgeQueue: expected PurgeQueueResponse\nbody: %s", wp.Body.String())
	}

	// Receive after purge — should be empty.
	wr2 := httptest.NewRecorder()
	handler.ServeHTTP(wr2, sqsReq(t, "ReceiveMessage", url.Values{
		"QueueUrl":            {queueURL},
		"MaxNumberOfMessages": {"10"},
	}))
	if wr2.Code != http.StatusOK {
		t.Fatalf("ReceiveMessage after purge: %d %s", wr2.Code, wr2.Body.String())
	}
	var postPurge struct {
		Result struct {
			Messages []struct{} `xml:"Message"`
		} `xml:"ReceiveMessageResult"`
	}
	_ = xml.Unmarshal(wr2.Body.Bytes(), &postPurge)
	if len(postPurge.Result.Messages) != 0 {
		t.Errorf("ReceiveMessage after purge: expected 0 messages, got %d", len(postPurge.Result.Messages))
	}
}

// ---- Test 8: GetQueueUrl ----

func TestSQS_GetQueueUrl(t *testing.T) {
	handler := newSQSGateway(t)
	want := mustCreateQueue(t, handler, "url-lookup-queue")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sqsReq(t, "GetQueueUrl", url.Values{"QueueName": {"url-lookup-queue"}}))
	if w.Code != http.StatusOK {
		t.Fatalf("GetQueueUrl: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Result struct {
			QueueUrl string `xml:"QueueUrl"`
		} `xml:"GetQueueUrlResult"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("GetQueueUrl: unmarshal: %v", err)
	}
	if resp.Result.QueueUrl != want {
		t.Errorf("GetQueueUrl: got %q, want %q", resp.Result.QueueUrl, want)
	}
}

// ---- Test 9: SendMessageBatch + DeleteMessageBatch ----

func TestSQS_BatchOperations(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "batch-queue")

	// SendMessageBatch with 3 entries.
	wb := httptest.NewRecorder()
	handler.ServeHTTP(wb, sqsReq(t, "SendMessageBatch", url.Values{
		"QueueUrl":                                    {queueURL},
		"SendMessageBatchRequestEntry.1.Id":           {"msg1"},
		"SendMessageBatchRequestEntry.1.MessageBody":  {"body one"},
		"SendMessageBatchRequestEntry.2.Id":           {"msg2"},
		"SendMessageBatchRequestEntry.2.MessageBody":  {"body two"},
		"SendMessageBatchRequestEntry.3.Id":           {"msg3"},
		"SendMessageBatchRequestEntry.3.MessageBody":  {"body three"},
	}))
	if wb.Code != http.StatusOK {
		t.Fatalf("SendMessageBatch: expected 200, got %d\nbody: %s", wb.Code, wb.Body.String())
	}
	batchBody := wb.Body.String()
	if !strings.Contains(batchBody, "SendMessageBatchResponse") {
		t.Errorf("SendMessageBatch: expected SendMessageBatchResponse\nbody: %s", batchBody)
	}

	// Receive all 3.
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsReq(t, "ReceiveMessage", url.Values{
		"QueueUrl":            {queueURL},
		"MaxNumberOfMessages": {"10"},
		"VisibilityTimeout":   {"300"},
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("ReceiveMessage batch: %d %s", wr.Code, wr.Body.String())
	}
	var recvResp struct {
		Result struct {
			Messages []struct {
				ReceiptHandle string `xml:"ReceiptHandle"`
			} `xml:"Message"`
		} `xml:"ReceiveMessageResult"`
	}
	if err := xml.Unmarshal(wr.Body.Bytes(), &recvResp); err != nil {
		t.Fatalf("ReceiveMessage batch: unmarshal: %v", err)
	}
	if len(recvResp.Result.Messages) != 3 {
		t.Fatalf("ReceiveMessage batch: expected 3 messages, got %d", len(recvResp.Result.Messages))
	}

	// DeleteMessageBatch.
	delForm := url.Values{"QueueUrl": {queueURL}}
	for i, m := range recvResp.Result.Messages {
		delForm.Set(strings.Join([]string{
			"DeleteMessageBatchRequestEntry",
			strings.Repeat(".", 1),
			strings.Repeat("1", 0), // placeholder
		}, ""), "")
		key := "DeleteMessageBatchRequestEntry." + string(rune('1'+i))
		delForm.Set(key+".Id", "del"+string(rune('1'+i)))
		delForm.Set(key+".ReceiptHandle", m.ReceiptHandle)
	}

	// Build the batch delete request manually to avoid the above awkward code.
	delVals := url.Values{
		"QueueUrl":                                         {queueURL},
		"DeleteMessageBatchRequestEntry.1.Id":              {"del1"},
		"DeleteMessageBatchRequestEntry.1.ReceiptHandle":   {recvResp.Result.Messages[0].ReceiptHandle},
		"DeleteMessageBatchRequestEntry.2.Id":              {"del2"},
		"DeleteMessageBatchRequestEntry.2.ReceiptHandle":   {recvResp.Result.Messages[1].ReceiptHandle},
		"DeleteMessageBatchRequestEntry.3.Id":              {"del3"},
		"DeleteMessageBatchRequestEntry.3.ReceiptHandle":   {recvResp.Result.Messages[2].ReceiptHandle},
	}
	wdel := httptest.NewRecorder()
	handler.ServeHTTP(wdel, sqsReq(t, "DeleteMessageBatch", delVals))
	if wdel.Code != http.StatusOK {
		t.Fatalf("DeleteMessageBatch: expected 200, got %d\nbody: %s", wdel.Code, wdel.Body.String())
	}
	if !strings.Contains(wdel.Body.String(), "DeleteMessageBatchResponse") {
		t.Errorf("DeleteMessageBatch: expected DeleteMessageBatchResponse\nbody: %s", wdel.Body.String())
	}
}
