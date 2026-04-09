package sqs_test

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	sqssvc "github.com/Viridian-Inc/cloudmock/services/sqs"
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
		"QueueUrl":               {queueURL},
		"MessageBody":            {"fifo message"},
		"MessageGroupId":         {"group-1"},
		"MessageDeduplicationId": {"dedup-1"},
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
		"QueueUrl":                                   {queueURL},
		"SendMessageBatchRequestEntry.1.Id":          {"msg1"},
		"SendMessageBatchRequestEntry.1.MessageBody": {"body one"},
		"SendMessageBatchRequestEntry.2.Id":          {"msg2"},
		"SendMessageBatchRequestEntry.2.MessageBody": {"body two"},
		"SendMessageBatchRequestEntry.3.Id":          {"msg3"},
		"SendMessageBatchRequestEntry.3.MessageBody": {"body three"},
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
		"QueueUrl":                                       {queueURL},
		"DeleteMessageBatchRequestEntry.1.Id":            {"del1"},
		"DeleteMessageBatchRequestEntry.1.ReceiptHandle": {recvResp.Result.Messages[0].ReceiptHandle},
		"DeleteMessageBatchRequestEntry.2.Id":            {"del2"},
		"DeleteMessageBatchRequestEntry.2.ReceiptHandle": {recvResp.Result.Messages[1].ReceiptHandle},
		"DeleteMessageBatchRequestEntry.3.Id":            {"del3"},
		"DeleteMessageBatchRequestEntry.3.ReceiptHandle": {recvResp.Result.Messages[2].ReceiptHandle},
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

// ---- Test 10: FIFO deduplication — duplicate MessageDeduplicationId is suppressed ----

func TestSQS_FIFODeduplication(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "dedup-test.fifo")

	// Send the same dedup ID twice.
	for i := 0; i < 2; i++ {
		ws := httptest.NewRecorder()
		handler.ServeHTTP(ws, sqsReq(t, "SendMessage", url.Values{
			"QueueUrl":               {queueURL},
			"MessageBody":            {"dedup body"},
			"MessageGroupId":         {"group-a"},
			"MessageDeduplicationId": {"dup-id-1"},
		}))
		if ws.Code != http.StatusOK {
			t.Fatalf("SendMessage FIFO dedup iter %d: %d %s", i, ws.Code, ws.Body.String())
		}
	}

	// Receive — should get exactly 1 message since the second was deduplicated.
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsReq(t, "ReceiveMessage", url.Values{
		"QueueUrl":            {queueURL},
		"MaxNumberOfMessages": {"10"},
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("ReceiveMessage: %d %s", wr.Code, wr.Body.String())
	}
	var recvResp struct {
		Result struct {
			Messages []struct{} `xml:"Message"`
		} `xml:"ReceiveMessageResult"`
	}
	if err := xml.Unmarshal(wr.Body.Bytes(), &recvResp); err != nil {
		t.Fatalf("ReceiveMessage: unmarshal: %v", err)
	}
	if len(recvResp.Result.Messages) != 1 {
		t.Errorf("Expected 1 message after dedup, got %d", len(recvResp.Result.Messages))
	}
}

// ---- Test 11: FIFO message group ordering ----

func TestSQS_FIFOMessageGroups(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "groups-test.fifo")

	// Send messages to two different groups.
	for i, gid := range []string{"group-A", "group-B", "group-A"} {
		ws := httptest.NewRecorder()
		handler.ServeHTTP(ws, sqsReq(t, "SendMessage", url.Values{
			"QueueUrl":               {queueURL},
			"MessageBody":            {gid + "-msg-" + string(rune('0'+i))},
			"MessageGroupId":         {gid},
			"MessageDeduplicationId": {"dedup-" + string(rune('0'+i))},
		}))
		if ws.Code != http.StatusOK {
			t.Fatalf("SendMessage group %s: %d %s", gid, ws.Code, ws.Body.String())
		}
	}

	// Receive all — should get all 3 messages.
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsReq(t, "ReceiveMessage", url.Values{
		"QueueUrl":            {queueURL},
		"MaxNumberOfMessages": {"10"},
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("ReceiveMessage: %d %s", wr.Code, wr.Body.String())
	}
	var recvResp struct {
		Result struct {
			Messages []struct {
				Body string `xml:"Body"`
			} `xml:"Message"`
		} `xml:"ReceiveMessageResult"`
	}
	if err := xml.Unmarshal(wr.Body.Bytes(), &recvResp); err != nil {
		t.Fatalf("ReceiveMessage: unmarshal: %v", err)
	}
	if len(recvResp.Result.Messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(recvResp.Result.Messages))
	}
}

