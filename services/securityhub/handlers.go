package securityhub

import (
	"net/http"
	"strings"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

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
		return service.NewAWSError("InvalidParameterException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func decodeBody(ctx *service.RequestContext) (map[string]any, *service.AWSError) {
	req := map[string]any{}
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return nil, awsErr
	}
	return req, nil
}

func getStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
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

func getInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

func getFloat(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		}
	}
	return 0
}

func getFloatPtr(m map[string]any, key string) *float64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return &n
		case int:
			f := float64(n)
			return &f
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

func parseTagMap(m map[string]any, key string) map[string]string {
	out := make(map[string]string)
	if v, ok := m[key]; ok {
		if tm, ok := v.(map[string]any); ok {
			for k, val := range tm {
				if s, ok := val.(string); ok {
					out[k] = s
				}
			}
		}
	}
	return out
}

func rfc3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// ── Hub lifecycle ────────────────────────────────────────────────────────────

func handleEnableSecurityHub(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	autoEnable := true
	if v := getBoolPtr(req, "EnableDefaultStandards"); v != nil {
		autoEnable = *v
	}
	hub, err := store.EnableHub(autoEnable, getStr(req, "ControlFindingGenerator"), parseTagMap(req, "Tags"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"HubArn":       hub.HubArn,
		"SubscribedAt": rfc3339(hub.SubscribedAt),
	})
}

func handleDisableSecurityHub(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	if err := store.DisableHub(); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDescribeHub(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	hub, err := store.GetHub()
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"HubArn":                  hub.HubArn,
		"SubscribedAt":            rfc3339(hub.SubscribedAt),
		"AutoEnableControls":      hub.AutoEnableControls,
		"ControlFindingGenerator": hub.ControlFindingGenerator,
	})
}

func handleUpdateSecurityHubConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.UpdateHubConfig(getBoolPtr(req, "AutoEnableControls"), getStr(req, "ControlFindingGenerator")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleEnableSecurityHubV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	hub, err := store.EnableHubV2(parseTagMap(req, "Tags"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"HubArn":       hub.HubArn,
		"SubscribedAt": rfc3339(hub.SubscribedAt),
	})
}

func handleDisableSecurityHubV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	if err := store.DisableHubV2(); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDescribeSecurityHubV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	hub, err := store.GetHubV2()
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"HubArn":       hub.HubArn,
		"SubscribedAt": rfc3339(hub.SubscribedAt),
	})
}

// ── Standards ────────────────────────────────────────────────────────────────

// staticStandards is a deterministic catalogue of well-known Security Hub standards.
var staticStandards = []map[string]any{
	{
		"StandardsArn":     "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0",
		"Name":             "CIS AWS Foundations Benchmark v1.2.0",
		"Description":      "The Center for Internet Security (CIS) AWS Foundations Benchmark v1.2.0.",
		"EnabledByDefault": true,
	},
	{
		"StandardsArn":     "arn:aws:securityhub:::ruleset/aws-foundational-security-best-practices/v/1.0.0",
		"Name":             "AWS Foundational Security Best Practices v1.0.0",
		"Description":      "The AWS Foundational Security Best Practices standard.",
		"EnabledByDefault": true,
	},
	{
		"StandardsArn":     "arn:aws:securityhub:::ruleset/pci-dss/v/3.2.1",
		"Name":             "PCI DSS v3.2.1",
		"Description":      "The Payment Card Industry Data Security Standard (PCI DSS) v3.2.1.",
		"EnabledByDefault": false,
	},
}

func handleDescribeStandards(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{"Standards": staticStandards})
}

func handleBatchEnableStandards(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	subs := getMapList(req, "StandardsSubscriptionRequests")
	if len(subs) == 0 {
		return jsonErr(service.NewAWSError("InvalidInputException",
			"StandardsSubscriptionRequests is required", http.StatusBadRequest))
	}
	out := make([]map[string]any, 0, len(subs))
	for _, sub := range subs {
		input := make(map[string]string)
		if im := getMap(sub, "StandardsInput"); im != nil {
			for k, v := range im {
				if s, ok := v.(string); ok {
					input[k] = s
				}
			}
		}
		ss, err := store.EnableStandards(getStr(sub, "StandardsArn"), input)
		if err != nil {
			out = append(out, map[string]any{
				"StandardsArn":          getStr(sub, "StandardsArn"),
				"StandardsStatus":       "FAILED",
				"StandardsStatusReason": map[string]any{"StatusReasonCode": err.Code},
			})
			continue
		}
		out = append(out, standardsSubscriptionToMap(ss))
	}
	return jsonOK(map[string]any{"StandardsSubscriptions": out})
}

func handleBatchDisableStandards(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	arns := getStrList(req, "StandardsSubscriptionArns")
	out := make([]map[string]any, 0, len(arns))
	for _, arn := range arns {
		entry := map[string]any{
			"StandardsSubscriptionArn": arn,
			"StandardsStatus":          "DELETING",
		}
		if err := store.DisableStandards(arn); err != nil {
			entry["StandardsStatus"] = "FAILED"
			entry["StandardsStatusReason"] = map[string]any{"StatusReasonCode": err.Code}
		}
		out = append(out, entry)
	}
	return jsonOK(map[string]any{"StandardsSubscriptions": out})
}

func handleGetEnabledStandards(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	subs := store.ListStandardsSubscriptions()
	out := make([]map[string]any, 0, len(subs))
	for _, s := range subs {
		out = append(out, standardsSubscriptionToMap(s))
	}
	return jsonOK(map[string]any{"StandardsSubscriptions": out})
}

