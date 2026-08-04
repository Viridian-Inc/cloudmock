package keyvault

const (
	controlPlaneAPIVersion       = "2024-11-01"
	legacyControlPlaneAPIVersion = "2023-07-01"
	dataPlaneAPIVersion          = "2025-07-01"
	legacyDataPlaneAPIVersion    = "7.4"
)

type Vault struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties VaultProperties   `json:"properties"`
}

type VaultProperties struct {
	TenantID                     string           `json:"tenantId,omitempty"`
	SKU                          VaultSKU         `json:"sku"`
	AccessPolicies               []map[string]any `json:"accessPolicies"`
	EnabledForDeployment         bool             `json:"enabledForDeployment,omitempty"`
	EnabledForDiskEncryption     bool             `json:"enabledForDiskEncryption,omitempty"`
	EnabledForTemplateDeployment bool             `json:"enabledForTemplateDeployment,omitempty"`
	VaultURI                     string           `json:"vaultUri"`
	ProvisioningState            string           `json:"provisioningState"`
	PublicNetworkAccess          string           `json:"publicNetworkAccess,omitempty"`
}

type VaultSKU struct {
	Family string `json:"family"`
	Name   string `json:"name"`
}

type SecretAttributes struct {
	Enabled       bool   `json:"enabled"`
	NotBefore     int64  `json:"nbf,omitempty"`
	Expires       int64  `json:"exp,omitempty"`
	Created       int64  `json:"created"`
	Updated       int64  `json:"updated"`
	RecoveryLevel string `json:"recoveryLevel"`
}

type SecretBundle struct {
	Value           string            `json:"value,omitempty"`
	ID              string            `json:"id"`
	KID             string            `json:"kid,omitempty"`
	Managed         bool              `json:"managed,omitempty"`
	PreviousVersion string            `json:"previousVersion,omitempty"`
	Attributes      SecretAttributes  `json:"attributes"`
	ContentType     string            `json:"contentType,omitempty"`
	Tags            map[string]string `json:"tags,omitempty"`
}

