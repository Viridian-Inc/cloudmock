package kafka

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/lifecycle"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

type ClusterState string

const (
	ClusterCreating     ClusterState = "CREATING"
	ClusterActive       ClusterState = "ACTIVE"
	ClusterUpdating     ClusterState = "UPDATING"
	ClusterDeleting     ClusterState = "DELETING"
	ClusterFailed       ClusterState = "FAILED"
	ClusterMaintenance  ClusterState = "MAINTENANCE"
	ClusterRebootingBroker ClusterState = "REBOOTING_BROKER"
)

type OperationState string

const (
	OpPending      OperationState = "PENDING"
	OpUpdateInProgress OperationState = "UPDATE_IN_PROGRESS"
	OpUpdateComplete   OperationState = "UPDATE_COMPLETE"
	OpUpdateFailed     OperationState = "UPDATE_FAILED"
)

type Cluster struct {
	ClusterName          string
	ClusterArn           string
	State                ClusterState
	ClusterType          string
	KafkaVersion         string
	NumberOfBrokerNodes  int
	BrokerNodeGroupInfo  map[string]any
	EncryptionInfo       map[string]any
	EnhancedMonitoring   string
	LoggingInfo          map[string]any
	CreationTime         time.Time
	Tags                 map[string]string
	Lifecycle            *lifecycle.Machine
}

type Configuration struct {
	Name             string
	Arn              string
	LatestRevision   ConfigurationRevision
	Revisions        []ConfigurationRevision
	Description      string
	KafkaVersions    []string
	State            string
	CreationTime     time.Time
	Tags             map[string]string
}

type ConfigurationRevision struct {
	Revision     int64
	Description  string
	CreationTime time.Time
	ServerProperties string
}

type ClusterOperation struct {
	OperationArn   string
	ClusterArn     string
	OperationType  string
	OperationState OperationState
	CreationTime   time.Time
	EndTime        *time.Time
}

type Store struct {
	mu             sync.RWMutex
	clusters       map[string]*Cluster          // keyed by ARN
	clustersByName map[string]*Cluster          // keyed by name
	configurations map[string]*Configuration    // keyed by ARN
	operations     map[string]*ClusterOperation // keyed by ARN
	tagsByArn      map[string]map[string]string
	accountID      string
	region         string
	lcConfig       *lifecycle.Config
}

func NewStore(accountID, region string) *Store {
	return &Store{
		clusters:       make(map[string]*Cluster),
		clustersByName: make(map[string]*Cluster),
		configurations: make(map[string]*Configuration),
		operations:     make(map[string]*ClusterOperation),
		tagsByArn:      make(map[string]map[string]string),
		accountID:      accountID,
		region:         region,
		lcConfig:       lifecycle.DefaultConfig(),
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) clusterARN(name string) string {
	return fmt.Sprintf("arn:aws:kafka:%s:%s:cluster/%s/%s", s.region, s.accountID, name, newUUID())
}

func (s *Store) configARN(name string) string {
	return fmt.Sprintf("arn:aws:kafka:%s:%s:configuration/%s/%s", s.region, s.accountID, name, newUUID())
}

func (s *Store) operationARN() string {
	return fmt.Sprintf("arn:aws:kafka:%s:%s:cluster-operation/%s", s.region, s.accountID, newUUID())
}

// Clusters.

func (s *Store) CreateCluster(name, kafkaVersion, clusterType string, numBrokers int, brokerNodeGroup, encryptionInfo, loggingInfo map[string]any, enhancedMonitoring string, tags map[string]string) (*Cluster, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.clustersByName[name]; exists {
		return nil, service.NewAWSError("ConflictException",
			fmt.Sprintf("Cluster %s already exists", name), http.StatusConflict)
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	if kafkaVersion == "" {
		kafkaVersion = "3.5.1"
	}
	if clusterType == "" {
		clusterType = "PROVISIONED"
	}
	if numBrokers <= 0 {
		numBrokers = 3
	}
	if enhancedMonitoring == "" {
		enhancedMonitoring = "DEFAULT"
	}

	arn := s.clusterARN(name)
	lc := lifecycle.NewMachine(
		lifecycle.State(ClusterCreating),
		[]lifecycle.Transition{
			{From: lifecycle.State(ClusterCreating), To: lifecycle.State(ClusterActive), Delay: 2 * time.Second},
		},
		s.lcConfig,
	)

	c := &Cluster{
		ClusterName:         name,
		ClusterArn:          arn,
		State:               ClusterState(lc.State()),
		ClusterType:         clusterType,
		KafkaVersion:        kafkaVersion,
		NumberOfBrokerNodes: numBrokers,
		BrokerNodeGroupInfo: brokerNodeGroup,
		EncryptionInfo:      encryptionInfo,
		EnhancedMonitoring:  enhancedMonitoring,
		LoggingInfo:         loggingInfo,
		CreationTime:        time.Now().UTC(),
		Tags:                tags,
		Lifecycle:           lc,
	}
	s.clusters[arn] = c
	s.clustersByName[name] = c
	s.tagsByArn[arn] = tags
	return c, nil
}

func (s *Store) DescribeCluster(arn string) (*Cluster, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.clusters[arn]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Cluster %s not found", arn), http.StatusNotFound)
	}
	c.State = ClusterState(c.Lifecycle.State())
	return c, nil
}

func (s *Store) ListClusters() []*Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Cluster, 0, len(s.clusters))
	for _, c := range s.clusters {
		c.State = ClusterState(c.Lifecycle.State())
		out = append(out, c)
	}
	return out
}

