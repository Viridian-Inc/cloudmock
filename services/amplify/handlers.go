package amplify

import (
	"encoding/json"
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
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
		return service.NewAWSError("BadRequestException", "Invalid JSON.", http.StatusBadRequest)
	}
	return nil
}

// ---- CreateApp ----

type createAppRequest struct {
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	Repository        string            `json:"repository"`
	Platform          string            `json:"platform"`
	IamServiceRoleArn string            `json:"iamServiceRoleArn"`
	Tags              map[string]string `json:"tags"`
}

type appJSON struct {
	AppId             string            `json:"appId"`
	AppArn            string            `json:"appArn"`
	Name              string            `json:"name"`
	Description       string            `json:"description,omitempty"`
	Repository        string            `json:"repository,omitempty"`
	Platform          string            `json:"platform"`
	DefaultDomain     string            `json:"defaultDomain"`
	CreateTime        string            `json:"createTime"`
	UpdateTime        string            `json:"updateTime"`
	Tags              map[string]string `json:"tags,omitempty"`
}

func appToJSON(app *App) appJSON {
	return appJSON{
		AppId: app.AppId, AppArn: app.AppArn, Name: app.Name,
		Description: app.Description, Repository: app.Repository,
		Platform: app.Platform, DefaultDomain: app.DefaultDomain,
		CreateTime: app.CreateTime.Format("2006-01-02T15:04:05Z"),
		UpdateTime: app.UpdateTime.Format("2006-01-02T15:04:05Z"),
		Tags: app.Tags,
	}
}

type createAppResponse struct {
	App appJSON `json:"app"`
}

func handleCreateApp(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createAppRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.Name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	app := store.CreateApp(req.Name, req.Description, req.Repository, req.Platform, req.IamServiceRoleArn, req.Tags)
	return jsonOK(&createAppResponse{App: appToJSON(app)})
}

// ---- GetApp ----

type getAppRequest struct {
	AppId string `json:"appId"`
}

type getAppResponse struct {
	App appJSON `json:"app"`
}

func handleGetApp(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getAppRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	if appID == "" {
		return jsonErr(service.ErrValidation("appId is required."))
	}
	app, ok := store.GetApp(appID)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "App not found.", http.StatusNotFound))
	}
	return jsonOK(&getAppResponse{App: appToJSON(app)})
}

// ---- ListApps ----

type listAppsResponse struct {
	Apps []appJSON `json:"apps"`
}

func handleListApps(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	apps := store.ListApps()
	items := make([]appJSON, 0, len(apps))
	for _, app := range apps {
		items = append(items, appToJSON(app))
	}
	return jsonOK(&listAppsResponse{Apps: items})
}

// ---- UpdateApp ----