type SecretListItem struct {
	ID          string            `json:"id"`
	KID         string            `json:"kid,omitempty"`
	Managed     bool              `json:"managed,omitempty"`
	Attributes  SecretAttributes  `json:"attributes"`
	ContentType string            `json:"contentType,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

type DeletedSecretBundle struct {
	ID                 string            `json:"id"`
	KID                string            `json:"kid,omitempty"`
	Managed            bool              `json:"managed,omitempty"`
	PreviousVersion    string            `json:"previousVersion,omitempty"`
	RecoveryID         string            `json:"recoveryId"`
	DeletedDate        int64             `json:"deletedDate"`
	ScheduledPurgeDate int64             `json:"scheduledPurgeDate"`
	Attributes         SecretAttributes  `json:"attributes"`
	ContentType        string            `json:"contentType,omitempty"`
	Tags               map[string]string `json:"tags,omitempty"`
}

type secretBackupBlob struct {
	Vault    string                  `json:"vault"`
	Name     string                  `json:"name"`
	Latest   SecretBundle            `json:"latest"`
	Versions map[string]SecretBundle `json:"versions"`
}

type CertificateAttributes struct {
	Enabled       bool   `json:"enabled"`
	NotBefore     int64  `json:"nbf,omitempty"`
	Expires       int64  `json:"exp,omitempty"`
	Created       int64  `json:"created"`
	Updated       int64  `json:"updated"`
	RecoveryLevel string `json:"recoveryLevel"`
}

type CertificateBundle struct {
	ID                string                `json:"id"`
	KID               string                `json:"kid"`
	SID               string                `json:"sid"`
	X5T               string                `json:"x5t"`
	CER               string                `json:"cer"`
	Attributes        CertificateAttributes `json:"attributes"`
	Policy            map[string]any        `json:"policy"`
	ContentType       string                `json:"contentType,omitempty"`
	Tags              map[string]string     `json:"tags,omitempty"`
	PreserveCertOrder bool                  `json:"preserveCertOrder,omitempty"`
}

type CertificateListItem struct {
	ID         string                `json:"id"`
	X5T        string                `json:"x5t"`
	Attributes CertificateAttributes `json:"attributes"`
	Tags       map[string]string     `json:"tags,omitempty"`
}

type DeletedCertificateBundle struct {
	RecoveryID         string                `json:"recoveryId"`
	DeletedDate        int64                 `json:"deletedDate"`
	ScheduledPurgeDate int64                 `json:"scheduledPurgeDate"`
	ID                 string                `json:"id"`
	KID                string                `json:"kid"`
	SID                string                `json:"sid"`
	X5T                string                `json:"x5t"`
	CER                string                `json:"cer"`
	Attributes         CertificateAttributes `json:"attributes"`
	Policy             map[string]any        `json:"policy"`
	ContentType        string                `json:"contentType,omitempty"`
	Tags               map[string]string     `json:"tags,omitempty"`
}

type CertificateOperation struct {
	ID                    string         `json:"id"`
	Issuer                map[string]any `json:"issuer,omitempty"`
	CSR                   string         `json:"csr,omitempty"`
	CancellationRequested bool           `json:"cancellation_requested"`
	Status                string         `json:"status"`
	StatusDetails         string         `json:"status_details,omitempty"`
	RequestID             string         `json:"request_id,omitempty"`
	Target                string         `json:"target,omitempty"`
	PreserveCertOrder     bool           `json:"preserveCertOrder,omitempty"`
}

type certificateBackupBlob struct {
	Vault    string                       `json:"vault"`
	Name     string                       `json:"name"`
	Latest   CertificateBundle            `json:"latest"`
	Versions map[string]CertificateBundle `json:"versions"`
}

type JsonWebKey struct {
	KID    string   `json:"kid"`
	KTY    string   `json:"kty"`
	KeyOps []string `json:"key_ops,omitempty"`
	N      string   `json:"n,omitempty"`
	E      string   `json:"e,omitempty"`
}

type KeyAttributes struct {
	Enabled       bool   `json:"enabled"`
	NotBefore     int64  `json:"nbf,omitempty"`
	Expires       int64  `json:"exp,omitempty"`
	Created       int64  `json:"created"`
	Updated       int64  `json:"updated"`
	RecoveryLevel string `json:"recoveryLevel"`
}

type KeyBundle struct {
	Key        JsonWebKey        `json:"key"`
	Managed    bool              `json:"managed,omitempty"`
	Attributes KeyAttributes     `json:"attributes"`
	Tags       map[string]string `json:"tags,omitempty"`
}

type keyBackupBlob struct {
	Vault    string               `json:"vault"`
	Name     string               `json:"name"`
	Latest   KeyBundle            `json:"latest"`
	Versions map[string]KeyBundle `json:"versions"`
}

type KeyListItem struct {
	KID        string            `json:"kid"`
	Managed    bool              `json:"managed,omitempty"`
	Attributes KeyAttributes     `json:"attributes"`
	Tags       map[string]string `json:"tags,omitempty"`
}

type DeletedKeyBundle struct {
	RecoveryID         string            `json:"recoveryId"`
	DeletedDate        int64             `json:"deletedDate"`
	ScheduledPurgeDate int64             `json:"scheduledPurgeDate"`
	Key                JsonWebKey        `json:"key"`
	Managed            bool              `json:"managed,omitempty"`
	Attributes         KeyAttributes     `json:"attributes"`
	Tags               map[string]string `json:"tags,omitempty"`
}

type DeletedKeyListItem struct {
	RecoveryID         string            `json:"recoveryId"`
	DeletedDate        int64             `json:"deletedDate"`
	ScheduledPurgeDate int64             `json:"scheduledPurgeDate"`
	KID                string            `json:"kid"`
	Managed            bool              `json:"managed,omitempty"`
	Attributes         KeyAttributes     `json:"attributes"`
	Tags               map[string]string `json:"tags,omitempty"`
}

type KeyOperationResult struct {
	KID   string `json:"kid"`
	Value string `json:"value"`
}

type KeyVerifyResult struct {
	Value bool `json:"value"`
}