func standardsSubscriptionToMap(s *StoredStandardsSubscription) map[string]any {
	input := make(map[string]any, len(s.StandardsInput))
	for k, v := range s.StandardsInput {
		input[k] = v
	}
	return map[string]any{
		"StandardsSubscriptionArn": s.StandardsSubscriptionArn,
		"StandardsArn":             s.StandardsArn,
		"StandardsInput":           input,
		"StandardsStatus":          s.StandardsStatus,
	}
}

func handleDescribeStandardsControls(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "StandardsSubscriptionArn")
	controls := []map[string]any{
		{
			"StandardsControlArn":    arn + "/control/CIS.1.1",
			"ControlStatus":          "ENABLED",
			"ControlStatusUpdatedAt": rfc3339(time.Now().UTC()),
			"ControlId":              "CIS.1.1",
			"Title":                  "Avoid the use of the root account",
			"Description":            "The root account has unrestricted access to all resources in the AWS account.",
			"RemediationUrl":         "https://docs.aws.amazon.com/console/securityhub/CIS.1.1/remediation",
			"SeverityRating":         "LOW",
			"RelatedRequirements":    []string{"CIS AWS Foundations 1.1"},
		},
	}
	return jsonOK(map[string]any{"Controls": controls})
}

func handleUpdateStandardsControl(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if getStr(req, "StandardsControlArn") == "" {
		return jsonErr(service.NewAWSError("InvalidInputException",
			"StandardsControlArn is required", http.StatusBadRequest))
	}
	return jsonOK(map[string]any{})
}

// ── Security controls ────────────────────────────────────────────────────────

var staticSecurityControlDefinitions = []map[string]any{
	{
		"SecurityControlId":     "IAM.1",
		"Title":                 "IAM root user access key should not exist",
		"Description":           "This control checks whether the root user access key is available.",
		"RemediationUrl":        "https://docs.aws.amazon.com/console/securityhub/IAM.1/remediation",
		"SeverityRating":        "CRITICAL",
		"CurrentRegionAvailability": "AVAILABLE",
	},
	{
		"SecurityControlId":     "S3.1",
		"Title":                 "S3 Block Public Access setting should be enabled",
		"Description":           "This control checks whether the S3 Block Public Access setting is enabled at the account level.",
		"RemediationUrl":        "https://docs.aws.amazon.com/console/securityhub/S3.1/remediation",
		"SeverityRating":        "MEDIUM",
		"CurrentRegionAvailability": "AVAILABLE",
	},
}

func handleListSecurityControlDefinitions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{"SecurityControlDefinitions": staticSecurityControlDefinitions})
}

func handleGetSecurityControlDefinition(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "SecurityControlId")
	for _, def := range staticSecurityControlDefinitions {
		if def["SecurityControlId"].(string) == id {
			return jsonOK(map[string]any{"SecurityControlDefinition": def})
		}
	}
	return jsonErr(service.NewAWSError("ResourceNotFoundException",
		"Security control definition not found: "+id, http.StatusBadRequest))
}

func handleBatchGetSecurityControls(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	ids := getStrList(req, "SecurityControlIds")
	found := make([]map[string]any, 0)
	unprocessed := make([]map[string]any, 0)
	for _, id := range ids {
		var def map[string]any
		for _, d := range staticSecurityControlDefinitions {
			if d["SecurityControlId"].(string) == id {
				def = d
				break
			}
		}
		if def == nil {
			unprocessed = append(unprocessed, map[string]any{
				"SecurityControlId": id,
				"ErrorCode":         "SecurityControlNotFound",
				"ErrorReason":       "Security control not found",
			})
			continue
		}
		found = append(found, map[string]any{
			"SecurityControlId":     id,
			"SecurityControlArn":    "arn:aws:securityhub:" + store.Region() + ":" + store.AccountID() + ":security-control/" + id,
			"Title":                 def["Title"],
			"Description":           def["Description"],
			"RemediationUrl":        def["RemediationUrl"],
			"SeverityRating":        def["SeverityRating"],
			"SecurityControlStatus": "ENABLED",
		})
	}
	return jsonOK(map[string]any{
		"SecurityControls":            found,
		"UnprocessedIds":              unprocessed,
	})
}

func handleUpdateSecurityControl(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if getStr(req, "SecurityControlId") == "" {
		return jsonErr(service.NewAWSError("InvalidInputException",
			"SecurityControlId is required", http.StatusBadRequest))
	}
	return jsonOK(map[string]any{})
}

func handleBatchGetStandardsControlAssociations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	requests := getMapList(req, "StandardsControlAssociationIds")
	out := make([]map[string]any, 0, len(requests))
	for _, r := range requests {
		out = append(out, map[string]any{
			"StandardsArn":              getStr(r, "StandardsArn"),
			"SecurityControlId":         getStr(r, "SecurityControlId"),
			"SecurityControlArn":        "arn:aws:securityhub:" + store.Region() + ":" + store.AccountID() + ":security-control/" + getStr(r, "SecurityControlId"),
			"AssociationStatus":         "ENABLED",
			"RelatedRequirements":       []string{},
			"UpdatedReason":             "",
			"StandardsControlTitle":     "Mock control",
			"StandardsControlDescription": "Mock control description",
		})
	}
	return jsonOK(map[string]any{"StandardsControlAssociationDetails": out, "UnprocessedAssociations": []any{}})
}

