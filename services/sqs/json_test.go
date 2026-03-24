package sqs_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sqsJSONReq builds a JSON-protocol POST request targeting the SQS service.
func sqsJSONReq(t *testing.T, target string, body any) *http.Request {
	t.Helper()

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal JSON body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", target)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/sqs/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// mustJSONCreateQueue creates a queue via JSON protocol and returns its URL.
func mustJSONCreateQueue(t *testing.T, handler http.Handler, name string) string {
	t.Helper()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sqsJSONReq(t, "AmazonSQS.CreateQueue", map[string]any{
		"QueueName": name,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("JSON CreateQueue %s: expected 200, got %d\nbody: %s", name, w.Code, w.Body.String())
	}

	var resp struct {
		QueueUrl string `json:"QueueUrl"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("JSON CreateQueue: unmarshal: %v\nbody: %s", err, w.Body.String())
	}
	if resp.QueueUrl == "" {
		t.Fatalf("JSON CreateQueue: QueueUrl is empty\nbody: %s", w.Body.String())
	}
	return resp.QueueUrl
}

// ---- Test: CreateQueue + ListQueues (JSON) ----

func TestSQSJSON_CreateQueueAndListQueues(t *testing.T) {
	handler := newSQSGateway(t)

	url1 := mustJSONCreateQueue(t, handler, "json-queue-1")
	url2 := mustJSONCreateQueue(t, handler, "json-queue-2")

	if !strings.Contains(url1, "json-queue-1") {
		t.Errorf("CreateQueue JSON: expected URL to contain queue name, got %s", url1)
	}
	if !strings.Contains(url2, "json-queue-2") {
		t.Errorf("CreateQueue JSON: expected URL to contain queue name, got %s", url2)
	}

	// ListQueues
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sqsJSONReq(t, "AmazonSQS.ListQueues", map[string]any{}))
	if w.Code != http.StatusOK {
		t.Fatalf("JSON ListQueues: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var listResp struct {
		QueueUrls []string `json:"QueueUrls"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("JSON ListQueues: unmarshal: %v", err)
	}
	if len(listResp.QueueUrls) < 2 {
		t.Fatalf("JSON ListQueues: expected at least 2 URLs, got %d", len(listResp.QueueUrls))
	}

	// ListQueues with prefix
	wf := httptest.NewRecorder()
	handler.ServeHTTP(wf, sqsJSONReq(t, "AmazonSQS.ListQueues", map[string]any{
		"QueueNamePrefix": "json-queue-1",
	}))
	if wf.Code != http.StatusOK {
		t.Fatalf("JSON ListQueues prefix: expected 200, got %d", wf.Code)
	}
	var filteredResp struct {
		QueueUrls []string `json:"QueueUrls"`
	}
	_ = json.Unmarshal(wf.Body.Bytes(), &filteredResp)
	if len(filteredResp.QueueUrls) != 1 {
		t.Errorf("JSON ListQueues prefix: expected 1, got %d", len(filteredResp.QueueUrls))
	}
}

// ---- Test: SendMessage + ReceiveMessage (JSON) ----

func TestSQSJSON_SendAndReceiveMessage(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustJSONCreateQueue(t, handler, "json-roundtrip")

	// SendMessage
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, sqsJSONReq(t, "AmazonSQS.SendMessage", map[string]any{
		"QueueUrl":    queueURL,
		"MessageBody": "hello json",
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("JSON SendMessage: expected 200, got %d\nbody: %s", ws.Code, ws.Body.String())
	}

	var sendResp struct {
		MessageId        string `json:"MessageId"`
		MD5OfMessageBody string `json:"MD5OfMessageBody"`
	}
	if err := json.Unmarshal(ws.Body.Bytes(), &sendResp); err != nil {
		t.Fatalf("JSON SendMessage: unmarshal: %v", err)
	}
	if sendResp.MessageId == "" {
		t.Error("JSON SendMessage: MessageId is empty")
	}
	if sendResp.MD5OfMessageBody == "" {
		t.Error("JSON SendMessage: MD5OfMessageBody is empty")
	}

	// ReceiveMessage
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsJSONReq(t, "AmazonSQS.ReceiveMessage", map[string]any{
		"QueueUrl":            queueURL,
		"MaxNumberOfMessages": 1,
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("JSON ReceiveMessage: expected 200, got %d\nbody: %s", wr.Code, wr.Body.String())
	}

	var recvResp struct {
		Messages []struct {
			MessageId     string `json:"MessageId"`
			ReceiptHandle string `json:"ReceiptHandle"`
			Body          string `json:"Body"`
			MD5OfBody     string `json:"MD5OfBody"`
		} `json:"Messages"`
	}
	if err := json.Unmarshal(wr.Body.Bytes(), &recvResp); err != nil {
		t.Fatalf("JSON ReceiveMessage: unmarshal: %v", err)
	}
	if len(recvResp.Messages) == 0 {
		t.Fatal("JSON ReceiveMessage: expected 1 message, got 0")
	}
	if recvResp.Messages[0].Body != "hello json" {
		t.Errorf("JSON ReceiveMessage: expected body 'hello json', got %q", recvResp.Messages[0].Body)
	}
}

// ---- Test: DeleteMessage (JSON) ----

func TestSQSJSON_DeleteMessage(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustJSONCreateQueue(t, handler, "json-delete-msg")

	// Send a message.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, sqsJSONReq(t, "AmazonSQS.SendMessage", map[string]any{
		"QueueUrl":    queueURL,
		"MessageBody": "delete me",
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("SendMessage: %d %s", ws.Code, ws.Body.String())
	}

	// Receive it.
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsJSONReq(t, "AmazonSQS.ReceiveMessage", map[string]any{
		"QueueUrl":          queueURL,
		"VisibilityTimeout": 300,
	}))
	if wr.Code != http.StatusOK {
		t.Fatalf("ReceiveMessage: %d %s", wr.Code, wr.Body.String())
	}

	var recvResp struct {
		Messages []struct {
			ReceiptHandle string `json:"ReceiptHandle"`
		} `json:"Messages"`
	}
	_ = json.Unmarshal(wr.Body.Bytes(), &recvResp)
	if len(recvResp.Messages) == 0 {
		t.Fatal("ReceiveMessage: expected 1 message, got 0")
	}
	receiptHandle := recvResp.Messages[0].ReceiptHandle

	// Delete the message.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, sqsJSONReq(t, "AmazonSQS.DeleteMessage", map[string]any{
		"QueueUrl":      queueURL,
		"ReceiptHandle": receiptHandle,
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("JSON DeleteMessage: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	// Receive again — should be empty.
	wr2 := httptest.NewRecorder()
	handler.ServeHTTP(wr2, sqsJSONReq(t, "AmazonSQS.ReceiveMessage", map[string]any{
		"QueueUrl":            queueURL,
		"MaxNumberOfMessages": 10,
	}))
	var recvResp2 struct {
		Messages []struct{} `json:"Messages"`
	}
	_ = json.Unmarshal(wr2.Body.Bytes(), &recvResp2)
	if len(recvResp2.Messages) != 0 {
		t.Errorf("ReceiveMessage after delete: expected 0 messages, got %d", len(recvResp2.Messages))
	}
}

// ---- Test: DeleteQueue (JSON) ----

func TestSQSJSON_DeleteQueue(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustJSONCreateQueue(t, handler, "json-deletable-queue")

	// Delete it.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, sqsJSONReq(t, "AmazonSQS.DeleteQueue", map[string]any{
		"QueueUrl": queueURL,
	}))
	if wd.Code != http.StatusOK {
		t.Fatalf("JSON DeleteQueue: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	// Verify it's gone from list.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, sqsJSONReq(t, "AmazonSQS.ListQueues", map[string]any{}))
	body := wl.Body.String()
	if strings.Contains(body, "json-deletable-queue") {
		t.Error("ListQueues: queue should not appear after deletion")
	}
}

// ---- Test: GetQueueUrl (JSON) ----

func TestSQSJSON_GetQueueUrl(t *testing.T) {
	handler := newSQSGateway(t)
	want := mustJSONCreateQueue(t, handler, "json-url-lookup")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sqsJSONReq(t, "AmazonSQS.GetQueueUrl", map[string]any{
		"QueueName": "json-url-lookup",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("JSON GetQueueUrl: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var resp struct {
		QueueUrl string `json:"QueueUrl"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.QueueUrl != want {
		t.Errorf("JSON GetQueueUrl: got %q, want %q", resp.QueueUrl, want)
	}
}

// ---- Test: GetQueueAttributes + SetQueueAttributes (JSON) ----

func TestSQSJSON_QueueAttributes(t *testing.T) {
	handler := newSQSGateway(t)

	// Create queue with attributes via JSON.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, sqsJSONReq(t, "AmazonSQS.CreateQueue", map[string]any{
		"QueueName":  "json-attr-queue",
		"Attributes": map[string]string{"VisibilityTimeout": "60"},
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateQueue: %d %s", w.Code, w.Body.String())
	}
	var createResp struct {
		QueueUrl string `json:"QueueUrl"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &createResp)
	queueURL := createResp.QueueUrl

	// GetQueueAttributes — All.
	wa := httptest.NewRecorder()
	handler.ServeHTTP(wa, sqsJSONReq(t, "AmazonSQS.GetQueueAttributes", map[string]any{
		"QueueUrl":       queueURL,
		"AttributeNames": []string{"All"},
	}))
	if wa.Code != http.StatusOK {
		t.Fatalf("JSON GetQueueAttributes: expected 200, got %d\nbody: %s", wa.Code, wa.Body.String())
	}

	var attrResp struct {
		Attributes map[string]string `json:"Attributes"`
	}
	_ = json.Unmarshal(wa.Body.Bytes(), &attrResp)
	if attrResp.Attributes["VisibilityTimeout"] != "60" {
		t.Errorf("GetQueueAttributes: VisibilityTimeout expected 60, got %s", attrResp.Attributes["VisibilityTimeout"])
	}
	if _, hasArn := attrResp.Attributes["QueueArn"]; !hasArn {
		t.Error("GetQueueAttributes: expected QueueArn attribute")
	}

	// SetQueueAttributes.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, sqsJSONReq(t, "AmazonSQS.SetQueueAttributes", map[string]any{
		"QueueUrl":   queueURL,
		"Attributes": map[string]string{"VisibilityTimeout": "120"},
	}))
	if ws.Code != http.StatusOK {
		t.Fatalf("JSON SetQueueAttributes: expected 200, got %d", ws.Code)
	}

	// Verify change.
	wa2 := httptest.NewRecorder()
	handler.ServeHTTP(wa2, sqsJSONReq(t, "AmazonSQS.GetQueueAttributes", map[string]any{
		"QueueUrl":       queueURL,
		"AttributeNames": []string{"VisibilityTimeout"},
	}))
	var attrResp2 struct {
		Attributes map[string]string `json:"Attributes"`
	}
	_ = json.Unmarshal(wa2.Body.Bytes(), &attrResp2)
	if attrResp2.Attributes["VisibilityTimeout"] != "120" {
		t.Errorf("SetQueueAttributes: expected 120, got %s", attrResp2.Attributes["VisibilityTimeout"])
	}
}

// ---- Test: PurgeQueue (JSON) ----

func TestSQSJSON_PurgeQueue(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustJSONCreateQueue(t, handler, "json-purge-queue")

	// Send messages.
	for i := 0; i < 3; i++ {
		ws := httptest.NewRecorder()
		handler.ServeHTTP(ws, sqsJSONReq(t, "AmazonSQS.SendMessage", map[string]any{
			"QueueUrl":    queueURL,
			"MessageBody": "msg",
		}))
	}

	// Purge.
	wp := httptest.NewRecorder()
	handler.ServeHTTP(wp, sqsJSONReq(t, "AmazonSQS.PurgeQueue", map[string]any{
		"QueueUrl": queueURL,
	}))
	if wp.Code != http.StatusOK {
		t.Fatalf("JSON PurgeQueue: expected 200, got %d", wp.Code)
	}

	// Receive — should be empty.
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsJSONReq(t, "AmazonSQS.ReceiveMessage", map[string]any{
		"QueueUrl":            queueURL,
		"MaxNumberOfMessages": 10,
	}))
	var recvResp struct {
		Messages []struct{} `json:"Messages"`
	}
	_ = json.Unmarshal(wr.Body.Bytes(), &recvResp)
	if len(recvResp.Messages) != 0 {
		t.Errorf("ReceiveMessage after purge: expected 0, got %d", len(recvResp.Messages))
	}
}

