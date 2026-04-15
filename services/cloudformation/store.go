package cloudformation

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ---- data types ----

// Parameter holds a single CloudFormation stack parameter.
type Parameter struct {
	ParameterKey   string
	ParameterValue string
}

// Tag holds a single resource tag.
type Tag struct {
	Key   string
	Value string
}

// Output holds a single CloudFormation stack output.
type Output struct {
	OutputKey   string
	OutputValue string
	Description string
	ExportName  string
}

// StackResource represents a resource parsed from the template.
type StackResource struct {
	LogicalResourceId  string
	PhysicalResourceId string
	ResourceType       string
	ResourceStatus     string
	Timestamp          time.Time
}

// StackEvent represents a single lifecycle event for a stack.
type StackEvent struct {
	EventId           string
	StackId           string
	StackName         string
	LogicalResourceId string
	ResourceType      string
	ResourceStatus    string
	Timestamp         time.Time
}

// ChangeSet holds metadata for a CloudFormation change set.
type ChangeSet struct {
	ChangeSetId     string
	ChangeSetName   string
	StackId         string
	StackName       string
	Status          string
	ExecutionStatus string
	Description     string
	CreationTime    time.Time
}

// Stack holds the full state of a CloudFormation stack.
type Stack struct {
	StackId      string
	StackName    string
	TemplateBody string
	Parameters   []Parameter
	Tags         []Tag
	Outputs      []Output
	StackStatus  string
	CreationTime time.Time
	Description  string
	Resources    []StackResource
	Events       []StackEvent
	ChangeSets   map[string]*ChangeSet // keyed by ChangeSetName
}

// ---- raw template parsing types ----

type cfnTemplate struct {
	Description string                     `json:"Description"`
	Parameters  map[string]cfnParameter    `json:"Parameters"`
	Resources   map[string]cfnResource     `json:"Resources"`
	Outputs     map[string]cfnOutput       `json:"Outputs"`
}

type cfnParameter struct {
	Type    string          `json:"Type"`
	Default json.RawMessage `json:"Default"`
}

type cfnResource struct {
	Type string `json:"Type"`
}

type cfnOutput struct {
	Value       any `json:"Value"`
	Description string      `json:"Description"`
	Export      *cfnExport  `json:"Export"`
}

type cfnExport struct {
	Name any `json:"Name"`
}

// ---- store ----

// StackStore manages all CloudFormation stacks in memory.
type StackStore struct {
	mu          sync.RWMutex
	stacks      map[string]*Stack // keyed by StackName
	accountID   string
	region      string
	provisioner *Provisioner
}

// NewStore creates a new StackStore.
func NewStore(accountID, region string) *StackStore {
	return &StackStore{
		stacks:    make(map[string]*Stack),
		accountID: accountID,
		region:    region,
	}
}

// SetProvisioner sets the resource provisioner for this store.
// When set, CreateStack will create real resources via cloudmock services.
func (s *StackStore) SetProvisioner(p *Provisioner) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.provisioner = p
}

// stackARN builds a CloudFormation stack ARN.
func (s *StackStore) stackARN(name, uid string) string {
	return fmt.Sprintf("arn:aws:cloudformation:%s:%s:stack/%s/%s", s.region, s.accountID, name, uid)
}