// ---- Test 12: Dead-letter queue redrive policy ----

func TestSQS_DeadLetterQueueRedrivePolicy(t *testing.T) {
	handler := newSQSGateway(t)

	// Create a DLQ.
	dlqURL := mustCreateQueue(t, handler, "my-dlq")

	// Get the DLQ ARN.
	wa := httptest.NewRecorder()
	handler.ServeHTTP(wa, sqsReq(t, "GetQueueAttributes", url.Values{
		"QueueUrl":        {dlqURL},
		"AttributeName.1": {"QueueArn"},
	}))
	if wa.Code != http.StatusOK {
		t.Fatalf("GetQueueAttributes DLQ: %d %s", wa.Code, wa.Body.String())
	}
	dlqBody := wa.Body.String()
	if !strings.Contains(dlqBody, "arn:aws:sqs:") {
		t.Fatalf("GetQueueAttributes DLQ: expected QueueArn in response\nbody: %s", dlqBody)
	}

	// Create a source queue with RedrivePolicy.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, sqsReq(t, "CreateQueue", url.Values{
		"QueueName":         {"source-queue"},
		"Attribute.1.Name":  {"RedrivePolicy"},
		"Attribute.1.Value": {`{"deadLetterTargetArn":"arn:aws:sqs:us-east-1:000000000000:my-dlq","maxReceiveCount":"3"}`},
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("CreateQueue with RedrivePolicy: %d %s", wc.Code, wc.Body.String())
	}
	var createResp struct {
		Result struct {
			QueueUrl string `xml:"QueueUrl"`
		} `xml:"CreateQueueResult"`
	}
	_ = xml.Unmarshal(wc.Body.Bytes(), &createResp)
	srcURL := createResp.Result.QueueUrl

	// Verify the RedrivePolicy attribute is stored.
	wa2 := httptest.NewRecorder()
	handler.ServeHTTP(wa2, sqsReq(t, "GetQueueAttributes", url.Values{
		"QueueUrl":        {srcURL},
		"AttributeName.1": {"All"},
	}))
	if wa2.Code != http.StatusOK {
		t.Fatalf("GetQueueAttributes source: %d %s", wa2.Code, wa2.Body.String())
	}
	if !strings.Contains(wa2.Body.String(), "RedrivePolicy") {
		t.Errorf("GetQueueAttributes: expected RedrivePolicy attribute\nbody: %s", wa2.Body.String())
	}
	if !strings.Contains(wa2.Body.String(), "deadLetterTargetArn") {
		t.Errorf("GetQueueAttributes: expected deadLetterTargetArn in RedrivePolicy\nbody: %s", wa2.Body.String())
	}
}

// ---- Test 13: ChangeMessageVisibility ----

func TestSQS_ChangeMessageVisibility(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "vis-change-queue")

	// Send a message.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, sqsReq(t, "SendMessage", url.Values{
		"QueueUrl":    {queueURL},
		"MessageBody": {"visibility test"},
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("SendMessage: %d %s", ws.Code, ws.Body.String())
	}

	// Receive it with a very long visibility timeout.
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsReq(t, "ReceiveMessage", url.Values{
		"QueueUrl":          {queueURL},
		"VisibilityTimeout": {"300"},
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("ReceiveMessage: %d %s", wr.Code, wr.Body.String())
	}
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

	// Change visibility to 0 — makes it immediately visible again.
	wv := httptest.NewRecorder()
	handler.ServeHTTP(wv, sqsReq(t, "ChangeMessageVisibility", url.Values{
		"QueueUrl":          {queueURL},
		"ReceiptHandle":     {receiptHandle},
		"VisibilityTimeout": {"0"},
	}))
	if wv.Code != http.StatusOK {
		t.Fatalf("ChangeMessageVisibility: expected 200, got %d\nbody: %s", wv.Code, wv.Body.String())
	}
	if !strings.Contains(wv.Body.String(), "ChangeMessageVisibilityResponse") {
		t.Errorf("Expected ChangeMessageVisibilityResponse\nbody: %s", wv.Body.String())
	}

	// Receive again — should get the same message back since visibility was set to 0.
	wr2 := httptest.NewRecorder()
	handler.ServeHTTP(wr2, sqsReq(t, "ReceiveMessage", url.Values{
		"QueueUrl":            {queueURL},
		"MaxNumberOfMessages": {"1"},
	}))
	if wr2.Code != http.StatusOK {
		t.Fatalf("ReceiveMessage 2: %d %s", wr2.Code, wr2.Body.String())
	}
	if !strings.Contains(wr2.Body.String(), "visibility test") {
		t.Errorf("Expected message to be visible again after ChangeMessageVisibility(0)\nbody: %s", wr2.Body.String())
	}
}