func handleBatchUpdateStandardsControlAssociations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	updates := getMapList(req, "StandardsControlAssociationUpdates")
	unprocessed := make([]map[string]any, 0)
	for _, u := range updates {
		if getStr(u, "StandardsArn") == "" || getStr(u, "SecurityControlId") == "" {
			unprocessed = append(unprocessed, map[string]any{
				"StandardsControlAssociationUpdate": u,
				"ErrorCode":                         "InvalidInput",
				"ErrorReason":                       "StandardsArn and SecurityControlId are required",
			})
		}
	}
	return jsonOK(map[string]any{"UnprocessedAssociationUpdates": unprocessed})
}

func handleListStandardsControlAssociations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{"StandardsControlAssociationSummaries": []any{}})
}

// ── Products ─────────────────────────────────────────────────────────────────

var staticProducts = []map[string]any{
	{
		"ProductArn":       "arn:aws:securityhub:us-east-1::product/aws/securityhub",
		"ProductName":      "Security Hub",
		"CompanyName":      "AWS",
		"Description":      "AWS Security Hub provides you with a comprehensive view of your security state in AWS.",
		"Categories":       []string{"AWS Service"},
		"IntegrationTypes": []string{"SEND_FINDINGS_TO_SECURITY_HUB", "RECEIVE_FINDINGS_FROM_SECURITY_HUB"},
	},
	{
		"ProductArn":       "arn:aws:securityhub:us-east-1::product/aws/guardduty",
		"ProductName":      "GuardDuty",
		"CompanyName":      "AWS",
		"Description":      "Amazon GuardDuty is a continuous security monitoring service.",
		"Categories":       []string{"AWS Service"},
		"IntegrationTypes": []string{"SEND_FINDINGS_TO_SECURITY_HUB"},
	},
}

func handleDescribeProducts(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{"Products": staticProducts})
}

func handleDescribeProductsV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{"ProductsV2": staticProducts})
}

func handleEnableImportFindingsForProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	sub, err := store.EnableImportFindingsForProduct(getStr(req, "ProductArn"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"ProductSubscriptionArn": sub.ProductSubscriptionArn})
}

func handleDisableImportFindingsForProduct(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.DisableImportFindingsForProduct(getStr(req, "ProductSubscriptionArn")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListEnabledProductsForImport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	subs := store.ListEnabledProducts()
	arns := make([]string, 0, len(subs))
	for _, s := range subs {
		arns = append(arns, s.ProductSubscriptionArn)
	}
	return jsonOK(map[string]any{"ProductSubscriptions": arns})
}

// ── Findings ─────────────────────────────────────────────────────────────────

func handleBatchImportFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	findings := getMapList(req, "Findings")
	success, failed := store.ImportFindings(findings)
	return jsonOK(map[string]any{
		"FailedCount":    len(failed),
		"SuccessCount":   success,
		"FailedFindings": failed,
	})
}

func handleBatchUpdateFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	identifiers := getMapList(req, "FindingIdentifiers")
	updates := buildBatchUpdateMap(req)
	processed, unprocessed := store.BatchUpdateFindings(identifiers, updates)
	return jsonOK(map[string]any{
		"ProcessedFindings":   processed,
		"UnprocessedFindings": unprocessed,
	})
}

func handleBatchUpdateFindingsV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	identifiers := getMapList(req, "FindingIdentifiers")
	updates := buildBatchUpdateMap(req)
	processed, unprocessed := store.BatchUpdateFindings(identifiers, updates)
	return jsonOK(map[string]any{
		"ProcessedFindings":   processed,
		"UnprocessedFindings": unprocessed,
	})
}

// buildBatchUpdateMap collects update fields from a BatchUpdateFindings request body.
func buildBatchUpdateMap(req map[string]any) map[string]any {
	updates := make(map[string]any)
	for _, f := range []string{
		"Note", "Severity", "VerificationState", "Confidence", "Criticality",
		"Types", "UserDefinedFields", "Workflow", "RelatedFindings",
	} {
		if v, ok := req[f]; ok {
			updates[f] = v
		}
	}
	return updates
}

func handleUpdateFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	filters := getMap(req, "Filters")
	updates := make(map[string]any)
	if note := getMap(req, "Note"); note != nil {
		updates["Note"] = note
	}
	if rs := getStr(req, "RecordState"); rs != "" {
		updates["RecordState"] = rs
	}
	store.UpdateFindings(filters, updates)
	return jsonOK(map[string]any{})
}

func handleGetFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	filters := getMap(req, "Filters")
	matches := store.GetFindings(filters)
	out := make([]map[string]any, 0, len(matches))
	for _, m := range matches {
		out = append(out, m.Data)
	}
	return jsonOK(map[string]any{"Findings": out})
}

func handleGetFindingsV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	filters := getMap(req, "Filters")
	matches := store.GetFindings(filters)
	out := make([]map[string]any, 0, len(matches))
	for _, m := range matches {
		out = append(out, m.Data)
	}
	return jsonOK(map[string]any{"Findings": out})
}

func handleGetFindingHistory(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	identifier := getMap(req, "FindingIdentifier")
	if identifier == nil {
		return jsonErr(service.NewAWSError("InvalidInputException",
			"FindingIdentifier is required", http.StatusBadRequest))
	}
	return jsonOK(map[string]any{"Records": []any{}})
}

