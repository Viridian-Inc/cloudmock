package securityhub

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Stored types ─────────────────────────────────────────────────────────────

// StoredHub represents the per-account Security Hub enablement state.
type StoredHub struct {
	HubArn                  string
	SubscribedAt            time.Time
	AutoEnableControls      bool
	ControlFindingGenerator string
	Tags                    map[string]string
}

// StoredStandardsSubscription tracks an enabled standard.
type StoredStandardsSubscription struct {
	StandardsSubscriptionArn string
	StandardsArn             string
	StandardsInput           map[string]string
	StandardsStatus          string
	StandardsStatusReason    map[string]any
}

// StoredProductSubscription tracks an enabled product import.
type StoredProductSubscription struct {
	ProductSubscriptionArn string
	ProductArn             string
}

// StoredInsight is a custom Security Hub insight.
type StoredInsight struct {
	InsightArn       string
	Name             string
	Filters          map[string]any
	GroupByAttribute string
}

// StoredFinding is an ASFF security finding.
type StoredFinding struct {
	ID     string
	Data   map[string]any
}

// StoredActionTarget is a custom action used by Security Hub findings.
type StoredActionTarget struct {
	ActionTargetArn string
	Name            string
	Description     string
}

// StoredInvitation models an inbound Security Hub member invitation.
type StoredInvitation struct {
	AccountID    string
	InvitationID string
	InvitedAt    time.Time
	MemberStatus string
}

// StoredMember models a Security Hub member account.
type StoredMember struct {
	AccountID       string
	Email           string
	MasterID        string
	AdministratorID string
	MemberStatus    string
	InvitedAt       time.Time
	UpdatedAt       time.Time
}

// StoredFindingAggregator persists a cross-region aggregator.
type StoredFindingAggregator struct {
	FindingAggregatorArn     string
	FindingAggregationRegion string
	RegionLinkingMode        string
	Regions                  []string
}

// StoredAutomationRule persists an automation rule (V1).
type StoredAutomationRule struct {
	RuleArn     string
	RuleID      string
	RuleName    string
	RuleStatus  string
	RuleOrder   int
	Description string
	IsTerminal  bool
	Criteria    map[string]any
	Actions     []map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   string
}

// StoredAutomationRuleV2 persists a V2 automation rule.
type StoredAutomationRuleV2 struct {
	RuleArn     string
	RuleID      string
	RuleName    string
	RuleStatus  string
	RuleOrder   float64
	Description string
	Criteria    map[string]any
	Actions     []map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// StoredConfigurationPolicy persists a configuration policy.
type StoredConfigurationPolicy struct {
	Arn                 string
	ID                  string
	Name                string
	Description         string
	UpdatedAt           time.Time
	CreatedAt           time.Time
	ConfigurationPolicy map[string]any
}

// StoredConfigurationPolicyAssociation persists a target binding.
type StoredConfigurationPolicyAssociation struct {
	TargetID                  string
	TargetType                string
	ConfigurationPolicyID     string
	AssociationType           string
	AssociationStatus         string
	AssociationStatusMessage  string
	UpdatedAt                 time.Time
}

// StoredConnectorV2 persists a V2 connector.
type StoredConnectorV2 struct {
	ConnectorArn       string
	ConnectorID        string
	Name               string
	Description        string
	ProviderName       string
	HealthStatus       string
	HealthMessage      string
	ClientSecret       string
	KmsKeyArn          string
	CreatedAt          time.Time
	LastUpdatedAt      time.Time
	ProviderSummary    map[string]any
	Tags               map[string]string
}

// StoredAggregatorV2 persists a V2 aggregator.
type StoredAggregatorV2 struct {
	AggregatorV2Arn   string
	AggregatorV2ID    string
	AggregationRegion string
	RegionLinkingMode string
	LinkedRegions     []string
}

// Store is the in-memory data store for Security Hub.
type Store struct {
	mu        sync.RWMutex
	accountID string
	region    string

	hub                          *StoredHub
	hubV2                        *StoredHub
	standardsSubscriptions       map[string]*StoredStandardsSubscription // arn -> sub
	productSubscriptions         map[string]*StoredProductSubscription   // arn -> sub
	insights                     map[string]*StoredInsight               // arn -> insight
	findings                     map[string]*StoredFinding               // id -> finding
	actionTargets                map[string]*StoredActionTarget          // arn -> action target
	invitations                  map[string]*StoredInvitation            // accountID -> invitation
	members                      map[string]*StoredMember                // accountID -> member
	findingAggregators           map[string]*StoredFindingAggregator     // arn -> aggregator
	automationRules              map[string]*StoredAutomationRule        // ruleArn -> rule
	automationRulesV2            map[string]*StoredAutomationRuleV2      // ruleArn -> rule
	configurationPolicies        map[string]*StoredConfigurationPolicy   // id -> policy
	configurationAssociations    map[string]*StoredConfigurationPolicyAssociation // targetID -> assoc
	connectorsV2                 map[string]*StoredConnectorV2           // id -> connector
	aggregatorsV2                map[string]*StoredAggregatorV2          // id -> aggregator
	resourceTags                 map[string]map[string]string            // arn -> tags
	enabledImportProducts        map[string]bool                         // productArn -> enabled
	administratorAccount         string
	administratorRelationship    string
	delegatedAdministratorAcct   string
	orgConfig                    map[string]any
	hubConfig                    map[string]any
	securityControlOverrides     map[string]map[string]any // controlID -> overrides
}

// NewStore creates an empty Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID:                 accountID,
		region:                    region,
		standardsSubscriptions:    make(map[string]*StoredStandardsSubscription),
		productSubscriptions:      make(map[string]*StoredProductSubscription),
		insights:                  make(map[string]*StoredInsight),
		findings:                  make(map[string]*StoredFinding),
		actionTargets:             make(map[string]*StoredActionTarget),
		invitations:               make(map[string]*StoredInvitation),
		members:                   make(map[string]*StoredMember),
		findingAggregators:        make(map[string]*StoredFindingAggregator),
		automationRules:           make(map[string]*StoredAutomationRule),
		automationRulesV2:         make(map[string]*StoredAutomationRuleV2),
		configurationPolicies:     make(map[string]*StoredConfigurationPolicy),
		configurationAssociations: make(map[string]*StoredConfigurationPolicyAssociation),
		connectorsV2:              make(map[string]*StoredConnectorV2),
		aggregatorsV2:             make(map[string]*StoredAggregatorV2),
		resourceTags:              make(map[string]map[string]string),
		enabledImportProducts:     make(map[string]bool),
		orgConfig: map[string]any{
			"AutoEnable":          false,
			"AutoEnableStandards": "NONE",
			"OrganizationConfiguration": map[string]any{
				"ConfigurationType": "LOCAL",
				"Status":            "ENABLED",
			},
		},
		hubConfig:                map[string]any{},
		securityControlOverrides: make(map[string]map[string]any),
	}
}

