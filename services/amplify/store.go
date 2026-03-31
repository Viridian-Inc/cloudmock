package amplify

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// App represents an Amplify application.
type App struct {
	AppId              string
	AppArn             string
	Name               string
	Description        string
	Repository         string
	Platform           string
	IamServiceRoleArn  string
	DefaultDomain      string
	EnableBranchAutoBuild bool
	CreateTime         time.Time
	UpdateTime         time.Time
	Tags               map[string]string
}

// Branch represents an Amplify branch.
type Branch struct {
	BranchArn          string
	BranchName         string
	AppId              string
	Description        string
	Stage              string
	Framework          string
	EnableAutoBuild    bool
	EnableNotification bool
	DisplayName        string
	CreateTime         time.Time
	UpdateTime         time.Time
	Tags               map[string]string
}

// DomainAssociation represents an Amplify domain association.
type DomainAssociation struct {
	DomainAssociationArn string
	DomainName           string
	AppId                string
	EnableAutoSubDomain  bool
	DomainStatus         string
	SubDomains           []SubDomain
	CertificateVerificationDNSRecord string
	Tags                 map[string]string
}

// SubDomain represents a sub-domain within a domain association.
type SubDomain struct {
	Prefix       string
	BranchName   string
	DnsRecord    string
	Verified     bool
}

// Webhook represents an Amplify webhook.
type Webhook struct {
	WebhookArn  string
	WebhookId   string
	WebhookUrl  string
	AppId       string
	BranchName  string
	Description string
	CreateTime  time.Time
	UpdateTime  time.Time
	Tags        map[string]string
}

// Job represents an Amplify build job.
type Job struct {
	JobArn     string
	JobId      string
	AppId      string
	BranchName string
	JobType    string
	Status     string
	CommitId   string
	CommitMessage string
	StartTime  time.Time
	EndTime    *time.Time
	Lifecycle  *lifecycle.Machine
	Tags       map[string]string
}

// Store manages all Amplify state in memory.
type Store struct {
	mu              sync.RWMutex
	apps            map[string]*App
	branches        map[string]map[string]*Branch            // appID -> branchName -> branch
	domains         map[string]map[string]*DomainAssociation // appID -> domainName -> domain
	webhooks        map[string]map[string]*Webhook           // appID -> webhookID -> webhook
	jobs            map[string]map[string][]*Job             // appID -> branchName -> jobs
	accountID       string
	region          string
	nextJobNum      map[string]int // appID:branchName -> next job number
	lifecycleConfig *lifecycle.Config
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		apps:            make(map[string]*App),
		branches:        make(map[string]map[string]*Branch),
		domains:         make(map[string]map[string]*DomainAssociation),
		webhooks:        make(map[string]map[string]*Webhook),
		jobs:            make(map[string]map[string][]*Job),
		accountID:       accountID,
		region:          region,
		nextJobNum:      make(map[string]int),
		lifecycleConfig: lifecycle.DefaultConfig(),
	}
}

func newID() string {
	b := make([]byte, 13)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)[:26]
}

func newShortID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)[:10]
}

func (s *Store) appARN(appID string) string {
	return fmt.Sprintf("arn:aws:amplify:%s:%s:apps/%s", s.region, s.accountID, appID)
}

func (s *Store) branchARN(appID, branchName string) string {
	return fmt.Sprintf("arn:aws:amplify:%s:%s:apps/%s/branches/%s", s.region, s.accountID, appID, branchName)
}

func (s *Store) domainARN(appID, domainName string) string {
	return fmt.Sprintf("arn:aws:amplify:%s:%s:apps/%s/domains/%s", s.region, s.accountID, appID, domainName)
}

func (s *Store) webhookARN(webhookID string) string {
	return fmt.Sprintf("arn:aws:amplify:%s:%s:webhooks/%s", s.region, s.accountID, webhookID)
}

func (s *Store) jobARN(appID, branchName, jobID string) string {
	return fmt.Sprintf("arn:aws:amplify:%s:%s:apps/%s/branches/%s/jobs/%s", s.region, s.accountID, appID, branchName, jobID)
}

