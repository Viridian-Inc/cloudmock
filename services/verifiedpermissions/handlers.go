package verifiedpermissions

import (
	gojson "github.com/goccy/go-json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("ValidationException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func emptyOK() (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleCreatePolicyStore(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	description, _ := params["description"].(string)
	validationSettings, _ := params["validationSettings"].(map[string]any)
	ps, awsErr := store.CreatePolicyStore(description, validationSettings)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"policyStoreId": ps.PolicyStoreId,
		"arn":           ps.Arn,
		"createdDate":   ps.CreatedDate.Format("2006-01-02T15:04:05Z"),
		"lastUpdatedDate": ps.LastUpdatedDate.Format("2006-01-02T15:04:05Z"),
	})
}

func handleGetPolicyStore(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["policyStoreId"].(string)
	ps, awsErr := store.GetPolicyStore(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	result := map[string]any{
		"policyStoreId":   ps.PolicyStoreId,
		"arn":             ps.Arn,
		"description":     ps.Description,
		"createdDate":     ps.CreatedDate.Format("2006-01-02T15:04:05Z"),
		"lastUpdatedDate": ps.LastUpdatedDate.Format("2006-01-02T15:04:05Z"),
	}
	if ps.ValidationSettings != nil {
		result["validationSettings"] = ps.ValidationSettings
	}
	return jsonOK(result)
}

func handleListPolicyStores(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	stores := store.ListPolicyStores()
	items := make([]map[string]any, 0, len(stores))
	for _, ps := range stores {
		items = append(items, map[string]any{
			"policyStoreId": ps.PolicyStoreId,
			"arn":           ps.Arn,
			"description":   ps.Description,
			"createdDate":   ps.CreatedDate.Format("2006-01-02T15:04:05Z"),
		})
	}
	return jsonOK(map[string]any{"policyStores": items})
}

func handleUpdatePolicyStore(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["policyStoreId"].(string)
	description, _ := params["description"].(string)
	validationSettings, _ := params["validationSettings"].(map[string]any)
	ps, awsErr := store.UpdatePolicyStore(id, description, validationSettings)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"policyStoreId":   ps.PolicyStoreId,
		"arn":             ps.Arn,
		"createdDate":     ps.CreatedDate.Format("2006-01-02T15:04:05Z"),
		"lastUpdatedDate": ps.LastUpdatedDate.Format("2006-01-02T15:04:05Z"),
	})
}

func handleDeletePolicyStore(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	id, _ := params["policyStoreId"].(string)
	if awsErr := store.DeletePolicyStore(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleCreatePolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	storeId, _ := params["policyStoreId"].(string)
	definition, _ := params["definition"].(map[string]any)
	p, awsErr := store.CreatePolicy(storeId, definition)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"policyStoreId": p.PolicyStoreId,
		"policyId":      p.PolicyId,
		"policyType":    p.PolicyType,
		"createdDate":   p.CreatedDate.Format("2006-01-02T15:04:05Z"),
	})
}

func handleGetPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	storeId, _ := params["policyStoreId"].(string)
	policyId, _ := params["policyId"].(string)
	p, awsErr := store.GetPolicy(storeId, policyId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"policyStoreId":   p.PolicyStoreId,
		"policyId":        p.PolicyId,
		"policyType":      p.PolicyType,
		"definition":      p.Definition,
		"createdDate":     p.CreatedDate.Format("2006-01-02T15:04:05Z"),
		"lastUpdatedDate": p.LastUpdatedDate.Format("2006-01-02T15:04:05Z"),
	})
}

func handleListPolicies(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	storeId, _ := params["policyStoreId"].(string)
	policies := store.ListPolicies(storeId)
	items := make([]map[string]any, 0, len(policies))
	for _, p := range policies {
		items = append(items, map[string]any{
			"policyStoreId": p.PolicyStoreId,
			"policyId":      p.PolicyId,
			"policyType":    p.PolicyType,
			"definition":    p.Definition,
			"createdDate":   p.CreatedDate.Format("2006-01-02T15:04:05Z"),
		})
	}
	return jsonOK(map[string]any{"policies": items})
}

func handleUpdatePolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	storeId, _ := params["policyStoreId"].(string)
	policyId, _ := params["policyId"].(string)
	definition, _ := params["definition"].(map[string]any)
	p, awsErr := store.UpdatePolicy(storeId, policyId, definition)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"policyStoreId":   p.PolicyStoreId,
		"policyId":        p.PolicyId,
		"policyType":      p.PolicyType,
		"lastUpdatedDate": p.LastUpdatedDate.Format("2006-01-02T15:04:05Z"),
	})
}

func handleDeletePolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	storeId, _ := params["policyStoreId"].(string)
	policyId, _ := params["policyId"].(string)
	if awsErr := store.DeletePolicy(storeId, policyId); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handlePutSchema(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	storeId, _ := params["policyStoreId"].(string)
	definition, _ := params["definition"].(map[string]any)
	schemaStr := ""
	if cedarJson, ok := definition["cedarJson"].(string); ok {
		schemaStr = cedarJson
	}
	sc, awsErr := store.PutSchema(storeId, schemaStr)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"policyStoreId":   sc.PolicyStoreId,
		"namespaces":      sc.Namespaces,
		"createdDate":     sc.CreatedDate.Format("2006-01-02T15:04:05Z"),
		"lastUpdatedDate": sc.LastUpdatedDate.Format("2006-01-02T15:04:05Z"),
	})
}

