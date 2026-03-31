package support

import (
	"fmt"
	"sync"
	"time"
)

// SupportCase represents an AWS Support case.
type SupportCase struct {
	CaseID               string
	DisplayID            string
	Subject              string
	Status               string
	ServiceCode          string
	SeverityCode         string
	CategoryCode         string
	CommunicationBody    string
	SubmittedBy          string
	TimeCreated          time.Time
	RecentCommunications []*Communication
	Language             string
	CCEmailAddresses     []string
}

// Communication represents a communication on a support case.
type Communication struct {
	CaseID       string
	Body         string
	SubmittedBy  string
	TimeCreated  time.Time
	AttachmentSet []string
}

// TrustedAdvisorCheck represents a Trusted Advisor check.
type TrustedAdvisorCheck struct {
	ID          string
	Name        string
	Description string
	Category    string
	Metadata    []string
}

// TrustedAdvisorCheckResult represents the result of a Trusted Advisor check.
type TrustedAdvisorCheckResult struct {
	CheckID       string
	Status        string
	Timestamp     string
	ResourcesSummary map[string]int64
	FlaggedResources []map[string]string
}

// ServiceInfo represents a support service.
type ServiceInfo struct {
	Code       string
	Name       string
	Categories []CategoryInfo
}

// CategoryInfo represents a support service category.
type CategoryInfo struct {
	Code string
	Name string
}

// SeverityLevel represents a support severity level.
type SeverityLevel struct {
	Code string
	Name string
}

// Store manages Support resources in memory.
type Store struct {
	mu        sync.RWMutex
	cases     map[string]*SupportCase
	checks    map[string]*TrustedAdvisorCheck
	accountID string
	region    string
	caseSeq   int
}

// NewStore returns a new empty Support Store.
func NewStore(accountID, region string) *Store {
	s := &Store{
		cases:     make(map[string]*SupportCase),
		checks:    make(map[string]*TrustedAdvisorCheck),
		accountID: accountID,
		region:    region,
	}
	s.initTrustedAdvisorChecks()
	return s
}

func (s *Store) initTrustedAdvisorChecks() {
	checks := []TrustedAdvisorCheck{
		{ID: "Pfx0RwqBli", Name: "Security Groups - Specific Ports Unrestricted", Description: "Checks security groups for rules that allow unrestricted access to specific ports.", Category: "security", Metadata: []string{"Region", "Security Group Name", "Security Group ID", "Protocol", "Port", "Status", "IP Address"}},
		{ID: "HCP4007jGY", Name: "S3 Bucket Permissions", Description: "Checks buckets that have open access permissions.", Category: "security", Metadata: []string{"Region", "Bucket Name", "ACL Allows List", "ACL Allows Upload/Delete", "Status"}},
		{ID: "1iG5NDGVre", Name: "IAM Password Policy", Description: "Checks the password policy for your account.", Category: "security", Metadata: []string{"Password Policy", "Status"}},
		{ID: "DAvU7jPFbW", Name: "Low Utilization Amazon EC2 Instances", Description: "Checks the Amazon EC2 instances that were running at any time during the last 14 days.", Category: "cost_optimizing", Metadata: []string{"Region", "Instance ID", "Instance Name", "Instance Type", "Estimated Monthly Savings", "Day 1-14 CPU Utilization"}},
		{ID: "Ti39halfu8", Name: "Idle Load Balancers", Description: "Checks for load balancers that have no active backend instances.", Category: "cost_optimizing", Metadata: []string{"Region", "Load Balancer Name", "Reason", "Estimated Monthly Savings"}},
		{ID: "hjLMh88uM8", Name: "Amazon EBS Over-Provisioned Volumes", Description: "Checks for volumes that appear over-provisioned.", Category: "cost_optimizing", Metadata: []string{"Region", "Volume ID", "Volume Name", "Volume Type", "Volume Size", "Monthly Storage Cost"}},
		{ID: "ZRxQlPsb6c", Name: "Service Limits", Description: "Checks for usage that is more than 80% of the service limit.", Category: "performance", Metadata: []string{"Region", "Service", "Limit Name", "Limit Amount", "Current Usage", "Status"}},
		{ID: "R365s2Qddf", Name: "Amazon RDS Multi-AZ", Description: "Checks for DB instances not deployed in Multi-AZ configuration.", Category: "fault_tolerance", Metadata: []string{"Region", "DB Instance", "VPC ID", "Multi-AZ", "Status"}},
	}
	for i := range checks {
		s.checks[checks[i].ID] = &checks[i]
	}
}