// CreateStack creates a new stack by parsing the template and recording metadata.
// If a provisioner is set, it also creates real resources in cloudmock services.
// Returns the new Stack or an error.
func (s *StackStore) CreateStack(name, templateBody string, params []Parameter, tags []Tag) (*Stack, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, exists := s.stacks[name]; exists && existing.StackStatus != "DELETE_COMPLETE" {
		return nil, fmt.Errorf("AlreadyExistsException: Stack [%s] already exists", name)
	}

	uid := newUUID()
	arn := s.stackARN(name, uid)
	now := time.Now().UTC()

	// Parse template.
	description, resources, outputs := parseTemplate(templateBody, params)

	// If a provisioner is set, create real resources.
	if s.provisioner != nil {
		paramMap := make(map[string]string, len(params))
		for _, p := range params {
			paramMap[p.ParameterKey] = p.ParameterValue
		}
		// Add stack name to pseudo-parameters.
		paramMap["AWS::StackName"] = name

		provisioned, err := s.provisioner.ProvisionStack(templateBody, paramMap)
		if err == nil && len(provisioned) > 0 {
			// Build a lookup from logical ID to provisioned resource.
			provMap := make(map[string]*ProvisionedResource, len(provisioned))
			for i := range provisioned {
				provMap[provisioned[i].LogicalId] = &provisioned[i]
			}

			// Update resources with physical IDs.
			for i, r := range resources {
				if pr, ok := provMap[r.LogicalResourceId]; ok {
					resources[i].PhysicalResourceId = pr.PhysicalId
					resources[i].ResourceStatus = pr.Status
				}
			}
		}
	}

	// Build creation events.
	events := []StackEvent{
		{
			EventId:           newUUID(),
			StackId:           arn,
			StackName:         name,
			LogicalResourceId: name,
			ResourceType:      "AWS::CloudFormation::Stack",
			ResourceStatus:    "CREATE_COMPLETE",
			Timestamp:         now,
		},
	}
	for _, r := range resources {
		events = append(events, StackEvent{
			EventId:           newUUID(),
			StackId:           arn,
			StackName:         name,
			LogicalResourceId: r.LogicalResourceId,
			ResourceType:      r.ResourceType,
			ResourceStatus:    r.ResourceStatus,
			Timestamp:         now,
		})
	}

	stack := &Stack{
		StackId:      arn,
		StackName:    name,
		TemplateBody: templateBody,
		Parameters:   params,
		Tags:         tags,
		Outputs:      outputs,
		StackStatus:  "CREATE_COMPLETE",
		CreationTime: now,
		Description:  description,
		Resources:    resources,
		Events:       events,
		ChangeSets:   make(map[string]*ChangeSet),
	}

	s.stacks[name] = stack
	return stack, nil
}

// DeleteStack marks a stack as DELETE_COMPLETE and deletes provisioned resources.
// Returns false if not found.
func (s *StackStore) DeleteStack(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	stack, ok := s.stacks[name]
	if !ok {
		return false
	}

	// Delete provisioned resources in reverse order.
	if s.provisioner != nil {
		var provisioned []ProvisionedResource
		for _, r := range stack.Resources {
			if r.PhysicalResourceId != "" {
				provisioned = append(provisioned, ProvisionedResource{
					LogicalId:  r.LogicalResourceId,
					PhysicalId: r.PhysicalResourceId,
					Type:       r.ResourceType,
					Status:     r.ResourceStatus,
				})
			}
		}
		if len(provisioned) > 0 {
			s.provisioner.DeleteResources(provisioned)
		}
	}

	stack.StackStatus = "DELETE_COMPLETE"
	return true
}

// GetStack returns a stack by name.
func (s *StackStore) GetStack(name string) (*Stack, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.stacks[name]
	return st, ok
}

// ListStacks returns all stacks, optionally filtered by status.
func (s *StackStore) ListStacks(statusFilters []string) []*Stack {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filterSet := make(map[string]bool, len(statusFilters))
	for _, f := range statusFilters {
		filterSet[f] = true
	}

	result := make([]*Stack, 0, len(s.stacks))
	for _, st := range s.stacks {
		if len(filterSet) == 0 || filterSet[st.StackStatus] {
			result = append(result, st)
		}
	}
	return result
}

// AllStacks returns all stacks (including DELETE_COMPLETE ones).
func (s *StackStore) AllStacks() []*Stack {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Stack, 0, len(s.stacks))
	for _, st := range s.stacks {
		result = append(result, st)
	}
	return result
}

// CreateChangeSet adds a change set to a stack.
func (s *StackStore) CreateChangeSet(stackName, changeSetName, description string) (*ChangeSet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stack, ok := s.stacks[stackName]
	if !ok {
		return nil, fmt.Errorf("ValidationError: Stack '%s' does not exist", stackName)
	}

	uid := newUUID()
	csARN := fmt.Sprintf("arn:aws:cloudformation:%s:%s:changeSet/%s/%s", s.region, s.accountID, changeSetName, uid)
	cs := &ChangeSet{
		ChangeSetId:     csARN,
		ChangeSetName:   changeSetName,
		StackId:         stack.StackId,
		StackName:       stackName,
		Status:          "CREATE_COMPLETE",
		ExecutionStatus: "AVAILABLE",
		Description:     description,
		CreationTime:    time.Now().UTC(),
	}
	stack.ChangeSets[changeSetName] = cs
	return cs, nil
}