type updateAppRequest struct {
	AppId             string `json:"appId"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	Platform          string `json:"platform"`
	IamServiceRoleArn string `json:"iamServiceRoleArn"`
}

type updateAppResponse struct {
	App appJSON `json:"app"`
}

func handleUpdateApp(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateAppRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	if appID == "" {
		return jsonErr(service.ErrValidation("appId is required."))
	}
	app, ok := store.UpdateApp(appID, req.Name, req.Description, req.Platform, req.IamServiceRoleArn)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "App not found.", http.StatusNotFound))
	}
	return jsonOK(&updateAppResponse{App: appToJSON(app)})
}

// ---- DeleteApp ----

type deleteAppRequest struct {
	AppId string `json:"appId"`
}

type deleteAppResponse struct {
	App appJSON `json:"app"`
}

func handleDeleteApp(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteAppRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	if appID == "" {
		return jsonErr(service.ErrValidation("appId is required."))
	}
	app, ok := store.GetApp(appID)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "App not found.", http.StatusNotFound))
	}
	store.DeleteApp(appID)
	return jsonOK(&deleteAppResponse{App: appToJSON(app)})
}

// ---- CreateBranch ----

type createBranchRequest struct {
	AppId           string            `json:"appId"`
	BranchName      string            `json:"branchName"`
	Description     string            `json:"description"`
	Stage           string            `json:"stage"`
	Framework       string            `json:"framework"`
	EnableAutoBuild bool              `json:"enableAutoBuild"`
	Tags            map[string]string `json:"tags"`
}

type branchJSON struct {
	BranchArn       string            `json:"branchArn"`
	BranchName      string            `json:"branchName"`
	Description     string            `json:"description,omitempty"`
	Stage           string            `json:"stage"`
	Framework       string            `json:"framework,omitempty"`
	EnableAutoBuild bool              `json:"enableAutoBuild"`
	DisplayName     string            `json:"displayName"`
	CreateTime      string            `json:"createTime"`
	UpdateTime      string            `json:"updateTime"`
	Tags            map[string]string `json:"tags,omitempty"`
}

func branchToJSON(b *Branch) branchJSON {
	return branchJSON{
		BranchArn: b.BranchArn, BranchName: b.BranchName,
		Description: b.Description, Stage: b.Stage,
		Framework: b.Framework, EnableAutoBuild: b.EnableAutoBuild,
		DisplayName: b.DisplayName,
		CreateTime: b.CreateTime.Format("2006-01-02T15:04:05Z"),
		UpdateTime: b.UpdateTime.Format("2006-01-02T15:04:05Z"),
		Tags: b.Tags,
	}
}

type createBranchResponse struct {
	Branch branchJSON `json:"branch"`
}

func handleCreateBranch(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createBranchRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	if appID == "" || req.BranchName == "" {
		return jsonErr(service.ErrValidation("appId and branchName are required."))
	}
	branch, ok := store.CreateBranch(appID, req.BranchName, req.Description, req.Stage, req.Framework, req.EnableAutoBuild, req.Tags)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "App not found or branch already exists.", http.StatusNotFound))
	}
	return jsonOK(&createBranchResponse{Branch: branchToJSON(branch)})
}

// ---- GetBranch ----

type getBranchRequest struct {
	AppId      string `json:"appId"`
	BranchName string `json:"branchName"`
}

type getBranchResponse struct {
	Branch branchJSON `json:"branch"`
}

func handleGetBranch(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getBranchRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	branchName := req.BranchName
	if branchName == "" {
		branchName = ctx.Params["branchName"]
	}
	if appID == "" || branchName == "" {
		return jsonErr(service.ErrValidation("appId and branchName are required."))
	}
	branch, ok := store.GetBranch(appID, branchName)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Branch not found.", http.StatusNotFound))
	}
	return jsonOK(&getBranchResponse{Branch: branchToJSON(branch)})
}

// ---- ListBranches ----

type listBranchesRequest struct {
	AppId string `json:"appId"`
}

type listBranchesResponse struct {
	Branches []branchJSON `json:"branches"`
}

func handleListBranches(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listBranchesRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	if appID == "" {
		return jsonErr(service.ErrValidation("appId is required."))
	}
	branches, ok := store.ListBranches(appID)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "App not found.", http.StatusNotFound))
	}
	items := make([]branchJSON, 0, len(branches))
	for _, b := range branches {
		items = append(items, branchToJSON(b))
	}
	return jsonOK(&listBranchesResponse{Branches: items})
}

// ---- UpdateBranch ----

type updateBranchRequest struct {
	AppId       string `json:"appId"`
	BranchName  string `json:"branchName"`
	Description string `json:"description"`
	Stage       string `json:"stage"`
	Framework   string `json:"framework"`
}

func handleUpdateBranch(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateBranchRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	branchName := req.BranchName
	if branchName == "" {
		branchName = ctx.Params["branchName"]
	}
	if appID == "" || branchName == "" {
		return jsonErr(service.ErrValidation("appId and branchName are required."))
	}
	branch, ok := store.UpdateBranch(appID, branchName, req.Description, req.Stage, req.Framework)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Branch not found.", http.StatusNotFound))
	}
	return jsonOK(&createBranchResponse{Branch: branchToJSON(branch)})
}

// ---- DeleteBranch ----

type deleteBranchRequest struct {
	AppId      string `json:"appId"`
	BranchName string `json:"branchName"`
}

func handleDeleteBranch(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteBranchRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	branchName := req.BranchName
	if branchName == "" {
		branchName = ctx.Params["branchName"]
	}
	if appID == "" || branchName == "" {
		return jsonErr(service.ErrValidation("appId and branchName are required."))
	}
	branch, ok := store.GetBranch(appID, branchName)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Branch not found.", http.StatusNotFound))
	}
	store.DeleteBranch(appID, branchName)
	return jsonOK(&createBranchResponse{Branch: branchToJSON(branch)})
}

// ---- CreateDomainAssociation ----

type subDomainSettingJSON struct {
	Prefix     string `json:"prefix"`
	BranchName string `json:"branchName"`
}

type createDomainAssociationRequest struct {
	AppId               string                 `json:"appId"`
	DomainName          string                 `json:"domainName"`
	EnableAutoSubDomain bool                   `json:"enableAutoSubDomain"`
	SubDomainSettings   []subDomainSettingJSON `json:"subDomainSettings"`
	Tags                map[string]string      `json:"tags"`
}

type subDomainJSON struct {
	SubDomainSetting subDomainSettingJSON `json:"subDomainSetting"`
	DnsRecord        string               `json:"dnsRecord"`
	Verified         bool                 `json:"verified"`
}

type domainAssociationJSON struct {
	DomainAssociationArn string          `json:"domainAssociationArn"`
	DomainName           string          `json:"domainName"`
	EnableAutoSubDomain  bool            `json:"enableAutoSubDomain"`
	DomainStatus         string          `json:"domainStatus"`
	SubDomains           []subDomainJSON `json:"subDomains"`
}

func domainToJSON(d *DomainAssociation) domainAssociationJSON {
	subs := make([]subDomainJSON, 0, len(d.SubDomains))
	for _, sd := range d.SubDomains {
		subs = append(subs, subDomainJSON{
			SubDomainSetting: subDomainSettingJSON{Prefix: sd.Prefix, BranchName: sd.BranchName},
			DnsRecord: sd.DnsRecord, Verified: sd.Verified,
		})
	}
	return domainAssociationJSON{
		DomainAssociationArn: d.DomainAssociationArn,
		DomainName: d.DomainName, EnableAutoSubDomain: d.EnableAutoSubDomain,
		DomainStatus: d.DomainStatus, SubDomains: subs,
	}
}

type createDomainAssociationResponse struct {
	DomainAssociation domainAssociationJSON `json:"domainAssociation"`
}

func handleCreateDomainAssociation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createDomainAssociationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	if appID == "" || req.DomainName == "" {
		return jsonErr(service.ErrValidation("appId and domainName are required."))
	}
	subDomains := make([]SubDomain, 0, len(req.SubDomainSettings))
	for _, sd := range req.SubDomainSettings {
		subDomains = append(subDomains, SubDomain{Prefix: sd.Prefix, BranchName: sd.BranchName})
	}
	domain, ok := store.CreateDomainAssociation(appID, req.DomainName, req.EnableAutoSubDomain, subDomains, req.Tags)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "App not found or domain already exists.", http.StatusNotFound))
	}
	return jsonOK(&createDomainAssociationResponse{DomainAssociation: domainToJSON(domain)})
}

// ---- GetDomainAssociation ----

type getDomainAssociationRequest struct {
	AppId      string `json:"appId"`
	DomainName string `json:"domainName"`
}

type getDomainAssociationResponse struct {
	DomainAssociation domainAssociationJSON `json:"domainAssociation"`
}

func handleGetDomainAssociation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getDomainAssociationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	domainName := req.DomainName
	if domainName == "" {
		domainName = ctx.Params["domainName"]
	}
	if appID == "" || domainName == "" {
		return jsonErr(service.ErrValidation("appId and domainName are required."))
	}
	domain, ok := store.GetDomainAssociation(appID, domainName)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Domain association not found.", http.StatusNotFound))
	}
	return jsonOK(&getDomainAssociationResponse{DomainAssociation: domainToJSON(domain)})
}

// ---- ListDomainAssociations ----

type listDomainAssociationsRequest struct {
	AppId string `json:"appId"`
}

type listDomainAssociationsResponse struct {
	DomainAssociations []domainAssociationJSON `json:"domainAssociations"`
}

func handleListDomainAssociations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listDomainAssociationsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	if appID == "" {
		return jsonErr(service.ErrValidation("appId is required."))
	}
	domains, ok := store.ListDomainAssociations(appID)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "App not found.", http.StatusNotFound))
	}
	items := make([]domainAssociationJSON, 0, len(domains))
	for _, d := range domains {
		items = append(items, domainToJSON(d))
	}
	return jsonOK(&listDomainAssociationsResponse{DomainAssociations: items})
}

// ---- UpdateDomainAssociation ----

type updateDomainAssociationRequest struct {
	AppId               string                 `json:"appId"`
	DomainName          string                 `json:"domainName"`
	EnableAutoSubDomain *bool                  `json:"enableAutoSubDomain"`
	SubDomainSettings   []subDomainSettingJSON `json:"subDomainSettings"`
}

func handleUpdateDomainAssociation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateDomainAssociationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	domainName := req.DomainName
	if domainName == "" {
		domainName = ctx.Params["domainName"]
	}
	if appID == "" || domainName == "" {
		return jsonErr(service.ErrValidation("appId and domainName are required."))
	}
	var subDomains []SubDomain
	if req.SubDomainSettings != nil {
		subDomains = make([]SubDomain, 0, len(req.SubDomainSettings))
		for _, sd := range req.SubDomainSettings {
			subDomains = append(subDomains, SubDomain{Prefix: sd.Prefix, BranchName: sd.BranchName})
		}
	}
	domain, ok := store.UpdateDomainAssociation(appID, domainName, req.EnableAutoSubDomain, subDomains)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Domain association not found.", http.StatusNotFound))
	}
	return jsonOK(&createDomainAssociationResponse{DomainAssociation: domainToJSON(domain)})
}

// ---- DeleteDomainAssociation ----

type deleteDomainAssociationRequest struct {
	AppId      string `json:"appId"`
	DomainName string `json:"domainName"`
}

func handleDeleteDomainAssociation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteDomainAssociationRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	domainName := req.DomainName
	if domainName == "" {
		domainName = ctx.Params["domainName"]
	}
	if appID == "" || domainName == "" {
		return jsonErr(service.ErrValidation("appId and domainName are required."))
	}
	domain, ok := store.GetDomainAssociation(appID, domainName)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Domain association not found.", http.StatusNotFound))
	}
	store.DeleteDomainAssociation(appID, domainName)
	return jsonOK(&createDomainAssociationResponse{DomainAssociation: domainToJSON(domain)})
}

// ---- CreateWebhook ----

type createWebhookRequest struct {
	AppId       string            `json:"appId"`
	BranchName  string            `json:"branchName"`
	Description string            `json:"description"`
	Tags        map[string]string `json:"tags"`
}

type webhookJSON struct {
	WebhookArn  string `json:"webhookArn"`
	WebhookId   string `json:"webhookId"`
	WebhookUrl  string `json:"webhookUrl"`
	BranchName  string `json:"branchName"`
	Description string `json:"description,omitempty"`
	CreateTime  string `json:"createTime"`
	UpdateTime  string `json:"updateTime"`
}

func webhookToJSON(wh *Webhook) webhookJSON {
	return webhookJSON{
		WebhookArn: wh.WebhookArn, WebhookId: wh.WebhookId,
		WebhookUrl: wh.WebhookUrl, BranchName: wh.BranchName,
		Description: wh.Description,
		CreateTime: wh.CreateTime.Format("2006-01-02T15:04:05Z"),
		UpdateTime: wh.UpdateTime.Format("2006-01-02T15:04:05Z"),
	}
}

type createWebhookResponse struct {
	Webhook webhookJSON `json:"webhook"`
}

func handleCreateWebhook(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req createWebhookRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	if appID == "" || req.BranchName == "" {
		return jsonErr(service.ErrValidation("appId and branchName are required."))
	}
	wh, ok := store.CreateWebhook(appID, req.BranchName, req.Description, req.Tags)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "App not found.", http.StatusNotFound))
	}
	return jsonOK(&createWebhookResponse{Webhook: webhookToJSON(wh)})
}

// ---- GetWebhook ----

type getWebhookRequest struct {
	WebhookId string `json:"webhookId"`
}

type getWebhookResponse struct {
	Webhook webhookJSON `json:"webhook"`
}

func handleGetWebhook(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getWebhookRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	whID := req.WebhookId
	if whID == "" {
		whID = ctx.Params["webhookId"]
	}
	if whID == "" {
		return jsonErr(service.ErrValidation("webhookId is required."))
	}
	wh, ok := store.GetWebhook(whID)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Webhook not found.", http.StatusNotFound))
	}
	return jsonOK(&getWebhookResponse{Webhook: webhookToJSON(wh)})
}

// ---- ListWebhooks ----

type listWebhooksRequest struct {
	AppId string `json:"appId"`
}

type listWebhooksResponse struct {
	Webhooks []webhookJSON `json:"webhooks"`
}

func handleListWebhooks(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listWebhooksRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	if appID == "" {
		return jsonErr(service.ErrValidation("appId is required."))
	}
	webhooks, ok := store.ListWebhooks(appID)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "App not found.", http.StatusNotFound))
	}
	items := make([]webhookJSON, 0, len(webhooks))
	for _, wh := range webhooks {
		items = append(items, webhookToJSON(wh))
	}
	return jsonOK(&listWebhooksResponse{Webhooks: items})
}

// ---- UpdateWebhook ----

type updateWebhookRequest struct {
	WebhookId   string `json:"webhookId"`
	BranchName  string `json:"branchName"`
	Description string `json:"description"`
}

func handleUpdateWebhook(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req updateWebhookRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	whID := req.WebhookId
	if whID == "" {
		whID = ctx.Params["webhookId"]
	}
	if whID == "" {
		return jsonErr(service.ErrValidation("webhookId is required."))
	}
	wh, ok := store.UpdateWebhook(whID, req.BranchName, req.Description)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Webhook not found.", http.StatusNotFound))
	}
	return jsonOK(&createWebhookResponse{Webhook: webhookToJSON(wh)})
}

// ---- DeleteWebhook ----

type deleteWebhookRequest struct {
	WebhookId string `json:"webhookId"`
}

func handleDeleteWebhook(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req deleteWebhookRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	whID := req.WebhookId
	if whID == "" {
		whID = ctx.Params["webhookId"]
	}
	if whID == "" {
		return jsonErr(service.ErrValidation("webhookId is required."))
	}
	wh, ok := store.GetWebhook(whID)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Webhook not found.", http.StatusNotFound))
	}
	store.DeleteWebhook(whID)
	return jsonOK(&createWebhookResponse{Webhook: webhookToJSON(wh)})
}

// ---- StartJob ----

type startJobRequest struct {
	AppId         string            `json:"appId"`
	BranchName    string            `json:"branchName"`
	JobType       string            `json:"jobType"`
	CommitId      string            `json:"commitId"`
	CommitMessage string            `json:"commitMessage"`
	Tags          map[string]string `json:"tags"`
}

type jobSummaryJSON struct {
	JobArn        string `json:"jobArn"`
	JobId         string `json:"jobId"`
	JobType       string `json:"jobType"`
	Status        string `json:"status"`
	CommitId      string `json:"commitId,omitempty"`
	CommitMessage string `json:"commitMessage,omitempty"`
	StartTime     string `json:"startTime"`
	EndTime       string `json:"endTime,omitempty"`
}

func jobToSummary(j *Job) jobSummaryJSON {
	s := jobSummaryJSON{
		JobArn: j.JobArn, JobId: j.JobId, JobType: j.JobType,
		Status: j.Status, CommitId: j.CommitId,
		CommitMessage: j.CommitMessage,
		StartTime: j.StartTime.Format("2006-01-02T15:04:05Z"),
	}
	if j.EndTime != nil {
		s.EndTime = j.EndTime.Format("2006-01-02T15:04:05Z")
	}
	return s
}

type startJobResponse struct {
	JobSummary jobSummaryJSON `json:"jobSummary"`
}

func handleStartJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req startJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	branchName := req.BranchName
	if branchName == "" {
		branchName = ctx.Params["branchName"]
	}
	if appID == "" || branchName == "" {
		return jsonErr(service.ErrValidation("appId and branchName are required."))
	}
	job, ok := store.StartJob(appID, branchName, req.JobType, req.CommitId, req.CommitMessage, req.Tags)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "App or branch not found.", http.StatusNotFound))
	}
	return jsonOK(&startJobResponse{JobSummary: jobToSummary(job)})
}

// ---- GetJob ----

type getJobRequest struct {
	AppId      string `json:"appId"`
	BranchName string `json:"branchName"`
	JobId      string `json:"jobId"`
}

type getJobResponse struct {
	Job struct {
		Summary jobSummaryJSON `json:"summary"`
		Steps   []any          `json:"steps"`
	} `json:"job"`
}

func handleGetJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req getJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	branchName := req.BranchName
	if branchName == "" {
		branchName = ctx.Params["branchName"]
	}
	jobID := req.JobId
	if jobID == "" {
		jobID = ctx.Params["jobId"]
	}
	if appID == "" || branchName == "" || jobID == "" {
		return jsonErr(service.ErrValidation("appId, branchName, and jobId are required."))
	}
	job, ok := store.GetJob(appID, branchName, jobID)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Job not found.", http.StatusNotFound))
	}
	resp := getJobResponse{}
	resp.Job.Summary = jobToSummary(job)
	resp.Job.Steps = []any{}
	return jsonOK(&resp)
}

// ---- ListJobs ----

type listJobsRequest struct {
	AppId      string `json:"appId"`
	BranchName string `json:"branchName"`
}

type listJobsResponse struct {
	JobSummaries []jobSummaryJSON `json:"jobSummaries"`
}

func handleListJobs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listJobsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	branchName := req.BranchName
	if branchName == "" {
		branchName = ctx.Params["branchName"]
	}
	if appID == "" || branchName == "" {
		return jsonErr(service.ErrValidation("appId and branchName are required."))
	}
	jobs, ok := store.ListJobs(appID, branchName)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "App or branch not found.", http.StatusNotFound))
	}
	items := make([]jobSummaryJSON, 0, len(jobs))
	for _, j := range jobs {
		items = append(items, jobToSummary(j))
	}
	return jsonOK(&listJobsResponse{JobSummaries: items})
}

// ---- StopJob ----

type stopJobRequest struct {
	AppId      string `json:"appId"`
	BranchName string `json:"branchName"`
	JobId      string `json:"jobId"`
}

func handleStopJob(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req stopJobRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	appID := req.AppId
	if appID == "" {
		appID = ctx.Params["appId"]
	}
	branchName := req.BranchName
	if branchName == "" {
		branchName = ctx.Params["branchName"]
	}
	jobID := req.JobId
	if jobID == "" {
		jobID = ctx.Params["jobId"]
	}
	if appID == "" || branchName == "" || jobID == "" {
		return jsonErr(service.ErrValidation("appId, branchName, and jobId are required."))
	}
	job, ok := store.StopJob(appID, branchName, jobID)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Job not found.", http.StatusNotFound))
	}
	return jsonOK(&startJobResponse{JobSummary: jobToSummary(job)})
}

// ---- TagResource ----

type tagResourceRequest struct {
	ResourceArn string            `json:"resourceArn"`
	Tags        map[string]string `json:"tags"`
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req tagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := req.ResourceArn
	if arn == "" {
		arn = ctx.Params["resourceArn"]
	}
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	if !store.TagResource(arn, req.Tags) {
		return jsonErr(service.NewAWSError("NotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- UntagResource ----

type untagResourceRequest struct {
	ResourceArn string   `json:"resourceArn"`
	TagKeys     []string `json:"tagKeys"`
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req untagResourceRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := req.ResourceArn
	if arn == "" {
		arn = ctx.Params["resourceArn"]
	}
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	if !store.UntagResource(arn, req.TagKeys) {
		return jsonErr(service.NewAWSError("NotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(struct{}{})
}

// ---- ListTagsForResource ----

type listTagsRequest struct {
	ResourceArn string `json:"resourceArn"`
}

type listTagsResponse struct {
	Tags map[string]string `json:"tags"`
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req listTagsRequest
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := req.ResourceArn
	if arn == "" {
		arn = ctx.Params["resourceArn"]
	}
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	tags, ok := store.ListTagsForResource(arn)
	if !ok {
		return jsonErr(service.NewAWSError("NotFoundException", "Resource not found.", http.StatusNotFound))
	}
	return jsonOK(&listTagsResponse{Tags: tags})
}
