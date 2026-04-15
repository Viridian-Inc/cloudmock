package servicecatalog

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Stored types ────────────────────────────────────────────────────────────

// StoredPortfolio is a persisted portfolio.
type StoredPortfolio struct {
	Id           string
	Arn          string
	DisplayName  string
	Description  string
	ProviderName string
	CreatedTime  time.Time
	Tags         map[string]string
}

// StoredPortfolioShare is a persisted portfolio share.
type StoredPortfolioShare struct {
	PortfolioId     string
	AccountId       string
	Type            string // ACCOUNT, ORGANIZATION, ORGANIZATIONAL_UNIT, ORGANIZATION_MEMBER_ACCOUNT
	Accepted        bool
	SharePrincipals bool
	ShareTagOptions bool
	Token           string
	Status          string // NOT_STARTED, IN_PROGRESS, COMPLETED, COMPLETED_WITH_ERRORS, ERROR
}

// StoredProduct is a persisted product (top-level catalog entry).
type StoredProduct struct {
	Id                 string
	Arn                string
	Name               string
	Owner              string
	Distributor        string
	Description        string
	ShortDescription   string
	Type               string // CLOUD_FORMATION_TEMPLATE, MARKETPLACE, TERRAFORM_OPEN_SOURCE
	CreatedTime        time.Time
	Status             string // CREATED, AVAILABLE, FAILED
	Tags               map[string]string
	SupportDescription string
	SupportEmail       string
	SupportUrl         string
}

// StoredProvisioningArtifact represents a provisioning artifact (a version of a product template).
type StoredProvisioningArtifact struct {
	Id          string
	ProductId   string
	Name        string
	Description string
	Type        string
	CreatedTime time.Time
	Active      bool
	Guidance    string // DEFAULT, DEPRECATED
	Info        map[string]string
}

// StoredConstraint is a persisted launch/notification/etc. constraint.
type StoredConstraint struct {
	Id          string
	Type        string // LAUNCH, NOTIFICATION, TEMPLATE, STACKSET, RESOURCE_UPDATE
	Description string
	Parameters  string
	PortfolioId string
	ProductId   string
	Status      string // AVAILABLE, CREATING, FAILED
	Owner       string
}

// StoredPrincipal is an IAM/IAM_PATTERN principal granted access to a portfolio.
type StoredPrincipal struct {
	PortfolioId  string
	PrincipalARN string
	PrincipalType string // IAM, IAM_PATTERN
}

// StoredTagOption is a persisted tag option.
type StoredTagOption struct {
	Id     string
	Key    string
	Value  string
	Active bool
	Owner  string
}

// StoredProvisionedProduct is a persisted provisioned product instance.
type StoredProvisionedProduct struct {
	Id                       string
	Name                     string
	Type                     string // CFN_STACK, CFN_STACKSET, TERRAFORM_OPEN_SOURCE
	Arn                      string
	Status                   string // AVAILABLE, UNDER_CHANGE, TAINTED, ERROR, PLAN_IN_PROGRESS
	StatusMessage            string
	CreatedTime              time.Time
	LastRecordId             string
	ProductId                string
	ProductName              string
	ProvisioningArtifactId   string
	ProvisioningArtifactName string
	UserArn                  string
	UserArnSession           string
	Tags                     map[string]string
	LaunchRoleArn            string
	IdempotencyToken         string
	Outputs                  []map[string]any
}

// StoredRecord is a persisted record (provision/update/terminate operation).
type StoredRecord struct {
	Id                     string
	ProvisionedProductName string
	ProvisionedProductType string
	RecordType             string // PROVISION_PRODUCT, UPDATE_PROVISIONED_PRODUCT, TERMINATE_PROVISIONED_PRODUCT
	ProvisionedProductId   string
	Status                 string // CREATED, IN_PROGRESS, IN_PROGRESS_IN_ERROR, SUCCEEDED, FAILED
	CreatedTime            time.Time
	UpdatedTime            time.Time
	ProductId              string
	ProvisioningArtifactId string
	PathId                 string
	RecordErrors           []map[string]any
	RecordTags             map[string]string
}

// StoredServiceAction is a persisted Service Catalog service action.
type StoredServiceAction struct {
	Id             string
	Name           string
	DefinitionType string // SSM_AUTOMATION
	Definition     map[string]string
	Description    string
}

// StoredPlan is a persisted provisioned product plan.
type StoredPlan struct {
	Id                     string
	Name                   string
	Type                   string // CLOUDFORMATION, TERRAFORMCLOUD
	ProductId              string
	PathId                 string
	ProvisioningArtifactId string
	NotificationArns       []string
	ProvisionedProductName string
	ProvisioningParameters []map[string]any
	Tags                   map[string]string
	StatusMessage          string
	Status                 string // CREATE_IN_PROGRESS, CREATE_SUCCESS, CREATE_FAILED, EXECUTE_IN_PROGRESS, EXECUTE_SUCCESS, EXECUTE_FAILED
	UpdatedTime            time.Time
	CreatedTime            time.Time
}

// StoredCopyOperation tracks a CopyProduct in-flight token.
type StoredCopyOperation struct {
	Token           string
	SourceProductArn string
	TargetProductId string
	Status          string // SUCCEEDED, IN_PROGRESS, FAILED
	StatusDetail    string
}

// ── Store ───────────────────────────────────────────────────────────────────

// Store is the in-memory data store for AWS Service Catalog.
type Store struct {
	mu        sync.RWMutex
	accountID string
	region    string

	portfolios      map[string]*StoredPortfolio                          // portfolioId -> portfolio
	portfolioShares map[string]map[string]*StoredPortfolioShare          // portfolioId -> shareKey -> share
	products        map[string]*StoredProduct                            // productId -> product
	artifacts       map[string]map[string]*StoredProvisioningArtifact    // productId -> artifactId -> artifact
	constraints     map[string]*StoredConstraint                         // constraintId -> constraint
	principals      map[string]map[string]*StoredPrincipal               // portfolioId -> principalARN -> principal
	tagOptions      map[string]*StoredTagOption                          // tagOptionId -> tag option
	provisioned     map[string]*StoredProvisionedProduct                 // provisionedProductId -> pp
	records         map[string]*StoredRecord                             // recordId -> record
	serviceActions  map[string]*StoredServiceAction                      // serviceActionId -> service action
	plans           map[string]*StoredPlan                               // planId -> plan
	copies          map[string]*StoredCopyOperation                      // copyToken -> copy op

	portfolioProducts map[string]map[string]bool   // portfolioId -> productId set
	productPortfolios map[string]map[string]bool   // productId -> portfolioId set

	tagOptionResources map[string]map[string]bool // tagOptionId -> resourceId set
	resourceTagOptions map[string]map[string]bool // resourceId -> tagOptionId set

	saArtifacts map[string]map[string]bool // serviceActionId -> "productId|artifactId" set
	artifactSAs map[string]map[string]bool // "productId|artifactId" -> serviceActionId set

	resourceBudgets map[string]map[string]bool // resourceId -> budgetName set
	budgetResources map[string]map[string]bool // budgetName -> resourceId set

	awsOrgAccess string // ENABLED, DISABLED, ENABLE_IN_PROGRESS, UNDER_CHANGE
}