// ---- Test 14: ChangeMessageVisibility — invalid receipt handle ----

func TestSQS_ChangeMessageVisibility_InvalidReceiptHandle(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "vis-invalid-queue")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sqsReq(t, "ChangeMessageVisibility", url.Values{
		"QueueUrl":          {queueURL},
		"ReceiptHandle":     {"not-a-valid-receipt-handle"},
		"VisibilityTimeout": {"30"},
	}))
	if w.Code == http.StatusOK {
		t.Error("ChangeMessageVisibility with invalid handle: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "ReceiptHandleIsInvalid") {
		t.Errorf("Expected ReceiptHandleIsInvalid error\nbody: %s", w.Body.String())
	}
}

// ---- Test 15: SetQueueAttributes ----

func TestSQS_SetQueueAttributes(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "set-attrs-queue")

	// Set VisibilityTimeout to 90.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, sqsReq(t, "SetQueueAttributes", url.Values{
		"QueueUrl":          {queueURL},
		"Attribute.1.Name":  {"VisibilityTimeout"},
		"Attribute.1.Value": {"90"},
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("SetQueueAttributes: expected 200, got %d\nbody: %s", ws.Code, ws.Body.String())
	}
	if !strings.Contains(ws.Body.String(), "SetQueueAttributesResponse") {
		t.Errorf("Expected SetQueueAttributesResponse\nbody: %s", ws.Body.String())
	}

	// Verify via GetQueueAttributes.
	wa := httptest.NewRecorder()
	handler.ServeHTTP(wa, sqsReq(t, "GetQueueAttributes", url.Values{
		"QueueUrl":        {queueURL},
		"AttributeName.1": {"VisibilityTimeout"},
	}))
	if wa.Code != http.StatusOK {
		t.Fatalf("GetQueueAttributes: %d %s", wa.Code, wa.Body.String())
	}
	if !strings.Contains(wa.Body.String(), "90") {
		t.Errorf("GetQueueAttributes: expected VisibilityTimeout=90\nbody: %s", wa.Body.String())
	}
}

// ---- Test 16: SendMessageBatch — empty batch error ----

func TestSQS_SendMessageBatch_EmptyBatch(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "empty-batch-queue")

	// Send an empty batch — no entries.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sqsReq(t, "SendMessageBatch", url.Values{
		"QueueUrl": {queueURL},
	}))
	if w.Code == http.StatusOK {
		t.Error("SendMessageBatch with empty batch: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "EmptyBatchRequest") {
		t.Errorf("Expected EmptyBatchRequest error\nbody: %s", w.Body.String())
	}
}

// ---- Test 17: SendMessageBatch — too many entries error ----

func TestSQS_SendMessageBatch_TooManyEntries(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "too-many-batch-queue")

	// Build 11 entries (max is 10).
	vals := url.Values{"QueueUrl": {queueURL}}
	for i := 1; i <= 11; i++ {
		prefix := fmt.Sprintf("SendMessageBatchRequestEntry.%d", i)
		vals.Set(prefix+".Id", fmt.Sprintf("msg%d", i))
		vals.Set(prefix+".MessageBody", fmt.Sprintf("body %d", i))
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sqsReq(t, "SendMessageBatch", vals))
	if w.Code == http.StatusOK {
		t.Error("SendMessageBatch with 11 entries: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "TooManyEntriesInBatchRequest") {
		t.Errorf("Expected TooManyEntriesInBatchRequest error\nbody: %s", w.Body.String())
	}
}

// ---- Test 18: DeleteMessageBatch — empty batch error ----

func TestSQS_DeleteMessageBatch_EmptyBatch(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "del-empty-batch-queue")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sqsReq(t, "DeleteMessageBatch", url.Values{
		"QueueUrl": {queueURL},
	}))
	if w.Code == http.StatusOK {
		t.Error("DeleteMessageBatch with empty batch: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "EmptyBatchRequest") {
		t.Errorf("Expected EmptyBatchRequest error\nbody: %s", w.Body.String())
	}
}

