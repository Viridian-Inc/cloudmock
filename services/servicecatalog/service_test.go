package servicecatalog_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/servicecatalog"
)

// ── Fixtures ────────────────────────────────────────────────────────────────

func newGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(svc.New(cfg.AccountID, cfg.Region))
	return gateway.New(cfg, reg)
}

func svcReq(t *testing.T, action string, body any) *http.Request {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWS242ServiceCatalogService."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/servicecatalog/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

// call dispatches an Action against the gateway and returns the parsed JSON body.
func call(t *testing.T, h http.Handler, action string, body any) (int, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, svcReq(t, action, body))
	if w.Body.Len() == 0 {
		return w.Code, map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s: parse response: %v\nbody: %s", action, err, w.Body.String())
	}
	return w.Code, out
}

// callOK requires a 200 response and returns the parsed body.
func callOK(t *testing.T, h http.Handler, action string, body any) map[string]any {
	t.Helper()
	code, out := call(t, h, action, body)
	if code != http.StatusOK {
		t.Fatalf("%s: expected 200, got %d\nbody: %v", action, code, out)
	}
	return out
}

// asMap helper.
func asMap(t *testing.T, v any, key string) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("%s: expected map, got %T", key, v)
	}
	return m
}

// asList helper.
func asList(t *testing.T, v any, key string) []any {
	t.Helper()
	if v == nil {
		return nil
	}
	a, ok := v.([]any)
	if !ok {
		t.Fatalf("%s: expected []any, got %T", key, v)
	}
	return a
}

// ── Portfolio lifecycle ────────────────────────────────────────────────────

func TestPortfolioLifecycle(t *testing.T) {
	h := newGateway(t)

	// Create
	resp := callOK(t, h, "CreatePortfolio", map[string]any{
		"DisplayName":  "Test Portfolio",
		"Description":  "A test portfolio",
		"ProviderName": "cloudmock",
		"Tags": []map[string]any{
			{"Key": "env", "Value": "dev"},
		},
	})
	detail := asMap(t, resp["PortfolioDetail"], "PortfolioDetail")
	id, _ := detail["Id"].(string)
	if id == "" || detail["DisplayName"] != "Test Portfolio" {
		t.Fatalf("CreatePortfolio: bad detail: %v", detail)
	}

	// Describe
	descResp := callOK(t, h, "DescribePortfolio", map[string]any{"Id": id})
	if asMap(t, descResp["PortfolioDetail"], "PortfolioDetail")["DisplayName"] != "Test Portfolio" {
		t.Fatalf("DescribePortfolio: mismatch: %v", descResp)
	}

	// List
	listResp := callOK(t, h, "ListPortfolios", nil)
	if len(asList(t, listResp["PortfolioDetails"], "PortfolioDetails")) != 1 {
		t.Fatalf("ListPortfolios: want 1, got %v", listResp)
	}

	// Update
	descCopy := "Updated description"
	upResp := callOK(t, h, "UpdatePortfolio", map[string]any{
		"Id":          id,
		"Description": descCopy,
	})
	if asMap(t, upResp["PortfolioDetail"], "PortfolioDetail")["Description"] != descCopy {
		t.Fatalf("UpdatePortfolio: bad: %v", upResp)
	}

	// Delete
	callOK(t, h, "DeletePortfolio", map[string]any{"Id": id})

	// Describe after delete returns 400
	code, _ := call(t, h, "DescribePortfolio", map[string]any{"Id": id})
	if code != http.StatusBadRequest {
		t.Fatalf("DescribePortfolio after delete: want 400, got %d", code)
	}
}

func TestCreatePortfolioMissingDisplayName(t *testing.T) {
	h := newGateway(t)
	code, body := call(t, h, "CreatePortfolio", map[string]any{"ProviderName": "cm"})
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%v)", code, body)
	}
	if body["__type"] != "InvalidParametersException" {
		t.Fatalf("expected InvalidParametersException, got %v", body)
	}
}

// ── Portfolio shares ───────────────────────────────────────────────────────

func TestPortfolioShareLifecycle(t *testing.T) {
	h := newGateway(t)
	port := callOK(t, h, "CreatePortfolio", map[string]any{
		"DisplayName": "P", "ProviderName": "cm",
	})
	portID := asMap(t, port["PortfolioDetail"], "PortfolioDetail")["Id"].(string)

	// Create share
	shareResp := callOK(t, h, "CreatePortfolioShare", map[string]any{
		"PortfolioId":     portID,
		"AccountId":       "111122223333",
		"SharePrincipals": true,
	})
	if shareResp["PortfolioShareToken"] == "" {
		t.Fatalf("CreatePortfolioShare: empty token")
	}

	// Describe shares
	descResp := callOK(t, h, "DescribePortfolioShares", map[string]any{
		"PortfolioId": portID,
		"Type":        "ACCOUNT",
	})
	shares := asList(t, descResp["PortfolioShareDetails"], "PortfolioShareDetails")
	if len(shares) != 1 {
		t.Fatalf("DescribePortfolioShares: want 1, got %d", len(shares))
	}

	// List access
	accessResp := callOK(t, h, "ListPortfolioAccess", map[string]any{"PortfolioId": portID})
	if len(asList(t, accessResp["AccountIds"], "AccountIds")) != 1 {
		t.Fatalf("ListPortfolioAccess: bad: %v", accessResp)
	}

	// Accept share
	callOK(t, h, "AcceptPortfolioShare", map[string]any{
		"PortfolioId":        portID,
		"PortfolioShareType": "ACCOUNT",
	})

	// ListAcceptedPortfolioShares now sees one
	listResp := callOK(t, h, "ListAcceptedPortfolioShares", nil)
	if len(asList(t, listResp["PortfolioDetails"], "PortfolioDetails")) != 1 {
		t.Fatalf("ListAcceptedPortfolioShares: want 1, got %v", listResp)
	}

	// Update share
	callOK(t, h, "UpdatePortfolioShare", map[string]any{
		"PortfolioId":     portID,
		"AccountId":       "111122223333",
		"SharePrincipals": false,
	})

	// Delete share
	callOK(t, h, "DeletePortfolioShare", map[string]any{
		"PortfolioId": portID,
		"AccountId":   "111122223333",
	})

	// Reject share on deleted -> 400
	code, _ := call(t, h, "RejectPortfolioShare", map[string]any{
		"PortfolioId":        portID,
		"PortfolioShareType": "ACCOUNT",
	})
	if code != http.StatusBadRequest {
		t.Fatalf("RejectPortfolioShare after delete: want 400, got %d", code)
	}

	// Describe share status (token introspection) — returns COMPLETED for any token
	statusResp := callOK(t, h, "DescribePortfolioShareStatus", map[string]any{
		"PortfolioShareToken": "any-token",
	})
	if statusResp["Status"] != "COMPLETED" {
		t.Fatalf("DescribePortfolioShareStatus: bad: %v", statusResp)
	}
}

