package azurearm

import (
	"net/http"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

const contentTypeJSON = "application/json"

// ErrorEnvelope is the standard Azure Resource Manager error response shape.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody describes an Azure Resource Manager error.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSONResponse returns a service response that writes Azure JSON directly.
func JSONResponse(statusCode int, body any) (*service.Response, error) {
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}
	return &service.Response{
		StatusCode:     statusCode,
		RawBody:        data,
		RawContentType: contentTypeJSON,
	}, nil
}

// ErrorResponse returns an Azure Resource Manager error envelope.
func ErrorResponse(statusCode int, code, message string) (*service.Response, error) {
	return JSONResponse(statusCode, ErrorEnvelope{
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
	})
}
