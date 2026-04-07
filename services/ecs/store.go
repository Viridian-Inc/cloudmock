package ecs

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// newUUID returns a random UUID v4 string.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ContainerDefinition describes a single container within a task definition.
type ContainerDefinition struct {
	Name         string
	Image        string
	CPU          int
	Memory       int
	PortMappings []PortMapping
	Environment  []KeyValuePair
	Essential    bool
}

// PortMapping describes a container port mapping.
type PortMapping struct {
	ContainerPort int
	HostPort      int
	Protocol      string
}

// KeyValuePair is a name/value pair used for environment variables, tags, etc.
type KeyValuePair struct {
	Name  string
	Value string
}

// Cluster holds ECS cluster metadata.
type Cluster struct {
	Name                             string
	ARN                              string
	Status                           string
	Tags                             map[string]string
	RegisteredContainerInstancesCount int
	RunningTasksCount                int
	PendingTasksCount                int
}

// TaskDefinition holds a registered ECS task definition.
type TaskDefinition struct {
	ARN                  string
	Family               string
	Revision             int
	ContainerDefinitions []ContainerDefinition
	Status               string // ACTIVE | INACTIVE
	NetworkMode          string
	RequiresCompatibilities []string
	CPU                  string
	Memory               string
}

// Service holds an ECS service.
type Service struct {
	ARN            string
	Name           string
	ClusterARN     string
	TaskDefinition string
	DesiredCount   int
	RunningCount   int
	Status         string
	LaunchType     string
}

// Task holds an ECS task.
type Task struct {
	ARN              string
	TaskDefinitionARN string
	ClusterARN       string
	LastStatus       string
	DesiredStatus    string
	StartedAt        *time.Time
	StoppedAt        *time.Time
	StopCode         string
	StoppedReason    string
}

// Store is the in-memory store for all ECS resources.
type Store struct {
	mu              sync.RWMutex
	accountID       string
	region          string
	clusters        map[string]*Cluster // keyed by name
	taskDefinitions map[string][]*TaskDefinition // keyed by family; slice indexed by revision-1
	services        map[string]map[string]*Service // clusterARN -> serviceName -> Service
	tasks           map[string]map[string]*Task    // clusterARN -> taskARN -> Task
	tags            map[string]map[string]string   // resourceARN -> tags
}

// NewStore creates an empty ECS Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID:       accountID,
		region:          region,
		clusters:        make(map[string]*Cluster),
		taskDefinitions: make(map[string][]*TaskDefinition),
		services:        make(map[string]map[string]*Service),
		tasks:           make(map[string]map[string]*Task),
		tags:            make(map[string]map[string]string),
	}
}

// ---- ARN builders ----

func (s *Store) clusterARN(name string) string {
	return fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", s.region, s.accountID, name)
}

func (s *Store) taskDefARN(family string, revision int) string {
	return fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/%s:%d", s.region, s.accountID, family, revision)
}

func (s *Store) serviceARN(clusterName, serviceName string) string {
	return fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", s.region, s.accountID, clusterName, serviceName)
}

func (s *Store) taskARN(clusterName, taskID string) string {
	return fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%s", s.region, s.accountID, clusterName, taskID)
}

// ---- cluster helpers ----

// resolveClusterName resolves a cluster name or ARN to a cluster name.
func (s *Store) resolveClusterName(nameOrARN string) string {
	// If it's an ARN, extract the name from the end.
	if len(nameOrARN) > 0 && nameOrARN[0] != 'a' {
		return nameOrARN
	}
	// Simple heuristic: ARN format ends with /name
	for i := len(nameOrARN) - 1; i >= 0; i-- {
		if nameOrARN[i] == '/' {
			return nameOrARN[i+1:]
		}
	}
	return nameOrARN
}

// ---- Cluster operations ----

// CreateCluster creates a new cluster.
func (s *Store) CreateCluster(name string, tags map[string]string) (*Cluster, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name == "" {
		name = "default"
	}
	if tags == nil {
		tags = make(map[string]string)
	}

	c := &Cluster{
		Name:   name,
		ARN:    s.clusterARN(name),
		Status: "ACTIVE",
		Tags:   tags,
	}
	s.clusters[name] = c
	return c, nil
}

