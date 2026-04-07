package route53_test

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	r53svc "github.com/Viridian-Inc/cloudmock/services/route53"
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

// ---- Test 7: ChangeResourceRecordSets with AAAA record ----

func TestRoute53_ChangeRecords_AAAA(t *testing.T) {
	handler := newR53Gateway(t)

	id := mustCreateZone(t, handler, "ipv6.example.com", "aaaa-ref-1")

	changeBody := `<?xml version="1.0" encoding="UTF-8"?>
<ChangeResourceRecordSetsRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeBatch>
    <Changes>
      <Change>
        <Action>CREATE</Action>
        <ResourceRecordSet>
          <Name>v6.ipv6.example.com</Name>
          <Type>AAAA</Type>
          <TTL>300</TTL>
          <ResourceRecords>
            <ResourceRecord>
              <Value>2001:db8::1</Value>
            </ResourceRecord>
          </ResourceRecords>
        </ResourceRecordSet>
      </Change>
    </Changes>
  </ChangeBatch>
</ChangeResourceRecordSetsRequest>`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone/"+id+"/rrset", changeBody))
	if w.Code != http.StatusOK {
		t.Fatalf("CREATE AAAA: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify the AAAA record appears in listing.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone/"+id+"/rrset", ""))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListResourceRecordSets: expected 200, got %d", wl.Code)
	}

	body := wl.Body.String()
	if !strings.Contains(body, "AAAA") {
		t.Errorf("expected AAAA record type in listing\nbody: %s", body)
	}
	if !strings.Contains(body, "2001:db8::1") {
		t.Errorf("expected 2001:db8::1 in listing\nbody: %s", body)
	}
}

// ---- Test 8: ChangeResourceRecordSets with MX record ----

func TestRoute53_ChangeRecords_MX(t *testing.T) {
	handler := newR53Gateway(t)

	id := mustCreateZone(t, handler, "mail.example.com", "mx-ref-1")

	changeBody := `<?xml version="1.0" encoding="UTF-8"?>
<ChangeResourceRecordSetsRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeBatch>
    <Changes>
      <Change>
        <Action>CREATE</Action>
        <ResourceRecordSet>
          <Name>mail.example.com</Name>
          <Type>MX</Type>
          <TTL>3600</TTL>
          <ResourceRecords>
            <ResourceRecord>
              <Value>10 mx1.mail.example.com</Value>
            </ResourceRecord>
            <ResourceRecord>
              <Value>20 mx2.mail.example.com</Value>
            </ResourceRecord>
          </ResourceRecords>
        </ResourceRecordSet>
      </Change>
    </Changes>
  </ChangeBatch>
</ChangeResourceRecordSetsRequest>`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone/"+id+"/rrset", changeBody))
	if w.Code != http.StatusOK {
		t.Fatalf("CREATE MX: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify both MX values appear.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone/"+id+"/rrset", ""))
	body := wl.Body.String()
	if !strings.Contains(body, "MX") {
		t.Errorf("expected MX record type in listing\nbody: %s", body)
	}
	if !strings.Contains(body, "10 mx1.mail.example.com") {
		t.Errorf("expected first MX value\nbody: %s", body)
	}
	if !strings.Contains(body, "20 mx2.mail.example.com") {
		t.Errorf("expected second MX value\nbody: %s", body)
	}
}

// ---- Test 9: ChangeResourceRecordSets with TXT record ----

func TestRoute53_ChangeRecords_TXT(t *testing.T) {
	handler := newR53Gateway(t)

	id := mustCreateZone(t, handler, "txt.example.com", "txt-ref-1")

	changeBody := `<?xml version="1.0" encoding="UTF-8"?>
<ChangeResourceRecordSetsRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeBatch>
    <Changes>
      <Change>
        <Action>CREATE</Action>
        <ResourceRecordSet>
          <Name>txt.example.com</Name>
          <Type>TXT</Type>
          <TTL>300</TTL>
          <ResourceRecords>
            <ResourceRecord>
              <Value>"v=spf1 include:_spf.google.com ~all"</Value>
            </ResourceRecord>
          </ResourceRecords>
        </ResourceRecordSet>
      </Change>
    </Changes>
  </ChangeBatch>
</ChangeResourceRecordSetsRequest>`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone/"+id+"/rrset", changeBody))
	if w.Code != http.StatusOK {
		t.Fatalf("CREATE TXT: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone/"+id+"/rrset", ""))
	body := wl.Body.String()
	if !strings.Contains(body, "TXT") {
		t.Errorf("expected TXT record type in listing\nbody: %s", body)
	}
	if !strings.Contains(body, "v=spf1") {
		t.Errorf("expected SPF value in TXT record\nbody: %s", body)
	}
}

