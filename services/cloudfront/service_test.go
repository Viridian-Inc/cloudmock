package cloudfront_test

import (
	"bytes"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	cfsvc "github.com/Viridian-Inc/cloudmock/services/cloudfront"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCFGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(cfsvc.New(cfg.AccountID, cfg.Region))
	return gateway.New(cfg, reg)
}

func cfReq(t *testing.T, method, path string, body []byte) *http.Request {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader([]byte{})
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "text/xml")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/cloudfront/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

func cfReqWithHeader(t *testing.T, method, path string, body []byte, key, value string) *http.Request {
	t.Helper()
	req := cfReq(t, method, path, body)
	req.Header.Set(key, value)
	return req
}

func cfBody(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	return w.Body.String()
}

func extractXMLVal(t *testing.T, xmlBody, tag string) string {
	t.Helper()
	start := strings.Index(xmlBody, "<"+tag+">")
	if start == -1 {
		t.Fatalf("tag <%s> not found in:\n%s", tag, xmlBody)
	}
	start += len("<" + tag + ">")
	end := strings.Index(xmlBody[start:], "</"+tag+">")
	if end == -1 {
		t.Fatalf("closing </%s> not found", tag)
	}
	return xmlBody[start : start+end]
}

func minDistributionConfig(callerRef, comment string) []byte {
	return minDistributionConfigEnabled(callerRef, comment, true)
}

func minDistributionConfigEnabled(callerRef, comment string, enabled bool) []byte {
	type Origins struct {
		XMLName  xml.Name `xml:"Origins"`
		Quantity int      `xml:"Quantity"`
		Items    []struct {
			XMLName    xml.Name `xml:"Origin"`
			Id         string   `xml:"Id"`
			DomainName string   `xml:"DomainName"`
		} `xml:"Items>Origin"`
	}
	type DefaultCacheBehavior struct {
		XMLName              xml.Name `xml:"DefaultCacheBehavior"`
		TargetOriginId       string   `xml:"TargetOriginId"`
		ViewerProtocolPolicy string   `xml:"ViewerProtocolPolicy"`
	}
	type Cfg struct {
		XMLName              xml.Name             `xml:"DistributionConfig"`
		CallerReference      string               `xml:"CallerReference"`
		Comment              string               `xml:"Comment"`
		Enabled              bool                 `xml:"Enabled"`
		Origins              Origins              `xml:"Origins"`
		DefaultCacheBehavior DefaultCacheBehavior `xml:"DefaultCacheBehavior"`
	}
	c := Cfg{
		CallerReference: callerRef,
		Comment:         comment,
		Enabled:         enabled,
		Origins: Origins{
			Quantity: 1,
			Items: []struct {
				XMLName    xml.Name `xml:"Origin"`
				Id         string   `xml:"Id"`
				DomainName string   `xml:"DomainName"`
			}{
				{Id: "origin-1", DomainName: "example.com"},
			},
		},
		DefaultCacheBehavior: DefaultCacheBehavior{
			TargetOriginId:       "origin-1",
			ViewerProtocolPolicy: "redirect-to-https",
		},
	}
	data, _ := xml.Marshal(c)
	return data
}

// ---- CreateDistribution ----

func TestCF_CreateDistribution(t *testing.T) {
	h := newCFGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfig("ref1", "test dist")))
	require.Equal(t, http.StatusCreated, w.Code, cfBody(t, w))
	body := cfBody(t, w)
	assert.Contains(t, body, "<Id>")
	assert.Contains(t, body, "cloudfront.net")
	assert.Contains(t, body, "InProgress")
	assert.NotEmpty(t, w.Header().Get("ETag"))
	assert.Contains(t, w.Header().Get("Location"), "/2020-05-31/distribution/")
}

func TestCF_CreateDistribution_CallerReferenceDedup(t *testing.T) {
	h := newCFGateway(t)

	// First create
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfig("dedup-ref", "dist1")))
	require.Equal(t, http.StatusCreated, w1.Code)
	id1 := extractXMLVal(t, cfBody(t, w1), "Id")

	// Same CallerReference should return same distribution
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfig("dedup-ref", "dist2")))
	require.Equal(t, http.StatusCreated, w2.Code)
	id2 := extractXMLVal(t, cfBody(t, w2), "Id")

	assert.Equal(t, id1, id2, "same CallerReference should return same distribution")
}

// ---- GetDistribution ----

