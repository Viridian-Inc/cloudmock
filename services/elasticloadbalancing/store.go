package elasticloadbalancing

import (
	"crypto/rand"
	"fmt"
	"sort"
	"sync"
	"time"
)

// LoadBalancer represents an ELB/ALB/NLB/GWLB load balancer.
type LoadBalancer struct {
	Name              string
	ARN               string
	DNSName           string
	CanonicalHostedZoneID string
	Type              string // application, network, gateway
	Scheme            string // internet-facing, internal
	State             string // provisioning, active, active_impaired, failed
	StateReason       string
	VpcID             string
	SecurityGroups    []string
	Subnets           []string
	AvailabilityZones []AvailabilityZone
	IpAddressType     string
	CreatedTime       time.Time
	Tags              map[string]string
	Attributes        map[string]string
}

// AvailabilityZone holds a subnet mapping for a load balancer.
type AvailabilityZone struct {
	ZoneName string
	SubnetID string
}

// TargetGroup represents an ELB target group.
type TargetGroup struct {
	Name                       string
	ARN                        string
	Protocol                   string
	ProtocolVersion            string
	Port                       int
	VpcID                      string
	TargetType                 string // instance, ip, lambda, alb
	HealthCheckEnabled         bool
	HealthCheckPath            string
	HealthCheckProtocol        string
	HealthCheckPort            string
	HealthCheckIntervalSeconds int
	HealthCheckTimeoutSeconds  int
	HealthyThreshold           int
	UnhealthyThreshold         int
	Matcher                    string // HTTP codes e.g. "200"
	LoadBalancerARNs           []string
	Targets                    map[string]*Target // key: targetID or targetID:port
	Tags                       map[string]string
	Attributes                 map[string]string
}

// Target represents a registered target in a target group.
type Target struct {
	ID               string
	Port             int
	AvailabilityZone string
	Health           string // healthy, unhealthy, initial, draining, unused, unavailable
	HealthReason     string
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
	AlpnPolicy      []string
	Tags            map[string]string
}

// Action represents a listener or rule action.
type Action struct {
	Type                string // forward, redirect, fixed-response, authenticate-oidc, authenticate-cognito
	TargetGroupARN      string
	Order               int
	RedirectConfig      *RedirectConfig
	FixedResponseConfig *FixedResponseConfig
}

// RedirectConfig holds redirect action configuration.
type RedirectConfig struct {
	Protocol   string
	Port       string
	Host       string
	Path       string
	Query      string
	StatusCode string // HTTP_301, HTTP_302
}

// FixedResponseConfig holds fixed-response action configuration.
type FixedResponseConfig struct {
	StatusCode  string
	ContentType string
	MessageBody string
}

// Rule represents a listener rule.
type Rule struct {
	ARN         string
	ListenerARN string
	Priority    string
	Conditions  []RuleCondition
	Actions     []Action
	IsDefault   bool
	Tags        map[string]string
}

// RuleCondition represents a rule condition.
type RuleCondition struct {
	Field  string
	Values []string
}

