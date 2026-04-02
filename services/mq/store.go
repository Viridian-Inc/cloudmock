package mq

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
	"github.com/neureaux/cloudmock/pkg/service"
)

type BrokerState string

const (
	BrokerCreationInProgress BrokerState = "CREATION_IN_PROGRESS"
	BrokerRunning            BrokerState = "RUNNING"
	BrokerRebootInProgress   BrokerState = "REBOOT_IN_PROGRESS"
	BrokerDeletionInProgress BrokerState = "DELETION_IN_PROGRESS"
	BrokerCriticalActionRequired BrokerState = "CRITICAL_ACTION_REQUIRED"
)

type Broker struct {
	BrokerId           string
	BrokerArn          string
	BrokerName         string
	BrokerState        BrokerState
	EngineType         string
	EngineVersion      string
	HostInstanceType   string
	DeploymentMode     string
	AutoMinorVersionUpgrade bool
	PubliclyAccessible bool
	SubnetIds          []string
	SecurityGroups     []string
	MaintenanceWindowStartTime map[string]any
	Logs               map[string]any
	BrokerInstances    []map[string]any
	CreationTime       time.Time
	Tags               map[string]string
	Lifecycle          *lifecycle.Machine
}

type MQConfiguration struct {
	Id               string
	Arn              string
	Name             string
	Description      string
	EngineType       string
	EngineVersion    string
	LatestRevisionId int
	Data             string
	CreationTime     time.Time
	Tags             map[string]string
	Revisions        []MQConfigurationRevision
}

type MQConfigurationRevision struct {
	RevisionId   int
	Description  string
	Data         string
	CreationTime time.Time
}

type MQUser struct {
	BrokerId       string
	Username       string
	ConsoleAccess  bool
	Groups         []string
}

type Store struct {
	mu             sync.RWMutex
	brokers        map[string]*Broker         // keyed by broker ID
	configurations map[string]*MQConfiguration // keyed by config ID
	users          map[string]map[string]*MQUser // keyed by brokerId then username
	tagsByArn      map[string]map[string]string
	accountID      string
	region         string
	lcConfig       *lifecycle.Config
}

func NewStore(accountID, region string) *Store {
	return &Store{
		brokers:        make(map[string]*Broker),
		configurations: make(map[string]*MQConfiguration),
		users:          make(map[string]map[string]*MQUser),
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

func (s *Store) brokerARN(id string) string {
	return fmt.Sprintf("arn:aws:mq:%s:%s:broker:%s", s.region, s.accountID, id)
}

func (s *Store) configARN(id string) string {
	return fmt.Sprintf("arn:aws:mq:%s:%s:configuration:%s", s.region, s.accountID, id)
}

// Brokers.

func (s *Store) CreateBroker(name, engineType, engineVersion, hostInstanceType, deploymentMode string, autoMinorUpgrade, publiclyAccessible bool, subnetIds, securityGroups []string, users []map[string]any, tags map[string]string) (*Broker, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tags == nil {
		tags = make(map[string]string)
	}
	if engineType == "" {
		engineType = "ACTIVEMQ"
	}
	if engineVersion == "" {
		engineVersion = "5.17.6"
	}
	if hostInstanceType == "" {
		hostInstanceType = "mq.m5.large"
	}
	if deploymentMode == "" {
		deploymentMode = "SINGLE_INSTANCE"
	}

	brokerId := newUUID()
	arn := s.brokerARN(brokerId)
	lc := lifecycle.NewMachine(
		lifecycle.State(BrokerCreationInProgress),
		[]lifecycle.Transition{
			{From: lifecycle.State(BrokerCreationInProgress), To: lifecycle.State(BrokerRunning), Delay: 2 * time.Second},
		},
		s.lcConfig,
	)

	instances := []map[string]any{
		{
			"consoleURL": fmt.Sprintf("https://%s.mq.%s.amazonaws.com:8162", brokerId, s.region),
			"endpoints": []string{
				fmt.Sprintf("ssl://%s.mq.%s.amazonaws.com:61617", brokerId, s.region),
				fmt.Sprintf("amqp+ssl://%s.mq.%s.amazonaws.com:5671", brokerId, s.region),
			},
			"ipAddress": "10.0.0.1",
		},
	}

	b := &Broker{
		BrokerId:                brokerId,
		BrokerArn:               arn,
		BrokerName:              name,
		BrokerState:             BrokerState(lc.State()),
		EngineType:              engineType,
		EngineVersion:           engineVersion,
		HostInstanceType:        hostInstanceType,
		DeploymentMode:          deploymentMode,
		AutoMinorVersionUpgrade: autoMinorUpgrade,
		PubliclyAccessible:      publiclyAccessible,
		SubnetIds:               subnetIds,
		SecurityGroups:          securityGroups,
		BrokerInstances:         instances,
		CreationTime:            time.Now().UTC(),
		Tags:                    tags,
		Lifecycle:               lc,
	}
	s.brokers[brokerId] = b
	s.tagsByArn[arn] = tags

	// Create initial users.
	s.users[brokerId] = make(map[string]*MQUser)
	for _, u := range users {
		username, _ := u["username"].(string)
		consoleAccess, _ := u["consoleAccess"].(bool)
		if username != "" {
			s.users[brokerId][username] = &MQUser{
				BrokerId:      brokerId,
				Username:      username,
				ConsoleAccess: consoleAccess,
			}
		}
	}

	return b, nil
}

func (s *Store) DescribeBroker(brokerId string) (*Broker, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.brokers[brokerId]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Broker %s not found", brokerId), http.StatusNotFound)
	}
	b.BrokerState = BrokerState(b.Lifecycle.State())
	return b, nil
}

func (s *Store) ListBrokers() []*Broker {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Broker, 0, len(s.brokers))
	for _, b := range s.brokers {
		b.BrokerState = BrokerState(b.Lifecycle.State())
		out = append(out, b)
	}
	return out
}

