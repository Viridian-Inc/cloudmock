package globalaccelerator_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/globalaccelerator"
)

func newGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(svc.New(cfg.AccountID, cfg.Region))
	return gateway.New(cfg, reg)
}

func doCall(t *testing.T, h http.Handler, action string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var data []byte
	if body == nil {
		data = []byte("{}")
	} else {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "GlobalAccelerator_V20180706."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/globalaccelerator/aws4_request, SignedHeaders=host, Signature=abc123")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("decode: %v\nbody: %s", err, w.Body.String())
	}
}

func mustOK(t *testing.T, w *httptest.ResponseRecorder, label string) {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("%s: want 200, got %d: %s", label, w.Code, w.Body.String())
	}
}

// ── Accelerator lifecycle ────────────────────────────────────────────────────

func TestAcceleratorLifecycle(t *testing.T) {
	h := newGateway(t)

	// Create
	w := doCall(t, h, "CreateAccelerator", map[string]any{
		"Name":          "my-accelerator",
		"IpAddressType": "IPV4",
		"Enabled":       true,
		"Tags": []map[string]any{
			{"Key": "env", "Value": "dev"},
		},
	})
	mustOK(t, w, "CreateAccelerator")
	var created struct {
		Accelerator struct {
			AcceleratorArn string `json:"AcceleratorArn"`
			Name           string `json:"Name"`
			Status         string `json:"Status"`
		} `json:"Accelerator"`
	}
	decode(t, w, &created)
	if created.Accelerator.AcceleratorArn == "" {
		t.Fatalf("expected AcceleratorArn in response, got %s", w.Body.String())
	}
	if created.Accelerator.Name != "my-accelerator" {
		t.Fatalf("unexpected Name: %s", created.Accelerator.Name)
	}
	arn := created.Accelerator.AcceleratorArn

	// Describe
	w = doCall(t, h, "DescribeAccelerator", map[string]any{"AcceleratorArn": arn})
	mustOK(t, w, "DescribeAccelerator")

	// List
	w = doCall(t, h, "ListAccelerators", nil)
	mustOK(t, w, "ListAccelerators")
	var listed struct {
		Accelerators []struct {
			AcceleratorArn string `json:"AcceleratorArn"`
		} `json:"Accelerators"`
	}
	decode(t, w, &listed)
	if len(listed.Accelerators) != 1 || listed.Accelerators[0].AcceleratorArn != arn {
		t.Fatalf("unexpected listed accelerators: %+v", listed)
	}

	// Update
	w = doCall(t, h, "UpdateAccelerator", map[string]any{
		"AcceleratorArn": arn,
		"Name":           "renamed-accelerator",
		"Enabled":        false,
	})
	mustOK(t, w, "UpdateAccelerator")
	var updated struct {
		Accelerator struct {
			Name    string `json:"Name"`
			Enabled bool   `json:"Enabled"`
		} `json:"Accelerator"`
	}
	decode(t, w, &updated)
	if updated.Accelerator.Name != "renamed-accelerator" || updated.Accelerator.Enabled {
		t.Fatalf("update didn't take effect: %+v", updated)
	}

	// Attributes
	w = doCall(t, h, "DescribeAcceleratorAttributes", map[string]any{"AcceleratorArn": arn})
	mustOK(t, w, "DescribeAcceleratorAttributes")
	w = doCall(t, h, "UpdateAcceleratorAttributes", map[string]any{
		"AcceleratorArn":   arn,
		"FlowLogsEnabled":  true,
		"FlowLogsS3Bucket": "my-bucket",
		"FlowLogsS3Prefix": "logs/",
	})
	mustOK(t, w, "UpdateAcceleratorAttributes")

	// Delete
	w = doCall(t, h, "DeleteAccelerator", map[string]any{"AcceleratorArn": arn})
	mustOK(t, w, "DeleteAccelerator")
	if w := doCall(t, h, "DescribeAccelerator", map[string]any{"AcceleratorArn": arn}); w.Code != http.StatusNotFound {
		t.Fatalf("DescribeAccelerator after delete: want 404, got %d", w.Code)
	}
}