// Reset clears all in-memory state.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hub = nil
	s.hubV2 = nil
	s.standardsSubscriptions = make(map[string]*StoredStandardsSubscription)
	s.productSubscriptions = make(map[string]*StoredProductSubscription)
	s.insights = make(map[string]*StoredInsight)
	s.findings = make(map[string]*StoredFinding)
	s.actionTargets = make(map[string]*StoredActionTarget)
	s.invitations = make(map[string]*StoredInvitation)
	s.members = make(map[string]*StoredMember)
	s.findingAggregators = make(map[string]*StoredFindingAggregator)
	s.automationRules = make(map[string]*StoredAutomationRule)
	s.automationRulesV2 = make(map[string]*StoredAutomationRuleV2)
	s.configurationPolicies = make(map[string]*StoredConfigurationPolicy)
	s.configurationAssociations = make(map[string]*StoredConfigurationPolicyAssociation)
	s.connectorsV2 = make(map[string]*StoredConnectorV2)
	s.aggregatorsV2 = make(map[string]*StoredAggregatorV2)
	s.resourceTags = make(map[string]map[string]string)
	s.enabledImportProducts = make(map[string]bool)
	s.administratorAccount = ""
	s.administratorRelationship = ""
	s.delegatedAdministratorAcct = ""
	s.orgConfig = map[string]any{
		"AutoEnable":          false,
		"AutoEnableStandards": "NONE",
		"OrganizationConfiguration": map[string]any{
			"ConfigurationType": "LOCAL",
			"Status":            "ENABLED",
		},
	}
	s.hubConfig = map[string]any{}
	s.securityControlOverrides = make(map[string]map[string]any)
}

// ── Hub ──────────────────────────────────────────────────────────────────────

// EnableHub enables Security Hub for the account.
func (s *Store) EnableHub(autoEnableControls bool, controlFindingGenerator string, tags map[string]string) (*StoredHub, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hub != nil {
		return nil, service.NewAWSError("ResourceConflictException",
			"Security Hub is already enabled for this account", http.StatusConflict)
	}
	if controlFindingGenerator == "" {
		controlFindingGenerator = "SECURITY_CONTROL"
	}
	hub := &StoredHub{
		HubArn:                  fmt.Sprintf("arn:aws:securityhub:%s:%s:hub/default", s.region, s.accountID),
		SubscribedAt:            time.Now().UTC(),
		AutoEnableControls:      autoEnableControls,
		ControlFindingGenerator: controlFindingGenerator,
		Tags:                    copyStringMap(tags),
	}
	s.hub = hub
	if s.resourceTags[hub.HubArn] == nil {
		s.resourceTags[hub.HubArn] = make(map[string]string)
	}
	for k, v := range tags {
		s.resourceTags[hub.HubArn][k] = v
	}
	return hub, nil
}

// DisableHub clears the hub enablement state and all dependent resources.
func (s *Store) DisableHub() *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hub == nil {
		return service.NewAWSError("InvalidAccessException",
			"Security Hub is not enabled for this account", http.StatusBadRequest)
	}
	s.hub = nil
	s.standardsSubscriptions = make(map[string]*StoredStandardsSubscription)
	s.productSubscriptions = make(map[string]*StoredProductSubscription)
	s.insights = make(map[string]*StoredInsight)
	s.findings = make(map[string]*StoredFinding)
	s.actionTargets = make(map[string]*StoredActionTarget)
	return nil
}

// GetHub returns the active hub or an error if Security Hub is not enabled.
func (s *Store) GetHub() (*StoredHub, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.hub == nil {
		return nil, service.NewAWSError("InvalidAccessException",
			"Security Hub is not enabled for this account", http.StatusBadRequest)
	}
	return s.hub, nil
}

// UpdateHubConfig updates auto-enable settings.
func (s *Store) UpdateHubConfig(autoEnableControls *bool, controlFindingGenerator string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hub == nil {
		return service.NewAWSError("InvalidAccessException",
			"Security Hub is not enabled for this account", http.StatusBadRequest)
	}
	if autoEnableControls != nil {
		s.hub.AutoEnableControls = *autoEnableControls
	}
	if controlFindingGenerator != "" {
		s.hub.ControlFindingGenerator = controlFindingGenerator
	}
	return nil
}

// EnableHubV2 enables Security Hub V2.
func (s *Store) EnableHubV2(tags map[string]string) (*StoredHub, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hubV2 != nil {
		return nil, service.NewAWSError("ResourceConflictException",
			"Security Hub V2 is already enabled for this account", http.StatusConflict)
	}
	hub := &StoredHub{
		HubArn:       fmt.Sprintf("arn:aws:securityhub:%s:%s:hub/v2/default", s.region, s.accountID),
		SubscribedAt: time.Now().UTC(),
		Tags:         copyStringMap(tags),
	}
	s.hubV2 = hub
	return hub, nil
}

// DisableHubV2 disables Security Hub V2.
func (s *Store) DisableHubV2() *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hubV2 == nil {
		return service.NewAWSError("InvalidAccessException",
			"Security Hub V2 is not enabled for this account", http.StatusBadRequest)
	}
	s.hubV2 = nil
	return nil
}

