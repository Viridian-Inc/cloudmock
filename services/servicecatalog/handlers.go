package servicecatalog

import (
	"net/http"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── HTTP / JSON helpers ─────────────────────────────────────────────────────

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
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidParametersException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

// ── Map / value helpers (Service Catalog uses PascalCase) ───────────────────

func getStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getStrPtr(m map[string]any, key string) *string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return &s
		}
	}
	return nil
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getBoolPtr(m map[string]any, key string) *bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return &b
		}
	}
	return nil
}

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if mm, ok := v.(map[string]any); ok {
			return mm
		}
	}
	return nil
}

func getStrMap(m map[string]any, key string) map[string]string {
	mm := getMap(m, key)
	if mm == nil {
		return nil
	}
	out := make(map[string]string, len(mm))
	for k, v := range mm {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

func getMapList(m map[string]any, key string) []map[string]any {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, x := range arr {
		if xm, ok := x.(map[string]any); ok {
			out = append(out, xm)
		}
	}
	return out
}

func getStrList(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, x := range arr {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// parseTagList parses the Service Catalog Tags array of {Key, Value} entries.
func parseTagList(m map[string]any, key string) map[string]string {
	out := make(map[string]string)
	for _, t := range getMapList(m, key) {
		k := getStr(t, "Key")
		v := getStr(t, "Value")
		if k != "" {
			out[k] = v
		}
	}
	return out
}

func tagListFromMap(m map[string]string) []map[string]any {
	out := make([]map[string]any, 0, len(m))
	for k, v := range m {
		out = append(out, map[string]any{"Key": k, "Value": v})
	}
	return out
}

func rfc3339(t time.Time) string { return t.Format(time.RFC3339) }

// ── Resource → response converters ──────────────────────────────────────────

func portfolioDetail(p *StoredPortfolio) map[string]any {
	return map[string]any{
		"Id":           p.Id,
		"ARN":          p.Arn,
		"DisplayName":  p.DisplayName,
		"Description":  p.Description,
		"ProviderName": p.ProviderName,
		"CreatedTime":  rfc3339(p.CreatedTime),
	}
}

func productViewDetail(p *StoredProduct) map[string]any {
	return map[string]any{
		"ProductViewSummary": productViewSummary(p),
		"Status":             p.Status,
		"ProductARN":         p.Arn,
		"CreatedTime":        rfc3339(p.CreatedTime),
	}
}

func productViewSummary(p *StoredProduct) map[string]any {
	return map[string]any{
		"Id":               p.Id,
		"ProductId":        p.Id,
		"Name":             p.Name,
		"Owner":            p.Owner,
		"ShortDescription": p.ShortDescription,
		"Type":             p.Type,
		"Distributor":      p.Distributor,
		"HasDefaultPath":   false,
		"SupportEmail":     p.SupportEmail,
		"SupportDescription": p.SupportDescription,
		"SupportUrl":       p.SupportUrl,
	}
}

func provisioningArtifactDetail(pa *StoredProvisioningArtifact) map[string]any {
	return map[string]any{
		"Id":          pa.Id,
		"Name":        pa.Name,
		"Description": pa.Description,
		"Type":        pa.Type,
		"CreatedTime": rfc3339(pa.CreatedTime),
		"Active":      pa.Active,
		"Guidance":    pa.Guidance,
	}
}

func provisioningArtifactSummary(pa *StoredProvisioningArtifact) map[string]any {
	return map[string]any{
		"Id":          pa.Id,
		"Name":        pa.Name,
		"Description": pa.Description,
		"CreatedTime": rfc3339(pa.CreatedTime),
		"ProvisioningArtifactMetadata": map[string]any{},
	}
}

func constraintDetail(c *StoredConstraint) map[string]any {
	return map[string]any{
		"ConstraintId": c.Id,
		"Type":         c.Type,
		"Description":  c.Description,
		"Owner":        c.Owner,
		"PortfolioId":  c.PortfolioId,
		"ProductId":    c.ProductId,
	}
}

func tagOptionDetail(t *StoredTagOption) map[string]any {
	return map[string]any{
		"Id":     t.Id,
		"Key":    t.Key,
		"Value":  t.Value,
		"Active": t.Active,
		"Owner":  t.Owner,
	}
}

func provisionedProductDetail(pp *StoredProvisionedProduct) map[string]any {
	return map[string]any{
		"Id":            pp.Id,
		"Name":          pp.Name,
		"Type":          pp.Type,
		"Arn":           pp.Arn,
		"Status":        pp.Status,
		"StatusMessage": pp.StatusMessage,
		"CreatedTime":   rfc3339(pp.CreatedTime),
		"LastRecordId":  pp.LastRecordId,
		"LastProvisioningRecordId": pp.LastRecordId,
		"LastSuccessfulProvisioningRecordId": pp.LastRecordId,
		"ProductId":     pp.ProductId,
		"ProductName":   pp.ProductName,
		"ProvisioningArtifactId":   pp.ProvisioningArtifactId,
		"ProvisioningArtifactName": pp.ProvisioningArtifactName,
		"UserArn":       pp.UserArn,
		"UserArnSession": pp.UserArnSession,
		"LaunchRoleArn": pp.LaunchRoleArn,
		"IdempotencyToken": pp.IdempotencyToken,
	}
}

func provisionedProductAttribute(pp *StoredProvisionedProduct) map[string]any {
	out := provisionedProductDetail(pp)
	out["Tags"] = tagListFromMap(pp.Tags)
	return out
}

func recordDetail(r *StoredRecord) map[string]any {
	return map[string]any{
		"RecordId":               r.Id,
		"ProvisionedProductName": r.ProvisionedProductName,
		"ProvisionedProductType": r.ProvisionedProductType,
		"RecordType":             r.RecordType,
		"ProvisionedProductId":   r.ProvisionedProductId,
		"Status":                 r.Status,
		"CreatedTime":            rfc3339(r.CreatedTime),
		"UpdatedTime":            rfc3339(r.UpdatedTime),
		"ProductId":              r.ProductId,
		"ProvisioningArtifactId": r.ProvisioningArtifactId,
		"PathId":                 r.PathId,
		"RecordErrors":           r.RecordErrors,
		"RecordTags":             tagListFromMap(r.RecordTags),
	}
}

func serviceActionDetail(sa *StoredServiceAction) map[string]any {
	return map[string]any{
		"ServiceActionSummary": map[string]any{
			"Id":             sa.Id,
			"Name":           sa.Name,
			"Description":    sa.Description,
			"DefinitionType": sa.DefinitionType,
		},
		"Definition": sa.Definition,
	}
}

func planDetail(p *StoredPlan) map[string]any {
	return map[string]any{
		"PlanName":               p.Name,
		"PlanId":                 p.Id,
		"ProductId":              p.ProductId,
		"PlanType":               p.Type,
		"PathId":                 p.PathId,
		"ProvisioningArtifactId": p.ProvisioningArtifactId,
		"ProvisionedProductName": p.ProvisionedProductName,
		"NotificationArns":       p.NotificationArns,
		"ProvisioningParameters": p.ProvisioningParameters,
		"Tags":                   tagListFromMap(p.Tags),
		"Status":                 p.Status,
		"StatusMessage":          p.StatusMessage,
		"UpdatedTime":            rfc3339(p.UpdatedTime),
	}
}

func planSummary(p *StoredPlan) map[string]any {
	return map[string]any{
		"PlanName":               p.Name,
		"PlanId":                 p.Id,
		"PlanType":               p.Type,
		"ProvisionProductId":     p.ProductId,
		"ProvisionProductName":   p.ProvisionedProductName,
		"ProvisioningArtifactId": p.ProvisioningArtifactId,
	}
}

// ── Portfolio handlers ─────────────────────────────────────────────────────

func handleCreatePortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	p, err := store.CreatePortfolio(
		getStr(req, "DisplayName"),
		getStr(req, "Description"),
		getStr(req, "ProviderName"),
		parseTagList(req, "Tags"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"PortfolioDetail": portfolioDetail(p),
		"Tags":            tagListFromMap(p.Tags),
	})
}

func handleDeletePortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.DeletePortfolio(getStr(req, "Id")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDescribePortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	p, err := store.GetPortfolio(getStr(req, "Id"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"PortfolioDetail": portfolioDetail(p),
		"Tags":            tagListFromMap(p.Tags),
		"TagOptions":      []any{},
		"Budgets":         []any{},
	})
}

func handleListPortfolios(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListPortfolios()
	out := make([]map[string]any, 0, len(list))
	for _, p := range list {
		out = append(out, portfolioDetail(p))
	}
	return jsonOK(map[string]any{"PortfolioDetails": out})
}

func handleListPortfoliosForProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListPortfoliosForProduct(getStr(req, "ProductId"))
	out := make([]map[string]any, 0, len(list))
	for _, p := range list {
		out = append(out, portfolioDetail(p))
	}
	return jsonOK(map[string]any{"PortfolioDetails": out})
}

func handleUpdatePortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	removeKeys := getStrList(req, "RemoveTags")
	addTags := parseTagList(req, "AddTags")
	p, err := store.UpdatePortfolio(
		getStr(req, "Id"),
		getStrPtr(req, "DisplayName"),
		getStrPtr(req, "Description"),
		getStrPtr(req, "ProviderName"),
		addTags,
		removeKeys,
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"PortfolioDetail": portfolioDetail(p),
		"Tags":            tagListFromMap(p.Tags),
	})
}