// ── Product + provisioning artifact lifecycle ──────────────────────────────

func TestProductLifecycle(t *testing.T) {
	h := newGateway(t)

	// Create product (with initial PA)
	prodResp := callOK(t, h, "CreateProduct", map[string]any{
		"Name":        "TestProd",
		"Owner":       "cloudmock",
		"Description": "a test product",
		"ProductType": "CLOUD_FORMATION_TEMPLATE",
		"ProvisioningArtifactParameters": map[string]any{
			"Name":        "v1",
			"Description": "version 1",
			"Type":        "CLOUD_FORMATION_TEMPLATE",
			"Info":        map[string]any{"LoadTemplateFromURL": "https://example.com/t.json"},
		},
		"Tags": []map[string]any{{"Key": "env", "Value": "dev"}},
	})
	prodID := asMap(t, asMap(t, prodResp["ProductViewDetail"], "ProductViewDetail")["ProductViewSummary"], "ProductViewSummary")["ProductId"].(string)
	paID := asMap(t, prodResp["ProvisioningArtifactDetail"], "ProvisioningArtifactDetail")["Id"].(string)

	// Describe
	descResp := callOK(t, h, "DescribeProduct", map[string]any{"Id": prodID})
	pas := asList(t, descResp["ProvisioningArtifacts"], "ProvisioningArtifacts")
	if len(pas) != 1 {
		t.Fatalf("DescribeProduct: want 1 PA, got %d", len(pas))
	}

	// Search
	searchResp := callOK(t, h, "SearchProducts", nil)
	if len(asList(t, searchResp["ProductViewSummaries"], "ProductViewSummaries")) != 1 {
		t.Fatalf("SearchProducts: bad: %v", searchResp)
	}

	// SearchProductsAsAdmin
	searchAdmin := callOK(t, h, "SearchProductsAsAdmin", nil)
	if len(asList(t, searchAdmin["ProductViewDetails"], "ProductViewDetails")) != 1 {
		t.Fatalf("SearchProductsAsAdmin: bad: %v", searchAdmin)
	}

	// Describe as admin
	descAdmin := callOK(t, h, "DescribeProductAsAdmin", map[string]any{"Id": prodID})
	if len(asList(t, descAdmin["ProvisioningArtifactSummaries"], "ProvisioningArtifactSummaries")) != 1 {
		t.Fatalf("DescribeProductAsAdmin: bad: %v", descAdmin)
	}

	// DescribeProductView
	pview := callOK(t, h, "DescribeProductView", map[string]any{"Id": prodID})
	if asMap(t, pview["ProductViewSummary"], "ProductViewSummary")["ProductId"] != prodID {
		t.Fatalf("DescribeProductView: bad: %v", pview)
	}

	// Update product
	upResp := callOK(t, h, "UpdateProduct", map[string]any{
		"Id":    prodID,
		"Owner": "newowner",
	})
	updated := asMap(t, asMap(t, upResp["ProductViewDetail"], "ProductViewDetail")["ProductViewSummary"], "ProductViewSummary")
	if updated["Owner"] != "newowner" {
		t.Fatalf("UpdateProduct: owner not updated: %v", updated)
	}

	// Create another PA
	pa2Resp := callOK(t, h, "CreateProvisioningArtifact", map[string]any{
		"ProductId": prodID,
		"Parameters": map[string]any{
			"Name": "v2",
			"Type": "CLOUD_FORMATION_TEMPLATE",
			"Info": map[string]any{"LoadTemplateFromURL": "https://example.com/v2.json"},
		},
	})
	pa2ID := asMap(t, pa2Resp["ProvisioningArtifactDetail"], "ProvisioningArtifactDetail")["Id"].(string)

	// List PAs
	listPA := callOK(t, h, "ListProvisioningArtifacts", map[string]any{"ProductId": prodID})
	if len(asList(t, listPA["ProvisioningArtifactDetails"], "ProvisioningArtifactDetails")) != 2 {
		t.Fatalf("ListProvisioningArtifacts: want 2, got %v", listPA)
	}

	// Describe PA
	descPA := callOK(t, h, "DescribeProvisioningArtifact", map[string]any{
		"ProductId":              prodID,
		"ProvisioningArtifactId": pa2ID,
	})
	if asMap(t, descPA["ProvisioningArtifactDetail"], "ProvisioningArtifactDetail")["Name"] != "v2" {
		t.Fatalf("DescribeProvisioningArtifact: bad: %v", descPA)
	}

	// Update PA
	updatedPA := callOK(t, h, "UpdateProvisioningArtifact", map[string]any{
		"ProductId":              prodID,
		"ProvisioningArtifactId": pa2ID,
		"Description":            "updated",
		"Active":                 false,
	})
	if asMap(t, updatedPA["ProvisioningArtifactDetail"], "ProvisioningArtifactDetail")["Active"] != false {
		t.Fatalf("UpdateProvisioningArtifact: bad: %v", updatedPA)
	}

	// Delete PA
	callOK(t, h, "DeleteProvisioningArtifact", map[string]any{
		"ProductId":              prodID,
		"ProvisioningArtifactId": pa2ID,
	})

	// Verify gone
	code, _ := call(t, h, "DescribeProvisioningArtifact", map[string]any{
		"ProductId":              prodID,
		"ProvisioningArtifactId": pa2ID,
	})
	if code != http.StatusBadRequest {
		t.Fatalf("DescribeProvisioningArtifact after delete: want 400, got %d", code)
	}

	// Delete product
	callOK(t, h, "DeleteProduct", map[string]any{"Id": prodID})

	// Verify gone
	code, _ = call(t, h, "DescribeProduct", map[string]any{"Id": prodID})
	if code != http.StatusBadRequest {
		t.Fatalf("DescribeProduct after delete: want 400, got %d", code)
	}
	_ = paID
}