// Store manages all ELB resources.
type Store struct {
	mu            sync.RWMutex
	loadBalancers map[string]*LoadBalancer // keyed by ARN
	targetGroups  map[string]*TargetGroup  // keyed by ARN
	listeners     map[string]*Listener     // keyed by ARN
	rules         map[string]*Rule         // keyed by ARN
	lbByName      map[string]string        // name -> ARN
	tgByName      map[string]string        // name -> ARN
	accountID     string
	region        string
	listenerSeq   int
	ruleSeq       int
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

func (s *Store) lbARN(name, lbType string) string {
	suffix := randomHex(8)
	prefix := "app"
	switch lbType {
	case "network":
		prefix = "net"
	case "gateway":
		prefix = "gwy"
	}
	return fmt.Sprintf("arn:aws:elasticloadbalancing:%s:%s:loadbalancer/%s/%s/%s", s.region, s.accountID, prefix, name, suffix)
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

// defaultLBAttributes returns the default attributes for a load balancer type.
func defaultLBAttributes(lbType string) map[string]string {
	attrs := map[string]string{
		"deletion_protection.enabled":                         "false",
		"access_logs.s3.enabled":                              "false",
		"access_logs.s3.bucket":                               "",
		"access_logs.s3.prefix":                               "",
		"idle_timeout.timeout_seconds":                        "60",
		"routing.http.drop_invalid_header_fields.enabled":     "false",
		"routing.http.desync_mitigation_mode":                 "defensive",
		"routing.http.x_amzn_tls_version_and_cipher_suite.enabled": "false",
		"routing.http.xff_client_port.enabled":                "false",
		"routing.http2.enabled":                               "true",
		"waf.fail_open.enabled":                               "false",
	}
	if lbType == "network" {
		attrs = map[string]string{
			"deletion_protection.enabled":                   "false",
			"access_logs.s3.enabled":                        "false",
			"access_logs.s3.bucket":                         "",
			"access_logs.s3.prefix":                         "",
			"load_balancing.cross_zone.enabled":             "false",
			"dns_record.client_routing_policy":              "any_availability_zone",
		}
	}
	return attrs
}

// defaultTGAttributes returns the default attributes for a target group.
func defaultTGAttributes() map[string]string {
	return map[string]string{
		"deregistration_delay.timeout_seconds":          "300",
		"stickiness.enabled":                            "false",
		"stickiness.type":                               "lb_cookie",
		"stickiness.lb_cookie.duration_seconds":         "86400",
		"slow_start.duration_seconds":                   "0",
		"load_balancing.algorithm.type":                 "round_robin",
		"target_group_health.dns_failover.minimum_healthy_targets.count": "1",
		"target_group_health.unhealthy_state_routing.minimum_healthy_targets.count": "1",
	}
}

// ---- LoadBalancer operations ----

func (s *Store) CreateLoadBalancer(name, lbType, scheme, ipAddressType, vpcID string, subnets, securityGroups []string, tags map[string]string) (*LoadBalancer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.lbByName[name]; exists {
		return nil, fmt.Errorf("DuplicateLoadBalancerName")
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

	arn := s.lbARN(name, lbType)

	azs := make([]AvailabilityZone, 0, len(subnets))
	for i, sub := range subnets {
		azs = append(azs, AvailabilityZone{
			ZoneName: fmt.Sprintf("%s%c", s.region, 'a'+rune(i%3)),
			SubnetID: sub,
		})
	}

	if tags == nil {
		tags = make(map[string]string)
	}

	lb := &LoadBalancer{
		Name:                  name,
		ARN:                   arn,
		DNSName:               s.dnsName(name, lbType),
		CanonicalHostedZoneID: "Z35SXDOTRQ7X7K",
		Type:                  lbType,
		Scheme:                scheme,
		State:                 "provisioning",
		VpcID:                 vpcID,
		SecurityGroups:        securityGroups,
		Subnets:               subnets,
		AvailabilityZones:     azs,
		IpAddressType:         ipAddressType,
		CreatedTime:           time.Now().UTC(),
		Tags:                  tags,
		Attributes:            defaultLBAttributes(lbType),
	}
	s.loadBalancers[arn] = lb
	s.lbByName[name] = arn

	// Transition to active immediately for mock purposes.
	lb.State = "active"

	return lb, nil
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
		sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
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
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
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

func (s *Store) GetLoadBalancerAttributes(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lb, ok := s.loadBalancers[arn]
	if !ok {
		return nil, false
	}
	result := make(map[string]string, len(lb.Attributes))
	for k, v := range lb.Attributes {
		result[k] = v
	}
	return result, true
}

func (s *Store) ModifyLoadBalancerAttributes(arn string, attrs map[string]string) (map[string]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	lb, ok := s.loadBalancers[arn]
	if !ok {
		return nil, false
	}
	for k, v := range attrs {
		lb.Attributes[k] = v
	}
	result := make(map[string]string, len(lb.Attributes))
	for k, v := range lb.Attributes {
		result[k] = v
	}
	return result, true
}

func (s *Store) SetSecurityGroups(arn string, securityGroups []string) ([]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	lb, ok := s.loadBalancers[arn]
	if !ok {
		return nil, false
	}
	lb.SecurityGroups = securityGroups
	return lb.SecurityGroups, true
}

func (s *Store) SetSubnets(arn string, subnets []string) ([]AvailabilityZone, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	lb, ok := s.loadBalancers[arn]
	if !ok {
		return nil, false
	}
	lb.Subnets = subnets
	azs := make([]AvailabilityZone, 0, len(subnets))
	for i, sub := range subnets {
		azs = append(azs, AvailabilityZone{
			ZoneName: fmt.Sprintf("%s%c", s.region, 'a'+rune(i%3)),
			SubnetID: sub,
		})
	}
	lb.AvailabilityZones = azs
	return azs, true
}

// ---- TargetGroup operations ----

func (s *Store) CreateTargetGroup(name, protocol string, port int, vpcID, targetType, healthPath, healthProtocol, healthPort string, tags map[string]string) (*TargetGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tgByName[name]; exists {
		return nil, fmt.Errorf("DuplicateTargetGroupName")
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

	if tags == nil {
		tags = make(map[string]string)
	}

	arn := s.tgARN(name)
	tg := &TargetGroup{
		Name:                       name,
		ARN:                        arn,
		Protocol:                   protocol,
		ProtocolVersion:            "HTTP1",
		Port:                       port,
		VpcID:                      vpcID,
		TargetType:                 targetType,
		HealthCheckEnabled:         true,
		HealthCheckPath:            healthPath,
		HealthCheckProtocol:        healthProtocol,
		HealthCheckPort:            healthPort,
		HealthCheckIntervalSeconds: 30,
		HealthCheckTimeoutSeconds:  5,
		HealthyThreshold:           5,
		UnhealthyThreshold:         2,
		Matcher:                    "200",
		Targets:                    make(map[string]*Target),
		Tags:                       tags,
		Attributes:                 defaultTGAttributes(),
	}
	s.targetGroups[arn] = tg
	s.tgByName[name] = arn
	return tg, nil
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
		sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
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
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (s *Store) ModifyTargetGroup(arn, healthPath, healthProtocol, healthPort string, healthyThresh, unhealthyThresh, intervalSeconds, timeoutSeconds int, healthCheckEnabled *bool, matcher string) (*TargetGroup, bool) {
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
	if intervalSeconds > 0 {
		tg.HealthCheckIntervalSeconds = intervalSeconds
	}
	if timeoutSeconds > 0 {
		tg.HealthCheckTimeoutSeconds = timeoutSeconds
	}
	if healthCheckEnabled != nil {
		tg.HealthCheckEnabled = *healthCheckEnabled
	}
	if matcher != "" {
		tg.Matcher = matcher
	}
	return tg, true
}

func (s *Store) DeleteTargetGroup(arn string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tg, ok := s.targetGroups[arn]
	if !ok {
		return false, fmt.Errorf("TargetGroupNotFound")
	}

	// Check if any listener references this target group
	for _, l := range s.listeners {
		for _, a := range l.DefaultActions {
			if a.TargetGroupARN == arn {
				return false, fmt.Errorf("ResourceInUse")
			}
		}
	}
	// Check rules too
	for _, r := range s.rules {
		for _, a := range r.Actions {
			if a.TargetGroupARN == arn {
				return false, fmt.Errorf("ResourceInUse")
			}
		}
	}

	delete(s.tgByName, tg.Name)
	delete(s.targetGroups, arn)
	return true, nil
}

func (s *Store) GetTargetGroupAttributes(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tg, ok := s.targetGroups[arn]
	if !ok {
		return nil, false
	}
	result := make(map[string]string, len(tg.Attributes))
	for k, v := range tg.Attributes {
		result[k] = v
	}
	return result, true
}

func (s *Store) ModifyTargetGroupAttributes(arn string, attrs map[string]string) (map[string]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tg, ok := s.targetGroups[arn]
	if !ok {
		return nil, false
	}
	for k, v := range attrs {
		tg.Attributes[k] = v
	}
	result := make(map[string]string, len(tg.Attributes))
	for k, v := range tg.Attributes {
		result[k] = v
	}
	return result, true
}

// ---- Target operations ----

func targetKey(id string, port int) string {
	if port > 0 {
		return fmt.Sprintf("%s:%d", id, port)
	}
	return id
}

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
		tCopy.HealthReason = "Elb.RegistrationInProgress"
		key := targetKey(t.ID, t.Port)
		tg.Targets[key] = &tCopy
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
		// Try exact match first, then try with just id (no port)
		if _, exists := tg.Targets[id]; exists {
			delete(tg.Targets, id)
		} else {
			// Search for matching target by ID
			for key, t := range tg.Targets {
				if t.ID == id {
					delete(tg.Targets, key)
					break
				}
			}
		}
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
		// For mock: promote initial to healthy on read (but not draining)
		if t.Health == "initial" {
			t.Health = "healthy"
			t.HealthReason = ""
		}
		result = append(result, t)
	}
	return result, true
}

// DeregisterTargetsWithDraining sets targets to draining state instead of removing.
func (s *Store) DeregisterTargetsWithDraining(tgARN string, targetIDs []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	tg, ok := s.targetGroups[tgARN]
	if !ok {
		return false
	}
	for _, id := range targetIDs {
		// Try exact key match first
		if t, exists := tg.Targets[id]; exists {
			t.Health = "draining"
			t.HealthReason = "Target.DeregistrationInProgress"
			continue
		}
		// Search by target ID
		for _, t := range tg.Targets {
			if t.ID == id {
				t.Health = "draining"
				t.HealthReason = "Target.DeregistrationInProgress"
				break
			}
		}
	}
	return true
}

// ---- Listener operations ----

func (s *Store) CreateListener(lbARN, protocol string, port int, defaultActions []Action, sslPolicy, certARN string, tags map[string]string) (*Listener, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.loadBalancers[lbARN]; !ok {
		return nil, fmt.Errorf("LoadBalancerNotFound")
	}

	// Check for duplicate port on same LB
	for _, l := range s.listeners {
		if l.LoadBalancerARN == lbARN && l.Port == port {
			return nil, fmt.Errorf("DuplicateListener")
		}
	}

	if tags == nil {
		tags = make(map[string]string)
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
		Tags:            tags,
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
		Tags:        make(map[string]string),
	}
	s.rules[ruleARN] = defaultRule

	return l, nil
}

func (s *Store) ListListeners(lbARN string, listenerARNs []string) []*Listener {
	s.mu.RLock()
	defer s.mu.RUnlock()

	arnSet := make(map[string]bool, len(listenerARNs))
	for _, a := range listenerARNs {
		arnSet[a] = true
	}

	result := make([]*Listener, 0)
	for _, l := range s.listeners {
		if len(arnSet) > 0 {
			if arnSet[l.ARN] {
				result = append(result, l)
			}
		} else if lbARN == "" || l.LoadBalancerARN == lbARN {
			result = append(result, l)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Port < result[j].Port })
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

func (s *Store) CreateRule(listenerARN, priority string, conditions []RuleCondition, actions []Action, tags map[string]string) (*Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.listeners[listenerARN]; !ok {
		return nil, fmt.Errorf("ListenerNotFound")
	}

	// Check for duplicate priority
	for _, r := range s.rules {
		if r.ListenerARN == listenerARN && r.Priority == priority && !r.IsDefault {
			return nil, fmt.Errorf("PriorityInUse")
		}
	}

	if tags == nil {
		tags = make(map[string]string)
	}

	arn := s.ruleARN(listenerARN)
	r := &Rule{
		ARN:         arn,
		ListenerARN: listenerARN,
		Priority:    priority,
		Conditions:  conditions,
		Actions:     actions,
		IsDefault:   false,
		Tags:        tags,
	}
	s.rules[arn] = r
	return r, nil
}

func (s *Store) GetRule(arn string) (*Rule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.rules[arn]
	return r, ok
}

func (s *Store) ListRules(listenerARN string, ruleARNs []string) []*Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	arnSet := make(map[string]bool, len(ruleARNs))
	for _, a := range ruleARNs {
		arnSet[a] = true
	}

	result := make([]*Rule, 0)
	for _, r := range s.rules {
		if len(arnSet) > 0 {
			if arnSet[r.ARN] {
				result = append(result, r)
			}
		} else if listenerARN == "" || r.ListenerARN == listenerARN {
			result = append(result, r)
		}
	}
	// Sort: default rule last, then by priority
	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDefault != result[j].IsDefault {
			return !result[i].IsDefault
		}
		return result[i].Priority < result[j].Priority
	})
	return result
}

func (s *Store) ModifyRule(arn string, conditions []RuleCondition, actions []Action) (*Rule, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.rules[arn]
	if !ok {
		return nil, false
	}
	if r.IsDefault {
		// Can only modify actions on default rule
		if len(actions) > 0 {
			r.Actions = actions
		}
		return r, true
	}
	if len(conditions) > 0 {
		r.Conditions = conditions
	}
	if len(actions) > 0 {
		r.Actions = actions
	}
	return r, true
}

func (s *Store) SetRulePriorities(priorities map[string]string) ([]*Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate all rule ARNs exist
	var rules []*Rule
	for ruleARN := range priorities {
		r, ok := s.rules[ruleARN]
		if !ok {
			return nil, fmt.Errorf("RuleNotFound")
		}
		if r.IsDefault {
			return nil, fmt.Errorf("OperationNotPermitted")
		}
		rules = append(rules, r)
	}

	// Apply priorities
	for ruleARN, priority := range priorities {
		s.rules[ruleARN].Priority = priority
	}

	return rules, nil
}

func (s *Store) DeleteRule(arn string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.rules[arn]
	if !ok {
		return false, fmt.Errorf("RuleNotFound")
	}
	if r.IsDefault {
		return false, fmt.Errorf("OperationNotPermitted")
	}
	delete(s.rules, arn)
	return true, nil
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
	if lb, ok := s.loadBalancers[arn]; ok {
		return lb.Tags
	}
	if tg, ok := s.targetGroups[arn]; ok {
		return tg.Tags
	}
	if l, ok := s.listeners[arn]; ok {
		return l.Tags
	}
	if r, ok := s.rules[arn]; ok {
		return r.Tags
	}
	return nil
}

// ---- utility ----

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