// NewStore creates an empty Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID:          accountID,
		region:             region,
		portfolios:         make(map[string]*StoredPortfolio),
		portfolioShares:    make(map[string]map[string]*StoredPortfolioShare),
		products:           make(map[string]*StoredProduct),
		artifacts:          make(map[string]map[string]*StoredProvisioningArtifact),
		constraints:        make(map[string]*StoredConstraint),
		principals:         make(map[string]map[string]*StoredPrincipal),
		tagOptions:         make(map[string]*StoredTagOption),
		provisioned:        make(map[string]*StoredProvisionedProduct),
		records:            make(map[string]*StoredRecord),
		serviceActions:     make(map[string]*StoredServiceAction),
		plans:              make(map[string]*StoredPlan),
		copies:             make(map[string]*StoredCopyOperation),
		portfolioProducts:  make(map[string]map[string]bool),
		productPortfolios:  make(map[string]map[string]bool),
		tagOptionResources: make(map[string]map[string]bool),
		resourceTagOptions: make(map[string]map[string]bool),
		saArtifacts:        make(map[string]map[string]bool),
		artifactSAs:        make(map[string]map[string]bool),
		resourceBudgets:    make(map[string]map[string]bool),
		budgetResources:    make(map[string]map[string]bool),
		awsOrgAccess:       "DISABLED",
	}
}