// GetHubV2 returns the V2 hub or an error.
func (s *Store) GetHubV2() (*StoredHub, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.hubV2 == nil {
		return nil, service.NewAWSError("InvalidAccessException",
			"Security Hub V2 is not enabled for this account", http.StatusBadRequest)
	}
	return s.hubV2, nil
}

// ── Standards ────────────────────────────────────────────────────────────────

// EnableStandards subscribes the account to a standard.
func (s *Store) EnableStandards(standardsArn string, input map[string]string) (*StoredStandardsSubscription, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if standardsArn == "" {
		return nil, service.NewAWSError("InvalidInputException",
			"StandardsArn is required", http.StatusBadRequest)
	}
	subID := generateID()
	sub := &StoredStandardsSubscription{
		StandardsSubscriptionArn: fmt.Sprintf("arn:aws:securityhub:%s:%s:subscription/%s/%s", s.region, s.accountID, lastSegment(standardsArn), subID),
		StandardsArn:             standardsArn,
		StandardsInput:           copyStringMap(input),
		StandardsStatus:          "READY",
	}
	s.standardsSubscriptions[sub.StandardsSubscriptionArn] = sub
	return sub, nil
}

// DisableStandards removes a standards subscription.
func (s *Store) DisableStandards(subscriptionArn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.standardsSubscriptions[subscriptionArn]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Standards subscription not found: "+subscriptionArn, http.StatusBadRequest)
	}
	delete(s.standardsSubscriptions, subscriptionArn)
	return nil
}

// ListStandardsSubscriptions returns all enabled standards.
func (s *Store) ListStandardsSubscriptions() []*StoredStandardsSubscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredStandardsSubscription, 0, len(s.standardsSubscriptions))
	for _, sub := range s.standardsSubscriptions {
		out = append(out, sub)
	}
	return out
}

// ── Products ─────────────────────────────────────────────────────────────────

// EnableImportFindingsForProduct subscribes to a product.
func (s *Store) EnableImportFindingsForProduct(productArn string) (*StoredProductSubscription, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if productArn == "" {
		return nil, service.NewAWSError("InvalidInputException",
			"ProductArn is required", http.StatusBadRequest)
	}
	subArn := fmt.Sprintf("arn:aws:securityhub:%s:%s:product-subscription/%s", s.region, s.accountID, lastSegment(productArn))
	if _, ok := s.productSubscriptions[subArn]; ok {
		return nil, service.NewAWSError("ResourceConflictException",
			"Product subscription already exists for "+productArn, http.StatusConflict)
	}
	sub := &StoredProductSubscription{
		ProductSubscriptionArn: subArn,
		ProductArn:             productArn,
	}
	s.productSubscriptions[subArn] = sub
	s.enabledImportProducts[productArn] = true
	return sub, nil
}

// DisableImportFindingsForProduct removes a product subscription by subscription ARN.
func (s *Store) DisableImportFindingsForProduct(subscriptionArn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.productSubscriptions[subscriptionArn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Product subscription not found: "+subscriptionArn, http.StatusBadRequest)
	}
	delete(s.productSubscriptions, subscriptionArn)
	delete(s.enabledImportProducts, sub.ProductArn)
	return nil
}

// ListEnabledProducts returns all enabled product subscriptions.
func (s *Store) ListEnabledProducts() []*StoredProductSubscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredProductSubscription, 0, len(s.productSubscriptions))
	for _, p := range s.productSubscriptions {
		out = append(out, p)
	}
	return out
}

// ── Insights ─────────────────────────────────────────────────────────────────

// CreateInsight stores a new insight.
func (s *Store) CreateInsight(name string, filters map[string]any, groupBy string) (*StoredInsight, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name == "" {
		return nil, service.NewAWSError("InvalidInputException",
			"Name is required", http.StatusBadRequest)
	}
	if groupBy == "" {
		return nil, service.NewAWSError("InvalidInputException",
			"GroupByAttribute is required", http.StatusBadRequest)
	}
	id := generateID()
	insight := &StoredInsight{
		InsightArn:       fmt.Sprintf("arn:aws:securityhub:%s:%s:insight/%s/custom/%s", s.region, s.accountID, s.accountID, id),
		Name:             name,
		Filters:          filters,
		GroupByAttribute: groupBy,
	}
	s.insights[insight.InsightArn] = insight
	return insight, nil
}

// UpdateInsight modifies fields on an existing insight.
func (s *Store) UpdateInsight(arn, name string, filters map[string]any, groupBy string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	insight, ok := s.insights[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Insight not found: "+arn, http.StatusBadRequest)
	}
	if name != "" {
		insight.Name = name
	}
	if filters != nil {
		insight.Filters = filters
	}
	if groupBy != "" {
		insight.GroupByAttribute = groupBy
	}
	return nil
}

// DeleteInsight removes an insight by ARN.
func (s *Store) DeleteInsight(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.insights[arn]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Insight not found: "+arn, http.StatusBadRequest)
	}
	delete(s.insights, arn)
	return nil
}

// ListInsights returns all insights, optionally filtered by ARN.
func (s *Store) ListInsights(arns []string) []*StoredInsight {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(arns) == 0 {
		out := make([]*StoredInsight, 0, len(s.insights))
		for _, i := range s.insights {
			out = append(out, i)
		}
		return out
	}
	out := make([]*StoredInsight, 0, len(arns))
	for _, a := range arns {
		if i, ok := s.insights[a]; ok {
			out = append(out, i)
		}
	}
	return out
}

// ── Findings ─────────────────────────────────────────────────────────────────

// ImportFindings persists ASFF findings.
func (s *Store) ImportFindings(findings []map[string]any) (success int, failed []map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range findings {
		id, _ := f["Id"].(string)
		if id == "" {
			failed = append(failed, map[string]any{
				"Id":           "",
				"ErrorCode":    "InvalidInput",
				"ErrorMessage": "Finding Id is required",
			})
			continue
		}
		s.findings[id] = &StoredFinding{ID: id, Data: f}
		success++
	}
	return success, failed
}

