package service

import "net/http"

// Service is the interface every AWS service mock must implement.
type Service interface {
	Name() string
	Actions() []Action
	HandleRequest(ctx *RequestContext) (*Response, error)
	HealthCheck() error
}

// Action describes a single AWS API action that a service supports.
type Action struct {
	Name      string
	Method    string
	IAMAction string
	Validator RequestValidator
}

// RequestValidator validates an incoming request and returns an AWSError on failure.
type RequestValidator func(ctx *RequestContext) *AWSError

// CallerIdentity holds identity information about the entity making a request.
type CallerIdentity struct {
	AccountID   string
	ARN         string
	UserID      string
	AccessKeyID string
	IsRoot      bool
}

// RequestContext carries all parsed request data into a service handler.
type RequestContext struct {
	Action     string
	Region     string
	AccountID  string
	Identity   *CallerIdentity
	RawRequest *http.Request
	Body       []byte
	Params     map[string]string
	Service    string
}