func handleGetFindingStatisticsV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	matches := store.GetFindings(nil)
	return jsonOK(map[string]any{
		"GroupByResults": []map[string]any{
			{"GroupByField": "Total", "GroupByValues": []map[string]any{{"Value": "All", "Count": len(matches)}}},
		},
	})
}

func handleGetFindingsTrendsV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{"Trends": []any{}})
}

// ── Insights ─────────────────────────────────────────────────────────────────

func handleCreateInsight(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	insight, err := store.CreateInsight(getStr(req, "Name"), getMap(req, "Filters"), getStr(req, "GroupByAttribute"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"InsightArn": insight.InsightArn})
}

func handleUpdateInsight(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.UpdateInsight(getStr(req, "InsightArn"), getStr(req, "Name"), getMap(req, "Filters"), getStr(req, "GroupByAttribute")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDeleteInsight(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "InsightArn")
	if err := store.DeleteInsight(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"InsightArn": arn})
}

func handleGetInsights(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	arns := getStrList(req, "InsightArns")
	insights := store.ListInsights(arns)
	out := make([]map[string]any, 0, len(insights))
	for _, i := range insights {
		out = append(out, insightToMap(i))
	}
	return jsonOK(map[string]any{"Insights": out})
}

func handleGetInsightResults(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "InsightArn")
	if arn == "" {
		return jsonErr(service.NewAWSError("InvalidInputException",
			"InsightArn is required", http.StatusBadRequest))
	}
	insights := store.ListInsights([]string{arn})
	if len(insights) == 0 {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"Insight not found: "+arn, http.StatusBadRequest))
	}
	return jsonOK(map[string]any{
		"InsightResults": map[string]any{
			"InsightArn":       arn,
			"GroupByAttribute": insights[0].GroupByAttribute,
			"ResultValues":     []any{},
		},
	})
}

func insightToMap(i *StoredInsight) map[string]any {
	return map[string]any{
		"InsightArn":       i.InsightArn,
		"Name":             i.Name,
		"Filters":          i.Filters,
		"GroupByAttribute": i.GroupByAttribute,
	}
}

// ── Action Targets ───────────────────────────────────────────────────────────

func handleCreateActionTarget(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	at, err := store.CreateActionTarget(getStr(req, "Name"), getStr(req, "Description"), getStr(req, "Id"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"ActionTargetArn": at.ActionTargetArn})
}

func handleUpdateActionTarget(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.UpdateActionTarget(getStr(req, "ActionTargetArn"), getStr(req, "Name"), getStr(req, "Description")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDeleteActionTarget(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ActionTargetArn")
	if err := store.DeleteActionTarget(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"ActionTargetArn": arn})
}

func handleDescribeActionTargets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	arns := getStrList(req, "ActionTargetArns")
	targets := store.DescribeActionTargets(arns)
	out := make([]map[string]any, 0, len(targets))
	for _, t := range targets {
		out = append(out, map[string]any{
			"ActionTargetArn": t.ActionTargetArn,
			"Name":            t.Name,
			"Description":     t.Description,
		})
	}
	return jsonOK(map[string]any{"ActionTargets": out})
}

// ── Members & Invitations ────────────────────────────────────────────────────

func handleCreateMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	accounts := getMapList(req, "AccountDetails")
	_, unprocessed := store.CreateMembers(accounts)
	return jsonOK(map[string]any{"UnprocessedAccounts": unprocessed})
}

func handleDeleteMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	ids := getStrList(req, "AccountIds")
	unprocessed := store.DeleteMembers(ids)
	return jsonOK(map[string]any{"UnprocessedAccounts": unprocessed})
}

func handleDisassociateMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	ids := getStrList(req, "AccountIds")
	store.DisassociateMembers(ids)
	return jsonOK(map[string]any{})
}

func handleGetMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	ids := getStrList(req, "AccountIds")
	members, unprocessed := store.GetMembers(ids)
	out := make([]map[string]any, 0, len(members))
	for _, m := range members {
		out = append(out, memberToMap(m))
	}
	return jsonOK(map[string]any{
		"Members":             out,
		"UnprocessedAccounts": unprocessed,
	})
}

func handleListMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	members := store.ListMembers()
	out := make([]map[string]any, 0, len(members))
	for _, m := range members {
		out = append(out, memberToMap(m))
	}
	return jsonOK(map[string]any{"Members": out})
}

func handleInviteMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	ids := getStrList(req, "AccountIds")
	unprocessed := store.InviteMembers(ids)
	return jsonOK(map[string]any{"UnprocessedAccounts": unprocessed})
}

func memberToMap(m *StoredMember) map[string]any {
	return map[string]any{
		"AccountId":       m.AccountID,
		"Email":           m.Email,
		"MasterId":        m.MasterID,
		"AdministratorId": m.AdministratorID,
		"MemberStatus":    m.MemberStatus,
		"InvitedAt":       rfc3339(m.InvitedAt),
		"UpdatedAt":       rfc3339(m.UpdatedAt),
	}
}

// ── Administrator / Invitations ──────────────────────────────────────────────

func handleAcceptAdministratorInvitation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.AcceptAdministrator(getStr(req, "AdministratorId")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleAcceptInvitation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.AcceptAdministrator(getStr(req, "MasterId")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDisassociateFromAdministratorAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	store.DisassociateAdministrator()
	return jsonOK(map[string]any{})
}

func handleDisassociateFromMasterAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	store.DisassociateAdministrator()
	return jsonOK(map[string]any{})
}

func handleGetAdministratorAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	id, status := store.GetAdministrator()
	return jsonOK(map[string]any{
		"Administrator": map[string]any{
			"AccountId":          id,
			"InvitationId":       "",
			"InvitedAt":          "",
			"MemberStatus":       status,
		},
	})
}

func handleGetMasterAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	id, status := store.GetAdministrator()
	return jsonOK(map[string]any{
		"Master": map[string]any{
			"AccountId":          id,
			"InvitationId":       "",
			"InvitedAt":          "",
			"MemberStatus":       status,
		},
	})
}

func handleListInvitations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	invs := store.ListInvitations()
	out := make([]map[string]any, 0, len(invs))
	for _, i := range invs {
		out = append(out, map[string]any{
			"AccountId":    i.AccountID,
			"InvitationId": i.InvitationID,
			"InvitedAt":    rfc3339(i.InvitedAt),
			"MemberStatus": i.MemberStatus,
		})
	}
	return jsonOK(map[string]any{"Invitations": out})
}

func handleDeclineInvitations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	ids := getStrList(req, "AccountIds")
	unprocessed := store.DeclineInvitations(ids)
	return jsonOK(map[string]any{"UnprocessedAccounts": unprocessed})
}

func handleDeleteInvitations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	ids := getStrList(req, "AccountIds")
	unprocessed := store.DeclineInvitations(ids)
	return jsonOK(map[string]any{"UnprocessedAccounts": unprocessed})
}

func handleGetInvitationsCount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	invs := store.ListInvitations()
	return jsonOK(map[string]any{"InvitationsCount": len(invs)})
}

// ── Organization ─────────────────────────────────────────────────────────────

func handleEnableOrganizationAdminAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.EnableOrgAdmin(getStr(req, "AdminAccountId")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDisableOrganizationAdminAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	if err := store.DisableOrgAdmin(); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListOrganizationAdminAccounts(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	admins := store.ListOrgAdmins()
	out := make([]map[string]any, 0, len(admins))
	for _, a := range admins {
		out = append(out, map[string]any{
			"AccountId": a,
			"Status":    "ENABLED",
		})
	}
	return jsonOK(map[string]any{"AdminAccounts": out})
}

func handleDescribeOrganizationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	cfg := store.GetOrgConfig()
	return jsonOK(cfg)
}

func handleUpdateOrganizationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	store.UpdateOrgConfig(req)
	return jsonOK(map[string]any{})
}

// ── Finding aggregators (V1) ─────────────────────────────────────────────────

