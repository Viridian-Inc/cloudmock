package autoscaling

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/eventbus"
	"github.com/neureaux/cloudmock/pkg/service"
)

// ServiceLocator provides access to other services for cross-service communication.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// AutoScalingService is the cloudmock implementation of the AWS Auto Scaling API.
type AutoScalingService struct {
	store   *Store
	locator ServiceLocator
	bus     *eventbus.Bus
}

// New returns a new AutoScalingService for the given AWS account ID and region.
func New(accountID, region string) *AutoScalingService {
	return &AutoScalingService{
		store: NewStore(accountID, region),
	}
}

// NewWithLocator returns a new AutoScalingService with a service locator for cross-service communication.
func NewWithLocator(accountID, region string, locator ServiceLocator) *AutoScalingService {
	return &AutoScalingService{
		store:   NewStore(accountID, region),
		locator: locator,
	}
}

// SetLocator sets the service locator for cross-service communication (EC2 instance creation).
func (s *AutoScalingService) SetLocator(locator ServiceLocator) {
	s.locator = locator
}

// SetEventBus sets the event bus for publishing autoscaling events.
func (s *AutoScalingService) SetEventBus(bus *eventbus.Bus) {
	s.bus = bus
}

// Name returns the AWS service name used for routing.
func (s *AutoScalingService) Name() string { return "autoscaling" }

// Actions returns the list of Auto Scaling API actions supported by this service.
func (s *AutoScalingService) Actions() []service.Action {
	return []service.Action{
		// Auto Scaling Groups
		{Name: "CreateAutoScalingGroup", Method: http.MethodPost, IAMAction: "autoscaling:CreateAutoScalingGroup"},
		{Name: "DescribeAutoScalingGroups", Method: http.MethodPost, IAMAction: "autoscaling:DescribeAutoScalingGroups"},
		{Name: "UpdateAutoScalingGroup", Method: http.MethodPost, IAMAction: "autoscaling:UpdateAutoScalingGroup"},
		{Name: "DeleteAutoScalingGroup", Method: http.MethodPost, IAMAction: "autoscaling:DeleteAutoScalingGroup"},
		// Launch Configurations
		{Name: "CreateLaunchConfiguration", Method: http.MethodPost, IAMAction: "autoscaling:CreateLaunchConfiguration"},
		{Name: "DescribeLaunchConfigurations", Method: http.MethodPost, IAMAction: "autoscaling:DescribeLaunchConfigurations"},
		{Name: "DeleteLaunchConfiguration", Method: http.MethodPost, IAMAction: "autoscaling:DeleteLaunchConfiguration"},
		// Scaling Policies
		{Name: "PutScalingPolicy", Method: http.MethodPost, IAMAction: "autoscaling:PutScalingPolicy"},
		{Name: "DescribePolicies", Method: http.MethodPost, IAMAction: "autoscaling:DescribePolicies"},
		{Name: "DeletePolicy", Method: http.MethodPost, IAMAction: "autoscaling:DeletePolicy"},
		// Capacity & Instances
		{Name: "SetDesiredCapacity", Method: http.MethodPost, IAMAction: "autoscaling:SetDesiredCapacity"},
		{Name: "DescribeAutoScalingInstances", Method: http.MethodPost, IAMAction: "autoscaling:DescribeAutoScalingInstances"},
		{Name: "AttachInstances", Method: http.MethodPost, IAMAction: "autoscaling:AttachInstances"},
		{Name: "DetachInstances", Method: http.MethodPost, IAMAction: "autoscaling:DetachInstances"},
		// Tags
		{Name: "CreateOrUpdateTags", Method: http.MethodPost, IAMAction: "autoscaling:CreateOrUpdateTags"},
		{Name: "DescribeTags", Method: http.MethodPost, IAMAction: "autoscaling:DescribeTags"},
		{Name: "DeleteTags", Method: http.MethodPost, IAMAction: "autoscaling:DeleteTags"},
	}
}

// HealthCheck always returns nil (no external dependencies).
func (s *AutoScalingService) HealthCheck() error { return nil }

// HandleRequest routes an incoming Auto Scaling request to the appropriate handler.
// Auto Scaling uses the query protocol (form-encoded, XML responses).
func (s *AutoScalingService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	action := ctx.Action
	if action == "" {
		form := parseForm(ctx)
		action = form.Get("Action")
	}

	switch action {
	// Auto Scaling Groups
	case "CreateAutoScalingGroup":
		return handleCreateAutoScalingGroup(ctx, s.store, s.locator, s.bus)
	case "DescribeAutoScalingGroups":
		return handleDescribeAutoScalingGroups(ctx, s.store)
	case "UpdateAutoScalingGroup":
		return handleUpdateAutoScalingGroup(ctx, s.store, s.locator, s.bus)
	case "DeleteAutoScalingGroup":
		return handleDeleteAutoScalingGroup(ctx, s.store, s.locator, s.bus)
	// Launch Configurations
	case "CreateLaunchConfiguration":
		return handleCreateLaunchConfiguration(ctx, s.store)
	case "DescribeLaunchConfigurations":
		return handleDescribeLaunchConfigurations(ctx, s.store)
	case "DeleteLaunchConfiguration":
		return handleDeleteLaunchConfiguration(ctx, s.store)
	// Scaling Policies
	case "PutScalingPolicy":
		return handlePutScalingPolicy(ctx, s.store)
	case "DescribePolicies":
		return handleDescribePolicies(ctx, s.store)
	case "DeletePolicy":
		return handleDeletePolicy(ctx, s.store)
	// Capacity & Instances
	case "SetDesiredCapacity":
		return handleSetDesiredCapacity(ctx, s.store, s.locator, s.bus)
	case "DescribeAutoScalingInstances":
		return handleDescribeAutoScalingInstances(ctx, s.store)
	case "AttachInstances":
		return handleAttachInstances(ctx, s.store)
	case "DetachInstances":
		return handleDetachInstances(ctx, s.store)
	// Tags
	case "CreateOrUpdateTags":
		return handleCreateOrUpdateTags(ctx, s.store)
	case "DescribeTags":
		return handleDescribeTags(ctx, s.store)
	case "DeleteTags":
		return handleDeleteTags(ctx, s.store)
	default:
		return &service.Response{Format: service.FormatXML},
			service.NewAWSError("InvalidAction",
				"The action "+action+" is not valid for this web service.",
				http.StatusBadRequest)
	}
}