// ---- Test 19: QueueDoesNotExist errors ----

func TestSQS_QueueDoesNotExist(t *testing.T) {
	handler := newSQSGateway(t)

	// SendMessage to a nonexistent queue.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, sqsReq(t, "SendMessage", url.Values{
		"QueueUrl":    {"http://sqs.us-east-1.localhost:4566/000000000000/no-such-queue"},
		"MessageBody": {"test"},
	}))
	if ws.Code == http.StatusOK {
		t.Error("SendMessage to nonexistent queue: expected error, got 200")
	}
	if !strings.Contains(ws.Body.String(), "NonExistentQueue") {
		t.Errorf("Expected NonExistentQueue error\nbody: %s", ws.Body.String())
	}

	// ReceiveMessage from a nonexistent queue.
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsReq(t, "ReceiveMessage", url.Values{
		"QueueUrl": {"http://sqs.us-east-1.localhost:4566/000000000000/no-such-queue"},
	}))
	if wr.Code == http.StatusOK {
		t.Error("ReceiveMessage from nonexistent queue: expected error, got 200")
	}
	if !strings.Contains(wr.Body.String(), "NonExistentQueue") {
		t.Errorf("Expected NonExistentQueue error\nbody: %s", wr.Body.String())
	}

	// DeleteQueue for a nonexistent queue.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, sqsReq(t, "DeleteQueue", url.Values{
		"QueueUrl": {"http://sqs.us-east-1.localhost:4566/000000000000/no-such-queue"},
	}))
	if wd.Code == http.StatusOK {
		t.Error("DeleteQueue nonexistent queue: expected error, got 200")
	}
	if !strings.Contains(wd.Body.String(), "NonExistentQueue") {
		t.Errorf("Expected NonExistentQueue error\nbody: %s", wd.Body.String())
	}

	// PurgeQueue for a nonexistent queue.
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, sqsReq(t, "PurgeQueue", url.Values{
		"QueueUrl": {"http://sqs.us-east-1.localhost:4566/000000000000/no-such-queue"},
	}))
	if wp.Code == http.StatusOK {
		t.Error("PurgeQueue nonexistent queue: expected error, got 200")
	}
	if !strings.Contains(wp.Body.String(), "NonExistentQueue") {
		t.Errorf("Expected NonExistentQueue error\nbody: %s", wp.Body.String())
	}

	// GetQueueUrl for a nonexistent queue.
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, sqsReq(t, "GetQueueUrl", url.Values{
		"QueueName": {"no-such-queue"},
	}))
	if wg.Code == http.StatusOK {
		t.Error("GetQueueUrl nonexistent queue: expected error, got 200")
	}
	if !strings.Contains(wg.Body.String(), "NonExistentQueue") {
		t.Errorf("Expected NonExistentQueue error\nbody: %s", wg.Body.String())
	}

	// GetQueueAttributes for a nonexistent queue.
	wga := httptest.NewRecorder()
	handler.ServeHTTP(wga, sqsReq(t, "GetQueueAttributes", url.Values{
		"QueueUrl":        {"http://sqs.us-east-1.localhost:4566/000000000000/no-such-queue"},
		"AttributeName.1": {"All"},
	}))
	if wga.Code == http.StatusOK {
		t.Error("GetQueueAttributes nonexistent queue: expected error, got 200")
	}
	if !strings.Contains(wga.Body.String(), "NonExistentQueue") {
		t.Errorf("Expected NonExistentQueue error\nbody: %s", wga.Body.String())
	}

	// ChangeMessageVisibility for a nonexistent queue.
	wcv := httptest.NewRecorder()
	handler.ServeHTTP(wcv, sqsReq(t, "ChangeMessageVisibility", url.Values{
		"QueueUrl":          {"http://sqs.us-east-1.localhost:4566/000000000000/no-such-queue"},
		"ReceiptHandle":     {"fake"},
		"VisibilityTimeout": {"30"},
	}))
	if wcv.Code == http.StatusOK {
		t.Error("ChangeMessageVisibility nonexistent queue: expected error, got 200")
	}
	if !strings.Contains(wcv.Body.String(), "NonExistentQueue") {
		t.Errorf("Expected NonExistentQueue error\nbody: %s", wcv.Body.String())
	}
}

