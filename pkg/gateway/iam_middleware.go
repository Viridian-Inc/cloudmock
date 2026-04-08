package gateway

import (
	"fmt"
	"net/http"

	iampkg "github.com/Viridian-Inc/cloudmock/pkg/iam"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// unauthenticatedActions lists AWS API actions that do not require an
// Authorization header. These are client-side Cognito operations that
// are performed by end-users (browsers/mobile apps) without AWS credentials.
var unauthenticatedActions = map[string]map[string]bool{
	"cognito-idp": {
		"SignUp":                     true,
		"ConfirmSignUp":             true,
		"InitiateAuth":              true,
		"RespondToAuthChallenge":    true,
		"ForgotPassword":            true,
		"ConfirmForgotPassword":     true,
		"GetUser":                   true,
		"ChangePassword":            true,
		"GlobalSignOut":             true,
		"RevokeToken":               true,
	},
}

// isUnauthenticatedAction returns true if the given service + action
// combination is a known unauthenticated API call (no Authorization header).
func isUnauthenticatedAction(svcName, action string) bool {
	if actions, ok := unauthenticatedActions[svcName]; ok {
		return actions[action]
	}
	return false
}

// authenticateRequest extracts and validates the caller's identity.
// When IAM mode is "none" it returns a root identity without checking credentials.
// Otherwise it looks up the access key from the store; missing keys yield a 403.
func (g *Gateway) authenticateRequest(r *http.Request) (*service.CallerIdentity, *service.AWSError) {
	if g.cfg.IAM.Mode == "none" {
		return &service.CallerIdentity{
			AccountID:   g.cfg.AccountID,
			ARN:         fmt.Sprintf("arn:aws:iam::%s:root", g.cfg.AccountID),
			UserID:      g.cfg.AccountID,
			AccessKeyID: g.cfg.IAM.RootAccessKey,
			IsRoot:      true,
		}, nil
	}

	// Presigned URL requests carry auth in query params — grant limited identity.
	if r.URL.Query().Get("X-Amz-Algorithm") != "" {
		return &service.CallerIdentity{
			AccountID:   g.cfg.AccountID,
			ARN:         fmt.Sprintf("arn:aws:iam::%s:user/presigned-user", g.cfg.AccountID),
			UserID:      "AIDAPRESIGNED",
			AccessKeyID: "presigned",
			IsRoot:      false,
		}, nil
	}

	keyID, err := iampkg.ExtractAccessKeyID(r)
	if err != nil {
		return nil, service.NewAWSError(
			"InvalidClientTokenId",
			"The security token included in the request is invalid.",
			http.StatusForbidden,
		)
	}

	if g.store == nil {
		return nil, service.NewAWSError(
			"InvalidClientTokenId",
			"The security token included in the request is invalid.",
			http.StatusForbidden,
		)
	}

	key, err := g.store.LookupAccessKey(keyID)
	if err != nil {
		return nil, service.NewAWSError(
			"InvalidClientTokenId",
			"The security token included in the request is invalid.",
			http.StatusForbidden,
		)
	}

	var arn string
	if key.IsRoot {
		arn = fmt.Sprintf("arn:aws:iam::%s:root", key.AccountID)
	} else {
		arn = fmt.Sprintf("arn:aws:iam::%s:user/%s", key.AccountID, key.UserName)
	}

	return &service.CallerIdentity{
		AccountID:   key.AccountID,
		ARN:         arn,
		UserID:      key.UserName,
		AccessKeyID: key.AccessKeyID,
		IsRoot:      key.IsRoot,
	}, nil
}

// authorizeRequest checks whether the given identity may perform iamAction on resource.
// When IAM mode is not "enforce" all requests are allowed. Root identities are always allowed.
func (g *Gateway) authorizeRequest(identity *service.CallerIdentity, iamAction, resource string) *service.AWSError {
	if g.cfg.IAM.Mode != "enforce" {
		return nil
	}

	if g.engine == nil {
		return nil
	}

	result := g.engine.Evaluate(&iampkg.EvalRequest{
		Principal: identity.UserID,
		Action:    iamAction,
		Resource:  resource,
		IsRoot:    identity.IsRoot,
	})

	if result.Decision == iampkg.Deny {
		return service.ErrAccessDenied(iamAction)
	}

	return nil
}