// DeleteCluster deletes a cluster by name or ARN.
func (s *Store) DeleteCluster(nameOrARN string) (*Cluster, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := s.resolveClusterName(nameOrARN)
	c, ok := s.clusters[name]
	if !ok {
		return nil, service.NewAWSError("ClusterNotFoundException",
			fmt.Sprintf("The specified cluster was not found: %s", nameOrARN),
			http.StatusBadRequest)
	}
	delete(s.clusters, name)
	return c, nil
}

// DescribeClusters returns clusters by names/ARNs. Empty slice returns all.
func (s *Store) DescribeClusters(namesOrARNs []string) ([]*Cluster, []clusterFailure) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(namesOrARNs) == 0 {
		out := make([]*Cluster, 0, len(s.clusters))
		for _, c := range s.clusters {
			out = append(out, c)
		}
		return out, nil
	}

	out := make([]*Cluster, 0, len(namesOrARNs))
	var failures []clusterFailure
	for _, ref := range namesOrARNs {
		name := s.resolveClusterName(ref)
		c, ok := s.clusters[name]
		if !ok {
			failures = append(failures, clusterFailure{
				ARN:    ref,
				Reason: fmt.Sprintf("MISSING: The specified cluster was not found: %s", ref),
			})
		} else {
			out = append(out, c)
		}
	}
	return out, failures
}

// ListClusters returns all cluster ARNs.
func (s *Store) ListClusters() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	arns := make([]string, 0, len(s.clusters))
	for _, c := range s.clusters {
		arns = append(arns, c.ARN)
	}
	return arns
}

// clusterFailure describes a cluster lookup failure.
type clusterFailure struct {
	ARN    string
	Reason string
}

// ---- Task Definition operations ----

// RegisterTaskDefinition registers a new task definition revision.
func (s *Store) RegisterTaskDefinition(
	family string,
	containerDefs []ContainerDefinition,
	networkMode string,
	requiresCompatibilities []string,
	cpu, memory string,
) (*TaskDefinition, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if family == "" {
		return nil, service.NewAWSError("InvalidParameterException",
			"family is required.", http.StatusBadRequest)
	}

	revision := len(s.taskDefinitions[family]) + 1
	td := &TaskDefinition{
		ARN:                  s.taskDefARN(family, revision),
		Family:               family,
		Revision:             revision,
		ContainerDefinitions: containerDefs,
		Status:               "ACTIVE",
		NetworkMode:          networkMode,
		RequiresCompatibilities: requiresCompatibilities,
		CPU:                  cpu,
		Memory:               memory,
	}
	s.taskDefinitions[family] = append(s.taskDefinitions[family], td)
	return td, nil
}

// DeregisterTaskDefinition marks a task definition revision as INACTIVE.
func (s *Store) DeregisterTaskDefinition(familyRevision string) (*TaskDefinition, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	td, awsErr := s.resolveTaskDefLocked(familyRevision)
	if awsErr != nil {
		return nil, awsErr
	}
	td.Status = "INACTIVE"
	return td, nil
}

// DescribeTaskDefinition retrieves a task definition by family:revision or ARN.
func (s *Store) DescribeTaskDefinition(familyRevision string) (*TaskDefinition, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.resolveTaskDefLocked(familyRevision)
}

// ListTaskDefinitions returns ARNs of task definitions, optionally filtered by family prefix.
func (s *Store) ListTaskDefinitions(familyPrefix string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var arns []string
	for family, revisions := range s.taskDefinitions {
		if familyPrefix != "" && len(family) < len(familyPrefix) {
			continue
		}
		if familyPrefix != "" && family[:len(familyPrefix)] != familyPrefix {
			continue
		}
		for _, td := range revisions {
			if td.Status == "ACTIVE" {
				arns = append(arns, td.ARN)
			}
		}
	}
	return arns
}