// ── Portfolio share handlers ───────────────────────────────────────────────

func handleCreatePortfolioShare(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	accountID := getStr(req, "AccountId")
	shareType := "ACCOUNT"
	if node := getMap(req, "OrganizationNode"); node != nil {
		shareType = getStr(node, "Type")
		accountID = getStr(node, "Value")
		if shareType == "" {
			shareType = "ORGANIZATION"
		}
	}
	token, err := store.CreatePortfolioShare(
		getStr(req, "PortfolioId"),
		accountID,
		shareType,
		getBool(req, "SharePrincipals"),
		getBool(req, "ShareTagOptions"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"PortfolioShareToken": token})
}

func handleDeletePortfolioShare(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	accountID := getStr(req, "AccountId")
	shareType := "ACCOUNT"
	if node := getMap(req, "OrganizationNode"); node != nil {
		shareType = getStr(node, "Type")
		accountID = getStr(node, "Value")
		if shareType == "" {
			shareType = "ORGANIZATION"
		}
	}
	token, err := store.DeletePortfolioShare(getStr(req, "PortfolioId"), accountID, shareType)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"PortfolioShareToken": token})
}

func handleDescribePortfolioShares(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	shares := store.ListPortfolioShares(getStr(req, "PortfolioId"), getStr(req, "Type"))
	out := make([]map[string]any, 0, len(shares))
	for _, sh := range shares {
		out = append(out, map[string]any{
			"PrincipalId":     sh.AccountId,
			"Type":            sh.Type,
			"Accepted":        sh.Accepted,
			"ShareTagOptions": sh.ShareTagOptions,
			"SharePrincipals": sh.SharePrincipals,
		})
	}
	return jsonOK(map[string]any{"PortfolioShareDetails": out})
}

func handleDescribePortfolioShareStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	token := getStr(req, "PortfolioShareToken")
	return jsonOK(map[string]any{
		"PortfolioShareToken": token,
		"Status":              "COMPLETED",
		"ShareDetails": map[string]any{
			"SuccessfulShares": []any{},
			"ShareErrors":      []any{},
		},
	})
}

func handleListPortfolioAccess(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	shares := store.ListPortfolioShares(getStr(req, "PortfolioId"), "ACCOUNT")
	accounts := make([]string, 0, len(shares))
	for _, sh := range shares {
		accounts = append(accounts, sh.AccountId)
	}
	return jsonOK(map[string]any{"AccountIds": accounts})
}

func handleUpdatePortfolioShare(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	accountID := getStr(req, "AccountId")
	shareType := "ACCOUNT"
	if node := getMap(req, "OrganizationNode"); node != nil {
		shareType = getStr(node, "Type")
		accountID = getStr(node, "Value")
		if shareType == "" {
			shareType = "ORGANIZATION"
		}
	}
	token, err := store.UpdatePortfolioShare(
		getStr(req, "PortfolioId"),
		accountID,
		shareType,
		getBoolPtr(req, "SharePrincipals"),
		getBoolPtr(req, "ShareTagOptions"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"PortfolioShareToken": token,
		"Status":              "COMPLETED",
	})
}