func TestCF_GetDistribution(t *testing.T) {
	h := newCFGateway(t)
	// Create
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfig("ref-get", "get test")))
	require.Equal(t, http.StatusCreated, w1.Code)
	distID := extractXMLVal(t, cfBody(t, w1), "Id")

	// Get
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, cfReq(t, http.MethodGet, "/2020-05-31/distribution/"+distID, nil))
	require.Equal(t, http.StatusOK, w2.Code)
	body := cfBody(t, w2)
	assert.Contains(t, body, distID)
	assert.Contains(t, body, "get test")
	assert.NotEmpty(t, w2.Header().Get("ETag"))
}

func TestCF_GetDistribution_NotFound(t *testing.T) {
	h := newCFGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodGet, "/2020-05-31/distribution/ENONEXISTENT", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, cfBody(t, w), "NoSuchDistribution")
}

// ---- ListDistributions ----

func TestCF_ListDistributions(t *testing.T) {
	h := newCFGateway(t)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, cfReq(t, http.MethodPost, "/2020-05-31/distribution",
			minDistributionConfig("list-ref-"+strings.Repeat("x", i), "list dist")))
		require.Equal(t, http.StatusCreated, w.Code)
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodGet, "/2020-05-31/distribution", nil))
	require.Equal(t, http.StatusOK, w.Code)
	body := cfBody(t, w)
	assert.Contains(t, body, "<Quantity>2</Quantity>")
}

// ---- UpdateDistribution ----

func TestCF_UpdateDistribution(t *testing.T) {
	h := newCFGateway(t)
	// Create
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfig("upd-ref", "original")))
	require.Equal(t, http.StatusCreated, w1.Code)
	distID := extractXMLVal(t, cfBody(t, w1), "Id")
	etag := w1.Header().Get("ETag")

	// Update with If-Match
	w2 := httptest.NewRecorder()
	req := cfReqWithHeader(t, http.MethodPut, "/2020-05-31/distribution/"+distID+"/config",
		minDistributionConfig("upd-ref", "updated comment"), "If-Match", etag)
	h.ServeHTTP(w2, req)
	require.Equal(t, http.StatusOK, w2.Code, cfBody(t, w2))
	body := cfBody(t, w2)
	assert.Contains(t, body, "updated comment")
	assert.NotEmpty(t, w2.Header().Get("ETag"))
	assert.NotEqual(t, etag, w2.Header().Get("ETag"), "ETag should change on update")
}

func TestCF_UpdateDistribution_InvalidIfMatch(t *testing.T) {
	h := newCFGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfig("ifm-ref", "dist")))
	require.Equal(t, http.StatusCreated, w1.Code)
	distID := extractXMLVal(t, cfBody(t, w1), "Id")

	w2 := httptest.NewRecorder()
	req := cfReqWithHeader(t, http.MethodPut, "/2020-05-31/distribution/"+distID+"/config",
		minDistributionConfig("ifm-ref", "updated"), "If-Match", "Ewrong-etag")
	h.ServeHTTP(w2, req)
	assert.Equal(t, http.StatusPreconditionFailed, w2.Code)
	assert.Contains(t, cfBody(t, w2), "InvalidIfMatchVersion")
}

func TestCF_UpdateDistribution_NotFound(t *testing.T) {
	h := newCFGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodPut, "/2020-05-31/distribution/ENOTFOUND/config",
		minDistributionConfig("ref", "comment")))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---- DeleteDistribution ----

func TestCF_DeleteDistribution_MustBeDisabled(t *testing.T) {
	h := newCFGateway(t)
	// Create enabled distribution
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfig("del-ena-ref", "enabled dist")))
	require.Equal(t, http.StatusCreated, w1.Code)
	distID := extractXMLVal(t, cfBody(t, w1), "Id")
	etag := w1.Header().Get("ETag")

	// Try to delete enabled distribution — should fail
	w2 := httptest.NewRecorder()
	req := cfReqWithHeader(t, http.MethodDelete, "/2020-05-31/distribution/"+distID, nil, "If-Match", etag)
	h.ServeHTTP(w2, req)
	assert.Equal(t, http.StatusConflict, w2.Code)
	assert.Contains(t, cfBody(t, w2), "DistributionNotDisabled")
}