// UpdateFindings applies a batched update to matching findings.
// updates is a flat map of fields to set on every matched finding.
func (s *Store) UpdateFindings(filters map[string]any, updates map[string]any) (matched int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range s.findings {
		if matchesFilters(f.Data, filters) {
			for k, v := range updates {
				f.Data[k] = v
			}
			matched++
		}
	}
	return matched
}

// BatchUpdateFindings updates specific fields by FindingIdentifier.
func (s *Store) BatchUpdateFindings(identifiers []map[string]any, updates map[string]any) (processed []map[string]any, unprocessed []map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ident := range identifiers {
		id, _ := ident["Id"].(string)
		productArn, _ := ident["ProductArn"].(string)
		f, ok := s.findings[id]
		if !ok {
			unprocessed = append(unprocessed, map[string]any{
				"FindingIdentifier": ident,
				"ErrorCode":         "FindingNotFound",
				"ErrorMessage":      "Finding not found: " + id,
			})
			continue
		}
		_ = productArn
		for k, v := range updates {
			f.Data[k] = v
		}
		processed = append(processed, map[string]any{
			"Id":         id,
			"ProductArn": productArn,
		})
	}
	return processed, unprocessed
}

// GetFindings returns findings matching the filter criteria.
func (s *Store) GetFindings(filters map[string]any) []*StoredFinding {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredFinding, 0)
	for _, f := range s.findings {
		if matchesFilters(f.Data, filters) {
			out = append(out, f)
		}
	}
	return out
}

// ── Action Targets ───────────────────────────────────────────────────────────

// CreateActionTarget adds a new custom action target.
func (s *Store) CreateActionTarget(name, description, identifier string) (*StoredActionTarget, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name == "" || description == "" || identifier == "" {
		return nil, service.NewAWSError("InvalidInputException",
			"Name, Description, and Id are required", http.StatusBadRequest)
	}
	arn := fmt.Sprintf("arn:aws:securityhub:%s:%s:action/custom/%s", s.region, s.accountID, identifier)
	if _, exists := s.actionTargets[arn]; exists {
		return nil, service.NewAWSError("ResourceConflictException",
			"Action target already exists: "+arn, http.StatusConflict)
	}
	at := &StoredActionTarget{
		ActionTargetArn: arn,
		Name:            name,
		Description:     description,
	}
	s.actionTargets[arn] = at
	return at, nil
}

// UpdateActionTarget changes fields on an existing action target.
func (s *Store) UpdateActionTarget(arn, name, description string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	at, ok := s.actionTargets[arn]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Action target not found: "+arn, http.StatusBadRequest)
	}
	if name != "" {
		at.Name = name
	}
	if description != "" {
		at.Description = description
	}
	return nil
}

// DeleteActionTarget removes an action target.
func (s *Store) DeleteActionTarget(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.actionTargets[arn]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Action target not found: "+arn, http.StatusBadRequest)
	}
	delete(s.actionTargets, arn)
	return nil
}

// DescribeActionTargets returns matching action targets, or all if arns empty.
func (s *Store) DescribeActionTargets(arns []string) []*StoredActionTarget {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(arns) == 0 {
		out := make([]*StoredActionTarget, 0, len(s.actionTargets))
		for _, at := range s.actionTargets {
			out = append(out, at)
		}
		return out
	}
	out := make([]*StoredActionTarget, 0, len(arns))
	for _, a := range arns {
		if at, ok := s.actionTargets[a]; ok {
			out = append(out, at)
		}
	}
	return out
}

// ── Members & Invitations ────────────────────────────────────────────────────

// CreateMembers stores accounts as Security Hub members.
func (s *Store) CreateMembers(accounts []map[string]any) (created []map[string]any, unprocessed []map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for _, a := range accounts {
		id, _ := a["AccountId"].(string)
		email, _ := a["Email"].(string)
		if id == "" {
			unprocessed = append(unprocessed, map[string]any{
				"AccountId":      "",
				"ProcessingResult": "AccountId is required",
			})
			continue
		}
		s.members[id] = &StoredMember{
			AccountID:    id,
			Email:        email,
			MemberStatus: "Created",
			InvitedAt:    time.Time{},
			UpdatedAt:    now,
		}
		created = append(created, map[string]any{"AccountId": id})
	}
	return created, unprocessed
}

// DeleteMembers removes member accounts.
func (s *Store) DeleteMembers(accountIDs []string) (unprocessed []map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range accountIDs {
		if _, ok := s.members[id]; !ok {
			unprocessed = append(unprocessed, map[string]any{
				"AccountId":        id,
				"ProcessingResult": "Member not found",
			})
			continue
		}
		delete(s.members, id)
	}
	return unprocessed
}

// DisassociateMembers detaches a member without deleting.
func (s *Store) DisassociateMembers(accountIDs []string) (unprocessed []map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range accountIDs {
		m, ok := s.members[id]
		if !ok {
			unprocessed = append(unprocessed, map[string]any{
				"AccountId":        id,
				"ProcessingResult": "Member not found",
			})
			continue
		}
		m.MemberStatus = "Removed"
		m.UpdatedAt = time.Now().UTC()
	}
	return unprocessed
}

// GetMembers returns the requested members.
func (s *Store) GetMembers(accountIDs []string) (members []*StoredMember, unprocessed []map[string]any) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, id := range accountIDs {
		if m, ok := s.members[id]; ok {
			members = append(members, m)
		} else {
			unprocessed = append(unprocessed, map[string]any{
				"AccountId":        id,
				"ProcessingResult": "Member not found",
			})
		}
	}
	return members, unprocessed
}

// ListMembers returns all member records.
func (s *Store) ListMembers() []*StoredMember {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredMember, 0, len(s.members))
	for _, m := range s.members {
		out = append(out, m)
	}
	return out
}

// InviteMembers sends invitations to existing members.
func (s *Store) InviteMembers(accountIDs []string) (unprocessed []map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for _, id := range accountIDs {
		m, ok := s.members[id]
		if !ok {
			unprocessed = append(unprocessed, map[string]any{
				"AccountId":        id,
				"ProcessingResult": "Member not found",
			})
			continue
		}
		m.MemberStatus = "Invited"
		m.InvitedAt = now
		m.UpdatedAt = now
	}
	return unprocessed
}