// ---- Test 10: Multiple record types in a single ChangeBatch ----

func TestRoute53_MultipleCREATEsInSingleBatch(t *testing.T) {
	handler := newR53Gateway(t)

	id := mustCreateZone(t, handler, "batch.example.com", "batch-ref-1")

	changeBody := `<?xml version="1.0" encoding="UTF-8"?>
<ChangeResourceRecordSetsRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeBatch>
    <Changes>
      <Change>
        <Action>CREATE</Action>
        <ResourceRecordSet>
          <Name>a.batch.example.com</Name>
          <Type>A</Type>
          <TTL>300</TTL>
          <ResourceRecords>
            <ResourceRecord><Value>10.0.0.1</Value></ResourceRecord>
          </ResourceRecords>
        </ResourceRecordSet>
      </Change>
      <Change>
        <Action>CREATE</Action>
        <ResourceRecordSet>
          <Name>b.batch.example.com</Name>
          <Type>AAAA</Type>
          <TTL>300</TTL>
          <ResourceRecords>
            <ResourceRecord><Value>2001:db8::2</Value></ResourceRecord>
          </ResourceRecords>
        </ResourceRecordSet>
      </Change>
      <Change>
        <Action>CREATE</Action>
        <ResourceRecordSet>
          <Name>c.batch.example.com</Name>
          <Type>CNAME</Type>
          <TTL>300</TTL>
          <ResourceRecords>
            <ResourceRecord><Value>target.batch.example.com</Value></ResourceRecord>
          </ResourceRecords>
        </ResourceRecordSet>
      </Change>
    </Changes>
  </ChangeBatch>
</ChangeResourceRecordSetsRequest>`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone/"+id+"/rrset", changeBody))
	if w.Code != http.StatusOK {
		t.Fatalf("batch CREATE: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// List and verify all three records plus NS+SOA.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone/"+id+"/rrset", ""))

	var resp struct {
		ResourceRecordSets []struct {
			Name string `xml:"Name"`
			Type string `xml:"Type"`
		} `xml:"ResourceRecordSets>ResourceRecordSet"`
	}
	if err := xml.Unmarshal(wl.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Expect: NS, SOA, A, AAAA, CNAME = 5 records.
	if len(resp.ResourceRecordSets) < 5 {
		t.Errorf("expected at least 5 record sets, got %d", len(resp.ResourceRecordSets))
	}

	types := make(map[string]bool)
	for _, rs := range resp.ResourceRecordSets {
		types[rs.Type] = true
	}
	for _, want := range []string{"NS", "SOA", "A", "AAAA", "CNAME"} {
		if !types[want] {
			t.Errorf("expected record type %s in listing", want)
		}
	}
}

// ---- Test 11: ListResourceRecordSets returns IsTruncated and MaxItems ----

func TestRoute53_ListResourceRecordSets_PaginationFields(t *testing.T) {
	handler := newR53Gateway(t)

	id := mustCreateZone(t, handler, "paginate.example.com", "page-ref-1")

	// Add several records so the listing has more content.
	for i := 0; i < 5; i++ {
		body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<ChangeResourceRecordSetsRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeBatch>
    <Changes>
      <Change>
        <Action>CREATE</Action>
        <ResourceRecordSet>
          <Name>host%d.paginate.example.com</Name>
          <Type>A</Type>
          <TTL>300</TTL>
          <ResourceRecords>
            <ResourceRecord><Value>10.0.0.%d</Value></ResourceRecord>
          </ResourceRecords>
        </ResourceRecordSet>
      </Change>
    </Changes>
  </ChangeBatch>
</ChangeResourceRecordSetsRequest>`, i, i+1)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone/"+id+"/rrset", body))
		if w.Code != http.StatusOK {
			t.Fatalf("CREATE host%d: expected 200, got %d", i, w.Code)
		}
	}

	// List and verify pagination fields.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone/"+id+"/rrset", ""))
	if wl.Code != http.StatusOK {
		t.Fatalf("ListResourceRecordSets: expected 200, got %d", wl.Code)
	}

	var listResp struct {
		IsTruncated        bool   `xml:"IsTruncated"`
		MaxItems           string `xml:"MaxItems"`
		ResourceRecordSets []struct {
			Name string `xml:"Name"`
			Type string `xml:"Type"`
		} `xml:"ResourceRecordSets>ResourceRecordSet"`
	}
	if err := xml.Unmarshal(wl.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// We have NS + SOA + 5 A records = 7.
	if len(listResp.ResourceRecordSets) != 7 {
		t.Errorf("expected 7 record sets, got %d", len(listResp.ResourceRecordSets))
	}
	if listResp.MaxItems == "" {
		t.Error("expected MaxItems to be set")
	}
	// IsTruncated should be false for this small set.
	if listResp.IsTruncated {
		t.Error("expected IsTruncated=false for small result set")
	}
}

// ---- Test 12: Error — NoSuchHostedZone for GetHostedZone ----

func TestRoute53_Error_NoSuchHostedZone_Get(t *testing.T) {
	handler := newR53Gateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone/ZNONEXISTENT999", ""))
	if w.Code != http.StatusNotFound {
		t.Fatalf("GetHostedZone (nonexistent): expected 404, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "NoSuchHostedZone") {
		t.Errorf("expected NoSuchHostedZone error code\nbody: %s", body)
	}
}

// ---- Test 13: Error — NoSuchHostedZone for DeleteHostedZone ----

func TestRoute53_Error_NoSuchHostedZone_Delete(t *testing.T) {
	handler := newR53Gateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodDelete, "/2013-04-01/hostedzone/ZNONEXISTENT999", ""))
	if w.Code != http.StatusNotFound {
		t.Fatalf("DeleteHostedZone (nonexistent): expected 404, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "NoSuchHostedZone") {
		t.Errorf("expected NoSuchHostedZone error code\nbody: %s", body)
	}
}

// ---- Test 14: Error — NoSuchHostedZone for ChangeResourceRecordSets ----

func TestRoute53_Error_NoSuchHostedZone_ChangeRecords(t *testing.T) {
	handler := newR53Gateway(t)

	changeBody := `<?xml version="1.0" encoding="UTF-8"?>
<ChangeResourceRecordSetsRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeBatch>
    <Changes>
      <Change>
        <Action>CREATE</Action>
        <ResourceRecordSet>
          <Name>test.nope.example.com</Name>
          <Type>A</Type>
          <TTL>300</TTL>
          <ResourceRecords>
            <ResourceRecord><Value>1.1.1.1</Value></ResourceRecord>
          </ResourceRecords>
        </ResourceRecordSet>
      </Change>
    </Changes>
  </ChangeBatch>
</ChangeResourceRecordSetsRequest>`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone/ZNONEXISTENT999/rrset", changeBody))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("ChangeRecords (nonexistent zone): expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "InvalidChangeBatch") {
		t.Errorf("expected InvalidChangeBatch error code\nbody: %s", body)
	}
}

// ---- Test 15: Error — NoSuchHostedZone for ListResourceRecordSets ----

func TestRoute53_Error_NoSuchHostedZone_ListRecords(t *testing.T) {
	handler := newR53Gateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone/ZNONEXISTENT999/rrset", ""))
	if w.Code != http.StatusNotFound {
		t.Fatalf("ListResourceRecordSets (nonexistent zone): expected 404, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "NoSuchHostedZone") {
		t.Errorf("expected NoSuchHostedZone error code\nbody: %s", body)
	}
}

// ---- Test 16: Error — InvalidChangeBatch for duplicate CREATE ----

func TestRoute53_Error_InvalidChangeBatch_DuplicateCREATE(t *testing.T) {
	handler := newR53Gateway(t)

	id := mustCreateZone(t, handler, "dup.example.com", "dup-ref-1")

	changeBody := `<?xml version="1.0" encoding="UTF-8"?>
<ChangeResourceRecordSetsRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeBatch>
    <Changes>
      <Change>
        <Action>CREATE</Action>
        <ResourceRecordSet>
          <Name>www.dup.example.com</Name>
          <Type>A</Type>
          <TTL>300</TTL>
          <ResourceRecords>
            <ResourceRecord><Value>1.2.3.4</Value></ResourceRecord>
          </ResourceRecords>
        </ResourceRecordSet>
      </Change>
    </Changes>
  </ChangeBatch>
</ChangeResourceRecordSetsRequest>`

	// First CREATE succeeds.
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone/"+id+"/rrset", changeBody))
	if w1.Code != http.StatusOK {
		t.Fatalf("first CREATE: expected 200, got %d", w1.Code)
	}

	// Second CREATE of same Name+Type should fail.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone/"+id+"/rrset", changeBody))
	if w2.Code == http.StatusOK {
		t.Fatalf("duplicate CREATE: expected error, got 200")
	}

	body := w2.Body.String()
	if !strings.Contains(body, "InvalidChangeBatch") {
		t.Errorf("expected InvalidChangeBatch error code\nbody: %s", body)
	}
}

// ---- Test 17: Error — InvalidChangeBatch for unknown action ----

func TestRoute53_Error_InvalidChangeBatch_UnknownAction(t *testing.T) {
	handler := newR53Gateway(t)

	id := mustCreateZone(t, handler, "badaction.example.com", "badaction-ref-1")

	changeBody := `<?xml version="1.0" encoding="UTF-8"?>
<ChangeResourceRecordSetsRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeBatch>
    <Changes>
      <Change>
        <Action>BADACTION</Action>
        <ResourceRecordSet>
          <Name>www.badaction.example.com</Name>
          <Type>A</Type>
          <TTL>300</TTL>
          <ResourceRecords>
            <ResourceRecord><Value>1.2.3.4</Value></ResourceRecord>
          </ResourceRecords>
        </ResourceRecordSet>
      </Change>
    </Changes>
  </ChangeBatch>
</ChangeResourceRecordSetsRequest>`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone/"+id+"/rrset", changeBody))
	if w.Code == http.StatusOK {
		t.Fatalf("unknown action: expected error, got 200")
	}

	body := w.Body.String()
	if !strings.Contains(body, "InvalidChangeBatch") {
		t.Errorf("expected InvalidChangeBatch error code\nbody: %s", body)
	}
}

// ---- Test 18: Error — CreateHostedZone missing Name ----

func TestRoute53_Error_CreateHostedZone_MissingName(t *testing.T) {
	handler := newR53Gateway(t)

	body := `<?xml version="1.0" encoding="UTF-8"?>
<CreateHostedZoneRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <Name></Name>
  <CallerReference>ref-missing-name</CallerReference>
</CreateHostedZoneRequest>`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone", body))
	if w.Code == http.StatusCreated || w.Code == http.StatusOK {
		t.Fatalf("CreateHostedZone (missing name): expected error, got %d", w.Code)
	}

	respBody := w.Body.String()
	if !strings.Contains(respBody, "ValidationError") {
		t.Errorf("expected ValidationError\nbody: %s", respBody)
	}
}

// ---- Test 19: Error — CreateHostedZone missing CallerReference ----

func TestRoute53_Error_CreateHostedZone_MissingCallerRef(t *testing.T) {
	handler := newR53Gateway(t)

	body := `<?xml version="1.0" encoding="UTF-8"?>
<CreateHostedZoneRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <Name>valid.example.com</Name>
  <CallerReference></CallerReference>
</CreateHostedZoneRequest>`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone", body))
	if w.Code == http.StatusCreated || w.Code == http.StatusOK {
		t.Fatalf("CreateHostedZone (missing CallerReference): expected error, got %d", w.Code)
	}

	respBody := w.Body.String()
	if !strings.Contains(respBody, "ValidationError") {
		t.Errorf("expected ValidationError\nbody: %s", respBody)
	}
}

// ---- Test 20: Error — CreateHostedZone malformed XML body ----

func TestRoute53_Error_CreateHostedZone_MalformedBody(t *testing.T) {
	handler := newR53Gateway(t)

	body := `this is not valid xml`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone", body))
	if w.Code == http.StatusCreated || w.Code == http.StatusOK {
		t.Fatalf("CreateHostedZone (malformed body): expected error, got %d", w.Code)
	}

	respBody := w.Body.String()
	if !strings.Contains(respBody, "ValidationError") {
		t.Errorf("expected ValidationError\nbody: %s", respBody)
	}
}

// ---- Test 21: UPSERT creates a record if it does not exist ----

func TestRoute53_UPSERT_CreatesNewRecord(t *testing.T) {
	handler := newR53Gateway(t)

	id := mustCreateZone(t, handler, "upcreate.example.com", "upcreate-ref-1")

	changeBody := `<?xml version="1.0" encoding="UTF-8"?>
<ChangeResourceRecordSetsRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeBatch>
    <Changes>
      <Change>
        <Action>UPSERT</Action>
        <ResourceRecordSet>
          <Name>new.upcreate.example.com</Name>
          <Type>A</Type>
          <TTL>300</TTL>
          <ResourceRecords>
            <ResourceRecord><Value>10.20.30.40</Value></ResourceRecord>
          </ResourceRecords>
        </ResourceRecordSet>
      </Change>
    </Changes>
  </ChangeBatch>
</ChangeResourceRecordSetsRequest>`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone/"+id+"/rrset", changeBody))
	if w.Code != http.StatusOK {
		t.Fatalf("UPSERT new record: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}

	// Verify it was created.
	wl := httptest.NewRecorder()
	handler.ServeHTTP(wl, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone/"+id+"/rrset", ""))
	if !strings.Contains(wl.Body.String(), "10.20.30.40") {
		t.Errorf("UPSERT should have created the record\nbody: %s", wl.Body.String())
	}
}

// ---- Test 22: CreateHostedZone response structure ----

func TestRoute53_CreateHostedZone_ResponseStructure(t *testing.T) {
	handler := newR53Gateway(t)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone",
		createZoneBody("structure.example.com", "struct-ref-1")))

	if w.Code != http.StatusCreated {
		t.Fatalf("CreateHostedZone: expected 201, got %d\nbody: %s", w.Code, w.Body.String())
	}

	var resp struct {
		HostedZone struct {
			Id              string `xml:"Id"`
			Name            string `xml:"Name"`
			CallerReference string `xml:"CallerReference"`
		} `xml:"HostedZone"`
		ChangeInfo struct {
			Id          string `xml:"Id"`
			Status      string `xml:"Status"`
			SubmittedAt string `xml:"SubmittedAt"`
		} `xml:"ChangeInfo"`
		DelegationSet struct {
			NameServers []string `xml:"NameServers>NameServer"`
		} `xml:"DelegationSet"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.HostedZone.Id == "" {
		t.Error("HostedZone.Id should not be empty")
	}
	if !strings.HasPrefix(resp.HostedZone.Id, "/hostedzone/") {
		t.Errorf("HostedZone.Id should start with /hostedzone/, got %s", resp.HostedZone.Id)
	}
	if !strings.Contains(resp.HostedZone.Name, "structure.example.com") {
		t.Errorf("expected zone name in response, got %s", resp.HostedZone.Name)
	}
	if resp.HostedZone.CallerReference != "struct-ref-1" {
		t.Errorf("expected CallerReference struct-ref-1, got %s", resp.HostedZone.CallerReference)
	}
	if resp.ChangeInfo.Status != "INSYNC" {
		t.Errorf("expected ChangeInfo.Status INSYNC, got %s", resp.ChangeInfo.Status)
	}
	if resp.ChangeInfo.SubmittedAt == "" {
		t.Error("ChangeInfo.SubmittedAt should not be empty")
	}
	if !strings.HasPrefix(resp.ChangeInfo.Id, "/change/") {
		t.Errorf("ChangeInfo.Id should start with /change/, got %s", resp.ChangeInfo.Id)
	}
	if len(resp.DelegationSet.NameServers) == 0 {
		t.Error("DelegationSet should have at least one NameServer")
	}
}

// ---- Test 23: ListHostedZones returns proper XML structure ----

func TestRoute53_ListHostedZones_Structure(t *testing.T) {
	handler := newR53Gateway(t)

	// Start with no zones — list should return empty.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone", ""))
	if w.Code != http.StatusOK {
		t.Fatalf("ListHostedZones (empty): expected 200, got %d", w.Code)
	}

	var resp struct {
		HostedZones []struct {
			Id   string `xml:"Id"`
			Name string `xml:"Name"`
		} `xml:"HostedZones>HostedZone"`
		IsTruncated bool   `xml:"IsTruncated"`
		MaxItems    string `xml:"MaxItems"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.HostedZones) != 0 {
		t.Errorf("expected 0 zones, got %d", len(resp.HostedZones))
	}
	if resp.IsTruncated {
		t.Error("expected IsTruncated=false for empty list")
	}
	if resp.MaxItems == "" {
		t.Error("expected MaxItems to be set")
	}

	// Create a zone and verify it appears.
	mustCreateZone(t, handler, "listed.example.com", "listed-ref-1")

	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, r53Req(t, http.MethodGet, "/2013-04-01/hostedzone", ""))

	var resp2 struct {
		HostedZones []struct {
			Id   string `xml:"Id"`
			Name string `xml:"Name"`
		} `xml:"HostedZones>HostedZone"`
	}
	if err := xml.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp2.HostedZones) != 1 {
		t.Errorf("expected 1 zone, got %d", len(resp2.HostedZones))
	}
}

// ---- Test 24: DeleteHostedZone response structure ----

func TestRoute53_DeleteHostedZone_ResponseStructure(t *testing.T) {
	handler := newR53Gateway(t)

	id := mustCreateZone(t, handler, "delstruct.example.com", "delstruct-ref-1")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodDelete, "/2013-04-01/hostedzone/"+id, ""))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteHostedZone: expected 200, got %d", w.Code)
	}

	var resp struct {
		ChangeInfo struct {
			Id          string `xml:"Id"`
			Status      string `xml:"Status"`
			SubmittedAt string `xml:"SubmittedAt"`
		} `xml:"ChangeInfo"`
	}
	if err := xml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ChangeInfo.Status != "INSYNC" {
		t.Errorf("expected Status INSYNC, got %s", resp.ChangeInfo.Status)
	}
	if !strings.HasPrefix(resp.ChangeInfo.Id, "/change/") {
		t.Errorf("expected ChangeInfo.Id to start with /change/, got %s", resp.ChangeInfo.Id)
	}
	if resp.ChangeInfo.SubmittedAt == "" {
		t.Error("expected SubmittedAt to be set")
	}
}

// ---- Test 25: Error — ChangeResourceRecordSets with malformed XML ----

func TestRoute53_Error_ChangeRecords_MalformedXML(t *testing.T) {
	handler := newR53Gateway(t)

	id := mustCreateZone(t, handler, "badxml.example.com", "badxml-ref-1")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone/"+id+"/rrset", "not xml"))
	if w.Code == http.StatusOK {
		t.Fatalf("ChangeRecords (malformed XML): expected error, got 200")
	}

	body := w.Body.String()
	if !strings.Contains(body, "ValidationError") {
		t.Errorf("expected ValidationError\nbody: %s", body)
	}
}

// ---- Test 26: Error — DELETE non-existent record returns InvalidChangeBatch ----

func TestRoute53_Error_InvalidChangeBatch_DeleteNonexistent(t *testing.T) {
	handler := newR53Gateway(t)

	id := mustCreateZone(t, handler, "delerr.example.com", "delerr-ref-1")

	changeBody := `<?xml version="1.0" encoding="UTF-8"?>
<ChangeResourceRecordSetsRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/">
  <ChangeBatch>
    <Changes>
      <Change>
        <Action>DELETE</Action>
        <ResourceRecordSet>
          <Name>ghost.delerr.example.com</Name>
          <Type>A</Type>
          <TTL>300</TTL>
          <ResourceRecords>
            <ResourceRecord><Value>9.9.9.9</Value></ResourceRecord>
          </ResourceRecords>
        </ResourceRecordSet>
      </Change>
    </Changes>
  </ChangeBatch>
</ChangeResourceRecordSetsRequest>`

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r53Req(t, http.MethodPost, "/2013-04-01/hostedzone/"+id+"/rrset", changeBody))
	if w.Code == http.StatusOK {
		t.Fatalf("DELETE nonexistent record: expected error, got 200")
	}

	body := w.Body.String()
	if !strings.Contains(body, "InvalidChangeBatch") {
		t.Errorf("expected InvalidChangeBatch error code\nbody: %s", body)
	}
}