// Reset clears all in-memory state.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.portfolios = make(map[string]*StoredPortfolio)
	s.portfolioShares = make(map[string]map[string]*StoredPortfolioShare)
	s.products = make(map[string]*StoredProduct)
	s.artifacts = make(map[string]map[string]*StoredProvisioningArtifact)
	s.constraints = make(map[string]*StoredConstraint)
	s.principals = make(map[string]map[string]*StoredPrincipal)
	s.tagOptions = make(map[string]*StoredTagOption)
	s.provisioned = make(map[string]*StoredProvisionedProduct)
	s.records = make(map[string]*StoredRecord)
	s.serviceActions = make(map[string]*StoredServiceAction)
	s.plans = make(map[string]*StoredPlan)
	s.copies = make(map[string]*StoredCopyOperation)
	s.portfolioProducts = make(map[string]map[string]bool)
	s.productPortfolios = make(map[string]map[string]bool)
	s.tagOptionResources = make(map[string]map[string]bool)
	s.resourceTagOptions = make(map[string]map[string]bool)
	s.saArtifacts = make(map[string]map[string]bool)
	s.artifactSAs = make(map[string]map[string]bool)
	s.resourceBudgets = make(map[string]map[string]bool)
	s.budgetResources = make(map[string]map[string]bool)
	s.awsOrgAccess = "DISABLED"
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func newID(prefix string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

func newToken() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func errNotFound(resource, id string) *service.AWSError {
	return service.NewAWSError("ResourceNotFoundException",
		fmt.Sprintf("%s %s not found", resource, id), http.StatusBadRequest)
}

func errInvalidParam(message string) *service.AWSError {
	return service.NewAWSError("InvalidParametersException", message, http.StatusBadRequest)
}

func errDuplicate(resource, name string) *service.AWSError {
	return service.NewAWSError("DuplicateResourceException",
		fmt.Sprintf("%s %s already exists", resource, name), http.StatusBadRequest)
}

func copyTagMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// ── Portfolios ──────────────────────────────────────────────────────────────

// CreatePortfolio inserts a new portfolio.
func (s *Store) CreatePortfolio(displayName, description, providerName string, tags map[string]string) (*StoredPortfolio, *service.AWSError) {
	if displayName == "" {
		return nil, errInvalidParam("DisplayName is required")
	}
	if providerName == "" {
		return nil, errInvalidParam("ProviderName is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newID("port")
	p := &StoredPortfolio{
		Id:           id,
		Arn:          fmt.Sprintf("arn:aws:catalog:%s:%s:portfolio/%s", s.region, s.accountID, id),
		DisplayName:  displayName,
		Description:  description,
		ProviderName: providerName,
		CreatedTime:  time.Now().UTC(),
		Tags:         copyTagMap(tags),
	}
	s.portfolios[id] = p
	return p, nil
}

// GetPortfolio returns a portfolio by id.
func (s *Store) GetPortfolio(id string) (*StoredPortfolio, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.portfolios[id]
	if !ok {
		return nil, errNotFound("Portfolio", id)
	}
	return p, nil
}

// DeletePortfolio removes a portfolio and all of its associations.
func (s *Store) DeletePortfolio(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.portfolios[id]; !ok {
		return errNotFound("Portfolio", id)
	}
	delete(s.portfolios, id)
	delete(s.portfolioShares, id)
	delete(s.principals, id)
	if products, ok := s.portfolioProducts[id]; ok {
		for productID := range products {
			if pp, ok := s.productPortfolios[productID]; ok {
				delete(pp, id)
			}
		}
		delete(s.portfolioProducts, id)
	}
	// drop any tag-option associations
	if tos, ok := s.resourceTagOptions[id]; ok {
		for to := range tos {
			if rs, ok := s.tagOptionResources[to]; ok {
				delete(rs, id)
			}
		}
		delete(s.resourceTagOptions, id)
	}
	if budgets, ok := s.resourceBudgets[id]; ok {
		for b := range budgets {
			if br, ok := s.budgetResources[b]; ok {
				delete(br, id)
			}
		}
		delete(s.resourceBudgets, id)
	}
	return nil
}

// ListPortfolios returns all portfolios.
func (s *Store) ListPortfolios() []*StoredPortfolio {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredPortfolio, 0, len(s.portfolios))
	for _, p := range s.portfolios {
		out = append(out, p)
	}
	return out
}

// UpdatePortfolio updates fields on a portfolio.
func (s *Store) UpdatePortfolio(id string, displayName, description, providerName *string, addTags map[string]string, removeTagKeys []string) (*StoredPortfolio, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.portfolios[id]
	if !ok {
		return nil, errNotFound("Portfolio", id)
	}
	if displayName != nil && *displayName != "" {
		p.DisplayName = *displayName
	}
	if description != nil {
		p.Description = *description
	}
	if providerName != nil && *providerName != "" {
		p.ProviderName = *providerName
	}
	for k, v := range addTags {
		p.Tags[k] = v
	}
	for _, k := range removeTagKeys {
		delete(p.Tags, k)
	}
	return p, nil
}

// ── Portfolio shares ────────────────────────────────────────────────────────

func shareKey(shareType, accountID string) string {
	return shareType + "|" + accountID
}

// CreatePortfolioShare records a portfolio share.
func (s *Store) CreatePortfolioShare(portfolioID, accountID, shareType string, sharePrincipals, shareTagOptions bool) (string, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.portfolios[portfolioID]; !ok {
		return "", errNotFound("Portfolio", portfolioID)
	}
	if shareType == "" {
		shareType = "ACCOUNT"
	}
	if accountID == "" {
		return "", errInvalidParam("AccountId or OrganizationNode is required")
	}
	if s.portfolioShares[portfolioID] == nil {
		s.portfolioShares[portfolioID] = make(map[string]*StoredPortfolioShare)
	}
	key := shareKey(shareType, accountID)
	token := newToken()
	s.portfolioShares[portfolioID][key] = &StoredPortfolioShare{
		PortfolioId:     portfolioID,
		AccountId:       accountID,
		Type:            shareType,
		Accepted:        false,
		SharePrincipals: sharePrincipals,
		ShareTagOptions: shareTagOptions,
		Token:           token,
		Status:          "COMPLETED",
	}
	return token, nil
}

// DeletePortfolioShare removes a share.
func (s *Store) DeletePortfolioShare(portfolioID, accountID, shareType string) (string, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.portfolios[portfolioID]; !ok {
		return "", errNotFound("Portfolio", portfolioID)
	}
	if shareType == "" {
		shareType = "ACCOUNT"
	}
	shares, ok := s.portfolioShares[portfolioID]
	if !ok {
		return "", errNotFound("PortfolioShare", accountID)
	}
	key := shareKey(shareType, accountID)
	share, ok := shares[key]
	if !ok {
		return "", errNotFound("PortfolioShare", accountID)
	}
	delete(shares, key)
	return share.Token, nil
}

// ListPortfolioShares returns shares for a portfolio.
func (s *Store) ListPortfolioShares(portfolioID, shareType string) []*StoredPortfolioShare {
	s.mu.RLock()
	defer s.mu.RUnlock()
	shares := s.portfolioShares[portfolioID]
	out := make([]*StoredPortfolioShare, 0, len(shares))
	for _, sh := range shares {
		if shareType == "" || sh.Type == shareType {
			out = append(out, sh)
		}
	}
	return out
}

// AcceptPortfolioShare marks a share as accepted.
func (s *Store) AcceptPortfolioShare(portfolioID, shareType string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	shares, ok := s.portfolioShares[portfolioID]
	if !ok {
		return errNotFound("PortfolioShare", portfolioID)
	}
	matched := false
	for _, sh := range shares {
		if shareType == "" || sh.Type == shareType {
			sh.Accepted = true
			matched = true
		}
	}
	if !matched {
		return errNotFound("PortfolioShare", portfolioID)
	}
	return nil
}

// RejectPortfolioShare removes shares of the given type.
func (s *Store) RejectPortfolioShare(portfolioID, shareType string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	shares, ok := s.portfolioShares[portfolioID]
	if !ok {
		return errNotFound("PortfolioShare", portfolioID)
	}
	matched := false
	for k, sh := range shares {
		if shareType == "" || sh.Type == shareType {
			delete(shares, k)
			matched = true
		}
	}
	if !matched {
		return errNotFound("PortfolioShare", portfolioID)
	}
	return nil
}

// UpdatePortfolioShare updates a share's flags.
func (s *Store) UpdatePortfolioShare(portfolioID, accountID, shareType string, sharePrincipals, shareTagOptions *bool) (string, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.portfolios[portfolioID]; !ok {
		return "", errNotFound("Portfolio", portfolioID)
	}
	if shareType == "" {
		shareType = "ACCOUNT"
	}
	shares := s.portfolioShares[portfolioID]
	if shares == nil {
		return "", errNotFound("PortfolioShare", accountID)
	}
	share, ok := shares[shareKey(shareType, accountID)]
	if !ok {
		return "", errNotFound("PortfolioShare", accountID)
	}
	if sharePrincipals != nil {
		share.SharePrincipals = *sharePrincipals
	}
	if shareTagOptions != nil {
		share.ShareTagOptions = *shareTagOptions
	}
	return share.Token, nil
}

// ── Products ────────────────────────────────────────────────────────────────

// CreateProduct inserts a new product (and its initial provisioning artifact, if provided).
func (s *Store) CreateProduct(name, owner, description, distributor, productType, supportDescription, supportEmail, supportUrl string, tags map[string]string, artifactName, artifactDescription, artifactType string, info map[string]string) (*StoredProduct, *StoredProvisioningArtifact, *service.AWSError) {
	if name == "" {
		return nil, nil, errInvalidParam("Name is required")
	}
	if owner == "" {
		return nil, nil, errInvalidParam("Owner is required")
	}
	if productType == "" {
		productType = "CLOUD_FORMATION_TEMPLATE"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newID("prod")
	p := &StoredProduct{
		Id:                 id,
		Arn:                fmt.Sprintf("arn:aws:catalog:%s:%s:product/%s", s.region, s.accountID, id),
		Name:               name,
		Owner:              owner,
		Description:        description,
		ShortDescription:   description,
		Distributor:        distributor,
		Type:               productType,
		CreatedTime:        time.Now().UTC(),
		Status:             "AVAILABLE",
		Tags:               copyTagMap(tags),
		SupportDescription: supportDescription,
		SupportEmail:       supportEmail,
		SupportUrl:         supportUrl,
	}
	s.products[id] = p

	var pa *StoredProvisioningArtifact
	if artifactName != "" || info != nil {
		paID := newID("pa")
		pa = &StoredProvisioningArtifact{
			Id:          paID,
			ProductId:   id,
			Name:        artifactName,
			Description: artifactDescription,
			Type:        artifactType,
			CreatedTime: time.Now().UTC(),
			Active:      true,
			Guidance:    "DEFAULT",
			Info:        info,
		}
		if pa.Type == "" {
			pa.Type = "CLOUD_FORMATION_TEMPLATE"
		}
		s.artifacts[id] = map[string]*StoredProvisioningArtifact{paID: pa}
	}
	return p, pa, nil
}

// GetProduct returns a product by id.
func (s *Store) GetProduct(id string) (*StoredProduct, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[id]
	if !ok {
		return nil, errNotFound("Product", id)
	}
	return p, nil
}

// GetProductByName returns a product by display name.
func (s *Store) GetProductByName(name string) (*StoredProduct, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.products {
		if p.Name == name {
			return p, nil
		}
	}
	return nil, errNotFound("Product", name)
}

// DeleteProduct removes a product and all of its artifacts.
func (s *Store) DeleteProduct(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.products[id]; !ok {
		return errNotFound("Product", id)
	}
	delete(s.products, id)
	delete(s.artifacts, id)
	if portfolios, ok := s.productPortfolios[id]; ok {
		for portID := range portfolios {
			if pp, ok := s.portfolioProducts[portID]; ok {
				delete(pp, id)
			}
		}
		delete(s.productPortfolios, id)
	}
	if tos, ok := s.resourceTagOptions[id]; ok {
		for to := range tos {
			if rs, ok := s.tagOptionResources[to]; ok {
				delete(rs, id)
			}
		}
		delete(s.resourceTagOptions, id)
	}
	if budgets, ok := s.resourceBudgets[id]; ok {
		for b := range budgets {
			if br, ok := s.budgetResources[b]; ok {
				delete(br, id)
			}
		}
		delete(s.resourceBudgets, id)
	}
	return nil
}

// ListProducts returns all products.
func (s *Store) ListProducts() []*StoredProduct {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredProduct, 0, len(s.products))
	for _, p := range s.products {
		out = append(out, p)
	}
	return out
}

// UpdateProduct updates fields on a product.
func (s *Store) UpdateProduct(id string, name, owner, description, distributor, supportDescription, supportEmail, supportUrl *string, addTags map[string]string, removeTagKeys []string) (*StoredProduct, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.products[id]
	if !ok {
		return nil, errNotFound("Product", id)
	}
	if name != nil && *name != "" {
		p.Name = *name
	}
	if owner != nil && *owner != "" {
		p.Owner = *owner
	}
	if description != nil {
		p.Description = *description
		p.ShortDescription = *description
	}
	if distributor != nil {
		p.Distributor = *distributor
	}
	if supportDescription != nil {
		p.SupportDescription = *supportDescription
	}
	if supportEmail != nil {
		p.SupportEmail = *supportEmail
	}
	if supportUrl != nil {
		p.SupportUrl = *supportUrl
	}
	for k, v := range addTags {
		p.Tags[k] = v
	}
	for _, k := range removeTagKeys {
		delete(p.Tags, k)
	}
	return p, nil
}

// ── Provisioning artifacts ──────────────────────────────────────────────────

// CreateProvisioningArtifact inserts a new artifact for a product.
func (s *Store) CreateProvisioningArtifact(productID, name, description, paType string, info map[string]string) (*StoredProvisioningArtifact, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.products[productID]; !ok {
		return nil, errNotFound("Product", productID)
	}
	if paType == "" {
		paType = "CLOUD_FORMATION_TEMPLATE"
	}
	if s.artifacts[productID] == nil {
		s.artifacts[productID] = make(map[string]*StoredProvisioningArtifact)
	}
	id := newID("pa")
	pa := &StoredProvisioningArtifact{
		Id:          id,
		ProductId:   productID,
		Name:        name,
		Description: description,
		Type:        paType,
		CreatedTime: time.Now().UTC(),
		Active:      true,
		Guidance:    "DEFAULT",
		Info:        info,
	}
	s.artifacts[productID][id] = pa
	return pa, nil
}

// GetProvisioningArtifact returns an artifact by id.
func (s *Store) GetProvisioningArtifact(productID, artifactID string) (*StoredProvisioningArtifact, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if productID != "" {
		if pas, ok := s.artifacts[productID]; ok {
			if pa, ok := pas[artifactID]; ok {
				return pa, nil
			}
		}
		return nil, errNotFound("ProvisioningArtifact", artifactID)
	}
	for _, pas := range s.artifacts {
		if pa, ok := pas[artifactID]; ok {
			return pa, nil
		}
	}
	return nil, errNotFound("ProvisioningArtifact", artifactID)
}

// DeleteProvisioningArtifact removes an artifact.
func (s *Store) DeleteProvisioningArtifact(productID, artifactID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	pas, ok := s.artifacts[productID]
	if !ok {
		return errNotFound("ProvisioningArtifact", artifactID)
	}
	if _, ok := pas[artifactID]; !ok {
		return errNotFound("ProvisioningArtifact", artifactID)
	}
	delete(pas, artifactID)
	// Drop associated service action links
	key := productID + "|" + artifactID
	if sas, ok := s.artifactSAs[key]; ok {
		for sa := range sas {
			if as, ok := s.saArtifacts[sa]; ok {
				delete(as, key)
			}
		}
		delete(s.artifactSAs, key)
	}
	return nil
}

// ListProvisioningArtifacts returns all artifacts for a product.
func (s *Store) ListProvisioningArtifacts(productID string) ([]*StoredProvisioningArtifact, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.products[productID]; !ok {
		return nil, errNotFound("Product", productID)
	}
	pas := s.artifacts[productID]
	out := make([]*StoredProvisioningArtifact, 0, len(pas))
	for _, pa := range pas {
		out = append(out, pa)
	}
	return out, nil
}

// UpdateProvisioningArtifact updates fields on an artifact.
func (s *Store) UpdateProvisioningArtifact(productID, artifactID string, name, description, guidance *string, active *bool) (*StoredProvisioningArtifact, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pas, ok := s.artifacts[productID]
	if !ok {
		return nil, errNotFound("ProvisioningArtifact", artifactID)
	}
	pa, ok := pas[artifactID]
	if !ok {
		return nil, errNotFound("ProvisioningArtifact", artifactID)
	}
	if name != nil && *name != "" {
		pa.Name = *name
	}
	if description != nil {
		pa.Description = *description
	}
	if guidance != nil && *guidance != "" {
		pa.Guidance = *guidance
	}
	if active != nil {
		pa.Active = *active
	}
	return pa, nil
}

// ── Constraints ─────────────────────────────────────────────────────────────

// CreateConstraint inserts a new constraint.
func (s *Store) CreateConstraint(portfolioID, productID, constraintType, parameters, description string) (*StoredConstraint, *service.AWSError) {
	if portfolioID == "" {
		return nil, errInvalidParam("PortfolioId is required")
	}
	if productID == "" {
		return nil, errInvalidParam("ProductId is required")
	}
	if constraintType == "" {
		return nil, errInvalidParam("Type is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.portfolios[portfolioID]; !ok {
		return nil, errNotFound("Portfolio", portfolioID)
	}
	if _, ok := s.products[productID]; !ok {
		return nil, errNotFound("Product", productID)
	}
	id := newID("cons")
	c := &StoredConstraint{
		Id:          id,
		Type:        constraintType,
		Description: description,
		Parameters:  parameters,
		PortfolioId: portfolioID,
		ProductId:   productID,
		Status:      "AVAILABLE",
		Owner:       s.accountID,
	}
	s.constraints[id] = c
	return c, nil
}

// GetConstraint returns a constraint by id.
func (s *Store) GetConstraint(id string) (*StoredConstraint, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.constraints[id]
	if !ok {
		return nil, errNotFound("Constraint", id)
	}
	return c, nil
}

// DeleteConstraint removes a constraint.
func (s *Store) DeleteConstraint(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.constraints[id]; !ok {
		return errNotFound("Constraint", id)
	}
	delete(s.constraints, id)
	return nil
}

// ListConstraintsForPortfolio returns all constraints in a portfolio.
func (s *Store) ListConstraintsForPortfolio(portfolioID, productID string) []*StoredConstraint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredConstraint, 0)
	for _, c := range s.constraints {
		if c.PortfolioId != portfolioID {
			continue
		}
		if productID != "" && c.ProductId != productID {
			continue
		}
		out = append(out, c)
	}
	return out
}

// UpdateConstraint updates fields on a constraint.
func (s *Store) UpdateConstraint(id string, description, parameters *string) (*StoredConstraint, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.constraints[id]
	if !ok {
		return nil, errNotFound("Constraint", id)
	}
	if description != nil {
		c.Description = *description
	}
	if parameters != nil {
		c.Parameters = *parameters
	}
	return c, nil
}

// ── Principals ──────────────────────────────────────────────────────────────

// AssociatePrincipal associates a principal ARN with a portfolio.
func (s *Store) AssociatePrincipal(portfolioID, arn, principalType string) *service.AWSError {
	if arn == "" {
		return errInvalidParam("PrincipalARN is required")
	}
	if principalType == "" {
		principalType = "IAM"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.portfolios[portfolioID]; !ok {
		return errNotFound("Portfolio", portfolioID)
	}
	if s.principals[portfolioID] == nil {
		s.principals[portfolioID] = make(map[string]*StoredPrincipal)
	}
	s.principals[portfolioID][arn] = &StoredPrincipal{
		PortfolioId:   portfolioID,
		PrincipalARN:  arn,
		PrincipalType: principalType,
	}
	return nil
}

// DisassociatePrincipal removes a principal from a portfolio.
func (s *Store) DisassociatePrincipal(portfolioID, arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.portfolios[portfolioID]; !ok {
		return errNotFound("Portfolio", portfolioID)
	}
	if pmap, ok := s.principals[portfolioID]; ok {
		delete(pmap, arn)
	}
	return nil
}

// ListPrincipals returns all principals for a portfolio.
func (s *Store) ListPrincipals(portfolioID string) []*StoredPrincipal {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pmap := s.principals[portfolioID]
	out := make([]*StoredPrincipal, 0, len(pmap))
	for _, p := range pmap {
		out = append(out, p)
	}
	return out
}

// ── Tag Options ─────────────────────────────────────────────────────────────

// CreateTagOption inserts a new tag option.
func (s *Store) CreateTagOption(key, value string) (*StoredTagOption, *service.AWSError) {
	if key == "" {
		return nil, errInvalidParam("Key is required")
	}
	if value == "" {
		return nil, errInvalidParam("Value is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.tagOptions {
		if existing.Key == key && existing.Value == value {
			return nil, errDuplicate("TagOption", key+":"+value)
		}
	}
	id := newID("tag")
	t := &StoredTagOption{
		Id:     id,
		Key:    key,
		Value:  value,
		Active: true,
		Owner:  s.accountID,
	}
	s.tagOptions[id] = t
	return t, nil
}

// GetTagOption returns a tag option by id.
func (s *Store) GetTagOption(id string) (*StoredTagOption, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tagOptions[id]
	if !ok {
		return nil, errNotFound("TagOption", id)
	}
	return t, nil
}

// DeleteTagOption removes a tag option.
func (s *Store) DeleteTagOption(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tagOptions[id]; !ok {
		return errNotFound("TagOption", id)
	}
	delete(s.tagOptions, id)
	if rs, ok := s.tagOptionResources[id]; ok {
		for r := range rs {
			if rt, ok := s.resourceTagOptions[r]; ok {
				delete(rt, id)
			}
		}
		delete(s.tagOptionResources, id)
	}
	return nil
}

// ListTagOptions returns all tag options that match the optional filters.
func (s *Store) ListTagOptions(filterKey, filterValue string, active *bool) []*StoredTagOption {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredTagOption, 0)
	for _, t := range s.tagOptions {
		if filterKey != "" && t.Key != filterKey {
			continue
		}
		if filterValue != "" && t.Value != filterValue {
			continue
		}
		if active != nil && t.Active != *active {
			continue
		}
		out = append(out, t)
	}
	return out
}

// UpdateTagOption updates fields on a tag option.
func (s *Store) UpdateTagOption(id string, value *string, active *bool) (*StoredTagOption, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tagOptions[id]
	if !ok {
		return nil, errNotFound("TagOption", id)
	}
	if value != nil && *value != "" {
		t.Value = *value
	}
	if active != nil {
		t.Active = *active
	}
	return t, nil
}

// AssociateTagOption associates a tag option with a resource.
func (s *Store) AssociateTagOption(resourceID, tagOptionID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tagOptions[tagOptionID]; !ok {
		return errNotFound("TagOption", tagOptionID)
	}
	if s.tagOptionResources[tagOptionID] == nil {
		s.tagOptionResources[tagOptionID] = make(map[string]bool)
	}
	if s.resourceTagOptions[resourceID] == nil {
		s.resourceTagOptions[resourceID] = make(map[string]bool)
	}
	s.tagOptionResources[tagOptionID][resourceID] = true
	s.resourceTagOptions[resourceID][tagOptionID] = true
	return nil
}

// DisassociateTagOption removes the association between a tag option and a resource.
func (s *Store) DisassociateTagOption(resourceID, tagOptionID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rs, ok := s.tagOptionResources[tagOptionID]; ok {
		delete(rs, resourceID)
	}
	if ts, ok := s.resourceTagOptions[resourceID]; ok {
		delete(ts, tagOptionID)
	}
	return nil
}

// ListResourcesForTagOption returns all resources associated with a tag option.
func (s *Store) ListResourcesForTagOption(tagOptionID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0)
	for id := range s.tagOptionResources[tagOptionID] {
		out = append(out, id)
	}
	return out
}

// ── Portfolio ↔ Product associations ────────────────────────────────────────

// AssociateProductWithPortfolio links a product to a portfolio.
func (s *Store) AssociateProductWithPortfolio(portfolioID, productID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.portfolios[portfolioID]; !ok {
		return errNotFound("Portfolio", portfolioID)
	}
	if _, ok := s.products[productID]; !ok {
		return errNotFound("Product", productID)
	}
	if s.portfolioProducts[portfolioID] == nil {
		s.portfolioProducts[portfolioID] = make(map[string]bool)
	}
	if s.productPortfolios[productID] == nil {
		s.productPortfolios[productID] = make(map[string]bool)
	}
	s.portfolioProducts[portfolioID][productID] = true
	s.productPortfolios[productID][portfolioID] = true
	return nil
}

// DisassociateProductFromPortfolio removes the link.
func (s *Store) DisassociateProductFromPortfolio(portfolioID, productID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pp, ok := s.portfolioProducts[portfolioID]; ok {
		delete(pp, productID)
	}
	if pp, ok := s.productPortfolios[productID]; ok {
		delete(pp, portfolioID)
	}
	return nil
}

// ListProductsForPortfolio returns the products linked to a portfolio.
func (s *Store) ListProductsForPortfolio(portfolioID string) []*StoredProduct {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredProduct, 0)
	for productID := range s.portfolioProducts[portfolioID] {
		if p, ok := s.products[productID]; ok {
			out = append(out, p)
		}
	}
	return out
}