func TestCreateAcceleratorValidation(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "CreateAccelerator", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("CreateAccelerator with no Name: want 400, got %d", w.Code)
	}
}

// ── Custom Routing Accelerator lifecycle ─────────────────────────────────────

func TestCustomRoutingAcceleratorLifecycle(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "CreateCustomRoutingAccelerator", map[string]any{"Name": "cr-accel"})
	mustOK(t, w, "CreateCustomRoutingAccelerator")
	var created struct {
		Accelerator struct {
			AcceleratorArn string `json:"AcceleratorArn"`
		} `json:"Accelerator"`
	}
	decode(t, w, &created)
	arn := created.Accelerator.AcceleratorArn
	if arn == "" {
		t.Fatalf("missing AcceleratorArn: %s", w.Body.String())
	}

	mustOK(t, doCall(t, h, "DescribeCustomRoutingAccelerator", map[string]any{"AcceleratorArn": arn}), "DescribeCustomRoutingAccelerator")

	w = doCall(t, h, "ListCustomRoutingAccelerators", nil)
	mustOK(t, w, "ListCustomRoutingAccelerators")
	var listed struct {
		Accelerators []struct {
			AcceleratorArn string `json:"AcceleratorArn"`
		} `json:"Accelerators"`
	}
	decode(t, w, &listed)
	if len(listed.Accelerators) != 1 {
		t.Fatalf("expected 1 custom routing accelerator, got %d", len(listed.Accelerators))
	}

	mustOK(t, doCall(t, h, "UpdateCustomRoutingAccelerator", map[string]any{
		"AcceleratorArn": arn,
		"Name":           "cr-accel-2",
	}), "UpdateCustomRoutingAccelerator")

	mustOK(t, doCall(t, h, "DescribeCustomRoutingAcceleratorAttributes", map[string]any{"AcceleratorArn": arn}), "DescribeCustomRoutingAcceleratorAttributes")
	mustOK(t, doCall(t, h, "UpdateCustomRoutingAcceleratorAttributes", map[string]any{
		"AcceleratorArn":   arn,
		"FlowLogsEnabled":  true,
		"FlowLogsS3Bucket": "bucket",
	}), "UpdateCustomRoutingAcceleratorAttributes")

	mustOK(t, doCall(t, h, "DeleteCustomRoutingAccelerator", map[string]any{"AcceleratorArn": arn}), "DeleteCustomRoutingAccelerator")
}

func TestCreateCustomRoutingAcceleratorValidation(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "CreateCustomRoutingAccelerator", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

// ── Listener lifecycle ───────────────────────────────────────────────────────

func TestListenerLifecycle(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "CreateAccelerator", map[string]any{"Name": "a1"})
	mustOK(t, w, "CreateAccelerator")
	var ac struct {
		Accelerator struct {
			AcceleratorArn string `json:"AcceleratorArn"`
		} `json:"Accelerator"`
	}
	decode(t, w, &ac)
	acceleratorArn := ac.Accelerator.AcceleratorArn

	// Create listener
	w = doCall(t, h, "CreateListener", map[string]any{
		"AcceleratorArn": acceleratorArn,
		"Protocol":       "TCP",
		"PortRanges": []map[string]any{
			{"FromPort": 80, "ToPort": 80},
		},
	})
	mustOK(t, w, "CreateListener")
	var ln struct {
		Listener struct {
			ListenerArn string `json:"ListenerArn"`
		} `json:"Listener"`
	}
	decode(t, w, &ln)
	listenerArn := ln.Listener.ListenerArn
	if listenerArn == "" {
		t.Fatalf("missing ListenerArn: %s", w.Body.String())
	}

	// Describe
	mustOK(t, doCall(t, h, "DescribeListener", map[string]any{"ListenerArn": listenerArn}), "DescribeListener")

	// List
	w = doCall(t, h, "ListListeners", map[string]any{"AcceleratorArn": acceleratorArn})
	mustOK(t, w, "ListListeners")
	var listed struct {
		Listeners []struct {
			ListenerArn string `json:"ListenerArn"`
		} `json:"Listeners"`
	}
	decode(t, w, &listed)
	if len(listed.Listeners) != 1 {
		t.Fatalf("expected 1 listener, got %d", len(listed.Listeners))
	}

	// Update
	mustOK(t, doCall(t, h, "UpdateListener", map[string]any{
		"ListenerArn":    listenerArn,
		"ClientAffinity": "SOURCE_IP",
	}), "UpdateListener")

	// Delete
	mustOK(t, doCall(t, h, "DeleteListener", map[string]any{"ListenerArn": listenerArn}), "DeleteListener")
	if w := doCall(t, h, "DescribeListener", map[string]any{"ListenerArn": listenerArn}); w.Code != http.StatusNotFound {
		t.Fatalf("DescribeListener after delete: want 404, got %d", w.Code)
	}
}

