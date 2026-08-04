package keyvault

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/azurearm"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// KeyVaultService implements a first-slice Azure Key Vault mock with ARM vault
// management and data-plane secret operations.
type KeyVaultService struct {
	mu             sync.RWMutex
	vaults         map[string]Vault
	secrets        map[string]map[string]SecretBundle
	secretVersions map[string]map[string]map[string]SecretBundle
	deletedSecrets map[string]map[string]deletedSecretState
	certificates   map[string]map[string]CertificateBundle
	certVersions   map[string]map[string]map[string]CertificateBundle
	deletedCerts   map[string]map[string]deletedCertificateState
	certOperations map[string]map[string]CertificateOperation
	keys           map[string]map[string]KeyBundle
	keyVersions    map[string]map[string]map[string]KeyBundle
	deletedKeys    map[string]map[string]deletedKeyState
	nextID         uint64
}

type deletedSecretState struct {
	Bundle   DeletedSecretBundle
	Latest   SecretBundle
	Versions map[string]SecretBundle
}

type deletedCertificateState struct {
	Bundle   DeletedCertificateBundle
	Latest   CertificateBundle
	Versions map[string]CertificateBundle
}

type deletedKeyState struct {
	Bundle   DeletedKeyBundle
	Latest   KeyBundle
	Versions map[string]KeyBundle
}

func New() *KeyVaultService {
	return &KeyVaultService{
		vaults:         make(map[string]Vault),
		secrets:        make(map[string]map[string]SecretBundle),
		secretVersions: make(map[string]map[string]map[string]SecretBundle),
		deletedSecrets: make(map[string]map[string]deletedSecretState),
		certificates:   make(map[string]map[string]CertificateBundle),
		certVersions:   make(map[string]map[string]map[string]CertificateBundle),
		deletedCerts:   make(map[string]map[string]deletedCertificateState),
		certOperations: make(map[string]map[string]CertificateOperation),
		keys:           make(map[string]map[string]KeyBundle),
		keyVersions:    make(map[string]map[string]map[string]KeyBundle),
		deletedKeys:    make(map[string]map[string]deletedKeyState),
	}
}

func (s *KeyVaultService) Name() string { return "Microsoft.KeyVault" }

func (s *KeyVaultService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdate", Method: http.MethodPut, IAMAction: "azure:Microsoft.KeyVault/vaults/write"},
		{Name: "Get", Method: http.MethodGet, IAMAction: "azure:Microsoft.KeyVault/vaults/read"},
		{Name: "List", Method: http.MethodGet, IAMAction: "azure:Microsoft.KeyVault/vaults/read"},
		{Name: "Delete", Method: http.MethodDelete, IAMAction: "azure:Microsoft.KeyVault/vaults/delete"},
		{Name: "SetSecret", Method: http.MethodPut, IAMAction: "azure:Microsoft.KeyVault/secrets/set"},
		{Name: "GetSecret", Method: http.MethodGet, IAMAction: "azure:Microsoft.KeyVault/secrets/get"},
		{Name: "ListSecrets", Method: http.MethodGet, IAMAction: "azure:Microsoft.KeyVault/secrets/list"},
		{Name: "ListSecretVersions", Method: http.MethodGet, IAMAction: "azure:Microsoft.KeyVault/secrets/versions/read"},
		{Name: "UpdateSecret", Method: http.MethodPatch, IAMAction: "azure:Microsoft.KeyVault/secrets/update"},
		{Name: "BackupSecret", Method: http.MethodPost, IAMAction: "azure:Microsoft.KeyVault/secrets/backup/action"},
		{Name: "RestoreSecret", Method: http.MethodPost, IAMAction: "azure:Microsoft.KeyVault/secrets/restore/action"},
		{Name: "DeleteSecret", Method: http.MethodDelete, IAMAction: "azure:Microsoft.KeyVault/secrets/delete"},
		{Name: "ListDeletedSecrets", Method: http.MethodGet, IAMAction: "azure:Microsoft.KeyVault/deletedsecrets/list"},
		{Name: "GetDeletedSecret", Method: http.MethodGet, IAMAction: "azure:Microsoft.KeyVault/deletedsecrets/get"},
		{Name: "RecoverDeletedSecret", Method: http.MethodPost, IAMAction: "azure:Microsoft.KeyVault/deletedsecrets/recover/action"},
		{Name: "PurgeDeletedSecret", Method: http.MethodDelete, IAMAction: "azure:Microsoft.KeyVault/deletedsecrets/purge/action"},
		{Name: "ImportCertificate", Method: http.MethodPost, IAMAction: "azure:Microsoft.KeyVault/certificates/import/action"},
		{Name: "BackupCertificate", Method: http.MethodPost, IAMAction: "azure:Microsoft.KeyVault/certificates/backup/action"},
		{Name: "RestoreCertificate", Method: http.MethodPost, IAMAction: "azure:Microsoft.KeyVault/certificates/restore/action"},
		{Name: "CreateCertificate", Method: http.MethodPost, IAMAction: "azure:Microsoft.KeyVault/certificates/create/action"},
		{Name: "GetCertificateOperation", Method: http.MethodGet, IAMAction: "azure:Microsoft.KeyVault/certificates/pending/read"},
		{Name: "GetCertificate", Method: http.MethodGet, IAMAction: "azure:Microsoft.KeyVault/certificates/get"},
		{Name: "ListCertificates", Method: http.MethodGet, IAMAction: "azure:Microsoft.KeyVault/certificates/list"},
		{Name: "ListCertificateVersions", Method: http.MethodGet, IAMAction: "azure:Microsoft.KeyVault/certificates/versions/read"},
		{Name: "DeleteCertificate", Method: http.MethodDelete, IAMAction: "azure:Microsoft.KeyVault/certificates/delete"},
		{Name: "ListDeletedCertificates", Method: http.MethodGet, IAMAction: "azure:Microsoft.KeyVault/deletedcertificates/list"},
		{Name: "GetDeletedCertificate", Method: http.MethodGet, IAMAction: "azure:Microsoft.KeyVault/deletedcertificates/get"},
		{Name: "RecoverDeletedCertificate", Method: http.MethodPost, IAMAction: "azure:Microsoft.KeyVault/deletedcertificates/recover/action"},
		{Name: "PurgeDeletedCertificate", Method: http.MethodDelete, IAMAction: "azure:Microsoft.KeyVault/deletedcertificates/purge/action"},
		{Name: "CreateKey", Method: http.MethodPost, IAMAction: "azure:Microsoft.KeyVault/keys/create"},
		{Name: "GetKey", Method: http.MethodGet, IAMAction: "azure:Microsoft.KeyVault/keys/get"},
		{Name: "ListKeys", Method: http.MethodGet, IAMAction: "azure:Microsoft.KeyVault/keys/list"},
		{Name: "ListKeyVersions", Method: http.MethodGet, IAMAction: "azure:Microsoft.KeyVault/keys/versions/read"},
		{Name: "UpdateKey", Method: http.MethodPatch, IAMAction: "azure:Microsoft.KeyVault/keys/update"},
		{Name: "BackupKey", Method: http.MethodPost, IAMAction: "azure:Microsoft.KeyVault/keys/backup/action"},
		{Name: "RestoreKey", Method: http.MethodPost, IAMAction: "azure:Microsoft.KeyVault/keys/restore/action"},
		{Name: "DeleteKey", Method: http.MethodDelete, IAMAction: "azure:Microsoft.KeyVault/keys/delete"},
		{Name: "ListDeletedKeys", Method: http.MethodGet, IAMAction: "azure:Microsoft.KeyVault/deletedkeys/list"},
		{Name: "GetDeletedKey", Method: http.MethodGet, IAMAction: "azure:Microsoft.KeyVault/deletedkeys/get"},
		{Name: "RecoverDeletedKey", Method: http.MethodPost, IAMAction: "azure:Microsoft.KeyVault/deletedkeys/recover/action"},
		{Name: "PurgeDeletedKey", Method: http.MethodDelete, IAMAction: "azure:Microsoft.KeyVault/deletedkeys/purge/action"},
		{Name: "Encrypt", Method: http.MethodPost, IAMAction: "azure:Microsoft.KeyVault/keys/encrypt/action"},
		{Name: "Decrypt", Method: http.MethodPost, IAMAction: "azure:Microsoft.KeyVault/keys/decrypt/action"},
		{Name: "Sign", Method: http.MethodPost, IAMAction: "azure:Microsoft.KeyVault/keys/sign/action"},
		{Name: "Verify", Method: http.MethodPost, IAMAction: "azure:Microsoft.KeyVault/keys/verify/action"},
		{Name: "WrapKey", Method: http.MethodPost, IAMAction: "azure:Microsoft.KeyVault/keys/wrap/action"},
		{Name: "UnwrapKey", Method: http.MethodPost, IAMAction: "azure:Microsoft.KeyVault/keys/unwrap/action"},
	}
}

func (s *KeyVaultService) HealthCheck() error { return nil }

func (s *KeyVaultService) ServiceKeys() []routing.ServiceKey {
	return []routing.ServiceKey{
		{Provider: routing.ProviderAzure, Service: "Microsoft.KeyVault/vaults", APIVersion: controlPlaneAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.KeyVault/vaults", APIVersion: legacyControlPlaneAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.KeyVault/secrets", APIVersion: dataPlaneAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.KeyVault/secrets", APIVersion: legacyDataPlaneAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.KeyVault/keys", APIVersion: dataPlaneAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.KeyVault/keys", APIVersion: legacyDataPlaneAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.KeyVault/certificates", APIVersion: dataPlaneAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.KeyVault/certificates", APIVersion: legacyDataPlaneAPIVersion},
	}
}

func (s *KeyVaultService) SupportsTemplateResource(resource map[string]any) bool {
	return strings.EqualFold(stringValue(resource["type"]), "Microsoft.KeyVault/vaults")
}

func (s *KeyVaultService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported template resource type %q", stringValue(resource["type"]))
	}
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("key vault template resource is missing name")
	}

	body := map[string]any{
		"location":   resource["location"],
		"tags":       resource["tags"],
		"properties": resource["properties"],
	}
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}
	resp, err := s.createOrUpdateVault(subscriptionID, resourceGroup, name, data)
	if err != nil {
		return nil, err
	}
	var out any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *KeyVaultService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}

	if vaultName, ok := dataPlaneVault(ctx.RawRequest); ok {
		return s.handleDataPlane(ctx, vaultName)
	}
	return s.handleControlPlane(ctx)
}