// ── Constraint lifecycle ───────────────────────────────────────────────────

func TestConstraintLifecycle(t *testing.T) {
	h := newGateway(t)

	port := callOK(t, h, "CreatePortfolio", map[string]any{
		"DisplayName": "P", "ProviderName": "cm",
	})
	portID := asMap(t, port["PortfolioDetail"], "PortfolioDetail")["Id"].(string)

	prod := callOK(t, h, "CreateProduct", map[string]any{
		"Name":  "Prod",
		"Owner": "cm",
		"ProvisioningArtifactParameters": map[string]any{
			"Name": "v1",
			"Info": map[string]any{"k": "v"},
		},
	})
	prodID := asMap(t, asMap(t, prod["ProductViewDetail"], "ProductViewDetail")["ProductViewSummary"], "ProductViewSummary")["ProductId"].(string)

	// Create constraint
	consResp := callOK(t, h, "CreateConstraint", map[string]any{
		"PortfolioId": portID,
		"ProductId":   prodID,
		"Type":        "LAUNCH",
		"Parameters":  `{"RoleArn":"arn:aws:iam::123:role/r"}`,
		"Description": "Launch constraint",
	})
	consID := asMap(t, consResp["ConstraintDetail"], "ConstraintDetail")["ConstraintId"].(string)

	// Describe constraint
	descResp := callOK(t, h, "DescribeConstraint", map[string]any{"Id": consID})
	if asMap(t, descResp["ConstraintDetail"], "ConstraintDetail")["Type"] != "LAUNCH" {
		t.Fatalf("DescribeConstraint: bad: %v", descResp)
	}

	// List constraints for portfolio
	listResp := callOK(t, h, "ListConstraintsForPortfolio", map[string]any{"PortfolioId": portID})
	if len(asList(t, listResp["ConstraintDetails"], "ConstraintDetails")) != 1 {
		t.Fatalf("ListConstraintsForPortfolio: want 1, got %v", listResp)
	}

	// Update constraint
	callOK(t, h, "UpdateConstraint", map[string]any{
		"Id":          consID,
		"Description": "updated",
	})

	// Delete constraint
	callOK(t, h, "DeleteConstraint", map[string]any{"Id": consID})

	code, _ := call(t, h, "DescribeConstraint", map[string]any{"Id": consID})
	if code != http.StatusBadRequest {
		t.Fatalf("DescribeConstraint after delete: want 400, got %d", code)
	}
}

// ── Principal association ─────────────────────────────────────────────────

func TestPrincipalAssociation(t *testing.T) {
	h := newGateway(t)
	port := callOK(t, h, "CreatePortfolio", map[string]any{
		"DisplayName": "P", "ProviderName": "cm",
	})
	portID := asMap(t, port["PortfolioDetail"], "PortfolioDetail")["Id"].(string)

	callOK(t, h, "AssociatePrincipalWithPortfolio", map[string]any{
		"PortfolioId":   portID,
		"PrincipalARN":  "arn:aws:iam::123:role/admin",
		"PrincipalType": "IAM",
	})

	listResp := callOK(t, h, "ListPrincipalsForPortfolio", map[string]any{"PortfolioId": portID})
	if len(asList(t, listResp["Principals"], "Principals")) != 1 {
		t.Fatalf("ListPrincipalsForPortfolio: want 1, got %v", listResp)
	}

	callOK(t, h, "DisassociatePrincipalFromPortfolio", map[string]any{
		"PortfolioId":  portID,
		"PrincipalARN": "arn:aws:iam::123:role/admin",
	})

	listResp = callOK(t, h, "ListPrincipalsForPortfolio", map[string]any{"PortfolioId": portID})
	if len(asList(t, listResp["Principals"], "Principals")) != 0 {
		t.Fatalf("ListPrincipalsForPortfolio: want 0 after disassoc, got %v", listResp)
	}
}