// AddInvitation records an inbound invitation for testing.
func (s *Store) AddInvitation(accountID, invitationID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invitations[accountID] = &StoredInvitation{
		AccountID:    accountID,
		InvitationID: invitationID,
		InvitedAt:    time.Now().UTC(),
		MemberStatus: "Invited",
	}
}

// ListInvitations returns all stored invitations.
func (s *Store) ListInvitations() []*StoredInvitation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredInvitation, 0, len(s.invitations))
	for _, i := range s.invitations {
		out = append(out, i)
	}
	return out
}

// DeclineInvitations removes invitations by account id.
func (s *Store) DeclineInvitations(accountIDs []string) (unprocessed []map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range accountIDs {
		if _, ok := s.invitations[id]; !ok {
			unprocessed = append(unprocessed, map[string]any{
				"AccountId":        id,
				"ProcessingResult": "Invitation not found",
			})
			continue
		}
		delete(s.invitations, id)
	}
	return unprocessed
}

// AcceptAdministrator stores the administrator account.
func (s *Store) AcceptAdministrator(adminID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if adminID == "" {
		return service.NewAWSError("InvalidInputException",
			"AdministratorId is required", http.StatusBadRequest)
	}
	s.administratorAccount = adminID
	s.administratorRelationship = "Enabled"
	return nil
}

// DisassociateAdministrator clears the administrator binding.
func (s *Store) DisassociateAdministrator() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.administratorAccount = ""
	s.administratorRelationship = ""
}

// GetAdministrator returns the active administrator.
func (s *Store) GetAdministrator() (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.administratorAccount, s.administratorRelationship
}

// ── Organization ─────────────────────────────────────────────────────────────

// EnableOrgAdmin sets the delegated admin account.
func (s *Store) EnableOrgAdmin(accountID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if accountID == "" {
		return service.NewAWSError("InvalidInputException",
			"AdminAccountId is required", http.StatusBadRequest)
	}
	s.delegatedAdministratorAcct = accountID
	return nil
}

// DisableOrgAdmin clears the delegated admin account.
func (s *Store) DisableOrgAdmin() *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.delegatedAdministratorAcct == "" {
		return service.NewAWSError("InvalidAccessException",
			"No delegated administrator is configured", http.StatusBadRequest)
	}
	s.delegatedAdministratorAcct = ""
	return nil
}

// ListOrgAdmins returns the configured delegated admins.
func (s *Store) ListOrgAdmins() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.delegatedAdministratorAcct == "" {
		return nil
	}
	return []string{s.delegatedAdministratorAcct}
}

// GetOrgConfig returns the current organization configuration.
func (s *Store) GetOrgConfig() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneMap(s.orgConfig)
}

// UpdateOrgConfig overwrites organization-config fields.
func (s *Store) UpdateOrgConfig(updates map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range updates {
		s.orgConfig[k] = v
	}
}

// ── Finding Aggregators (V1) ─────────────────────────────────────────────────

// CreateFindingAggregator stores a cross-region aggregator.
func (s *Store) CreateFindingAggregator(linkingMode string, regions []string) (*StoredFindingAggregator, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := generateID()
	agg := &StoredFindingAggregator{
		FindingAggregatorArn:     fmt.Sprintf("arn:aws:securityhub:%s:%s:finding-aggregator/%s", s.region, s.accountID, id),
		FindingAggregationRegion: s.region,
		RegionLinkingMode:        linkingMode,
		Regions:                  append([]string(nil), regions...),
	}
	s.findingAggregators[agg.FindingAggregatorArn] = agg
	return agg, nil
}

// UpdateFindingAggregator updates linking mode and regions.
func (s *Store) UpdateFindingAggregator(arn, linkingMode string, regions []string) (*StoredFindingAggregator, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	agg, ok := s.findingAggregators[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Finding aggregator not found: "+arn, http.StatusBadRequest)
	}
	if linkingMode != "" {
		agg.RegionLinkingMode = linkingMode
	}
	if regions != nil {
		agg.Regions = append([]string(nil), regions...)
	}
	return agg, nil
}

// GetFindingAggregator returns a single aggregator by ARN.
func (s *Store) GetFindingAggregator(arn string) (*StoredFindingAggregator, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	agg, ok := s.findingAggregators[arn]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Finding aggregator not found: "+arn, http.StatusBadRequest)
	}
	return agg, nil
}

// DeleteFindingAggregator removes an aggregator.
func (s *Store) DeleteFindingAggregator(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.findingAggregators[arn]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Finding aggregator not found: "+arn, http.StatusBadRequest)
	}
	delete(s.findingAggregators, arn)
	return nil
}

// ListFindingAggregators returns all aggregators.
func (s *Store) ListFindingAggregators() []*StoredFindingAggregator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredFindingAggregator, 0, len(s.findingAggregators))
	for _, a := range s.findingAggregators {
		out = append(out, a)
	}
	return out
}

// ── Aggregators V2 ───────────────────────────────────────────────────────────

// CreateAggregatorV2 stores a V2 aggregator.
func (s *Store) CreateAggregatorV2(linkingMode string, regions []string) (*StoredAggregatorV2, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := generateID()
	agg := &StoredAggregatorV2{
		AggregatorV2Arn:   fmt.Sprintf("arn:aws:securityhub:%s:%s:aggregator/v2/%s", s.region, s.accountID, id),
		AggregatorV2ID:    id,
		AggregationRegion: s.region,
		RegionLinkingMode: linkingMode,
		LinkedRegions:     append([]string(nil), regions...),
	}
	s.aggregatorsV2[id] = agg
	return agg, nil
}

