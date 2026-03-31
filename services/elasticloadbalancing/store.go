package elasticloadbalancing

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// LoadBalancer represents an ELB/ALB/NLB load balancer.
type LoadBalancer struct {
	Name             string
	ARN              string
	DNSName          string
	Type             string // application, network, gateway
	Scheme           string // internet-facing, internal
	State            string // active, provisioning, failed
	VpcID            string
	SecurityGroups   []string
	Subnets          []string
	AvailabilityZones []AvailabilityZone
	IpAddressType    string
	CreatedTime      time.Time
	Tags             map[string]string
}

// AvailabilityZone holds a subnet mapping for a load balancer.
type AvailabilityZone struct {
	ZoneName string
	SubnetID string
}

// TargetGroup represents an ELB target group.
type TargetGroup struct {
	Name                string
	ARN                 string
	Protocol            string
	Port                int
	VpcID               string
	TargetType          string // instance, ip, lambda, alb
	HealthCheckEnabled  bool
	HealthCheckPath     string
	HealthCheckProtocol string
	HealthCheckPort     string
	HealthyThreshold    int
	UnhealthyThreshold  int
	Targets             map[string]*Target // key: targetID
	Tags                map[string]string
}

// Target represents a registered target in a target group.
type Target struct {
	ID     string
	Port   int
	Health string // healthy, unhealthy, initial, draining, unused
}

// Listener represents a load balancer listener.
type Listener struct {
	ARN             string
	LoadBalancerARN string
	Protocol        string
	Port            int
	DefaultActions  []Action
	SslPolicy       string
	CertificateARN  string
	Tags            map[string]string
}

// Action represents a listener action.
type Action struct {
	Type           string // forward, redirect, fixed-response
	TargetGroupARN string
	Order          int
}

// Rule represents a listener rule.
type Rule struct {
	ARN         string
	ListenerARN string
	Priority    string
	Conditions  []RuleCondition
	Actions     []Action
	IsDefault   bool
}

// RuleCondition represents a rule condition.
type RuleCondition struct {
	Field  string
	Values []string
}

// Store manages all ELB resources.
type Store struct {
	mu             sync.RWMutex
	loadBalancers  map[string]*LoadBalancer  // keyed by ARN
	targetGroups   map[string]*TargetGroup   // keyed by ARN
	listeners      map[string]*Listener      // keyed by ARN
	rules          map[string]*Rule          // keyed by ARN
	lbByName       map[string]string         // name -> ARN
	tgByName       map[string]string         // name -> ARN
	accountID      string
	region         string
	listenerSeq    int
	ruleSeq        int
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		loadBalancers: make(map[string]*LoadBalancer),
		targetGroups:  make(map[string]*TargetGroup),
		listeners:     make(map[string]*Listener),
		rules:         make(map[string]*Rule),
		lbByName:      make(map[string]string),
		tgByName:      make(map[string]string),
		accountID:     accountID,
		region:        region,
	}
}

// ---- ARN helpers ----

func (s *Store) lbARN(name string) string {
	suffix := randomHex(8)
	return fmt.Sprintf("arn:aws:elasticloadbalancing:%s:%s:loadbalancer/app/%s/%s", s.region, s.accountID, name, suffix)
}

func (s *Store) nlbARN(name string) string {
	suffix := randomHex(8)
	return fmt.Sprintf("arn:aws:elasticloadbalancing:%s:%s:loadbalancer/net/%s/%s", s.region, s.accountID, name, suffix)
}

func (s *Store) tgARN(name string) string {
	suffix := randomHex(8)
	return fmt.Sprintf("arn:aws:elasticloadbalancing:%s:%s:targetgroup/%s/%s", s.region, s.accountID, name, suffix)
}

func (s *Store) listenerARN(lbARN string) string {
	s.listenerSeq++
	return fmt.Sprintf("%s/listener/%s", lbARN, randomHex(8))
}

func (s *Store) ruleARN(listenerARN string) string {
	s.ruleSeq++
	return fmt.Sprintf("%s/rule/%s", listenerARN, randomHex(8))
}