// ListPortfoliosForProduct returns the portfolios that contain a product.
func (s *Store) ListPortfoliosForProduct(productID string) []*StoredPortfolio {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredPortfolio, 0)
	for portfolioID := range s.productPortfolios[productID] {
		if p, ok := s.portfolios[portfolioID]; ok {
			out = append(out, p)
		}
	}
	return out
}

// ── Service actions ─────────────────────────────────────────────────────────

// CreateServiceAction inserts a new service action.
func (s *Store) CreateServiceAction(name, defType string, definition map[string]string, description string) (*StoredServiceAction, *service.AWSError) {
	if name == "" {
		return nil, errInvalidParam("Name is required")
	}
	if defType == "" {
		defType = "SSM_AUTOMATION"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newID("act")
	sa := &StoredServiceAction{
		Id:             id,
		Name:           name,
		DefinitionType: defType,
		Definition:     definition,
		Description:    description,
	}
	s.serviceActions[id] = sa
	return sa, nil
}

// GetServiceAction returns a service action by id.
func (s *Store) GetServiceAction(id string) (*StoredServiceAction, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sa, ok := s.serviceActions[id]
	if !ok {
		return nil, errNotFound("ServiceAction", id)
	}
	return sa, nil
}

// DeleteServiceAction removes a service action.
func (s *Store) DeleteServiceAction(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.serviceActions[id]; !ok {
		return errNotFound("ServiceAction", id)
	}
	delete(s.serviceActions, id)
	if as, ok := s.saArtifacts[id]; ok {
		for k := range as {
			if sas, ok := s.artifactSAs[k]; ok {
				delete(sas, id)
			}
		}
		delete(s.saArtifacts, id)
	}
	return nil
}

// ListServiceActions returns all service actions.
func (s *Store) ListServiceActions() []*StoredServiceAction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredServiceAction, 0, len(s.serviceActions))
	for _, sa := range s.serviceActions {
		out = append(out, sa)
	}
	return out
}

