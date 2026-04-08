package sts

import (
	"encoding/xml"
	"net/http"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ---- XML types ----

// xmlGetCallerIdentityResponse is the top-level XML envelope for GetCallerIdentity.
type xmlGetCallerIdentityResponse struct {
	XMLName xml.Name                    `xml:"GetCallerIdentityResponse"`
	Xmlns   string                      `xml:"xmlns,attr"`
	Result  xmlGetCallerIdentityResult  `xml:"GetCallerIdentityResult"`
	Meta    xmlResponseMetadata         `xml:"ResponseMetadata"`
}

type xmlGetCallerIdentityResult struct {
	Arn     string `xml:"Arn"`
	UserID  string `xml:"UserId"`
	Account string `xml:"Account"`
}

// xmlAssumeRoleResponse is the top-level XML envelope for AssumeRole.
type xmlAssumeRoleResponse struct {
	XMLName xml.Name              `xml:"AssumeRoleResponse"`
	Xmlns   string                `xml:"xmlns,attr"`
	Result  xmlAssumeRoleResult   `xml:"AssumeRoleResult"`
	Meta    xmlResponseMetadata   `xml:"ResponseMetadata"`
}

type xmlAssumeRoleResult struct {
	Credentials      xmlCredentials      `xml:"Credentials"`
	AssumedRoleUser  xmlAssumedRoleUser  `xml:"AssumedRoleUser"`
}

type xmlAssumedRoleUser struct {
	Arn            string `xml:"Arn"`
	AssumedRoleID  string `xml:"AssumedRoleId"`
}

// xmlGetSessionTokenResponse is the top-level XML envelope for GetSessionToken.
type xmlGetSessionTokenResponse struct {
	XMLName xml.Name                   `xml:"GetSessionTokenResponse"`
	Xmlns   string                     `xml:"xmlns,attr"`
	Result  xmlGetSessionTokenResult   `xml:"GetSessionTokenResult"`
	Meta    xmlResponseMetadata        `xml:"ResponseMetadata"`
}

type xmlGetSessionTokenResult struct {
	Credentials xmlCredentials `xml:"Credentials"`
}

// xmlCredentials is reused across AssumeRole and GetSessionToken responses.
type xmlCredentials struct {
	AccessKeyID     string `xml:"AccessKeyId"`
	SecretAccessKey string `xml:"SecretAccessKey"`
	SessionToken    string `xml:"SessionToken"`
	Expiration      string `xml:"Expiration"`
}

// xmlResponseMetadata holds the RequestId returned in every STS response.
type xmlResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

const stsXmlns = "https://sts.amazonaws.com/doc/2011-06-15/"

// ---- handlers ----

func handleGetCallerIdentity(ctx *service.RequestContext) (*service.Response, error) {
	identity := ctx.Identity

	resp := &xmlGetCallerIdentityResponse{
		Xmlns: stsXmlns,
		Result: xmlGetCallerIdentityResult{
			Arn:     identity.ARN,
			UserID:  identity.UserID,
			Account: identity.AccountID,
		},
		Meta: xmlResponseMetadata{RequestID: newRequestID()},
	}

	data, err := xml.Marshal(resp)
	if err != nil {
		return nil, err
	}
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        data,
		RawContentType: "text/xml",
	}, nil
}

func handleAssumeRole(ctx *service.RequestContext, accountID string, credMapper CredentialMapper) (*service.Response, error) {
	formVals, err := parseFormBody(ctx.Body)
	if err != nil {
		return &service.Response{Format: service.FormatXML},
			service.NewAWSError("InvalidRequest", "Failed to parse request body.", http.StatusBadRequest)
	}

	roleArn := formVals.Get("RoleArn")
	sessionName := formVals.Get("RoleSessionName")

	if roleArn == "" {
		return &service.Response{Format: service.FormatXML},
			service.ErrValidation("RoleArn is required.")
	}
	if sessionName == "" {
		return &service.Response{Format: service.FormatXML},
			service.ErrValidation("RoleSessionName is required.")
	}

	// Parse target account ID from the role ARN.
	// ARN format: arn:aws:iam::{accountID}:role/{roleName}
	targetAccountID := accountID
	if parts := strings.Split(roleArn, ":"); len(parts) >= 5 && parts[4] != "" {
		targetAccountID = parts[4]
	}

	creds := generateCredentials()

	// If a credential mapper is available and this is a cross-account assume,
	// register the temporary credential against the target account.
	if credMapper != nil && targetAccountID != "" {
		credMapper.MapCredential(creds.AccessKeyID, targetAccountID)
	}

	assumedRoleID := "AROA" + randomHex(16) + ":" + sessionName
	assumedRoleArn := roleArn + "/" + sessionName

	resp := &xmlAssumeRoleResponse{
		Xmlns: stsXmlns,
		Result: xmlAssumeRoleResult{
			Credentials: xmlCredentials{
				AccessKeyID:     creds.AccessKeyID,
				SecretAccessKey: creds.SecretAccessKey,
				SessionToken:    creds.SessionToken,
				Expiration:      creds.Expiration.Format("2006-01-02T15:04:05Z"),
			},
			AssumedRoleUser: xmlAssumedRoleUser{
				Arn:           assumedRoleArn,
				AssumedRoleID: assumedRoleID,
			},
		},
		Meta: xmlResponseMetadata{RequestID: newRequestID()},
	}

	data, err := xml.Marshal(resp)
	if err != nil {
		return nil, err
	}
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        data,
		RawContentType: "text/xml",
	}, nil
}

func handleGetSessionToken(ctx *service.RequestContext) (*service.Response, error) {
	creds := generateCredentials()

	resp := &xmlGetSessionTokenResponse{
		Xmlns: stsXmlns,
		Result: xmlGetSessionTokenResult{
			Credentials: xmlCredentials{
				AccessKeyID:     creds.AccessKeyID,
				SecretAccessKey: creds.SecretAccessKey,
				SessionToken:    creds.SessionToken,
				Expiration:      creds.Expiration.Format("2006-01-02T15:04:05Z"),
			},
		},
		Meta: xmlResponseMetadata{RequestID: newRequestID()},
	}

	data, err := xml.Marshal(resp)
	if err != nil {
		return nil, err
	}
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        data,
		RawContentType: "text/xml",
	}, nil
}