func (s *Store) dnsName(name, lbType string) string {
	suffix := randomHex(8)
	if lbType == "network" {
		return fmt.Sprintf("%s-%s.elb.%s.amazonaws.com", name, suffix, s.region)
	}
	return fmt.Sprintf("%s-%s.%s.elb.amazonaws.com", name, suffix, s.region)
}

// ---- LoadBalancer operations ----

func (s *Store) CreateLoadBalancer(name, lbType, scheme, ipAddressType, vpcID string, subnets, securityGroups []string) (*LoadBalancer, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.lbByName[name]; exists {
		return nil, false
	}

	if lbType == "" {
		lbType = "application"
	}
	if scheme == "" {
		scheme = "internet-facing"
	}
	if ipAddressType == "" {
		ipAddressType = "ipv4"
	}

	var arn string
	if lbType == "network" {
		arn = s.nlbARN(name)
	} else {
		arn = s.lbARN(name)
	}

	azs := make([]AvailabilityZone, 0, len(subnets))
	for i, sub := range subnets {
		azs = append(azs, AvailabilityZone{
			ZoneName: fmt.Sprintf("%s%c", s.region, 'a'+rune(i%3)),
			SubnetID: sub,
		})
	}

	lb := &LoadBalancer{
		Name:              name,
		ARN:               arn,
		DNSName:           s.dnsName(name, lbType),
		Type:              lbType,
		Scheme:            scheme,
		State:             "active",
		VpcID:             vpcID,
		SecurityGroups:    securityGroups,
		Subnets:           subnets,
		AvailabilityZones: azs,
		IpAddressType:     ipAddressType,
		CreatedTime:       time.Now().UTC(),
		Tags:              make(map[string]string),
	}
	s.loadBalancers[arn] = lb
	s.lbByName[name] = arn
	return lb, true
}

func (s *Store) GetLoadBalancer(arn string) (*LoadBalancer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lb, ok := s.loadBalancers[arn]
	return lb, ok
}

func (s *Store) GetLoadBalancerByName(name string) (*LoadBalancer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	arn, ok := s.lbByName[name]
	if !ok {
		return nil, false
	}
	return s.loadBalancers[arn], true
}

func (s *Store) ListLoadBalancers(names []string, arns []string) []*LoadBalancer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(names) == 0 && len(arns) == 0 {
		result := make([]*LoadBalancer, 0, len(s.loadBalancers))
		for _, lb := range s.loadBalancers {
			result = append(result, lb)
		}
		return result
	}

	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	arnSet := make(map[string]bool, len(arns))
	for _, a := range arns {
		arnSet[a] = true
	}

	result := make([]*LoadBalancer, 0)
	for _, lb := range s.loadBalancers {
		if nameSet[lb.Name] || arnSet[lb.ARN] {
			result = append(result, lb)
		}
	}
	return result
}

func (s *Store) DeleteLoadBalancer(arn string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	lb, ok := s.loadBalancers[arn]
	if !ok {
		return false
	}
	delete(s.lbByName, lb.Name)
	delete(s.loadBalancers, arn)

	// Delete associated listeners and their rules.
	for lARN, l := range s.listeners {
		if l.LoadBalancerARN == arn {
			for rARN, r := range s.rules {
				if r.ListenerARN == lARN {
					delete(s.rules, rARN)
				}
			}
			delete(s.listeners, lARN)
		}
	}
	return true
}

// ---- TargetGroup operations ----

func (s *Store) CreateTargetGroup(name, protocol string, port int, vpcID, targetType, healthPath, healthProtocol, healthPort string) (*TargetGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tgByName[name]; exists {
		return nil, false
	}

	if targetType == "" {
		targetType = "instance"
	}
	if healthProtocol == "" {
		healthProtocol = protocol
	}
	if healthPort == "" {
		healthPort = "traffic-port"
	}
	if healthPath == "" && (protocol == "HTTP" || protocol == "HTTPS") {
		healthPath = "/"
	}

	arn := s.tgARN(name)
	tg := &TargetGroup{
		Name:                name,
		ARN:                 arn,
		Protocol:            protocol,
		Port:                port,
		VpcID:               vpcID,
		TargetType:          targetType,
		HealthCheckEnabled:  true,
		HealthCheckPath:     healthPath,
		HealthCheckProtocol: healthProtocol,
		HealthCheckPort:     healthPort,
		HealthyThreshold:    5,
		UnhealthyThreshold:  2,
		Targets:             make(map[string]*Target),
		Tags:                make(map[string]string),
	}
	s.targetGroups[arn] = tg
	s.tgByName[name] = arn
	return tg, true
}

