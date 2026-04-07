package lakeformation

import (
	"encoding/json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidInputException", "Invalid JSON.", http.StatusBadRequest)
	}
	return nil
}

// ---- JSON types ----

type principalJSON struct {
	DataLakePrincipalIdentifier string `json:"DataLakePrincipalIdentifier"`
}

type databaseResourceJSON struct {
	Name      string `json:"Name"`
	CatalogId string `json:"CatalogId,omitempty"`
}

type tableResourceJSON struct {
	DatabaseName string `json:"DatabaseName"`
	Name         string `json:"Name"`
	CatalogId    string `json:"CatalogId,omitempty"`
}

type resourceJSON struct {
	Database *databaseResourceJSON `json:"Database,omitempty"`
	Table    *tableResourceJSON    `json:"Table,omitempty"`
}

type permissionJSON struct {
	Principal                  principalJSON `json:"Principal"`
	Resource                   resourceJSON  `json:"Resource"`
	Permissions                []string      `json:"Permissions"`
	PermissionsWithGrantOption []string      `json:"PermissionsWithGrantOption,omitempty"`
}

type lfTagJSON struct {
	TagKey    string   `json:"TagKey"`
	TagValues []string `json:"TagValues"`
}

// ---- Resource handlers ----

type registerResourceRequest struct {
	ResourceArn string `json:"ResourceArn"`
	RoleArn     string `json:"RoleArn"`
}

func handleRegisterResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req registerResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.RegisterResource(req.ResourceArn, req.RoleArn) {
		return jsonErr(service.ErrAlreadyExists("Resource", req.ResourceArn))
	}
	return jsonOK(struct{}{})
}

type deregisterResourceRequest struct {
	ResourceArn string `json:"ResourceArn"`
}

func handleDeregisterResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deregisterResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.DeregisterResource(req.ResourceArn) {
		return jsonErr(service.ErrNotFound("Resource", req.ResourceArn))
	}
	return jsonOK(struct{}{})
}

type resourceInfoJSON struct {
	ResourceArn string  `json:"ResourceArn"`
	RoleArn     string  `json:"RoleArn,omitempty"`
}

func handleListResources(_ *service.RequestContext, store *Store) (*service.Response, error) {
	resources := store.ListResources()
	list := make([]resourceInfoJSON, 0, len(resources))
	for _, r := range resources {
		list = append(list, resourceInfoJSON{ResourceArn: r.ResourceArn, RoleArn: r.RoleArn})
	}
	return jsonOK(map[string]any{"ResourceInfoList": list})
}

// ---- Permission handlers ----

type grantPermissionsRequest struct {
	Principal                  principalJSON `json:"Principal"`
	Resource                   resourceJSON  `json:"Resource"`
	Permissions                []string      `json:"Permissions"`
	PermissionsWithGrantOption []string      `json:"PermissionsWithGrantOption"`
}

func toPermResource(r resourceJSON) PermissionResource {
	pr := PermissionResource{}
	if r.Database != nil {
		pr.Database = &DatabaseResource{Name: r.Database.Name, CatalogId: r.Database.CatalogId}
	}
	if r.Table != nil {
		pr.Table = &TableResource{DatabaseName: r.Table.DatabaseName, Name: r.Table.Name, CatalogId: r.Table.CatalogId}
	}
	return pr
}

func handleGrantPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req grantPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Principal.DataLakePrincipalIdentifier == "" {
		return jsonErr(service.ErrValidation("Principal.DataLakePrincipalIdentifier is required."))
	}
	if len(req.Permissions) == 0 {
		return jsonErr(service.ErrValidation("At least one permission is required."))
	}
	if req.Resource.Database == nil && req.Resource.Table == nil {
		return jsonErr(service.ErrValidation("Resource must specify a Database or Table."))
	}
	perm := &Permission{
		Principal:                  DataLakePrincipal{DataLakePrincipalIdentifier: req.Principal.DataLakePrincipalIdentifier},
		Resource:                   toPermResource(req.Resource),
		Permissions:                req.Permissions,
		PermissionsWithGrantOption: req.PermissionsWithGrantOption,
	}
	store.GrantPermissions(perm)
	return jsonOK(struct{}{})
}

type revokePermissionsRequest struct {
	Principal principalJSON `json:"Principal"`
	Resource  resourceJSON  `json:"Resource"`
}

func handleRevokePermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req revokePermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	store.RevokePermissions(req.Principal.DataLakePrincipalIdentifier, toPermResource(req.Resource))
	return jsonOK(struct{}{})
}

type getEffectivePermissionsRequest struct {
	ResourceArn string `json:"ResourceArn"`
}