func TestCF_DeleteDistribution_AfterDisable(t *testing.T) {
	h := newCFGateway(t)
	// Create disabled distribution
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfigEnabled("del-dis-ref", "dis dist", false)))
	require.Equal(t, http.StatusCreated, w1.Code)
	distID := extractXMLVal(t, cfBody(t, w1), "Id")
	etag := w1.Header().Get("ETag")

	// Delete disabled distribution — should succeed
	w2 := httptest.NewRecorder()
	req := cfReqWithHeader(t, http.MethodDelete, "/2020-05-31/distribution/"+distID, nil, "If-Match", etag)
	h.ServeHTTP(w2, req)
	require.Equal(t, http.StatusNoContent, w2.Code)

	// Verify gone
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, cfReq(t, http.MethodGet, "/2020-05-31/distribution/"+distID, nil))
	assert.Equal(t, http.StatusNotFound, w3.Code)
}

func TestCF_DeleteDistribution(t *testing.T) {
	h := newCFGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfigEnabled("ref-del", "del test", false)))
	require.Equal(t, http.StatusCreated, w1.Code)
	distID := extractXMLVal(t, cfBody(t, w1), "Id")

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, cfReq(t, http.MethodDelete, "/2020-05-31/distribution/"+distID, nil))
	require.Equal(t, http.StatusNoContent, w2.Code)

	// Verify gone
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, cfReq(t, http.MethodGet, "/2020-05-31/distribution/"+distID, nil))
	assert.Equal(t, http.StatusNotFound, w3.Code)
}

func TestCF_DeleteDistribution_NotFound(t *testing.T) {
	h := newCFGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodDelete, "/2020-05-31/distribution/ENOPE", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCF_DeleteDistribution_InvalidIfMatch(t *testing.T) {
	h := newCFGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfigEnabled("del-ifm-ref", "dist", false)))
	require.Equal(t, http.StatusCreated, w1.Code)
	distID := extractXMLVal(t, cfBody(t, w1), "Id")

	w2 := httptest.NewRecorder()
	req := cfReqWithHeader(t, http.MethodDelete, "/2020-05-31/distribution/"+distID, nil, "If-Match", "Ewrong")
	h.ServeHTTP(w2, req)
	assert.Equal(t, http.StatusPreconditionFailed, w2.Code)
}

// ---- Invalidation ----

func TestCF_CreateInvalidation(t *testing.T) {
	h := newCFGateway(t)
	// Create distribution
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfig("inv-ref", "inv test")))
	require.Equal(t, http.StatusCreated, w1.Code)
	distID := extractXMLVal(t, cfBody(t, w1), "Id")

	// Create invalidation
	type Paths struct {
		Quantity int      `xml:"Quantity"`
		Items    []string `xml:"Items>Path"`
	}
	type Batch struct {
		XMLName         xml.Name `xml:"InvalidationBatch"`
		CallerReference string   `xml:"CallerReference"`
		Paths           Paths    `xml:"Paths"`
	}
	batch := Batch{
		CallerReference: "inv-1",
		Paths:           Paths{Quantity: 2, Items: []string{"/images/*", "/css/*"}},
	}
	batchData, _ := xml.Marshal(batch)

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, cfReq(t, http.MethodPost, "/2020-05-31/distribution/"+distID+"/invalidation", batchData))
	require.Equal(t, http.StatusCreated, w2.Code)
	body := cfBody(t, w2)
	assert.Contains(t, body, "<Id>")
	assert.Contains(t, body, "InProgress")
}

func TestCF_GetInvalidation(t *testing.T) {
	h := newCFGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfig("ginv-ref", "ginv")))
	require.Equal(t, http.StatusCreated, w1.Code)
	distID := extractXMLVal(t, cfBody(t, w1), "Id")

	type Paths struct {
		Quantity int      `xml:"Quantity"`
		Items    []string `xml:"Items>Path"`
	}
	type Batch struct {
		XMLName         xml.Name `xml:"InvalidationBatch"`
		CallerReference string   `xml:"CallerReference"`
		Paths           Paths    `xml:"Paths"`
	}
	batchData, _ := xml.Marshal(Batch{CallerReference: "ginv-1", Paths: Paths{Quantity: 1, Items: []string{"/*"}}})

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, cfReq(t, http.MethodPost, "/2020-05-31/distribution/"+distID+"/invalidation", batchData))
	require.Equal(t, http.StatusCreated, w2.Code)
	invID := extractXMLVal(t, cfBody(t, w2), "Id")

	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, cfReq(t, http.MethodGet, "/2020-05-31/distribution/"+distID+"/invalidation/"+invID, nil))
	require.Equal(t, http.StatusOK, w3.Code)
	assert.Contains(t, cfBody(t, w3), invID)
}