// CreateCase creates a new support case.
func (s *Store) CreateCase(subject, serviceCode, severityCode, categoryCode, body, submittedBy, language string, ccEmails []string) (*SupportCase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.caseSeq++
	caseID := fmt.Sprintf("case-%012d", s.caseSeq)
	displayID := fmt.Sprintf("%d", 1000000000+s.caseSeq)

	sc := &SupportCase{
		CaseID:            caseID,
		DisplayID:         displayID,
		Subject:           subject,
		Status:            "opened",
		ServiceCode:       serviceCode,
		SeverityCode:      severityCode,
		CategoryCode:      categoryCode,
		CommunicationBody: body,
		SubmittedBy:       submittedBy,
		TimeCreated:       time.Now().UTC(),
		Language:          language,
		CCEmailAddresses:  ccEmails,
	}

	if body != "" {
		sc.RecentCommunications = []*Communication{
			{CaseID: caseID, Body: body, SubmittedBy: submittedBy, TimeCreated: time.Now().UTC()},
		}
	}

	s.cases[caseID] = sc
	return sc, nil
}

// GetCase retrieves a case by ID.
func (s *Store) GetCase(caseID string) (*SupportCase, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sc, ok := s.cases[caseID]
	return sc, ok
}

// ListCases returns all cases, optionally filtered.
func (s *Store) ListCases(includeResolved bool) []*SupportCase {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*SupportCase, 0, len(s.cases))
	for _, sc := range s.cases {
		if !includeResolved && sc.Status == "resolved" {
			continue
		}
		out = append(out, sc)
	}
	return out
}

// ResolveCase resolves a case.
func (s *Store) ResolveCase(caseID string) (*SupportCase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc, ok := s.cases[caseID]
	if !ok {
		return nil, fmt.Errorf("case not found: %s", caseID)
	}
	sc.Status = "resolved"
	return sc, nil
}

// AddCommunication adds a communication to a case.
func (s *Store) AddCommunication(caseID, body, submittedBy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc, ok := s.cases[caseID]
	if !ok {
		return fmt.Errorf("case not found: %s", caseID)
	}
	comm := &Communication{
		CaseID:      caseID,
		Body:        body,
		SubmittedBy: submittedBy,
		TimeCreated: time.Now().UTC(),
	}
	sc.RecentCommunications = append(sc.RecentCommunications, comm)
	return nil
}

// GetCommunications returns all communications for a case.
func (s *Store) GetCommunications(caseID string) []*Communication {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sc, ok := s.cases[caseID]
	if !ok {
		return nil
	}
	return sc.RecentCommunications
}

// ListTrustedAdvisorChecks returns all Trusted Advisor checks.
func (s *Store) ListTrustedAdvisorChecks() []*TrustedAdvisorCheck {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*TrustedAdvisorCheck, 0, len(s.checks))
	for _, c := range s.checks {
		out = append(out, c)
	}
	return out
}

// GetTrustedAdvisorCheckResult returns the result for a check.
func (s *Store) GetTrustedAdvisorCheckResult(checkID string) (*TrustedAdvisorCheckResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	check, ok := s.checks[checkID]
	if !ok {
		return nil, false
	}

	result := &TrustedAdvisorCheckResult{
		CheckID:   checkID,
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		ResourcesSummary: map[string]int64{
			"resourcesProcessed": 25,
			"resourcesFlagged":   0,
			"resourcesIgnored":   0,
			"resourcesSuppressed": 0,
		},
	}

	// Add some mock flagged resources for cost checks
	if check.Category == "cost_optimizing" {
		result.Status = "warning"
		result.ResourcesSummary["resourcesFlagged"] = 2
		result.FlaggedResources = []map[string]string{
			{"status": "warning", "region": "us-east-1", "resourceId": "i-0123456789abcdef0"},
			{"status": "warning", "region": "us-west-2", "resourceId": "i-0fedcba9876543210"},
		}
	}

	return result, true
}

// GetServices returns the list of support services.
func (s *Store) GetServices() []ServiceInfo {
	return []ServiceInfo{
		{Code: "amazon-ec2", Name: "Amazon Elastic Compute Cloud", Categories: []CategoryInfo{
			{Code: "general-guidance", Name: "General Guidance"},
			{Code: "instance-issue", Name: "Instance Issue"},
		}},
		{Code: "amazon-s3", Name: "Amazon Simple Storage Service", Categories: []CategoryInfo{
			{Code: "general-guidance", Name: "General Guidance"},
		}},
		{Code: "amazon-rds", Name: "Amazon Relational Database Service", Categories: []CategoryInfo{
			{Code: "general-guidance", Name: "General Guidance"},
			{Code: "performance", Name: "Performance"},
		}},
		{Code: "aws-lambda", Name: "AWS Lambda", Categories: []CategoryInfo{
			{Code: "general-guidance", Name: "General Guidance"},
		}},
		{Code: "general-info", Name: "General Info and Getting Started", Categories: []CategoryInfo{
			{Code: "general-info", Name: "General Info"},
		}},
	}
}

// GetSeverityLevels returns the severity levels.
func (s *Store) GetSeverityLevels() []SeverityLevel {
	return []SeverityLevel{
		{Code: "low", Name: "Low"},
		{Code: "normal", Name: "Normal"},
		{Code: "high", Name: "High"},
		{Code: "urgent", Name: "Urgent"},
		{Code: "critical", Name: "Critical"},
	}
}
