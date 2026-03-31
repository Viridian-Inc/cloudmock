package shield

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ShieldService is the cloudmock implementation of the AWS Shield API.
type ShieldService struct {
	store     *Store
	accountID string
	region    string
}

// New returns a new ShieldService for the given AWS account ID and region.
func New(accountID, region string) *ShieldService {
	return &ShieldService{
		store:     NewStore(accountID, region),
		accountID: accountID,
		region:    region,
	}
}

// Name returns the AWS service name used for routing.
func (s *ShieldService) Name() string { return "shield" }

// Actions returns the list of Shield API actions supported by this service.
func (s *ShieldService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateProtection", Method: http.MethodPost, IAMAction: "shield:CreateProtection"},
		{Name: "DescribeProtection", Method: http.MethodPost, IAMAction: "shield:DescribeProtection"},
		{Name: "ListProtections", Method: http.MethodPost, IAMAction: "shield:ListProtections"},
		{Name: "DeleteProtection", Method: http.MethodPost, IAMAction: "shield:DeleteProtection"},
		{Name: "CreateSubscription", Method: http.MethodPost, IAMAction: "shield:CreateSubscription"},
		{Name: "DescribeSubscription", Method: http.MethodPost, IAMAction: "shield:DescribeSubscription"},
		{Name: "DescribeAttack", Method: http.MethodPost, IAMAction: "shield:DescribeAttack"},
		{Name: "ListAttacks", Method: http.MethodPost, IAMAction: "shield:ListAttacks"},
		{Name: "CreateProtectionGroup", Method: http.MethodPost, IAMAction: "shield:CreateProtectionGroup"},
		{Name: "DescribeProtectionGroup", Method: http.MethodPost, IAMAction: "shield:DescribeProtectionGroup"},
		{Name: "ListProtectionGroups", Method: http.MethodPost, IAMAction: "shield:ListProtectionGroups"},
		{Name: "UpdateProtectionGroup", Method: http.MethodPost, IAMAction: "shield:UpdateProtectionGroup"},
		{Name: "DeleteProtectionGroup", Method: http.MethodPost, IAMAction: "shield:DeleteProtectionGroup"},
		{Name: "TagResource", Method: http.MethodPost, IAMAction: "shield:TagResource"},
		{Name: "UntagResource", Method: http.MethodPost, IAMAction: "shield:UntagResource"},
		{Name: "ListTagsForResource", Method: http.MethodPost, IAMAction: "shield:ListTagsForResource"},
	}
}

// HealthCheck always returns nil.
func (s *ShieldService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Shield request to the appropriate handler.
func (s *ShieldService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	switch ctx.Action {
	case "CreateProtection":
		return handleCreateProtection(ctx, s.store)
	case "DescribeProtection":
		return handleDescribeProtection(ctx, s.store)
	case "ListProtections":
		return handleListProtections(ctx, s.store)
	case "DeleteProtection":
		return handleDeleteProtection(ctx, s.store)
	case "CreateSubscription":
		return handleCreateSubscription(ctx, s.store)
	case "DescribeSubscription":
		return handleDescribeSubscription(ctx, s.store)
	case "DescribeAttack":
		return handleDescribeAttack(ctx, s.store)
	case "ListAttacks":
		return handleListAttacks(ctx, s.store)
	case "CreateProtectionGroup":
		return handleCreateProtectionGroup(ctx, s.store)
	case "DescribeProtectionGroup":
		return handleDescribeProtectionGroup(ctx, s.store)
	case "ListProtectionGroups":
		return handleListProtectionGroups(ctx, s.store)
	case "UpdateProtectionGroup":
		return handleUpdateProtectionGroup(ctx, s.store)
	case "DeleteProtectionGroup":
		return handleDeleteProtectionGroup(ctx, s.store)
	case "TagResource":
		return handleTagResource(ctx, s.store)
	case "UntagResource":
		return handleUntagResource(ctx, s.store)
	case "ListTagsForResource":
		return handleListTagsForResource(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatJSON},
			service.NewAWSError("InvalidAction",
				"The action "+ctx.Action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
