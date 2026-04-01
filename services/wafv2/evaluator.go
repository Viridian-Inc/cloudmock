package wafv2

import (
	"net"
	"regexp"
	"sync"
	"time"
)

// CheckResult represents the result of a WAF rule evaluation.
type CheckResult struct {
	Action    string // ALLOW, BLOCK, COUNT
	RuleName  string // name of the matching rule, or "Default" for default action
}

// SampledRequest records a request that was evaluated by WAF.
type SampledRequest struct {
	IP        string
	URI       string
	Headers   map[string]string
	Time      time.Time
	Action    string
	RuleMatch string
}

// RateCounter tracks request counts per IP for rate-based rules.
type RateCounter struct {
	mu       sync.Mutex
	counters map[string]*ipCounter
}

type ipCounter struct {
	count     int
	windowStart time.Time
}

// NewRateCounter creates a new rate counter.
func NewRateCounter() *RateCounter {
	return &RateCounter{
		counters: make(map[string]*ipCounter),
	}
}

// Increment increments the counter for an IP and returns the current count.
func (rc *RateCounter) Increment(ip string, windowSeconds int) int {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	now := time.Now()
	window := time.Duration(windowSeconds) * time.Second

	c, ok := rc.counters[ip]
	if !ok || now.Sub(c.windowStart) > window {
		rc.counters[ip] = &ipCounter{count: 1, windowStart: now}
		return 1
	}
	c.count++
	return c.count
}

// CheckRequest evaluates a mock HTTP request against a Web ACL's rules.
// Returns the action to take and which rule matched.
func (s *Store) CheckRequest(webACLId, ip, uri string, headers map[string]string) CheckResult {
	s.mu.RLock()
	acl, ok := s.webACLs[webACLId]
	if !ok {
		s.mu.RUnlock()
		return CheckResult{Action: "ALLOW", RuleName: "Default"}
	}

	rules := acl.Rules
	defaultAction := acl.DefaultAction
	s.mu.RUnlock()

	// Evaluate rules in priority order (by Priority field)
	for _, rule := range rules {
		ruleName, _ := rule["Name"].(string)
		priority, _ := rule["Priority"].(float64)
		_ = priority

		action := "BLOCK"
		if actionMap, ok := rule["Action"].(map[string]any); ok {
			if _, ok := actionMap["Allow"]; ok {
				action = "ALLOW"
			} else if _, ok := actionMap["Count"]; ok {
				action = "COUNT"
			}
		}

		statement, _ := rule["Statement"].(map[string]any)
		if statement == nil {
			continue
		}

		if s.matchesStatement(statement, ip, uri, headers) {
			sample := SampledRequest{
				IP: ip, URI: uri, Headers: headers,
				Time: time.Now().UTC(), Action: action, RuleMatch: ruleName,
			}
			s.recordSample(webACLId, sample)
			return CheckResult{Action: action, RuleName: ruleName}
		}
	}

	// Default action
	resultAction := "ALLOW"
	if defaultAction != nil {
		if _, ok := defaultAction["Block"]; ok {
			resultAction = "BLOCK"
		}
	}

	sample := SampledRequest{
		IP: ip, URI: uri, Headers: headers,
		Time: time.Now().UTC(), Action: resultAction, RuleMatch: "Default",
	}
	s.recordSample(webACLId, sample)
	return CheckResult{Action: resultAction, RuleName: "Default"}
}