func toPermJSON(p *Permission) permissionJSON {
	pj := permissionJSON{
		Principal:                  principalJSON{DataLakePrincipalIdentifier: p.Principal.DataLakePrincipalIdentifier},
		Permissions:                p.Permissions,
		PermissionsWithGrantOption: p.PermissionsWithGrantOption,
	}
	if p.Resource.Database != nil {
		pj.Resource.Database = &databaseResourceJSON{Name: p.Resource.Database.Name, CatalogId: p.Resource.Database.CatalogId}
	}
	if p.Resource.Table != nil {
		pj.Resource.Table = &tableResourceJSON{DatabaseName: p.Resource.Table.DatabaseName, Name: p.Resource.Table.Name, CatalogId: p.Resource.Table.CatalogId}
	}
	return pj
}

func handleGetEffectivePermissionsForPath(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getEffectivePermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := store.GetEffectivePermissionsForPath(req.ResourceArn)
	list := make([]permissionJSON, 0, len(perms))
	for _, p := range perms {
		list = append(list, toPermJSON(p))
	}
	return jsonOK(map[string]any{"Permissions": list})
}

type listPermissionsRequest struct {
	Principal *principalJSON `json:"Principal"`
}

func handleListPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listPermissionsRequest
	parseJSON(ctx.Body, &req)
	principal := ""
	if req.Principal != nil {
		principal = req.Principal.DataLakePrincipalIdentifier
	}
	perms := store.ListPermissions(principal)
	list := make([]permissionJSON, 0, len(perms))
	for _, p := range perms {
		list = append(list, toPermJSON(p))
	}
	return jsonOK(map[string]any{"PrincipalResourcePermissions": list})
}

// ---- Data Lake Settings handlers ----

type principalPermissionsJSON struct {
	Principal   principalJSON `json:"Principal"`
	Permissions []string      `json:"Permissions"`
}

type dataLakeSettingsJSON struct {
	DataLakeAdmins                []principalJSON            `json:"DataLakeAdmins"`
	CreateDatabaseDefaultPermissions []principalPermissionsJSON `json:"CreateDatabaseDefaultPermissions,omitempty"`
	CreateTableDefaultPermissions    []principalPermissionsJSON `json:"CreateTableDefaultPermissions,omitempty"`
}

func handleGetDataLakeSettings(_ *service.RequestContext, store *Store) (*service.Response, error) {
	settings := store.GetDataLakeSettings()
	admins := make([]principalJSON, 0, len(settings.DataLakeAdmins))
	for _, a := range settings.DataLakeAdmins {
		admins = append(admins, principalJSON{DataLakePrincipalIdentifier: a.DataLakePrincipalIdentifier})
	}
	return jsonOK(map[string]any{"DataLakeSettings": dataLakeSettingsJSON{DataLakeAdmins: admins}})
}

type putDataLakeSettingsRequest struct {
	DataLakeSettings dataLakeSettingsJSON `json:"DataLakeSettings"`
}

func handlePutDataLakeSettings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req putDataLakeSettingsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	admins := make([]DataLakePrincipal, 0, len(req.DataLakeSettings.DataLakeAdmins))
	for _, a := range req.DataLakeSettings.DataLakeAdmins {
		admins = append(admins, DataLakePrincipal{DataLakePrincipalIdentifier: a.DataLakePrincipalIdentifier})
	}
	store.PutDataLakeSettings(DataLakeSettings{DataLakeAdmins: admins})
	return jsonOK(struct{}{})
}

// ---- LFTag handlers ----

type createLFTagRequest struct {
	TagKey    string   `json:"TagKey"`
	TagValues []string `json:"TagValues"`
}

func handleCreateLFTag(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createLFTagRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.CreateLFTag(req.TagKey, req.TagValues) {
		return jsonErr(service.ErrAlreadyExists("LFTag", req.TagKey))
	}
	return jsonOK(struct{}{})
}

type getLFTagRequest struct {
	TagKey string `json:"TagKey"`
}

func handleGetLFTag(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getLFTagRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	t, ok := store.GetLFTag(req.TagKey)
	if !ok {
		return jsonErr(service.ErrNotFound("LFTag", req.TagKey))
	}
	return jsonOK(lfTagJSON{TagKey: t.TagKey, TagValues: t.TagValues})
}

func handleListLFTags(_ *service.RequestContext, store *Store) (*service.Response, error) {
	tags := store.ListLFTags()
	list := make([]lfTagJSON, 0, len(tags))
	for _, t := range tags {
		list = append(list, lfTagJSON{TagKey: t.TagKey, TagValues: t.TagValues})
	}
	return jsonOK(map[string]any{"LFTags": list})
}

type deleteLFTagRequest struct {
	TagKey string `json:"TagKey"`
}

func handleDeleteLFTag(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteLFTagRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.DeleteLFTag(req.TagKey) {
		return jsonErr(service.ErrNotFound("LFTag", req.TagKey))
	}
	return jsonOK(struct{}{})
}