// ---- Test: SendMessageBatch + DeleteMessageBatch (JSON) ----

func TestSQSJSON_BatchOperations(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustJSONCreateQueue(t, handler, "json-batch-queue")

	// SendMessageBatch.
	wb := httptest.NewRecorder()
	handler.ServeHTTP(wb, sqsJSONReq(t, "AmazonSQS.SendMessageBatch", map[string]any{
		"QueueUrl": queueURL,
		"Entries": []map[string]any{
			{"Id": "msg1", "MessageBody": "body one"},
			{"Id": "msg2", "MessageBody": "body two"},
			{"Id": "msg3", "MessageBody": "body three"},
		},
	}))
	if wb.Code != http.StatusOK {
		t.Fatalf("JSON SendMessageBatch: expected 200, got %d\nbody: %s", wb.Code, wb.Body.String())
	}

	var batchResp struct {
		Successful []struct {
			Id        string `json:"Id"`
			MessageId string `json:"MessageId"`
		} `json:"Successful"`
	}
	_ = json.Unmarshal(wb.Body.Bytes(), &batchResp)
	if len(batchResp.Successful) != 3 {
		t.Fatalf("SendMessageBatch: expected 3 successful, got %d", len(batchResp.Successful))
	}

	// Receive all 3.
	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsJSONReq(t, "AmazonSQS.ReceiveMessage", map[string]any{
		"QueueUrl":            queueURL,
		"MaxNumberOfMessages": 10,
		"VisibilityTimeout":   300,
	}))
	var recvResp struct {
		Messages []struct {
			ReceiptHandle string `json:"ReceiptHandle"`
		} `json:"Messages"`
	}
	_ = json.Unmarshal(wr.Body.Bytes(), &recvResp)
	if len(recvResp.Messages) != 3 {
		t.Fatalf("ReceiveMessage batch: expected 3, got %d", len(recvResp.Messages))
	}

	// DeleteMessageBatch.
	entries := make([]map[string]any, 0, 3)
	for i, m := range recvResp.Messages {
		entries = append(entries, map[string]any{
			"Id":            strings.Repeat("d", 1) + string(rune('1'+i)),
			"ReceiptHandle": m.ReceiptHandle,
		})
	}

	wdel := httptest.NewRecorder()
	handler.ServeHTTP(wdel, sqsJSONReq(t, "AmazonSQS.DeleteMessageBatch", map[string]any{
		"QueueUrl": queueURL,
		"Entries":  entries,
	}))
	if wdel.Code != http.StatusOK {
		t.Fatalf("JSON DeleteMessageBatch: expected 200, got %d\nbody: %s", wdel.Code, wdel.Body.String())
	}

	var delResp struct {
		Successful []struct {
			Id string `json:"Id"`
		} `json:"Successful"`
	}
	_ = json.Unmarshal(wdel.Body.Bytes(), &delResp)
	if len(delResp.Successful) != 3 {
		t.Errorf("DeleteMessageBatch: expected 3 successful, got %d", len(delResp.Successful))
	}
}

