package elasticloadbalancing

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ServiceLocator provides access to other services for cross-service communication.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// ELBService is the cloudmock implementation of the AWS Elastic Load Balancing v2 API.
type ELBService struct {
	store   *Store
	locator ServiceLocator
}

// New returns a new ELBService for the given AWS account ID and region.
func New(accountID, region string) *ELBService {
	return &ELBService{
		store: NewStore(accountID, region),
	}
}

// NewWithLocator returns a new ELBService with a service locator for cross-service communication.
func NewWithLocator(accountID, region string, locator ServiceLocator) *ELBService {
	return &ELBService{
		store:   NewStore(accountID, region),
		locator: locator,
	}
}

// SetLocator sets the service locator for cross-service communication.
func (s *ELBService) SetLocator(locator ServiceLocator) {
	s.locator = locator
}

// Name returns the AWS service name used for routing.
func (s *ELBService) Name() string { return "elasticloadbalancing" }

// Actions returns the list of ELBv2 API actions supported by this service.
func (s *ELBService) Actions() []service.Action {
	return []service.Action{
		// Load Balancers
		{Name: "CreateLoadBalancer", Method: http.MethodPost, IAMAction: "elasticloadbalancing:CreateLoadBalancer"},
		{Name: "DescribeLoadBalancers", Method: http.MethodPost, IAMAction: "elasticloadbalancing:DescribeLoadBalancers"},
		{Name: "DeleteLoadBalancer", Method: http.MethodPost, IAMAction: "elasticloadbalancing:DeleteLoadBalancer"},
		{Name: "ModifyLoadBalancerAttributes", Method: http.MethodPost, IAMAction: "elasticloadbalancing:ModifyLoadBalancerAttributes"},
		{Name: "DescribeLoadBalancerAttributes", Method: http.MethodPost, IAMAction: "elasticloadbalancing:DescribeLoadBalancerAttributes"},
		{Name: "SetSecurityGroups", Method: http.MethodPost, IAMAction: "elasticloadbalancing:SetSecurityGroups"},
		{Name: "SetSubnets", Method: http.MethodPost, IAMAction: "elasticloadbalancing:SetSubnets"},
		// Target Groups
		{Name: "CreateTargetGroup", Method: http.MethodPost, IAMAction: "elasticloadbalancing:CreateTargetGroup"},
		{Name: "DescribeTargetGroups", Method: http.MethodPost, IAMAction: "elasticloadbalancing:DescribeTargetGroups"},
		{Name: "DeleteTargetGroup", Method: http.MethodPost, IAMAction: "elasticloadbalancing:DeleteTargetGroup"},
		{Name: "ModifyTargetGroup", Method: http.MethodPost, IAMAction: "elasticloadbalancing:ModifyTargetGroup"},
		{Name: "DescribeTargetGroupAttributes", Method: http.MethodPost, IAMAction: "elasticloadbalancing:DescribeTargetGroupAttributes"},
		{Name: "ModifyTargetGroupAttributes", Method: http.MethodPost, IAMAction: "elasticloadbalancing:ModifyTargetGroupAttributes"},
		// Targets
		{Name: "RegisterTargets", Method: http.MethodPost, IAMAction: "elasticloadbalancing:RegisterTargets"},
		{Name: "DeregisterTargets", Method: http.MethodPost, IAMAction: "elasticloadbalancing:DeregisterTargets"},
		{Name: "DescribeTargetHealth", Method: http.MethodPost, IAMAction: "elasticloadbalancing:DescribeTargetHealth"},
		// Listeners
		{Name: "CreateListener", Method: http.MethodPost, IAMAction: "elasticloadbalancing:CreateListener"},
		{Name: "DescribeListeners", Method: http.MethodPost, IAMAction: "elasticloadbalancing:DescribeListeners"},
		{Name: "DeleteListener", Method: http.MethodPost, IAMAction: "elasticloadbalancing:DeleteListener"},
		{Name: "ModifyListener", Method: http.MethodPost, IAMAction: "elasticloadbalancing:ModifyListener"},
		// Rules
		{Name: "CreateRule", Method: http.MethodPost, IAMAction: "elasticloadbalancing:CreateRule"},
		{Name: "DescribeRules", Method: http.MethodPost, IAMAction: "elasticloadbalancing:DescribeRules"},
		{Name: "DeleteRule", Method: http.MethodPost, IAMAction: "elasticloadbalancing:DeleteRule"},
		{Name: "ModifyRule", Method: http.MethodPost, IAMAction: "elasticloadbalancing:ModifyRule"},
		{Name: "SetRulePriorities", Method: http.MethodPost, IAMAction: "elasticloadbalancing:SetRulePriorities"},
		// Tags
		{Name: "AddTags", Method: http.MethodPost, IAMAction: "elasticloadbalancing:AddTags"},
		{Name: "RemoveTags", Method: http.MethodPost, IAMAction: "elasticloadbalancing:RemoveTags"},
		{Name: "DescribeTags", Method: http.MethodPost, IAMAction: "elasticloadbalancing:DescribeTags"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *ELBService) HealthCheck() error { return nil }

// GetStore returns the store for cross-service access (e.g., AutoScaling registering targets).
func (s *ELBService) GetStore() *Store { return s.store }

// HandleRequest routes an incoming ELBv2 request to the appropriate handler.
// ELBv2 uses the query protocol (form-encoded, XML responses).
func (s *ELBService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	action := ctx.Action
	if action == "" {
		form := parseForm(ctx)
		action = form.Get("Action")
	}

	switch action {
	// Load Balancers
	case "CreateLoadBalancer":
		return handleCreateLoadBalancer(ctx, s.store)
	case "DescribeLoadBalancers":
		return handleDescribeLoadBalancers(ctx, s.store)
	case "DeleteLoadBalancer":
		return handleDeleteLoadBalancer(ctx, s.store)
	case "ModifyLoadBalancerAttributes":
		return handleModifyLoadBalancerAttributes(ctx, s.store)
	case "DescribeLoadBalancerAttributes":
		return handleDescribeLoadBalancerAttributes(ctx, s.store)
	case "SetSecurityGroups":
		return handleSetSecurityGroups(ctx, s.store)
	case "SetSubnets":
		return handleSetSubnets(ctx, s.store)
	// Target Groups
	case "CreateTargetGroup":
		return handleCreateTargetGroup(ctx, s.store)
	case "DescribeTargetGroups":
		return handleDescribeTargetGroups(ctx, s.store)
	case "DeleteTargetGroup":
		return handleDeleteTargetGroup(ctx, s.store)
	case "ModifyTargetGroup":
		return handleModifyTargetGroup(ctx, s.store)
	case "DescribeTargetGroupAttributes":
		return handleDescribeTargetGroupAttributes(ctx, s.store)
	case "ModifyTargetGroupAttributes":
		return handleModifyTargetGroupAttributes(ctx, s.store)
	// Targets
	case "RegisterTargets":
		return handleRegisterTargets(ctx, s.store)
	case "DeregisterTargets":
		return handleDeregisterTargets(ctx, s.store)
	case "DescribeTargetHealth":
		return handleDescribeTargetHealth(ctx, s.store)
	// Listeners
	case "CreateListener":
		return handleCreateListener(ctx, s.store)
	case "DescribeListeners":
		return handleDescribeListeners(ctx, s.store)
	case "DeleteListener":
		return handleDeleteListener(ctx, s.store)
	case "ModifyListener":
		return handleModifyListener(ctx, s.store)
	// Rules
	case "CreateRule":
		return handleCreateRule(ctx, s.store)
	case "DescribeRules":
		return handleDescribeRules(ctx, s.store)
	case "DeleteRule":
		return handleDeleteRule(ctx, s.store)
	case "ModifyRule":
		return handleModifyRule(ctx, s.store)
	case "SetRulePriorities":
		return handleSetRulePriorities(ctx, s.store)
	// Tags
	case "AddTags":
		return handleAddTags(ctx, s.store)
	case "RemoveTags":
		return handleRemoveTags(ctx, s.store)
	case "DescribeTags":
		return handleDescribeTagsForResource(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatXML},
			service.NewAWSError("InvalidAction",
				"The action "+action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
