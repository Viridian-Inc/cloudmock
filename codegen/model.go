package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// BotocoreModel represents a parsed AWS service model from botocore.
type BotocoreModel struct {
	Metadata   ServiceMetadata              `json:"metadata"`
	Operations map[string]Operation         `json:"operations"`
	Shapes     map[string]Shape             `json:"shapes"`
}

type ServiceMetadata struct {
	APIVersion     string `json:"apiVersion"`
	EndpointPrefix string `json:"endpointPrefix"`
	Protocol       string `json:"protocol"`       // "json", "rest-json", "rest-xml", "query", "ec2"
	ServiceID      string `json:"serviceId"`
	SigningName    string `json:"signingName"`
	TargetPrefix   string `json:"targetPrefix"`   // For "json" protocol (X-Amz-Target)
	JSONVersion    string `json:"jsonVersion"`
}

type Operation struct {
	Name   string        `json:"name"`
	HTTP   OperationHTTP `json:"http"`
	Input  *ShapeRef     `json:"input"`
	Output *ShapeRef     `json:"output"`
	Errors []ShapeRef    `json:"errors"`
}

type OperationHTTP struct {
	Method     string `json:"method"`
	RequestURI string `json:"requestUri"`
}

type ShapeRef struct {
	Shape string `json:"shape"`
}

type Shape struct {
	Type     string               `json:"type"`     // "structure", "string", "integer", "long", "boolean", "timestamp", "blob", "list", "map"
	Required []string             `json:"required"`
	Members  map[string]ShapeMember `json:"members"`
	Member   *ShapeRef            `json:"member"` // For list types
	Key      *ShapeRef            `json:"key"`    // For map types
	Value    *ShapeRef            `json:"value"`  // For map types
	Enum     []string             `json:"enum"`
	Min      *float64             `json:"min"`
	Max      *float64             `json:"max"`
	Pattern  string               `json:"pattern"`
	Error    *ErrorInfo           `json:"error"`
}

type ShapeMember struct {
	Shape        string `json:"shape"`
	Location     string `json:"location"`     // "uri", "querystring", "header"
	LocationName string `json:"locationName"` // The wire name
	IdempotencyToken bool `json:"idempotencyToken"`
}

type ErrorInfo struct {
	Code       string `json:"code"`
	HTTPStatus int    `json:"httpStatusCode"`
	Fault      bool   `json:"senderFault"`
}

// LoadModel loads a botocore service model from the given service name.
// It searches the botocore data directory for the service and latest version.
func LoadModel(botocorePath, serviceName string) (*BotocoreModel, error) {
	svcDir := filepath.Join(botocorePath, serviceName)
	if _, err := os.Stat(svcDir); err != nil {
		return nil, fmt.Errorf("service %q not found in %s", serviceName, botocorePath)
	}

	// Find latest API version
	entries, err := os.ReadDir(svcDir)
	if err != nil {
		return nil, err
	}
	var versions []string
	for _, e := range entries {
		if e.IsDir() {
			versions = append(versions, e.Name())
		}
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found for %s", serviceName)
	}
	sort.Strings(versions)
	latest := versions[len(versions)-1]

	// Load service-2.json.gz or service-2.json
	modelPath := filepath.Join(svcDir, latest, "service-2.json.gz")
	if _, err := os.Stat(modelPath); err != nil {
		modelPath = filepath.Join(svcDir, latest, "service-2.json")
	}

	return loadModelFile(modelPath)
}

func loadModelFile(path string) (*BotocoreModel, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var reader *json.Decoder
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		reader = json.NewDecoder(gz)
	} else {
		reader = json.NewDecoder(f)
	}

	var model BotocoreModel
	if err := reader.Decode(&model); err != nil {
		return nil, fmt.Errorf("decode model: %w", err)
	}
	return &model, nil
}

// SigningName returns the service name used for SigV4 credential scope routing.
func (m *BotocoreModel) SigningName() string {
	if m.Metadata.SigningName != "" {
		return m.Metadata.SigningName
	}
	return m.Metadata.EndpointPrefix
}

// SortedOperations returns operation names sorted alphabetically.
func (m *BotocoreModel) SortedOperations() []string {
	names := make([]string, 0, len(m.Operations))
	for name := range m.Operations {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// FilterOperations returns only operations whose names are in the given set.
// If opNames is nil/empty, all operations are returned.
func (m *BotocoreModel) FilterOperations(opNames []string) map[string]Operation {
	if len(opNames) == 0 {
		return m.Operations
	}
	set := make(map[string]bool, len(opNames))
	for _, n := range opNames {
		set[n] = true
	}
	result := make(map[string]Operation, len(opNames))
	for name, op := range m.Operations {
		if set[name] {
			result[name] = op
		}
	}
	return result
}