// ---- Test 20: ReceiptHandleIsInvalid error ----

func TestSQS_ReceiptHandleIsInvalid(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "invalid-rh-queue")

	// DeleteMessage with an invalid receipt handle.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, sqsReq(t, "DeleteMessage", url.Values{
		"QueueUrl":      {queueURL},
		"ReceiptHandle": {"totally-bogus-receipt-handle"},
	}))
	if wd.Code == http.StatusOK {
		t.Error("DeleteMessage with invalid receipt handle: expected error, got 200")
	}
	if !strings.Contains(wd.Body.String(), "ReceiptHandleIsInvalid") {
		t.Errorf("Expected ReceiptHandleIsInvalid error\nbody: %s", wd.Body.String())
	}
}

// ---- Test 21: DeleteMessageBatch — partial failure with invalid receipt handles ----

func TestSQS_DeleteMessageBatch_PartialFailure(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "partial-del-batch")

	// Send a message and receive it to get a valid receipt handle.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, sqsReq(t, "SendMessage", url.Values{
		"QueueUrl":    {queueURL},
		"MessageBody": {"real message"},
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("SendMessage: %d %s", ws.Code, ws.Body.String())
	}

	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsReq(t, "ReceiveMessage", url.Values{
		"QueueUrl":          {queueURL},
		"VisibilityTimeout": {"300"},
	}))
	var recvResp struct {
		Result struct {
			Messages []struct {
				ReceiptHandle string `xml:"ReceiptHandle"`
			} `xml:"Message"`
		} `xml:"ReceiveMessageResult"`
	}
	_ = xml.Unmarshal(wr.Body.Bytes(), &recvResp)
	if len(recvResp.Result.Messages) == 0 {
		t.Fatal("ReceiveMessage: expected 1 message")
	}
	validRH := recvResp.Result.Messages[0].ReceiptHandle

	// Delete batch: one valid, one invalid.
	wdel := httptest.NewRecorder()
	handler.ServeHTTP(wdel, sqsReq(t, "DeleteMessageBatch", url.Values{
		"QueueUrl":                                       {queueURL},
		"DeleteMessageBatchRequestEntry.1.Id":            {"ok"},
		"DeleteMessageBatchRequestEntry.1.ReceiptHandle": {validRH},
		"DeleteMessageBatchRequestEntry.2.Id":            {"bad"},
		"DeleteMessageBatchRequestEntry.2.ReceiptHandle": {"invalid-handle"},
	}))
	if wdel.Code != http.StatusOK {
		t.Fatalf("DeleteMessageBatch partial: expected 200, got %d\nbody: %s", wdel.Code, wdel.Body.String())
	}
	body := wdel.Body.String()

	// Should have one successful and one failed.
	if !strings.Contains(body, "DeleteMessageBatchResultEntry") {
		t.Errorf("Expected DeleteMessageBatchResultEntry in response\nbody: %s", body)
	}
	if !strings.Contains(body, "BatchResultErrorEntry") {
		t.Errorf("Expected BatchResultErrorEntry for invalid handle\nbody: %s", body)
	}
	if !strings.Contains(body, "ReceiptHandleIsInvalid") {
		t.Errorf("Expected ReceiptHandleIsInvalid in error entry\nbody: %s", body)
	}
}

// ---- Test 22: Long polling — WaitTimeSeconds returns immediately when messages exist ----

func TestSQS_LongPolling(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "longpoll-queue")

	// Send a message first.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, sqsReq(t, "SendMessage", url.Values{
		"QueueUrl":    {queueURL},
		"MessageBody": {"long poll message"},
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("SendMessage: %d %s", ws.Code, ws.Body.String())
	}

	// Receive with WaitTimeSeconds (should return immediately since messages are available).
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsReq(t, "ReceiveMessage", url.Values{
		"QueueUrl":        {queueURL},
		"WaitTimeSeconds": {"20"},
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("ReceiveMessage with WaitTimeSeconds: %d %s", wr.Code, wr.Body.String())
	}
	if !strings.Contains(wr.Body.String(), "long poll message") {
		t.Errorf("Expected message body in long poll response\nbody: %s", wr.Body.String())
	}
}

// ---- Test 23: Long polling — WaitTimeSeconds returns empty when no messages ----

