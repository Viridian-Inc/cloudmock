package service

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testXMLBody struct {
	XMLName xml.Name `xml:"TestResponse"`
	Value   string   `xml:"Value"`
}

func TestWriteXMLResponse(t *testing.T) {
	w := httptest.NewRecorder()
	body := &testXMLBody{Value: "hello"}

	err := WriteXMLResponse(w, http.StatusOK, body)
	require.NoError(t, err)

	result := w.Result()
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Equal(t, "text/xml", result.Header.Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "<Value>hello</Value>")
	assert.Contains(t, w.Body.String(), "<TestResponse>")
}

func TestWriteJSONResponse(t *testing.T) {
	w := httptest.NewRecorder()
	body := map[string]string{"Key": "value"}

	err := WriteJSONResponse(w, http.StatusOK, body)
	require.NoError(t, err)

	result := w.Result()
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Equal(t, "application/x-amz-json-1.1", result.Header.Get("Content-Type"))
	assert.Contains(t, w.Body.String(), `"Key"`)
	assert.Contains(t, w.Body.String(), `"value"`)
}

func TestWriteErrorResponse_XML(t *testing.T) {
	w := httptest.NewRecorder()
	awsErr := NewAWSError("NoSuchBucket", "The specified bucket does not exist", http.StatusNotFound)

	err := WriteErrorResponse(w, awsErr, FormatXML)
	require.NoError(t, err)

	result := w.Result()
	assert.Equal(t, http.StatusNotFound, result.StatusCode)
	assert.Equal(t, "text/xml", result.Header.Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "<Code>NoSuchBucket</Code>")
	assert.Contains(t, w.Body.String(), "<Message>The specified bucket does not exist</Message>")
}

func TestWriteErrorResponse_JSON(t *testing.T) {
	w := httptest.NewRecorder()
	awsErr := NewAWSError("NoSuchBucket", "The specified bucket does not exist", http.StatusNotFound)

	err := WriteErrorResponse(w, awsErr, FormatJSON)
	require.NoError(t, err)

	result := w.Result()
	assert.Equal(t, http.StatusNotFound, result.StatusCode)
	assert.Equal(t, "application/x-amz-json-1.1", result.Header.Get("Content-Type"))
	assert.Contains(t, w.Body.String(), `"__type"`)
	assert.Contains(t, w.Body.String(), `"NoSuchBucket"`)
	assert.Contains(t, w.Body.String(), `"Message"`)
}