func TestCF_GetInvalidation_NotFound(t *testing.T) {
	h := newCFGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfig("nfinv-ref", "nf")))
	require.Equal(t, http.StatusCreated, w1.Code)
	distID := extractXMLVal(t, cfBody(t, w1), "Id")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodGet, "/2020-05-31/distribution/"+distID+"/invalidation/INOPE", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCF_ListInvalidations(t *testing.T) {
	h := newCFGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfig("linv-ref", "linv")))
	require.Equal(t, http.StatusCreated, w1.Code)
	distID := extractXMLVal(t, cfBody(t, w1), "Id")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodGet, "/2020-05-31/distribution/"+distID+"/invalidation", nil))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, cfBody(t, w), "<Quantity>0</Quantity>")
}

func TestCF_CreateInvalidation_NoDistribution(t *testing.T) {
	h := newCFGateway(t)
	type Paths struct {
		Quantity int      `xml:"Quantity"`
		Items    []string `xml:"Items>Path"`
	}
	type Batch struct {
		XMLName         xml.Name `xml:"InvalidationBatch"`
		CallerReference string   `xml:"CallerReference"`
		Paths           Paths    `xml:"Paths"`
	}
	batchData, _ := xml.Marshal(Batch{CallerReference: "cr", Paths: Paths{Quantity: 1, Items: []string{"/*"}}})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodPost, "/2020-05-31/distribution/ENOTFOUND/invalidation", batchData))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---- Cache Policies ----

func cachePolicyConfig(name, comment string) []byte {
	type Cfg struct {
		XMLName    xml.Name `xml:"CachePolicyConfig"`
		Name       string   `xml:"Name"`
		Comment    string   `xml:"Comment"`
		DefaultTTL int64    `xml:"DefaultTTL"`
		MaxTTL     int64    `xml:"MaxTTL"`
		MinTTL     int64    `xml:"MinTTL"`
	}
	data, _ := xml.Marshal(Cfg{Name: name, Comment: comment, DefaultTTL: 86400, MaxTTL: 31536000, MinTTL: 0})
	return data
}

func TestCF_CreateCachePolicy(t *testing.T) {
	h := newCFGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodPost, "/2020-05-31/cache-policy", cachePolicyConfig("my-policy", "test")))
	require.Equal(t, http.StatusCreated, w.Code, cfBody(t, w))
	body := cfBody(t, w)
	assert.Contains(t, body, "<Id>")
	assert.Contains(t, body, "my-policy")
	assert.NotEmpty(t, w.Header().Get("ETag"))
	assert.Contains(t, w.Header().Get("Location"), "/2020-05-31/cache-policy/")
}

func TestCF_CreateCachePolicy_Duplicate(t *testing.T) {
	h := newCFGateway(t)
	h.ServeHTTP(httptest.NewRecorder(), cfReq(t, http.MethodPost, "/2020-05-31/cache-policy", cachePolicyConfig("dup-policy", "")))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodPost, "/2020-05-31/cache-policy", cachePolicyConfig("dup-policy", "")))
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestCF_GetCachePolicy(t *testing.T) {
	h := newCFGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/cache-policy", cachePolicyConfig("get-pol", "comment")))
	require.Equal(t, http.StatusCreated, w1.Code)
	id := extractXMLVal(t, cfBody(t, w1), "Id")

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, cfReq(t, http.MethodGet, "/2020-05-31/cache-policy/"+id, nil))
	require.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, cfBody(t, w2), "get-pol")
	assert.NotEmpty(t, w2.Header().Get("ETag"))
}

func TestCF_GetCachePolicy_NotFound(t *testing.T) {
	h := newCFGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodGet, "/2020-05-31/cache-policy/nonexistent-id", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, cfBody(t, w), "NoSuchCachePolicy")
}

func TestCF_ListCachePolicies(t *testing.T) {
	h := newCFGateway(t)
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, cfReq(t, http.MethodPost, "/2020-05-31/cache-policy",
			cachePolicyConfig("list-pol-"+strings.Repeat("x", i+1), "")))
		require.Equal(t, http.StatusCreated, w.Code)
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodGet, "/2020-05-31/cache-policy", nil))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, cfBody(t, w), "<Quantity>3</Quantity>")
}