func (s *Store) DeleteBroker(brokerId string) *service.AWSError {
	s.mu.Lock()
	b, ok := s.brokers[brokerId]
	if !ok {
		s.mu.Unlock()
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("Broker %s not found", brokerId), http.StatusNotFound)
	}
	delete(s.brokers, brokerId)
	delete(s.tagsByArn, b.BrokerArn)
	delete(s.users, brokerId)
	lc := b.Lifecycle
	s.mu.Unlock()
	if lc != nil {
		lc.ForceState(lifecycle.State(BrokerDeletionInProgress))
	}
	return nil
}

func (s *Store) UpdateBroker(brokerId string, hostInstanceType, engineVersion string, autoMinorUpgrade *bool) (*Broker, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.brokers[brokerId]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Broker %s not found", brokerId), http.StatusNotFound)
	}
	if hostInstanceType != "" {
		b.HostInstanceType = hostInstanceType
	}
	if engineVersion != "" {
		b.EngineVersion = engineVersion
	}
	if autoMinorUpgrade != nil {
		b.AutoMinorVersionUpgrade = *autoMinorUpgrade
	}
	return b, nil
}

func (s *Store) RebootBroker(brokerId string) *service.AWSError {
	s.mu.Lock()
	b, ok := s.brokers[brokerId]
	if !ok {
		s.mu.Unlock()
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("Broker %s not found", brokerId), http.StatusNotFound)
	}
	lc := b.Lifecycle
	s.mu.Unlock()
	if lc != nil {
		lc.ForceState(lifecycle.State(BrokerRebootInProgress))
	}
	return nil
}

// Configurations.

func (s *Store) CreateConfiguration(name, description, engineType, engineVersion, data string, tags map[string]string) (*MQConfiguration, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tags == nil {
		tags = make(map[string]string)
	}
	if engineType == "" {
		engineType = "ACTIVEMQ"
	}
	id := newUUID()
	now := time.Now().UTC()
	cfg := &MQConfiguration{
		Id:               id,
		Arn:              s.configARN(id),
		Name:             name,
		Description:      description,
		EngineType:       engineType,
		EngineVersion:    engineVersion,
		LatestRevisionId: 1,
		Data:             data,
		CreationTime:     now,
		Tags:             tags,
		Revisions: []MQConfigurationRevision{
			{RevisionId: 1, Description: description, Data: data, CreationTime: now},
		},
	}
	s.configurations[id] = cfg
	s.tagsByArn[cfg.Arn] = tags
	return cfg, nil
}

func (s *Store) DescribeConfiguration(id string) (*MQConfiguration, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.configurations[id]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Configuration %s not found", id), http.StatusNotFound)
	}
	return cfg, nil
}

func (s *Store) ListConfigurations() []*MQConfiguration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*MQConfiguration, 0, len(s.configurations))
	for _, cfg := range s.configurations {
		out = append(out, cfg)
	}
	return out
}