func (s *KeyVaultService) handleControlPlane(ctx *service.RequestContext) (*service.Response, error) {
	parts := splitPath(ctx.RawRequest.URL.EscapedPath())
	providerIndex := segmentIndex(parts, "providers")
	if providerIndex < 4 || providerIndex+2 >= len(parts) ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[providerIndex+1], "Microsoft.KeyVault") ||
		!strings.EqualFold(parts[providerIndex+2], "vaults") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The key vault route is not implemented.")
	}

	subscriptionID := parts[1]
	resourceGroup := parts[3]
	if len(parts) == providerIndex+3 && ctx.RawRequest.Method == http.MethodGet {
		return s.listVaults(subscriptionID, resourceGroup)
	}
	if len(parts) != providerIndex+4 {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The key vault route is not implemented.")
	}

	name := parts[providerIndex+3]
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateVault(subscriptionID, resourceGroup, name, ctx.Body)
	case http.MethodGet:
		return s.getVault(subscriptionID, resourceGroup, name)
	case http.MethodDelete:
		return s.deleteVault(subscriptionID, resourceGroup, name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *KeyVaultService) createOrUpdateVault(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		Tags       map[string]any `json:"tags"`
		Properties struct {
			TenantID                     string           `json:"tenantId"`
			SKU                          VaultSKU         `json:"sku"`
			AccessPolicies               []map[string]any `json:"accessPolicies"`
			EnabledForDeployment         bool             `json:"enabledForDeployment"`
			EnabledForDiskEncryption     bool             `json:"enabledForDiskEncryption"`
			EnabledForTemplateDeployment bool             `json:"enabledForTemplateDeployment"`
			PublicNetworkAccess          string           `json:"publicNetworkAccess"`
		} `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Location == "" {
		input.Location = "eastus"
	}
	if input.Properties.SKU.Family == "" {
		input.Properties.SKU.Family = "A"
	}
	if input.Properties.SKU.Name == "" {
		input.Properties.SKU.Name = "standard"
	}
	if input.Properties.PublicNetworkAccess == "" {
		input.Properties.PublicNetworkAccess = "Enabled"
	}
	if input.Properties.AccessPolicies == nil {
		input.Properties.AccessPolicies = []map[string]any{}
	}

	key := vaultKey(subscriptionID, resourceGroup, name)
	vault := Vault{
		ID:       vaultResourceID(subscriptionID, resourceGroup, name),
		Name:     name,
		Type:     "Microsoft.KeyVault/vaults",
		Location: input.Location,
		Tags:     stringifyTags(input.Tags),
		Properties: VaultProperties{
			TenantID:                     input.Properties.TenantID,
			SKU:                          input.Properties.SKU,
			AccessPolicies:               input.Properties.AccessPolicies,
			EnabledForDeployment:         input.Properties.EnabledForDeployment,
			EnabledForDiskEncryption:     input.Properties.EnabledForDiskEncryption,
			EnabledForTemplateDeployment: input.Properties.EnabledForTemplateDeployment,
			VaultURI:                     vaultURI(name),
			ProvisioningState:            "Succeeded",
			PublicNetworkAccess:          input.Properties.PublicNetworkAccess,
		},
	}

	s.mu.Lock()
	_, existed := s.vaults[key]
	s.vaults[key] = vault
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, vault)
}

func (s *KeyVaultService) getVault(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	vault, ok := s.vaults[vaultKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Key vault %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, vault)
}

func (s *KeyVaultService) listVaults(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := vaultKeyPrefix(subscriptionID, resourceGroup)

	s.mu.RLock()
	values := make([]Vault, 0)
	for key, vault := range s.vaults {
		if strings.HasPrefix(key, prefix) {
			values = append(values, vault)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *KeyVaultService) deleteVault(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	key := vaultKey(subscriptionID, resourceGroup, name)

	s.mu.Lock()
	if _, ok := s.vaults[key]; !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Key vault %q could not be found.", name))
	}
	delete(s.vaults, key)
	delete(s.secrets, strings.ToLower(name))
	delete(s.secretVersions, strings.ToLower(name))
	delete(s.deletedSecrets, strings.ToLower(name))
	delete(s.certificates, strings.ToLower(name))
	delete(s.certVersions, strings.ToLower(name))
	delete(s.deletedCerts, strings.ToLower(name))
	delete(s.keys, strings.ToLower(name))
	delete(s.keyVersions, strings.ToLower(name))
	delete(s.deletedKeys, strings.ToLower(name))
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *KeyVaultService) handleDataPlane(ctx *service.RequestContext, vaultName string) (*service.Response, error) {
	parts := splitPath(ctx.RawRequest.URL.EscapedPath())
	if len(parts) == 0 {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The key vault route is not implemented.")
	}
	switch {
	case strings.EqualFold(parts[0], "secrets"):
		return s.handleSecrets(ctx, vaultName)
	case strings.EqualFold(parts[0], "deletedsecrets"):
		return s.handleDeletedSecrets(ctx, vaultName)
	case strings.EqualFold(parts[0], "certificates"):
		return s.handleCertificates(ctx, vaultName)
	case strings.EqualFold(parts[0], "deletedcertificates"):
		return s.handleDeletedCertificates(ctx, vaultName)
	case strings.EqualFold(parts[0], "keys"):
		return s.handleKeys(ctx, vaultName)
	case strings.EqualFold(parts[0], "deletedkeys"):
		return s.handleDeletedKeys(ctx, vaultName)
	default:
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The key vault route is not implemented.")
	}
}

func (s *KeyVaultService) handleSecrets(ctx *service.RequestContext, vaultName string) (*service.Response, error) {
	parts := splitPath(ctx.RawRequest.URL.EscapedPath())
	if len(parts) == 0 || !strings.EqualFold(parts[0], "secrets") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The key vault secret route is not implemented.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		if len(parts) == 2 {
			return s.setSecret(vaultName, parts[1], ctx.Body)
		}
	case http.MethodGet:
		if len(parts) == 1 {
			return s.listSecrets(vaultName, ctx.RawRequest)
		}
		if len(parts) == 3 && strings.EqualFold(parts[2], "versions") {
			return s.listSecretVersions(vaultName, parts[1], ctx.RawRequest)
		}
		if len(parts) == 2 || len(parts) == 3 {
			version := ""
			if len(parts) == 3 {
				version = parts[2]
			}
			return s.getSecret(vaultName, parts[1], version)
		}
	case http.MethodPatch:
		if len(parts) == 3 && !strings.EqualFold(parts[2], "versions") {
			return s.updateSecretProperties(vaultName, parts[1], parts[2], ctx.Body)
		}
	case http.MethodPost:
		if len(parts) == 2 && strings.EqualFold(parts[1], "restore") {
			return s.restoreSecret(vaultName, ctx.Body)
		}
		if len(parts) == 3 && strings.EqualFold(parts[2], "backup") {
			return s.backupSecret(vaultName, parts[1])
		}
	case http.MethodDelete:
		if len(parts) == 2 {
			return s.deleteSecret(vaultName, parts[1])
		}
	}
	return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
}

func (s *KeyVaultService) handleDeletedSecrets(ctx *service.RequestContext, vaultName string) (*service.Response, error) {
	parts := splitPath(ctx.RawRequest.URL.EscapedPath())
	if len(parts) == 0 || !strings.EqualFold(parts[0], "deletedsecrets") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The deleted secret route is not implemented.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodGet:
		if len(parts) == 1 {
			return s.listDeletedSecrets(vaultName, ctx.RawRequest)
		}
		if len(parts) == 2 {
			return s.getDeletedSecret(vaultName, parts[1])
		}
	case http.MethodPost:
		if len(parts) == 3 && strings.EqualFold(parts[2], "recover") {
			return s.recoverDeletedSecret(vaultName, parts[1])
		}
	case http.MethodDelete:
		if len(parts) == 2 {
			return s.purgeDeletedSecret(vaultName, parts[1])
		}
	}
	return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
}

func (s *KeyVaultService) setSecret(vaultName, name string, body []byte) (*service.Response, error) {
	var input struct {
		Value       string         `json:"value"`
		Attributes  map[string]any `json:"attributes"`
		ContentType string         `json:"contentType"`
		Tags        map[string]any `json:"tags"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The request content was invalid.")
		}
	}

	now := time.Now().UTC().Unix()
	enabled := true
	if value, ok := input.Attributes["enabled"].(bool); ok {
		enabled = value
	}
	var notBefore int64
	if value, ok := int64MapValue(input.Attributes, "nbf"); ok {
		notBefore = value
	}
	var expires int64
	if value, ok := int64MapValue(input.Attributes, "exp"); ok {
		expires = value
	}

	s.mu.Lock()
	version := s.nextVersion()
	bundle := SecretBundle{
		Value:       input.Value,
		ID:          secretID(vaultName, name, version),
		ContentType: input.ContentType,
		Tags:        stringifyTags(input.Tags),
		Attributes: SecretAttributes{
			Enabled:       enabled,
			NotBefore:     notBefore,
			Expires:       expires,
			Created:       now,
			Updated:       now,
			RecoveryLevel: "Recoverable+Purgeable",
		},
	}
	vaultSecrets := s.secrets[strings.ToLower(vaultName)]
	if vaultSecrets == nil {
		vaultSecrets = make(map[string]SecretBundle)
		s.secrets[strings.ToLower(vaultName)] = vaultSecrets
	}
	vaultVersions := s.secretVersions[strings.ToLower(vaultName)]
	if vaultVersions == nil {
		vaultVersions = make(map[string]map[string]SecretBundle)
		s.secretVersions[strings.ToLower(vaultName)] = vaultVersions
	}
	secretVersions := vaultVersions[strings.ToLower(name)]
	if secretVersions == nil {
		secretVersions = make(map[string]SecretBundle)
		vaultVersions[strings.ToLower(name)] = secretVersions
	}
	vaultSecrets[strings.ToLower(name)] = bundle
	secretVersions[version] = bundle
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, bundle)
}

func (s *KeyVaultService) getSecret(vaultName, name, version string) (*service.Response, error) {
	s.mu.RLock()
	vaultKey := strings.ToLower(vaultName)
	secretKey := strings.ToLower(name)
	var bundle SecretBundle
	var ok bool
	if version == "" {
		bundle, ok = s.secrets[vaultKey][secretKey]
	} else {
		bundle, ok = s.secretVersions[vaultKey][secretKey][version]
	}
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "SecretNotFound", fmt.Sprintf("Secret %q could not be found.", name))
	}
	if !bundle.Attributes.Enabled {
		return azurearm.ErrorResponse(http.StatusForbidden, "Forbidden", fmt.Sprintf("Secret %q is disabled.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, bundle)
}

func (s *KeyVaultService) listSecrets(vaultName string, req *http.Request) (*service.Response, error) {
	s.mu.RLock()
	values := make([]SecretListItem, 0, len(s.secrets[strings.ToLower(vaultName)]))
	for _, bundle := range s.secrets[strings.ToLower(vaultName)] {
		values = append(values, SecretListItem{
			ID:          secretBaseIDFromBundle(bundle),
			KID:         bundle.KID,
			Managed:     bundle.Managed,
			Attributes:  bundle.Attributes,
			ContentType: bundle.ContentType,
			Tags:        bundle.Tags,
		})
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	options, invalidResp, err := keyVaultListOptionsFromRequest(req)
	if invalidResp != nil || err != nil {
		return invalidResp, err
	}
	return azurearm.JSONResponse(http.StatusOK, keyVaultListPage(req, values, options))
}

func (s *KeyVaultService) listSecretVersions(vaultName, name string, req *http.Request) (*service.Response, error) {
	s.mu.RLock()
	versionMap := s.secretVersions[strings.ToLower(vaultName)][strings.ToLower(name)]
	if len(versionMap) == 0 {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "SecretNotFound", fmt.Sprintf("Secret %q could not be found.", name))
	}
	values := make([]SecretListItem, 0, len(versionMap))
	for _, bundle := range versionMap {
		values = append(values, SecretListItem{
			ID:          bundle.ID,
			KID:         bundle.KID,
			Managed:     bundle.Managed,
			Attributes:  bundle.Attributes,
			ContentType: bundle.ContentType,
			Tags:        bundle.Tags,
		})
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	options, invalidResp, err := keyVaultListOptionsFromRequest(req)
	if invalidResp != nil || err != nil {
		return invalidResp, err
	}
	return azurearm.JSONResponse(http.StatusOK, keyVaultListPage(req, values, options))
}

func (s *KeyVaultService) updateSecretProperties(vaultName, name, version string, body []byte) (*service.Response, error) {
	var input struct {
		Attributes  map[string]any `json:"attributes"`
		ContentType string         `json:"contentType"`
		Tags        map[string]any `json:"tags"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The request content was invalid.")
		}
	}

	vaultKey := strings.ToLower(vaultName)
	secretKey := strings.ToLower(name)
	now := time.Now().UTC().Unix()

	s.mu.Lock()
	defer s.mu.Unlock()

	bundle, ok := s.secretVersions[vaultKey][secretKey][version]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "SecretNotFound", fmt.Sprintf("Secret version %q could not be found.", version))
	}
	if input.ContentType != "" {
		bundle.ContentType = input.ContentType
	}
	if input.Tags != nil {
		bundle.Tags = stringifyTags(input.Tags)
	}
	if input.Attributes != nil {
		if value, ok := input.Attributes["enabled"].(bool); ok {
			bundle.Attributes.Enabled = value
		}
		if value, ok := int64MapValue(input.Attributes, "nbf"); ok {
			bundle.Attributes.NotBefore = value
		}
		if value, ok := int64MapValue(input.Attributes, "exp"); ok {
			bundle.Attributes.Expires = value
		}
	}
	bundle.Attributes.Updated = now
	s.secretVersions[vaultKey][secretKey][version] = bundle
	if latest, ok := s.secrets[vaultKey][secretKey]; ok && strings.HasSuffix(latest.ID, "/"+version) {
		s.secrets[vaultKey][secretKey] = bundle
	}

	return azurearm.JSONResponse(http.StatusOK, bundle)
}

func (s *KeyVaultService) backupSecret(vaultName, name string) (*service.Response, error) {
	vaultKey := strings.ToLower(vaultName)
	secretKey := strings.ToLower(name)

	s.mu.RLock()
	latest, ok := s.secrets[vaultKey][secretKey]
	versionMap := cloneSecretVersions(s.secretVersions[vaultKey][secretKey])
	s.mu.RUnlock()
	if !ok || len(versionMap) == 0 {
		return azurearm.ErrorResponse(http.StatusNotFound, "SecretNotFound", fmt.Sprintf("Secret %q could not be found.", name))
	}

	data, err := gojson.Marshal(secretBackupBlob{
		Vault:    vaultName,
		Name:     name,
		Latest:   latest,
		Versions: versionMap,
	})
	if err != nil {
		return nil, err
	}
	return azurearm.JSONResponse(http.StatusOK, map[string]string{
		"value": base64.RawURLEncoding.EncodeToString(data),
	})
}

func (s *KeyVaultService) restoreSecret(vaultName string, body []byte) (*service.Response, error) {
	var input struct {
		Value string `json:"value"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The request content was invalid.")
		}
	}
	if input.Value == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The secret backup value is required.")
	}

	data, err := decodeSecretBackupValue(input.Value)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The secret backup value must be base64url encoded.")
	}
	backup, ok := parseSecretBackupBlob(data)
	if !ok {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The secret backup value was invalid.")
	}
	if backup.Latest.ID == "" {
		backup.Latest = latestSecretBundle(backup.Versions)
	}

	vaultKey := strings.ToLower(vaultName)
	secretKey := strings.ToLower(backup.Name)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.secrets[vaultKey][secretKey]; exists {
		return azurearm.ErrorResponse(http.StatusConflict, "Conflict", fmt.Sprintf("Secret %q already exists.", backup.Name))
	}
	if s.secrets[vaultKey] == nil {
		s.secrets[vaultKey] = make(map[string]SecretBundle)
	}
	if s.secretVersions[vaultKey] == nil {
		s.secretVersions[vaultKey] = make(map[string]map[string]SecretBundle)
	}
	s.secretVersions[vaultKey][secretKey] = make(map[string]SecretBundle, len(backup.Versions))
	for version, bundle := range backup.Versions {
		if version == "" {
			version = versionFromSecretID(bundle.ID)
		}
		s.secretVersions[vaultKey][secretKey][version] = rewriteSecretBundleForVault(bundle, vaultName, backup.Name)
	}
	latestVersion := versionFromSecretID(backup.Latest.ID)
	latest := s.secretVersions[vaultKey][secretKey][latestVersion]
	if latest.ID == "" {
		latest = rewriteSecretBundleForVault(latestSecretBundle(backup.Versions), vaultName, backup.Name)
	}
	s.secrets[vaultKey][secretKey] = latest
	if s.deletedSecrets[vaultKey] != nil {
		delete(s.deletedSecrets[vaultKey], secretKey)
	}

	return azurearm.JSONResponse(http.StatusOK, latest)
}