// ---- Test: ChangeMessageVisibility (JSON) ----

func TestSQSJSON_ChangeMessageVisibility(t *testing.T) {
	handler := newSQSGateway(t)
	queueURL := mustJSONCreateQueue(t, handler, "json-vis-queue")

	// Send and receive.
	ws := httptest.NewRecorder()
	handler.ServeHTTP(ws, sqsJSONReq(t, "AmazonSQS.SendMessage", map[string]any{
		"QueueUrl":    queueURL,
		"MessageBody": "visibility test",
	}))

	wr := httptest.NewRecorder()
	handler.ServeHTTP(wr, sqsJSONReq(t, "AmazonSQS.ReceiveMessage", map[string]any{
		"QueueUrl":          queueURL,
		"VisibilityTimeout": 300,
	}))
	var recvResp struct {
		Messages []struct {
			ReceiptHandle string `json:"ReceiptHandle"`
		} `json:"Messages"`
	}
	_ = json.Unmarshal(wr.Body.Bytes(), &recvResp)
	if len(recvResp.Messages) == 0 {
		t.Fatal("ReceiveMessage: expected 1 message")
	}

	// ChangeMessageVisibility.
	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, sqsJSONReq(t, "AmazonSQS.ChangeMessageVisibility", map[string]any{
		"QueueUrl":          queueURL,
		"ReceiptHandle":     recvResp.Messages[0].ReceiptHandle,
		"VisibilityTimeout": 0,
	}))
	if wc.Code != http.StatusOK {
		t.Fatalf("JSON ChangeMessageVisibility: expected 200, got %d\nbody: %s", wc.Code, wc.Body.String())
	}
}

// ---- Test: XML protocol still works (regression) ----

func TestSQSJSON_XMLProtocolStillWorks(t *testing.T) {
	handler := newSQSGateway(t)

	// Use the existing XML helper from service_test.go.
	queueURL := mustCreateQueue(t, handler, "xml-still-works")
	if !strings.Contains(queueURL, "xml-still-works") {
		t.Errorf("XML CreateQueue still works: %s", queueURL)
	}
}