// UpdateServiceAction updates fields on a service action.
func (s *Store) UpdateServiceAction(id string, name, description *string, definition map[string]string) (*StoredServiceAction, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sa, ok := s.serviceActions[id]
	if !ok {
		return nil, errNotFound("ServiceAction", id)
	}
	if name != nil && *name != "" {
		sa.Name = *name
	}
	if description != nil {
		sa.Description = *description
	}
	if definition != nil {
		sa.Definition = definition
	}
	return sa, nil
}

// AssociateServiceActionWithArtifact links a service action to a (product, artifact).
func (s *Store) AssociateServiceActionWithArtifact(productID, artifactID, serviceActionID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.serviceActions[serviceActionID]; !ok {
		return errNotFound("ServiceAction", serviceActionID)
	}
	pas, ok := s.artifacts[productID]
	if !ok {
		return errNotFound("ProvisioningArtifact", artifactID)
	}
	if _, ok := pas[artifactID]; !ok {
		return errNotFound("ProvisioningArtifact", artifactID)
	}
	key := productID + "|" + artifactID
	if s.saArtifacts[serviceActionID] == nil {
		s.saArtifacts[serviceActionID] = make(map[string]bool)
	}
	if s.artifactSAs[key] == nil {
		s.artifactSAs[key] = make(map[string]bool)
	}
	s.saArtifacts[serviceActionID][key] = true
	s.artifactSAs[key][serviceActionID] = true
	return nil
}