func handleAcceptPortfolioShare(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.AcceptPortfolioShare(getStr(req, "PortfolioId"), getStr(req, "PortfolioShareType")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleRejectPortfolioShare(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.RejectPortfolioShare(getStr(req, "PortfolioId"), getStr(req, "PortfolioShareType")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListAcceptedPortfolioShares(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	out := make([]map[string]any, 0)
	for _, p := range store.ListPortfolios() {
		shares := store.ListPortfolioShares(p.Id, "")
		for _, sh := range shares {
			if sh.Accepted {
				out = append(out, portfolioDetail(p))
				break
			}
		}
	}
	return jsonOK(map[string]any{"PortfolioDetails": out})
}

func handleListOrganizationPortfolioAccess(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	shareType := getStr(req, "OrganizationNodeType")
	shares := store.ListPortfolioShares(getStr(req, "PortfolioId"), shareType)
	out := make([]map[string]any, 0, len(shares))
	for _, sh := range shares {
		out = append(out, map[string]any{
			"Type":  sh.Type,
			"Value": sh.AccountId,
		})
	}
	return jsonOK(map[string]any{"OrganizationNodes": out})
}

// ── Product handlers ───────────────────────────────────────────────────────

func handleCreateProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	pap := getMap(req, "ProvisioningArtifactParameters")
	var artName, artDesc, artType string
	var info map[string]string
	if pap != nil {
		artName = getStr(pap, "Name")
		artDesc = getStr(pap, "Description")
		artType = getStr(pap, "Type")
		info = getStrMap(pap, "Info")
	}
	prod, pa, err := store.CreateProduct(
		getStr(req, "Name"),
		getStr(req, "Owner"),
		getStr(req, "Description"),
		getStr(req, "Distributor"),
		getStr(req, "ProductType"),
		getStr(req, "SupportDescription"),
		getStr(req, "SupportEmail"),
		getStr(req, "SupportUrl"),
		parseTagList(req, "Tags"),
		artName, artDesc, artType, info,
	)
	if err != nil {
		return jsonErr(err)
	}
	resp := map[string]any{
		"ProductViewDetail": productViewDetail(prod),
		"Tags":              tagListFromMap(prod.Tags),
	}
	if pa != nil {
		resp["ProvisioningArtifactDetail"] = provisioningArtifactDetail(pa)
	}
	return jsonOK(resp)
}

func handleDeleteProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.DeleteProduct(getStr(req, "Id")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDescribeProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "Id")
	name := getStr(req, "Name")
	var p *StoredProduct
	var err *service.AWSError
	if id != "" {
		p, err = store.GetProduct(id)
	} else if name != "" {
		p, err = store.GetProductByName(name)
	} else {
		return jsonErr(errInvalidParam("Id or Name is required"))
	}
	if err != nil {
		return jsonErr(err)
	}
	pas, _ := store.ListProvisioningArtifacts(p.Id)
	artifacts := make([]map[string]any, 0, len(pas))
	for _, pa := range pas {
		artifacts = append(artifacts, provisioningArtifactDetail(pa))
	}
	return jsonOK(map[string]any{
		"ProductViewSummary":     productViewSummary(p),
		"ProvisioningArtifacts":  artifacts,
		"Budgets":                []any{},
		"LaunchPaths":            []any{},
	})
}

func handleDescribeProductAsAdmin(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "Id")
	name := getStr(req, "Name")
	var p *StoredProduct
	var err *service.AWSError
	if id != "" {
		p, err = store.GetProduct(id)
	} else if name != "" {
		p, err = store.GetProductByName(name)
	} else {
		return jsonErr(errInvalidParam("Id or Name is required"))
	}
	if err != nil {
		return jsonErr(err)
	}
	pas, _ := store.ListProvisioningArtifacts(p.Id)
	summaries := make([]map[string]any, 0, len(pas))
	for _, pa := range pas {
		summaries = append(summaries, provisioningArtifactSummary(pa))
	}
	return jsonOK(map[string]any{
		"ProductViewDetail":             productViewDetail(p),
		"ProvisioningArtifactSummaries": summaries,
		"Tags":                          tagListFromMap(p.Tags),
		"TagOptions":                    []any{},
		"Budgets":                       []any{},
	})
}

func handleDescribeProductView(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	p, err := store.GetProduct(getStr(req, "Id"))
	if err != nil {
		return jsonErr(err)
	}
	pas, _ := store.ListProvisioningArtifacts(p.Id)
	views := make([]map[string]any, 0, len(pas))
	for _, pa := range pas {
		views = append(views, map[string]any{
			"Id":          pa.Id,
			"Name":        pa.Name,
			"Description": pa.Description,
			"Type":        pa.Type,
			"CreatedTime": rfc3339(pa.CreatedTime),
		})
	}
	return jsonOK(map[string]any{
		"ProductViewSummary":    productViewSummary(p),
		"ProvisioningArtifacts": views,
	})
}

func handleSearchProducts(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	_ = req
	list := store.ListProducts()
	views := make([]map[string]any, 0, len(list))
	for _, p := range list {
		views = append(views, productViewSummary(p))
	}
	return jsonOK(map[string]any{
		"ProductViewSummaries":   views,
		"ProductViewAggregations": map[string]any{},
	})
}

func handleSearchProductsAsAdmin(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListProducts()
	details := make([]map[string]any, 0, len(list))
	for _, p := range list {
		details = append(details, productViewDetail(p))
	}
	return jsonOK(map[string]any{"ProductViewDetails": details})
}

func handleUpdateProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	p, err := store.UpdateProduct(
		getStr(req, "Id"),
		getStrPtr(req, "Name"),
		getStrPtr(req, "Owner"),
		getStrPtr(req, "Description"),
		getStrPtr(req, "Distributor"),
		getStrPtr(req, "SupportDescription"),
		getStrPtr(req, "SupportEmail"),
		getStrPtr(req, "SupportUrl"),
		parseTagList(req, "AddTags"),
		getStrList(req, "RemoveTags"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"ProductViewDetail": productViewDetail(p),
		"Tags":              tagListFromMap(p.Tags),
	})
}

// ── Provisioning artifact handlers ─────────────────────────────────────────

func handleCreateProvisioningArtifact(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	params := getMap(req, "Parameters")
	if params == nil {
		return jsonErr(errInvalidParam("Parameters is required"))
	}
	pa, err := store.CreateProvisioningArtifact(
		getStr(req, "ProductId"),
		getStr(params, "Name"),
		getStr(params, "Description"),
		getStr(params, "Type"),
		getStrMap(params, "Info"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"ProvisioningArtifactDetail": provisioningArtifactDetail(pa),
		"Info":                       pa.Info,
		"Status":                     "AVAILABLE",
	})
}