// GetChangeSet retrieves a change set from a stack.
func (s *StackStore) GetChangeSet(stackName, changeSetName string) (*ChangeSet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stack, ok := s.stacks[stackName]
	if !ok {
		return nil, false
	}
	cs, ok := stack.ChangeSets[changeSetName]
	return cs, ok
}

// ExecuteChangeSet marks a change set as EXECUTED.
func (s *StackStore) ExecuteChangeSet(stackName, changeSetName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	stack, ok := s.stacks[stackName]
	if !ok {
		return false
	}
	cs, ok := stack.ChangeSets[changeSetName]
	if !ok {
		return false
	}
	cs.ExecutionStatus = "EXECUTE_COMPLETE"
	cs.Status = "UPDATE_COMPLETE"
	return true
}

// DeleteChangeSet removes a change set from a stack.
func (s *StackStore) DeleteChangeSet(stackName, changeSetName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	stack, ok := s.stacks[stackName]
	if !ok {
		return false
	}
	if _, ok := stack.ChangeSets[changeSetName]; !ok {
		return false
	}
	delete(stack.ChangeSets, changeSetName)
	return true
}

// ListExports returns all Outputs that have an ExportName across all live stacks.
func (s *StackStore) ListExports() []exportEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var entries []exportEntry
	for _, st := range s.stacks {
		if st.StackStatus == "DELETE_COMPLETE" {
			continue
		}
		for _, o := range st.Outputs {
			if o.ExportName != "" {
				entries = append(entries, exportEntry{
					ExportingStackId: st.StackId,
					Name:             o.ExportName,
					Value:            o.OutputValue,
				})
			}
		}
	}
	return entries
}

type exportEntry struct {
	ExportingStackId string
	Name             string
	Value            string
}

// ---- template parsing ----

// parseTemplate attempts to parse a JSON CloudFormation template and extract
// description, resources, and outputs. YAML templates fall through gracefully.
func parseTemplate(templateBody string, providedParams []Parameter) (description string, resources []StackResource, outputs []Output) {
	var tmpl cfnTemplate
	if err := json.Unmarshal([]byte(templateBody), &tmpl); err != nil {
		// Not valid JSON — store as-is, return empty metadata.
		return "", nil, nil
	}

	description = tmpl.Description

	// Build a lookup map of provided parameter values.
	paramValues := make(map[string]string, len(providedParams))
	for _, p := range providedParams {
		paramValues[p.ParameterKey] = p.ParameterValue
	}

	// Fill in defaults for any template parameter not supplied.
	for key, defn := range tmpl.Parameters {
		if _, supplied := paramValues[key]; !supplied && defn.Default != nil {
			var defVal string
			// Default may be a string, number, etc. — unmarshal as string first.
			if err := json.Unmarshal(defn.Default, &defVal); err != nil {
				// Try as a raw JSON number/bool.
				defVal = string(defn.Default)
			}
			paramValues[key] = defVal
		}
	}

	// Extract resources.
	now := time.Now().UTC()
	for logicalID, res := range tmpl.Resources {
		resources = append(resources, StackResource{
			LogicalResourceId: logicalID,
			ResourceType:      res.Type,
			ResourceStatus:    "CREATE_COMPLETE",
			Timestamp:         now,
		})
	}

	// Extract outputs — store values as-is (no intrinsic function resolution).
	for key, out := range tmpl.Outputs {
		var valStr string
		if out.Value != nil {
			switch v := out.Value.(type) {
			case string:
				valStr = v
			default:
				// Could be a Ref or Fn::GetAtt map — marshal back to JSON string.
				b, _ := json.Marshal(v)
				valStr = string(b)
			}
		}

		var exportName string
		if out.Export != nil && out.Export.Name != nil {
			switch n := out.Export.Name.(type) {
			case string:
				exportName = n
			default:
				b, _ := json.Marshal(n)
				exportName = string(b)
			}
		}

		outputs = append(outputs, Output{
			OutputKey:   key,
			OutputValue: valStr,
			Description: out.Description,
			ExportName:  exportName,
		})
	}

	return description, resources, outputs
}

// ---- helpers ----

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