func (s *Store) DeleteCluster(arn string) *service.AWSError {
	s.mu.Lock()
	c, ok := s.clusters[arn]
	if !ok {
		s.mu.Unlock()
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("Cluster %s not found", arn), http.StatusNotFound)
	}
	lc := c.Lifecycle
	delete(s.clusters, arn)
	delete(s.clustersByName, c.ClusterName)
	delete(s.tagsByArn, arn)
	s.mu.Unlock()
	if lc != nil {
		lc.ForceState(lifecycle.State(ClusterDeleting))
	}
	return nil
}

func (s *Store) createOperation(clusterArn, opType string) *ClusterOperation {
	op := &ClusterOperation{
		OperationArn:   s.operationARN(),
		ClusterArn:     clusterArn,
		OperationType:  opType,
		OperationState: OpUpdateComplete,
		CreationTime:   time.Now().UTC(),
	}
	now := time.Now().UTC()
	op.EndTime = &now
	s.operations[op.OperationArn] = op
	return op
}

func (s *Store) UpdateBrokerCount(arn string, targetCount int) (*ClusterOperation, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clusters[arn]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Cluster %s not found", arn), http.StatusNotFound)
	}
	c.NumberOfBrokerNodes = targetCount
	op := s.createOperation(arn, "UPDATE_BROKER_COUNT")
	return op, nil
}

func (s *Store) UpdateBrokerStorage(arn string, targetStoragePerBroker int) (*ClusterOperation, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.clusters[arn]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Cluster %s not found", arn), http.StatusNotFound)
	}
	op := s.createOperation(arn, "UPDATE_BROKER_STORAGE")
	return op, nil
}

func (s *Store) UpdateClusterConfiguration(arn, configArn string, configRevision int64) (*ClusterOperation, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.clusters[arn]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Cluster %s not found", arn), http.StatusNotFound)
	}
	op := s.createOperation(arn, "UPDATE_CLUSTER_CONFIGURATION")
	return op, nil
}

func (s *Store) RebootBroker(arn string, brokerIds []string) (*ClusterOperation, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.clusters[arn]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Cluster %s not found", arn), http.StatusNotFound)
	}
	op := s.createOperation(arn, "REBOOT_BROKER")
	return op, nil
}

func (s *Store) GetBootstrapBrokers(arn string) (string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.clusters[arn]
	if !ok {
		return "", service.NewAWSError("NotFoundException",
			fmt.Sprintf("Cluster %s not found", arn), http.StatusNotFound)
	}
	brokers := ""
	for i := 1; i <= c.NumberOfBrokerNodes; i++ {
		if i > 1 {
			brokers += ","
		}
		brokers += fmt.Sprintf("b-%d.%s.kafka.%s.amazonaws.com:9092", i, c.ClusterName, s.region)
	}
	return brokers, nil
}

func (s *Store) ListNodes(arn string) ([]map[string]any, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.clusters[arn]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Cluster %s not found", arn), http.StatusNotFound)
	}
	nodes := make([]map[string]any, 0, c.NumberOfBrokerNodes)
	for i := 1; i <= c.NumberOfBrokerNodes; i++ {
		nodes = append(nodes, map[string]any{
			"nodeType": "BROKER",
			"nodeARN":  fmt.Sprintf("%s/broker/%d", arn, i),
			"nodeInfo": map[string]any{
				"brokerNodeInfo": map[string]any{
					"brokerId":        float64(i),
					"attachedENIId":   fmt.Sprintf("eni-%s", newUUID()[:12]),
					"clientSubnet":    "subnet-mock",
					"clientVpcIpAddress": fmt.Sprintf("10.0.%d.%d", i, i),
					"endpoints":      []string{fmt.Sprintf("b-%d.%s.kafka.%s.amazonaws.com", i, c.ClusterName, s.region)},
				},
			},
		})
	}
	return nodes, nil
}