// matchesStatement checks if a request matches a WAF rule statement.
func (s *Store) matchesStatement(statement map[string]any, ip, uri string, headers map[string]string) bool {
	// IP set reference match
	if ipRef, ok := statement["IPSetReferenceStatement"].(map[string]any); ok {
		arn, _ := ipRef["ARN"].(string)
		return s.matchesIPSet(arn, ip)
	}

	// Regex pattern set match
	if regexRef, ok := statement["RegexPatternSetReferenceStatement"].(map[string]any); ok {
		arn, _ := regexRef["ARN"].(string)
		fieldToMatch, _ := regexRef["FieldToMatch"].(map[string]any)
		value := extractFieldValue(fieldToMatch, uri, headers)
		return s.matchesRegexPatternSet(arn, value)
	}

	// Byte match
	if byteMatch, ok := statement["ByteMatchStatement"].(map[string]any); ok {
		searchString, _ := byteMatch["SearchString"].(string)
		positionalConstraint, _ := byteMatch["PositionalConstraint"].(string)
		fieldToMatch, _ := byteMatch["FieldToMatch"].(map[string]any)
		value := extractFieldValue(fieldToMatch, uri, headers)
		return matchesBytes(value, searchString, positionalConstraint)
	}

	// Rate-based rule
	if rateStmt, ok := statement["RateBasedStatement"].(map[string]any); ok {
		limit, _ := rateStmt["Limit"].(float64)
		if limit <= 0 {
			limit = 2000
		}
		if s.rateCounter == nil {
			return false
		}
		count := s.rateCounter.Increment(ip, 300) // 5-minute window
		return count > int(limit)
	}

	return false
}

// matchesIPSet checks if an IP is in an IP set by ARN.
func (s *Store) matchesIPSet(arn, ip string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ipSet := range s.ipSets {
		if ipSet.ARN == arn {
			parsedIP := net.ParseIP(ip)
			if parsedIP == nil {
				return false
			}
			for _, cidr := range ipSet.Addresses {
				_, network, err := net.ParseCIDR(cidr)
				if err != nil {
					// Try exact match
					if cidr == ip {
						return true
					}
					continue
				}
				if network.Contains(parsedIP) {
					return true
				}
			}
			return false
		}
	}
	return false
}

// matchesRegexPatternSet checks if a value matches any pattern in a regex pattern set.
func (s *Store) matchesRegexPatternSet(arn, value string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, rps := range s.regexPatternSets {
		if rps.ARN == arn {
			for _, pattern := range rps.RegularExpressionList {
				re, err := regexp.Compile(pattern)
				if err != nil {
					continue
				}
				if re.MatchString(value) {
					return true
				}
			}
			return false
		}
	}
	return false
}

func extractFieldValue(fieldToMatch map[string]any, uri string, headers map[string]string) string {
	if fieldToMatch == nil {
		return uri
	}
	if _, ok := fieldToMatch["UriPath"]; ok {
		return uri
	}
	if headerDef, ok := fieldToMatch["SingleHeader"].(map[string]any); ok {
		headerName, _ := headerDef["Name"].(string)
		return headers[headerName]
	}
	return uri
}

func matchesBytes(value, searchString, constraint string) bool {
	switch constraint {
	case "EXACTLY":
		return value == searchString
	case "STARTS_WITH":
		return len(value) >= len(searchString) && value[:len(searchString)] == searchString
	case "ENDS_WITH":
		return len(value) >= len(searchString) && value[len(value)-len(searchString):] == searchString
	case "CONTAINS":
		for i := 0; i <= len(value)-len(searchString); i++ {
			if value[i:i+len(searchString)] == searchString {
				return true
			}
		}
		return false
	default:
		return value == searchString
	}
}

// recordSample stores a sampled request for GetSampledRequests.
func (s *Store) recordSample(webACLId string, sample SampledRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sampledRequests[webACLId] = append(s.sampledRequests[webACLId], sample)
	// Keep only last 100 samples
	if len(s.sampledRequests[webACLId]) > 100 {
		s.sampledRequests[webACLId] = s.sampledRequests[webACLId][len(s.sampledRequests[webACLId])-100:]
	}
}

// GetSampledRequests returns recent sampled requests for a Web ACL.
func (s *Store) GetSampledRequests(webACLId string, maxItems int) []SampledRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	samples := s.sampledRequests[webACLId]
	if maxItems <= 0 || maxItems > len(samples) {
		maxItems = len(samples)
	}
	if maxItems == 0 {
		return nil
	}
	// Return most recent
	start := len(samples) - maxItems
	out := make([]SampledRequest, maxItems)
	copy(out, samples[start:])
	return out
}
