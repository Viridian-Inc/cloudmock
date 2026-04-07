package support

import (
	"net/http"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func str(params map[string]any, key string) string {
	if v, ok := params[key].(string); ok {
		return v
	}
	return ""
}

func strSlice(params map[string]any, key string) []string {
	if v, ok := params[key].([]any); ok {
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func boolVal(params map[string]any, key string) bool {
	if v, ok := params[key].(bool); ok {
		return v
	}
	return false
}

func caseResponse(sc *SupportCase) map[string]any {
	resp := map[string]any{
		"caseId":            sc.CaseID,
		"displayId":         sc.DisplayID,
		"subject":           sc.Subject,
		"status":            sc.Status,
		"serviceCode":       sc.ServiceCode,
		"severityCode":      sc.SeverityCode,
		"categoryCode":      sc.CategoryCode,
		"submittedBy":       sc.SubmittedBy,
		"timeCreated":       sc.TimeCreated.Format(time.RFC3339),
		"language":          sc.Language,
		"ccEmailAddresses":  sc.CCEmailAddresses,
	}

	if len(sc.RecentCommunications) > 0 {
		comms := make([]map[string]any, 0, len(sc.RecentCommunications))
		for _, c := range sc.RecentCommunications {
			comms = append(comms, map[string]any{
				"body":        c.Body,
				"submittedBy": c.SubmittedBy,
				"timeCreated": c.TimeCreated.Format(time.RFC3339),
			})
		}
		resp["recentCommunications"] = map[string]any{
			"communications": comms,
		}
	}
	return resp
}

// validSeverityCodes is the set of allowed severity codes for support cases.
var validSeverityCodes = map[string]bool{
	"low": true, "normal": true, "high": true, "urgent": true, "critical": true,
}

func handleCreateCase(params map[string]any, store *Store) (*service.Response, error) {
	subject := str(params, "subject")
	if subject == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"1 validation error detected: Value at 'subject' failed to satisfy constraint: Member must not be null",
			http.StatusBadRequest))
	}

	severityCode := str(params, "severityCode")
	if severityCode != "" && !validSeverityCodes[severityCode] {
		return jsonErr(service.NewAWSError("CaseCreationLimitExceeded",
			"Invalid severity code: "+severityCode+". Allowed values: low, normal, high, urgent, critical",
			http.StatusBadRequest))
	}

	communicationBody := str(params, "communicationBody")
	if communicationBody == "" {
		return jsonErr(service.NewAWSError("ValidationException",
			"1 validation error detected: Value at 'communicationBody' failed to satisfy constraint: Member must not be null",
			http.StatusBadRequest))
	}

	sc, _ := store.CreateCase(
		subject,
		str(params, "serviceCode"),
		severityCode,
		str(params, "categoryCode"),
		communicationBody,
		str(params, "submittedBy"),
		str(params, "language"),
		strSlice(params, "ccEmailAddresses"),
	)

	return jsonOK(map[string]any{"caseId": sc.CaseID})
}

func handleDescribeCases(params map[string]any, store *Store) (*service.Response, error) {
	includeResolved := boolVal(params, "includeResolvedCases")

	var caseIDs []string
	if ids, ok := params["caseIdList"].([]any); ok {
		for _, id := range ids {
			if sv, ok := id.(string); ok {
				caseIDs = append(caseIDs, sv)
			}
		}
	}

	cases := store.ListCases(includeResolved)

	if len(caseIDs) > 0 {
		idSet := make(map[string]bool)
		for _, id := range caseIDs {
			idSet[id] = true
		}
		filtered := make([]*SupportCase, 0)
		for _, sc := range cases {
			if idSet[sc.CaseID] {
				filtered = append(filtered, sc)
			}
		}
		cases = filtered
	}

	out := make([]map[string]any, 0, len(cases))
	for _, sc := range cases {
		out = append(out, caseResponse(sc))
	}

	return jsonOK(map[string]any{"cases": out})
}