func (s *KeyVaultService) deleteSecret(vaultName, name string) (*service.Response, error) {
	vaultKey := strings.ToLower(vaultName)
	secretKey := strings.ToLower(name)
	now := time.Now().UTC().Unix()

	s.mu.Lock()
	bundle, ok := s.secrets[vaultKey][secretKey]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "SecretNotFound", fmt.Sprintf("Secret %q could not be found.", name))
	}
	versions := cloneSecretVersions(s.secretVersions[vaultKey][secretKey])
	if len(versions) == 0 {
		versions[versionFromSecretID(bundle.ID)] = bundle
	}
	deleted := DeletedSecretBundle{
		ID:                 bundle.ID,
		KID:                bundle.KID,
		Managed:            bundle.Managed,
		PreviousVersion:    bundle.PreviousVersion,
		RecoveryID:         "https://" + vaultName + ".vault.azure.net/deletedsecrets/" + name,
		DeletedDate:        now,
		ScheduledPurgeDate: now + int64(90*24*time.Hour/time.Second),
		Attributes:         bundle.Attributes,
		ContentType:        bundle.ContentType,
		Tags:               bundle.Tags,
	}
	if s.deletedSecrets[vaultKey] == nil {
		s.deletedSecrets[vaultKey] = make(map[string]deletedSecretState)
	}
	s.deletedSecrets[vaultKey][secretKey] = deletedSecretState{
		Bundle:   deleted,
		Latest:   bundle,
		Versions: versions,
	}
	delete(s.secrets[vaultKey], secretKey)
	if s.secretVersions[vaultKey] != nil {
		delete(s.secretVersions[vaultKey], secretKey)
	}
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, deleted)
}

func (s *KeyVaultService) getDeletedSecret(vaultName, name string) (*service.Response, error) {
	s.mu.RLock()
	state, ok := s.deletedSecrets[strings.ToLower(vaultName)][strings.ToLower(name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "SecretNotFound", fmt.Sprintf("Deleted secret %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, state.Bundle)
}

func (s *KeyVaultService) listDeletedSecrets(vaultName string, req *http.Request) (*service.Response, error) {
	s.mu.RLock()
	values := make([]DeletedSecretBundle, 0, len(s.deletedSecrets[strings.ToLower(vaultName)]))
	for _, state := range s.deletedSecrets[strings.ToLower(vaultName)] {
		values = append(values, state.Bundle)
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].RecoveryID < values[j].RecoveryID })
	options, invalidResp, err := keyVaultListOptionsFromRequest(req)
	if invalidResp != nil || err != nil {
		return invalidResp, err
	}
	return azurearm.JSONResponse(http.StatusOK, keyVaultListPage(req, values, options))
}

type keyVaultListOptions struct {
	MaxResults int
	Skip       int
}

func keyVaultListOptionsFromRequest(req *http.Request) (keyVaultListOptions, *service.Response, error) {
	options := keyVaultListOptions{MaxResults: 25}
	query := req.URL.Query()
	if raw := query.Get("maxresults"); raw != "" {
		maxResults, err := strconv.Atoi(raw)
		if err != nil || maxResults < 1 || maxResults > 25 {
			resp, respErr := azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The maxresults query parameter must be between 1 and 25.")
			return options, resp, respErr
		}
		options.MaxResults = maxResults
	}
	if raw := query.Get("$skiptoken"); raw != "" {
		skip, err := strconv.Atoi(raw)
		if err != nil || skip < 0 {
			resp, respErr := azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The skip token was invalid.")
			return options, resp, respErr
		}
		options.Skip = skip
	}
	return options, nil, nil
}

func keyVaultListPage[T any](req *http.Request, values []T, options keyVaultListOptions) map[string]any {
	start := options.Skip
	if start > len(values) {
		start = len(values)
	}
	end := start + options.MaxResults
	if end > len(values) {
		end = len(values)
	}
	var nextLink any
	if end < len(values) {
		nextLink = keyVaultListNextLink(req, end)
	}
	return map[string]any{
		"value":    values[start:end],
		"nextLink": nextLink,
	}
}

func keyVaultListNextLink(req *http.Request, skip int) string {
	next := *req.URL
	query := next.Query()
	query.Set("$skiptoken", strconv.Itoa(skip))
	next.RawQuery = query.Encode()
	return next.String()
}

func (s *KeyVaultService) recoverDeletedSecret(vaultName, name string) (*service.Response, error) {
	vaultKey := strings.ToLower(vaultName)
	secretKey := strings.ToLower(name)

	s.mu.Lock()
	state, ok := s.deletedSecrets[vaultKey][secretKey]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "SecretNotFound", fmt.Sprintf("Deleted secret %q could not be found.", name))
	}
	if s.secrets[vaultKey] == nil {
		s.secrets[vaultKey] = make(map[string]SecretBundle)
	}
	if s.secretVersions[vaultKey] == nil {
		s.secretVersions[vaultKey] = make(map[string]map[string]SecretBundle)
	}
	s.secrets[vaultKey][secretKey] = state.Latest
	s.secretVersions[vaultKey][secretKey] = cloneSecretVersions(state.Versions)
	delete(s.deletedSecrets[vaultKey], secretKey)
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, state.Latest)
}

func (s *KeyVaultService) purgeDeletedSecret(vaultName, name string) (*service.Response, error) {
	vaultKey := strings.ToLower(vaultName)
	secretKey := strings.ToLower(name)

	s.mu.Lock()
	if _, ok := s.deletedSecrets[vaultKey][secretKey]; !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "SecretNotFound", fmt.Sprintf("Deleted secret %q could not be found.", name))
	}
	delete(s.deletedSecrets[vaultKey], secretKey)
	if s.secrets[vaultKey] != nil {
		delete(s.secrets[vaultKey], secretKey)
	}
	if s.secretVersions[vaultKey] != nil {
		delete(s.secretVersions[vaultKey], secretKey)
	}
	s.mu.Unlock()

	return &service.Response{StatusCode: http.StatusNoContent}, nil
}

func (s *KeyVaultService) handleCertificates(ctx *service.RequestContext, vaultName string) (*service.Response, error) {
	parts := splitPath(ctx.RawRequest.URL.EscapedPath())
	if len(parts) == 0 || !strings.EqualFold(parts[0], "certificates") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The key vault certificate route is not implemented.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPost:
		if len(parts) == 2 && strings.EqualFold(parts[1], "restore") {
			return s.restoreCertificate(vaultName, ctx.Body)
		}
		if len(parts) == 3 && strings.EqualFold(parts[2], "create") {
			return s.createCertificate(vaultName, parts[1], ctx.Body)
		}
		if len(parts) == 3 && strings.EqualFold(parts[2], "import") {
			return s.importCertificate(vaultName, parts[1], ctx.Body)
		}
		if len(parts) == 3 && strings.EqualFold(parts[2], "backup") {
			return s.backupCertificate(vaultName, parts[1])
		}
	case http.MethodGet:
		if len(parts) == 1 {
			return s.listCertificates(vaultName, ctx.RawRequest)
		}
		if len(parts) == 3 && strings.EqualFold(parts[2], "policy") {
			return s.getCertificatePolicy(vaultName, parts[1])
		}
		if len(parts) == 3 && strings.EqualFold(parts[2], "pending") {
			return s.getCertificateOperation(vaultName, parts[1])
		}
		if len(parts) == 3 && strings.EqualFold(parts[2], "versions") {
			return s.listCertificateVersions(vaultName, parts[1], ctx.RawRequest)
		}
		if len(parts) == 2 || len(parts) == 3 {
			version := ""
			if len(parts) == 3 {
				version = parts[2]
			}
			return s.getCertificate(vaultName, parts[1], version)
		}
	case http.MethodPatch:
		if len(parts) == 3 && strings.EqualFold(parts[2], "policy") {
			return s.updateCertificatePolicy(vaultName, parts[1], ctx.Body)
		}
		if len(parts) == 3 && !strings.EqualFold(parts[2], "versions") {
			return s.updateCertificate(vaultName, parts[1], parts[2], ctx.Body)
		}
	case http.MethodDelete:
		if len(parts) == 2 {
			return s.deleteCertificate(vaultName, parts[1])
		}
	}
	return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
}

func (s *KeyVaultService) handleDeletedCertificates(ctx *service.RequestContext, vaultName string) (*service.Response, error) {
	parts := splitPath(ctx.RawRequest.URL.EscapedPath())
	if len(parts) == 0 || !strings.EqualFold(parts[0], "deletedcertificates") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The deleted certificate route is not implemented.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodGet:
		if len(parts) == 1 {
			return s.listDeletedCertificates(vaultName, ctx.RawRequest)
		}
		if len(parts) == 2 {
			return s.getDeletedCertificate(vaultName, parts[1])
		}
	case http.MethodPost:
		if len(parts) == 3 && strings.EqualFold(parts[2], "recover") {
			return s.recoverDeletedCertificate(vaultName, parts[1])
		}
	case http.MethodDelete:
		if len(parts) == 2 {
			return s.purgeDeletedCertificate(vaultName, parts[1])
		}
	}
	return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
}