func (s *Store) UpdateConfiguration(id, description, data string) (*MQConfiguration, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, ok := s.configurations[id]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Configuration %s not found", id), http.StatusNotFound)
	}
	if description != "" {
		cfg.Description = description
	}
	if data != "" {
		cfg.Data = data
	}
	cfg.LatestRevisionId++
	cfg.Revisions = append(cfg.Revisions, MQConfigurationRevision{
		RevisionId:   cfg.LatestRevisionId,
		Description:  cfg.Description,
		Data:         cfg.Data,
		CreationTime: time.Now().UTC(),
	})
	return cfg, nil
}

// DescribeConfigurationRevision returns a specific configuration revision.
func (s *Store) DescribeConfigurationRevision(id string, revisionID int) (*MQConfiguration, *MQConfigurationRevision, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.configurations[id]
	if !ok {
		return nil, nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Configuration %s not found", id), http.StatusNotFound)
	}
	for i := range cfg.Revisions {
		if cfg.Revisions[i].RevisionId == revisionID {
			return cfg, &cfg.Revisions[i], nil
		}
	}
	return nil, nil, service.NewAWSError("NotFoundException",
		fmt.Sprintf("Revision %d not found for configuration %s", revisionID, id), http.StatusNotFound)
}

// ListConfigurationRevisions lists all revisions for a configuration.
func (s *Store) ListConfigurationRevisions(id string) (*MQConfiguration, []MQConfigurationRevision, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.configurations[id]
	if !ok {
		return nil, nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Configuration %s not found", id), http.StatusNotFound)
	}
	revs := make([]MQConfigurationRevision, len(cfg.Revisions))
	copy(revs, cfg.Revisions)
	return cfg, revs, nil
}

// Users.

func (s *Store) CreateUser(brokerId, username string, consoleAccess bool, groups []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	brokerUsers, ok := s.users[brokerId]
	if !ok {
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("Broker %s not found", brokerId), http.StatusNotFound)
	}
	if _, exists := brokerUsers[username]; exists {
		return service.NewAWSError("ConflictException",
			fmt.Sprintf("User %s already exists", username), http.StatusConflict)
	}
	brokerUsers[username] = &MQUser{
		BrokerId:      brokerId,
		Username:      username,
		ConsoleAccess: consoleAccess,
		Groups:        groups,
	}
	return nil
}

func (s *Store) DescribeUser(brokerId, username string) (*MQUser, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	brokerUsers, ok := s.users[brokerId]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Broker %s not found", brokerId), http.StatusNotFound)
	}
	u, ok := brokerUsers[username]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("User %s not found", username), http.StatusNotFound)
	}
	return u, nil
}

func (s *Store) ListUsers(brokerId string) ([]*MQUser, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	brokerUsers, ok := s.users[brokerId]
	if !ok {
		return nil, service.NewAWSError("NotFoundException",
			fmt.Sprintf("Broker %s not found", brokerId), http.StatusNotFound)
	}
	out := make([]*MQUser, 0, len(brokerUsers))
	for _, u := range brokerUsers {
		out = append(out, u)
	}
	return out, nil
}

func (s *Store) UpdateUser(brokerId, username string, consoleAccess *bool, groups []string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	brokerUsers, ok := s.users[brokerId]
	if !ok {
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("Broker %s not found", brokerId), http.StatusNotFound)
	}
	u, ok := brokerUsers[username]
	if !ok {
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("User %s not found", username), http.StatusNotFound)
	}
	if consoleAccess != nil {
		u.ConsoleAccess = *consoleAccess
	}
	if groups != nil {
		u.Groups = groups
	}
	return nil
}

func (s *Store) DeleteUser(brokerId, username string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	brokerUsers, ok := s.users[brokerId]
	if !ok {
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("Broker %s not found", brokerId), http.StatusNotFound)
	}
	if _, ok := brokerUsers[username]; !ok {
		return service.NewAWSError("NotFoundException",
			fmt.Sprintf("User %s not found", username), http.StatusNotFound)
	}
	delete(brokerUsers, username)
	return nil
}

// Tags.

func (s *Store) CreateTags(arn string, tags map[string]string) *service.AWSError {
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

func (s *Store) DeleteTags(arn string, tagKeys []string) *service.AWSError {
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

func (s *Store) ListTags(arn string) (map[string]string, *service.AWSError) {
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
