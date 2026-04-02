package awserr

import (
	"fmt"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

// NewError creates a service.AWSError with the given code and message.
func NewError(code string, statusCode int, format string, args ...any) *service.AWSError {
	return service.NewAWSError(code, fmt.Sprintf(format, args...), statusCode)
}

// NotFound returns a 404 error with the given code for a resource that does not exist.
func NotFound(code, resourceType, id string) *service.AWSError {
	return NewError(code, http.StatusNotFound, "%s not found: %s", resourceType, id)
}

// AlreadyExists returns a 409 error with the given code for a resource that already exists.
func AlreadyExists(code, resourceType, id string) *service.AWSError {
	return NewError(code, http.StatusConflict, "%s already exists: %s", resourceType, id)
}

// InvalidParameter returns a 400 error for an invalid parameter value.
func InvalidParameter(code, message string) *service.AWSError {
	return NewError(code, http.StatusBadRequest, "%s", message)
}

// ValidationException returns a 400 ValidationException error.
func ValidationException(message string) *service.AWSError {
	return NewError("ValidationException", http.StatusBadRequest, "%s", message)
}

// LimitExceeded returns a 400 error indicating a service limit was exceeded.
func LimitExceeded(code, message string) *service.AWSError {
	return NewError(code, http.StatusBadRequest, "%s", message)
}

// AccessDenied returns a 403 error indicating the caller lacks permissions.
func AccessDenied(code, message string) *service.AWSError {
	return NewError(code, http.StatusForbidden, "%s", message)
}

// ---------------------------------------------------------------------------
// Per-service error code constants (match exact codes from AWS API docs).
// ---------------------------------------------------------------------------

// ElastiCache
const (
	ElastiCacheClusterNotFound      = "CacheClusterNotFound"
	ElastiCacheClusterAlreadyExists = "CacheClusterAlreadyExists"
	ElastiCacheSubnetGroupNotFound  = "CacheSubnetGroupNotFoundFault"
)

// ELB (Elastic Load Balancing v2)
const (
	ELBLoadBalancerNotFound  = "LoadBalancerNotFound"
	ELBTargetGroupNotFound   = "TargetGroupNotFound"
	ELBListenerNotFound      = "ListenerNotFound"
	ELBDuplicateLoadBalancer = "DuplicateLoadBalancerName"
)

// CloudFront
const (
	CloudFrontDistributionNotFound            = "NoSuchDistribution"
	CloudFrontOriginAccessIdentityNotFound    = "NoSuchCloudFrontOriginAccessIdentity"
)

// ACM (AWS Certificate Manager)
const (
	ACMCertificateNotFound = "ResourceNotFoundException"
	ACMInvalidArn          = "InvalidArnException"
)

// Organizations
const (
	OrgsAccountNotFound = "AccountNotFoundException"
	OrgsOUNotFound      = "OrganizationalUnitNotFoundException"
	OrgsPolicyNotFound  = "PolicyNotFoundException"
)

// Neptune
const (
	NeptuneClusterNotFound  = "DBClusterNotFoundFault"
	NeptuneInstanceNotFound = "DBInstanceNotFoundFault"
)

// Redshift
const (
	RedshiftClusterNotFound      = "ClusterNotFound"
	RedshiftClusterAlreadyExists = "ClusterAlreadyExists"
)
