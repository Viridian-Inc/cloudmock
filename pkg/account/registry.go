package account

import (
	"fmt"
	"sync"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ServiceFactory creates a service instance for a given account ID and region.
type ServiceFactory func(accountID, region string) service.Service

// Registry manages multiple AWS accounts with isolated service instances.
// Each account gets its own set of services, created on-demand from registered
// factories. This enables cross-account STS AssumeRole and Organizations
// integration where each account has isolated resources.
type Registry struct {
	mu        sync.RWMutex
	accounts  map[string]*Account
	defaultID string
	region    string
	factories map[string]ServiceFactory // service name -> factory

	// Credential-to-account mapping for STS temporary credentials.
	credMap map[string]string // accessKeyID -> accountID
	credMu  sync.RWMutex
}

// Account represents an isolated AWS account with its own service instances.
type Account struct {
	ID       string
	Name     string
	services map[string]service.Service // lazy-initialized per service
	mu       sync.Mutex
}

// NewRegistry creates a new account registry with a default account.
// The default account is created automatically and is used when no specific
// account is resolved from credentials.
func NewRegistry(defaultID, region string) *Registry {
	r := &Registry{
		accounts:  make(map[string]*Account),
		defaultID: defaultID,
		region:    region,
		factories: make(map[string]ServiceFactory),
		credMap:   make(map[string]string),
	}

	// Create the default account.
	r.accounts[defaultID] = &Account{
		ID:       defaultID,
		Name:     "Default Account",
		services: make(map[string]service.Service),
	}

	return r
}

// RegisterFactory registers a factory for creating service instances per account.
// When a service is requested for an account that doesn't have it yet, the
// factory is called to create a new isolated instance.
func (r *Registry) RegisterFactory(serviceName string, factory ServiceFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[serviceName] = factory
}

// CreateAccount creates a new isolated account. Returns an error if an account
// with the same ID already exists.
func (r *Registry) CreateAccount(id, name string) (*Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.accounts[id]; exists {
		return nil, fmt.Errorf("account %q already exists", id)
	}

	acct := &Account{
		ID:       id,
		Name:     name,
		services: make(map[string]service.Service),
	}
	r.accounts[id] = acct
	return acct, nil
}

// GetAccount returns an account by ID.
func (r *Registry) GetAccount(id string) (*Account, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	acct, ok := r.accounts[id]
	return acct, ok
}

// ListAccounts returns all accounts in the registry.
func (r *Registry) ListAccounts() []*Account {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Account, 0, len(r.accounts))
	for _, acct := range r.accounts {
		out = append(out, acct)
	}
	return out
}

// GetService returns a service for the given account, creating it on-demand
// from the registered factory if it doesn't exist yet. Returns false if the
// account doesn't exist or no factory is registered for the service.
func (r *Registry) GetService(accountID, serviceName string) (service.Service, bool) {
	r.mu.RLock()
	acct, ok := r.accounts[accountID]
	r.mu.RUnlock()
	if !ok {
		return nil, false
	}

	acct.mu.Lock()
	defer acct.mu.Unlock()

	// Check if already initialized.
	if svc, ok := acct.services[serviceName]; ok {
		return svc, true
	}

	// Look up the factory.
	r.mu.RLock()
	factory, hasFactory := r.factories[serviceName]
	r.mu.RUnlock()
	if !hasFactory {
		return nil, false
	}

	// Create and cache the service instance.
	svc := factory(accountID, r.region)
	acct.services[serviceName] = svc
	return svc, true
}

// MapCredential associates a temporary credential (access key ID) with an account.
// This is called by STS when issuing credentials for cross-account AssumeRole.
func (r *Registry) MapCredential(accessKeyID, accountID string) {
	r.credMu.Lock()
	defer r.credMu.Unlock()
	r.credMap[accessKeyID] = accountID
}

// ResolveCredential returns the account ID for a temporary credential.
func (r *Registry) ResolveCredential(accessKeyID string) (string, bool) {
	r.credMu.RLock()
	defer r.credMu.RUnlock()
	id, ok := r.credMap[accessKeyID]
	return id, ok
}

// ProvisionAccount creates a new account, satisfying the organizations.AccountProvisioner interface.
// If the account already exists, it returns nil (idempotent).
func (r *Registry) ProvisionAccount(id, name string) error {
	_, err := r.CreateAccount(id, name)
	if err != nil {
		// Treat duplicate as success (idempotent).
		if _, exists := r.GetAccount(id); exists {
			return nil
		}
		return err
	}
	return nil
}

// Default returns the default account ID.
func (r *Registry) Default() string {
	return r.defaultID
}