func handleCreateFindingAggregator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	agg, err := store.CreateFindingAggregator(getStr(req, "RegionLinkingMode"), getStrList(req, "Regions"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(findingAggregatorToMap(agg))
}

func handleUpdateFindingAggregator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	agg, err := store.UpdateFindingAggregator(getStr(req, "FindingAggregatorArn"), getStr(req, "RegionLinkingMode"), getStrList(req, "Regions"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(findingAggregatorToMap(agg))
}

func handleGetFindingAggregator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	agg, err := store.GetFindingAggregator(getStr(req, "FindingAggregatorArn"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(findingAggregatorToMap(agg))
}

func handleDeleteFindingAggregator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.DeleteFindingAggregator(getStr(req, "FindingAggregatorArn")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListFindingAggregators(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	aggs := store.ListFindingAggregators()
	out := make([]map[string]any, 0, len(aggs))
	for _, a := range aggs {
		out = append(out, map[string]any{"FindingAggregatorArn": a.FindingAggregatorArn})
	}
	return jsonOK(map[string]any{"FindingAggregators": out})
}

func findingAggregatorToMap(a *StoredFindingAggregator) map[string]any {
	return map[string]any{
		"FindingAggregatorArn":     a.FindingAggregatorArn,
		"FindingAggregationRegion": a.FindingAggregationRegion,
		"RegionLinkingMode":        a.RegionLinkingMode,
		"Regions":                  a.Regions,
	}
}

// ── Aggregators V2 ───────────────────────────────────────────────────────────

func handleCreateAggregatorV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	agg, err := store.CreateAggregatorV2(getStr(req, "RegionLinkingMode"), getStrList(req, "LinkedRegions"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(aggregatorV2ToMap(agg))
}

func handleUpdateAggregatorV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "AggregatorV2Arn")
	if id == "" {
		id = getStr(req, "AggregatorV2Id")
	}
	id = lastSegment(id)
	agg, err := store.UpdateAggregatorV2(id, getStr(req, "RegionLinkingMode"), getStrList(req, "LinkedRegions"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(aggregatorV2ToMap(agg))
}

func handleGetAggregatorV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "AggregatorV2Arn")
	if id == "" {
		id = getStr(req, "AggregatorV2Id")
	}
	agg, err := store.GetAggregatorV2(lastSegment(id))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(aggregatorV2ToMap(agg))
}

func handleDeleteAggregatorV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "AggregatorV2Arn")
	if id == "" {
		id = getStr(req, "AggregatorV2Id")
	}
	if err := store.DeleteAggregatorV2(lastSegment(id)); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListAggregatorsV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	aggs := store.ListAggregatorsV2()
	out := make([]map[string]any, 0, len(aggs))
	for _, a := range aggs {
		out = append(out, aggregatorV2ToMap(a))
	}
	return jsonOK(map[string]any{"AggregatorsV2": out})
}

func aggregatorV2ToMap(a *StoredAggregatorV2) map[string]any {
	return map[string]any{
		"AggregatorV2Arn":   a.AggregatorV2Arn,
		"AggregationRegion": a.AggregationRegion,
		"RegionLinkingMode": a.RegionLinkingMode,
		"LinkedRegions":     a.LinkedRegions,
	}
}

// ── Automation rules (V1) ────────────────────────────────────────────────────

func handleCreateAutomationRule(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	rule, err := store.CreateAutomationRule(
		getStr(req, "RuleName"),
		getStr(req, "Description"),
		getStr(req, "RuleStatus"),
		getInt(req, "RuleOrder"),
		getBool(req, "IsTerminal"),
		getMap(req, "Criteria"),
		getMapList(req, "Actions"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"RuleArn": rule.RuleArn})
}

func handleBatchGetAutomationRules(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	arns := getStrList(req, "AutomationRulesArns")
	rules, unprocessed := store.GetAutomationRules(arns)
	out := make([]map[string]any, 0, len(rules))
	for _, r := range rules {
		out = append(out, automationRuleToMap(r))
	}
	return jsonOK(map[string]any{
		"Rules":              out,
		"UnprocessedAutomationRules": unprocessed,
	})
}

func handleBatchUpdateAutomationRules(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	updates := getMapList(req, "UpdateAutomationRulesRequestItems")
	processed, unprocessed := store.UpdateAutomationRules(updates)
	out := make([]map[string]any, 0, len(processed))
	for _, arn := range processed {
		out = append(out, map[string]any{"RuleArn": arn})
	}
	return jsonOK(map[string]any{
		"ProcessedAutomationRules":   out,
		"UnprocessedAutomationRules": unprocessed,
	})
}

func handleBatchDeleteAutomationRules(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	arns := getStrList(req, "AutomationRulesArns")
	processed, unprocessed := store.DeleteAutomationRules(arns)
	out := make([]map[string]any, 0, len(processed))
	for _, a := range processed {
		out = append(out, map[string]any{"RuleArn": a})
	}
	return jsonOK(map[string]any{
		"ProcessedAutomationRules":   out,
		"UnprocessedAutomationRules": unprocessed,
	})
}

func handleListAutomationRules(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	rules := store.ListAutomationRules()
	out := make([]map[string]any, 0, len(rules))
	for _, r := range rules {
		out = append(out, automationRuleToMap(r))
	}
	return jsonOK(map[string]any{"AutomationRulesMetadata": out})
}

func automationRuleToMap(r *StoredAutomationRule) map[string]any {
	return map[string]any{
		"RuleArn":     r.RuleArn,
		"RuleId":      r.RuleID,
		"RuleName":    r.RuleName,
		"Description": r.Description,
		"RuleStatus":  r.RuleStatus,
		"RuleOrder":   r.RuleOrder,
		"IsTerminal":  r.IsTerminal,
		"Criteria":    r.Criteria,
		"Actions":     r.Actions,
		"CreatedAt":   rfc3339(r.CreatedAt),
		"UpdatedAt":   rfc3339(r.UpdatedAt),
		"CreatedBy":   r.CreatedBy,
	}
}

// ── Automation rules V2 ──────────────────────────────────────────────────────

func handleCreateAutomationRuleV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	rule, err := store.CreateAutomationRuleV2(
		getStr(req, "RuleName"),
		getStr(req, "Description"),
		getStr(req, "RuleStatus"),
		getFloat(req, "RuleOrder"),
		getMap(req, "Criteria"),
		getMapList(req, "Actions"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"RuleArn": rule.RuleArn,
		"RuleId":  rule.RuleID,
	})
}

func handleGetAutomationRuleV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "Identifier")
	if id == "" {
		id = getStr(req, "RuleId")
	}
	rule, err := store.GetAutomationRuleV2(lastSegment(id))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(automationRuleV2ToMap(rule))
}

func handleUpdateAutomationRuleV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "Identifier")
	if id == "" {
		id = getStr(req, "RuleId")
	}
	if id == "" {
		id = getStr(req, "RuleArn")
	}
	rule, err := store.UpdateAutomationRuleV2(
		lastSegment(id),
		getStr(req, "RuleName"),
		getStr(req, "Description"),
		getStr(req, "RuleStatus"),
		getFloatPtr(req, "RuleOrder"),
		getMap(req, "Criteria"),
		getMapList(req, "Actions"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(automationRuleV2ToMap(rule))
}

func handleDeleteAutomationRuleV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "Identifier")
	if id == "" {
		id = getStr(req, "RuleId")
	}
	if id == "" {
		id = getStr(req, "RuleArn")
	}
	if err := store.DeleteAutomationRuleV2(lastSegment(id)); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListAutomationRulesV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	rules := store.ListAutomationRulesV2()
	out := make([]map[string]any, 0, len(rules))
	for _, r := range rules {
		out = append(out, automationRuleV2ToMap(r))
	}
	return jsonOK(map[string]any{"Rules": out})
}

func automationRuleV2ToMap(r *StoredAutomationRuleV2) map[string]any {
	return map[string]any{
		"RuleArn":     r.RuleArn,
		"RuleId":      r.RuleID,
		"RuleName":    r.RuleName,
		"Description": r.Description,
		"RuleStatus":  r.RuleStatus,
		"RuleOrder":   r.RuleOrder,
		"Criteria":    r.Criteria,
		"Actions":     r.Actions,
		"CreatedAt":   rfc3339(r.CreatedAt),
		"UpdatedAt":   rfc3339(r.UpdatedAt),
	}
}