func handleDeleteProvisioningArtifact(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.DeleteProvisioningArtifact(getStr(req, "ProductId"), getStr(req, "ProvisioningArtifactId")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDescribeProvisioningArtifact(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	pa, err := store.GetProvisioningArtifact(getStr(req, "ProductId"), getStr(req, "ProvisioningArtifactId"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"ProvisioningArtifactDetail": provisioningArtifactDetail(pa),
		"Info":                       pa.Info,
		"Status":                     "AVAILABLE",
	})
}

func handleListProvisioningArtifacts(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	pas, err := store.ListProvisioningArtifacts(getStr(req, "ProductId"))
	if err != nil {
		return jsonErr(err)
	}
	out := make([]map[string]any, 0, len(pas))
	for _, pa := range pas {
		out = append(out, provisioningArtifactDetail(pa))
	}
	return jsonOK(map[string]any{"ProvisioningArtifactDetails": out})
}

func handleUpdateProvisioningArtifact(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	pa, err := store.UpdateProvisioningArtifact(
		getStr(req, "ProductId"),
		getStr(req, "ProvisioningArtifactId"),
		getStrPtr(req, "Name"),
		getStrPtr(req, "Description"),
		getStrPtr(req, "Guidance"),
		getBoolPtr(req, "Active"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"ProvisioningArtifactDetail": provisioningArtifactDetail(pa),
		"Info":                       pa.Info,
		"Status":                     "AVAILABLE",
	})
}

// ── Constraint handlers ────────────────────────────────────────────────────

func handleCreateConstraint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	c, err := store.CreateConstraint(
		getStr(req, "PortfolioId"),
		getStr(req, "ProductId"),
		getStr(req, "Type"),
		getStr(req, "Parameters"),
		getStr(req, "Description"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"ConstraintDetail":     constraintDetail(c),
		"ConstraintParameters": c.Parameters,
		"Status":               c.Status,
	})
}

func handleDeleteConstraint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.DeleteConstraint(getStr(req, "Id")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDescribeConstraint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	c, err := store.GetConstraint(getStr(req, "Id"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"ConstraintDetail":     constraintDetail(c),
		"ConstraintParameters": c.Parameters,
		"Status":               c.Status,
	})
}

func handleListConstraintsForPortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListConstraintsForPortfolio(getStr(req, "PortfolioId"), getStr(req, "ProductId"))
	out := make([]map[string]any, 0, len(list))
	for _, c := range list {
		out = append(out, constraintDetail(c))
	}
	return jsonOK(map[string]any{"ConstraintDetails": out})
}

func handleUpdateConstraint(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	c, err := store.UpdateConstraint(
		getStr(req, "Id"),
		getStrPtr(req, "Description"),
		getStrPtr(req, "Parameters"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"ConstraintDetail":     constraintDetail(c),
		"ConstraintParameters": c.Parameters,
		"Status":               c.Status,
	})
}

// ── Principal handlers ─────────────────────────────────────────────────────

func handleAssociatePrincipalWithPortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.AssociatePrincipal(
		getStr(req, "PortfolioId"),
		getStr(req, "PrincipalARN"),
		getStr(req, "PrincipalType"),
	); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDisassociatePrincipalFromPortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.DisassociatePrincipal(
		getStr(req, "PortfolioId"),
		getStr(req, "PrincipalARN"),
	); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListPrincipalsForPortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListPrincipals(getStr(req, "PortfolioId"))
	out := make([]map[string]any, 0, len(list))
	for _, p := range list {
		out = append(out, map[string]any{
			"PrincipalARN":  p.PrincipalARN,
			"PrincipalType": p.PrincipalType,
		})
	}
	return jsonOK(map[string]any{"Principals": out})
}

// ── Tag option handlers ────────────────────────────────────────────────────

func handleCreateTagOption(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	t, err := store.CreateTagOption(getStr(req, "Key"), getStr(req, "Value"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"TagOptionDetail": tagOptionDetail(t)})
}

