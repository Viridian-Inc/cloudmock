package iotdata

import (
	gojson "github.com/goccy/go-json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func emptyOK() (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func getStr(p map[string]any, k string) string {
	if v, ok := p[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(p map[string]any, k string) int {
	if v, ok := p[k]; ok {
		if n, ok := v.(float64); ok {
			return int(n)
		}
	}
	return 0
}

func getMap(p map[string]any, k string) map[string]any {
	if v, ok := p[k]; ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return nil
}

func handleGetThingShadow(p map[string]any, store *Store) (*service.Response, error) {
	thingName := getStr(p, "thingName")
	if thingName == "" {
		return jsonErr(service.ErrValidation("thingName is required."))
	}
	shadowName := getStr(p, "shadowName") // empty = classic shadow
	shadow, awsErr := store.GetThingShadow(thingName, shadowName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"state":     shadow.State,
		"metadata":  shadow.Metadata,
		"version":   shadow.Version,
		"timestamp": shadow.Timestamp.Unix(),
	})
}

func handleUpdateThingShadow(p map[string]any, body []byte, store *Store) (*service.Response, error) {
	thingName := getStr(p, "thingName")
	if thingName == "" {
		return jsonErr(service.ErrValidation("thingName is required."))
	}
	shadowName := getStr(p, "shadowName")

	// Parse state from body.
	var payload map[string]any
	if len(body) > 0 {
		gojson.Unmarshal(body, &payload)
	}
	state := getMap(payload, "state")
	if state == nil {
		state = make(map[string]any)
	}

	shadow, awsErr := store.UpdateThingShadow(thingName, shadowName, state)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"state":     shadow.State,
		"metadata":  shadow.Metadata,
		"version":   shadow.Version,
		"timestamp": shadow.Timestamp.Unix(),
	})
}

func handleDeleteThingShadow(p map[string]any, store *Store) (*service.Response, error) {
	thingName := getStr(p, "thingName")
	if thingName == "" {
		return jsonErr(service.ErrValidation("thingName is required."))
	}
	shadowName := getStr(p, "shadowName")
	if awsErr := store.DeleteThingShadow(thingName, shadowName); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"version": 0})
}

func handleListNamedShadowsForThing(p map[string]any, store *Store) (*service.Response, error) {
	thingName := getStr(p, "thingName")
	if thingName == "" {
		return jsonErr(service.ErrValidation("thingName is required."))
	}
	names, awsErr := store.ListNamedShadowsForThing(thingName)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"results": names, "timestamp": 0})
}

func handlePublish(p map[string]any, body []byte, store *Store) (*service.Response, error) {
	topic := getStr(p, "topic")
	if topic == "" {
		return jsonErr(service.ErrValidation("topic is required."))
	}
	qos := getInt(p, "qos")
	store.Publish(topic, body, qos)
	return emptyOK()
}