func (s *Store) GetTargetGroup(arn string) (*TargetGroup, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tg, ok := s.targetGroups[arn]
	return tg, ok
}

func (s *Store) ListTargetGroups(names []string, arns []string, lbARN string) []*TargetGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(names) == 0 && len(arns) == 0 && lbARN == "" {
		result := make([]*TargetGroup, 0, len(s.targetGroups))
		for _, tg := range s.targetGroups {
			result = append(result, tg)
		}
		return result
	}

	// Build lookup sets.
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	arnSet := make(map[string]bool, len(arns))
	for _, a := range arns {
		arnSet[a] = true
	}

	// If lbARN filter, find target groups referenced by listeners of that LB.
	tgARNsForLB := make(map[string]bool)
	if lbARN != "" {
		for _, l := range s.listeners {
			if l.LoadBalancerARN == lbARN {
				for _, a := range l.DefaultActions {
					if a.TargetGroupARN != "" {
						tgARNsForLB[a.TargetGroupARN] = true
					}
				}
			}
		}
	}

	result := make([]*TargetGroup, 0)
	for _, tg := range s.targetGroups {
		if nameSet[tg.Name] || arnSet[tg.ARN] || tgARNsForLB[tg.ARN] {
			result = append(result, tg)
		}
	}
	return result
}

func (s *Store) ModifyTargetGroup(arn, healthPath, healthProtocol, healthPort string, healthyThresh, unhealthyThresh int) (*TargetGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tg, ok := s.targetGroups[arn]
	if !ok {
		return nil, false
	}
	if healthPath != "" {
		tg.HealthCheckPath = healthPath
	}
	if healthProtocol != "" {
		tg.HealthCheckProtocol = healthProtocol
	}
	if healthPort != "" {
		tg.HealthCheckPort = healthPort
	}
	if healthyThresh > 0 {
		tg.HealthyThreshold = healthyThresh
	}
	if unhealthyThresh > 0 {
		tg.UnhealthyThreshold = unhealthyThresh
	}
	return tg, true
}

func (s *Store) DeleteTargetGroup(arn string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	tg, ok := s.targetGroups[arn]
	if !ok {
		return false
	}
	delete(s.tgByName, tg.Name)
	delete(s.targetGroups, arn)
	return true
}

// ---- Target operations ----

func (s *Store) RegisterTargets(tgARN string, targets []Target) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	tg, ok := s.targetGroups[tgARN]
	if !ok {
		return false
	}
	for _, t := range targets {
		tCopy := t
		tCopy.Health = "initial"
		tg.Targets[t.ID] = &tCopy
	}
	return true
}

func (s *Store) DeregisterTargets(tgARN string, targetIDs []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	tg, ok := s.targetGroups[tgARN]
	if !ok {
		return false
	}
	for _, id := range targetIDs {
		delete(tg.Targets, id)
	}
	return true
}

func (s *Store) DescribeTargetHealth(tgARN string) ([]*Target, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tg, ok := s.targetGroups[tgARN]
	if !ok {
		return nil, false
	}
	result := make([]*Target, 0, len(tg.Targets))
	for _, t := range tg.Targets {
		// Simulate that targets become healthy after registration.
		if t.Health == "initial" {
			t.Health = "healthy"
		}
		result = append(result, t)
	}
	return result, true
}

// ---- Listener operations ----

