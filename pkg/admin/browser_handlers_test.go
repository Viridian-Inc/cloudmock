package admin_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"

	"github.com/Viridian-Inc/cloudmock/pkg/admin"
	apigwsvc "github.com/Viridian-Inc/cloudmock/services/apigateway"
	ebsvc "github.com/Viridian-Inc/cloudmock/services/eventbridge"
	r53svc "github.com/Viridian-Inc/cloudmock/services/route53"
	snssvc "github.com/Viridian-Inc/cloudmock/services/sns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBrowserAPI(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()

	reg.Register(snssvc.New(cfg.AccountID, cfg.Region))
	reg.Register(ebsvc.New(cfg.AccountID, cfg.Region))
	reg.Register(apigwsvc.New(cfg.AccountID, cfg.Region))
	reg.Register(r53svc.New(cfg.AccountID, cfg.Region))

	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	return admin.New(cfg, reg, rl, rs)
}

// snsAction invokes an SNS query-style action through the gateway wrapping the
// admin API so the SNS service has data to inspect.
func snsAction(t *testing.T, gw http.Handler, action string, form map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	body := ""
	first := true
	for k, v := range form {
		if !first {
			body += "&"
		}
		first = false
		body += k + "=" + v
	}
	req := httptest.NewRequest(http.MethodPost, "/?Action="+action, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Host", "sns.us-east-1.amazonaws.com")
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/sns/aws4_request, SignedHeaders=host, Signature=abc")
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)
	return w
}

func TestBrowserSNS_Empty(t *testing.T) {
	api := newBrowserAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/api/sns/topics", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var out struct {
		Topics []any `json:"topics"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Empty(t, out.Topics)
}

func TestBrowserEventBridge_Empty(t *testing.T) {
	api := newBrowserAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/api/eventbridge/buses", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var out struct {
		Buses []struct {
			Name string `json:"name"`
		} `json:"buses"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	// EventBridge seeds a default bus at construction time.
	require.Len(t, out.Buses, 1)
	assert.Equal(t, "default", out.Buses[0].Name)
}

func TestBrowserAPIGateway_Empty(t *testing.T) {
	api := newBrowserAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/api/apigateway/apis", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var out struct {
		APIs []any `json:"apis"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Empty(t, out.APIs)
}

func TestBrowserRoute53_Empty(t *testing.T) {
	api := newBrowserAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/api/route53/zones", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var out struct {
		Zones []any `json:"zones"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Empty(t, out.Zones)
}

func TestBrowserAnomalies_NoDetector(t *testing.T) {
	api := newBrowserAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/api/browser/anomalies", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var out struct {
		Anomalies []any `json:"anomalies"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Empty(t, out.Anomalies)
}

func TestBrowserLogs_NoBuffer(t *testing.T) {
	api := newBrowserAPI(t)
	req := httptest.NewRequest(http.MethodGet, "/api/browser/logs", nil)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var out struct {
		Logs []any `json:"logs"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Empty(t, out.Logs)
}

func TestBrowserSNS_ReturnsTopicsFromStore(t *testing.T) {
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	snsInst := snssvc.New(cfg.AccountID, cfg.Region)
	reg.Register(snsInst)
	reg.Register(ebsvc.New(cfg.AccountID, cfg.Region))
	reg.Register(apigwsvc.New(cfg.AccountID, cfg.Region))
	reg.Register(r53svc.New(cfg.AccountID, cfg.Region))
	rl := gateway.NewRequestLog(100)
	rs := gateway.NewRequestStats()
	adminAPI := admin.New(cfg, reg, rl, rs)
	gw := gateway.New(cfg, reg)

	// Create a topic via the gateway.
	resp := snsAction(t, gw, "CreateTopic", map[string]string{"Name": "browser-test"})
	require.Equal(t, http.StatusOK, resp.Code, "body: %s", resp.Body.String())

	// Query the browser endpoint.
	req := httptest.NewRequest(http.MethodGet, "/api/sns/topics", nil)
	w := httptest.NewRecorder()
	adminAPI.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var out struct {
		Topics []struct {
			TopicArn          string `json:"topicArn"`
			Name              string `json:"name"`
			SubscriptionCount int    `json:"subscriptionCount"`
		} `json:"topics"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	require.Len(t, out.Topics, 1)
	assert.Equal(t, "browser-test", out.Topics[0].Name)
	assert.Equal(t, 0, out.Topics[0].SubscriptionCount)
}