// UpdateAggregatorV2 updates a V2 aggregator.
func (s *Store) UpdateAggregatorV2(id, linkingMode string, regions []string) (*StoredAggregatorV2, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	agg, ok := s.aggregatorsV2[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Aggregator V2 not found: "+id, http.StatusBadRequest)
	}
	if linkingMode != "" {
		agg.RegionLinkingMode = linkingMode
	}
	if regions != nil {
		agg.LinkedRegions = append([]string(nil), regions...)
	}
	return agg, nil
}

// GetAggregatorV2 returns a V2 aggregator.
func (s *Store) GetAggregatorV2(id string) (*StoredAggregatorV2, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	agg, ok := s.aggregatorsV2[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Aggregator V2 not found: "+id, http.StatusBadRequest)
	}
	return agg, nil
}

// DeleteAggregatorV2 removes a V2 aggregator.
func (s *Store) DeleteAggregatorV2(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.aggregatorsV2[id]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Aggregator V2 not found: "+id, http.StatusBadRequest)
	}
	delete(s.aggregatorsV2, id)
	return nil
}

// ListAggregatorsV2 returns all V2 aggregators.
func (s *Store) ListAggregatorsV2() []*StoredAggregatorV2 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredAggregatorV2, 0, len(s.aggregatorsV2))
	for _, a := range s.aggregatorsV2 {
		out = append(out, a)
	}
	return out
}

// ── Automation Rules (V1) ────────────────────────────────────────────────────

// CreateAutomationRule stores a V1 automation rule.
func (s *Store) CreateAutomationRule(name, description, status string, ruleOrder int, isTerminal bool, criteria map[string]any, actions []map[string]any) (*StoredAutomationRule, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name == "" {
		return nil, service.NewAWSError("InvalidInputException",
			"RuleName is required", http.StatusBadRequest)
	}
	id := generateID()
	if status == "" {
		status = "ENABLED"
	}
	rule := &StoredAutomationRule{
		RuleArn:     fmt.Sprintf("arn:aws:securityhub:%s:%s:automation-rule/%s", s.region, s.accountID, id),
		RuleID:      id,
		RuleName:    name,
		Description: description,
		RuleStatus:  status,
		RuleOrder:   ruleOrder,
		IsTerminal:  isTerminal,
		Criteria:    criteria,
		Actions:     actions,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		CreatedBy:   "cloudmock",
	}
	s.automationRules[rule.RuleArn] = rule
	return rule, nil
}

// GetAutomationRules returns rules by ARN.
func (s *Store) GetAutomationRules(arns []string) (rules []*StoredAutomationRule, unprocessed []map[string]any) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range arns {
		if r, ok := s.automationRules[a]; ok {
			rules = append(rules, r)
		} else {
			unprocessed = append(unprocessed, map[string]any{
				"RuleArn":      a,
				"ErrorCode":    "RuleNotFound",
				"ErrorMessage": "Rule not found: " + a,
			})
		}
	}
	return rules, unprocessed
}

// UpdateAutomationRules applies updates to rules.
func (s *Store) UpdateAutomationRules(updates []map[string]any) (processed []string, unprocessed []map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range updates {
		arn, _ := u["RuleArn"].(string)
		r, ok := s.automationRules[arn]
		if !ok {
			unprocessed = append(unprocessed, map[string]any{
				"RuleArn":      arn,
				"ErrorCode":    "RuleNotFound",
				"ErrorMessage": "Rule not found: " + arn,
			})
			continue
		}
		if name, ok := u["RuleName"].(string); ok && name != "" {
			r.RuleName = name
		}
		if desc, ok := u["Description"].(string); ok && desc != "" {
			r.Description = desc
		}
		if status, ok := u["RuleStatus"].(string); ok && status != "" {
			r.RuleStatus = status
		}
		if order, ok := u["RuleOrder"].(float64); ok {
			r.RuleOrder = int(order)
		}
		if term, ok := u["IsTerminal"].(bool); ok {
			r.IsTerminal = term
		}
		if c, ok := u["Criteria"].(map[string]any); ok {
			r.Criteria = c
		}
		if a, ok := u["Actions"].([]any); ok {
			r.Actions = toMapList(a)
		}
		r.UpdatedAt = time.Now().UTC()
		processed = append(processed, arn)
	}
	return processed, unprocessed
}

// DeleteAutomationRules removes rules by ARN.
func (s *Store) DeleteAutomationRules(arns []string) (processed []string, unprocessed []map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, a := range arns {
		if _, ok := s.automationRules[a]; !ok {
			unprocessed = append(unprocessed, map[string]any{
				"RuleArn":      a,
				"ErrorCode":    "RuleNotFound",
				"ErrorMessage": "Rule not found: " + a,
			})
			continue
		}
		delete(s.automationRules, a)
		processed = append(processed, a)
	}
	return processed, unprocessed
}

// ListAutomationRules returns all V1 rules.
func (s *Store) ListAutomationRules() []*StoredAutomationRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredAutomationRule, 0, len(s.automationRules))
	for _, r := range s.automationRules {
		out = append(out, r)
	}
	return out
}

// ── Automation Rules V2 ──────────────────────────────────────────────────────