// ── Tag option lifecycle ──────────────────────────────────────────────────

func TestTagOptionLifecycle(t *testing.T) {
	h := newGateway(t)

	// Create
	createResp := callOK(t, h, "CreateTagOption", map[string]any{
		"Key":   "env",
		"Value": "prod",
	})
	tID := asMap(t, createResp["TagOptionDetail"], "TagOptionDetail")["Id"].(string)

	// Duplicate -> error
	code, _ := call(t, h, "CreateTagOption", map[string]any{"Key": "env", "Value": "prod"})
	if code != http.StatusBadRequest {
		t.Fatalf("Duplicate CreateTagOption: want 400, got %d", code)
	}

	// Describe
	desc := callOK(t, h, "DescribeTagOption", map[string]any{"Id": tID})
	if asMap(t, desc["TagOptionDetail"], "TagOptionDetail")["Key"] != "env" {
		t.Fatalf("DescribeTagOption: bad: %v", desc)
	}

	// List (with filter)
	listResp := callOK(t, h, "ListTagOptions", map[string]any{
		"Filters": map[string]any{"Key": "env"},
	})
	if len(asList(t, listResp["TagOptionDetails"], "TagOptionDetails")) != 1 {
		t.Fatalf("ListTagOptions: want 1, got %v", listResp)
	}

	// Update
	upResp := callOK(t, h, "UpdateTagOption", map[string]any{
		"Id":     tID,
		"Active": false,
	})
	if asMap(t, upResp["TagOptionDetail"], "TagOptionDetail")["Active"] != false {
		t.Fatalf("UpdateTagOption: bad: %v", upResp)
	}

	// Associate with a portfolio resource
	port := callOK(t, h, "CreatePortfolio", map[string]any{
		"DisplayName": "P", "ProviderName": "cm",
	})
	portID := asMap(t, port["PortfolioDetail"], "PortfolioDetail")["Id"].(string)

	callOK(t, h, "AssociateTagOptionWithResource", map[string]any{
		"ResourceId":  portID,
		"TagOptionId": tID,
	})

	resResp := callOK(t, h, "ListResourcesForTagOption", map[string]any{"TagOptionId": tID})
	if len(asList(t, resResp["ResourceDetails"], "ResourceDetails")) != 1 {
		t.Fatalf("ListResourcesForTagOption: want 1, got %v", resResp)
	}

	callOK(t, h, "DisassociateTagOptionFromResource", map[string]any{
		"ResourceId":  portID,
		"TagOptionId": tID,
	})

	// Delete
	callOK(t, h, "DeleteTagOption", map[string]any{"Id": tID})
	code, _ = call(t, h, "DescribeTagOption", map[string]any{"Id": tID})
	if code != http.StatusBadRequest {
		t.Fatalf("DescribeTagOption after delete: want 400, got %d", code)
	}
}

// ── Portfolio↔product association ──────────────────────────────────────────

func TestPortfolioProductAssociation(t *testing.T) {
	h := newGateway(t)

	port := callOK(t, h, "CreatePortfolio", map[string]any{"DisplayName": "P", "ProviderName": "cm"})
	portID := asMap(t, port["PortfolioDetail"], "PortfolioDetail")["Id"].(string)
	prod := callOK(t, h, "CreateProduct", map[string]any{
		"Name":  "Prod",
		"Owner": "cm",
		"ProvisioningArtifactParameters": map[string]any{
			"Name": "v1",
			"Info": map[string]any{"k": "v"},
		},
	})
	prodID := asMap(t, asMap(t, prod["ProductViewDetail"], "ProductViewDetail")["ProductViewSummary"], "ProductViewSummary")["ProductId"].(string)

	callOK(t, h, "AssociateProductWithPortfolio", map[string]any{
		"PortfolioId": portID,
		"ProductId":   prodID,
	})

	// ListPortfoliosForProduct sees 1
	listResp := callOK(t, h, "ListPortfoliosForProduct", map[string]any{"ProductId": prodID})
	if len(asList(t, listResp["PortfolioDetails"], "PortfolioDetails")) != 1 {
		t.Fatalf("ListPortfoliosForProduct: want 1, got %v", listResp)
	}

	// LaunchPaths reflects the portfolio
	lpResp := callOK(t, h, "ListLaunchPaths", map[string]any{"ProductId": prodID})
	if len(asList(t, lpResp["LaunchPathSummaries"], "LaunchPathSummaries")) != 1 {
		t.Fatalf("ListLaunchPaths: want 1, got %v", lpResp)
	}

	callOK(t, h, "DisassociateProductFromPortfolio", map[string]any{
		"PortfolioId": portID,
		"ProductId":   prodID,
	})

	listResp = callOK(t, h, "ListPortfoliosForProduct", map[string]any{"ProductId": prodID})
	if len(asList(t, listResp["PortfolioDetails"], "PortfolioDetails")) != 0 {
		t.Fatalf("ListPortfoliosForProduct after disassoc: want 0, got %v", listResp)
	}
}

// ── Service action lifecycle + association ────────────────────────────────