func TestSQS_LongPolling_Empty(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "longpoll-empty-queue")

	// Receive with WaitTimeSeconds from empty queue (should return empty, not block forever).
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsReq(t, "ReceiveMessage", url.Values{
		"QueueUrl":        {queueURL},
		"WaitTimeSeconds": {"1"},
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("ReceiveMessage with WaitTimeSeconds on empty queue: %d %s", wr.Code, wr.Body.String())
	}
	var recvResp struct {
		Result struct {
			Messages []struct{} `xml:"Message"`
		} `xml:"ReceiveMessageResult"`
	}
	_ = xml.Unmarshal(wr.Body.Bytes(), &recvResp)
	if len(recvResp.Result.Messages) != 0 {
		t.Errorf("Expected 0 messages from empty long-poll, got %d", len(recvResp.Result.Messages))
	}
}

// ---- Test 24: SendMessageBatch with FIFO deduplication ----

func TestSQS_SendMessageBatch_FIFODedup(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "batch-fifo.fifo")

	// Send a batch with two entries having the same dedup ID.
	wb := httptest.NewRecorder()
	handler.ServeHTTP(wb, sqsReq(t, "SendMessageBatch", url.Values{
		"QueueUrl":                                              {queueURL},
		"SendMessageBatchRequestEntry.1.Id":                     {"entry1"},
		"SendMessageBatchRequestEntry.1.MessageBody":            {"body one"},
		"SendMessageBatchRequestEntry.1.MessageGroupId":         {"grp"},
		"SendMessageBatchRequestEntry.1.MessageDeduplicationId": {"same-dedup"},
		"SendMessageBatchRequestEntry.2.Id":                     {"entry2"},
		"SendMessageBatchRequestEntry.2.MessageBody":            {"body two different"},
		"SendMessageBatchRequestEntry.2.MessageGroupId":         {"grp"},
		"SendMessageBatchRequestEntry.2.MessageDeduplicationId": {"same-dedup"},
	}))
	if wb.Code != http.StatusOK {
		t.Fatalf("SendMessageBatch FIFO dedup: %d %s", wb.Code, wb.Body.String())
	}
	if !strings.Contains(wb.Body.String(), "SendMessageBatchResponse") {
		t.Errorf("Expected SendMessageBatchResponse\nbody: %s", wb.Body.String())
	}

	// Receive — should get only 1 message (second was deduplicated).
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsReq(t, "ReceiveMessage", url.Values{
		"QueueUrl":            {queueURL},
		"MaxNumberOfMessages": {"10"},
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("ReceiveMessage: %d %s", wr.Code, wr.Body.String())
	}
	var recvResp struct {
		Result struct {
			Messages []struct{} `xml:"Message"`
		} `xml:"ReceiveMessageResult"`
	}
	_ = xml.Unmarshal(wr.Body.Bytes(), &recvResp)
	if len(recvResp.Result.Messages) != 1 {
		t.Errorf("Expected 1 message after FIFO batch dedup, got %d", len(recvResp.Result.Messages))
	}
}

// ---- Test 25: InvalidAction error ----

func TestSQS_InvalidAction(t *testing.T) {
	handler := newSQSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sqsReq(t, "NonExistentAction", nil))
	if w.Code == http.StatusOK {
		t.Error("InvalidAction: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "InvalidAction") {
		t.Errorf("Expected InvalidAction error\nbody: %s", w.Body.String())
	}
}

// ---- Test 26: Purge queue on nonexistent queue ----

func TestSQS_PurgeQueue_NonExistent(t *testing.T) {
	handler := newSQSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sqsReq(t, "PurgeQueue", url.Values{
		"QueueUrl": {"http://sqs.us-east-1.localhost:4566/000000000000/ghost-queue"},
	}))
	if w.Code == http.StatusOK {
		t.Error("PurgeQueue nonexistent: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "NonExistentQueue") {
		t.Errorf("Expected NonExistentQueue error\nbody: %s", w.Body.String())
	}
}

// ---- Test 27: CreateQueue is idempotent ----

func TestSQS_CreateQueueIdempotent(t *testing.T) {
	handler := newSQSGateway(t)

	url1 := mustCreateQueue(t, handler, "idempotent-queue")
	url2 := mustCreateQueue(t, handler, "idempotent-queue")

	if url1 != url2 {
		t.Errorf("CreateQueue should be idempotent: url1=%s url2=%s", url1, url2)
	}

	// ListQueues should show only one queue with that name.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, sqsReq(t, "ListQueues", url.Values{
		"QueueNamePrefix": {"idempotent-queue"},
	}))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListQueues: %d %s", wl.Code, wl.Body.String())
	}
	body := wl.Body.String()
	count := strings.Count(body, "idempotent-queue")
	if count != 1 {
		t.Errorf("Expected exactly 1 queue URL with name, found %d occurrences\nbody: %s", count, body)
	}
}