// DisassociateServiceActionFromArtifact removes the association.
func (s *Store) DisassociateServiceActionFromArtifact(productID, artifactID, serviceActionID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := productID + "|" + artifactID
	if as, ok := s.saArtifacts[serviceActionID]; ok {
		delete(as, key)
	}
	if sas, ok := s.artifactSAs[key]; ok {
		delete(sas, serviceActionID)
	}
	return nil
}

// ListServiceActionsForArtifact returns service actions linked to a (product, artifact).
func (s *Store) ListServiceActionsForArtifact(productID, artifactID string) []*StoredServiceAction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := productID + "|" + artifactID
	out := make([]*StoredServiceAction, 0)
	for saID := range s.artifactSAs[key] {
		if sa, ok := s.serviceActions[saID]; ok {
			out = append(out, sa)
		}
	}
	return out
}

// ListArtifactsForServiceAction returns the (product, artifact) pairs linked to a service action.
func (s *Store) ListArtifactsForServiceAction(serviceActionID string) []*StoredProvisioningArtifact {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredProvisioningArtifact, 0)
	for key := range s.saArtifacts[serviceActionID] {
		parts := strings.SplitN(key, "|", 2)
		if len(parts) != 2 {
			continue
		}
		if pas, ok := s.artifacts[parts[0]]; ok {
			if pa, ok := pas[parts[1]]; ok {
				out = append(out, pa)
			}
		}
	}
	return out
}