func TestServiceActionLifecycle(t *testing.T) {
	h := newGateway(t)

	// Create
	saResp := callOK(t, h, "CreateServiceAction", map[string]any{
		"Name":           "RestartEC2",
		"DefinitionType": "SSM_AUTOMATION",
		"Definition": map[string]any{
			"Name":    "AWS-RestartEC2Instance",
			"Version": "1",
		},
		"Description": "Restart instance",
	})
	saID := asMap(t, asMap(t, saResp["ServiceActionDetail"], "ServiceActionDetail")["ServiceActionSummary"], "ServiceActionSummary")["Id"].(string)

	// Describe
	descResp := callOK(t, h, "DescribeServiceAction", map[string]any{"Id": saID})
	if asMap(t, asMap(t, descResp["ServiceActionDetail"], "ServiceActionDetail")["ServiceActionSummary"], "ServiceActionSummary")["Name"] != "RestartEC2" {
		t.Fatalf("DescribeServiceAction: bad: %v", descResp)
	}

	// List
	listResp := callOK(t, h, "ListServiceActions", nil)
	if len(asList(t, listResp["ServiceActionSummaries"], "ServiceActionSummaries")) != 1 {
		t.Fatalf("ListServiceActions: want 1, got %v", listResp)
	}

	// DescribeServiceActionExecutionParameters
	callOK(t, h, "DescribeServiceActionExecutionParameters", map[string]any{
		"ProvisionedProductId": "anything",
		"ServiceActionId":      saID,
	})

	// Update
	callOK(t, h, "UpdateServiceAction", map[string]any{
		"Id":          saID,
		"Description": "Updated",
	})

	// Need a product + PA for association
	prod := callOK(t, h, "CreateProduct", map[string]any{
		"Name":  "Prod",
		"Owner": "cm",
		"ProvisioningArtifactParameters": map[string]any{
			"Name": "v1",
			"Info": map[string]any{"k": "v"},
		},
	})
	prodID := asMap(t, asMap(t, prod["ProductViewDetail"], "ProductViewDetail")["ProductViewSummary"], "ProductViewSummary")["ProductId"].(string)
	paDetail := asMap(t, prod["ProvisioningArtifactDetail"], "ProvisioningArtifactDetail")
	paID := paDetail["Id"].(string)

	// Associate
	callOK(t, h, "AssociateServiceActionWithProvisioningArtifact", map[string]any{
		"ProductId":              prodID,
		"ProvisioningArtifactId": paID,
		"ServiceActionId":        saID,
	})

	// List service actions for PA
	listSAforPA := callOK(t, h, "ListServiceActionsForProvisioningArtifact", map[string]any{
		"ProductId":              prodID,
		"ProvisioningArtifactId": paID,
	})
	if len(asList(t, listSAforPA["ServiceActionSummaries"], "ServiceActionSummaries")) != 1 {
		t.Fatalf("ListServiceActionsForProvisioningArtifact: want 1, got %v", listSAforPA)
	}

	// List PAs for service action
	listPAforSA := callOK(t, h, "ListProvisioningArtifactsForServiceAction", map[string]any{
		"ServiceActionId": saID,
	})
	if len(asList(t, listPAforSA["ProvisioningArtifactViews"], "ProvisioningArtifactViews")) != 1 {
		t.Fatalf("ListProvisioningArtifactsForServiceAction: want 1, got %v", listPAforSA)
	}

	// Batch associate / disassociate
	callOK(t, h, "BatchAssociateServiceActionWithProvisioningArtifact", map[string]any{
		"ServiceActionAssociations": []map[string]any{
			{"ProductId": prodID, "ProvisioningArtifactId": paID, "ServiceActionId": saID},
		},
	})
	callOK(t, h, "BatchDisassociateServiceActionFromProvisioningArtifact", map[string]any{
		"ServiceActionAssociations": []map[string]any{
			{"ProductId": prodID, "ProvisioningArtifactId": paID, "ServiceActionId": saID},
		},
	})

	// Disassociate single
	callOK(t, h, "DisassociateServiceActionFromProvisioningArtifact", map[string]any{
		"ProductId":              prodID,
		"ProvisioningArtifactId": paID,
		"ServiceActionId":        saID,
	})

	// Delete
	callOK(t, h, "DeleteServiceAction", map[string]any{"Id": saID})
	code, _ := call(t, h, "DescribeServiceAction", map[string]any{"Id": saID})
	if code != http.StatusBadRequest {
		t.Fatalf("DescribeServiceAction after delete: want 400, got %d", code)
	}
}

// ── Provisioned product lifecycle ─────────────────────────────────────────