func (s *KeyVaultService) importCertificate(vaultName, name string, body []byte) (*service.Response, error) {
	var input struct {
		Value             string         `json:"value"`
		Attributes        map[string]any `json:"attributes"`
		Policy            map[string]any `json:"policy"`
		PreserveCertOrder bool           `json:"preserveCertOrder"`
		Tags              map[string]any `json:"tags"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The request content was invalid.")
		}
	}
	if input.Value == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The certificate value is required.")
	}

	now := time.Now().UTC().Unix()
	enabled := true
	if value, ok := input.Attributes["enabled"].(bool); ok {
		enabled = value
	}
	notBefore, _ := int64MapValue(input.Attributes, "nbf")
	expires, _ := int64MapValue(input.Attributes, "exp")
	policy := certificatePolicy(vaultName, name, input.Policy, enabled, now)
	contentType := certificatePolicyContentType(policy)
	tags := stringifyTags(input.Tags)

	s.mu.Lock()
	version := s.nextVersion()
	bundle := CertificateBundle{
		ID:                certificateID(vaultName, name, version),
		KID:               keyID(vaultName, name, version),
		SID:               secretID(vaultName, name, version),
		X5T:               certificateThumbprint(input.Value),
		CER:               input.Value,
		ContentType:       contentType,
		Policy:            policy,
		Tags:              tags,
		PreserveCertOrder: input.PreserveCertOrder,
		Attributes: CertificateAttributes{
			Enabled:       enabled,
			NotBefore:     notBefore,
			Expires:       expires,
			Created:       now,
			Updated:       now,
			RecoveryLevel: "Recoverable+Purgeable",
		},
	}
	vaultKey := strings.ToLower(vaultName)
	certKey := strings.ToLower(name)
	if s.certificates[vaultKey] == nil {
		s.certificates[vaultKey] = make(map[string]CertificateBundle)
	}
	if s.certVersions[vaultKey] == nil {
		s.certVersions[vaultKey] = make(map[string]map[string]CertificateBundle)
	}
	if s.certVersions[vaultKey][certKey] == nil {
		s.certVersions[vaultKey][certKey] = make(map[string]CertificateBundle)
	}
	s.certificates[vaultKey][certKey] = bundle
	s.certVersions[vaultKey][certKey][version] = bundle
	s.linkCertificateSecretLocked(vaultName, name, version, input.Value, contentType, tags, bundle.Attributes)
	s.linkCertificateKeyLocked(vaultName, name, version, policy, tags, bundle.Attributes)
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, bundle)
}

func (s *KeyVaultService) createCertificate(vaultName, name string, body []byte) (*service.Response, error) {
	var input struct {
		Attributes        map[string]any `json:"attributes"`
		Policy            map[string]any `json:"policy"`
		PreserveCertOrder bool           `json:"preserveCertOrder"`
		Tags              map[string]any `json:"tags"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The request content was invalid.")
		}
	}

	now := time.Now().UTC().Unix()
	enabled := true
	if value, ok := input.Attributes["enabled"].(bool); ok {
		enabled = value
	}
	notBefore, _ := int64MapValue(input.Attributes, "nbf")
	expires, _ := int64MapValue(input.Attributes, "exp")
	policy := certificatePolicy(vaultName, name, input.Policy, enabled, now)
	contentType := certificatePolicyContentType(policy)
	tags := stringifyTags(input.Tags)

	s.mu.Lock()
	version := s.nextVersion()
	certificateValue := base64.StdEncoding.EncodeToString([]byte("cloudmock-certificate:" + strings.ToLower(vaultName) + ":" + strings.ToLower(name) + ":" + version))
	bundle := CertificateBundle{
		ID:                certificateID(vaultName, name, version),
		KID:               keyID(vaultName, name, version),
		SID:               secretID(vaultName, name, version),
		X5T:               certificateThumbprint(certificateValue),
		CER:               certificateValue,
		ContentType:       contentType,
		Policy:            policy,
		Tags:              tags,
		PreserveCertOrder: input.PreserveCertOrder,
		Attributes: CertificateAttributes{
			Enabled:       enabled,
			NotBefore:     notBefore,
			Expires:       expires,
			Created:       now,
			Updated:       now,
			RecoveryLevel: "Recoverable+Purgeable",
		},
	}
	vaultKey := strings.ToLower(vaultName)
	certKey := strings.ToLower(name)
	if s.certificates[vaultKey] == nil {
		s.certificates[vaultKey] = make(map[string]CertificateBundle)
	}
	if s.certVersions[vaultKey] == nil {
		s.certVersions[vaultKey] = make(map[string]map[string]CertificateBundle)
	}
	if s.certVersions[vaultKey][certKey] == nil {
		s.certVersions[vaultKey][certKey] = make(map[string]CertificateBundle)
	}
	s.certificates[vaultKey][certKey] = bundle
	s.certVersions[vaultKey][certKey][version] = bundle
	s.linkCertificateSecretLocked(vaultName, name, version, certificateValue, contentType, tags, bundle.Attributes)
	s.linkCertificateKeyLocked(vaultName, name, version, policy, tags, bundle.Attributes)
	operation := CertificateOperation{
		ID:                    "https://" + vaultName + ".vault.azure.net/certificates/" + name + "/pending",
		Issuer:                certificatePolicyIssuer(policy),
		CSR:                   base64.StdEncoding.EncodeToString([]byte("cloudmock-csr:" + strings.ToLower(vaultName) + ":" + strings.ToLower(name) + ":" + version)),
		CancellationRequested: false,
		Status:                "completed",
		StatusDetails:         "Certificate operation completed by CloudMock.",
		RequestID:             fmt.Sprintf("%016x", s.nextID),
		Target:                bundle.ID,
		PreserveCertOrder:     input.PreserveCertOrder,
	}
	if s.certOperations[vaultKey] == nil {
		s.certOperations[vaultKey] = make(map[string]CertificateOperation)
	}
	s.certOperations[vaultKey][certKey] = operation
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusAccepted, operation)
}

func (s *KeyVaultService) getCertificateOperation(vaultName, name string) (*service.Response, error) {
	s.mu.RLock()
	operation, ok := s.certOperations[strings.ToLower(vaultName)][strings.ToLower(name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "CertificateOperationNotFound", fmt.Sprintf("Certificate operation for %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, operation)
}

func (s *KeyVaultService) backupCertificate(vaultName, name string) (*service.Response, error) {
	vaultKey := strings.ToLower(vaultName)
	certKey := strings.ToLower(name)

	s.mu.RLock()
	latest, ok := s.certificates[vaultKey][certKey]
	versionMap := cloneCertificateVersions(s.certVersions[vaultKey][certKey])
	s.mu.RUnlock()
	if !ok || len(versionMap) == 0 {
		return azurearm.ErrorResponse(http.StatusNotFound, "CertificateNotFound", fmt.Sprintf("Certificate %q could not be found.", name))
	}

	data, err := gojson.Marshal(certificateBackupBlob{
		Vault:    vaultName,
		Name:     name,
		Latest:   latest,
		Versions: versionMap,
	})
	if err != nil {
		return nil, err
	}
	return azurearm.JSONResponse(http.StatusOK, map[string]string{
		"value": base64.RawURLEncoding.EncodeToString(data),
	})
}

func (s *KeyVaultService) restoreCertificate(vaultName string, body []byte) (*service.Response, error) {
	var input struct {
		Value string `json:"value"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The request content was invalid.")
		}
	}
	if input.Value == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The certificate backup value is required.")
	}

	data, err := base64.RawURLEncoding.DecodeString(input.Value)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The certificate backup value must be base64url encoded.")
	}
	var backup certificateBackupBlob
	if err := gojson.Unmarshal(data, &backup); err != nil || backup.Name == "" || len(backup.Versions) == 0 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The certificate backup value was invalid.")
	}
	if backup.Latest.ID == "" {
		backup.Latest = latestCertificateBundle(backup.Versions)
	}

	vaultKey := strings.ToLower(vaultName)
	certKey := strings.ToLower(backup.Name)

	s.mu.Lock()
	if _, exists := s.certificates[vaultKey][certKey]; exists {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusConflict, "Conflict", fmt.Sprintf("Certificate %q already exists.", backup.Name))
	}
	if s.certificates[vaultKey] == nil {
		s.certificates[vaultKey] = make(map[string]CertificateBundle)
	}
	if s.certVersions[vaultKey] == nil {
		s.certVersions[vaultKey] = make(map[string]map[string]CertificateBundle)
	}
	s.certVersions[vaultKey][certKey] = make(map[string]CertificateBundle, len(backup.Versions))
	for version, bundle := range backup.Versions {
		if version == "" {
			version = versionFromSecretID(bundle.ID)
		}
		rewritten := rewriteCertificateBundleForVault(bundle, vaultName, backup.Name)
		s.certVersions[vaultKey][certKey][version] = rewritten
		s.linkCertificateSecretLocked(vaultName, backup.Name, version, rewritten.CER, rewritten.ContentType, rewritten.Tags, rewritten.Attributes)
		s.linkCertificateKeyLocked(vaultName, backup.Name, version, rewritten.Policy, rewritten.Tags, rewritten.Attributes)
	}
	latestVersion := versionFromSecretID(backup.Latest.ID)
	latest := s.certVersions[vaultKey][certKey][latestVersion]
	if latest.ID == "" {
		latest = rewriteCertificateBundleForVault(latestCertificateBundle(backup.Versions), vaultName, backup.Name)
		latestVersion = versionFromSecretID(latest.ID)
	}
	s.certificates[vaultKey][certKey] = latest
	if latest.ID != "" {
		version := versionFromSecretID(latest.ID)
		if latestVersion != "" {
			version = latestVersion
		}
		s.linkCertificateSecretLocked(vaultName, backup.Name, version, latest.CER, latest.ContentType, latest.Tags, latest.Attributes)
		s.linkCertificateKeyLocked(vaultName, backup.Name, version, latest.Policy, latest.Tags, latest.Attributes)
	}
	if s.deletedCerts[vaultKey] != nil {
		delete(s.deletedCerts[vaultKey], certKey)
	}
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, latest)
}

func (s *KeyVaultService) getCertificate(vaultName, name, version string) (*service.Response, error) {
	s.mu.RLock()
	vaultKey := strings.ToLower(vaultName)
	certKey := strings.ToLower(name)
	var bundle CertificateBundle
	var ok bool
	if version == "" {
		bundle, ok = s.certificates[vaultKey][certKey]
	} else {
		bundle, ok = s.certVersions[vaultKey][certKey][version]
	}
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "CertificateNotFound", fmt.Sprintf("Certificate %q could not be found.", name))
	}
	if !bundle.Attributes.Enabled {
		return azurearm.ErrorResponse(http.StatusForbidden, "Forbidden", fmt.Sprintf("Certificate %q is disabled.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, bundle)
}

func (s *KeyVaultService) listCertificates(vaultName string, req *http.Request) (*service.Response, error) {
	s.mu.RLock()
	values := make([]CertificateListItem, 0, len(s.certificates[strings.ToLower(vaultName)]))
	for _, bundle := range s.certificates[strings.ToLower(vaultName)] {
		values = append(values, CertificateListItem{
			ID:         certificateBaseIDFromBundle(bundle),
			X5T:        bundle.X5T,
			Attributes: bundle.Attributes,
			Tags:       bundle.Tags,
		})
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	options, invalidResp, err := keyVaultListOptionsFromRequest(req)
	if invalidResp != nil || err != nil {
		return invalidResp, err
	}
	return azurearm.JSONResponse(http.StatusOK, keyVaultListPage(req, values, options))
}

func (s *KeyVaultService) getCertificatePolicy(vaultName, name string) (*service.Response, error) {
	s.mu.RLock()
	bundle, ok := s.certificates[strings.ToLower(vaultName)][strings.ToLower(name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "CertificateNotFound", fmt.Sprintf("Certificate %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, bundle.Policy)
}

func (s *KeyVaultService) updateCertificatePolicy(vaultName, name string, body []byte) (*service.Response, error) {
	var input map[string]any
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The request content was invalid.")
		}
	}

	vaultKey := strings.ToLower(vaultName)
	certKey := strings.ToLower(name)
	now := time.Now().UTC().Unix()

	s.mu.Lock()
	defer s.mu.Unlock()

	bundle, ok := s.certificates[vaultKey][certKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "CertificateNotFound", fmt.Sprintf("Certificate %q could not be found.", name))
	}
	policy := cloneAnyMap(bundle.Policy)
	mergeAnyMap(policy, input)
	bundle.Policy = certificatePolicy(vaultName, name, policy, bundle.Attributes.Enabled, bundle.Attributes.Created)
	bundle.ContentType = certificatePolicyContentType(bundle.Policy)
	bundle.Attributes.Updated = now

	version := versionFromSecretID(bundle.ID)
	s.certificates[vaultKey][certKey] = bundle
	if s.certVersions[vaultKey][certKey] != nil {
		s.certVersions[vaultKey][certKey][version] = bundle
	}
	s.updateLinkedCertificateSecretLocked(vaultKey, certKey, version, bundle)
	s.updateLinkedCertificateKeyLocked(vaultKey, certKey, bundle)

	return azurearm.JSONResponse(http.StatusOK, bundle.Policy)
}

func (s *KeyVaultService) updateCertificate(vaultName, name, version string, body []byte) (*service.Response, error) {
	var input struct {
		Attributes map[string]any `json:"attributes"`
		Policy     map[string]any `json:"policy"`
		Tags       map[string]any `json:"tags"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The request content was invalid.")
		}
	}

	vaultKey := strings.ToLower(vaultName)
	certKey := strings.ToLower(name)
	now := time.Now().UTC().Unix()

	s.mu.Lock()
	defer s.mu.Unlock()

	bundle, ok := s.certVersions[vaultKey][certKey][version]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "CertificateNotFound", fmt.Sprintf("Certificate version %q could not be found.", version))
	}
	if input.Attributes != nil {
		if value, ok := input.Attributes["enabled"].(bool); ok {
			bundle.Attributes.Enabled = value
		}
		if value, ok := int64MapValue(input.Attributes, "nbf"); ok {
			bundle.Attributes.NotBefore = value
		}
		if value, ok := int64MapValue(input.Attributes, "exp"); ok {
			bundle.Attributes.Expires = value
		}
	}
	if input.Policy != nil {
		policy := cloneAnyMap(bundle.Policy)
		mergeAnyMap(policy, input.Policy)
		bundle.Policy = certificatePolicy(vaultName, name, policy, bundle.Attributes.Enabled, bundle.Attributes.Created)
		bundle.ContentType = certificatePolicyContentType(bundle.Policy)
	}
	if input.Tags != nil {
		bundle.Tags = stringifyTags(input.Tags)
	}
	bundle.Attributes.Updated = now

	s.certVersions[vaultKey][certKey][version] = bundle
	if latest, ok := s.certificates[vaultKey][certKey]; ok && strings.HasSuffix(latest.ID, "/"+version) {
		s.certificates[vaultKey][certKey] = bundle
	}
	s.updateLinkedCertificateSecretLocked(vaultKey, certKey, version, bundle)
	s.updateLinkedCertificateKeyLocked(vaultKey, certKey, bundle)

	return azurearm.JSONResponse(http.StatusOK, bundle)
}