// ── Provisioned products ────────────────────────────────────────────────────

// ProvisionProduct creates a new provisioned product and a PROVISION record.
func (s *Store) ProvisionProduct(name, productID, artifactID, pathID, idempotencyToken string, tags map[string]string, userArn string) (*StoredProvisionedProduct, *StoredRecord, *service.AWSError) {
	if name == "" {
		return nil, nil, errInvalidParam("ProvisionedProductName is required")
	}
	if productID == "" {
		return nil, nil, errInvalidParam("ProductId is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	prod, ok := s.products[productID]
	if !ok {
		return nil, nil, errNotFound("Product", productID)
	}
	for _, pp := range s.provisioned {
		if pp.Name == name {
			return nil, nil, errDuplicate("ProvisionedProduct", name)
		}
	}
	var artifact *StoredProvisioningArtifact
	if pas, ok := s.artifacts[productID]; ok {
		if artifactID != "" {
			if pa, ok := pas[artifactID]; ok {
				artifact = pa
			}
		} else {
			for _, pa := range pas {
				artifact = pa
				artifactID = pa.Id
				break
			}
		}
	}
	if artifact == nil && artifactID != "" {
		return nil, nil, errNotFound("ProvisioningArtifact", artifactID)
	}
	id := newID("pp")
	now := time.Now().UTC()
	recordID := newID("rec")
	pp := &StoredProvisionedProduct{
		Id:                       id,
		Name:                     name,
		Type:                     "CFN_STACK",
		Arn:                      fmt.Sprintf("arn:aws:servicecatalog:%s:%s:stack/%s/%s", s.region, s.accountID, name, id),
		Status:                   "AVAILABLE",
		StatusMessage:            "Successfully provisioned",
		CreatedTime:              now,
		LastRecordId:             recordID,
		ProductId:                productID,
		ProductName:              prod.Name,
		ProvisioningArtifactId:   artifactID,
		UserArn:                  userArn,
		UserArnSession:           userArn,
		Tags:                     copyTagMap(tags),
		IdempotencyToken:         idempotencyToken,
		Outputs:                  []map[string]any{},
	}
	if artifact != nil {
		pp.ProvisioningArtifactName = artifact.Name
	}
	s.provisioned[id] = pp

	rec := &StoredRecord{
		Id:                     recordID,
		ProvisionedProductName: name,
		ProvisionedProductType: "CFN_STACK",
		RecordType:             "PROVISION_PRODUCT",
		ProvisionedProductId:   id,
		Status:                 "SUCCEEDED",
		CreatedTime:             now,
		UpdatedTime:             now,
		ProductId:               productID,
		ProvisioningArtifactId:  artifactID,
		PathId:                  pathID,
		RecordTags:              copyTagMap(tags),
	}
	s.records[recordID] = rec
	return pp, rec, nil
}

// UpdateProvisionedProduct creates an UPDATE record and updates the provisioned product.
func (s *Store) UpdateProvisionedProduct(idOrName, productID, artifactID, pathID string, tags map[string]string) (*StoredProvisionedProduct, *StoredRecord, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pp := s.findProvisionedLocked(idOrName)
	if pp == nil {
		return nil, nil, errNotFound("ProvisionedProduct", idOrName)
	}
	if productID != "" {
		pp.ProductId = productID
	}
	if artifactID != "" {
		pp.ProvisioningArtifactId = artifactID
	}
	if tags != nil {
		pp.Tags = copyTagMap(tags)
	}
	now := time.Now().UTC()
	recordID := newID("rec")
	pp.LastRecordId = recordID
	pp.Status = "AVAILABLE"
	rec := &StoredRecord{
		Id:                     recordID,
		ProvisionedProductName: pp.Name,
		ProvisionedProductType: pp.Type,
		RecordType:             "UPDATE_PROVISIONED_PRODUCT",
		ProvisionedProductId:   pp.Id,
		Status:                 "SUCCEEDED",
		CreatedTime:             now,
		UpdatedTime:             now,
		ProductId:               pp.ProductId,
		ProvisioningArtifactId:  pp.ProvisioningArtifactId,
		PathId:                  pathID,
		RecordTags:              copyTagMap(pp.Tags),
	}
	s.records[recordID] = rec
	return pp, rec, nil
}

// TerminateProvisionedProduct creates a TERMINATE record and removes the provisioned product.
func (s *Store) TerminateProvisionedProduct(idOrName string) (*StoredRecord, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pp := s.findProvisionedLocked(idOrName)
	if pp == nil {
		return nil, errNotFound("ProvisionedProduct", idOrName)
	}
	now := time.Now().UTC()
	recordID := newID("rec")
	rec := &StoredRecord{
		Id:                     recordID,
		ProvisionedProductName: pp.Name,
		ProvisionedProductType: pp.Type,
		RecordType:             "TERMINATE_PROVISIONED_PRODUCT",
		ProvisionedProductId:   pp.Id,
		Status:                 "SUCCEEDED",
		CreatedTime:             now,
		UpdatedTime:             now,
		ProductId:               pp.ProductId,
		ProvisioningArtifactId:  pp.ProvisioningArtifactId,
	}
	s.records[recordID] = rec
	delete(s.provisioned, pp.Id)
	return rec, nil
}

// GetProvisionedProduct fetches a provisioned product by id or name.
func (s *Store) GetProvisionedProduct(idOrName string) (*StoredProvisionedProduct, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pp := s.findProvisionedLocked(idOrName)
	if pp == nil {
		return nil, errNotFound("ProvisionedProduct", idOrName)
	}
	return pp, nil
}

func (s *Store) findProvisionedLocked(idOrName string) *StoredProvisionedProduct {
	if pp, ok := s.provisioned[idOrName]; ok {
		return pp
	}
	for _, pp := range s.provisioned {
		if pp.Name == idOrName {
			return pp
		}
	}
	return nil
}

// ListProvisionedProducts returns all provisioned products.
func (s *Store) ListProvisionedProducts() []*StoredProvisionedProduct {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredProvisionedProduct, 0, len(s.provisioned))
	for _, pp := range s.provisioned {
		out = append(out, pp)
	}
	return out
}

// SetProvisionedProductOutputs replaces the outputs slice (for tests/notification).
func (s *Store) SetProvisionedProductOutputs(idOrName string, outputs []map[string]any) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	pp := s.findProvisionedLocked(idOrName)
	if pp == nil {
		return errNotFound("ProvisionedProduct", idOrName)
	}
	pp.Outputs = outputs
	return nil
}