func (s *Store) CreateListener(lbARN, protocol string, port int, defaultActions []Action, sslPolicy, certARN string) (*Listener, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.loadBalancers[lbARN]; !ok {
		return nil, false
	}

	arn := s.listenerARN(lbARN)
	l := &Listener{
		ARN:             arn,
		LoadBalancerARN: lbARN,
		Protocol:        protocol,
		Port:            port,
		DefaultActions:  defaultActions,
		SslPolicy:       sslPolicy,
		CertificateARN:  certARN,
		Tags:            make(map[string]string),
	}
	s.listeners[arn] = l

	// Create a default rule for this listener.
	ruleARN := s.ruleARN(arn)
	defaultRule := &Rule{
		ARN:         ruleARN,
		ListenerARN: arn,
		Priority:    "default",
		Actions:     defaultActions,
		IsDefault:   true,
	}
	s.rules[ruleARN] = defaultRule

	return l, true
}

func (s *Store) ListListeners(lbARN string) []*Listener {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Listener, 0)
	for _, l := range s.listeners {
		if lbARN == "" || l.LoadBalancerARN == lbARN {
			result = append(result, l)
		}
	}
	return result
}

func (s *Store) GetListener(arn string) (*Listener, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l, ok := s.listeners[arn]
	return l, ok
}

func (s *Store) ModifyListener(arn, protocol string, port int, defaultActions []Action, sslPolicy, certARN string) (*Listener, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	l, ok := s.listeners[arn]
	if !ok {
		return nil, false
	}
	if protocol != "" {
		l.Protocol = protocol
	}
	if port > 0 {
		l.Port = port
	}
	if len(defaultActions) > 0 {
		l.DefaultActions = defaultActions
	}
	if sslPolicy != "" {
		l.SslPolicy = sslPolicy
	}
	if certARN != "" {
		l.CertificateARN = certARN
	}
	return l, true
}

func (s *Store) DeleteListener(arn string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.listeners[arn]; !ok {
		return false
	}
	delete(s.listeners, arn)

	// Delete associated rules.
	for rARN, r := range s.rules {
		if r.ListenerARN == arn {
			delete(s.rules, rARN)
		}
	}
	return true
}

// ---- Rule operations ----

func (s *Store) CreateRule(listenerARN, priority string, conditions []RuleCondition, actions []Action) (*Rule, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.listeners[listenerARN]; !ok {
		return nil, false
	}

	arn := s.ruleARN(listenerARN)
	r := &Rule{
		ARN:         arn,
		ListenerARN: listenerARN,
		Priority:    priority,
		Conditions:  conditions,
		Actions:     actions,
		IsDefault:   false,
	}
	s.rules[arn] = r
	return r, true
}

func (s *Store) ListRules(listenerARN string) []*Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Rule, 0)
	for _, r := range s.rules {
		if listenerARN == "" || r.ListenerARN == listenerARN {
			result = append(result, r)
		}
	}
	return result
}

func (s *Store) DeleteRule(arn string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.rules[arn]
	if !ok {
		return false
	}
	if r.IsDefault {
		return false // cannot delete default rule
	}
	delete(s.rules, arn)
	return true
}

// ---- Tag operations ----

func (s *Store) AddTags(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	target := s.tagMapByARN(arn)
	if target == nil {
		return false
	}
	for k, v := range tags {
		target[k] = v
	}
	return true
}

func (s *Store) RemoveTags(arn string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	target := s.tagMapByARN(arn)
	if target == nil {
		return false
	}
	for _, k := range keys {
		delete(target, k)
	}
	return true
}

func (s *Store) ListTags(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	target := s.tagMapByARN(arn)
	if target == nil {
		return nil, false
	}
	result := make(map[string]string, len(target))
	for k, v := range target {
		result[k] = v
	}
	return result, true
}

func (s *Store) tagMapByARN(arn string) map[string]string {
	for _, lb := range s.loadBalancers {
		if lb.ARN == arn {
			return lb.Tags
		}
	}
	for _, tg := range s.targetGroups {
		if tg.ARN == arn {
			return tg.Tags
		}
	}
	for _, l := range s.listeners {
		if l.ARN == arn {
			return l.Tags
		}
	}
	return nil
}

// ---- utility ----

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