func (s *KeyVaultService) listCertificateVersions(vaultName, name string, req *http.Request) (*service.Response, error) {
	s.mu.RLock()
	versionMap := s.certVersions[strings.ToLower(vaultName)][strings.ToLower(name)]
	if len(versionMap) == 0 {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "CertificateNotFound", fmt.Sprintf("Certificate %q could not be found.", name))
	}
	values := make([]CertificateListItem, 0, len(versionMap))
	for _, bundle := range versionMap {
		values = append(values, CertificateListItem{
			ID:         bundle.ID,
			X5T:        bundle.X5T,
			Attributes: bundle.Attributes,
			Tags:       bundle.Tags,
		})
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	options, invalidResp, err := keyVaultListOptionsFromRequest(req)
	if invalidResp != nil || err != nil {
		return invalidResp, err
	}
	return azurearm.JSONResponse(http.StatusOK, keyVaultListPage(req, values, options))
}

func (s *KeyVaultService) deleteCertificate(vaultName, name string) (*service.Response, error) {
	vaultKey := strings.ToLower(vaultName)
	certKey := strings.ToLower(name)
	now := time.Now().UTC().Unix()

	s.mu.Lock()
	bundle, ok := s.certificates[vaultKey][certKey]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "CertificateNotFound", fmt.Sprintf("Certificate %q could not be found.", name))
	}
	versions := cloneCertificateVersions(s.certVersions[vaultKey][certKey])
	if len(versions) == 0 {
		versions[versionFromSecretID(bundle.ID)] = bundle
	}
	deleted := DeletedCertificateBundle{
		RecoveryID:         "https://" + vaultName + ".vault.azure.net/deletedcertificates/" + name,
		DeletedDate:        now,
		ScheduledPurgeDate: now + int64(90*24*time.Hour/time.Second),
		ID:                 bundle.ID,
		KID:                bundle.KID,
		SID:                bundle.SID,
		X5T:                bundle.X5T,
		CER:                bundle.CER,
		Attributes:         bundle.Attributes,
		Policy:             bundle.Policy,
		ContentType:        bundle.ContentType,
		Tags:               bundle.Tags,
	}
	if s.deletedCerts[vaultKey] == nil {
		s.deletedCerts[vaultKey] = make(map[string]deletedCertificateState)
	}
	s.deletedCerts[vaultKey][certKey] = deletedCertificateState{
		Bundle:   deleted,
		Latest:   bundle,
		Versions: versions,
	}
	delete(s.certificates[vaultKey], certKey)
	if s.certVersions[vaultKey] != nil {
		delete(s.certVersions[vaultKey], certKey)
	}
	if s.secrets[vaultKey] != nil {
		delete(s.secrets[vaultKey], certKey)
	}
	if s.secretVersions[vaultKey] != nil {
		delete(s.secretVersions[vaultKey], certKey)
	}
	if s.keys[vaultKey] != nil {
		delete(s.keys[vaultKey], certKey)
	}
	if s.keyVersions[vaultKey] != nil {
		delete(s.keyVersions[vaultKey], certKey)
	}
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, deleted)
}

func (s *KeyVaultService) getDeletedCertificate(vaultName, name string) (*service.Response, error) {
	s.mu.RLock()
	state, ok := s.deletedCerts[strings.ToLower(vaultName)][strings.ToLower(name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "CertificateNotFound", fmt.Sprintf("Deleted certificate %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, state.Bundle)
}

func (s *KeyVaultService) listDeletedCertificates(vaultName string, req *http.Request) (*service.Response, error) {
	s.mu.RLock()
	values := make([]DeletedCertificateBundle, 0, len(s.deletedCerts[strings.ToLower(vaultName)]))
	for _, state := range s.deletedCerts[strings.ToLower(vaultName)] {
		values = append(values, state.Bundle)
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].RecoveryID < values[j].RecoveryID })
	options, invalidResp, err := keyVaultListOptionsFromRequest(req)
	if invalidResp != nil || err != nil {
		return invalidResp, err
	}
	return azurearm.JSONResponse(http.StatusOK, keyVaultListPage(req, values, options))
}

func (s *KeyVaultService) recoverDeletedCertificate(vaultName, name string) (*service.Response, error) {
	vaultKey := strings.ToLower(vaultName)
	certKey := strings.ToLower(name)

	s.mu.Lock()
	state, ok := s.deletedCerts[vaultKey][certKey]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "CertificateNotFound", fmt.Sprintf("Deleted certificate %q could not be found.", name))
	}
	if s.certificates[vaultKey] == nil {
		s.certificates[vaultKey] = make(map[string]CertificateBundle)
	}
	if s.certVersions[vaultKey] == nil {
		s.certVersions[vaultKey] = make(map[string]map[string]CertificateBundle)
	}
	s.certificates[vaultKey][certKey] = state.Latest
	s.certVersions[vaultKey][certKey] = cloneCertificateVersions(state.Versions)
	for _, bundle := range state.Versions {
		version := versionFromSecretID(bundle.ID)
		s.linkCertificateSecretLocked(vaultName, name, version, bundle.CER, bundle.ContentType, bundle.Tags, bundle.Attributes)
		s.linkCertificateKeyLocked(vaultName, name, version, bundle.Policy, bundle.Tags, bundle.Attributes)
	}
	version := versionFromSecretID(state.Latest.ID)
	s.linkCertificateKeyLocked(vaultName, name, version, state.Latest.Policy, state.Latest.Tags, state.Latest.Attributes)
	delete(s.deletedCerts[vaultKey], certKey)
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, state.Latest)
}

func (s *KeyVaultService) purgeDeletedCertificate(vaultName, name string) (*service.Response, error) {
	vaultKey := strings.ToLower(vaultName)
	certKey := strings.ToLower(name)

	s.mu.Lock()
	if _, ok := s.deletedCerts[vaultKey][certKey]; !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "CertificateNotFound", fmt.Sprintf("Deleted certificate %q could not be found.", name))
	}
	delete(s.deletedCerts[vaultKey], certKey)
	if s.certificates[vaultKey] != nil {
		delete(s.certificates[vaultKey], certKey)
	}
	if s.certVersions[vaultKey] != nil {
		delete(s.certVersions[vaultKey], certKey)
	}
	s.mu.Unlock()

	return &service.Response{StatusCode: http.StatusNoContent}, nil
}

func (s *KeyVaultService) handleKeys(ctx *service.RequestContext, vaultName string) (*service.Response, error) {
	parts := splitPath(ctx.RawRequest.URL.EscapedPath())
	if len(parts) == 0 || !strings.EqualFold(parts[0], "keys") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The key vault key route is not implemented.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodGet:
		if len(parts) == 1 {
			return s.listKeys(vaultName, ctx.RawRequest)
		}
		if len(parts) == 3 && strings.EqualFold(parts[2], "versions") {
			return s.listKeyVersions(vaultName, parts[1], ctx.RawRequest)
		}
		if len(parts) == 2 || len(parts) == 3 {
			version := ""
			if len(parts) == 3 {
				version = parts[2]
			}
			return s.getKey(vaultName, parts[1], version)
		}
	case http.MethodPost:
		switch {
		case len(parts) == 2 && strings.EqualFold(parts[1], "restore"):
			return s.restoreKey(vaultName, ctx.Body)
		case len(parts) == 3 && strings.EqualFold(parts[2], "create"):
			return s.createKey(vaultName, parts[1], ctx.Body)
		case len(parts) == 3 && strings.EqualFold(parts[2], "backup"):
			return s.backupKey(vaultName, parts[1])
		case len(parts) == 4 && strings.EqualFold(parts[3], "encrypt"):
			return s.encryptWithKey(vaultName, parts[1], parts[2], ctx.Body)
		case len(parts) == 4 && strings.EqualFold(parts[3], "decrypt"):
			return s.decryptWithKey(vaultName, parts[1], parts[2], ctx.Body)
		case len(parts) == 4 && strings.EqualFold(parts[3], "sign"):
			return s.signWithKey(vaultName, parts[1], parts[2], ctx.Body)
		case len(parts) == 4 && strings.EqualFold(parts[3], "verify"):
			return s.verifyWithKey(vaultName, parts[1], parts[2], ctx.Body)
		case len(parts) == 4 && strings.EqualFold(parts[3], "wrapkey"):
			return s.wrapWithKey(vaultName, parts[1], parts[2], ctx.Body)
		case len(parts) == 4 && strings.EqualFold(parts[3], "unwrapkey"):
			return s.unwrapWithKey(vaultName, parts[1], parts[2], ctx.Body)
		}
	case http.MethodPatch:
		if len(parts) == 3 {
			return s.updateKey(vaultName, parts[1], parts[2], ctx.Body)
		}
	case http.MethodDelete:
		if len(parts) == 2 {
			return s.deleteKey(vaultName, parts[1])
		}
	}
	return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
}

func (s *KeyVaultService) handleDeletedKeys(ctx *service.RequestContext, vaultName string) (*service.Response, error) {
	parts := splitPath(ctx.RawRequest.URL.EscapedPath())
	if len(parts) == 0 || !strings.EqualFold(parts[0], "deletedkeys") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The deleted key route is not implemented.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodGet:
		if len(parts) == 1 {
			return s.listDeletedKeys(vaultName, ctx.RawRequest)
		}
		if len(parts) == 2 {
			return s.getDeletedKey(vaultName, parts[1])
		}
	case http.MethodPost:
		if len(parts) == 3 && strings.EqualFold(parts[2], "recover") {
			return s.recoverDeletedKey(vaultName, parts[1])
		}
	case http.MethodDelete:
		if len(parts) == 2 {
			return s.purgeDeletedKey(vaultName, parts[1])
		}
	}
	return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
}

