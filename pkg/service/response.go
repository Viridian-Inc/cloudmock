package service

import (
	"encoding/xml"
	"fmt"
	"net/http"

	gojson "github.com/goccy/go-json"
)

// ResponseFormat indicates the wire format for a response.
type ResponseFormat int

const (
	FormatXML  ResponseFormat = iota
	FormatJSON ResponseFormat = iota
)

// Response holds the components of an HTTP response before it is written.
type Response struct {
	StatusCode     int
	Body           any
	Format         ResponseFormat
	Headers        map[string]string
	RawBody        []byte // if set, write these bytes directly instead of marshaling Body
	RawContentType string // Content-Type to use when writing RawBody
}

// WriteXMLResponse marshals body as XML and writes it with Content-Type text/xml.
func WriteXMLResponse(w http.ResponseWriter, statusCode int, body any) error {
	data, err := xml.Marshal(body)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/xml")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(statusCode)
	_, err = w.Write(data)
	return err
}

// WriteJSONResponse marshals body as JSON and writes it with Content-Type application/x-amz-json-1.1.
func WriteJSONResponse(w http.ResponseWriter, statusCode int, body any) error {
	data, err := gojson.Marshal(body)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(statusCode)
	_, err = w.Write(data)
	return err
}

// WriteErrorResponse writes an AWSError in the specified format.
func WriteErrorResponse(w http.ResponseWriter, awsErr *AWSError, format ResponseFormat) error {
	switch format {
	case FormatJSON:
		return WriteJSONResponse(w, awsErr.StatusCode(), awsErr)
	default:
		return WriteXMLResponse(w, awsErr.StatusCode(), awsErr)
	}
}