func TestCreateListenerValidation(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "CreateListener", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("CreateListener with no fields: want 400, got %d", w.Code)
	}
}

// ── Custom Routing Listener lifecycle ────────────────────────────────────────

func TestCustomRoutingListenerLifecycle(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "CreateCustomRoutingAccelerator", map[string]any{"Name": "cra"})
	mustOK(t, w, "CreateCustomRoutingAccelerator")
	var ac struct {
		Accelerator struct {
			AcceleratorArn string `json:"AcceleratorArn"`
		} `json:"Accelerator"`
	}
	decode(t, w, &ac)
	acceleratorArn := ac.Accelerator.AcceleratorArn

	w = doCall(t, h, "CreateCustomRoutingListener", map[string]any{
		"AcceleratorArn": acceleratorArn,
		"PortRanges": []map[string]any{
			{"FromPort": 1000, "ToPort": 2000},
		},
	})
	mustOK(t, w, "CreateCustomRoutingListener")
	var ln struct {
		Listener struct {
			ListenerArn string `json:"ListenerArn"`
		} `json:"Listener"`
	}
	decode(t, w, &ln)
	listenerArn := ln.Listener.ListenerArn

	mustOK(t, doCall(t, h, "DescribeCustomRoutingListener", map[string]any{"ListenerArn": listenerArn}), "DescribeCustomRoutingListener")

	w = doCall(t, h, "ListCustomRoutingListeners", map[string]any{"AcceleratorArn": acceleratorArn})
	mustOK(t, w, "ListCustomRoutingListeners")

	mustOK(t, doCall(t, h, "UpdateCustomRoutingListener", map[string]any{
		"ListenerArn": listenerArn,
		"PortRanges": []map[string]any{
			{"FromPort": 1000, "ToPort": 3000},
		},
	}), "UpdateCustomRoutingListener")

	mustOK(t, doCall(t, h, "DeleteCustomRoutingListener", map[string]any{"ListenerArn": listenerArn}), "DeleteCustomRoutingListener")
}