func TestProvisionedProductLifecycle(t *testing.T) {
	h := newGateway(t)

	// Setup product + PA
	prod := callOK(t, h, "CreateProduct", map[string]any{
		"Name":  "Prod",
		"Owner": "cm",
		"ProvisioningArtifactParameters": map[string]any{
			"Name": "v1",
			"Info": map[string]any{"k": "v"},
		},
	})
	prodID := asMap(t, asMap(t, prod["ProductViewDetail"], "ProductViewDetail")["ProductViewSummary"], "ProductViewSummary")["ProductId"].(string)
	paID := asMap(t, prod["ProvisioningArtifactDetail"], "ProvisioningArtifactDetail")["Id"].(string)

	// ProvisionProduct -> creates ProvisionedProduct + Record
	provResp := callOK(t, h, "ProvisionProduct", map[string]any{
		"ProvisionedProductName": "my-stack",
		"ProductId":              prodID,
		"ProvisioningArtifactId": paID,
		"ProvisionToken":         "tok-1",
	})
	rec := asMap(t, provResp["RecordDetail"], "RecordDetail")
	if rec["RecordType"] != "PROVISION_PRODUCT" || rec["Status"] != "SUCCEEDED" {
		t.Fatalf("ProvisionProduct: bad record: %v", rec)
	}
	ppID := rec["ProvisionedProductId"].(string)
	recID := rec["RecordId"].(string)

	// Duplicate -> 400
	code, _ := call(t, h, "ProvisionProduct", map[string]any{
		"ProvisionedProductName": "my-stack",
		"ProductId":              prodID,
		"ProvisioningArtifactId": paID,
	})
	if code != http.StatusBadRequest {
		t.Fatalf("Duplicate ProvisionProduct: want 400, got %d", code)
	}

	// Describe by id
	desc := callOK(t, h, "DescribeProvisionedProduct", map[string]any{"Id": ppID})
	pd := asMap(t, desc["ProvisionedProductDetail"], "ProvisionedProductDetail")
	if pd["Name"] != "my-stack" {
		t.Fatalf("DescribeProvisionedProduct: bad: %v", pd)
	}

	// Describe by name
	descByName := callOK(t, h, "DescribeProvisionedProduct", map[string]any{"Name": "my-stack"})
	if asMap(t, descByName["ProvisionedProductDetail"], "ProvisionedProductDetail")["Id"] != ppID {
		t.Fatalf("DescribeProvisionedProduct by name: bad")
	}

	// Scan / Search
	scanResp := callOK(t, h, "ScanProvisionedProducts", nil)
	if len(asList(t, scanResp["ProvisionedProducts"], "ProvisionedProducts")) != 1 {
		t.Fatalf("ScanProvisionedProducts: bad: %v", scanResp)
	}
	searchResp := callOK(t, h, "SearchProvisionedProducts", nil)
	if asLen, _ := searchResp["TotalResultsCount"].(float64); int(asLen) != 1 {
		t.Fatalf("SearchProvisionedProducts: bad: %v", searchResp)
	}

	// DescribeRecord
	recResp := callOK(t, h, "DescribeRecord", map[string]any{"Id": recID})
	if asMap(t, recResp["RecordDetail"], "RecordDetail")["RecordType"] != "PROVISION_PRODUCT" {
		t.Fatalf("DescribeRecord: bad: %v", recResp)
	}

	// ListRecordHistory
	histResp := callOK(t, h, "ListRecordHistory", map[string]any{
		"SearchFilter": map[string]any{"Key": "provisionedproductid", "Value": ppID},
	})
	if len(asList(t, histResp["RecordDetails"], "RecordDetails")) != 1 {
		t.Fatalf("ListRecordHistory: want 1, got %v", histResp)
	}

	// Update
	upResp := callOK(t, h, "UpdateProvisionedProduct", map[string]any{
		"ProvisionedProductId":   ppID,
		"ProductId":              prodID,
		"ProvisioningArtifactId": paID,
	})
	upRec := asMap(t, upResp["RecordDetail"], "RecordDetail")
	if upRec["RecordType"] != "UPDATE_PROVISIONED_PRODUCT" {
		t.Fatalf("UpdateProvisionedProduct: bad record: %v", upRec)
	}

	// UpdateProvisionedProductProperties
	propResp := callOK(t, h, "UpdateProvisionedProductProperties", map[string]any{
		"ProvisionedProductId":         ppID,
		"ProvisionedProductProperties": map[string]any{"OWNER": "arn:aws:iam::123:user/x"},
	})
	if propResp["ProvisionedProductId"] != ppID {
		t.Fatalf("UpdateProvisionedProductProperties: bad: %v", propResp)
	}

	// GetProvisionedProductOutputs
	outResp := callOK(t, h, "GetProvisionedProductOutputs", map[string]any{"ProvisionedProductId": ppID})
	if _, ok := outResp["Outputs"]; !ok {
		t.Fatalf("GetProvisionedProductOutputs: missing Outputs")
	}

	// ListStackInstancesForProvisionedProduct
	siResp := callOK(t, h, "ListStackInstancesForProvisionedProduct", map[string]any{
		"ProvisionedProductId": ppID,
	})
	if _, ok := siResp["StackInstances"]; !ok {
		t.Fatalf("ListStackInstancesForProvisionedProduct: missing StackInstances")
	}

	// Notify workflow results (no-ops)
	callOK(t, h, "NotifyProvisionProductEngineWorkflowResult", map[string]any{})
	callOK(t, h, "NotifyTerminateProvisionedProductEngineWorkflowResult", map[string]any{})
	callOK(t, h, "NotifyUpdateProvisionedProductEngineWorkflowResult", map[string]any{})

	// Terminate
	termResp := callOK(t, h, "TerminateProvisionedProduct", map[string]any{"ProvisionedProductId": ppID})
	if asMap(t, termResp["RecordDetail"], "RecordDetail")["RecordType"] != "TERMINATE_PROVISIONED_PRODUCT" {
		t.Fatalf("TerminateProvisionedProduct: bad: %v", termResp)
	}

	// Verify gone
	code, _ = call(t, h, "DescribeProvisionedProduct", map[string]any{"Id": ppID})
	if code != http.StatusBadRequest {
		t.Fatalf("DescribeProvisionedProduct after terminate: want 400, got %d", code)
	}
}

// ── ImportAsProvisionedProduct ────────────────────────────────────────────

