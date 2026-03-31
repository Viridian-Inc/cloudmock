package cloudcontrol

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func str(params map[string]any, key string) string {
	if v, ok := params[key].(string); ok {
		return v
	}
	return ""
}

func requestResponse(req *ResourceRequest) map[string]any {
	return map[string]any{
		"RequestToken":  req.RequestToken,
		"OperationType": req.OperationType,
		"TypeName":      req.TypeName,
		"Identifier":    req.Identifier,
		"OperationStatus": req.StatusCode,
		"StatusMessage":   req.StatusMessage,
		"EventTime":       req.EventTime,
	}
}

func resourceResponse(r *Resource) map[string]any {
	return map[string]any{
		"TypeName":       r.TypeName,
		"Identifier":     r.Identifier,
		"Properties":     r.Properties,
	}
}

func handleCreateResource(params map[string]any, store *Store) (*service.Response, error) {
	typeName := str(params, "TypeName")
	if typeName == "" {
		return jsonErr(service.ErrValidation("TypeName is required"))
	}
	identifier := str(params, "Identifier")
	if identifier == "" {
		return jsonErr(service.ErrValidation("Identifier is required"))
	}
	properties := str(params, "DesiredState")

	req, _, err := store.CreateResource(typeName, identifier, properties)
	if err != nil {
		return jsonErr(service.ErrAlreadyExists("Resource", typeName+"/"+identifier))
	}
	return jsonOK(map[string]any{"ProgressEvent": requestResponse(req)})
}

func handleGetResource(params map[string]any, store *Store) (*service.Response, error) {
	typeName := str(params, "TypeName")
	identifier := str(params, "Identifier")
	if typeName == "" || identifier == "" {
		return jsonErr(service.ErrValidation("TypeName and Identifier are required"))
	}
	r, ok := store.GetResource(typeName, identifier)
	if !ok {
		return jsonErr(service.ErrNotFound("Resource", typeName+"/"+identifier))
	}
	return jsonOK(map[string]any{"ResourceDescription": resourceResponse(r)})
}

func handleListResources(params map[string]any, store *Store) (*service.Response, error) {
	typeName := str(params, "TypeName")
	if typeName == "" {
		return jsonErr(service.ErrValidation("TypeName is required"))
	}
	resources := store.ListResources(typeName)
	out := make([]map[string]any, 0, len(resources))
	for _, r := range resources {
		out = append(out, resourceResponse(r))
	}
	return jsonOK(map[string]any{"ResourceDescriptions": out})
}

func handleUpdateResource(params map[string]any, store *Store) (*service.Response, error) {
	typeName := str(params, "TypeName")
	identifier := str(params, "Identifier")
	patchDoc := str(params, "PatchDocument")
	if typeName == "" || identifier == "" {
		return jsonErr(service.ErrValidation("TypeName and Identifier are required"))
	}
	req, err := store.UpdateResource(typeName, identifier, patchDoc)
	if err != nil {
		return jsonErr(service.ErrNotFound("Resource", typeName+"/"+identifier))
	}
	return jsonOK(map[string]any{"ProgressEvent": requestResponse(req)})
}

func handleDeleteResource(params map[string]any, store *Store) (*service.Response, error) {
	typeName := str(params, "TypeName")
	identifier := str(params, "Identifier")
	if typeName == "" || identifier == "" {
		return jsonErr(service.ErrValidation("TypeName and Identifier are required"))
	}
	req, err := store.DeleteResource(typeName, identifier)
	if err != nil {
		return jsonErr(service.ErrNotFound("Resource", typeName+"/"+identifier))
	}
	return jsonOK(map[string]any{"ProgressEvent": requestResponse(req)})
}

func handleGetResourceRequestStatus(params map[string]any, store *Store) (*service.Response, error) {
	token := str(params, "RequestToken")
	if token == "" {
		return jsonErr(service.ErrValidation("RequestToken is required"))
	}
	req, ok := store.GetResourceRequestStatus(token)
	if !ok {
		return jsonErr(service.ErrNotFound("ResourceRequest", token))
	}
	return jsonOK(map[string]any{"ProgressEvent": requestResponse(req)})
}

func handleListResourceRequests(store *Store) (*service.Response, error) {
	requests := store.ListResourceRequests()
	out := make([]map[string]any, 0, len(requests))
	for _, req := range requests {
		out = append(out, requestResponse(req))
	}
	return jsonOK(map[string]any{"ResourceRequestStatusSummaries": out})
}