// resolveTaskDefLocked resolves a family:revision string or ARN to a TaskDefinition (caller holds lock).
func (s *Store) resolveTaskDefLocked(ref string) (*TaskDefinition, *service.AWSError) {
	// Try to find by ARN first.
	for _, revs := range s.taskDefinitions {
		for _, td := range revs {
			if td.ARN == ref {
				return td, nil
			}
		}
	}

	// Parse family:revision.
	family, revision, err := parseTaskDefRef(ref)
	if err != nil {
		// Try as family name only (latest active).
		if revs, ok := s.taskDefinitions[ref]; ok {
			for i := len(revs) - 1; i >= 0; i-- {
				if revs[i].Status == "ACTIVE" {
					return revs[i], nil
				}
			}
		}
		return nil, service.NewAWSError("ClientException",
			fmt.Sprintf("Invalid task definition ARN or family:revision: %s", ref),
			http.StatusBadRequest)
	}

	revs, ok := s.taskDefinitions[family]
	if !ok || revision < 1 || revision > len(revs) {
		return nil, service.NewAWSError("ClientException",
			fmt.Sprintf("The specified task definition does not exist: %s", ref),
			http.StatusBadRequest)
	}
	return revs[revision-1], nil
}

// parseTaskDefRef parses "family:revision" into components.
func parseTaskDefRef(ref string) (family string, revision int, err error) {
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == ':' {
			family = ref[:i]
			_, parseErr := fmt.Sscanf(ref[i+1:], "%d", &revision)
			if parseErr != nil {
				return "", 0, parseErr
			}
			return family, revision, nil
		}
	}
	return "", 0, fmt.Errorf("no colon found in %q", ref)
}

// ---- Service operations ----

// CreateService creates a new ECS service in the given cluster.
func (s *Store) CreateService(clusterNameOrARN, serviceName, taskDef string, desiredCount int, launchType string) (*Service, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterName := s.resolveClusterName(clusterNameOrARN)
	if clusterNameOrARN == "" {
		clusterName = "default"
	}
	cluster, ok := s.clusters[clusterName]
	if !ok {
		return nil, service.NewAWSError("ClusterNotFoundException",
			fmt.Sprintf("The specified cluster was not found: %s", clusterNameOrARN),
			http.StatusBadRequest)
	}

	if s.services[cluster.ARN] == nil {
		s.services[cluster.ARN] = make(map[string]*Service)
	}
	if _, exists := s.services[cluster.ARN][serviceName]; exists {
		return nil, service.NewAWSError("InvalidParameterException",
			fmt.Sprintf("Creation of service was not idempotent: %s", serviceName),
			http.StatusBadRequest)
	}

	if launchType == "" {
		launchType = "EC2"
	}

	svc := &Service{
		ARN:            s.serviceARN(clusterName, serviceName),
		Name:           serviceName,
		ClusterARN:     cluster.ARN,
		TaskDefinition: taskDef,
		DesiredCount:   desiredCount,
		RunningCount:   0,
		Status:         "ACTIVE",
		LaunchType:     launchType,
	}
	s.services[cluster.ARN][serviceName] = svc
	return svc, nil
}

// DeleteService deletes an ECS service.
func (s *Store) DeleteService(clusterNameOrARN, serviceNameOrARN string, force bool) (*Service, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterName := s.resolveClusterName(clusterNameOrARN)
	if clusterNameOrARN == "" {
		clusterName = "default"
	}
	cluster, ok := s.clusters[clusterName]
	if !ok {
		return nil, service.NewAWSError("ClusterNotFoundException",
			fmt.Sprintf("The specified cluster was not found: %s", clusterNameOrARN),
			http.StatusBadRequest)
	}

	svcMap := s.services[cluster.ARN]
	if svcMap == nil {
		return nil, service.NewAWSError("ServiceNotFoundException",
			fmt.Sprintf("Service not found: %s", serviceNameOrARN),
			http.StatusBadRequest)
	}

	// Try by name first, then ARN.
	svc := svcMap[serviceNameOrARN]
	if svc == nil {
		for _, sv := range svcMap {
			if sv.ARN == serviceNameOrARN {
				svc = sv
				break
			}
		}
	}
	if svc == nil {
		return nil, service.NewAWSError("ServiceNotFoundException",
			fmt.Sprintf("Service not found: %s", serviceNameOrARN),
			http.StatusBadRequest)
	}

	svc.Status = "INACTIVE"
	svc.DesiredCount = 0
	delete(svcMap, svc.Name)
	return svc, nil
}