func handleDeleteTagOption(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.DeleteTagOption(getStr(req, "Id")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDescribeTagOption(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	t, err := store.GetTagOption(getStr(req, "Id"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"TagOptionDetail": tagOptionDetail(t)})
}

func handleListTagOptions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	filters := getMap(req, "Filters")
	var key, value string
	var active *bool
	if filters != nil {
		key = getStr(filters, "Key")
		value = getStr(filters, "Value")
		active = getBoolPtr(filters, "Active")
	}
	list := store.ListTagOptions(key, value, active)
	out := make([]map[string]any, 0, len(list))
	for _, t := range list {
		out = append(out, tagOptionDetail(t))
	}
	return jsonOK(map[string]any{"TagOptionDetails": out})
}

func handleUpdateTagOption(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	t, err := store.UpdateTagOption(
		getStr(req, "Id"),
		getStrPtr(req, "Value"),
		getBoolPtr(req, "Active"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"TagOptionDetail": tagOptionDetail(t)})
}

func handleAssociateTagOptionWithResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.AssociateTagOption(getStr(req, "ResourceId"), getStr(req, "TagOptionId")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDisassociateTagOptionFromResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.DisassociateTagOption(getStr(req, "ResourceId"), getStr(req, "TagOptionId")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListResourcesForTagOption(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	resources := store.ListResourcesForTagOption(getStr(req, "TagOptionId"))
	out := make([]map[string]any, 0, len(resources))
	for _, id := range resources {
		out = append(out, map[string]any{
			"Id":          id,
			"Name":        id,
			"Description": "",
			"CreatedTime": rfc3339(time.Now().UTC()),
		})
	}
	return jsonOK(map[string]any{"ResourceDetails": out})
}

// ── Portfolio ↔ Product association handlers ───────────────────────────────

func handleAssociateProductWithPortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.AssociateProductWithPortfolio(getStr(req, "PortfolioId"), getStr(req, "ProductId")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDisassociateProductFromPortfolio(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.DisassociateProductFromPortfolio(getStr(req, "PortfolioId"), getStr(req, "ProductId")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── Service action association handlers ────────────────────────────────────

func handleAssociateServiceActionWithProvisioningArtifact(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.AssociateServiceActionWithArtifact(
		getStr(req, "ProductId"),
		getStr(req, "ProvisioningArtifactId"),
		getStr(req, "ServiceActionId"),
	); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDisassociateServiceActionFromProvisioningArtifact(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.DisassociateServiceActionFromArtifact(
		getStr(req, "ProductId"),
		getStr(req, "ProvisioningArtifactId"),
		getStr(req, "ServiceActionId"),
	); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleBatchAssociateServiceActionWithProvisioningArtifact(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	failed := make([]map[string]any, 0)
	for _, assoc := range getMapList(req, "ServiceActionAssociations") {
		err := store.AssociateServiceActionWithArtifact(
			getStr(assoc, "ProductId"),
			getStr(assoc, "ProvisioningArtifactId"),
			getStr(assoc, "ServiceActionId"),
		)
		if err != nil {
			failed = append(failed, map[string]any{
				"ServiceActionId":         getStr(assoc, "ServiceActionId"),
				"ProductId":               getStr(assoc, "ProductId"),
				"ProvisioningArtifactId":  getStr(assoc, "ProvisioningArtifactId"),
				"ErrorCode":               err.Code,
				"ErrorMessage":            err.Message,
			})
		}
	}
	return jsonOK(map[string]any{"FailedServiceActionAssociations": failed})
}

func handleBatchDisassociateServiceActionFromProvisioningArtifact(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	failed := make([]map[string]any, 0)
	for _, assoc := range getMapList(req, "ServiceActionAssociations") {
		_ = store.DisassociateServiceActionFromArtifact(
			getStr(assoc, "ProductId"),
			getStr(assoc, "ProvisioningArtifactId"),
			getStr(assoc, "ServiceActionId"),
		)
	}
	return jsonOK(map[string]any{"FailedServiceActionAssociations": failed})
}

func handleListServiceActionsForProvisioningArtifact(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListServiceActionsForArtifact(getStr(req, "ProductId"), getStr(req, "ProvisioningArtifactId"))
	out := make([]map[string]any, 0, len(list))
	for _, sa := range list {
		out = append(out, map[string]any{
			"Id":             sa.Id,
			"Name":           sa.Name,
			"Description":    sa.Description,
			"DefinitionType": sa.DefinitionType,
		})
	}
	return jsonOK(map[string]any{"ServiceActionSummaries": out})
}

func handleListProvisioningArtifactsForServiceAction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	pas := store.ListArtifactsForServiceAction(getStr(req, "ServiceActionId"))
	out := make([]map[string]any, 0, len(pas))
	for _, pa := range pas {
		out = append(out, map[string]any{
			"ProductViewSummary":   map[string]any{"ProductId": pa.ProductId},
			"ProvisioningArtifact": provisioningArtifactDetail(pa),
		})
	}
	return jsonOK(map[string]any{"ProvisioningArtifactViews": out})
}

// ── Service action handlers ────────────────────────────────────────────────

func handleCreateServiceAction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	sa, err := store.CreateServiceAction(
		getStr(req, "Name"),
		getStr(req, "DefinitionType"),
		getStrMap(req, "Definition"),
		getStr(req, "Description"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"ServiceActionDetail": serviceActionDetail(sa)})
}

func handleDeleteServiceAction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.DeleteServiceAction(getStr(req, "Id")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDescribeServiceAction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	sa, err := store.GetServiceAction(getStr(req, "Id"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"ServiceActionDetail": serviceActionDetail(sa)})
}

func handleDescribeServiceActionExecutionParameters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"ServiceActionParameters": []any{}})
}

func handleListServiceActions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListServiceActions()
	out := make([]map[string]any, 0, len(list))
	for _, sa := range list {
		out = append(out, map[string]any{
			"Id":             sa.Id,
			"Name":           sa.Name,
			"Description":    sa.Description,
			"DefinitionType": sa.DefinitionType,
		})
	}
	return jsonOK(map[string]any{"ServiceActionSummaries": out})
}

func handleUpdateServiceAction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	sa, err := store.UpdateServiceAction(
		getStr(req, "Id"),
		getStrPtr(req, "Name"),
		getStrPtr(req, "Description"),
		getStrMap(req, "Definition"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"ServiceActionDetail": serviceActionDetail(sa)})
}

// ── Provisioned product handlers ───────────────────────────────────────────

func handleProvisionProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	pp, rec, err := store.ProvisionProduct(
		getStr(req, "ProvisionedProductName"),
		getStr(req, "ProductId"),
		getStr(req, "ProvisioningArtifactId"),
		getStr(req, "PathId"),
		getStr(req, "ProvisionToken"),
		parseTagList(req, "Tags"),
		"arn:aws:iam::"+store.accountID+":user/cloudmock",
	)
	if err != nil {
		return jsonErr(err)
	}
	_ = pp
	return jsonOK(map[string]any{"RecordDetail": recordDetail(rec)})
}

func handleUpdateProvisionedProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "ProvisionedProductId")
	if id == "" {
		id = getStr(req, "ProvisionedProductName")
	}
	_, rec, err := store.UpdateProvisionedProduct(
		id,
		getStr(req, "ProductId"),
		getStr(req, "ProvisioningArtifactId"),
		getStr(req, "PathId"),
		parseTagList(req, "Tags"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"RecordDetail": recordDetail(rec)})
}

func handleTerminateProvisionedProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "ProvisionedProductId")
	if id == "" {
		id = getStr(req, "ProvisionedProductName")
	}
	rec, err := store.TerminateProvisionedProduct(id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"RecordDetail": recordDetail(rec)})
}

func handleDescribeProvisionedProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "Id")
	if id == "" {
		id = getStr(req, "Name")
	}
	pp, err := store.GetProvisionedProduct(id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"ProvisionedProductDetail": provisionedProductDetail(pp),
		"CloudWatchDashboards":     []any{},
	})
}

func handleScanProvisionedProducts(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListProvisionedProducts()
	out := make([]map[string]any, 0, len(list))
	for _, pp := range list {
		out = append(out, provisionedProductDetail(pp))
	}
	return jsonOK(map[string]any{"ProvisionedProducts": out})
}

func handleSearchProvisionedProducts(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListProvisionedProducts()
	out := make([]map[string]any, 0, len(list))
	for _, pp := range list {
		out = append(out, provisionedProductAttribute(pp))
	}
	return jsonOK(map[string]any{
		"ProvisionedProducts": out,
		"TotalResultsCount":   len(out),
	})
}

func handleListLaunchPaths(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	productID := getStr(req, "ProductId")
	if _, err := store.GetProduct(productID); err != nil {
		return jsonErr(err)
	}
	portfolios := store.ListPortfoliosForProduct(productID)
	paths := make([]map[string]any, 0, len(portfolios))
	for _, p := range portfolios {
		paths = append(paths, map[string]any{
			"Id":          "lpv-" + p.Id,
			"Name":        p.DisplayName,
			"ConstraintSummaries": []any{},
			"Tags":        []any{},
		})
	}
	return jsonOK(map[string]any{"LaunchPathSummaries": paths})
}

func handleDescribeRecord(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	r, err := store.GetRecord(getStr(req, "Id"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"RecordDetail":  recordDetail(r),
		"RecordOutputs": []any{},
	})
}

func handleListRecordHistory(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	var ppID, ppName string
	if filt := getMap(req, "SearchFilter"); filt != nil {
		key := getStr(filt, "Key")
		val := getStr(filt, "Value")
		if key == "provisionedproduct" {
			ppName = val
		} else if key == "provisionedproductid" {
			ppID = val
		}
	}
	records := store.ListRecords(ppID, ppName)
	out := make([]map[string]any, 0, len(records))
	for _, r := range records {
		out = append(out, recordDetail(r))
	}
	return jsonOK(map[string]any{"RecordDetails": out})
}

func handleGetProvisionedProductOutputs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "ProvisionedProductId")
	if id == "" {
		id = getStr(req, "ProvisionedProductName")
	}
	pp, err := store.GetProvisionedProduct(id)
	if err != nil {
		return jsonErr(err)
	}
	outputs := pp.Outputs
	if outputs == nil {
		outputs = []map[string]any{}
	}
	return jsonOK(map[string]any{"Outputs": outputs})
}

func handleExecuteProvisionedProductServiceAction(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	pp, err := store.GetProvisionedProduct(getStr(req, "ProvisionedProductId"))
	if err != nil {
		return jsonErr(err)
	}
	if _, err := store.GetServiceAction(getStr(req, "ServiceActionId")); err != nil {
		return jsonErr(err)
	}
	rec := &StoredRecord{
		Id:                     newID("rec"),
		ProvisionedProductName: pp.Name,
		ProvisionedProductType: pp.Type,
		RecordType:             "EXECUTE_PROVISIONED_PRODUCT_SERVICE_ACTION",
		ProvisionedProductId:   pp.Id,
		Status:                 "SUCCEEDED",
		CreatedTime:             time.Now().UTC(),
		UpdatedTime:             time.Now().UTC(),
	}
	store.mu.Lock()
	store.records[rec.Id] = rec
	store.mu.Unlock()
	return jsonOK(map[string]any{"RecordDetail": recordDetail(rec)})
}

func handleNotifyProvisionProductEngineWorkflowResult(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	_ = req
	return jsonOK(map[string]any{})
}

func handleNotifyTerminateProvisionedProductEngineWorkflowResult(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	_ = req
	return jsonOK(map[string]any{})
}

func handleNotifyUpdateProvisionedProductEngineWorkflowResult(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	_ = req
	return jsonOK(map[string]any{})
}

func handleUpdateProvisionedProductProperties(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	pp, recordID, err := store.UpdateProvisionedProductProperties(
		getStr(req, "ProvisionedProductId"),
		getMap(req, "ProvisionedProductProperties"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"ProvisionedProductId":         pp.Id,
		"ProvisionedProductProperties": getMap(req, "ProvisionedProductProperties"),
		"RecordId":                     recordID,
		"Status":                       "SUCCEEDED",
	})
}

func handleImportAsProvisionedProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	pp, rec, err := store.ProvisionProduct(
		getStr(req, "ProvisionedProductName"),
		getStr(req, "ProductId"),
		getStr(req, "ProvisioningArtifactId"),
		"",
		getStr(req, "IdempotencyToken"),
		nil,
		"arn:aws:iam::"+store.accountID+":user/cloudmock",
	)
	if err != nil {
		return jsonErr(err)
	}
	pp.Type = "CFN_STACK"
	return jsonOK(map[string]any{"RecordDetail": recordDetail(rec)})
}

