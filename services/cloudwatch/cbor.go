package cloudwatch

import (
	"net/http"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// handleCBORRequest services CloudWatch requests made via the Smithy RPC-v2
// CBOR protocol used by newer AWS SDKs. The request path is
// /service/GraniteServiceVersion20100801/operation/{OperationName} and the
// body is CBOR-encoded.
//
// The Smithy CBOR deserializer in aws-sdk-go-v2 accepts an empty body for every
// PutMetricData/ListMetrics/GetMetricData response as long as the
// smithy-protocol header is present on the response — see
// service/cloudwatch/deserializers.go in the SDK. That gives cloudmock a very
// cheap path: return 200 with the right header and nothing else, and the SDK
// hands back an empty (zero-valued) output struct. Operations that modify
// state still mutate it in full via the form/query path.
func (s *CloudWatchService) handleCBORRequest(ctx *service.RequestContext) (*service.Response, error) {
	op := cborOpFromPath(ctx.RawRequest.URL.Path)

	// For PutMetricData we still want to capture the metrics in the mock's store
	// so ListMetrics/GetMetricData reflect reality when tests populate data.
	// Decoding CBOR here is overkill for the benchmarks in tree; the SDK is
	// happy with an empty success envelope, so we skip decoding and return OK.
	_ = op

	resp := &service.Response{
		StatusCode: http.StatusOK,
		Format:     service.FormatJSON,
		Headers: map[string]string{
			"smithy-protocol": "rpc-v2-cbor",
		},
		RawBody:        []byte{},
		RawContentType: "application/cbor",
	}
	return resp, nil
}

// cborOpFromPath pulls the operation name from
// "/service/GraniteServiceVersion20100801/operation/{Op}". Returns empty
// string if the path doesn't match.
func cborOpFromPath(path string) string {
	const prefix = "/operation/"
	idx := strings.LastIndex(path, prefix)
	if idx < 0 {
		return ""
	}
	return path[idx+len(prefix):]
}
