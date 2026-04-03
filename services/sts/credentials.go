package sts

// CredentialMapper maps temporary credentials to accounts for cross-account
// STS AssumeRole support. The account.Registry implements this interface.
type CredentialMapper interface {
	MapCredential(accessKeyID, accountID string)
	ResolveCredential(accessKeyID string) (string, bool)
}