func TestCF_UpdateCachePolicy(t *testing.T) {
	h := newCFGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/cache-policy", cachePolicyConfig("upd-pol", "original")))
	require.Equal(t, http.StatusCreated, w1.Code)
	id := extractXMLVal(t, cfBody(t, w1), "Id")
	etag := w1.Header().Get("ETag")

	w2 := httptest.NewRecorder()
	req := cfReqWithHeader(t, http.MethodPut, "/2020-05-31/cache-policy/"+id,
		cachePolicyConfig("upd-pol", "updated"), "If-Match", etag)
	h.ServeHTTP(w2, req)
	require.Equal(t, http.StatusOK, w2.Code, cfBody(t, w2))
	assert.Contains(t, cfBody(t, w2), "upd-pol")
	newETag := w2.Header().Get("ETag")
	assert.NotEqual(t, etag, newETag)
}

func TestCF_DeleteCachePolicy(t *testing.T) {
	h := newCFGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/cache-policy", cachePolicyConfig("del-pol", "")))
	require.Equal(t, http.StatusCreated, w1.Code)
	id := extractXMLVal(t, cfBody(t, w1), "Id")
	etag := w1.Header().Get("ETag")

	w2 := httptest.NewRecorder()
	req := cfReqWithHeader(t, http.MethodDelete, "/2020-05-31/cache-policy/"+id, nil, "If-Match", etag)
	h.ServeHTTP(w2, req)
	require.Equal(t, http.StatusNoContent, w2.Code)

	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, cfReq(t, http.MethodGet, "/2020-05-31/cache-policy/"+id, nil))
	assert.Equal(t, http.StatusNotFound, w3.Code)
}

func TestCF_DeleteCachePolicy_InvalidIfMatch(t *testing.T) {
	h := newCFGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/cache-policy", cachePolicyConfig("del-ifm-pol", "")))
	require.Equal(t, http.StatusCreated, w1.Code)
	id := extractXMLVal(t, cfBody(t, w1), "Id")

	w2 := httptest.NewRecorder()
	req := cfReqWithHeader(t, http.MethodDelete, "/2020-05-31/cache-policy/"+id, nil, "If-Match", "Ewrong")
	h.ServeHTTP(w2, req)
	assert.Equal(t, http.StatusPreconditionFailed, w2.Code)
}

// ---- Origin Request Policies ----

func originRequestPolicyConfig(name, comment string) []byte {
	type Cfg struct {
		XMLName xml.Name `xml:"OriginRequestPolicyConfig"`
		Name    string   `xml:"Name"`
		Comment string   `xml:"Comment"`
	}
	data, _ := xml.Marshal(Cfg{Name: name, Comment: comment})
	return data
}

func TestCF_CreateOriginRequestPolicy(t *testing.T) {
	h := newCFGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodPost, "/2020-05-31/origin-request-policy", originRequestPolicyConfig("my-orp", "test")))
	require.Equal(t, http.StatusCreated, w.Code, cfBody(t, w))
	body := cfBody(t, w)
	assert.Contains(t, body, "<Id>")
	assert.Contains(t, body, "my-orp")
	assert.NotEmpty(t, w.Header().Get("ETag"))
}

func TestCF_CreateOriginRequestPolicy_Duplicate(t *testing.T) {
	h := newCFGateway(t)
	h.ServeHTTP(httptest.NewRecorder(), cfReq(t, http.MethodPost, "/2020-05-31/origin-request-policy", originRequestPolicyConfig("dup-orp", "")))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodPost, "/2020-05-31/origin-request-policy", originRequestPolicyConfig("dup-orp", "")))
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestCF_GetOriginRequestPolicy(t *testing.T) {
	h := newCFGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/origin-request-policy", originRequestPolicyConfig("get-orp", "comment")))
	require.Equal(t, http.StatusCreated, w1.Code)
	id := extractXMLVal(t, cfBody(t, w1), "Id")

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, cfReq(t, http.MethodGet, "/2020-05-31/origin-request-policy/"+id, nil))
	require.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, cfBody(t, w2), "get-orp")
}

func TestCF_GetOriginRequestPolicy_NotFound(t *testing.T) {
	h := newCFGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodGet, "/2020-05-31/origin-request-policy/nonexistent", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCF_ListOriginRequestPolicies(t *testing.T) {
	h := newCFGateway(t)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, cfReq(t, http.MethodPost, "/2020-05-31/origin-request-policy",
			originRequestPolicyConfig("list-orp-"+strings.Repeat("y", i+1), "")))
		require.Equal(t, http.StatusCreated, w.Code)
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodGet, "/2020-05-31/origin-request-policy", nil))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, cfBody(t, w), "<Quantity>2</Quantity>")
}