func handleGetSchema(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	storeId, _ := params["policyStoreId"].(string)
	sc, awsErr := store.GetSchema(storeId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"policyStoreId": sc.PolicyStoreId,
		"schema":        sc.Schema,
		"namespaces":    sc.Namespaces,
		"createdDate":   sc.CreatedDate.Format("2006-01-02T15:04:05Z"),
	})
}

func handleIsAuthorized(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	storeId, _ := params["policyStoreId"].(string)
	decision, details := store.IsAuthorized(storeId)
	return jsonOK(map[string]any{
		"decision":            decision,
		"determiningPolicies": details["determiningPolicies"],
		"errors":              details["errors"],
	})
}

func handleIsAuthorizedWithToken(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	// Same as IsAuthorized for mock purposes
	return handleIsAuthorized(ctx, store)
}

func handleCreatePolicyTemplate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	storeId, _ := params["policyStoreId"].(string)
	description, _ := params["description"].(string)
	statement, _ := params["statement"].(string)
	pt, awsErr := store.CreatePolicyTemplate(storeId, description, statement)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"policyStoreId":    pt.PolicyStoreId,
		"policyTemplateId": pt.PolicyTemplateId,
		"createdDate":      pt.CreatedDate.Format("2006-01-02T15:04:05Z"),
	})
}

func handleGetPolicyTemplate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	storeId, _ := params["policyStoreId"].(string)
	templateId, _ := params["policyTemplateId"].(string)
	pt, awsErr := store.GetPolicyTemplate(storeId, templateId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"policyStoreId":    pt.PolicyStoreId,
		"policyTemplateId": pt.PolicyTemplateId,
		"description":      pt.Description,
		"statement":        pt.Statement,
		"createdDate":      pt.CreatedDate.Format("2006-01-02T15:04:05Z"),
		"lastUpdatedDate":  pt.LastUpdatedDate.Format("2006-01-02T15:04:05Z"),
	})
}

func handleListPolicyTemplates(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	storeId, _ := params["policyStoreId"].(string)
	templates := store.ListPolicyTemplates(storeId)
	items := make([]map[string]any, 0, len(templates))
	for _, pt := range templates {
		items = append(items, map[string]any{
			"policyStoreId":    pt.PolicyStoreId,
			"policyTemplateId": pt.PolicyTemplateId,
			"description":      pt.Description,
			"createdDate":      pt.CreatedDate.Format("2006-01-02T15:04:05Z"),
		})
	}
	return jsonOK(map[string]any{"policyTemplates": items})
}

func handleUpdatePolicyTemplate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	storeId, _ := params["policyStoreId"].(string)
	templateId, _ := params["policyTemplateId"].(string)
	description, _ := params["description"].(string)
	statement, _ := params["statement"].(string)
	tmpl, awsErr := store.UpdatePolicyTemplate(storeId, templateId, description, statement)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"policyStoreId":   tmpl.PolicyStoreId,
		"policyTemplateId": tmpl.PolicyTemplateId,
		"createdDate":     tmpl.CreatedDate.Format("2006-01-02T15:04:05Z"),
		"lastUpdatedDate": tmpl.LastUpdatedDate.Format("2006-01-02T15:04:05Z"),
	})
}

func handleDeletePolicyTemplate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	storeId, _ := params["policyStoreId"].(string)
	templateId, _ := params["policyTemplateId"].(string)
	if awsErr := store.DeletePolicyTemplate(storeId, templateId); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleCreateIdentitySource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	storeId, _ := params["policyStoreId"].(string)
	principalEntityType, _ := params["principalEntityType"].(string)
	config, _ := params["configuration"].(map[string]any)
	is, awsErr := store.CreateIdentitySource(storeId, principalEntityType, config)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"policyStoreId":    is.PolicyStoreId,
		"identitySourceId": is.IdentitySourceId,
		"createdDate":      is.CreatedDate.Format("2006-01-02T15:04:05Z"),
	})
}

func handleGetIdentitySource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	storeId, _ := params["policyStoreId"].(string)
	isId, _ := params["identitySourceId"].(string)
	is, awsErr := store.GetIdentitySource(storeId, isId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"policyStoreId":       is.PolicyStoreId,
		"identitySourceId":    is.IdentitySourceId,
		"principalEntityType": is.PrincipalEntityType,
		"configuration":       is.Configuration,
		"createdDate":         is.CreatedDate.Format("2006-01-02T15:04:05Z"),
		"lastUpdatedDate":     is.LastUpdatedDate.Format("2006-01-02T15:04:05Z"),
	})
}

func handleListIdentitySources(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	storeId, _ := params["policyStoreId"].(string)
	sources := store.ListIdentitySources(storeId)
	items := make([]map[string]any, 0, len(sources))
	for _, is := range sources {
		items = append(items, map[string]any{
			"policyStoreId":       is.PolicyStoreId,
			"identitySourceId":    is.IdentitySourceId,
			"principalEntityType": is.PrincipalEntityType,
			"createdDate":         is.CreatedDate.Format("2006-01-02T15:04:05Z"),
		})
	}
	return jsonOK(map[string]any{"identitySources": items})
}

func handleDeleteIdentitySource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}
	storeId, _ := params["policyStoreId"].(string)
	isId, _ := params["identitySourceId"].(string)
	if awsErr := store.DeleteIdentitySource(storeId, isId); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}
