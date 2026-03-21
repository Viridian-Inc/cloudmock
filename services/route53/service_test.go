package route53_test

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	r53svc "github.com/neureaux/cloudmock/services/route53"
)

// newR53Gateway builds a full gateway stack with the Route 53 service registered and IAM disabled.
func newR53Gateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"

	reg := routing.NewRegistry()
	reg.Register(r53svc.New(cfg.AccountID, cfg.Region))

	return gateway.New(cfg, reg)
}

// r53Req builds an HTTP request targeting the Route 53 service.
func r53Req(t *testing.T, method, path string, body string) *http.Request {
	t.Helper()
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/route53/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// createZoneBody returns a CreateHostedZoneRequest XML body.
func createZoneBody(name, callerRef string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<CreateHostedZoneRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <Name>%s</Name>
  <CallerReference>%s</CallerReference>
</CreateHostedZoneRequest>`, name, callerRef)
}

// mustCreateZone is a test helper that creates a hosted zone and returns its short ID.
func mustCreateZone(t *testing.T, handler http.Handler, name, callerRef string) string {
	t.Helper()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone", createZoneBody(name, callerRef)))

	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("CreateHostedZone %s: expected 201, got %d\nbody: %s", name, w.Code, w.Body.String())
	}

	var resp struct {
		HostedZone struct {
			Id string `xml:"Id"`
		} `xml:"HostedZone"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("CreateHostedZone: unmarshal response: %v\nbody: %s", err, w.Body.String())
	}
	if resp.HostedZone.Id == "" {
		t.Fatalf("CreateHostedZone: Id is empty\nbody: %s", w.Body.String())
	}

	// Strip /hostedzone/ prefix to return the short ID.
	id := resp.HostedZone.Id
	id = strings.TrimPrefix(id, "/hostedzone/")
	return id
}

// ---- Test 1: CreateHostedZone + ListHostedZones ----