func (s *KeyVaultService) createKey(vaultName, name string, body []byte) (*service.Response, error) {
	var input struct {
		KTY        string         `json:"kty"`
		KeyOps     []string       `json:"key_ops"`
		KeySize    int            `json:"key_size"`
		Attributes map[string]any `json:"attributes"`
		Tags       map[string]any `json:"tags"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The request content was invalid.")
		}
	}
	if input.KTY == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The key type kty is required.")
	}
	if len(input.KeyOps) == 0 {
		input.KeyOps = []string{"encrypt", "decrypt"}
	}

	now := time.Now().UTC().Unix()
	enabled := true
	if value, ok := input.Attributes["enabled"].(bool); ok {
		enabled = value
	}
	notBefore, _ := int64MapValue(input.Attributes, "nbf")
	expires, _ := int64MapValue(input.Attributes, "exp")

	s.mu.Lock()
	version := s.nextVersion()
	bundle := KeyBundle{
		Key: JsonWebKey{
			KID:    keyID(vaultName, name, version),
			KTY:    input.KTY,
			KeyOps: append([]string(nil), input.KeyOps...),
			N:      base64.RawURLEncoding.EncodeToString([]byte("cloudmock:" + strings.ToLower(vaultName) + ":" + strings.ToLower(name) + ":n")),
			E:      "AQAB",
		},
		Tags: stringifyTags(input.Tags),
		Attributes: KeyAttributes{
			Enabled:       enabled,
			NotBefore:     notBefore,
			Expires:       expires,
			Created:       now,
			Updated:       now,
			RecoveryLevel: "Recoverable+Purgeable",
		},
	}
	vaultKeys := s.keys[strings.ToLower(vaultName)]
	if vaultKeys == nil {
		vaultKeys = make(map[string]KeyBundle)
		s.keys[strings.ToLower(vaultName)] = vaultKeys
	}
	vaultKey := strings.ToLower(vaultName)
	keyKey := strings.ToLower(name)
	if s.keyVersions[vaultKey] == nil {
		s.keyVersions[vaultKey] = make(map[string]map[string]KeyBundle)
	}
	if s.keyVersions[vaultKey][keyKey] == nil {
		s.keyVersions[vaultKey][keyKey] = make(map[string]KeyBundle)
	}
	vaultKeys[keyKey] = bundle
	s.keyVersions[vaultKey][keyKey][version] = bundle
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, bundle)
}

func (s *KeyVaultService) getKey(vaultName, name, version string) (*service.Response, error) {
	bundle, ok := s.keyBundle(vaultName, name, version)
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "KeyNotFound", fmt.Sprintf("Key %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, bundle)
}

func (s *KeyVaultService) listKeys(vaultName string, req *http.Request) (*service.Response, error) {
	vaultKey := strings.ToLower(vaultName)

	s.mu.RLock()
	values := make([]KeyListItem, 0, len(s.keys[vaultKey]))
	for _, bundle := range s.keys[vaultKey] {
		values = append(values, KeyListItem{
			KID:        keyBaseIDFromBundle(bundle),
			Managed:    bundle.Managed,
			Attributes: bundle.Attributes,
			Tags:       bundle.Tags,
		})
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].KID < values[j].KID })
	options, invalidResp, err := keyVaultListOptionsFromRequest(req)
	if invalidResp != nil || err != nil {
		return invalidResp, err
	}
	return azurearm.JSONResponse(http.StatusOK, keyVaultListPage(req, values, options))
}

func (s *KeyVaultService) listKeyVersions(vaultName, name string, req *http.Request) (*service.Response, error) {
	vaultKey := strings.ToLower(vaultName)
	keyKey := strings.ToLower(name)

	s.mu.RLock()
	versionMap := s.keyVersions[vaultKey][keyKey]
	if len(versionMap) == 0 {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "KeyNotFound", fmt.Sprintf("Key %q could not be found.", name))
	}
	values := make([]KeyListItem, 0, len(versionMap))
	for _, bundle := range versionMap {
		values = append(values, KeyListItem{
			KID:        bundle.Key.KID,
			Managed:    bundle.Managed,
			Attributes: bundle.Attributes,
			Tags:       bundle.Tags,
		})
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].KID < values[j].KID })
	options, invalidResp, err := keyVaultListOptionsFromRequest(req)
	if invalidResp != nil || err != nil {
		return invalidResp, err
	}
	return azurearm.JSONResponse(http.StatusOK, keyVaultListPage(req, values, options))
}

func (s *KeyVaultService) updateKey(vaultName, name, version string, body []byte) (*service.Response, error) {
	var input struct {
		KeyOps     []string       `json:"key_ops"`
		Attributes map[string]any `json:"attributes"`
		Tags       map[string]any `json:"tags"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The request content was invalid.")
		}
	}

	vaultKey := strings.ToLower(vaultName)
	keyKey := strings.ToLower(name)
	now := time.Now().UTC().Unix()

	s.mu.Lock()
	defer s.mu.Unlock()

	bundle, ok := s.keyVersions[vaultKey][keyKey][version]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "KeyNotFound", fmt.Sprintf("Key version %q could not be found.", version))
	}
	if input.KeyOps != nil {
		bundle.Key.KeyOps = append([]string(nil), input.KeyOps...)
	}
	if input.Attributes != nil {
		if value, ok := input.Attributes["enabled"].(bool); ok {
			bundle.Attributes.Enabled = value
		}
		if value, ok := int64MapValue(input.Attributes, "nbf"); ok {
			bundle.Attributes.NotBefore = value
		}
		if value, ok := int64MapValue(input.Attributes, "exp"); ok {
			bundle.Attributes.Expires = value
		}
	}
	if input.Tags != nil {
		bundle.Tags = stringifyTags(input.Tags)
	}
	bundle.Attributes.Updated = now

	s.keyVersions[vaultKey][keyKey][version] = bundle
	if latest, ok := s.keys[vaultKey][keyKey]; ok && latest.Key.KID == bundle.Key.KID {
		s.keys[vaultKey][keyKey] = bundle
	}

	return azurearm.JSONResponse(http.StatusOK, bundle)
}

func (s *KeyVaultService) backupKey(vaultName, name string) (*service.Response, error) {
	vaultKey := strings.ToLower(vaultName)
	keyKey := strings.ToLower(name)

	s.mu.RLock()
	latest, ok := s.keys[vaultKey][keyKey]
	versionMap := cloneKeyVersions(s.keyVersions[vaultKey][keyKey])
	s.mu.RUnlock()
	if !ok || len(versionMap) == 0 {
		return azurearm.ErrorResponse(http.StatusNotFound, "KeyNotFound", fmt.Sprintf("Key %q could not be found.", name))
	}

	data, err := gojson.Marshal(keyBackupBlob{
		Vault:    vaultName,
		Name:     name,
		Latest:   latest,
		Versions: versionMap,
	})
	if err != nil {
		return nil, err
	}
	return azurearm.JSONResponse(http.StatusOK, map[string]string{
		"value": base64.RawURLEncoding.EncodeToString(data),
	})
}

func (s *KeyVaultService) restoreKey(vaultName string, body []byte) (*service.Response, error) {
	var input struct {
		Value string `json:"value"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The request content was invalid.")
		}
	}
	if input.Value == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The key backup value is required.")
	}

	data, err := base64.RawURLEncoding.DecodeString(input.Value)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The key backup value must be base64url encoded.")
	}
	var backup keyBackupBlob
	if err := gojson.Unmarshal(data, &backup); err != nil || backup.Name == "" || len(backup.Versions) == 0 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The key backup value was invalid.")
	}
	if backup.Latest.Key.KID == "" {
		backup.Latest = latestKeyBundle(backup.Versions)
	}

	vaultKey := strings.ToLower(vaultName)
	keyKey := strings.ToLower(backup.Name)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.keys[vaultKey][keyKey]; exists {
		return azurearm.ErrorResponse(http.StatusConflict, "Conflict", fmt.Sprintf("Key %q already exists.", backup.Name))
	}
	if s.keys[vaultKey] == nil {
		s.keys[vaultKey] = make(map[string]KeyBundle)
	}
	if s.keyVersions[vaultKey] == nil {
		s.keyVersions[vaultKey] = make(map[string]map[string]KeyBundle)
	}
	s.keyVersions[vaultKey][keyKey] = make(map[string]KeyBundle, len(backup.Versions))
	for version, bundle := range backup.Versions {
		s.keyVersions[vaultKey][keyKey][version] = rewriteKeyBundleForVault(bundle, vaultName, backup.Name)
	}
	latestVersion := versionFromSecretID(backup.Latest.Key.KID)
	latest := s.keyVersions[vaultKey][keyKey][latestVersion]
	if latest.Key.KID == "" {
		latest = rewriteKeyBundleForVault(latestKeyBundle(backup.Versions), vaultName, backup.Name)
	}
	s.keys[vaultKey][keyKey] = latest
	if s.deletedKeys[vaultKey] != nil {
		delete(s.deletedKeys[vaultKey], keyKey)
	}

	return azurearm.JSONResponse(http.StatusOK, latest)
}

func (s *KeyVaultService) deleteKey(vaultName, name string) (*service.Response, error) {
	vaultKey := strings.ToLower(vaultName)
	keyKey := strings.ToLower(name)
	now := time.Now().UTC().Unix()

	s.mu.Lock()
	latest, ok := s.keys[vaultKey][keyKey]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "KeyNotFound", fmt.Sprintf("Key %q could not be found.", name))
	}
	versions := cloneKeyVersions(s.keyVersions[vaultKey][keyKey])
	if len(versions) == 0 {
		versions[versionFromSecretID(latest.Key.KID)] = latest
	}
	deleted := DeletedKeyBundle{
		RecoveryID:         "https://" + vaultName + ".vault.azure.net/deletedkeys/" + name,
		DeletedDate:        now,
		ScheduledPurgeDate: now + int64(90*24*time.Hour/time.Second),
		Key:                latest.Key,
		Managed:            latest.Managed,
		Attributes:         latest.Attributes,
		Tags:               latest.Tags,
	}
	if s.deletedKeys[vaultKey] == nil {
		s.deletedKeys[vaultKey] = make(map[string]deletedKeyState)
	}
	s.deletedKeys[vaultKey][keyKey] = deletedKeyState{
		Bundle:   deleted,
		Latest:   latest,
		Versions: versions,
	}
	delete(s.keys[vaultKey], keyKey)
	if s.keyVersions[vaultKey] != nil {
		delete(s.keyVersions[vaultKey], keyKey)
	}
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, deleted)
}

func (s *KeyVaultService) getDeletedKey(vaultName, name string) (*service.Response, error) {
	s.mu.RLock()
	state, ok := s.deletedKeys[strings.ToLower(vaultName)][strings.ToLower(name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "KeyNotFound", fmt.Sprintf("Deleted key %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, state.Bundle)
}

func (s *KeyVaultService) listDeletedKeys(vaultName string, req *http.Request) (*service.Response, error) {
	vaultKey := strings.ToLower(vaultName)

	s.mu.RLock()
	values := make([]DeletedKeyListItem, 0, len(s.deletedKeys[vaultKey]))
	for _, state := range s.deletedKeys[vaultKey] {
		values = append(values, DeletedKeyListItem{
			RecoveryID:         state.Bundle.RecoveryID,
			DeletedDate:        state.Bundle.DeletedDate,
			ScheduledPurgeDate: state.Bundle.ScheduledPurgeDate,
			KID:                keyBaseIDFromBundle(state.Latest),
			Managed:            state.Bundle.Managed,
			Attributes:         state.Bundle.Attributes,
			Tags:               state.Bundle.Tags,
		})
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].RecoveryID < values[j].RecoveryID })
	options, invalidResp, err := keyVaultListOptionsFromRequest(req)
	if invalidResp != nil || err != nil {
		return invalidResp, err
	}
	return azurearm.JSONResponse(http.StatusOK, keyVaultListPage(req, values, options))
}

func (s *KeyVaultService) recoverDeletedKey(vaultName, name string) (*service.Response, error) {
	vaultKey := strings.ToLower(vaultName)
	keyKey := strings.ToLower(name)

	s.mu.Lock()
	state, ok := s.deletedKeys[vaultKey][keyKey]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "KeyNotFound", fmt.Sprintf("Deleted key %q could not be found.", name))
	}
	if s.keys[vaultKey] == nil {
		s.keys[vaultKey] = make(map[string]KeyBundle)
	}
	if s.keyVersions[vaultKey] == nil {
		s.keyVersions[vaultKey] = make(map[string]map[string]KeyBundle)
	}
	s.keys[vaultKey][keyKey] = state.Latest
	s.keyVersions[vaultKey][keyKey] = cloneKeyVersions(state.Versions)
	delete(s.deletedKeys[vaultKey], keyKey)
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, state.Latest)
}

func (s *KeyVaultService) purgeDeletedKey(vaultName, name string) (*service.Response, error) {
	vaultKey := strings.ToLower(vaultName)
	keyKey := strings.ToLower(name)

	s.mu.Lock()
	if _, ok := s.deletedKeys[vaultKey][keyKey]; !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "KeyNotFound", fmt.Sprintf("Deleted key %q could not be found.", name))
	}
	delete(s.deletedKeys[vaultKey], keyKey)
	if s.keys[vaultKey] != nil {
		delete(s.keys[vaultKey], keyKey)
	}
	if s.keyVersions[vaultKey] != nil {
		delete(s.keyVersions[vaultKey], keyKey)
	}
	s.mu.Unlock()

	return &service.Response{StatusCode: http.StatusNoContent}, nil
}

func (s *KeyVaultService) encryptWithKey(vaultName, name, version string, body []byte) (*service.Response, error) {
	var input struct {
		Algorithm string `json:"alg"`
		Value     string `json:"value"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The request content was invalid.")
		}
	}
	if input.Algorithm == "" || input.Value == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The encryption alg and value are required.")
	}
	bundle, ok := s.keyBundle(vaultName, name, version)
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "KeyNotFound", fmt.Sprintf("Key %q could not be found.", name))
	}

	plaintext, err := base64.RawURLEncoding.DecodeString(input.Value)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The encryption value must be base64url encoded.")
	}
	ciphertext := base64.RawURLEncoding.EncodeToString([]byte(keyOperationPrefix(bundle.Key.KID, input.Algorithm) + string(plaintext)))
	return azurearm.JSONResponse(http.StatusOK, KeyOperationResult{KID: bundle.Key.KID, Value: ciphertext})
}

