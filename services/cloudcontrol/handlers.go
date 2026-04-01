package cloudcontrol

import (
	"encoding/json"
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
		"RequestToken":    req.RequestToken,
		"OperationType":   req.OperationType,
		"TypeName":        req.TypeName,
		"Identifier":      req.Identifier,
		"OperationStatus": req.StatusCode,
		"StatusMessage":   req.StatusMessage,
		"EventTime":       req.EventTime,
	}
}

func resourceResponse(r *Resource) map[string]any {
	return map[string]any{
		"TypeName":   r.TypeName,
		"Identifier": r.Identifier,
		"Properties": r.Properties,
	}
}

// tryProxyCreate attempts to proxy a CreateResource to the backing service via the locator.
// Returns true if the proxy was attempted (regardless of success), false to fall back to generic.
func tryProxyCreate(typeName, identifier, properties string, locator ServiceLocator, ctx *service.RequestContext) bool {
	if locator == nil {
		return false
	}
	svcName, ok := ResourceTypeToService[typeName]
	if !ok {
		return false
	}
	backingSvc, err := locator.Lookup(svcName)
	if err != nil {
		return false
	}

	// Build a proxy request to the backing service based on resource type.
	var proxyAction string
	var proxyBody map[string]any

	switch typeName {
	case "AWS::S3::Bucket":
		proxyAction = "CreateBucket"
		proxyBody = map[string]any{"Bucket": identifier}
	case "AWS::DynamoDB::Table":
		proxyAction = "CreateTable"
		// Parse properties if available, fall back to minimal table def.
		proxyBody = map[string]any{
			"TableName":            identifier,
			"AttributeDefinitions": []map[string]any{{"AttributeName": "id", "AttributeType": "S"}},
			"KeySchema":            []map[string]any{{"AttributeName": "id", "KeyType": "HASH"}},
			"BillingMode":          "PAY_PER_REQUEST",
		}
	case "AWS::SQS::Queue":
		proxyAction = "CreateQueue"
		proxyBody = map[string]any{"QueueName": identifier}
	case "AWS::SNS::Topic":
		proxyAction = "CreateTopic"
		proxyBody = map[string]any{"Name": identifier}
	default:
		return false
	}

	// Override with parsed properties if available.
	if properties != "" {
		var parsed map[string]any
		if json.Unmarshal([]byte(properties), &parsed) == nil {
			for k, v := range parsed {
				proxyBody[k] = v
			}
		}
	}

	bodyBytes, _ := json.Marshal(proxyBody)
	proxyCtx := &service.RequestContext{
		Action:     proxyAction,
		Region:     ctx.Region,
		AccountID:  ctx.AccountID,
		Identity:   ctx.Identity,
		RawRequest: ctx.RawRequest,
		Body:       bodyBytes,
	}

	// Fire and forget — we still track in our store regardless.
	backingSvc.HandleRequest(proxyCtx)
	return true
}

func handleCreateResource(params map[string]any, store *Store, locator ServiceLocator, ctx *service.RequestContext) (*service.Response, error) {
	typeName := str(params, "TypeName")
	if typeName == "" {
		return jsonErr(service.NewAWSError("TypeNameNotFound",
			"TypeName is required", http.StatusBadRequest))
	}
	identifier := str(params, "Identifier")
	if identifier == "" {
		return jsonErr(service.NewAWSError("IdentifierNotFound",
			"Identifier is required", http.StatusBadRequest))
	}
	properties := str(params, "DesiredState")

	// Attempt to proxy to the backing service.
	tryProxyCreate(typeName, identifier, properties, locator, ctx)

	req, _, err := store.CreateResource(typeName, identifier, properties)
	if err != nil {
		return jsonErr(service.NewAWSError("AlreadyExists",
			"Resource already exists: "+typeName+"/"+identifier, http.StatusConflict))
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