// DescribeServices returns ECS services by names/ARNs in a cluster.
func (s *Store) DescribeServices(clusterNameOrARN string, serviceNames []string) ([]*Service, []serviceFailure) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clusterName := s.resolveClusterName(clusterNameOrARN)
	if clusterNameOrARN == "" {
		clusterName = "default"
	}
	cluster, ok := s.clusters[clusterName]
	if !ok {
		var failures []serviceFailure
		for _, n := range serviceNames {
			failures = append(failures, serviceFailure{ARN: n, Reason: "MISSING"})
		}
		return nil, failures
	}

	svcMap := s.services[cluster.ARN]
	var out []*Service
	var failures []serviceFailure
	for _, ref := range serviceNames {
		svc := svcMap[ref]
		if svc == nil && svcMap != nil {
			for _, sv := range svcMap {
				if sv.ARN == ref {
					svc = sv
					break
				}
			}
		}
		if svc == nil {
			failures = append(failures, serviceFailure{ARN: ref, Reason: "MISSING"})
		} else {
			out = append(out, svc)
		}
	}
	return out, failures
}

// ListServices returns service ARNs for a cluster.
func (s *Store) ListServices(clusterNameOrARN string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clusterName := s.resolveClusterName(clusterNameOrARN)
	if clusterNameOrARN == "" {
		clusterName = "default"
	}
	cluster, ok := s.clusters[clusterName]
	if !ok {
		return nil
	}

	svcMap := s.services[cluster.ARN]
	arns := make([]string, 0, len(svcMap))
	for _, sv := range svcMap {
		arns = append(arns, sv.ARN)
	}
	return arns
}

// UpdateService updates a service's desired count and/or task definition.
func (s *Store) UpdateService(clusterNameOrARN, serviceNameOrARN string, desiredCount *int, taskDef string) (*Service, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterName := s.resolveClusterName(clusterNameOrARN)
	if clusterNameOrARN == "" {
		clusterName = "default"
	}
	cluster, ok := s.clusters[clusterName]
	if !ok {
		return nil, service.NewAWSError("ClusterNotFoundException",
			fmt.Sprintf("The specified cluster was not found: %s", clusterNameOrARN),
			http.StatusBadRequest)
	}

	svcMap := s.services[cluster.ARN]
	var svc *Service
	if svcMap != nil {
		svc = svcMap[serviceNameOrARN]
		if svc == nil {
			for _, sv := range svcMap {
				if sv.ARN == serviceNameOrARN {
					svc = sv
					break
				}
			}
		}
	}
	if svc == nil {
		return nil, service.NewAWSError("ServiceNotFoundException",
			fmt.Sprintf("Service not found: %s", serviceNameOrARN),
			http.StatusBadRequest)
	}

	if desiredCount != nil {
		svc.DesiredCount = *desiredCount
	}
	if taskDef != "" {
		svc.TaskDefinition = taskDef
	}
	return svc, nil
}

// serviceFailure describes a service lookup failure.
type serviceFailure struct {
	ARN    string
	Reason string
}

// ---- Task operations ----