func TestImportAsProvisionedProduct(t *testing.T) {
	h := newGateway(t)
	prod := callOK(t, h, "CreateProduct", map[string]any{
		"Name":  "Prod",
		"Owner": "cm",
		"ProvisioningArtifactParameters": map[string]any{
			"Name": "v1",
			"Info": map[string]any{"k": "v"},
		},
	})
	prodID := asMap(t, asMap(t, prod["ProductViewDetail"], "ProductViewDetail")["ProductViewSummary"], "ProductViewSummary")["ProductId"].(string)

	resp := callOK(t, h, "ImportAsProvisionedProduct", map[string]any{
		"ProvisionedProductName": "imported",
		"ProductId":              prodID,
		"IdempotencyToken":       "tok",
	})
	rec := asMap(t, resp["RecordDetail"], "RecordDetail")
	if rec["ProvisionedProductName"] != "imported" {
		t.Fatalf("ImportAsProvisionedProduct: bad: %v", rec)
	}
}

// ── Provisioned product plan lifecycle ────────────────────────────────────

func TestProvisionedProductPlanLifecycle(t *testing.T) {
	h := newGateway(t)
	prod := callOK(t, h, "CreateProduct", map[string]any{
		"Name":  "Prod",
		"Owner": "cm",
		"ProvisioningArtifactParameters": map[string]any{
			"Name": "v1",
			"Info": map[string]any{"k": "v"},
		},
	})
	prodID := asMap(t, asMap(t, prod["ProductViewDetail"], "ProductViewDetail")["ProductViewSummary"], "ProductViewSummary")["ProductId"].(string)
	paID := asMap(t, prod["ProvisioningArtifactDetail"], "ProvisioningArtifactDetail")["Id"].(string)

	createResp := callOK(t, h, "CreateProvisionedProductPlan", map[string]any{
		"PlanName":               "plan-1",
		"PlanType":               "CLOUDFORMATION",
		"ProductId":              prodID,
		"ProvisioningArtifactId": paID,
		"ProvisionedProductName": "my-stack",
	})
	planID := createResp["PlanId"].(string)

	// Describe
	descResp := callOK(t, h, "DescribeProvisionedProductPlan", map[string]any{"PlanId": planID})
	if asMap(t, descResp["ProvisionedProductPlanDetails"], "ProvisionedProductPlanDetails")["PlanId"] != planID {
		t.Fatalf("DescribeProvisionedProductPlan: bad: %v", descResp)
	}

	// List
	listResp := callOK(t, h, "ListProvisionedProductPlans", map[string]any{})
	if len(asList(t, listResp["ProvisionedProductPlans"], "ProvisionedProductPlans")) != 1 {
		t.Fatalf("ListProvisionedProductPlans: want 1, got %v", listResp)
	}

	// Execute
	execResp := callOK(t, h, "ExecuteProvisionedProductPlan", map[string]any{"PlanId": planID})
	if asMap(t, execResp["RecordDetail"], "RecordDetail")["Status"] != "SUCCEEDED" {
		t.Fatalf("ExecuteProvisionedProductPlan: bad: %v", execResp)
	}

	// Delete
	callOK(t, h, "DeleteProvisionedProductPlan", map[string]any{"PlanId": planID})
	code, _ := call(t, h, "DescribeProvisionedProductPlan", map[string]any{"PlanId": planID})
	if code != http.StatusBadRequest {
		t.Fatalf("DescribeProvisionedProductPlan after delete: want 400, got %d", code)
	}
}

// ── Budget association ────────────────────────────────────────────────────

func TestBudgetAssociation(t *testing.T) {
	h := newGateway(t)
	port := callOK(t, h, "CreatePortfolio", map[string]any{"DisplayName": "P", "ProviderName": "cm"})
	portID := asMap(t, port["PortfolioDetail"], "PortfolioDetail")["Id"].(string)

	callOK(t, h, "AssociateBudgetWithResource", map[string]any{
		"BudgetName": "monthly",
		"ResourceId": portID,
	})

	listResp := callOK(t, h, "ListBudgetsForResource", map[string]any{"ResourceId": portID})
	if len(asList(t, listResp["Budgets"], "Budgets")) != 1 {
		t.Fatalf("ListBudgetsForResource: want 1, got %v", listResp)
	}

	callOK(t, h, "DisassociateBudgetFromResource", map[string]any{
		"BudgetName": "monthly",
		"ResourceId": portID,
	})

	listResp = callOK(t, h, "ListBudgetsForResource", map[string]any{"ResourceId": portID})
	if len(asList(t, listResp["Budgets"], "Budgets")) != 0 {
		t.Fatalf("ListBudgetsForResource after disassoc: want 0, got %v", listResp)
	}
}

// ── AWS Organizations access ──────────────────────────────────────────────

func TestAWSOrganizationsAccess(t *testing.T) {
	h := newGateway(t)

	statusResp := callOK(t, h, "GetAWSOrganizationsAccessStatus", nil)
	if statusResp["AccessStatus"] != "DISABLED" {
		t.Fatalf("expected DISABLED initially, got %v", statusResp)
	}

	callOK(t, h, "EnableAWSOrganizationsAccess", nil)
	statusResp = callOK(t, h, "GetAWSOrganizationsAccessStatus", nil)
	if statusResp["AccessStatus"] != "ENABLED" {
		t.Fatalf("expected ENABLED, got %v", statusResp)
	}

	callOK(t, h, "DisableAWSOrganizationsAccess", nil)
	statusResp = callOK(t, h, "GetAWSOrganizationsAccessStatus", nil)
	if statusResp["AccessStatus"] != "DISABLED" {
		t.Fatalf("expected DISABLED after disable, got %v", statusResp)
	}
}

// ── Copy product ──────────────────────────────────────────────────────────

