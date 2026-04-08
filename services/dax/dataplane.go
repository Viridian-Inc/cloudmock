package dax

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	dynamodbsvc "github.com/Viridian-Inc/cloudmock/services/dynamodb"
)

// DataPlane is an HTTP handler that proxies DynamoDB requests through a DAX cache.
type DataPlane struct {
	daxService *DAXService
	ddbService *dynamodbsvc.DynamoDBService
}

// NewDataPlane creates a new DAX data-plane proxy.
func NewDataPlane(daxSvc *DAXService, ddbSvc *dynamodbsvc.DynamoDBService) *DataPlane {
	return &DataPlane{daxService: daxSvc, ddbService: ddbSvc}
}

// ClusterStats returns cache stats for a cluster.
func (dp *DataPlane) ClusterStats(clusterName string) CacheStats {
	return dp.daxService.GetStore().GetClusterCache(clusterName).Stats()
}

func (dp *DataPlane) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Stats endpoint
	if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/stats/") {
		clusterName := strings.TrimPrefix(r.URL.Path, "/stats/")
		writeJSON(w, dp.ClusterStats(clusterName))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	target := r.Header.Get("X-Amz-Target")
	action := target
	if idx := strings.LastIndex(target, "."); idx >= 0 {
		action = target[idx+1:]
	}

	cluster := r.Header.Get("X-Dax-Cluster")
	if cluster == "" {
		cluster = "default"
	}
	cache := dp.daxService.GetStore().GetClusterCache(cluster)

	switch action {
	case "GetItem":
		dp.handleGetItem(w, body, cache)
	case "PutItem", "UpdateItem", "DeleteItem":
		dp.handleWriteThrough(w, action, body, cache, cluster)
	case "Query", "Scan":
		dp.handleQueryReadThrough(w, action, body, cache)
	case "":
		http.Error(w, "missing X-Amz-Target header", http.StatusBadRequest)
	default:
		dp.passThrough(w, action, body)
	}
}

func (dp *DataPlane) handleGetItem(w http.ResponseWriter, body []byte, cache *Cache) {
	var req struct {
		TableName string         `json:"TableName"`
		Key       map[string]any `json:"Key"`
	}
	if err := gojson.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	pk, sk := extractKeyStrings(req.Key)
	if cached := cache.GetItem(req.TableName, pk, sk); cached != nil {
		writeJSON(w, cached)
		return
	}
	resp := dp.forwardToDynamo("GetItem", body)
	if resp == nil {
		http.Error(w, "DynamoDB request failed", http.StatusInternalServerError)
		return
	}
	cache.SetItem(req.TableName, pk, sk, resp)
	writeJSON(w, resp)
}

func (dp *DataPlane) handleWriteThrough(w http.ResponseWriter, action string, body []byte, cache *Cache, cluster string) {
	resp := dp.forwardToDynamo(action, body)
	if resp == nil {
		http.Error(w, "DynamoDB write failed", http.StatusInternalServerError)
		return
	}
	var req struct {
		TableName string         `json:"TableName"`
		Key       map[string]any `json:"Key"`
		Item      map[string]any `json:"Item"`
	}
	gojson.Unmarshal(body, &req)
	strategy := dp.daxService.GetStore().GetWriteStrategy(cluster)

	if action == "PutItem" && req.Item != nil {
		// PutItem includes non-key attributes in Item, so we cannot reliably
		// extract only key attributes without the table schema.  Invalidate
		// the entire table to guarantee correctness.
		if strategy == "update-cache" {
			pk, sk := extractKeyStrings(req.Item)
			cache.SetItem(req.TableName, pk, sk, resp)
		} else {
			cache.InvalidateTable(req.TableName)
		}
	} else if req.Key != nil {
		pk, sk := extractKeyStrings(req.Key)
		cache.InvalidateItem(req.TableName, pk, sk)
	}
	cache.IncrWriteThroughs()
	writeJSON(w, resp)
}

func (dp *DataPlane) handleQueryReadThrough(w http.ResponseWriter, action string, body []byte, cache *Cache) {
	var req struct {
		TableName string `json:"TableName"`
	}
	gojson.Unmarshal(body, &req)
	qKey := req.TableName + "|" + queryHash(body)

	if cached := cache.GetQuery(qKey); cached != nil {
		writeJSON(w, cached)
		return
	}
	resp := dp.forwardToDynamo(action, body)
	if resp == nil {
		http.Error(w, "DynamoDB query failed", http.StatusInternalServerError)
		return
	}
	cache.SetQuery(qKey, resp)
	writeJSON(w, resp)
}

func (dp *DataPlane) passThrough(w http.ResponseWriter, action string, body []byte) {
	resp := dp.forwardToDynamo(action, body)
	if resp == nil {
		http.Error(w, "DynamoDB request failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, resp)
}

func (dp *DataPlane) forwardToDynamo(action string, body []byte) any {
	ctx := &service.RequestContext{
		Action:     action,
		Body:       body,
		Region:     "us-east-1",
		AccountID:  "123456789012",
		RawRequest: &http.Request{Method: http.MethodPost},
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
	resp, err := dp.ddbService.HandleRequest(ctx)
	if err != nil {
		return nil
	}
	// DynamoDB handlers return RawBody ([]byte) rather than Body.
	if len(resp.RawBody) > 0 {
		var parsed any
		if err := gojson.Unmarshal(resp.RawBody, &parsed); err != nil {
			return nil
		}
		return parsed
	}
	return resp.Body
}

func extractKeyStrings(key map[string]any) (string, string) {
	// Sort attribute names for deterministic key extraction.
	names := make([]string, 0, len(key))
	for k := range key {
		names = append(names, k)
	}
	sort.Strings(names)

	pk, sk := "", ""
	for _, name := range names {
		v := key[name]
		if m, ok := v.(map[string]any); ok {
			for _, val := range m {
				if pk == "" {
					pk = fmt.Sprintf("%v", val)
				} else if sk == "" {
					sk = fmt.Sprintf("%v", val)
					return pk, sk
				}
			}
		}
	}
	return pk, sk
}

func queryHash(body []byte) string {
	h := sha256.Sum256(body)
	return fmt.Sprintf("%x", h[:16])
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	gojson.NewEncoder(w).Encode(v)
}