// RunTask creates task objects for a cluster/task-definition (no actual execution).
func (s *Store) RunTask(clusterNameOrARN, taskDef string, count int) ([]*Task, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterName := s.resolveClusterName(clusterNameOrARN)
	if clusterNameOrARN == "" {
		clusterName = "default"
	}
	cluster, ok := s.clusters[clusterName]
	if !ok {
		return nil, service.NewAWSError("ClusterNotFoundException",
			fmt.Sprintf("The specified cluster was not found: %s", clusterNameOrARN),
			http.StatusBadRequest)
	}

	// Resolve task definition ARN.
	tdARN := taskDef
	if revs, ok2 := s.taskDefinitions[taskDef]; ok2 {
		// family name provided — use latest active.
		for i := len(revs) - 1; i >= 0; i-- {
			if revs[i].Status == "ACTIVE" {
				tdARN = revs[i].ARN
				break
			}
		}
	} else {
		// Try family:revision parse.
		td, err := s.resolveTaskDefLocked(taskDef)
		if err == nil {
			tdARN = td.ARN
		}
	}

	if s.tasks[cluster.ARN] == nil {
		s.tasks[cluster.ARN] = make(map[string]*Task)
	}

	now := time.Now().UTC()
	tasks := make([]*Task, 0, count)
	for i := 0; i < count; i++ {
		taskID := newUUID()
		t := &Task{
			ARN:               s.taskARN(clusterName, taskID),
			TaskDefinitionARN: tdARN,
			ClusterARN:        cluster.ARN,
			LastStatus:        "RUNNING",
			DesiredStatus:     "RUNNING",
			StartedAt:         &now,
		}
		s.tasks[cluster.ARN][t.ARN] = t
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// StopTask marks a task as STOPPED.
func (s *Store) StopTask(clusterNameOrARN, taskARN, reason string) (*Task, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clusterName := s.resolveClusterName(clusterNameOrARN)
	if clusterNameOrARN == "" {
		clusterName = "default"
	}
	cluster, ok := s.clusters[clusterName]
	if !ok {
		return nil, service.NewAWSError("ClusterNotFoundException",
			fmt.Sprintf("The specified cluster was not found: %s", clusterNameOrARN),
			http.StatusBadRequest)
	}

	taskMap := s.tasks[cluster.ARN]
	if taskMap == nil {
		return nil, service.NewAWSError("InvalidParameterException",
			fmt.Sprintf("Task not found: %s", taskARN),
			http.StatusBadRequest)
	}

	// Try by ARN directly, or as partial ID.
	t := taskMap[taskARN]
	if t == nil {
		for _, tk := range taskMap {
			if tk.ARN == taskARN {
				t = tk
				break
			}
		}
	}
	if t == nil {
		return nil, service.NewAWSError("InvalidParameterException",
			fmt.Sprintf("Task not found: %s", taskARN),
			http.StatusBadRequest)
	}

	now := time.Now().UTC()
	t.LastStatus = "STOPPED"
	t.DesiredStatus = "STOPPED"
	t.StoppedAt = &now
	t.StopCode = "TaskFailedToStart"
	if reason != "" {
		t.StoppedReason = reason
	} else {
		t.StoppedReason = "Task stopped by user"
	}
	return t, nil
}

// DescribeTasks returns tasks by ARNs.
func (s *Store) DescribeTasks(clusterNameOrARN string, taskARNs []string) ([]*Task, []taskFailure) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clusterName := s.resolveClusterName(clusterNameOrARN)
	if clusterNameOrARN == "" {
		clusterName = "default"
	}
	cluster, ok := s.clusters[clusterName]
	if !ok {
		var failures []taskFailure
		for _, a := range taskARNs {
			failures = append(failures, taskFailure{ARN: a, Reason: "MISSING"})
		}
		return nil, failures
	}

	taskMap := s.tasks[cluster.ARN]
	var out []*Task
	var failures []taskFailure
	for _, ref := range taskARNs {
		t := taskMap[ref]
		if t == nil && taskMap != nil {
			for _, tk := range taskMap {
				if tk.ARN == ref {
					t = tk
					break
				}
			}
		}
		if t == nil {
			failures = append(failures, taskFailure{ARN: ref, Reason: "MISSING"})
		} else {
			out = append(out, t)
		}
	}
	return out, failures
}

// ListTasks returns task ARNs for a cluster, optionally filtered by service name.
func (s *Store) ListTasks(clusterNameOrARN, serviceName string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clusterName := s.resolveClusterName(clusterNameOrARN)
	if clusterNameOrARN == "" {
		clusterName = "default"
	}
	cluster, ok := s.clusters[clusterName]
	if !ok {
		return nil
	}

	taskMap := s.tasks[cluster.ARN]
	arns := make([]string, 0, len(taskMap))
	for _, t := range taskMap {
		arns = append(arns, t.ARN)
	}
	return arns
}

// taskFailure describes a task lookup failure.
type taskFailure struct {
	ARN    string
	Reason string
}

// ---- Tag operations ----

// TagResource adds/replaces tags on any ECS resource by ARN.
func (s *Store) TagResource(resourceARN string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tags[resourceARN] == nil {
		s.tags[resourceARN] = make(map[string]string)
	}
	for k, v := range tags {
		s.tags[resourceARN][k] = v
	}
	return nil
}

// UntagResource removes tags from a resource by ARN.
func (s *Store) UntagResource(resourceARN string, tagKeys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()

	m := s.tags[resourceARN]
	if m == nil {
		return nil
	}
	for _, k := range tagKeys {
		delete(m, k)
	}
	return nil
}

// ListTagsForResource returns tags for a resource by ARN.
func (s *Store) ListTagsForResource(resourceARN string) (map[string]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m := s.tags[resourceARN]
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out, nil
}
