package awserr

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewError(t *testing.T) {
	err := NewError("TestCode", http.StatusTeapot, "message %s", "here")
	require.NotNil(t, err)
	assert.Equal(t, "TestCode", err.Code)
	assert.Equal(t, http.StatusTeapot, err.StatusCode())
	assert.Contains(t, err.Message, "message here")
}

func TestNotFound(t *testing.T) {
	err := NotFound(ElastiCacheClusterNotFound, "CacheCluster", "my-cluster")
	require.NotNil(t, err)
	assert.Equal(t, ElastiCacheClusterNotFound, err.Code)
	assert.Equal(t, http.StatusNotFound, err.StatusCode())
	assert.Contains(t, err.Message, "my-cluster")
}

func TestAlreadyExists(t *testing.T) {
	err := AlreadyExists(ElastiCacheClusterAlreadyExists, "CacheCluster", "my-cluster")
	require.NotNil(t, err)
	assert.Equal(t, ElastiCacheClusterAlreadyExists, err.Code)
	assert.Equal(t, http.StatusConflict, err.StatusCode())
	assert.Contains(t, err.Message, "my-cluster")
}

func TestInvalidParameter(t *testing.T) {
	err := InvalidParameter("InvalidParameterValue", "port must be between 1 and 65535")
	require.NotNil(t, err)
	assert.Equal(t, "InvalidParameterValue", err.Code)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode())
	assert.Contains(t, err.Message, "port")
}

func TestValidationException(t *testing.T) {
	err := ValidationException("field X is required")
	require.NotNil(t, err)
	assert.Equal(t, "ValidationException", err.Code)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode())
	assert.Contains(t, err.Message, "field X")
}

func TestLimitExceeded(t *testing.T) {
	err := LimitExceeded("ClusterQuotaForCustomerExceeded", "cluster limit of 10 reached")
	require.NotNil(t, err)
	assert.Equal(t, "ClusterQuotaForCustomerExceeded", err.Code)
	assert.Equal(t, http.StatusBadRequest, err.StatusCode())
}

func TestAccessDenied(t *testing.T) {
	err := AccessDenied("AccessDenied", "not authorized to call elasticache:CreateCacheCluster")
	require.NotNil(t, err)
	assert.Equal(t, "AccessDenied", err.Code)
	assert.Equal(t, http.StatusForbidden, err.StatusCode())
}

func TestErrorConstants_ELB(t *testing.T) {
	assert.Equal(t, "LoadBalancerNotFound", ELBLoadBalancerNotFound)
	assert.Equal(t, "TargetGroupNotFound", ELBTargetGroupNotFound)
	assert.Equal(t, "ListenerNotFound", ELBListenerNotFound)
	assert.Equal(t, "DuplicateLoadBalancerName", ELBDuplicateLoadBalancer)
}

func TestErrorConstants_CloudFront(t *testing.T) {
	assert.Equal(t, "NoSuchDistribution", CloudFrontDistributionNotFound)
	assert.Equal(t, "NoSuchCloudFrontOriginAccessIdentity", CloudFrontOriginAccessIdentityNotFound)
}

func TestErrorConstants_ACM(t *testing.T) {
	assert.Equal(t, "ResourceNotFoundException", ACMCertificateNotFound)
	assert.Equal(t, "InvalidArnException", ACMInvalidArn)
}

func TestErrorConstants_Organizations(t *testing.T) {
	assert.Equal(t, "AccountNotFoundException", OrgsAccountNotFound)
	assert.Equal(t, "OrganizationalUnitNotFoundException", OrgsOUNotFound)
	assert.Equal(t, "PolicyNotFoundException", OrgsPolicyNotFound)
}

func TestErrorConstants_Neptune(t *testing.T) {
	assert.Equal(t, "DBClusterNotFoundFault", NeptuneClusterNotFound)
	assert.Equal(t, "DBInstanceNotFoundFault", NeptuneInstanceNotFound)
}

func TestErrorConstants_Redshift(t *testing.T) {
	assert.Equal(t, "ClusterNotFound", RedshiftClusterNotFound)
	assert.Equal(t, "ClusterAlreadyExists", RedshiftClusterAlreadyExists)
}

func TestNotFound_ELBLoadBalancer(t *testing.T) {
	err := NotFound(ELBLoadBalancerNotFound, "LoadBalancer", "my-lb")
	require.NotNil(t, err)
	assert.Equal(t, ELBLoadBalancerNotFound, err.Code)
	assert.Equal(t, http.StatusNotFound, err.StatusCode())
	assert.Contains(t, err.Message, "my-lb")
}

func TestAlreadyExists_Redshift(t *testing.T) {
	err := AlreadyExists(RedshiftClusterAlreadyExists, "Cluster", "my-cluster")
	require.NotNil(t, err)
	assert.Equal(t, RedshiftClusterAlreadyExists, err.Code)
	assert.Equal(t, http.StatusConflict, err.StatusCode())
}