// CreateApp creates a new Amplify app.
func (s *Store) CreateApp(name, description, repository, platform, iamRole string, tags map[string]string) *App {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tags == nil {
		tags = make(map[string]string)
	}
	if platform == "" {
		platform = "WEB"
	}
	id := newID()
	now := time.Now().UTC()
	app := &App{
		AppId: id, AppArn: s.appARN(id), Name: name,
		Description: description, Repository: repository,
		Platform: platform, IamServiceRoleArn: iamRole,
		DefaultDomain: fmt.Sprintf("%s.amplifyapp.com", id),
		CreateTime: now, UpdateTime: now, Tags: tags,
	}
	s.apps[id] = app
	s.branches[id] = make(map[string]*Branch)
	s.domains[id] = make(map[string]*DomainAssociation)
	s.webhooks[id] = make(map[string]*Webhook)
	s.jobs[id] = make(map[string][]*Job)
	return app
}

// GetApp returns an app by ID.
func (s *Store) GetApp(appID string) (*App, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	app, ok := s.apps[appID]
	return app, ok
}

// ListApps returns all apps.
func (s *Store) ListApps() []*App {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*App, 0, len(s.apps))
	for _, app := range s.apps {
		result = append(result, app)
	}
	return result
}

// UpdateApp updates an app.
func (s *Store) UpdateApp(appID, name, description, platform, iamRole string) (*App, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	app, ok := s.apps[appID]
	if !ok {
		return nil, false
	}
	if name != "" {
		app.Name = name
	}
	if description != "" {
		app.Description = description
	}
	if platform != "" {
		app.Platform = platform
	}
	if iamRole != "" {
		app.IamServiceRoleArn = iamRole
	}
	app.UpdateTime = time.Now().UTC()
	return app, true
}

// DeleteApp removes an app and all associated resources.
func (s *Store) DeleteApp(appID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apps[appID]; !ok {
		return false
	}
	delete(s.apps, appID)
	delete(s.branches, appID)
	delete(s.domains, appID)
	delete(s.webhooks, appID)
	delete(s.jobs, appID)
	return true
}