func TestCreateCustomRoutingListenerValidation(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "CreateCustomRoutingListener", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

// ── Endpoint Group lifecycle ─────────────────────────────────────────────────

func TestEndpointGroupLifecycle(t *testing.T) {
	h := newGateway(t)

	// Set up accelerator + listener
	w := doCall(t, h, "CreateAccelerator", map[string]any{"Name": "a1"})
	mustOK(t, w, "CreateAccelerator")
	var ac struct {
		Accelerator struct {
			AcceleratorArn string `json:"AcceleratorArn"`
		} `json:"Accelerator"`
	}
	decode(t, w, &ac)

	w = doCall(t, h, "CreateListener", map[string]any{
		"AcceleratorArn": ac.Accelerator.AcceleratorArn,
		"Protocol":       "TCP",
		"PortRanges":     []map[string]any{{"FromPort": 80, "ToPort": 80}},
	})
	mustOK(t, w, "CreateListener")
	var ln struct {
		Listener struct {
			ListenerArn string `json:"ListenerArn"`
		} `json:"Listener"`
	}
	decode(t, w, &ln)
	listenerArn := ln.Listener.ListenerArn

	// Create endpoint group
	w = doCall(t, h, "CreateEndpointGroup", map[string]any{
		"ListenerArn":         listenerArn,
		"EndpointGroupRegion": "us-east-1",
		"EndpointConfigurations": []map[string]any{
			{"EndpointId": "i-12345", "Weight": 100},
		},
	})
	mustOK(t, w, "CreateEndpointGroup")
	var eg struct {
		EndpointGroup struct {
			EndpointGroupArn string `json:"EndpointGroupArn"`
		} `json:"EndpointGroup"`
	}
	decode(t, w, &eg)
	groupArn := eg.EndpointGroup.EndpointGroupArn

	// Describe
	mustOK(t, doCall(t, h, "DescribeEndpointGroup", map[string]any{"EndpointGroupArn": groupArn}), "DescribeEndpointGroup")

	// List
	w = doCall(t, h, "ListEndpointGroups", map[string]any{"ListenerArn": listenerArn})
	mustOK(t, w, "ListEndpointGroups")
	var listed struct {
		EndpointGroups []struct {
			EndpointGroupArn string `json:"EndpointGroupArn"`
		} `json:"EndpointGroups"`
	}
	decode(t, w, &listed)
	if len(listed.EndpointGroups) != 1 {
		t.Fatalf("expected 1 endpoint group, got %d", len(listed.EndpointGroups))
	}

	// Update
	mustOK(t, doCall(t, h, "UpdateEndpointGroup", map[string]any{
		"EndpointGroupArn":      groupArn,
		"TrafficDialPercentage": 50,
	}), "UpdateEndpointGroup")

	// Add endpoint
	mustOK(t, doCall(t, h, "AddEndpoints", map[string]any{
		"EndpointGroupArn": groupArn,
		"EndpointConfigurations": []map[string]any{
			{"EndpointId": "i-67890", "Weight": 50},
		},
	}), "AddEndpoints")

	// Remove endpoint
	mustOK(t, doCall(t, h, "RemoveEndpoints", map[string]any{
		"EndpointGroupArn": groupArn,
		"EndpointIdentifiers": []map[string]any{
			{"EndpointId": "i-12345"},
		},
	}), "RemoveEndpoints")

	// Delete
	mustOK(t, doCall(t, h, "DeleteEndpointGroup", map[string]any{"EndpointGroupArn": groupArn}), "DeleteEndpointGroup")
	if w := doCall(t, h, "DescribeEndpointGroup", map[string]any{"EndpointGroupArn": groupArn}); w.Code != http.StatusNotFound {
		t.Fatalf("DescribeEndpointGroup after delete: want 404, got %d", w.Code)
	}
}

func TestCreateEndpointGroupValidation(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "CreateEndpointGroup", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

// ── Custom Routing Endpoint Group lifecycle ──────────────────────────────────

func TestCustomRoutingEndpointGroupLifecycle(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "CreateCustomRoutingAccelerator", map[string]any{"Name": "cra"})
	mustOK(t, w, "CreateCustomRoutingAccelerator")
	var ac struct {
		Accelerator struct {
			AcceleratorArn string `json:"AcceleratorArn"`
		} `json:"Accelerator"`
	}
	decode(t, w, &ac)
	acceleratorArn := ac.Accelerator.AcceleratorArn

	w = doCall(t, h, "CreateCustomRoutingListener", map[string]any{
		"AcceleratorArn": acceleratorArn,
		"PortRanges":     []map[string]any{{"FromPort": 1000, "ToPort": 2000}},
	})
	mustOK(t, w, "CreateCustomRoutingListener")
	var ln struct {
		Listener struct {
			ListenerArn string `json:"ListenerArn"`
		} `json:"Listener"`
	}
	decode(t, w, &ln)
	listenerArn := ln.Listener.ListenerArn

	w = doCall(t, h, "CreateCustomRoutingEndpointGroup", map[string]any{
		"ListenerArn":         listenerArn,
		"EndpointGroupRegion": "us-east-1",
		"DestinationConfigurations": []map[string]any{
			{"FromPort": 1000, "ToPort": 2000, "Protocols": []string{"TCP"}},
		},
	})
	mustOK(t, w, "CreateCustomRoutingEndpointGroup")
	var eg struct {
		EndpointGroup struct {
			EndpointGroupArn string `json:"EndpointGroupArn"`
		} `json:"EndpointGroup"`
	}
	decode(t, w, &eg)
	groupArn := eg.EndpointGroup.EndpointGroupArn

	mustOK(t, doCall(t, h, "DescribeCustomRoutingEndpointGroup", map[string]any{"EndpointGroupArn": groupArn}), "DescribeCustomRoutingEndpointGroup")

	w = doCall(t, h, "ListCustomRoutingEndpointGroups", map[string]any{"ListenerArn": listenerArn})
	mustOK(t, w, "ListCustomRoutingEndpointGroups")

	mustOK(t, doCall(t, h, "AddCustomRoutingEndpoints", map[string]any{
		"EndpointGroupArn": groupArn,
		"EndpointConfigurations": []map[string]any{
			{"EndpointId": "subnet-12345"},
		},
	}), "AddCustomRoutingEndpoints")

	mustOK(t, doCall(t, h, "AllowCustomRoutingTraffic", map[string]any{
		"EndpointGroupArn":     groupArn,
		"EndpointId":           "subnet-12345",
		"DestinationAddresses": []string{"10.0.0.1"},
		"DestinationPorts":     []int{80},
	}), "AllowCustomRoutingTraffic")

	mustOK(t, doCall(t, h, "DenyCustomRoutingTraffic", map[string]any{
		"EndpointGroupArn":     groupArn,
		"EndpointId":           "subnet-12345",
		"DestinationAddresses": []string{"10.0.0.1"},
		"DestinationPorts":     []int{80},
	}), "DenyCustomRoutingTraffic")

	mustOK(t, doCall(t, h, "ListCustomRoutingPortMappings", map[string]any{"AcceleratorArn": acceleratorArn}), "ListCustomRoutingPortMappings")

	mustOK(t, doCall(t, h, "ListCustomRoutingPortMappingsByDestination", map[string]any{
		"EndpointId":         "subnet-12345",
		"DestinationAddress": "10.0.0.1",
	}), "ListCustomRoutingPortMappingsByDestination")

	mustOK(t, doCall(t, h, "RemoveCustomRoutingEndpoints", map[string]any{
		"EndpointGroupArn": groupArn,
		"EndpointIds":      []string{"subnet-12345"},
	}), "RemoveCustomRoutingEndpoints")

	mustOK(t, doCall(t, h, "DeleteCustomRoutingEndpointGroup", map[string]any{"EndpointGroupArn": groupArn}), "DeleteCustomRoutingEndpointGroup")
}

func TestCreateCustomRoutingEndpointGroupValidation(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "CreateCustomRoutingEndpointGroup", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

// ── BYOIP CIDR lifecycle ─────────────────────────────────────────────────────

func TestByoipCidrLifecycle(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "ProvisionByoipCidr", map[string]any{"Cidr": "192.168.0.0/24"})
	mustOK(t, w, "ProvisionByoipCidr")

	mustOK(t, doCall(t, h, "AdvertiseByoipCidr", map[string]any{"Cidr": "192.168.0.0/24"}), "AdvertiseByoipCidr")

	w = doCall(t, h, "ListByoipCidrs", nil)
	mustOK(t, w, "ListByoipCidrs")
	var listed struct {
		ByoipCidrs []struct {
			Cidr  string `json:"Cidr"`
			State string `json:"State"`
		} `json:"ByoipCidrs"`
	}
	decode(t, w, &listed)
	if len(listed.ByoipCidrs) != 1 || listed.ByoipCidrs[0].Cidr != "192.168.0.0/24" {
		t.Fatalf("unexpected listed cidrs: %+v", listed)
	}
	if listed.ByoipCidrs[0].State != "ADVERTISING" {
		t.Fatalf("expected ADVERTISING state, got %s", listed.ByoipCidrs[0].State)
	}

	mustOK(t, doCall(t, h, "WithdrawByoipCidr", map[string]any{"Cidr": "192.168.0.0/24"}), "WithdrawByoipCidr")
	mustOK(t, doCall(t, h, "DeprovisionByoipCidr", map[string]any{"Cidr": "192.168.0.0/24"}), "DeprovisionByoipCidr")
}

func TestProvisionByoipCidrValidation(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "ProvisionByoipCidr", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

// ── Cross Account Attachment lifecycle ───────────────────────────────────────

func TestCrossAccountAttachmentLifecycle(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "CreateCrossAccountAttachment", map[string]any{
		"Name":       "att-1",
		"Principals": []string{"123456789012"},
		"Resources": []map[string]any{
			{"EndpointId": "i-12345", "Region": "us-east-1"},
		},
	})
	mustOK(t, w, "CreateCrossAccountAttachment")
	var att struct {
		CrossAccountAttachment struct {
			AttachmentArn string `json:"AttachmentArn"`
		} `json:"CrossAccountAttachment"`
	}
	decode(t, w, &att)
	arn := att.CrossAccountAttachment.AttachmentArn
	if arn == "" {
		t.Fatalf("missing AttachmentArn: %s", w.Body.String())
	}

	mustOK(t, doCall(t, h, "DescribeCrossAccountAttachment", map[string]any{"AttachmentArn": arn}), "DescribeCrossAccountAttachment")

	w = doCall(t, h, "ListCrossAccountAttachments", nil)
	mustOK(t, w, "ListCrossAccountAttachments")
	var listed struct {
		CrossAccountAttachments []struct {
			AttachmentArn string `json:"AttachmentArn"`
		} `json:"CrossAccountAttachments"`
	}
	decode(t, w, &listed)
	if len(listed.CrossAccountAttachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(listed.CrossAccountAttachments))
	}

	mustOK(t, doCall(t, h, "UpdateCrossAccountAttachment", map[string]any{
		"AttachmentArn": arn,
		"Name":          "att-2",
		"AddPrincipals": []string{"210987654321"},
	}), "UpdateCrossAccountAttachment")

	mustOK(t, doCall(t, h, "ListCrossAccountResourceAccounts", nil), "ListCrossAccountResourceAccounts")
	mustOK(t, doCall(t, h, "ListCrossAccountResources", map[string]any{
		"ResourceOwnerAwsAccountId": "123456789012",
	}), "ListCrossAccountResources")

	mustOK(t, doCall(t, h, "DeleteCrossAccountAttachment", map[string]any{"AttachmentArn": arn}), "DeleteCrossAccountAttachment")
}