func TestCF_DeleteOriginRequestPolicy(t *testing.T) {
	h := newCFGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/origin-request-policy", originRequestPolicyConfig("del-orp", "")))
	require.Equal(t, http.StatusCreated, w1.Code)
	id := extractXMLVal(t, cfBody(t, w1), "Id")
	etag := w1.Header().Get("ETag")

	w2 := httptest.NewRecorder()
	req := cfReqWithHeader(t, http.MethodDelete, "/2020-05-31/origin-request-policy/"+id, nil, "If-Match", etag)
	h.ServeHTTP(w2, req)
	require.Equal(t, http.StatusNoContent, w2.Code)

	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, cfReq(t, http.MethodGet, "/2020-05-31/origin-request-policy/"+id, nil))
	assert.Equal(t, http.StatusNotFound, w3.Code)
}

// ---- Functions ----

func functionCreateBody(name, comment, runtime string) []byte {
	type FnCfg struct {
		Comment string `xml:"Comment"`
		Runtime string `xml:"Runtime"`
	}
	type Req struct {
		XMLName        xml.Name `xml:"CreateFunctionRequest"`
		Name           string   `xml:"Name"`
		FunctionConfig FnCfg    `xml:"FunctionConfig"`
		FunctionCode   []byte   `xml:"FunctionCode"`
	}
	data, _ := xml.Marshal(Req{
		Name: name,
		FunctionConfig: FnCfg{Comment: comment, Runtime: runtime},
		FunctionCode:   []byte("function handler(event) { return event.request; }"),
	})
	return data
}

func TestCF_CreateFunction(t *testing.T) {
	h := newCFGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodPost, "/2020-05-31/function", functionCreateBody("my-fn", "test func", "cloudfront-js-2.0")))
	require.Equal(t, http.StatusCreated, w.Code, cfBody(t, w))
	body := cfBody(t, w)
	assert.Contains(t, body, "my-fn")
	assert.Contains(t, body, "DEVELOPMENT")
	assert.NotEmpty(t, w.Header().Get("ETag"))
}

func TestCF_CreateFunction_Duplicate(t *testing.T) {
	h := newCFGateway(t)
	h.ServeHTTP(httptest.NewRecorder(), cfReq(t, http.MethodPost, "/2020-05-31/function", functionCreateBody("dup-fn", "", "cloudfront-js-1.0")))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodPost, "/2020-05-31/function", functionCreateBody("dup-fn", "", "cloudfront-js-1.0")))
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestCF_GetFunction(t *testing.T) {
	h := newCFGateway(t)
	h.ServeHTTP(httptest.NewRecorder(), cfReq(t, http.MethodPost, "/2020-05-31/function", functionCreateBody("get-fn", "comment", "cloudfront-js-2.0")))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodGet, "/2020-05-31/function/get-fn", nil))
	require.Equal(t, http.StatusOK, w.Code)
	body := cfBody(t, w)
	assert.Contains(t, body, "get-fn")
	assert.Contains(t, body, "cloudfront-js-2.0")
}

func TestCF_GetFunction_NotFound(t *testing.T) {
	h := newCFGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodGet, "/2020-05-31/function/nonexistent-fn", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, cfBody(t, w), "NoSuchFunctionExists")
}

func TestCF_ListFunctions(t *testing.T) {
	h := newCFGateway(t)
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, cfReq(t, http.MethodPost, "/2020-05-31/function",
			functionCreateBody("list-fn-"+strings.Repeat("z", i+1), "", "cloudfront-js-1.0")))
		require.Equal(t, http.StatusCreated, w.Code)
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodGet, "/2020-05-31/function", nil))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, cfBody(t, w), "<Quantity>3</Quantity>")
}