// ── Configuration policies ───────────────────────────────────────────────────

func handleCreateConfigurationPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	p, err := store.CreateConfigurationPolicy(getStr(req, "Name"), getStr(req, "Description"), getMap(req, "ConfigurationPolicy"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(configurationPolicyToMap(p))
}

func handleUpdateConfigurationPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "Identifier")
	if id == "" {
		id = getStr(req, "Id")
	}
	p, err := store.UpdateConfigurationPolicy(id, getStr(req, "Name"), getStr(req, "Description"), getMap(req, "ConfigurationPolicy"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(configurationPolicyToMap(p))
}

func handleDeleteConfigurationPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "Identifier")
	if id == "" {
		id = getStr(req, "Id")
	}
	if err := store.DeleteConfigurationPolicy(id); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleGetConfigurationPolicy(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "Identifier")
	if id == "" {
		id = getStr(req, "Id")
	}
	p, err := store.GetConfigurationPolicy(id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(configurationPolicyToMap(p))
}

func handleListConfigurationPolicies(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	pols := store.ListConfigurationPolicies()
	out := make([]map[string]any, 0, len(pols))
	for _, p := range pols {
		out = append(out, map[string]any{
			"Arn":         p.Arn,
			"Id":          p.ID,
			"Name":        p.Name,
			"Description": p.Description,
			"UpdatedAt":   rfc3339(p.UpdatedAt),
		})
	}
	return jsonOK(map[string]any{"ConfigurationPolicySummaries": out})
}

func handleStartConfigurationPolicyAssociation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	target := getMap(req, "Target")
	targetID := ""
	targetType := ""
	if target != nil {
		if id := getStr(target, "AccountId"); id != "" {
			targetID = id
			targetType = "ACCOUNT"
		} else if id := getStr(target, "OrganizationalUnitId"); id != "" {
			targetID = id
			targetType = "ORGANIZATIONAL_UNIT"
		} else if id := getStr(target, "RootId"); id != "" {
			targetID = id
			targetType = "ROOT"
		}
	}
	assoc, err := store.StartConfigurationPolicyAssociation(getStr(req, "ConfigurationPolicyIdentifier"), targetID, targetType)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(configAssocToMap(assoc))
}

func handleStartConfigurationPolicyDisassociation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	target := getMap(req, "Target")
	targetID := ""
	if target != nil {
		if id := getStr(target, "AccountId"); id != "" {
			targetID = id
		} else if id := getStr(target, "OrganizationalUnitId"); id != "" {
			targetID = id
		} else if id := getStr(target, "RootId"); id != "" {
			targetID = id
		}
	}
	if err := store.StartConfigurationPolicyDisassociation(getStr(req, "ConfigurationPolicyIdentifier"), targetID); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleGetConfigurationPolicyAssociation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	target := getMap(req, "Target")
	targetID := ""
	if target != nil {
		if id := getStr(target, "AccountId"); id != "" {
			targetID = id
		} else if id := getStr(target, "OrganizationalUnitId"); id != "" {
			targetID = id
		} else if id := getStr(target, "RootId"); id != "" {
			targetID = id
		}
	}
	a, err := store.GetConfigurationPolicyAssociation(targetID)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(configAssocToMap(a))
}

func handleBatchGetConfigurationPolicyAssociations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	targets := getMapList(req, "ConfigurationPolicyAssociationIdentifiers")
	ids := make([]string, 0, len(targets))
	for _, t := range targets {
		if v := getStr(t, "AccountId"); v != "" {
			ids = append(ids, v)
		} else if v := getStr(t, "OrganizationalUnitId"); v != "" {
			ids = append(ids, v)
		} else if v := getStr(t, "RootId"); v != "" {
			ids = append(ids, v)
		}
	}
	found, unprocessed := store.BatchGetConfigurationPolicyAssociations(ids)
	out := make([]map[string]any, 0, len(found))
	for _, a := range found {
		out = append(out, configAssocToMap(a))
	}
	return jsonOK(map[string]any{
		"ConfigurationPolicyAssociations":           out,
		"UnprocessedConfigurationPolicyAssociations": unprocessed,
	})
}

func handleListConfigurationPolicyAssociations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	assocs := store.ListConfigurationPolicyAssociations()
	out := make([]map[string]any, 0, len(assocs))
	for _, a := range assocs {
		out = append(out, configAssocToMap(a))
	}
	return jsonOK(map[string]any{"ConfigurationPolicyAssociationSummaries": out})
}

func configurationPolicyToMap(p *StoredConfigurationPolicy) map[string]any {
	return map[string]any{
		"Arn":                 p.Arn,
		"Id":                  p.ID,
		"Name":                p.Name,
		"Description":         p.Description,
		"UpdatedAt":           rfc3339(p.UpdatedAt),
		"CreatedAt":           rfc3339(p.CreatedAt),
		"ConfigurationPolicy": p.ConfigurationPolicy,
	}
}

func configAssocToMap(a *StoredConfigurationPolicyAssociation) map[string]any {
	return map[string]any{
		"TargetId":                 a.TargetID,
		"TargetType":               a.TargetType,
		"ConfigurationPolicyId":    a.ConfigurationPolicyID,
		"AssociationType":          a.AssociationType,
		"AssociationStatus":        a.AssociationStatus,
		"AssociationStatusMessage": a.AssociationStatusMessage,
		"UpdatedAt":                rfc3339(a.UpdatedAt),
	}
}

// ── Connectors V2 ────────────────────────────────────────────────────────────