// UpdateProvisionedProductProperties updates ad-hoc properties on a provisioned product.
func (s *Store) UpdateProvisionedProductProperties(id string, props map[string]any) (*StoredProvisionedProduct, string, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pp := s.findProvisionedLocked(id)
	if pp == nil {
		return nil, "", errNotFound("ProvisionedProduct", id)
	}
	if v, ok := props["OWNER"]; ok {
		if s, ok := v.(string); ok {
			pp.UserArn = s
		}
	}
	recordID := newID("rec")
	pp.LastRecordId = recordID
	now := time.Now().UTC()
	s.records[recordID] = &StoredRecord{
		Id:                   recordID,
		ProvisionedProductName: pp.Name,
		ProvisionedProductType: pp.Type,
		RecordType:           "UPDATE_PROVISIONED_PRODUCT_PROPERTIES",
		ProvisionedProductId: pp.Id,
		Status:               "SUCCEEDED",
		CreatedTime:           now,
		UpdatedTime:           now,
	}
	return pp, recordID, nil
}

// ── Records ─────────────────────────────────────────────────────────────────

// GetRecord returns a record by id.
func (s *Store) GetRecord(id string) (*StoredRecord, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.records[id]
	if !ok {
		return nil, errNotFound("Record", id)
	}
	return r, nil
}

// ListRecords returns records, optionally filtered by provisioned-product id/name.
func (s *Store) ListRecords(provisionedProductID, provisionedProductName string) []*StoredRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredRecord, 0, len(s.records))
	for _, r := range s.records {
		if provisionedProductID != "" && r.ProvisionedProductId != provisionedProductID {
			continue
		}
		if provisionedProductName != "" && r.ProvisionedProductName != provisionedProductName {
			continue
		}
		out = append(out, r)
	}
	return out
}

// ── Plans ───────────────────────────────────────────────────────────────────

// CreatePlan inserts a new provisioned product plan.
func (s *Store) CreatePlan(name, planType, productID, pathID, artifactID, ppName string, params []map[string]any, notifications []string, tags map[string]string) (*StoredPlan, *service.AWSError) {
	if name == "" {
		return nil, errInvalidParam("PlanName is required")
	}
	if planType == "" {
		planType = "CLOUDFORMATION"
	}
	if productID == "" {
		return nil, errInvalidParam("ProductId is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.products[productID]; !ok {
		return nil, errNotFound("Product", productID)
	}
	id := newID("plan")
	now := time.Now().UTC()
	p := &StoredPlan{
		Id:                     id,
		Name:                   name,
		Type:                   planType,
		ProductId:              productID,
		PathId:                 pathID,
		ProvisioningArtifactId: artifactID,
		ProvisionedProductName: ppName,
		ProvisioningParameters: params,
		NotificationArns:       notifications,
		Tags:                   copyTagMap(tags),
		Status:                 "CREATE_SUCCESS",
		StatusMessage:          "Plan created",
		CreatedTime:             now,
		UpdatedTime:             now,
	}
	s.plans[id] = p
	return p, nil
}

// GetPlan returns a plan by id.
func (s *Store) GetPlan(id string) (*StoredPlan, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.plans[id]
	if !ok {
		return nil, errNotFound("Plan", id)
	}
	return p, nil
}

// DeletePlan removes a plan.
func (s *Store) DeletePlan(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.plans[id]; !ok {
		return errNotFound("Plan", id)
	}
	delete(s.plans, id)
	return nil
}

// ListPlans returns all plans, optionally filtered by provisioned-product name.
func (s *Store) ListPlans(provisionedProductName string) []*StoredPlan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredPlan, 0, len(s.plans))
	for _, p := range s.plans {
		if provisionedProductName != "" && p.ProvisionedProductName != provisionedProductName {
			continue
		}
		out = append(out, p)
	}
	return out
}

// ExecutePlan moves a plan into EXECUTE_SUCCESS.
func (s *Store) ExecutePlan(id string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.plans[id]
	if !ok {
		return errNotFound("Plan", id)
	}
	p.Status = "EXECUTE_SUCCESS"
	p.StatusMessage = "Plan executed"
	p.UpdatedTime = time.Now().UTC()
	return nil
}

// ── Budgets ─────────────────────────────────────────────────────────────────

// AssociateBudget links a budget name to a resource.
func (s *Store) AssociateBudget(budgetName, resourceID string) *service.AWSError {
	if budgetName == "" || resourceID == "" {
		return errInvalidParam("BudgetName and ResourceId are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.resourceBudgets[resourceID] == nil {
		s.resourceBudgets[resourceID] = make(map[string]bool)
	}
	if s.budgetResources[budgetName] == nil {
		s.budgetResources[budgetName] = make(map[string]bool)
	}
	s.resourceBudgets[resourceID][budgetName] = true
	s.budgetResources[budgetName][resourceID] = true
	return nil
}

// DisassociateBudget unlinks a budget from a resource.
func (s *Store) DisassociateBudget(budgetName, resourceID string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rb, ok := s.resourceBudgets[resourceID]; ok {
		delete(rb, budgetName)
	}
	if br, ok := s.budgetResources[budgetName]; ok {
		delete(br, resourceID)
	}
	return nil
}

// ListBudgetsForResource returns all budgets associated with a resource.
func (s *Store) ListBudgetsForResource(resourceID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0)
	for b := range s.resourceBudgets[resourceID] {
		out = append(out, b)
	}
	return out
}

// ── AWS Organizations access ────────────────────────────────────────────────

// SetAWSOrganizationsAccess updates the AWS Organizations access flag.
func (s *Store) SetAWSOrganizationsAccess(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.awsOrgAccess = status
}

// GetAWSOrganizationsAccess returns the current Organizations status.
func (s *Store) GetAWSOrganizationsAccess() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.awsOrgAccess
}

// ── Copy operations ─────────────────────────────────────────────────────────

// StartCopyProduct registers an in-flight copy operation and returns its token.
func (s *Store) StartCopyProduct(sourceArn, targetID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	token := newToken()
	s.copies[token] = &StoredCopyOperation{
		Token:           token,
		SourceProductArn: sourceArn,
		TargetProductId: targetID,
		Status:          "SUCCEEDED",
		StatusDetail:    "Copy completed",
	}
	return token
}

// GetCopyOperation returns a copy operation by token.
func (s *Store) GetCopyOperation(token string) (*StoredCopyOperation, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.copies[token]
	if !ok {
		return nil, errNotFound("CopyOperation", token)
	}
	return c, nil
}