func TestCopyProduct(t *testing.T) {
	h := newGateway(t)

	resp := callOK(t, h, "CopyProduct", map[string]any{
		"SourceProductArn": "arn:aws:catalog::123:product/prod-source",
	})
	token := resp["CopyProductToken"].(string)
	if token == "" {
		t.Fatalf("CopyProduct: empty token")
	}

	statusResp := callOK(t, h, "DescribeCopyProductStatus", map[string]any{
		"CopyProductToken": token,
	})
	if statusResp["CopyProductStatus"] != "SUCCEEDED" {
		t.Fatalf("DescribeCopyProductStatus: bad: %v", statusResp)
	}

	// Missing source -> 400
	code, _ := call(t, h, "CopyProduct", map[string]any{})
	if code != http.StatusBadRequest {
		t.Fatalf("CopyProduct without source: want 400, got %d", code)
	}
}

// ── DescribeProvisioningParameters ────────────────────────────────────────

func TestDescribeProvisioningParameters(t *testing.T) {
	h := newGateway(t)
	prod := callOK(t, h, "CreateProduct", map[string]any{
		"Name":  "Prod",
		"Owner": "cm",
		"ProvisioningArtifactParameters": map[string]any{
			"Name": "v1",
			"Info": map[string]any{"k": "v"},
		},
	})
	prodID := asMap(t, asMap(t, prod["ProductViewDetail"], "ProductViewDetail")["ProductViewSummary"], "ProductViewSummary")["ProductId"].(string)
	paID := asMap(t, prod["ProvisioningArtifactDetail"], "ProvisioningArtifactDetail")["Id"].(string)

	resp := callOK(t, h, "DescribeProvisioningParameters", map[string]any{
		"ProductId":              prodID,
		"ProvisioningArtifactId": paID,
	})
	if _, ok := resp["ProvisioningArtifactParameters"]; !ok {
		t.Fatalf("DescribeProvisioningParameters: missing field: %v", resp)
	}
}

// ── ExecuteProvisionedProductServiceAction ────────────────────────────────

func TestExecuteProvisionedProductServiceAction(t *testing.T) {
	h := newGateway(t)

	// Create product, PA, provision it
	prod := callOK(t, h, "CreateProduct", map[string]any{
		"Name":  "Prod",
		"Owner": "cm",
		"ProvisioningArtifactParameters": map[string]any{
			"Name": "v1",
			"Info": map[string]any{"k": "v"},
		},
	})
	prodID := asMap(t, asMap(t, prod["ProductViewDetail"], "ProductViewDetail")["ProductViewSummary"], "ProductViewSummary")["ProductId"].(string)
	paID := asMap(t, prod["ProvisioningArtifactDetail"], "ProvisioningArtifactDetail")["Id"].(string)

	provResp := callOK(t, h, "ProvisionProduct", map[string]any{
		"ProvisionedProductName": "stack-x",
		"ProductId":              prodID,
		"ProvisioningArtifactId": paID,
	})
	ppID := asMap(t, provResp["RecordDetail"], "RecordDetail")["ProvisionedProductId"].(string)

	saResp := callOK(t, h, "CreateServiceAction", map[string]any{
		"Name":           "Stop",
		"DefinitionType": "SSM_AUTOMATION",
		"Definition":     map[string]any{"Name": "AWS-StopEC2Instance"},
	})
	saID := asMap(t, asMap(t, saResp["ServiceActionDetail"], "ServiceActionDetail")["ServiceActionSummary"], "ServiceActionSummary")["Id"].(string)

	execResp := callOK(t, h, "ExecuteProvisionedProductServiceAction", map[string]any{
		"ProvisionedProductId": ppID,
		"ServiceActionId":      saID,
	})
	if asMap(t, execResp["RecordDetail"], "RecordDetail")["Status"] != "SUCCEEDED" {
		t.Fatalf("ExecuteProvisionedProductServiceAction: bad: %v", execResp)
	}
}

// ── Reset ─────────────────────────────────────────────────────────────────

func TestStoreReset(t *testing.T) {
	store := svc.NewStore("123456789012", "us-east-1")
	if _, err := store.CreatePortfolio("P1", "desc", "cm", nil); err != nil {
		t.Fatalf("CreatePortfolio: %v", err)
	}
	if got := len(store.ListPortfolios()); got != 1 {
		t.Fatalf("expected 1 portfolio, got %d", got)
	}
	store.Reset()
	if got := len(store.ListPortfolios()); got != 0 {
		t.Fatalf("expected 0 portfolios after Reset, got %d", got)
	}
}

// ── ListOrganizationPortfolioAccess ───────────────────────────────────────

func TestListOrganizationPortfolioAccess(t *testing.T) {
	h := newGateway(t)
	port := callOK(t, h, "CreatePortfolio", map[string]any{
		"DisplayName": "P", "ProviderName": "cm",
	})
	portID := asMap(t, port["PortfolioDetail"], "PortfolioDetail")["Id"].(string)

	callOK(t, h, "CreatePortfolioShare", map[string]any{
		"PortfolioId": portID,
		"OrganizationNode": map[string]any{
			"Type":  "ORGANIZATION",
			"Value": "o-123",
		},
	})

	resp := callOK(t, h, "ListOrganizationPortfolioAccess", map[string]any{
		"PortfolioId":          portID,
		"OrganizationNodeType": "ORGANIZATION",
	})
	if len(asList(t, resp["OrganizationNodes"], "OrganizationNodes")) != 1 {
		t.Fatalf("ListOrganizationPortfolioAccess: want 1, got %v", resp)
	}
}