// CreateAutomationRuleV2 stores a V2 rule.
func (s *Store) CreateAutomationRuleV2(name, description, status string, ruleOrder float64, criteria map[string]any, actions []map[string]any) (*StoredAutomationRuleV2, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name == "" {
		return nil, service.NewAWSError("InvalidInputException",
			"RuleName is required", http.StatusBadRequest)
	}
	id := generateID()
	if status == "" {
		status = "ENABLED"
	}
	rule := &StoredAutomationRuleV2{
		RuleArn:     fmt.Sprintf("arn:aws:securityhub:%s:%s:automation-rule/v2/%s", s.region, s.accountID, id),
		RuleID:      id,
		RuleName:    name,
		Description: description,
		RuleStatus:  status,
		RuleOrder:   ruleOrder,
		Criteria:    criteria,
		Actions:     actions,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	s.automationRulesV2[id] = rule
	return rule, nil
}

// GetAutomationRuleV2 returns a single V2 rule by id.
func (s *Store) GetAutomationRuleV2(id string) (*StoredAutomationRuleV2, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.automationRulesV2[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Automation rule V2 not found: "+id, http.StatusBadRequest)
	}
	return r, nil
}

// UpdateAutomationRuleV2 updates a V2 rule.
func (s *Store) UpdateAutomationRuleV2(id, name, description, status string, ruleOrder *float64, criteria map[string]any, actions []map[string]any) (*StoredAutomationRuleV2, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.automationRulesV2[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Automation rule V2 not found: "+id, http.StatusBadRequest)
	}
	if name != "" {
		r.RuleName = name
	}
	if description != "" {
		r.Description = description
	}
	if status != "" {
		r.RuleStatus = status
	}
	if ruleOrder != nil {
		r.RuleOrder = *ruleOrder
	}
	if criteria != nil {
		r.Criteria = criteria
	}
	if actions != nil {
		r.Actions = actions
	}
	r.UpdatedAt = time.Now().UTC()
	return r, nil
}

// DeleteAutomationRuleV2 removes a V2 rule.
func (s *Store) DeleteAutomationRuleV2(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.automationRulesV2[id]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Automation rule V2 not found: "+id, http.StatusBadRequest)
	}
	delete(s.automationRulesV2, id)
	return nil
}

// ListAutomationRulesV2 returns all V2 rules.
func (s *Store) ListAutomationRulesV2() []*StoredAutomationRuleV2 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredAutomationRuleV2, 0, len(s.automationRulesV2))
	for _, r := range s.automationRulesV2 {
		out = append(out, r)
	}
	return out
}

// ── Configuration Policies ───────────────────────────────────────────────────

// CreateConfigurationPolicy stores a new policy.
func (s *Store) CreateConfigurationPolicy(name, description string, configurationPolicy map[string]any) (*StoredConfigurationPolicy, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name == "" {
		return nil, service.NewAWSError("InvalidInputException",
			"Name is required", http.StatusBadRequest)
	}
	id := generateID()
	now := time.Now().UTC()
	p := &StoredConfigurationPolicy{
		Arn:                 fmt.Sprintf("arn:aws:securityhub:%s:%s:configuration-policy/%s", s.region, s.accountID, id),
		ID:                  id,
		Name:                name,
		Description:         description,
		CreatedAt:           now,
		UpdatedAt:           now,
		ConfigurationPolicy: configurationPolicy,
	}
	s.configurationPolicies[id] = p
	return p, nil
}

// UpdateConfigurationPolicy mutates fields on an existing policy.
func (s *Store) UpdateConfigurationPolicy(id, name, description string, configurationPolicy map[string]any) (*StoredConfigurationPolicy, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.lookupPolicyLocked(id)
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Configuration policy not found: "+id, http.StatusBadRequest)
	}
	if name != "" {
		p.Name = name
	}
	if description != "" {
		p.Description = description
	}
	if configurationPolicy != nil {
		p.ConfigurationPolicy = configurationPolicy
	}
	p.UpdatedAt = time.Now().UTC()
	return p, nil
}

// DeleteConfigurationPolicy removes a policy and its associations.
func (s *Store) DeleteConfigurationPolicy(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.lookupPolicyLocked(id)
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Configuration policy not found: "+id, http.StatusBadRequest)
	}
	delete(s.configurationPolicies, p.ID)
	for k, a := range s.configurationAssociations {
		if a.ConfigurationPolicyID == p.ID {
			delete(s.configurationAssociations, k)
		}
	}
	return nil
}

// GetConfigurationPolicy returns a policy by id or arn.
func (s *Store) GetConfigurationPolicy(id string) (*StoredConfigurationPolicy, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.lookupPolicyLocked(id)
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Configuration policy not found: "+id, http.StatusBadRequest)
	}
	return p, nil
}

// ListConfigurationPolicies returns all stored policies.
func (s *Store) ListConfigurationPolicies() []*StoredConfigurationPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredConfigurationPolicy, 0, len(s.configurationPolicies))
	for _, p := range s.configurationPolicies {
		out = append(out, p)
	}
	return out
}

// lookupPolicyLocked finds a policy by ID or ARN. Caller must hold lock.
func (s *Store) lookupPolicyLocked(idOrArn string) (*StoredConfigurationPolicy, bool) {
	if p, ok := s.configurationPolicies[idOrArn]; ok {
		return p, true
	}
	for _, p := range s.configurationPolicies {
		if p.Arn == idOrArn {
			return p, true
		}
	}
	return nil, false
}

// StartConfigurationPolicyAssociation binds a policy to a target.
func (s *Store) StartConfigurationPolicyAssociation(policyID, targetID, targetType string) (*StoredConfigurationPolicyAssociation, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if targetID == "" {
		return nil, service.NewAWSError("InvalidInputException",
			"Target identifier is required", http.StatusBadRequest)
	}
	p, ok := s.lookupPolicyLocked(policyID)
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Configuration policy not found: "+policyID, http.StatusBadRequest)
	}
	assoc := &StoredConfigurationPolicyAssociation{
		TargetID:                 targetID,
		TargetType:               targetType,
		ConfigurationPolicyID:    p.ID,
		AssociationType:          "APPLIED",
		AssociationStatus:        "SUCCESS",
		AssociationStatusMessage: "Association started",
		UpdatedAt:                time.Now().UTC(),
	}
	s.configurationAssociations[targetID] = assoc
	return assoc, nil
}

// StartConfigurationPolicyDisassociation removes a policy/target binding.
func (s *Store) StartConfigurationPolicyDisassociation(policyID, targetID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.configurationAssociations[targetID]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Association not found for target: "+targetID, http.StatusBadRequest)
	}
	if policyID != "" && a.ConfigurationPolicyID != policyID {
		// Soft-tolerate: still remove if it matches the target.
	}
	delete(s.configurationAssociations, targetID)
	return nil
}

// GetConfigurationPolicyAssociation returns one association.
func (s *Store) GetConfigurationPolicyAssociation(targetID string) (*StoredConfigurationPolicyAssociation, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.configurationAssociations[targetID]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Association not found for target: "+targetID, http.StatusBadRequest)
	}
	return a, nil
}

