package iam

import "sync"

// Engine evaluates IAM policies for principals using AWS IAM semantics.
type Engine struct {
	mu       sync.RWMutex
	policies map[string][]*Policy // principal -> policies
}

// NewEngine creates and returns a new Engine.
func NewEngine() *Engine {
	return &Engine{
		policies: make(map[string][]*Policy),
	}
}

// AddPolicy attaches a policy to a principal. Multiple policies may be added
// for the same principal; they are evaluated together.
func (e *Engine) AddPolicy(principal string, policy *Policy) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.policies[principal] = append(e.policies[principal], policy)
}

// RemovePolicies removes all policies for a principal.
func (e *Engine) RemovePolicies(principal string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.policies, principal)
}

// Evaluate applies AWS IAM evaluation logic to the given request:
//  1. Root is always allowed.
//  2. Collect all statements whose Action and Resource match the request.
//  3. If any matching statement has Effect "Deny" → explicit Deny.
//  4. If any matching statement has Effect "Allow" → Allow.
//  5. Otherwise → implicit Deny.
func (e *Engine) Evaluate(req *EvalRequest) *EvalResult {
	// Rule 1: root bypasses all policy checks.
	if req.IsRoot {
		return &EvalResult{Decision: Allow, Reason: "root"}
	}

	e.mu.RLock()
	policies := e.policies[req.Principal]
	e.mu.RUnlock()

	var allowStmt *Statement

	for _, policy := range policies {
		for i := range policy.Statements {
			stmt := &policy.Statements[i]
			if !statementMatches(stmt, req.Action, req.Resource) {
				continue
			}
			switch stmt.Effect {
			case "Deny":
				// Rule 3: explicit Deny wins immediately.
				return &EvalResult{
					Decision:         Deny,
					Reason:           "explicit deny",
					MatchedStatement: stmt,
				}
			case "Allow":
				// Record the first matching Allow; keep scanning for Denies.
				if allowStmt == nil {
					allowStmt = stmt
				}
			}
		}
	}

	// Rule 4: at least one Allow matched and no Deny overrode it.
	if allowStmt != nil {
		return &EvalResult{
			Decision:         Allow,
			Reason:           "allow",
			MatchedStatement: allowStmt,
		}
	}

	// Rule 5: implicit Deny.
	return &EvalResult{Decision: Deny, Reason: "implicit deny"}
}

// statementMatches returns true when the statement's Actions and Resources
// each contain at least one pattern that matches the given action/resource.
func statementMatches(stmt *Statement, action, resource string) bool {
	actionMatch := false
	for _, a := range stmt.Actions {
		if WildcardMatch(a, action) {
			actionMatch = true
			break
		}
	}
	if !actionMatch {
		return false
	}

	for _, r := range stmt.Resources {
		if WildcardMatch(r, resource) {
			return true
		}
	}
	return false
}