// ── Plan handlers ──────────────────────────────────────────────────────────

func handleCreateProvisionedProductPlan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	p, err := store.CreatePlan(
		getStr(req, "PlanName"),
		getStr(req, "PlanType"),
		getStr(req, "ProductId"),
		getStr(req, "PathId"),
		getStr(req, "ProvisioningArtifactId"),
		getStr(req, "ProvisionedProductName"),
		getMapList(req, "ProvisioningParameters"),
		getStrList(req, "NotificationArns"),
		parseTagList(req, "Tags"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"PlanName":               p.Name,
		"PlanId":                 p.Id,
		"ProvisionProductId":     p.ProductId,
		"ProvisionedProductName": p.ProvisionedProductName,
		"ProvisioningArtifactId": p.ProvisioningArtifactId,
	})
}

func handleDeleteProvisionedProductPlan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.DeletePlan(getStr(req, "PlanId")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDescribeProvisionedProductPlan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	p, err := store.GetPlan(getStr(req, "PlanId"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"ProvisionedProductPlanDetails": planDetail(p),
		"ResourceChanges":               []any{},
	})
}

func handleListProvisionedProductPlans(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListPlans(getStr(req, "ProvisionProductId"))
	out := make([]map[string]any, 0, len(list))
	for _, p := range list {
		out = append(out, planSummary(p))
	}
	return jsonOK(map[string]any{"ProvisionedProductPlans": out})
}

func handleExecuteProvisionedProductPlan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.ExecutePlan(getStr(req, "PlanId")); err != nil {
		return jsonErr(err)
	}
	rec := &StoredRecord{
		Id:          newID("rec"),
		Status:      "SUCCEEDED",
		RecordType:  "EXECUTE_PROVISIONED_PRODUCT_PLAN",
		CreatedTime: time.Now().UTC(),
		UpdatedTime: time.Now().UTC(),
	}
	store.mu.Lock()
	store.records[rec.Id] = rec
	store.mu.Unlock()
	return jsonOK(map[string]any{"RecordDetail": recordDetail(rec)})
}

// ── Budget handlers ────────────────────────────────────────────────────────

func handleAssociateBudgetWithResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.AssociateBudget(getStr(req, "BudgetName"), getStr(req, "ResourceId")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDisassociateBudgetFromResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.DisassociateBudget(getStr(req, "BudgetName"), getStr(req, "ResourceId")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListBudgetsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	budgets := store.ListBudgetsForResource(getStr(req, "ResourceId"))
	out := make([]map[string]any, 0, len(budgets))
	for _, b := range budgets {
		out = append(out, map[string]any{"BudgetName": b})
	}
	return jsonOK(map[string]any{"Budgets": out})
}

// ── AWS Organizations access ───────────────────────────────────────────────

func handleEnableAWSOrganizationsAccess(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	store.SetAWSOrganizationsAccess("ENABLED")
	return jsonOK(map[string]any{})
}

func handleDisableAWSOrganizationsAccess(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	store.SetAWSOrganizationsAccess("DISABLED")
	return jsonOK(map[string]any{})
}

func handleGetAWSOrganizationsAccessStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{"AccessStatus": store.GetAWSOrganizationsAccess()})
}

// ── Copy product ───────────────────────────────────────────────────────────

func handleCopyProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	source := getStr(req, "SourceProductArn")
	if source == "" {
		return jsonErr(errInvalidParam("SourceProductArn is required"))
	}
	target := getStr(req, "TargetProductId")
	token := store.StartCopyProduct(source, target)
	return jsonOK(map[string]any{"CopyProductToken": token})
}

func handleDescribeCopyProductStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	op, err := store.GetCopyOperation(getStr(req, "CopyProductToken"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"CopyProductStatus": op.Status,
		"StatusDetail":      op.StatusDetail,
		"TargetProductId":   op.TargetProductId,
	})
}

// ── DescribeProvisioningParameters ─────────────────────────────────────────

func handleDescribeProvisioningParameters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	productID := getStr(req, "ProductId")
	if productID == "" {
		productID = getStr(req, "ProductName")
		if productID != "" {
			if p, err := store.GetProductByName(productID); err == nil {
				productID = p.Id
			}
		}
	}
	if _, err := store.GetProduct(productID); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"ProvisioningArtifactParameters":     []any{},
		"ConstraintSummaries":                 []any{},
		"UsageInstructions":                   []any{},
		"TagOptions":                          []any{},
		"ProvisioningArtifactPreferences":     map[string]any{},
		"ProvisioningArtifactOutputs":         []any{},
		"ProvisioningArtifactOutputKeys":      []any{},
	})
}

// ── ListStackInstancesForProvisionedProduct ────────────────────────────────

func handleListStackInstancesForProvisionedProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "ProvisionedProductId")
	if _, err := store.GetProvisionedProduct(id); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"StackInstances": []any{}})
}