func TestCF_UpdateFunction(t *testing.T) {
	h := newCFGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/function", functionCreateBody("upd-fn", "original", "cloudfront-js-1.0")))
	require.Equal(t, http.StatusCreated, w1.Code)
	etag := w1.Header().Get("ETag")

	type FnCfg struct {
		Comment string `xml:"Comment"`
		Runtime string `xml:"Runtime"`
	}
	type UpdateReq struct {
		XMLName        xml.Name `xml:"UpdateFunctionRequest"`
		FunctionConfig FnCfg    `xml:"FunctionConfig"`
		FunctionCode   []byte   `xml:"FunctionCode"`
	}
	updateBody, _ := xml.Marshal(UpdateReq{
		FunctionConfig: FnCfg{Comment: "updated comment", Runtime: "cloudfront-js-2.0"},
		FunctionCode:   []byte("function handler(event) { return event.response; }"),
	})

	w2 := httptest.NewRecorder()
	req := cfReqWithHeader(t, http.MethodPut, "/2020-05-31/function/upd-fn", updateBody, "If-Match", etag)
	h.ServeHTTP(w2, req)
	require.Equal(t, http.StatusOK, w2.Code, cfBody(t, w2))
	assert.Contains(t, cfBody(t, w2), "cloudfront-js-2.0")
	assert.NotEqual(t, etag, w2.Header().Get("ETag"))
}

func TestCF_DeleteFunction(t *testing.T) {
	h := newCFGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/function", functionCreateBody("del-fn", "", "cloudfront-js-1.0")))
	require.Equal(t, http.StatusCreated, w1.Code)
	etag := w1.Header().Get("ETag")

	w2 := httptest.NewRecorder()
	req := cfReqWithHeader(t, http.MethodDelete, "/2020-05-31/function/del-fn", nil, "If-Match", etag)
	h.ServeHTTP(w2, req)
	require.Equal(t, http.StatusNoContent, w2.Code)

	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, cfReq(t, http.MethodGet, "/2020-05-31/function/del-fn", nil))
	assert.Equal(t, http.StatusNotFound, w3.Code)
}

func TestCF_PublishFunction(t *testing.T) {
	h := newCFGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/function", functionCreateBody("pub-fn", "", "cloudfront-js-2.0")))
	require.Equal(t, http.StatusCreated, w1.Code)
	etag := w1.Header().Get("ETag")

	// Publish
	w2 := httptest.NewRecorder()
	req := cfReqWithHeader(t, http.MethodPost, "/2020-05-31/function/pub-fn/publish", nil, "If-Match", etag)
	h.ServeHTTP(w2, req)
	require.Equal(t, http.StatusOK, w2.Code, cfBody(t, w2))
	body := cfBody(t, w2)
	assert.Contains(t, body, "LIVE")
	assert.Contains(t, body, "DEPLOYED")
}

func TestCF_TestFunction(t *testing.T) {
	h := newCFGateway(t)
	h.ServeHTTP(httptest.NewRecorder(), cfReq(t, http.MethodPost, "/2020-05-31/function", functionCreateBody("test-fn", "", "cloudfront-js-2.0")))

	type TestReq struct {
		XMLName     xml.Name `xml:"TestFunctionRequest"`
		Stage       string   `xml:"Stage"`
		EventObject []byte   `xml:"EventObject"`
	}
	testBody, _ := xml.Marshal(TestReq{
		Stage:       "DEVELOPMENT",
		EventObject: []byte(`{"request":{"method":"GET","uri":"/test"}}`),
	})

	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodPost, "/2020-05-31/function/test-fn/test", testBody))
	require.Equal(t, http.StatusOK, w.Code, cfBody(t, w))
}

func TestCF_Function_PublishInvalidIfMatch(t *testing.T) {
	h := newCFGateway(t)
	h.ServeHTTP(httptest.NewRecorder(), cfReq(t, http.MethodPost, "/2020-05-31/function", functionCreateBody("pub-ifm-fn", "", "cloudfront-js-1.0")))

	w := httptest.NewRecorder()
	req := cfReqWithHeader(t, http.MethodPost, "/2020-05-31/function/pub-ifm-fn/publish", nil, "If-Match", "Ewrong")
	h.ServeHTTP(w, req)
	assert.Equal(t, http.StatusPreconditionFailed, w.Code)
}

// ---- Tags ----

func TestCF_TagResource(t *testing.T) {
	h := newCFGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfig("tag-ref", "tag test")))
	require.Equal(t, http.StatusCreated, w1.Code)
	body := cfBody(t, w1)
	arn := extractXMLVal(t, body, "ARN")

	type Tag struct {
		XMLName xml.Name `xml:"Tag"`
		Key     string   `xml:"Key"`
		Value   string   `xml:"Value"`
	}
	type Tags struct {
		XMLName xml.Name `xml:"Tags"`
		Items   []Tag    `xml:"Items>Tag"`
	}
	tagsData, _ := xml.Marshal(Tags{Items: []Tag{{Key: "env", Value: "staging"}}})

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, cfReq(t, http.MethodPost, "/2020-05-31/tagging?Operation=Tag&Resource="+arn, tagsData))
	require.Equal(t, http.StatusNoContent, w2.Code)

	// List tags
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, cfReq(t, http.MethodGet, "/2020-05-31/tagging?Resource="+arn, nil))
	require.Equal(t, http.StatusOK, w3.Code)
	assert.Contains(t, cfBody(t, w3), "env")
	assert.Contains(t, cfBody(t, w3), "staging")
}