func (s *KeyVaultService) decryptWithKey(vaultName, name, version string, body []byte) (*service.Response, error) {
	var input struct {
		Algorithm string `json:"alg"`
		Value     string `json:"value"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The request content was invalid.")
		}
	}
	if input.Algorithm == "" || input.Value == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The decryption alg and value are required.")
	}
	bundle, ok := s.keyBundle(vaultName, name, version)
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "KeyNotFound", fmt.Sprintf("Key %q could not be found.", name))
	}

	decoded, err := base64.RawURLEncoding.DecodeString(input.Value)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The decryption value must be base64url encoded.")
	}
	prefix := keyOperationPrefix(bundle.Key.KID, input.Algorithm)
	decodedText := string(decoded)
	if !strings.HasPrefix(decodedText, prefix) {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The decryption value was not produced by this key and algorithm.")
	}
	plaintext := strings.TrimPrefix(decodedText, prefix)
	return azurearm.JSONResponse(http.StatusOK, KeyOperationResult{KID: bundle.Key.KID, Value: base64.RawURLEncoding.EncodeToString([]byte(plaintext))})
}

func (s *KeyVaultService) signWithKey(vaultName, name, version string, body []byte) (*service.Response, error) {
	var input struct {
		Algorithm string `json:"alg"`
		Value     string `json:"value"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The request content was invalid.")
		}
	}
	if input.Algorithm == "" || input.Value == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The signing alg and value are required.")
	}
	bundle, ok := s.keyBundle(vaultName, name, version)
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "KeyNotFound", fmt.Sprintf("Key %q could not be found.", name))
	}
	if _, err := base64.RawURLEncoding.DecodeString(input.Value); err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The signing value must be base64url encoded.")
	}
	return azurearm.JSONResponse(http.StatusOK, KeyOperationResult{
		KID:   bundle.Key.KID,
		Value: keySignatureValue(bundle.Key.KID, input.Algorithm, input.Value),
	})
}

func (s *KeyVaultService) verifyWithKey(vaultName, name, version string, body []byte) (*service.Response, error) {
	var input struct {
		Algorithm string `json:"alg"`
		Digest    string `json:"digest"`
		Value     string `json:"value"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The request content was invalid.")
		}
	}
	if input.Algorithm == "" || input.Digest == "" || input.Value == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The verification alg, digest, and value are required.")
	}
	bundle, ok := s.keyBundle(vaultName, name, version)
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "KeyNotFound", fmt.Sprintf("Key %q could not be found.", name))
	}
	if _, err := base64.RawURLEncoding.DecodeString(input.Digest); err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The verification digest must be base64url encoded.")
	}
	if _, err := base64.RawURLEncoding.DecodeString(input.Value); err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The verification value must be base64url encoded.")
	}
	expected := keySignatureValue(bundle.Key.KID, input.Algorithm, input.Digest)
	return azurearm.JSONResponse(http.StatusOK, KeyVerifyResult{Value: input.Value == expected})
}

func (s *KeyVaultService) wrapWithKey(vaultName, name, version string, body []byte) (*service.Response, error) {
	var input struct {
		Algorithm string `json:"alg"`
		Value     string `json:"value"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The request content was invalid.")
		}
	}
	if input.Algorithm == "" || input.Value == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The wrap alg and value are required.")
	}
	bundle, ok := s.keyBundle(vaultName, name, version)
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "KeyNotFound", fmt.Sprintf("Key %q could not be found.", name))
	}
	plaintext, err := base64.RawURLEncoding.DecodeString(input.Value)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The wrap value must be base64url encoded.")
	}
	wrapped := base64.RawURLEncoding.EncodeToString([]byte(keyWrapPrefix(bundle.Key.KID, input.Algorithm) + string(plaintext)))
	return azurearm.JSONResponse(http.StatusOK, KeyOperationResult{KID: bundle.Key.KID, Value: wrapped})
}

