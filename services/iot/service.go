package iot

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// IoTService is the cloudmock implementation of the AWS IoT Core API.
type IoTService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new IoTService for the given AWS account ID and region.
func New(accountID, region string) *IoTService {
	return &IoTService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *IoTService) Name() string { return "iot" }

// Actions returns the list of IoT API actions supported by this service.
func (s *IoTService) Actions() []service.Action {
	return []service.Action{
		// Things
		{Name: "CreateThing", Method: http.MethodPost, IAMAction: "iot:CreateThing"},
		{Name: "DescribeThing", Method: http.MethodGet, IAMAction: "iot:DescribeThing"},
		{Name: "ListThings", Method: http.MethodGet, IAMAction: "iot:ListThings"},
		{Name: "UpdateThing", Method: http.MethodPatch, IAMAction: "iot:UpdateThing"},
		{Name: "DeleteThing", Method: http.MethodDelete, IAMAction: "iot:DeleteThing"},
		// Thing types
		{Name: "CreateThingType", Method: http.MethodPost, IAMAction: "iot:CreateThingType"},
		{Name: "DescribeThingType", Method: http.MethodGet, IAMAction: "iot:DescribeThingType"},
		{Name: "ListThingTypes", Method: http.MethodGet, IAMAction: "iot:ListThingTypes"},
		{Name: "DeleteThingType", Method: http.MethodDelete, IAMAction: "iot:DeleteThingType"},
		// Thing groups
		{Name: "CreateThingGroup", Method: http.MethodPost, IAMAction: "iot:CreateThingGroup"},
		{Name: "DescribeThingGroup", Method: http.MethodGet, IAMAction: "iot:DescribeThingGroup"},
		{Name: "ListThingGroups", Method: http.MethodGet, IAMAction: "iot:ListThingGroups"},
		{Name: "DeleteThingGroup", Method: http.MethodDelete, IAMAction: "iot:DeleteThingGroup"},
		{Name: "AddThingToThingGroup", Method: http.MethodPut, IAMAction: "iot:AddThingToThingGroup"},
		{Name: "RemoveThingFromThingGroup", Method: http.MethodPut, IAMAction: "iot:RemoveThingFromThingGroup"},
		// Policies
		{Name: "CreatePolicy", Method: http.MethodPost, IAMAction: "iot:CreatePolicy"},
		{Name: "GetPolicy", Method: http.MethodGet, IAMAction: "iot:GetPolicy"},
		{Name: "ListPolicies", Method: http.MethodGet, IAMAction: "iot:ListPolicies"},
		{Name: "DeletePolicy", Method: http.MethodDelete, IAMAction: "iot:DeletePolicy"},
		{Name: "AttachPolicy", Method: http.MethodPut, IAMAction: "iot:AttachPolicy"},
		{Name: "DetachPolicy", Method: http.MethodPost, IAMAction: "iot:DetachPolicy"},
		{Name: "ListAttachedPolicies", Method: http.MethodPost, IAMAction: "iot:ListAttachedPolicies"},
		// Certificates
		{Name: "CreateKeysAndCertificate", Method: http.MethodPost, IAMAction: "iot:CreateKeysAndCertificate"},
		{Name: "DescribeCertificate", Method: http.MethodGet, IAMAction: "iot:DescribeCertificate"},
		{Name: "ListCertificates", Method: http.MethodGet, IAMAction: "iot:ListCertificates"},
		{Name: "DeleteCertificate", Method: http.MethodDelete, IAMAction: "iot:DeleteCertificate"},
		{Name: "AttachThingPrincipal", Method: http.MethodPut, IAMAction: "iot:AttachThingPrincipal"},
		{Name: "DetachThingPrincipal", Method: http.MethodDelete, IAMAction: "iot:DetachThingPrincipal"},
		{Name: "ListThingPrincipals", Method: http.MethodGet, IAMAction: "iot:ListThingPrincipals"},
		// Topic rules
		{Name: "CreateTopicRule", Method: http.MethodPost, IAMAction: "iot:CreateTopicRule"},
		{Name: "GetTopicRule", Method: http.MethodGet, IAMAction: "iot:GetTopicRule"},
		{Name: "ListTopicRules", Method: http.MethodGet, IAMAction: "iot:ListTopicRules"},
		{Name: "DeleteTopicRule", Method: http.MethodDelete, IAMAction: "iot:DeleteTopicRule"},
		// Tags
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "iot:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "iot:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodGet, IAMAction: "iot:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *IoTService) HealthCheck() error { return nil }

// HandleRequest routes an incoming IoT request to the appropriate handler.
func (s *IoTService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	var params map[string]any
	if len(ctx.Body) > 0 {
		json.Unmarshal(ctx.Body, &params)
	}
	if params == nil {
		params = make(map[string]any)
	}
	for k, v := range ctx.Params {
		if _, exists := params[k]; !exists {
			params[k] = v
		}
	}

	switch ctx.Action {
	case "CreateThing":
		return handleCreateThing(params, s.store)
	case "DescribeThing":
		return handleDescribeThing(params, s.store)
	case "ListThings":
		return handleListThings(s.store)
	case "UpdateThing":
		return handleUpdateThing(params, s.store)
	case "DeleteThing":
		return handleDeleteThing(params, s.store)
	case "CreateThingType":
		return handleCreateThingType(params, s.store)
	case "DescribeThingType":
		return handleDescribeThingType(params, s.store)
	case "ListThingTypes":
		return handleListThingTypes(s.store)
	case "DeleteThingType":
		return handleDeleteThingType(params, s.store)
	case "CreateThingGroup":
		return handleCreateThingGroup(params, s.store)
	case "DescribeThingGroup":
		return handleDescribeThingGroup(params, s.store)
	case "ListThingGroups":
		return handleListThingGroups(s.store)
	case "DeleteThingGroup":
		return handleDeleteThingGroup(params, s.store)
	case "AddThingToThingGroup":
		return handleAddThingToThingGroup(params, s.store)
	case "RemoveThingFromThingGroup":
		return handleRemoveThingFromThingGroup(params, s.store)
	case "CreatePolicy":
		return handleCreatePolicy(params, s.store)
	case "GetPolicy":
		return handleGetPolicy(params, s.store)
	case "ListPolicies":
		return handleListPolicies(s.store)
	case "DeletePolicy":
		return handleDeletePolicy(params, s.store)
	case "AttachPolicy":
		return handleAttachPolicy(params, s.store)
	case "DetachPolicy":
		return handleDetachPolicy(params, s.store)
	case "ListAttachedPolicies":
		return handleListAttachedPolicies(params, s.store)
	case "CreateKeysAndCertificate":
		return handleCreateKeysAndCertificate(params, s.store)
	case "DescribeCertificate":
		return handleDescribeCertificate(params, s.store)
	case "ListCertificates":
		return handleListCertificates(s.store)
	case "DeleteCertificate":
		return handleDeleteCertificate(params, s.store)
	case "AttachThingPrincipal":
		return handleAttachThingPrincipal(params, s.store)
	case "DetachThingPrincipal":
		return handleDetachThingPrincipal(params, s.store)
	case "ListThingPrincipals":
		return handleListThingPrincipals(params, s.store)
	case "CreateTopicRule":
		return handleCreateTopicRule(params, s.store)
	case "GetTopicRule":
		return handleGetTopicRule(params, s.store)
	case "ListTopicRules":
		return handleListTopicRules(s.store)
	case "DeleteTopicRule":
		return handleDeleteTopicRule(params, s.store)
	case "TagResource":
		return handleTagResource(params, s.store)
	case "UntagResource":
		return handleUntagResource(params, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(params, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