// CreateBranch creates a new branch for an app.
func (s *Store) CreateBranch(appID, branchName, description, stage, framework string, enableAutoBuild bool, tags map[string]string) (*Branch, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apps[appID]; !ok {
		return nil, false
	}
	if _, ok := s.branches[appID][branchName]; ok {
		return nil, false
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	if stage == "" {
		stage = "NONE"
	}
	now := time.Now().UTC()
	branch := &Branch{
		BranchArn: s.branchARN(appID, branchName), BranchName: branchName,
		AppId: appID, Description: description, Stage: stage,
		Framework: framework, EnableAutoBuild: enableAutoBuild,
		DisplayName: branchName, CreateTime: now, UpdateTime: now, Tags: tags,
	}
	s.branches[appID][branchName] = branch
	return branch, true
}

// GetBranch returns a branch.
func (s *Store) GetBranch(appID, branchName string) (*Branch, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	branches, ok := s.branches[appID]
	if !ok {
		return nil, false
	}
	branch, ok := branches[branchName]
	return branch, ok
}

// ListBranches returns all branches for an app.
func (s *Store) ListBranches(appID string) ([]*Branch, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	branches, ok := s.branches[appID]
	if !ok {
		return nil, false
	}
	result := make([]*Branch, 0, len(branches))
	for _, b := range branches {
		result = append(result, b)
	}
	return result, true
}

// UpdateBranch updates a branch.
func (s *Store) UpdateBranch(appID, branchName, description, stage, framework string) (*Branch, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	branches, ok := s.branches[appID]
	if !ok {
		return nil, false
	}
	branch, ok := branches[branchName]
	if !ok {
		return nil, false
	}
	if description != "" {
		branch.Description = description
	}
	if stage != "" {
		branch.Stage = stage
	}
	if framework != "" {
		branch.Framework = framework
	}
	branch.UpdateTime = time.Now().UTC()
	return branch, true
}

// DeleteBranch removes a branch.
func (s *Store) DeleteBranch(appID, branchName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	branches, ok := s.branches[appID]
	if !ok {
		return false
	}
	if _, ok := branches[branchName]; !ok {
		return false
	}
	delete(branches, branchName)
	return true
}

// CreateDomainAssociation creates a new domain association.
func (s *Store) CreateDomainAssociation(appID, domainName string, enableAutoSubDomain bool, subDomains []SubDomain, tags map[string]string) (*DomainAssociation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apps[appID]; !ok {
		return nil, false
	}
	if _, ok := s.domains[appID][domainName]; ok {
		return nil, false
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	domain := &DomainAssociation{
		DomainAssociationArn: s.domainARN(appID, domainName),
		DomainName: domainName, AppId: appID,
		EnableAutoSubDomain: enableAutoSubDomain,
		DomainStatus: "CREATING", SubDomains: subDomains, Tags: tags,
	}
	s.domains[appID][domainName] = domain
	return domain, true
}

// GetDomainAssociation returns a domain association.
func (s *Store) GetDomainAssociation(appID, domainName string) (*DomainAssociation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	domains, ok := s.domains[appID]
	if !ok {
		return nil, false
	}
	domain, ok := domains[domainName]
	return domain, ok
}

// ListDomainAssociations returns all domain associations for an app.
func (s *Store) ListDomainAssociations(appID string) ([]*DomainAssociation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	domains, ok := s.domains[appID]
	if !ok {
		return nil, false
	}
	result := make([]*DomainAssociation, 0, len(domains))
	for _, d := range domains {
		result = append(result, d)
	}
	return result, true
}

// UpdateDomainAssociation updates a domain association.
func (s *Store) UpdateDomainAssociation(appID, domainName string, enableAutoSubDomain *bool, subDomains []SubDomain) (*DomainAssociation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	domains, ok := s.domains[appID]
	if !ok {
		return nil, false
	}
	domain, ok := domains[domainName]
	if !ok {
		return nil, false
	}
	if enableAutoSubDomain != nil {
		domain.EnableAutoSubDomain = *enableAutoSubDomain
	}
	if subDomains != nil {
		domain.SubDomains = subDomains
	}
	return domain, true
}

// DeleteDomainAssociation removes a domain association.
func (s *Store) DeleteDomainAssociation(appID, domainName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	domains, ok := s.domains[appID]
	if !ok {
		return false
	}
	if _, ok := domains[domainName]; !ok {
		return false
	}
	delete(domains, domainName)
	return true
}

// CreateWebhook creates a new webhook.
func (s *Store) CreateWebhook(appID, branchName, description string, tags map[string]string) (*Webhook, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apps[appID]; !ok {
		return nil, false
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	id := newShortID()
	now := time.Now().UTC()
	webhook := &Webhook{
		WebhookArn: s.webhookARN(id), WebhookId: id,
		WebhookUrl: fmt.Sprintf("https://webhooks.amplify.%s.amazonaws.com/prod/webhooks?id=%s&token=%s", s.region, id, newShortID()),
		AppId: appID, BranchName: branchName, Description: description,
		CreateTime: now, UpdateTime: now, Tags: tags,
	}
	s.webhooks[appID][id] = webhook
	return webhook, true
}

// GetWebhook returns a webhook.
func (s *Store) GetWebhook(webhookID string) (*Webhook, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, webhooks := range s.webhooks {
		if wh, ok := webhooks[webhookID]; ok {
			return wh, true
		}
	}
	return nil, false
}

// ListWebhooks returns all webhooks for an app.
func (s *Store) ListWebhooks(appID string) ([]*Webhook, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	webhooks, ok := s.webhooks[appID]
	if !ok {
		return nil, false
	}
	result := make([]*Webhook, 0, len(webhooks))
	for _, wh := range webhooks {
		result = append(result, wh)
	}
	return result, true
}

// UpdateWebhook updates a webhook.
func (s *Store) UpdateWebhook(webhookID, branchName, description string) (*Webhook, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, webhooks := range s.webhooks {
		if wh, ok := webhooks[webhookID]; ok {
			if branchName != "" {
				wh.BranchName = branchName
			}
			if description != "" {
				wh.Description = description
			}
			wh.UpdateTime = time.Now().UTC()
			return wh, true
		}
	}
	return nil, false
}

// DeleteWebhook removes a webhook.
func (s *Store) DeleteWebhook(webhookID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, webhooks := range s.webhooks {
		if _, ok := webhooks[webhookID]; ok {
			delete(webhooks, webhookID)
			return true
		}
	}
	return false
}

// StartJob starts a new build job.
func (s *Store) StartJob(appID, branchName, jobType, commitID, commitMessage string, tags map[string]string) (*Job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apps[appID]; !ok {
		return nil, false
	}
	branches, ok := s.branches[appID]
	if !ok {
		return nil, false
	}
	if _, ok := branches[branchName]; !ok {
		return nil, false
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	if jobType == "" {
		jobType = "RELEASE"
	}

	key := appID + ":" + branchName
	s.nextJobNum[key]++
	jobID := fmt.Sprintf("%d", s.nextJobNum[key])

	transitions := []lifecycle.Transition{
		{From: "PENDING", To: "RUNNING", Delay: 2 * time.Second},
		{From: "RUNNING", To: "SUCCEED", Delay: 5 * time.Second},
	}
	lm := lifecycle.NewMachine("PENDING", transitions, s.lifecycleConfig)

	now := time.Now().UTC()
	job := &Job{
		JobArn: s.jobARN(appID, branchName, jobID), JobId: jobID,
		AppId: appID, BranchName: branchName, JobType: jobType,
		Status: "PENDING", CommitId: commitID, CommitMessage: commitMessage,
		StartTime: now, Lifecycle: lm, Tags: tags,
	}

	lm.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		job.Status = string(to)
		if string(to) == "SUCCEED" || string(to) == "FAIL" {
			t := time.Now().UTC()
			job.EndTime = &t
		}
	})

	if s.jobs[appID] == nil {
		s.jobs[appID] = make(map[string][]*Job)
	}
	s.jobs[appID][branchName] = append(s.jobs[appID][branchName], job)
	return job, true
}

// GetJob returns a job.
func (s *Store) GetJob(appID, branchName, jobID string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	appJobs, ok := s.jobs[appID]
	if !ok {
		return nil, false
	}
	branchJobs, ok := appJobs[branchName]
	if !ok {
		return nil, false
	}
	for _, job := range branchJobs {
		if job.JobId == jobID {
			job.Status = string(job.Lifecycle.State())
			return job, true
		}
	}
	return nil, false
}

// ListJobs returns all jobs for a branch.
func (s *Store) ListJobs(appID, branchName string) ([]*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	appJobs, ok := s.jobs[appID]
	if !ok {
		return nil, false
	}
	branchJobs, ok := appJobs[branchName]
	if !ok {
		return nil, false
	}
	for _, job := range branchJobs {
		job.Status = string(job.Lifecycle.State())
	}
	result := make([]*Job, len(branchJobs))
	copy(result, branchJobs)
	return result, true
}

// StopJob stops a running job.
func (s *Store) StopJob(appID, branchName, jobID string) (*Job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	appJobs, ok := s.jobs[appID]
	if !ok {
		return nil, false
	}
	branchJobs, ok := appJobs[branchName]
	if !ok {
		return nil, false
	}
	for _, job := range branchJobs {
		if job.JobId == jobID {
			job.Lifecycle.Stop()
			job.Lifecycle.ForceState("CANCELLED")
			job.Status = "CANCELLED"
			t := time.Now().UTC()
			job.EndTime = &t
			return job, true
		}
	}
	return nil, false
}

// TagResource applies tags to a resource by ARN.
func (s *Store) TagResource(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r := s.findTagsByARN(arn); r != nil {
		for k, v := range tags {
			r[k] = v
		}
		return true
	}
	return false
}

// UntagResource removes tags from a resource by ARN.
func (s *Store) UntagResource(arn string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r := s.findTagsByARN(arn); r != nil {
		for _, k := range keys {
			delete(r, k)
		}
		return true
	}
	return false
}

// ListTagsForResource returns tags for a resource by ARN.
func (s *Store) ListTagsForResource(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if r := s.findTagsByARN(arn); r != nil {
		cp := make(map[string]string, len(r))
		for k, v := range r {
			cp[k] = v
		}
		return cp, true
	}
	return nil, false
}

func (s *Store) findTagsByARN(arn string) map[string]string {
	for _, app := range s.apps {
		if app.AppArn == arn {
			return app.Tags
		}
	}
	for _, branches := range s.branches {
		for _, b := range branches {
			if b.BranchArn == arn {
				return b.Tags
			}
		}
	}
	for _, webhooks := range s.webhooks {
		for _, wh := range webhooks {
			if wh.WebhookArn == arn {
				return wh.Tags
			}
		}
	}
	return nil
}