func TestCreateCrossAccountAttachmentValidation(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "CreateCrossAccountAttachment", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

// ── Tag handlers ─────────────────────────────────────────────────────────────

func TestTagsLifecycle(t *testing.T) {
	h := newGateway(t)
	arn := "arn:aws:globalaccelerator::000000000000:accelerator/abc"

	mustOK(t, doCall(t, h, "TagResource", map[string]any{
		"ResourceArn": arn,
		"Tags": []map[string]any{
			{"Key": "env", "Value": "dev"},
		},
	}), "TagResource")

	w := doCall(t, h, "ListTagsForResource", map[string]any{"ResourceArn": arn})
	mustOK(t, w, "ListTagsForResource")
	var listed struct {
		Tags []struct {
			Key   string `json:"Key"`
			Value string `json:"Value"`
		} `json:"Tags"`
	}
	decode(t, w, &listed)
	if len(listed.Tags) != 1 || listed.Tags[0].Value != "dev" {
		t.Fatalf("unexpected tags: %+v", listed)
	}

	mustOK(t, doCall(t, h, "UntagResource", map[string]any{
		"ResourceArn": arn,
		"TagKeys":     []string{"env"},
	}), "UntagResource")

	w = doCall(t, h, "ListTagsForResource", map[string]any{"ResourceArn": arn})
	mustOK(t, w, "ListTagsForResource")
	decode(t, w, &listed)
	if len(listed.Tags) != 0 {
		t.Fatalf("expected 0 tags after untag, got %d", len(listed.Tags))
	}
}

func TestTagResourceValidation(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "TagResource", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}
