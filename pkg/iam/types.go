package iam

// Decision represents the result of an IAM policy evaluation.
type Decision int

const (
	// Deny is the default decision when no policy allows the request,
	// or when an explicit Deny statement matches.
	Deny Decision = iota
	// Allow is returned when at least one Allow statement matches and
	// no Deny statement overrides it.
	Allow
)

// Policy is an AWS-style IAM policy document.
type Policy struct {
	Version    string      `json:"Version"`
	Statements []Statement `json:"Statement"`
}

// Statement is a single IAM policy statement.
type Statement struct {
	SID        string                       `json:"Sid,omitempty"`
	Effect     string                       `json:"Effect"` // "Allow" or "Deny"
	Actions    []string                     `json:"Action"`
	Resources  []string                     `json:"Resource"`
	Conditions map[string]map[string]string `json:"Condition,omitempty"`
}

// EvalRequest contains the input parameters for a policy evaluation.
type EvalRequest struct {
	Principal string
	Action    string
	Resource  string
	IsRoot    bool
}

// EvalResult holds the outcome of an Evaluate call.
type EvalResult struct {
	Decision         Decision
	Reason           string
	MatchedStatement *Statement
}