func TestRoute53_CreateAndListHostedZones(t *testing.T) {
	handler := newR53Gateway(t)

	id1 := mustCreateZone(t, handler, "example.com", "ref-1")
	id2 := mustCreateZone(t, handler, "myapp.io", "ref-2")

	if id1 == "" || id2 == "" {
		t.Fatal("zone IDs must not be empty")
	}

	// ListHostedZones.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone", ""))
	if w.Code != http.StatusOK {
		t.Fatalf("ListHostedZones: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "example.com") {
		t.Errorf("ListHostedZones: expected example.com in response\nbody: %s", body)
	}
	if !strings.Contains(body, "myapp.io") {
		t.Errorf("ListHostedZones: expected myapp.io in response\nbody: %s", body)
	}

	// Verify response structure.
	var listResp struct {
		HostedZones []struct {
			Id   string `xml:"Id"`
			Name string `xml:"Name"`
		} `xml:"HostedZones>HostedZone"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("ListHostedZones: unmarshal: %v\nbody: %s", err, body)
	}
	if len(listResp.HostedZones) < 2 {
		t.Errorf("ListHostedZones: expected at least 2 zones, got %d", len(listResp.HostedZones))
	}
}

// ---- Test 2: GetHostedZone ----

func TestRoute53_GetHostedZone(t *testing.T) {
	handler := newR53Gateway(t)

	id := mustCreateZone(t, handler, "gettest.net", "get-ref-1")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone/"+id, ""))
	if w.Code != http.StatusOK {
		t.Fatalf("GetHostedZone: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var resp struct {
		HostedZone struct {
			Id   string `xml:"Id"`
			Name string `xml:"Name"`
		} `xml:"HostedZone"`
		DelegationSet struct {
			NameServers []string `xml:"NameServers>NameServer"`
		} `xml:"DelegationSet"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("GetHostedZone: unmarshal: %v\nbody: %s", err, w.Body.String())
	}

	if !strings.Contains(resp.HostedZone.Id, id) {
		t.Errorf("GetHostedZone: expected Id to contain %s, got %s", id, resp.HostedZone.Id)
	}
	if !strings.Contains(resp.HostedZone.Name, "gettest.net") {
		t.Errorf("GetHostedZone: expected Name to contain gettest.net, got %s", resp.HostedZone.Name)
	}
	if len(resp.DelegationSet.NameServers) == 0 {
		t.Error("GetHostedZone: expected at least one NameServer in DelegationSet")
	}

	// Test non-existent zone returns 404.
	wNotFound := httptest.NewRecorder()
	handler.ServeHTTP(wNotFound, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone/ZNONEXISTENT", ""))
	if wNotFound.Code != http.StatusNotFound {
		t.Errorf("GetHostedZone (not found): expected 404, got %d", wNotFound.Code)
	}
}

// ---- Test 3: ChangeResourceRecordSets (CREATE) + ListResourceRecordSets ----

func TestRoute53_ChangeAndListResourceRecordSets(t *testing.T) {
	handler := newR53Gateway(t)

	id := mustCreateZone(t, handler, "records.example.com", "rrset-ref-1")

	// CREATE an A record.
	changeBody := `<?xml version="1.0" encoding="UTF-8"?>
<ChangeResourceRecordSetsRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeBatch>
    <Changes>
      <Change>
        <Action>CREATE</Action>
        <ResourceRecordSet>
          <Name>www.records.example.com</Name>
          <Type>A</Type>
          <TTL>300</TTL>
          <ResourceRecords>
            <ResourceRecord>
              <Value>1.2.3.4</Value>
            </ResourceRecord>
          </ResourceRecords>
        </ResourceRecordSet>
      </Change>
    </Changes>
  </ChangeBatch>
</ChangeResourceRecordSetsRequest>`

	wc := httptest.NewRecorder()
	handler.ServeHTTP(wc, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone/"+id+"/rrset", changeBody))
	if wc.Code != http.StatusOK {
		t.Fatalf("ChangeResourceRecordSets (CREATE): expected 200, got %d\nbody: %s", wc.Code, wc.Body.String())
	}

	for _, want := range []string{"ChangeInfo", "INSYNC", "SubmittedAt"} {
		if !strings.Contains(wc.Body.String(), want) {
			t.Errorf("ChangeResourceRecordSets: expected %q in response\nbody: %s", want, wc.Body.String())
		}
	}

	// ListResourceRecordSets — should include NS, SOA, and our new A record.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone/"+id+"/rrset", ""))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListResourceRecordSets: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}

	listBody := wl.Body.String()
	for _, want := range []string{"www.records.example.com", "1.2.3.4", "NS", "SOA"} {
		if !strings.Contains(listBody, want) {
			t.Errorf("ListResourceRecordSets: expected %q in response\nbody: %s", want, listBody)
		}
	}
}

// ---- Test 4: ChangeResourceRecordSets UPSERT and DELETE ----

func TestRoute53_UpsertAndDeleteRecords(t *testing.T) {
	handler := newR53Gateway(t)

	id := mustCreateZone(t, handler, "upsert.example.com", "upsert-ref-1")

	// Helper to send a change batch.
	sendChange := func(action, name, rrType, ttl, value string) *httptest.ResponseRecorder {
		body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<ChangeResourceRecordSetsRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeBatch>
    <Changes>
      <Change>
        <Action>%s</Action>
        <ResourceRecordSet>
          <Name>%s</Name>
          <Type>%s</Type>
          <TTL>%s</TTL>
          <ResourceRecords>
            <ResourceRecord>
              <Value>%s</Value>
            </ResourceRecord>
          </ResourceRecords>
        </ResourceRecordSet>
      </Change>
    </Changes>
  </ChangeBatch>
</ChangeResourceRecordSetsRequest>`, action, name, rrType, ttl, value)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone/"+id+"/rrset", body))
		return w
	}

	// CREATE initial CNAME record.
	wCreate := sendChange("CREATE", "api.upsert.example.com", "CNAME", "300", "old-target.example.com")
	if wCreate.Code != http.StatusOK {
		t.Fatalf("CREATE CNAME: expected 200, got %d\nbody: %s", wCreate.Code, wCreate.Body.String())
	}

	// UPSERT — change the CNAME value.
	wUpsert := sendChange("UPSERT", "api.upsert.example.com", "CNAME", "300", "new-target.example.com")
	if wUpsert.Code != http.StatusOK {
		t.Fatalf("UPSERT CNAME: expected 200, got %d\nbody: %s", wUpsert.Code, wUpsert.Body.String())
	}

	// Verify updated value appears.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone/"+id+"/rrset", ""))
	if !strings.Contains(wl.Body.String(), "new-target.example.com") {
		t.Errorf("after UPSERT: expected new-target.example.com\nbody: %s", wl.Body.String())
	}
	if strings.Contains(wl.Body.String(), "old-target.example.com") {
		t.Errorf("after UPSERT: old-target.example.com should be gone\nbody: %s", wl.Body.String())
	}

	// DELETE the CNAME record.
	wDelete := sendChange("DELETE", "api.upsert.example.com", "CNAME", "300", "new-target.example.com")
	if wDelete.Code != http.StatusOK {
		t.Fatalf("DELETE CNAME: expected 200, got %d\nbody: %s", wDelete.Code, wDelete.Body.String())
	}

	// Verify record is gone.
	wl2 := httptest.NewRecorder()
	handler.ServeHTTP(wl2, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone/"+id+"/rrset", ""))
	if strings.Contains(wl2.Body.String(), "api.upsert.example.com") {
		t.Errorf("after DELETE: api.upsert.example.com should not be present\nbody: %s", wl2.Body.String())
	}

	// DELETE non-existent record should return error.
	wBadDelete := sendChange("DELETE", "nonexistent.upsert.example.com", "A", "300", "9.9.9.9")
	if wBadDelete.Code == http.StatusOK {
		t.Errorf("DELETE non-existent record: expected error, got 200\nbody: %s", wBadDelete.Body.String())
	}
}

// ---- Test 5: DeleteHostedZone ----

func TestRoute53_DeleteHostedZone(t *testing.T) {
	handler := newR53Gateway(t)

	id := mustCreateZone(t, handler, "tobedeleted.com", "del-ref-1")

	// Verify it appears in list.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone", ""))
	if !strings.Contains(wl.Body.String(), "tobedeleted.com") {
		t.Fatal("ListHostedZones: zone not found before deletion")
	}

	// Delete it.
	wd := httptest.NewRecorder()
	handler.ServeHTTP(wd, r53Req(t, http.MethodDelete, "/2013-04-01/hostedzone/"+id, ""))
	if wd.Code != http.StatusOK {
		t.Fatalf("DeleteHostedZone: expected 200, got %d\nbody: %s", wd.Code, wd.Body.String())
	}

	for _, want := range []string{"ChangeInfo", "INSYNC"} {
		if !strings.Contains(wd.Body.String(), want) {
			t.Errorf("DeleteHostedZone: expected %q in response\nbody: %s", want, wd.Body.String())
		}
	}

	// Verify it's gone from list.
	wl2 := httptest.NewRecorder()
	handler.ServeHTTP(wl2, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone", ""))
	if strings.Contains(wl2.Body.String(), "tobedeleted.com") {
		t.Error("ListHostedZones: zone should not appear after deletion")
	}

	// GetHostedZone after deletion should return 404.
	wg := httptest.NewRecorder()
	handler.ServeHTTP(wg, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone/"+id, ""))
	if wg.Code != http.StatusNotFound {
		t.Errorf("GetHostedZone after delete: expected 404, got %d", wg.Code)
	}

	// Delete again — should get 404.
	wd2 := httptest.NewRecorder()
	handler.ServeHTTP(wd2, r53Req(t, http.MethodDelete, "/2013-04-01/hostedzone/"+id, ""))
	if wd2.Code != http.StatusNotFound {
		t.Errorf("DeleteHostedZone (already deleted): expected 404, got %d", wd2.Code)
	}
}

// ---- Test 6: Auto-created NS and SOA records ----

func TestRoute53_DefaultNSAndSOARecords(t *testing.T) {
	handler := newR53Gateway(t)

	id := mustCreateZone(t, handler, "nstest.example.com", "ns-ref-1")

	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone/"+id+"/rrset", ""))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListResourceRecordSets: expected 200, got %d\nbody: %s", wl.Code, wl.Body.String())
	}

	var resp struct {
		ResourceRecordSets []struct {
			Type string `xml:"Type"`
		} `xml:"ResourceRecordSets>ResourceRecordSet"`
	}
	if err := xml.Unmarshal(wl.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	hasNS := false
	hasSOA := false
	for _, rs := range resp.ResourceRecordSets {
		if rs.Type == "NS" {
			hasNS = true
		}
		if rs.Type == "SOA" {
			hasSOA = true
		}
	}
	if !hasNS {
		t.Errorf("expected auto-created NS record\nbody: %s", wl.Body.String())
	}
	if !hasSOA {
		t.Errorf("expected auto-created SOA record\nbody: %s", wl.Body.String())
	}
}

