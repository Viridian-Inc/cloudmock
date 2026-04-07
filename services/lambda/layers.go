package lambda

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// Layer represents a Lambda layer.
type Layer struct {
	LayerName string
	LayerArn  string
	Versions  []*LayerVersion
}

// LayerVersion represents a specific version of a layer.
type LayerVersion struct {
	LayerName             string
	LayerArn              string
	LayerVersionArn       string
	Version               int64
	Description           string
	CompatibleRuntimes    []string
	CompatibleArchitectures []string
	CreatedDate           string
	CodeSha256            string
	CodeSize              int64
	Content               []byte
}

// LayerStore manages Lambda layers.
type LayerStore struct {
	mu     sync.RWMutex
	layers map[string]*Layer // layerName -> Layer
	region string
	acctID string
}

func NewLayerStore(accountID, region string) *LayerStore {
	return &LayerStore{
		layers: make(map[string]*Layer),
		region: region,
		acctID: accountID,
	}
}

func (s *LayerStore) PublishVersion(name, description string, runtimes, architectures []string, code []byte) *LayerVersion {
	s.mu.Lock()
	defer s.mu.Unlock()

	layer, ok := s.layers[name]
	if !ok {
		layer = &Layer{
			LayerName: name,
			LayerArn:  fmt.Sprintf("arn:aws:lambda:%s:%s:layer:%s", s.region, s.acctID, name),
		}
		s.layers[name] = layer
	}

	version := int64(len(layer.Versions) + 1)
	hash := sha256.Sum256(code)

	lv := &LayerVersion{
		LayerName:             name,
		LayerArn:              layer.LayerArn,
		LayerVersionArn:       fmt.Sprintf("%s:%d", layer.LayerArn, version),
		Version:               version,
		Description:           description,
		CompatibleRuntimes:    runtimes,
		CompatibleArchitectures: architectures,
		CreatedDate:           time.Now().UTC().Format(time.RFC3339Nano),
		CodeSha256:            hex.EncodeToString(hash[:]),
		CodeSize:              int64(len(code)),
		Content:               code,
	}
	layer.Versions = append(layer.Versions, lv)
	return lv
}

func (s *LayerStore) GetVersion(name string, version int64) (*LayerVersion, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	layer, ok := s.layers[name]
	if !ok {
		return nil, false
	}
	for _, lv := range layer.Versions {
		if lv.Version == version {
			return lv, true
		}
	}
	return nil, false
}

func (s *LayerStore) ListLayers() []*Layer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Layer, 0, len(s.layers))
	for _, l := range s.layers {
		out = append(out, l)
	}
	return out
}

func (s *LayerStore) ListVersions(name string) []*LayerVersion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	layer, ok := s.layers[name]
	if !ok {
		return nil
	}
	out := make([]*LayerVersion, len(layer.Versions))
	copy(out, layer.Versions)
	return out
}

func (s *LayerStore) DeleteVersion(name string, version int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	layer, ok := s.layers[name]
	if !ok {
		return false
	}
	for i, lv := range layer.Versions {
		if lv.Version == version {
			layer.Versions = append(layer.Versions[:i], layer.Versions[i+1:]...)
			return true
		}
	}
	return false
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handlePublishLayerVersion(ctx *service.RequestContext, layers *LayerStore, layerName string) (*service.Response, error) {
	var req struct {
		Description             string   `json:"Description"`
		CompatibleRuntimes      []string `json:"CompatibleRuntimes"`
		CompatibleArchitectures []string `json:"CompatibleArchitectures"`
		Content                 struct {
			ZipFile []byte `json:"ZipFile"`
		} `json:"Content"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return jsonErr(service.NewAWSError("InvalidParameterValueException",
			"Invalid request body.", http.StatusBadRequest))
	}

	code := req.Content.ZipFile
	if len(code) == 0 {
		code = make([]byte, 32)
		rand.Read(code)
	}

	lv := layers.PublishVersion(layerName, req.Description, req.CompatibleRuntimes, req.CompatibleArchitectures, code)

	return jsonOK(map[string]any{
		"LayerArn":                lv.LayerArn,
		"LayerVersionArn":        lv.LayerVersionArn,
		"Version":                lv.Version,
		"Description":            lv.Description,
		"CompatibleRuntimes":     lv.CompatibleRuntimes,
		"CompatibleArchitectures": lv.CompatibleArchitectures,
		"CreatedDate":            lv.CreatedDate,
		"Content": map[string]any{
			"CodeSha256": lv.CodeSha256,
			"CodeSize":   lv.CodeSize,
		},
	})
}

func handleGetLayerVersion(ctx *service.RequestContext, layers *LayerStore, layerName string, versionStr string) (*service.Response, error) {
	version, err := strconv.ParseInt(versionStr, 10, 64)
	if err != nil {
		return jsonErr(service.NewAWSError("InvalidParameterValueException",
			"Invalid version number.", http.StatusBadRequest))
	}

	lv, ok := layers.GetVersion(layerName, version)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Layer version %s:%d not found.", layerName, version), http.StatusNotFound))
	}

	return jsonOK(map[string]any{
		"LayerArn":         lv.LayerArn,
		"LayerVersionArn":  lv.LayerVersionArn,
		"Version":          lv.Version,
		"Description":      lv.Description,
		"CompatibleRuntimes": lv.CompatibleRuntimes,
		"CreatedDate":      lv.CreatedDate,
	})
}

func handleListLayers(_ *service.RequestContext, layers *LayerStore) (*service.Response, error) {
	allLayers := layers.ListLayers()
	items := make([]map[string]any, 0, len(allLayers))
	for _, l := range allLayers {
		latest := l.Versions[len(l.Versions)-1]
		items = append(items, map[string]any{
			"LayerName":      l.LayerName,
			"LayerArn":       l.LayerArn,
			"LatestMatchingVersion": map[string]any{
				"LayerVersionArn": latest.LayerVersionArn,
				"Version":         latest.Version,
				"Description":     latest.Description,
				"CreatedDate":     latest.CreatedDate,
			},
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i]["LayerName"].(string) < items[j]["LayerName"].(string)
	})
	return jsonOK(map[string]any{"Layers": items})
}

func handleListLayerVersions(_ *service.RequestContext, layers *LayerStore, layerName string) (*service.Response, error) {
	versions := layers.ListVersions(layerName)
	items := make([]map[string]any, 0, len(versions))
	for _, lv := range versions {
		items = append(items, map[string]any{
			"LayerVersionArn": lv.LayerVersionArn,
			"Version":         lv.Version,
			"Description":     lv.Description,
			"CreatedDate":     lv.CreatedDate,
		})
	}
	return jsonOK(map[string]any{"LayerVersions": items})
}

func handleDeleteLayerVersion(_ *service.RequestContext, layers *LayerStore, layerName string, versionStr string) (*service.Response, error) {
	version, err := strconv.ParseInt(versionStr, 10, 64)
	if err != nil {
		return jsonErr(service.NewAWSError("InvalidParameterValueException",
			"Invalid version number.", http.StatusBadRequest))
	}
	if !layers.DeleteVersion(layerName, version) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Layer version %s:%d not found.", layerName, version), http.StatusNotFound))
	}
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatJSON}, nil
}
