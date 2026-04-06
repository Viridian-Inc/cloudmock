package globalaccelerator_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	svc "github.com/neureaux/cloudmock/services/globalaccelerator"
)

func newGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(svc.New(cfg.AccountID, cfg.Region))
	return gateway.New(cfg, reg)
}

func svcReq(t *testing.T, action string, body any) *http.Request {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "GlobalAccelerator_V20180706."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/globalaccelerator/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}


func TestAddCustomRoutingEndpoints(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "AddCustomRoutingEndpoints", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("AddCustomRoutingEndpoints: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestAddEndpoints(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "AddEndpoints", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("AddEndpoints: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestAdvertiseByoipCidr(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "AdvertiseByoipCidr", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("AdvertiseByoipCidr: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestAllowCustomRoutingTraffic(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "AllowCustomRoutingTraffic", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("AllowCustomRoutingTraffic: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateAccelerator(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateAccelerator", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateAccelerator: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateCrossAccountAttachment(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateCrossAccountAttachment", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateCrossAccountAttachment: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateCustomRoutingAccelerator(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateCustomRoutingAccelerator", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateCustomRoutingAccelerator: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateCustomRoutingEndpointGroup(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateCustomRoutingEndpointGroup", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateCustomRoutingEndpointGroup: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateCustomRoutingListener(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateCustomRoutingListener", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateCustomRoutingListener: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateEndpointGroup(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateEndpointGroup", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateEndpointGroup: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateListener(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateListener", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateListener: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteAccelerator(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteAccelerator", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteAccelerator: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteCrossAccountAttachment(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteCrossAccountAttachment", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteCrossAccountAttachment: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteCustomRoutingAccelerator(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteCustomRoutingAccelerator", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteCustomRoutingAccelerator: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteCustomRoutingEndpointGroup(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteCustomRoutingEndpointGroup", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteCustomRoutingEndpointGroup: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteCustomRoutingListener(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteCustomRoutingListener", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteCustomRoutingListener: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteEndpointGroup(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteEndpointGroup", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteEndpointGroup: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteListener(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteListener", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteListener: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDenyCustomRoutingTraffic(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DenyCustomRoutingTraffic", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DenyCustomRoutingTraffic: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeprovisionByoipCidr(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeprovisionByoipCidr", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeprovisionByoipCidr: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeAccelerator(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeAccelerator", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeAccelerator: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeAcceleratorAttributes(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeAcceleratorAttributes", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeAcceleratorAttributes: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeCrossAccountAttachment(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeCrossAccountAttachment", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeCrossAccountAttachment: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeCustomRoutingAccelerator(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeCustomRoutingAccelerator", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeCustomRoutingAccelerator: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeCustomRoutingAcceleratorAttributes(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeCustomRoutingAcceleratorAttributes", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeCustomRoutingAcceleratorAttributes: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeCustomRoutingEndpointGroup(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeCustomRoutingEndpointGroup", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeCustomRoutingEndpointGroup: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeCustomRoutingListener(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeCustomRoutingListener", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeCustomRoutingListener: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeEndpointGroup(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeEndpointGroup", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeEndpointGroup: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDescribeListener(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DescribeListener", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeListener: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListAccelerators(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListAccelerators", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListAccelerators: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListByoipCidrs(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListByoipCidrs", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListByoipCidrs: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCrossAccountAttachments(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCrossAccountAttachments", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCrossAccountAttachments: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCrossAccountResourceAccounts(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCrossAccountResourceAccounts", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCrossAccountResourceAccounts: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCrossAccountResources(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCrossAccountResources", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCrossAccountResources: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCustomRoutingAccelerators(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCustomRoutingAccelerators", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCustomRoutingAccelerators: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCustomRoutingEndpointGroups(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCustomRoutingEndpointGroups", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCustomRoutingEndpointGroups: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCustomRoutingListeners(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCustomRoutingListeners", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCustomRoutingListeners: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCustomRoutingPortMappings(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCustomRoutingPortMappings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCustomRoutingPortMappings: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListCustomRoutingPortMappingsByDestination(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListCustomRoutingPortMappingsByDestination", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListCustomRoutingPortMappingsByDestination: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListEndpointGroups(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListEndpointGroups", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListEndpointGroups: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListListeners(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListListeners", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListListeners: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListTagsForResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTagsForResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTagsForResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestProvisionByoipCidr(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ProvisionByoipCidr", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ProvisionByoipCidr: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestRemoveCustomRoutingEndpoints(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "RemoveCustomRoutingEndpoints", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("RemoveCustomRoutingEndpoints: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestRemoveEndpoints(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "RemoveEndpoints", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("RemoveEndpoints: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestTagResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "TagResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("TagResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUntagResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UntagResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UntagResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAccelerator(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateAccelerator", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateAccelerator: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAcceleratorAttributes(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateAcceleratorAttributes", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateAcceleratorAttributes: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateCrossAccountAttachment(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateCrossAccountAttachment", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateCrossAccountAttachment: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateCustomRoutingAccelerator(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateCustomRoutingAccelerator", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateCustomRoutingAccelerator: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateCustomRoutingAcceleratorAttributes(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateCustomRoutingAcceleratorAttributes", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateCustomRoutingAcceleratorAttributes: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateCustomRoutingListener(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateCustomRoutingListener", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateCustomRoutingListener: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateEndpointGroup(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateEndpointGroup", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateEndpointGroup: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateListener(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UpdateListener", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateListener: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestWithdrawByoipCidr(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "WithdrawByoipCidr", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("WithdrawByoipCidr: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