// ---- Test 28: SendMessage with message attributes ----

func TestSQS_SendMessageWithAttributes(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "msg-attrs-queue")

	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, sqsReq(t, "SendMessage", url.Values{
		"QueueUrl":                             {queueURL},
		"MessageBody":                          {"body with attrs"},
		"MessageAttribute.1.Name":              {"CustomAttr"},
		"MessageAttribute.1.Value.DataType":    {"String"},
		"MessageAttribute.1.Value.StringValue": {"custom-value"},
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("SendMessage with attrs: %d %s", ws.Code, ws.Body.String())
	}
	if !strings.Contains(ws.Body.String(), "MessageId") {
		t.Errorf("Expected MessageId in response\nbody: %s", ws.Body.String())
	}
	if !strings.Contains(ws.Body.String(), "MD5OfMessageBody") {
		t.Errorf("Expected MD5OfMessageBody in response\nbody: %s", ws.Body.String())
	}
}

// ---- Test 29: ReceiveMessage respects MaxNumberOfMessages ----

func TestSQS_ReceiveMessage_MaxCount(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "max-count-queue")

	// Send 5 messages.
	for i := 0; i < 5; i++ {
		ws := httptest.NewRecorder()
		handler.ServeHTTP(ws, sqsReq(t, "SendMessage", url.Values{
			"QueueUrl":    {queueURL},
			"MessageBody": {fmt.Sprintf("msg-%d", i)},
		}))
		if ws.Code != http.StatusOK {
			t.Fatalf("SendMessage %d: %d", i, ws.Code)
		}
	}

	// Receive with max 3.
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsReq(t, "ReceiveMessage", url.Values{
		"QueueUrl":            {queueURL},
		"MaxNumberOfMessages": {"3"},
		"VisibilityTimeout":   {"0"},
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("ReceiveMessage: %d %s", wr.Code, wr.Body.String())
	}
	var recvResp struct {
		Result struct {
			Messages []struct{} `xml:"Message"`
		} `xml:"ReceiveMessageResult"`
	}
	if err := xml.Unmarshal(wr.Body.Bytes(), &recvResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(recvResp.Result.Messages) != 3 {
		t.Errorf("Expected exactly 3 messages, got %d", len(recvResp.Result.Messages))
	}
}

// ---- Test 30: SetQueueAttributes on nonexistent queue ----

func TestSQS_SetQueueAttributes_NonExistent(t *testing.T) {
	handler := newSQSGateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sqsReq(t, "SetQueueAttributes", url.Values{
		"QueueUrl":          {"http://sqs.us-east-1.localhost:4566/000000000000/no-such-queue"},
		"Attribute.1.Name":  {"VisibilityTimeout"},
		"Attribute.1.Value": {"60"},
	}))
	if w.Code == http.StatusOK {
		t.Error("SetQueueAttributes nonexistent: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "NonExistentQueue") {
		t.Errorf("Expected NonExistentQueue error\nbody: %s", w.Body.String())
	}
}

// ---- Test 31: DeleteMessageBatch — too many entries ----

func TestSQS_DeleteMessageBatch_TooManyEntries(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustCreateQueue(t, handler, "del-too-many-queue")

	// Build 11 delete entries.
	vals := url.Values{"QueueUrl": {queueURL}}
	for i := 1; i <= 11; i++ {
		prefix := fmt.Sprintf("DeleteMessageBatchRequestEntry.%d", i)
		vals.Set(prefix+".Id", fmt.Sprintf("del%d", i))
		vals.Set(prefix+".ReceiptHandle", fmt.Sprintf("fake-rh-%d", i))
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sqsReq(t, "DeleteMessageBatch", vals))
	if w.Code == http.StatusOK {
		t.Error("DeleteMessageBatch with 11 entries: expected error, got 200")
	}
	if !strings.Contains(w.Body.String(), "TooManyEntriesInBatchRequest") {
		t.Errorf("Expected TooManyEntriesInBatchRequest error\nbody: %s", w.Body.String())
	}
}