func TestCF_UntagResource(t *testing.T) {
	h := newCFGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfig("untag-ref", "untag test")))
	require.Equal(t, http.StatusCreated, w1.Code)
	arn := extractXMLVal(t, cfBody(t, w1), "ARN")

	// Tag first
	type Tag struct {
		XMLName xml.Name `xml:"Tag"`
		Key     string   `xml:"Key"`
		Value   string   `xml:"Value"`
	}
	type Tags struct {
		XMLName xml.Name `xml:"Tags"`
		Items   []Tag    `xml:"Items>Tag"`
	}
	tagsData, _ := xml.Marshal(Tags{Items: []Tag{{Key: "del-key", Value: "val"}}})
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, cfReq(t, http.MethodPost, "/2020-05-31/tagging?Operation=Tag&Resource="+arn, tagsData))
	require.Equal(t, http.StatusNoContent, w2.Code)

	// Untag
	type TagKeys struct {
		XMLName xml.Name `xml:"TagKeys"`
		Items   []string `xml:"Items>Key"`
	}
	keysData, _ := xml.Marshal(TagKeys{Items: []string{"del-key"}})
	w3 := httptest.NewRecorder()
	h.ServeHTTP(w3, cfReq(t, http.MethodPost, "/2020-05-31/tagging?Operation=Untag&Resource="+arn, keysData))
	require.Equal(t, http.StatusNoContent, w3.Code)

	// Verify tag removed
	w4 := httptest.NewRecorder()
	h.ServeHTTP(w4, cfReq(t, http.MethodGet, "/2020-05-31/tagging?Resource="+arn, nil))
	require.Equal(t, http.StatusOK, w4.Code)
	assert.NotContains(t, cfBody(t, w4), "del-key")
}

// ---- ARN / domain name format ----

func TestCF_Distribution_RealisticDomain(t *testing.T) {
	h := newCFGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfig("domain-ref", "domain test")))
	require.Equal(t, http.StatusCreated, w.Code)
	body := cfBody(t, w)

	// Domain should be in the format d{13chars}.cloudfront.net
	domainName := extractXMLVal(t, body, "DomainName")
	assert.Contains(t, domainName, "cloudfront.net")
	assert.True(t, len(domainName) > len(".cloudfront.net"), "domain should have a prefix")
	assert.True(t, domainName[0] == 'd', "domain should start with 'd'")
}

func TestCF_Distribution_ARNFormat(t *testing.T) {
	h := newCFGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfig("arn-ref", "arn test")))
	require.Equal(t, http.StatusCreated, w.Code)
	arn := extractXMLVal(t, cfBody(t, w), "ARN")
	assert.Contains(t, arn, "arn:aws:cloudfront::")
	assert.Contains(t, arn, ":distribution/")
}

// ---- Not implemented path ----

func TestCF_NotImplemented(t *testing.T) {
	h := newCFGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodGet, "/2020-05-31/bogus-path", nil))
	assert.Equal(t, http.StatusNotImplemented, w.Code)
}

// ---- Tag on nonexistent resource ----

func TestCF_ListTags_NotFound(t *testing.T) {
	h := newCFGateway(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, cfReq(t, http.MethodGet, "/2020-05-31/tagging?Resource=arn:aws:cloudfront::000:distribution/ENOPE", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---- ETag round-trip on Get ----

func TestCF_GetDistribution_ETagPresent(t *testing.T) {
	h := newCFGateway(t)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, cfReq(t, http.MethodPost, "/2020-05-31/distribution", minDistributionConfig("etag-ref", "etag test")))
	require.Equal(t, http.StatusCreated, w1.Code)
	distID := extractXMLVal(t, cfBody(t, w1), "Id")

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, cfReq(t, http.MethodGet, "/2020-05-31/distribution/"+distID, nil))
	require.Equal(t, http.StatusOK, w2.Code)
	etag := w2.Header().Get("ETag")
	assert.NotEmpty(t, etag)
	assert.True(t, strings.HasPrefix(etag, "E"), "ETag should start with E")
}