func handleCreateConnectorV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	c, err := store.CreateConnectorV2(
		getStr(req, "Name"),
		getStr(req, "Description"),
		getStr(req, "Provider"),
		getStr(req, "KmsKeyArn"),
		getMap(req, "ProviderSummary"),
		parseTagMap(req, "Tags"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"ConnectorArn": c.ConnectorArn,
		"ConnectorId":  c.ConnectorID,
	})
}

func handleGetConnectorV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "ConnectorId")
	if id == "" {
		id = getStr(req, "ConnectorArn")
	}
	c, err := store.GetConnectorV2(lastSegment(id))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(connectorV2ToMap(c))
}

func handleUpdateConnectorV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "ConnectorId")
	if id == "" {
		id = getStr(req, "ConnectorArn")
	}
	c, err := store.UpdateConnectorV2(lastSegment(id), getStr(req, "Description"), getMap(req, "ProviderSummary"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(connectorV2ToMap(c))
}

func handleDeleteConnectorV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "ConnectorId")
	if id == "" {
		id = getStr(req, "ConnectorArn")
	}
	if err := store.DeleteConnectorV2(lastSegment(id)); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListConnectorsV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	conns := store.ListConnectorsV2()
	out := make([]map[string]any, 0, len(conns))
	for _, c := range conns {
		out = append(out, connectorV2ToMap(c))
	}
	return jsonOK(map[string]any{"Connectors": out})
}

func handleRegisterConnectorV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "ConnectorId")
	if id == "" {
		id = getStr(req, "ConnectorArn")
	}
	c, err := store.GetConnectorV2(lastSegment(id))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(connectorV2ToMap(c))
}

func connectorV2ToMap(c *StoredConnectorV2) map[string]any {
	return map[string]any{
		"ConnectorArn":     c.ConnectorArn,
		"ConnectorId":      c.ConnectorID,
		"Name":             c.Name,
		"Description":      c.Description,
		"Provider":         c.ProviderName,
		"HealthStatus":     c.HealthStatus,
		"KmsKeyArn":        c.KmsKeyArn,
		"CreatedAt":        rfc3339(c.CreatedAt),
		"LastUpdatedAt":    rfc3339(c.LastUpdatedAt),
		"ProviderSummary":  c.ProviderSummary,
	}
}

// ── Tickets V2 ───────────────────────────────────────────────────────────────

func handleCreateTicketV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	if getStr(req, "ConnectorId") == "" {
		return jsonErr(service.NewAWSError("InvalidInputException",
			"ConnectorId is required", http.StatusBadRequest))
	}
	return jsonOK(map[string]any{
		"TicketId":  generateID(),
		"TicketSrcUrl": "https://example.invalid/tickets/" + generateID(),
	})
}

// ── Resources V2 ─────────────────────────────────────────────────────────────

func handleGetResourcesV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{"Resources": []any{}})
}

func handleGetResourcesStatisticsV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"GroupByResults": []map[string]any{
			{"GroupByField": "Total", "GroupByValues": []map[string]any{{"Value": "All", "Count": 0}}},
		},
	})
}

func handleGetResourcesTrendsV2(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{"Trends": []any{}})
}

// ── Tags ─────────────────────────────────────────────────────────────────────

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ResourceArn")
	if arn == "" {
		// Allow path-based ARN passed via query parameter.
		if v, ok := ctx.Params["resourceArn"]; ok {
			arn = v
		}
	}
	if arn == "" && ctx.RawRequest != nil {
		arn = arnFromPath(ctx.RawRequest.URL.Path)
	}
	if arn == "" {
		return jsonErr(service.NewAWSError("InvalidInputException",
			"ResourceArn is required", http.StatusBadRequest))
	}
	store.TagResource(arn, parseTagMap(req, "Tags"))
	return jsonOK(map[string]any{})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, awsErr := decodeBody(ctx)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ResourceArn")
	if arn == "" && ctx.RawRequest != nil {
		arn = arnFromPath(ctx.RawRequest.URL.Path)
	}
	if arn == "" {
		return jsonErr(service.NewAWSError("InvalidInputException",
			"ResourceArn is required", http.StatusBadRequest))
	}
	keys := getStrList(req, "TagKeys")
	if len(keys) == 0 && ctx.Params != nil {
		if v, ok := ctx.Params["tagKeys"]; ok {
			keys = strings.Split(v, ",")
		}
	}
	store.UntagResource(arn, keys)
	return jsonOK(map[string]any{})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	req, _ := decodeBody(ctx)
	arn := ""
	if req != nil {
		arn = getStr(req, "ResourceArn")
	}
	if arn == "" && ctx.RawRequest != nil {
		arn = arnFromPath(ctx.RawRequest.URL.Path)
	}
	if arn == "" {
		return jsonErr(service.NewAWSError("InvalidInputException",
			"ResourceArn is required", http.StatusBadRequest))
	}
	tags := store.ListTags(arn)
	tagsMap := make(map[string]any, len(tags))
	for k, v := range tags {
		tagsMap[k] = v
	}
	return jsonOK(map[string]any{"Tags": tagsMap})
}

// arnFromPath extracts an ARN segment from a Security Hub REST path, if present.
// Real Security Hub embeds the URL-encoded ARN after /tags/.
func arnFromPath(path string) string {
	const marker = "/tags/"
	if i := strings.Index(path, marker); i >= 0 {
		return path[i+len(marker):]
	}
	return ""
}