// Configurations.

func (s *Store) CreateConfiguration(name, description, kafkaVersion, serverProperties string, tags map[string]string) (*Configuration, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tags == nil {
		tags = make(map[string]string)
	}
	arn := s.configARN(name)
	now := time.Now().UTC()
	cfg := &Configuration{
		Name:        name,
		Arn:         arn,
		Description: description,
		KafkaVersions: []string{kafkaVersion},
		State:       "ACTIVE",
		LatestRevision: ConfigurationRevision{
			Revision:         1,
			Description:      description,
			CreationTime:     now,
			ServerProperties: serverProperties,
		},
		Revisions: []ConfigurationRevision{
			{Revision: 1, Description: description, CreationTime: now, ServerProperties: serverProperties},
		},
		CreationTime: now,
		Tags:         tags,
	}
	s.configurations[arn] = cfg
	s.tagsByArn[arn] = tags
	return cfg, nil
}

func (s *Store) DescribeConfiguration(arn string) (*Configuration, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.configurations[arn]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Configuration %s not found", arn), http.StatusNotFound)
	}
	return cfg, nil
}

func (s *Store) ListConfigurations() []*Configuration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Configuration, 0, len(s.configurations))
	for _, cfg := range s.configurations {
		out = append(out, cfg)
	}
	return out
}

func (s *Store) UpdateConfiguration(arn, description, serverProperties string) (*Configuration, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, ok := s.configurations[arn]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Configuration %s not found", arn), http.StatusNotFound)
	}
	now := time.Now().UTC()
	newRev := ConfigurationRevision{
		Revision:         cfg.LatestRevision.Revision + 1,
		Description:      description,
		CreationTime:     now,
		ServerProperties: serverProperties,
	}
	cfg.LatestRevision = newRev
	cfg.Revisions = append(cfg.Revisions, newRev)
	if description != "" {
		cfg.Description = description
	}
	return cfg, nil
}

// DescribeConfigurationRevision returns a specific revision of a configuration.
func (s *Store) DescribeConfigurationRevision(arn string, revision int64) (*Configuration, *ConfigurationRevision, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.configurations[arn]
	if !ok {
		return nil, nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Configuration %s not found", arn), http.StatusNotFound)
	}
	for i := range cfg.Revisions {
		if cfg.Revisions[i].Revision == revision {
			return cfg, &cfg.Revisions[i], nil
		}
	}
	return nil, nil, service.NewAWSError("NotFoundException",
		fmt.Sprintf("Revision %d not found for configuration %s", revision, arn), http.StatusNotFound)
}

// ListConfigurationRevisions returns all revisions for a configuration.
func (s *Store) ListConfigurationRevisions(arn string) (*Configuration, []ConfigurationRevision, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.configurations[arn]
	if !ok {
		return nil, nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Configuration %s not found", arn), http.StatusNotFound)
	}
	revs := make([]ConfigurationRevision, len(cfg.Revisions))
	copy(revs, cfg.Revisions)
	return cfg, revs, nil
}

func (s *Store) DeleteConfiguration(arn string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.configurations[arn]
	if !ok {
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("Configuration %s not found", arn), http.StatusNotFound)
	}
	delete(s.configurations, arn)
	delete(s.tagsByArn, arn)
	return nil
}

// Operations.

func (s *Store) ListClusterOperations(clusterArn string) []*ClusterOperation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ClusterOperation, 0)
	for _, op := range s.operations {
		if op.ClusterArn == clusterArn {
			out = append(out, op)
		}
	}
	return out
}

func (s *Store) DescribeClusterOperation(opArn string) (*ClusterOperation, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	op, ok := s.operations[opArn]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Operation %s not found", opArn), http.StatusNotFound)
	}
	return op, nil
}

// Tags.

func (s *Store) TagResource(arn string, tags map[string]string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tagsByArn[arn]
	if !ok {
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("Resource %s not found", arn), http.StatusNotFound)
	}
	for k, v := range tags {
		existing[k] = v
	}
	return nil
}

func (s *Store) UntagResource(arn string, tagKeys []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.tagsByArn[arn]
	if !ok {
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("Resource %s not found", arn), http.StatusNotFound)
	}
	for _, k := range tagKeys {
		delete(existing, k)
	}
	return nil
}

func (s *Store) ListTagsForResource(arn string) (map[string]string, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	existing, ok := s.tagsByArn[arn]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Resource %s not found", arn), http.StatusNotFound)
	}
	cp := make(map[string]string, len(existing))
	for k, v := range existing {
		cp[k] = v
	}
	return cp, nil
}