// BatchGetConfigurationPolicyAssociations returns multiple associations.
func (s *Store) BatchGetConfigurationPolicyAssociations(targetIDs []string) (found []*StoredConfigurationPolicyAssociation, unprocessed []map[string]any) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, id := range targetIDs {
		if a, ok := s.configurationAssociations[id]; ok {
			found = append(found, a)
		} else {
			unprocessed = append(unprocessed, map[string]any{
				"TargetId":     id,
				"ErrorCode":    "AssociationNotFound",
				"ErrorMessage": "No association found for target " + id,
			})
		}
	}
	return found, unprocessed
}

// ListConfigurationPolicyAssociations returns all associations.
func (s *Store) ListConfigurationPolicyAssociations() []*StoredConfigurationPolicyAssociation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredConfigurationPolicyAssociation, 0, len(s.configurationAssociations))
	for _, a := range s.configurationAssociations {
		out = append(out, a)
	}
	return out
}

// ── Connectors V2 ────────────────────────────────────────────────────────────

// CreateConnectorV2 stores a connector.
func (s *Store) CreateConnectorV2(name, description, providerName, kmsKeyArn string, providerSummary map[string]any, tags map[string]string) (*StoredConnectorV2, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name == "" {
		return nil, service.NewAWSError("InvalidInputException",
			"Name is required", http.StatusBadRequest)
	}
	id := generateID()
	now := time.Now().UTC()
	c := &StoredConnectorV2{
		ConnectorArn:    fmt.Sprintf("arn:aws:securityhub:%s:%s:connector/v2/%s", s.region, s.accountID, id),
		ConnectorID:     id,
		Name:            name,
		Description:     description,
		ProviderName:    providerName,
		HealthStatus:    "HEALTHY",
		ClientSecret:    "secret-" + id,
		KmsKeyArn:       kmsKeyArn,
		CreatedAt:       now,
		LastUpdatedAt:   now,
		ProviderSummary: providerSummary,
		Tags:            copyStringMap(tags),
	}
	s.connectorsV2[id] = c
	return c, nil
}

// GetConnectorV2 returns a connector by id.
func (s *Store) GetConnectorV2(id string) (*StoredConnectorV2, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.connectorsV2[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Connector V2 not found: "+id, http.StatusBadRequest)
	}
	return c, nil
}

// UpdateConnectorV2 modifies a connector.
func (s *Store) UpdateConnectorV2(id, description string, providerSummary map[string]any) (*StoredConnectorV2, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.connectorsV2[id]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Connector V2 not found: "+id, http.StatusBadRequest)
	}
	if description != "" {
		c.Description = description
	}
	if providerSummary != nil {
		c.ProviderSummary = providerSummary
	}
	c.LastUpdatedAt = time.Now().UTC()
	return c, nil
}

// DeleteConnectorV2 removes a connector.
func (s *Store) DeleteConnectorV2(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.connectorsV2[id]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Connector V2 not found: "+id, http.StatusBadRequest)
	}
	delete(s.connectorsV2, id)
	return nil
}

// ListConnectorsV2 returns all connectors.
func (s *Store) ListConnectorsV2() []*StoredConnectorV2 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredConnectorV2, 0, len(s.connectorsV2))
	for _, c := range s.connectorsV2 {
		out = append(out, c)
	}
	return out
}

// ── Tags ─────────────────────────────────────────────────────────────────────

// TagResource attaches tags to an ARN.
func (s *Store) TagResource(arn string, tags map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.resourceTags[arn] == nil {
		s.resourceTags[arn] = make(map[string]string)
	}
	for k, v := range tags {
		s.resourceTags[arn][k] = v
	}
}

// UntagResource removes tag keys from an ARN.
func (s *Store) UntagResource(arn string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.resourceTags[arn]; ok {
		for _, k := range keys {
			delete(m, k)
		}
	}
}

// ListTags returns tags for an ARN.
func (s *Store) ListTags(arn string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string)
	if m, ok := s.resourceTags[arn]; ok {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// AccountID returns the configured account id.
func (s *Store) AccountID() string { return s.accountID }

// Region returns the configured region.
func (s *Store) Region() string { return s.region }

// ── Helpers ──────────────────────────────────────────────────────────────────

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func copyStringMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func cloneMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func toMapList(arr []any) []map[string]any {
	out := make([]map[string]any, 0, len(arr))
	for _, v := range arr {
		if m, ok := v.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func lastSegment(arn string) string {
	for i := len(arn) - 1; i >= 0; i-- {
		if arn[i] == '/' || arn[i] == ':' {
			return arn[i+1:]
		}
	}
	return arn
}

// matchesFilters does a best-effort match of stored finding data against
// Security Hub filter criteria. Filters are loose key/value comparisons —
// for parity with the real API we accept any filter shape and treat unknown
// filters as match-all.
func matchesFilters(finding map[string]any, filters map[string]any) bool {
	if len(filters) == 0 {
		return true
	}
	for key, criterion := range filters {
		// Criterion is typically a list of {"Value": "...", "Comparison": "EQUALS"} maps.
		fv, exists := finding[key]
		critList, ok := criterion.([]any)
		if !ok {
			continue
		}
		if len(critList) == 0 {
			continue
		}
		matched := false
		for _, c := range critList {
			cm, ok := c.(map[string]any)
			if !ok {
				continue
			}
			val, _ := cm["Value"].(string)
			cmp, _ := cm["Comparison"].(string)
			if cmp == "" {
				cmp = "EQUALS"
			}
			if !exists {
				continue
			}
			fvStr, _ := fv.(string)
			switch cmp {
			case "EQUALS":
				if fvStr == val {
					matched = true
				}
			case "NOT_EQUALS":
				if fvStr != val {
					matched = true
				}
			case "PREFIX":
				if len(val) <= len(fvStr) && fvStr[:len(val)] == val {
					matched = true
				}
			case "CONTAINS":
				if containsSubstr(fvStr, val) {
					matched = true
				}
			default:
				if fvStr == val {
					matched = true
				}
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func containsSubstr(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