func (s *KeyVaultService) unwrapWithKey(vaultName, name, version string, body []byte) (*service.Response, error) {
	var input struct {
		Algorithm string `json:"alg"`
		Value     string `json:"value"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The request content was invalid.")
		}
	}
	if input.Algorithm == "" || input.Value == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The unwrap alg and value are required.")
	}
	bundle, ok := s.keyBundle(vaultName, name, version)
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "KeyNotFound", fmt.Sprintf("Key %q could not be found.", name))
	}
	decoded, err := base64.RawURLEncoding.DecodeString(input.Value)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The unwrap value must be base64url encoded.")
	}
	prefix := keyWrapPrefix(bundle.Key.KID, input.Algorithm)
	decodedText := string(decoded)
	if !strings.HasPrefix(decodedText, prefix) {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadParameter", "The unwrap value was not produced by this key and algorithm.")
	}
	plaintext := strings.TrimPrefix(decodedText, prefix)
	return azurearm.JSONResponse(http.StatusOK, KeyOperationResult{KID: bundle.Key.KID, Value: base64.RawURLEncoding.EncodeToString([]byte(plaintext))})
}

func (s *KeyVaultService) keyBundle(vaultName, name, version string) (KeyBundle, bool) {
	vaultKey := strings.ToLower(vaultName)
	keyKey := strings.ToLower(name)

	s.mu.RLock()
	if version != "" {
		bundle, ok := s.keyVersions[vaultKey][keyKey][version]
		s.mu.RUnlock()
		return bundle, ok
	}
	bundle, ok := s.keys[vaultKey][keyKey]
	s.mu.RUnlock()
	return bundle, ok
}

func (s *KeyVaultService) nextVersion() string {
	s.nextID++
	return fmt.Sprintf("%016x%016x", uint64(time.Now().UTC().UnixNano()), s.nextID)
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	raw := strings.Split(path, "/")
	parts := raw[:0]
	for _, part := range raw {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func segmentIndex(parts []string, segment string) int {
	for i, part := range parts {
		if strings.EqualFold(part, segment) {
			return i
		}
	}
	return -1
}

func dataPlaneVault(r *http.Request) (string, bool) {
	host := strings.ToLower(r.Host)
	if host == "" {
		host = strings.ToLower(r.Header.Get("Host"))
	}
	if colon := strings.IndexByte(host, ':'); colon >= 0 {
		host = host[:colon]
	}
	for _, suffix := range []string{
		".vault.azure.net",
		".vault.usgovcloudapi.net",
		".vault.azure.cn",
		".vault.microsoftazure.de",
	} {
		if strings.HasSuffix(host, suffix) {
			name := strings.TrimSuffix(host, suffix)
			return name, name != ""
		}
	}
	return "", false
}

func vaultKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func vaultKeyPrefix(subscriptionID, resourceGroup string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"
}

func vaultResourceID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.KeyVault/vaults/" + name
}

func vaultURI(name string) string {
	return "https://" + name + ".vault.azure.net/"
}

func secretID(vaultName, name, version string) string {
	return "https://" + vaultName + ".vault.azure.net/secrets/" + name + "/" + version
}

func keyID(vaultName, name, version string) string {
	return "https://" + vaultName + ".vault.azure.net/keys/" + name + "/" + version
}

func certificateID(vaultName, name, version string) string {
	return "https://" + vaultName + ".vault.azure.net/certificates/" + name + "/" + version
}

func keyOperationPrefix(kid, algorithm string) string {
	return "cloudmock-keyop:" + kid + ":" + algorithm + ":"
}

func keyWrapPrefix(kid, algorithm string) string {
	return "cloudmock-keywrap:" + kid + ":" + algorithm + ":"
}

func keySignatureValue(kid, algorithm, digest string) string {
	return base64.RawURLEncoding.EncodeToString([]byte("cloudmock-signature:" + kid + ":" + algorithm + ":" + digest))
}

func certificatePolicyID(vaultName, name string) string {
	return "https://" + vaultName + ".vault.azure.net/certificates/" + name + "/policy"
}

func secretBaseIDFromBundle(bundle SecretBundle) string {
	if before, _, ok := strings.Cut(bundle.ID, "/secrets/"); ok {
		rest := strings.TrimPrefix(bundle.ID, before+"/secrets/")
		if slash := strings.LastIndexByte(rest, '/'); slash >= 0 {
			return before + "/secrets/" + rest[:slash]
		}
	}
	if slash := strings.LastIndexByte(bundle.ID, '/'); slash >= 0 {
		return bundle.ID[:slash]
	}
	return bundle.ID
}

func certificateBaseIDFromBundle(bundle CertificateBundle) string {
	if before, _, ok := strings.Cut(bundle.ID, "/certificates/"); ok {
		rest := strings.TrimPrefix(bundle.ID, before+"/certificates/")
		if slash := strings.LastIndexByte(rest, '/'); slash >= 0 {
			return before + "/certificates/" + rest[:slash]
		}
	}
	if slash := strings.LastIndexByte(bundle.ID, '/'); slash >= 0 {
		return bundle.ID[:slash]
	}
	return bundle.ID
}

func keyBaseIDFromBundle(bundle KeyBundle) string {
	if before, _, ok := strings.Cut(bundle.Key.KID, "/keys/"); ok {
		rest := strings.TrimPrefix(bundle.Key.KID, before+"/keys/")
		if slash := strings.LastIndexByte(rest, '/'); slash >= 0 {
			return before + "/keys/" + rest[:slash]
		}
	}
	if slash := strings.LastIndexByte(bundle.Key.KID, '/'); slash >= 0 {
		return bundle.Key.KID[:slash]
	}
	return bundle.Key.KID
}

func versionFromSecretID(id string) string {
	if slash := strings.LastIndexByte(id, '/'); slash >= 0 && slash+1 < len(id) {
		return id[slash+1:]
	}
	return ""
}

func certificateThumbprint(value string) string {
	sum := sha1.Sum([]byte(value))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func cloneSecretVersions(values map[string]SecretBundle) map[string]SecretBundle {
	out := make(map[string]SecretBundle, len(values))
	for version, bundle := range values {
		out[version] = bundle
	}
	return out
}

func cloneCertificateVersions(values map[string]CertificateBundle) map[string]CertificateBundle {
	out := make(map[string]CertificateBundle, len(values))
	for version, bundle := range values {
		out[version] = bundle
	}
	return out
}

func cloneKeyVersions(values map[string]KeyBundle) map[string]KeyBundle {
	out := make(map[string]KeyBundle, len(values))
	for version, bundle := range values {
		out[version] = bundle
	}
	return out
}

func decodeSecretBackupValue(value string) ([]byte, error) {
	if data, err := base64.RawURLEncoding.DecodeString(value); err == nil {
		return data, nil
	}
	if data, err := base64.URLEncoding.DecodeString(value); err == nil {
		return data, nil
	}
	return base64.StdEncoding.DecodeString(value)
}

func parseSecretBackupBlob(data []byte) (secretBackupBlob, bool) {
	var raw struct {
		Vault    string          `json:"vault"`
		Name     string          `json:"name"`
		Latest   SecretBundle    `json:"latest"`
		Versions json.RawMessage `json:"versions"`
	}
	if err := gojson.Unmarshal(data, &raw); err != nil || raw.Name == "" || len(raw.Versions) == 0 {
		return secretBackupBlob{}, false
	}

	versions := map[string]SecretBundle{}
	var versionMap map[string]SecretBundle
	if err := gojson.Unmarshal(raw.Versions, &versionMap); err == nil && len(versionMap) > 0 {
		for version, bundle := range versionMap {
			if version == "" {
				version = versionFromSecretID(bundle.ID)
			}
			if version == "" {
				return secretBackupBlob{}, false
			}
			versions[version] = bundle
		}
	} else {
		var versionList []SecretBundle
		if err := gojson.Unmarshal(raw.Versions, &versionList); err != nil || len(versionList) == 0 {
			return secretBackupBlob{}, false
		}
		for _, bundle := range versionList {
			version := versionFromSecretID(bundle.ID)
			if version == "" {
				return secretBackupBlob{}, false
			}
			versions[version] = bundle
		}
	}

	latest := raw.Latest
	if latest.ID == "" {
		latest = latestSecretBundle(versions)
	}
	if latest.ID == "" {
		return secretBackupBlob{}, false
	}
	return secretBackupBlob{
		Vault:    raw.Vault,
		Name:     raw.Name,
		Latest:   latest,
		Versions: versions,
	}, true
}

func latestSecretBundle(values map[string]SecretBundle) SecretBundle {
	var latest SecretBundle
	for _, bundle := range values {
		if latest.ID == "" || strings.Compare(bundle.ID, latest.ID) > 0 {
			latest = bundle
		}
	}
	return latest
}

func latestCertificateBundle(values map[string]CertificateBundle) CertificateBundle {
	var latest CertificateBundle
	for _, bundle := range values {
		if latest.ID == "" || strings.Compare(bundle.ID, latest.ID) > 0 {
			latest = bundle
		}
	}
	return latest
}

func latestKeyBundle(values map[string]KeyBundle) KeyBundle {
	var latest KeyBundle
	for _, bundle := range values {
		if latest.Key.KID == "" || strings.Compare(bundle.Key.KID, latest.Key.KID) > 0 {
			latest = bundle
		}
	}
	return latest
}

func rewriteSecretBundleForVault(bundle SecretBundle, vaultName, name string) SecretBundle {
	version := versionFromSecretID(bundle.ID)
	if version != "" {
		bundle.ID = secretID(vaultName, name, version)
		if bundle.KID != "" {
			bundle.KID = keyID(vaultName, name, version)
		}
	}
	return bundle
}

func rewriteCertificateBundleForVault(bundle CertificateBundle, vaultName, name string) CertificateBundle {
	version := versionFromSecretID(bundle.ID)
	if version != "" {
		bundle.ID = certificateID(vaultName, name, version)
		bundle.KID = keyID(vaultName, name, version)
		bundle.SID = secretID(vaultName, name, version)
	}
	if bundle.Policy != nil {
		bundle.Policy = cloneAnyMap(bundle.Policy)
		bundle.Policy["id"] = certificatePolicyID(vaultName, name)
	}
	return bundle
}

func rewriteKeyBundleForVault(bundle KeyBundle, vaultName, name string) KeyBundle {
	version := versionFromSecretID(bundle.Key.KID)
	if version != "" {
		bundle.Key.KID = keyID(vaultName, name, version)
	}
	return bundle
}

func certificatePolicy(vaultName, name string, input map[string]any, enabled bool, now int64) map[string]any {
	policy := cloneAnyMap(input)
	if policy == nil {
		policy = map[string]any{}
	}
	policy["id"] = certificatePolicyID(vaultName, name)
	if _, ok := policy["key_props"].(map[string]any); !ok {
		policy["key_props"] = map[string]any{"exportable": true, "kty": "RSA", "key_size": 2048, "reuse_key": false}
	}
	if _, ok := policy["secret_props"].(map[string]any); !ok {
		policy["secret_props"] = map[string]any{"contentType": "application/x-pkcs12"}
	}
	if _, ok := policy["issuer"].(map[string]any); !ok {
		policy["issuer"] = map[string]any{"name": "Unknown"}
	}
	policy["attributes"] = map[string]any{"enabled": enabled, "created": now, "updated": now}
	return policy
}

func certificatePolicyContentType(policy map[string]any) string {
	if props, ok := policy["secret_props"].(map[string]any); ok {
		if value, ok := props["contentType"].(string); ok && value != "" {
			return value
		}
	}
	return "application/x-pkcs12"
}

func certificatePolicyKeyType(policy map[string]any) string {
	if props, ok := policy["key_props"].(map[string]any); ok {
		if value, ok := props["kty"].(string); ok && value != "" {
			return value
		}
	}
	return "RSA"
}

func certificatePolicyIssuer(policy map[string]any) map[string]any {
	if issuer, ok := policy["issuer"].(map[string]any); ok && len(issuer) > 0 {
		return cloneAnyMap(issuer)
	}
	return map[string]any{"name": "Unknown"}
}

func cloneAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		if nested, ok := value.(map[string]any); ok {
			out[key] = cloneAnyMap(nested)
			continue
		}
		out[key] = value
	}
	return out
}

func mergeAnyMap(dst, src map[string]any) {
	for key, value := range src {
		if nested, ok := value.(map[string]any); ok {
			if existing, ok := dst[key].(map[string]any); ok {
				mergeAnyMap(existing, nested)
				continue
			}
			dst[key] = cloneAnyMap(nested)
			continue
		}
		dst[key] = value
	}
}

func int64MapValue(values map[string]any, key string) (int64, bool) {
	switch value := values[key].(type) {
	case float64:
		return int64(value), true
	case int64:
		return value, true
	case int:
		return int64(value), true
	case json.Number:
		parsed, err := value.Int64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func certificateMirrorTimes(attributes CertificateAttributes) (int64, int64) {
	created := attributes.Created
	updated := attributes.Updated
	if updated == 0 {
		updated = created
	}
	return created, updated
}

func (s *KeyVaultService) linkCertificateSecretLocked(vaultName, name, version, value, contentType string, tags map[string]string, certAttributes CertificateAttributes) {
	vaultKey := strings.ToLower(vaultName)
	secretKey := strings.ToLower(name)
	created, updated := certificateMirrorTimes(certAttributes)
	if s.secrets[vaultKey] == nil {
		s.secrets[vaultKey] = make(map[string]SecretBundle)
	}
	if s.secretVersions[vaultKey] == nil {
		s.secretVersions[vaultKey] = make(map[string]map[string]SecretBundle)
	}
	if s.secretVersions[vaultKey][secretKey] == nil {
		s.secretVersions[vaultKey][secretKey] = make(map[string]SecretBundle)
	}
	var previousVersion string
	if latest := s.secrets[vaultKey][secretKey]; latest.ID != "" {
		previousVersion = versionFromSecretID(latest.ID)
		if previousVersion == version {
			previousVersion = latest.PreviousVersion
		}
	}
	bundle := SecretBundle{
		Value:           value,
		ID:              secretID(vaultName, name, version),
		KID:             keyID(vaultName, name, version),
		Managed:         true,
		PreviousVersion: previousVersion,
		ContentType:     contentType,
		Tags:            tags,
		Attributes: SecretAttributes{
			Enabled:       certAttributes.Enabled,
			NotBefore:     certAttributes.NotBefore,
			Expires:       certAttributes.Expires,
			Created:       created,
			Updated:       updated,
			RecoveryLevel: "Recoverable+Purgeable",
		},
	}
	s.secrets[vaultKey][secretKey] = bundle
	s.secretVersions[vaultKey][secretKey][version] = bundle
}

func (s *KeyVaultService) updateLinkedCertificateSecretLocked(vaultKey, secretKey, version string, bundle CertificateBundle) {
	secretVersions := s.secretVersions[vaultKey][secretKey]
	secret, ok := secretVersions[version]
	if !ok {
		return
	}
	secret.ContentType = bundle.ContentType
	secret.Tags = bundle.Tags
	secret.Attributes.Enabled = bundle.Attributes.Enabled
	secret.Attributes.NotBefore = bundle.Attributes.NotBefore
	secret.Attributes.Expires = bundle.Attributes.Expires
	secret.Attributes.Updated = bundle.Attributes.Updated
	secretVersions[version] = secret
	if latest, ok := s.secrets[vaultKey][secretKey]; ok && strings.HasSuffix(latest.ID, "/"+version) {
		s.secrets[vaultKey][secretKey] = secret
	}
}

func (s *KeyVaultService) linkCertificateKeyLocked(vaultName, name, version string, policy map[string]any, tags map[string]string, certAttributes CertificateAttributes) {
	vaultKey := strings.ToLower(vaultName)
	keyKey := strings.ToLower(name)
	created, updated := certificateMirrorTimes(certAttributes)
	if s.keys[vaultKey] == nil {
		s.keys[vaultKey] = make(map[string]KeyBundle)
	}
	if s.keyVersions[vaultKey] == nil {
		s.keyVersions[vaultKey] = make(map[string]map[string]KeyBundle)
	}
	if s.keyVersions[vaultKey][keyKey] == nil {
		s.keyVersions[vaultKey][keyKey] = make(map[string]KeyBundle)
	}
	bundle := KeyBundle{
		Key: JsonWebKey{
			KID:    keyID(vaultName, name, version),
			KTY:    certificatePolicyKeyType(policy),
			KeyOps: []string{"sign", "verify"},
			N:      base64.RawURLEncoding.EncodeToString([]byte("cloudmock:" + vaultKey + ":" + keyKey + ":cert:n")),
			E:      "AQAB",
		},
		Managed: true,
		Tags:    tags,
		Attributes: KeyAttributes{
			Enabled:       certAttributes.Enabled,
			NotBefore:     certAttributes.NotBefore,
			Expires:       certAttributes.Expires,
			Created:       created,
			Updated:       updated,
			RecoveryLevel: "Recoverable+Purgeable",
		},
	}
	s.keys[vaultKey][keyKey] = bundle
	s.keyVersions[vaultKey][keyKey][version] = bundle
}

func (s *KeyVaultService) updateLinkedCertificateKeyLocked(vaultKey, keyKey string, bundle CertificateBundle) {
	keyBundle, ok := s.keys[vaultKey][keyKey]
	if !ok || keyBundle.Key.KID != bundle.KID {
		return
	}
	keyBundle.Tags = bundle.Tags
	keyBundle.Attributes.Enabled = bundle.Attributes.Enabled
	keyBundle.Attributes.NotBefore = bundle.Attributes.NotBefore
	keyBundle.Attributes.Expires = bundle.Attributes.Expires
	keyBundle.Attributes.Updated = bundle.Attributes.Updated
	s.keys[vaultKey][keyKey] = keyBundle
	version := versionFromSecretID(bundle.KID)
	if versionMap := s.keyVersions[vaultKey][keyKey]; versionMap != nil {
		versionMap[version] = keyBundle
	}
}

func stringifyTags(tags map[string]any) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	out := make(map[string]string, len(tags))
	for key, value := range tags {
		out[key] = fmt.Sprint(value)
	}
	return out
}

func stringValue(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}
