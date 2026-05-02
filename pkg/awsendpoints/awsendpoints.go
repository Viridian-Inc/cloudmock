// Package awsendpoints resolves AWS service names to their real public
// hostnames and extracts service/action identifiers from incoming AWS SDK
// requests.
//
// The endpoint table is intentionally hand-curated rather than derived from
// the AWS SDK's endpoint-resolver data: AWS hostnames are not strictly
// uniform (SES → email.<region>, Cognito user pools → cognito-idp, IAM/Route53/
// CloudFront are global), so a small allowlist gives fail-closed behavior for
// services we have not deliberately added.
package awsendpoints

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// endpoints maps AWS service names to endpoint format strings.
// Use fmt.Sprintf(pattern, region) for regional services; global services
// have no %s and are returned as-is.
var (
	endpointsMu sync.RWMutex
	endpoints   = map[string]string{
		"s3":             "s3.%s.amazonaws.com",
		"dynamodb":       "dynamodb.%s.amazonaws.com",
		"sqs":            "sqs.%s.amazonaws.com",
		"sns":            "sns.%s.amazonaws.com",
		"lambda":         "lambda.%s.amazonaws.com",
		"iam":            "iam.amazonaws.com",
		"sts":            "sts.%s.amazonaws.com",
		"kms":            "kms.%s.amazonaws.com",
		"kinesis":        "kinesis.%s.amazonaws.com",
		"firehose":       "firehose.%s.amazonaws.com",
		"logs":           "logs.%s.amazonaws.com",
		"events":         "events.%s.amazonaws.com",
		"cloudwatch":     "monitoring.%s.amazonaws.com",
		"cloudformation": "cloudformation.%s.amazonaws.com",
		"ec2":            "ec2.%s.amazonaws.com",
		"ecs":            "ecs.%s.amazonaws.com",
		"eks":            "eks.%s.amazonaws.com",
		"rds":            "rds.%s.amazonaws.com",
		"route53":        "route53.amazonaws.com",
		"cloudfront":     "cloudfront.amazonaws.com",
		"ses":            "email.%s.amazonaws.com",
		"cognito-idp":    "cognito-idp.%s.amazonaws.com",
		"apigateway":     "apigateway.%s.amazonaws.com",
		"secretsmanager": "secretsmanager.%s.amazonaws.com",
		"ssm":            "ssm.%s.amazonaws.com",
		"stepfunctions":  "states.%s.amazonaws.com",
		"codebuild":      "codebuild.%s.amazonaws.com",
		"codepipeline":   "codepipeline.%s.amazonaws.com",
		"configservice":  "config.%s.amazonaws.com",
		"cloudtrail":     "cloudtrail.%s.amazonaws.com",
	}
)

// Resolve returns the real AWS hostname for a service and region, or "" if
// the service is not in the allowlist.
func Resolve(service, region string) string {
	endpointsMu.RLock()
	pattern, ok := endpoints[service]
	endpointsMu.RUnlock()
	if !ok {
		return ""
	}
	if !strings.Contains(pattern, "%s") {
		return pattern // global service
	}
	return fmt.Sprintf(pattern, region)
}

// ServiceFromAuth extracts the AWS service from the SigV4 Authorization
// header credential scope: Credential=AKID/date/region/SERVICE/aws4_request.
// Returns "" if the header is missing or unparseable.
func ServiceFromAuth(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}

	const prefix = "Credential="
	idx := strings.Index(auth, prefix)
	if idx < 0 {
		return ""
	}
	rest := auth[idx+len(prefix):]

	end := strings.IndexAny(rest, ", ")
	if end >= 0 {
		rest = rest[:end]
	}

	parts := strings.Split(rest, "/")
	if len(parts) < 4 {
		return ""
	}
	return strings.ToLower(parts[3])
}

// Action extracts the AWS action name from the request. JSON-RPC services
// carry it in the X-Amz-Target header (e.g. "DynamoDB_20120810.GetItem"),
// query-based services carry it in the Action query parameter.
func Action(r *http.Request) string {
	if target := r.Header.Get("X-Amz-Target"); target != "" {
		if dot := strings.LastIndex(target, "."); dot >= 0 {
			return target[dot+1:]
		}
	}
	return r.URL.Query().Get("Action")
}

// Override swaps the endpoint pattern for a single service and returns a
// cleanup function that restores the previous value (or removes the entry
// if the service was not previously known). Intended for tests that point
// the recorder/contract proxy at a httptest.Server.
func Override(service, hostPattern string) func() {
	endpointsMu.Lock()
	prev, existed := endpoints[service]
	endpoints[service] = hostPattern
	endpointsMu.Unlock()

	return func() {
		endpointsMu.Lock()
		defer endpointsMu.Unlock()
		if existed {
			endpoints[service] = prev
		} else {
			delete(endpoints, service)
		}
	}
}
