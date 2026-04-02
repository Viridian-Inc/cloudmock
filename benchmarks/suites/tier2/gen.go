package tier2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/neureaux/cloudmock/benchmarks/harness"
	"github.com/neureaux/cloudmock/pkg/stub"
	"github.com/neureaux/cloudmock/services/stubs"
)

func GenerateAll() []harness.Suite {
	models := stubs.AllModels()
	var suites []harness.Suite
	for _, m := range models {
		suites = append(suites, newStubSuite(m))
	}
	return suites
}

type stubSuite struct {
	model *stub.ServiceModel
}

func newStubSuite(model *stub.ServiceModel) harness.Suite {
	return &stubSuite{model: model}
}

func (s *stubSuite) Name() string { return s.model.ServiceName }
func (s *stubSuite) Tier() int    { return 2 }

func (s *stubSuite) Operations() []harness.Operation {
	var ops []harness.Operation
	for _, action := range s.model.Actions {
		a := action
		ops = append(ops, harness.Operation{
			Name: a.Name,
			Run:  s.makeRunner(a),
			Validate: func(resp any) []harness.Finding {
				return []harness.Finding{harness.CheckNotNil(resp, a.Name+"Response")}
			},
		})
	}
	return ops
}

func (s *stubSuite) makeRunner(action stub.Action) func(ctx context.Context, endpoint string) (any, error) {
	return func(ctx context.Context, endpoint string) (any, error) {
		body := buildRequestBody(s.model, action)

		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}

		switch s.model.Protocol {
		case "json":
			req.Header.Set("Content-Type", "application/x-amz-json-1.1")
			req.Header.Set("X-Amz-Target", s.model.TargetPrefix+"."+action.Name)
		case "query", "rest-xml":
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		case "rest-json":
			req.Header.Set("Content-Type", "application/json")
		}

		req.Header.Set("Authorization",
			fmt.Sprintf("AWS4-HMAC-SHA256 Credential=test/20260101/us-east-1/%s/aws4_request, SignedHeaders=host, Signature=fake",
				s.model.ServiceName))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]any
		json.Unmarshal(respBody, &result)

		if resp.StatusCode >= 400 {
			return result, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		return result, nil
	}
}

func buildRequestBody(model *stub.ServiceModel, action stub.Action) []byte {
	switch model.Protocol {
	case "json", "rest-json":
		params := make(map[string]any)
		for _, f := range action.InputFields {
			if f.Required {
				params[f.Name] = stubValue(f)
			}
		}
		data, _ := json.Marshal(params)
		return data
	case "query":
		var parts []string
		parts = append(parts, "Action="+action.Name, "Version=2012-11-05")
		for _, f := range action.InputFields {
			if f.Required {
				parts = append(parts, fmt.Sprintf("%s=%v", f.Name, stubValue(f)))
			}
		}
		return []byte(strings.Join(parts, "&"))
	default:
		return []byte{}
	}
}

func stubValue(f stub.Field) any {
	switch f.Type {
	case "string":
		return fmt.Sprintf("bench-%s", strings.ToLower(f.Name))
	case "integer":
		return 1
	case "boolean":
		return true
	default:
		return "bench-value"
	}
}