func handleResolveCase(params map[string]any, store *Store) (*service.Response, error) {
	caseID := str(params, "caseId")
	if caseID == "" {
		return jsonErr(service.ErrValidation("caseId is required"))
	}

	sc, err := store.ResolveCase(caseID)
	if err != nil {
		return jsonErr(service.ErrNotFound("Case", caseID))
	}

	return jsonOK(map[string]any{
		"initialCaseStatus": "opened",
		"finalCaseStatus":   sc.Status,
	})
}

func handleDescribeTrustedAdvisorChecks(store *Store) (*service.Response, error) {
	checks := store.ListTrustedAdvisorChecks()
	out := make([]map[string]any, 0, len(checks))
	for _, c := range checks {
		out = append(out, map[string]any{
			"id":          c.ID,
			"name":        c.Name,
			"description": c.Description,
			"category":    c.Category,
			"metadata":    c.Metadata,
		})
	}
	return jsonOK(map[string]any{"checks": out})
}

func handleDescribeTrustedAdvisorCheckResult(params map[string]any, store *Store) (*service.Response, error) {
	checkID := str(params, "checkId")
	if checkID == "" {
		return jsonErr(service.ErrValidation("checkId is required"))
	}

	result, ok := store.GetTrustedAdvisorCheckResult(checkID)
	if !ok {
		return jsonErr(service.ErrNotFound("TrustedAdvisorCheck", checkID))
	}

	return jsonOK(map[string]any{
		"result": map[string]any{
			"checkId":          result.CheckID,
			"status":           result.Status,
			"timestamp":        result.Timestamp,
			"resourcesSummary": result.ResourcesSummary,
			"flaggedResources": result.FlaggedResources,
		},
	})
}

func handleRefreshTrustedAdvisorCheck(params map[string]any, store *Store) (*service.Response, error) {
	checkID := str(params, "checkId")
	if checkID == "" {
		return jsonErr(service.ErrValidation("checkId is required"))
	}

	return jsonOK(map[string]any{
		"status": map[string]any{
			"checkId":                    checkID,
			"status":                     "enqueued",
			"millisUntilNextRefreshable": 3600000,
		},
	})
}

func handleDescribeServices(store *Store) (*service.Response, error) {
	services := store.GetServices()
	out := make([]map[string]any, 0, len(services))
	for _, svc := range services {
		cats := make([]map[string]any, 0, len(svc.Categories))
		for _, c := range svc.Categories {
			cats = append(cats, map[string]any{"code": c.Code, "name": c.Name})
		}
		out = append(out, map[string]any{
			"code":       svc.Code,
			"name":       svc.Name,
			"categories": cats,
		})
	}
	return jsonOK(map[string]any{"services": out})
}

func handleDescribeSeverityLevels(store *Store) (*service.Response, error) {
	levels := store.GetSeverityLevels()
	out := make([]map[string]any, 0, len(levels))
	for _, l := range levels {
		out = append(out, map[string]any{"code": l.Code, "name": l.Name})
	}
	return jsonOK(map[string]any{"severityLevels": out})
}

func handleAddCommunicationToCase(params map[string]any, store *Store) (*service.Response, error) {
	caseID := str(params, "caseId")
	body := str(params, "communicationBody")
	if caseID == "" || body == "" {
		return jsonErr(service.ErrValidation("caseId and communicationBody are required"))
	}

	if err := store.AddCommunication(caseID, body, "cloudmock-user"); err != nil {
		return jsonErr(service.ErrNotFound("Case", caseID))
	}
	return jsonOK(map[string]any{"result": true})
}

func handleDescribeCommunications(params map[string]any, store *Store) (*service.Response, error) {
	caseID := str(params, "caseId")
	if caseID == "" {
		return jsonErr(service.ErrValidation("caseId is required"))
	}

	comms := store.GetCommunications(caseID)
	if comms == nil {
		return jsonErr(service.ErrNotFound("Case", caseID))
	}

	out := make([]map[string]any, 0, len(comms))
	for _, c := range comms {
		out = append(out, map[string]any{
			"caseId":      c.CaseID,
			"body":        c.Body,
			"submittedBy": c.SubmittedBy,
			"timeCreated": c.TimeCreated.Format(time.RFC3339),
		})
	}
	return jsonOK(map[string]any{"communications": out})
}