type updateLFTagRequest struct {
	TagKey         string   `json:"TagKey"`
	TagValuesToAdd    []string `json:"TagValuesToAdd"`
	TagValuesToDelete []string `json:"TagValuesToDelete"`
}

func handleUpdateLFTag(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateLFTagRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if !store.UpdateLFTag(req.TagKey, req.TagValuesToAdd, req.TagValuesToDelete) {
		return jsonErr(service.ErrNotFound("LFTag", req.TagKey))
	}
	return jsonOK(struct{}{})
}

// ---- Resource LFTag handlers ----

type addLFTagsToResourceRequest struct {
	Resource resourceJSON `json:"Resource"`
	LFTags   []lfTagJSON  `json:"LFTags"`
}

func resourceKey(r resourceJSON) string {
	if r.Database != nil {
		return "database:" + r.Database.Name
	}
	if r.Table != nil {
		return "table:" + r.Table.DatabaseName + "/" + r.Table.Name
	}
	return ""
}

func handleAddLFTagsToResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req addLFTagsToResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	pairs := make([]LFTagPair, len(req.LFTags))
	for i, t := range req.LFTags {
		pairs[i] = LFTagPair{TagKey: t.TagKey, TagValues: t.TagValues}
	}
	store.AddLFTagsToResource(resourceKey(req.Resource), pairs)
	return jsonOK(map[string]any{"Failures": []any{}})
}

type removeLFTagsFromResourceRequest struct {
	Resource resourceJSON `json:"Resource"`
	LFTags   []lfTagJSON  `json:"LFTags"`
}

func handleRemoveLFTagsFromResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req removeLFTagsFromResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	keys := make([]string, len(req.LFTags))
	for i, t := range req.LFTags {
		keys[i] = t.TagKey
	}
	store.RemoveLFTagsFromResource(resourceKey(req.Resource), keys)
	return jsonOK(map[string]any{"Failures": []any{}})
}

type getResourceLFTagsRequest struct {
	Resource resourceJSON `json:"Resource"`
}

func handleGetResourceLFTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getResourceLFTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	tags := store.GetResourceLFTags(resourceKey(req.Resource))
	list := make([]lfTagJSON, 0, len(tags))
	for _, t := range tags {
		list = append(list, lfTagJSON{TagKey: t.TagKey, TagValues: t.TagValues})
	}
	return jsonOK(map[string]any{"LFTagOnDatabase": list})
}

// ---- BatchGrantPermissions ----

type batchPermissionsEntryJSON struct {
	Id          string        `json:"Id"`
	Principal   principalJSON `json:"Principal"`
	Resource    resourceJSON  `json:"Resource"`
	Permissions []string      `json:"Permissions"`
}

type batchGrantPermissionsRequest struct {
	Entries []batchPermissionsEntryJSON `json:"Entries"`
}

func handleBatchGrantPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req batchGrantPermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := make([]*Permission, 0, len(req.Entries))
	for _, e := range req.Entries {
		if e.Principal.DataLakePrincipalIdentifier == "" {
			continue
		}
		perms = append(perms, &Permission{
			Principal:   DataLakePrincipal{DataLakePrincipalIdentifier: e.Principal.DataLakePrincipalIdentifier},
			Resource:    toPermResource(e.Resource),
			Permissions: e.Permissions,
		})
	}
	store.BatchGrantPermissions(perms)
	return jsonOK(map[string]any{"Failures": []any{}})
}

// ---- BatchRevokePermissions ----

type batchRevokePermissionsRequest struct {
	Entries []batchPermissionsEntryJSON `json:"Entries"`
}

func handleBatchRevokePermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req batchRevokePermissionsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	perms := make([]*Permission, 0, len(req.Entries))
	for _, e := range req.Entries {
		perms = append(perms, &Permission{
			Principal: DataLakePrincipal{DataLakePrincipalIdentifier: e.Principal.DataLakePrincipalIdentifier},
			Resource:  toPermResource(e.Resource),
		})
	}
	store.BatchRevokePermissions(perms)
	return jsonOK(map[string]any{"Failures": []any{}})
}

// ---- DescribeResource ----

type describeResourceRequest struct {
	ResourceArn string `json:"ResourceArn"`
}

func handleDescribeResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req describeResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.ResourceArn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	r, ok := store.DescribeResource(req.ResourceArn)
	if !ok {
		return jsonErr(service.ErrNotFound("Resource", req.ResourceArn))
	}
	return jsonOK(map[string]any{
		"ResourceInfo": map[string]any{
			"ResourceArn":    r.ResourceArn,
			"RoleArn":        r.RoleArn,
			"LastModified":   float64(r.RegisteredTime.Unix()),
		},
	})
}
