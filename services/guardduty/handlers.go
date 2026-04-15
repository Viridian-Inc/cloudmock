package guardduty

import (
	"net/http"
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
		return service.NewAWSError("BadRequestException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func getStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
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

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getBoolPtr(m map[string]any, key string) *bool {
	v, ok := m[key]
	if !ok {
		return nil
	}
	if b, ok := v.(bool); ok {
		return &b
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

// getStringMap reads a top-level {"k1":"v1"} object as map[string]string.
func getStringMap(m map[string]any, key string) map[string]string {
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

func rfc3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func validDetectorID(id string) *service.AWSError {
	if id == "" {
		return service.NewAWSError("BadRequestException",
			"detectorId is required.", http.StatusBadRequest)
	}
	return nil
}

// ── Conversion helpers ──────────────────────────────────────────────────────

func detectorMap(d *StoredDetector) map[string]any {
	out := map[string]any{
		"createdAt":                  rfc3339(d.CreatedAt),
		"updatedAt":                  rfc3339(d.UpdatedAt),
		"findingPublishingFrequency": d.FindingPublishingFrequency,
		"serviceRole":                d.ServiceRole,
		"status":                     d.Status,
		"tags":                       d.Tags,
	}
	if d.DataSources != nil {
		out["dataSources"] = d.DataSources
	}
	if d.Features != nil {
		out["features"] = d.Features
	}
	return out
}

func filterMap(f *StoredFilter) map[string]any {
	return map[string]any{
		"action":          f.Action,
		"description":     f.Description,
		"findingCriteria": f.FindingCriteria,
		"name":            f.Name,
		"rank":            f.Rank,
		"tags":            f.Tags,
	}
}

func ipSetMap(set *StoredIPSet) map[string]any {
	out := map[string]any{
		"format":   set.Format,
		"location": set.Location,
		"name":     set.Name,
		"status":   set.Status,
		"tags":     set.Tags,
	}
	if set.ExpectedBucketOwner != "" {
		out["expectedBucketOwner"] = set.ExpectedBucketOwner
	}
	return out
}

func threatIntelSetMap(set *StoredThreatIntelSet) map[string]any {
	out := map[string]any{
		"format":   set.Format,
		"location": set.Location,
		"name":     set.Name,
		"status":   set.Status,
		"tags":     set.Tags,
	}
	if set.ExpectedBucketOwner != "" {
		out["expectedBucketOwner"] = set.ExpectedBucketOwner
	}
	return out
}

func entitySetMap(set *StoredEntitySet) map[string]any {
	out := map[string]any{
		"format":    set.Format,
		"location":  set.Location,
		"name":      set.Name,
		"status":    set.Status,
		"tags":      set.Tags,
		"createdAt": rfc3339(set.CreatedAt),
		"updatedAt": rfc3339(set.UpdatedAt),
	}
	if set.ExpectedBucketOwner != "" {
		out["expectedBucketOwner"] = set.ExpectedBucketOwner
	}
	if set.ErrorDetails != "" {
		out["errorDetails"] = set.ErrorDetails
	}
	return out
}

func memberMap(m *StoredMember) map[string]any {
	return map[string]any{
		"accountId":          m.AccountID,
		"detectorId":         m.DetectorID,
		"email":              m.Email,
		"masterId":           m.MasterID,
		"administratorId":    m.AdministratorID,
		"relationshipStatus": m.RelationshipStatus,
		"invitedAt":          rfc3339(m.InvitedAt),
		"updatedAt":          rfc3339(m.UpdatedAt),
	}
}

func invitationMap(inv *StoredInvitation) map[string]any {
	return map[string]any{
		"accountId":          inv.AccountID,
		"invitationId":       inv.InvitationID,
		"relationshipStatus": inv.RelationshipStatus,
		"invitedAt":          rfc3339(inv.InvitedAt),
	}
}

func publishingDestinationMap(d *StoredPublishingDestination) map[string]any {
	return map[string]any{
		"destinationId":                   d.DestinationID,
		"destinationType":                 d.DestinationType,
		"destinationProperties":           d.DestinationProperties,
		"status":                          d.Status,
		"publishingFailureStartTimestamp": d.PublishingFailureStartTimestamp,
	}
}

func findingMap(f *StoredFinding) map[string]any {
	return map[string]any{
		"accountId":     f.AccountID,
		"arn":           f.Arn,
		"confidence":    f.Confidence,
		"createdAt":     rfc3339(f.CreatedAt),
		"description":   f.Description,
		"id":            f.ID,
		"partition":     f.Partition,
		"region":        f.Region,
		"resource":      f.Resource,
		"schemaVersion": f.SchemaVersion,
		"service":       f.Service,
		"severity":      f.Severity,
		"title":         f.Title,
		"type":          f.Type,
		"updatedAt":     rfc3339(f.UpdatedAt),
	}
}

func malwareScanMap(scan *StoredMalwareScan) map[string]any {
	return map[string]any{
		"scanId":         scan.ScanID,
		"detectorId":     scan.DetectorID,
		"resourceArn":    scan.ResourceArn,
		"resourceType":   scan.ResourceType,
		"scanType":       scan.ScanType,
		"scanStatus":     scan.ScanStatus,
		"scanStartedAt":  rfc3339(scan.ScanStartedAt),
		"scanCompletedAt": rfc3339(scan.ScanEndedAt),
		"triggerDetails": scan.TriggerDetails,
	}
}

func malwareProtectionPlanMap(p *StoredMalwareProtectionPlan) map[string]any {
	return map[string]any{
		"malwareProtectionPlanId": p.PlanID,
		"arn":                     p.Arn,
		"role":                    p.Role,
		"protectedResource":       p.ProtectedResource,
		"actions":                 p.Actions,
		"status":                  p.Status,
		"statusReasons":           p.StatusReasons,
		"tags":                    p.Tags,
		"createdAt":               rfc3339(p.CreatedAt),
	}
}

// ── Detector handlers ───────────────────────────────────────────────────────

func handleCreateDetector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	enable := getBool(req, "enable")
	freq := getStr(req, "findingPublishingFrequency")
	tags := getStringMap(req, "tags")
	dataSources := getMap(req, "dataSources")
	features := getMapList(req, "features")
	d := store.CreateDetector(enable, freq, tags, dataSources, features)
	return jsonOK(map[string]any{"detectorId": d.DetectorID})
}

func handleGetDetector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "detectorId")
	if id == "" {
		id = ctx.Params["detectorId"]
	}
	if err := validDetectorID(id); err != nil {
		return jsonErr(err)
	}
	d, err := store.GetDetector(id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(detectorMap(d))
}

func handleListDetectors(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	ids := store.ListDetectors()
	return jsonOK(map[string]any{"detectorIds": ids})
}

func handleUpdateDetector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "detectorId")
	if id == "" {
		id = ctx.Params["detectorId"]
	}
	if err := validDetectorID(id); err != nil {
		return jsonErr(err)
	}
	enable := getBoolPtr(req, "enable")
	freq := getStr(req, "findingPublishingFrequency")
	dataSources := getMap(req, "dataSources")
	features := getMapList(req, "features")
	if err := store.UpdateDetector(id, enable, freq, dataSources, features); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDeleteDetector(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "detectorId")
	if id == "" {
		id = ctx.Params["detectorId"]
	}
	if err := validDetectorID(id); err != nil {
		return jsonErr(err)
	}
	if err := store.DeleteDetector(id); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── Filter handlers ─────────────────────────────────────────────────────────

func handleCreateFilter(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	name := getStr(req, "name")
	if name == "" {
		return jsonErr(service.NewAWSError("BadRequestException", "name is required.", http.StatusBadRequest))
	}
	action := getStr(req, "action")
	desc := getStr(req, "description")
	rank := getInt(req, "rank")
	criteria := getMap(req, "findingCriteria")
	tags := getStringMap(req, "tags")
	f, err := store.CreateFilter(detectorID, name, desc, action, rank, criteria, tags)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"name": f.Name})
}

func handleGetFilter(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	name := getStr(req, "filterName")
	if name == "" {
		name = ctx.Params["filterName"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	f, err := store.GetFilter(detectorID, name)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(filterMap(f))
}

func handleUpdateFilter(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	name := getStr(req, "filterName")
	if name == "" {
		name = ctx.Params["filterName"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	action := getStr(req, "action")
	desc := getStr(req, "description")
	rank := getInt(req, "rank")
	criteria := getMap(req, "findingCriteria")
	f, err := store.UpdateFilter(detectorID, name, desc, action, rank, criteria)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"name": f.Name})
}

func handleDeleteFilter(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	name := getStr(req, "filterName")
	if name == "" {
		name = ctx.Params["filterName"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	if err := store.DeleteFilter(detectorID, name); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListFilters(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	names, err := store.ListFilters(detectorID)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"filterNames": names})
}

// ── IPSet handlers ──────────────────────────────────────────────────────────

func handleCreateIPSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	name := getStr(req, "name")
	format := getStr(req, "format")
	location := getStr(req, "location")
	if name == "" || format == "" || location == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"name, format, and location are required.", http.StatusBadRequest))
	}
	tags := getStringMap(req, "tags")
	set, err := store.CreateIPSet(detectorID, name, format, location, getBool(req, "activate"), getStr(req, "expectedBucketOwner"), tags)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"ipSetId": set.IPSetID})
}

func handleGetIPSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	id := getStr(req, "ipSetId")
	if id == "" {
		id = ctx.Params["ipSetId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	set, err := store.GetIPSet(detectorID, id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(ipSetMap(set))
}

func handleUpdateIPSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	id := getStr(req, "ipSetId")
	if id == "" {
		id = ctx.Params["ipSetId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	if err := store.UpdateIPSet(detectorID, id, getStr(req, "name"), getStr(req, "location"), getBoolPtr(req, "activate")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDeleteIPSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	id := getStr(req, "ipSetId")
	if id == "" {
		id = ctx.Params["ipSetId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	if err := store.DeleteIPSet(detectorID, id); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListIPSets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	ids, err := store.ListIPSets(detectorID)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"ipSetIds": ids})
}

// ── ThreatIntelSet handlers ─────────────────────────────────────────────────

func handleCreateThreatIntelSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	name := getStr(req, "name")
	format := getStr(req, "format")
	location := getStr(req, "location")
	if name == "" || format == "" || location == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"name, format, and location are required.", http.StatusBadRequest))
	}
	tags := getStringMap(req, "tags")
	set, err := store.CreateThreatIntelSet(detectorID, name, format, location, getBool(req, "activate"), getStr(req, "expectedBucketOwner"), tags)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"threatIntelSetId": set.ThreatIntelSetID})
}

func handleGetThreatIntelSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	id := getStr(req, "threatIntelSetId")
	if id == "" {
		id = ctx.Params["threatIntelSetId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	set, err := store.GetThreatIntelSet(detectorID, id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(threatIntelSetMap(set))
}

func handleUpdateThreatIntelSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	id := getStr(req, "threatIntelSetId")
	if id == "" {
		id = ctx.Params["threatIntelSetId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	if err := store.UpdateThreatIntelSet(detectorID, id, getStr(req, "name"), getStr(req, "location"), getBoolPtr(req, "activate")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDeleteThreatIntelSet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	id := getStr(req, "threatIntelSetId")
	if id == "" {
		id = ctx.Params["threatIntelSetId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	if err := store.DeleteThreatIntelSet(detectorID, id); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListThreatIntelSets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	ids, err := store.ListThreatIntelSets(detectorID)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"threatIntelSetIds": ids})
}

// ── Threat / Trusted Entity Set handlers ────────────────────────────────────

func entitySetCreate(ctx *service.RequestContext, store *Store, kind string) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	name := getStr(req, "name")
	format := getStr(req, "format")
	location := getStr(req, "location")
	if name == "" || format == "" || location == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"name, format, and location are required.", http.StatusBadRequest))
	}
	tags := getStringMap(req, "tags")
	set, err := store.CreateEntitySet(kind, detectorID, name, format, location, getBool(req, "activate"), getStr(req, "expectedBucketOwner"), tags)
	if err != nil {
		return jsonErr(err)
	}
	if kind == "threat" {
		return jsonOK(map[string]any{"threatEntitySetId": set.EntitySetID})
	}
	return jsonOK(map[string]any{"trustedEntitySetId": set.EntitySetID})
}

func entitySetGet(ctx *service.RequestContext, store *Store, kind, idKey string) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	id := getStr(req, idKey)
	if id == "" {
		id = ctx.Params[idKey]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	set, err := store.GetEntitySet(kind, detectorID, id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(entitySetMap(set))
}

func entitySetUpdate(ctx *service.RequestContext, store *Store, kind, idKey string) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	id := getStr(req, idKey)
	if id == "" {
		id = ctx.Params[idKey]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	if err := store.UpdateEntitySet(kind, detectorID, id, getStr(req, "name"), getStr(req, "location"), getBoolPtr(req, "activate")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func entitySetDelete(ctx *service.RequestContext, store *Store, kind, idKey string) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	id := getStr(req, idKey)
	if id == "" {
		id = ctx.Params[idKey]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	if err := store.DeleteEntitySet(kind, detectorID, id); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func entitySetList(ctx *service.RequestContext, store *Store, kind, listKey string) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	ids, err := store.ListEntitySets(kind, detectorID)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{listKey: ids})
}

func handleCreateThreatEntitySet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return entitySetCreate(ctx, store, "threat")
}

func handleGetThreatEntitySet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return entitySetGet(ctx, store, "threat", "threatEntitySetId")
}

func handleUpdateThreatEntitySet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return entitySetUpdate(ctx, store, "threat", "threatEntitySetId")
}

func handleDeleteThreatEntitySet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return entitySetDelete(ctx, store, "threat", "threatEntitySetId")
}

func handleListThreatEntitySets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return entitySetList(ctx, store, "threat", "threatEntitySetIds")
}

func handleCreateTrustedEntitySet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return entitySetCreate(ctx, store, "trusted")
}

func handleGetTrustedEntitySet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return entitySetGet(ctx, store, "trusted", "trustedEntitySetId")
}

func handleUpdateTrustedEntitySet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return entitySetUpdate(ctx, store, "trusted", "trustedEntitySetId")
}

func handleDeleteTrustedEntitySet(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return entitySetDelete(ctx, store, "trusted", "trustedEntitySetId")
}

func handleListTrustedEntitySets(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return entitySetList(ctx, store, "trusted", "trustedEntitySetIds")
}

// ── Members handlers ────────────────────────────────────────────────────────

func unprocessedAccountsFromList(ids []string, reason string) []map[string]any {
	out := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, map[string]any{
			"accountId": id,
			"result":    reason,
		})
	}
	return out
}

func handleCreateMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	details := getMapList(req, "accountDetails")
	if len(details) == 0 {
		return jsonErr(service.NewAWSError("BadRequestException",
			"accountDetails is required.", http.StatusBadRequest))
	}
	unprocessed := make([]map[string]any, 0)
	for _, d := range details {
		accountID := getStr(d, "accountId")
		email := getStr(d, "email")
		if _, err := store.CreateMember(detectorID, accountID, email); err != nil {
			unprocessed = append(unprocessed, map[string]any{
				"accountId": accountID,
				"result":    err.Message,
			})
		}
	}
	return jsonOK(map[string]any{"unprocessedAccounts": unprocessed})
}

func handleGetMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	ids := getStrList(req, "accountIds")
	found, missing, err := store.GetMembers(detectorID, ids)
	if err != nil {
		return jsonErr(err)
	}
	out := make([]map[string]any, 0, len(found))
	for _, m := range found {
		out = append(out, memberMap(m))
	}
	return jsonOK(map[string]any{
		"members":             out,
		"unprocessedAccounts": unprocessedAccountsFromList(missing, "Account not found"),
	})
}

func handleListMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	ms, err := store.ListMembers(detectorID)
	if err != nil {
		return jsonErr(err)
	}
	out := make([]map[string]any, 0, len(ms))
	for _, m := range ms {
		out = append(out, memberMap(m))
	}
	return jsonOK(map[string]any{"members": out})
}

func handleDeleteMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	ids := getStrList(req, "accountIds")
	if err := store.DeleteMembers(detectorID, ids); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"unprocessedAccounts": []map[string]any{}})
}

func handleDisassociateMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	ids := getStrList(req, "accountIds")
	if err := store.SetMemberStatus(detectorID, ids, "REMOVED"); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"unprocessedAccounts": []map[string]any{}})
}

func handleInviteMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	ids := getStrList(req, "accountIds")
	if err := store.InviteMembers(detectorID, ids); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"unprocessedAccounts": []map[string]any{}})
}

func handleStartMonitoringMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	ids := getStrList(req, "accountIds")
	if err := store.SetMemberStatus(detectorID, ids, "ENABLED"); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"unprocessedAccounts": []map[string]any{}})
}

func handleStopMonitoringMembers(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	ids := getStrList(req, "accountIds")
	if err := store.SetMemberStatus(detectorID, ids, "DISABLED"); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"unprocessedAccounts": []map[string]any{}})
}

func handleUpdateMemberDetectors(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	ids := getStrList(req, "accountIds")
	if err := store.UpdateMemberDetectors(detectorID, ids, getMap(req, "dataSources")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"unprocessedAccounts": []map[string]any{}})
}

func handleGetMemberDetectors(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	ids := getStrList(req, "accountIds")
	configs, missing, err := store.MemberDetectors(detectorID, ids)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"members":             configs,
		"unprocessedAccounts": unprocessedAccountsFromList(missing, "Account not found"),
	})
}

// ── Administrator / Master handlers ─────────────────────────────────────────

func handleGetAdministratorAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	master := store.MasterAccount()
	if master == nil {
		return jsonOK(map[string]any{
			"administrator": map[string]any{},
		})
	}
	return jsonOK(map[string]any{
		"administrator": map[string]any{
			"accountId":          master.AdministratorID,
			"invitationId":       master.AccountID,
			"relationshipStatus": master.RelationshipStatus,
			"invitedAt":          rfc3339(master.InvitedAt),
		},
	})
}

func handleGetMasterAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	master := store.MasterAccount()
	if master == nil {
		return jsonOK(map[string]any{
			"master": map[string]any{},
		})
	}
	return jsonOK(map[string]any{
		"master": map[string]any{
			"accountId":          master.MasterID,
			"invitationId":       master.AccountID,
			"relationshipStatus": master.RelationshipStatus,
			"invitedAt":          rfc3339(master.InvitedAt),
		},
	})
}

func handleAcceptAdministratorInvitation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	adminID := getStr(req, "administratorId")
	invitationID := getStr(req, "invitationId")
	if adminID == "" || invitationID == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"administratorId and invitationId are required.", http.StatusBadRequest))
	}
	if err := store.AcceptInvitation(detectorID, invitationID, adminID); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleAcceptInvitation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	masterID := getStr(req, "masterId")
	invitationID := getStr(req, "invitationId")
	if masterID == "" || invitationID == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"masterId and invitationId are required.", http.StatusBadRequest))
	}
	if err := store.AcceptInvitation(detectorID, invitationID, masterID); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDisassociateFromAdministratorAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	store.DisassociateMaster()
	return jsonOK(map[string]any{})
}

func handleDisassociateFromMasterAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return handleDisassociateFromAdministratorAccount(ctx, store)
}

// ── Invitations handlers ────────────────────────────────────────────────────

func handleListInvitations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	invs := store.ListInvitations()
	out := make([]map[string]any, 0, len(invs))
	for _, inv := range invs {
		out = append(out, invitationMap(inv))
	}
	return jsonOK(map[string]any{"invitations": out})
}

func handleGetInvitationsCount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	count := len(store.ListInvitations())
	return jsonOK(map[string]any{"invitationsCount": count})
}

func handleDeclineInvitations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	ids := getStrList(req, "accountIds")
	store.DeleteInvitations(ids)
	return jsonOK(map[string]any{"unprocessedAccounts": []map[string]any{}})
}

func handleDeleteInvitations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	ids := getStrList(req, "accountIds")
	store.DeleteInvitations(ids)
	return jsonOK(map[string]any{"unprocessedAccounts": []map[string]any{}})
}

// ── Findings handlers ───────────────────────────────────────────────────────

func handleCreateSampleFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	if err := store.CreateSampleFindings(detectorID, getStrList(req, "findingTypes")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleGetFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	ids := getStrList(req, "findingIds")
	found, err := store.GetFindings(detectorID, ids)
	if err != nil {
		return jsonErr(err)
	}
	out := make([]map[string]any, 0, len(found))
	for _, f := range found {
		out = append(out, findingMap(f))
	}
	return jsonOK(map[string]any{"findings": out})
}

func handleListFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	ids, err := store.ListFindings(detectorID)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"findingIds": ids})
}

func handleArchiveFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	if err := store.SetFindingsArchived(detectorID, getStrList(req, "findingIds"), true); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleUnarchiveFindings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	if err := store.SetFindingsArchived(detectorID, getStrList(req, "findingIds"), false); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleUpdateFindingsFeedback(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	feedback := getStr(req, "feedback")
	if feedback == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"feedback is required.", http.StatusBadRequest))
	}
	if err := store.SetFindingsFeedback(detectorID, getStrList(req, "findingIds"), feedback); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleGetFindingsStatistics(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	bySeverity, err := store.FindingsStatistics(detectorID)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"findingStatistics": map[string]any{
			"countBySeverity": bySeverity,
		},
	})
}

// ── Organization handlers ───────────────────────────────────────────────────

func handleDescribeOrganizationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	cfg, err := store.OrgConfig(detectorID)
	if err != nil {
		return jsonErr(err)
	}
	out := map[string]any{
		"autoEnable":                cfg.AutoEnable,
		"memberAccountLimitReached": cfg.MemberAccountLimitReached,
	}
	if cfg.AutoEnableOrganizationMembers != "" {
		out["autoEnableOrganizationMembers"] = cfg.AutoEnableOrganizationMembers
	}
	if cfg.DataSources != nil {
		out["dataSources"] = cfg.DataSources
	}
	if cfg.Features != nil {
		out["features"] = cfg.Features
	}
	return jsonOK(out)
}

func handleUpdateOrganizationConfiguration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	if err := store.UpdateOrgConfig(
		detectorID,
		getBool(req, "autoEnable"),
		getStr(req, "autoEnableOrganizationMembers"),
		getMap(req, "dataSources"),
		getMapList(req, "features"),
	); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleEnableOrganizationAdminAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.EnableOrgAdminAccount(getStr(req, "adminAccountId")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDisableOrganizationAdminAccount(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.DisableOrgAdminAccount(getStr(req, "adminAccountId")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListOrganizationAdminAccounts(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	accounts := store.OrgAdminAccounts()
	out := make([]map[string]any, 0, len(accounts))
	for id, status := range accounts {
		out = append(out, map[string]any{
			"adminAccountId": id,
			"adminStatus":    status,
		})
	}
	return jsonOK(map[string]any{"adminAccounts": out})
}

func handleGetOrganizationStatistics(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"organizationDetails": map[string]any{
			"updatedAt": rfc3339(time.Now().UTC()),
			"organizationStatistics": map[string]any{
				"totalAccountsCount":    len(store.OrgAdminAccounts()),
				"memberAccountsCount":   0,
				"activeAccountsCount":   0,
				"enabledAccountsCount":  0,
			},
		},
	})
}

// ── Publishing destinations handlers ────────────────────────────────────────

func handleCreatePublishingDestination(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	d, err := store.CreatePublishingDestination(detectorID, getStr(req, "destinationType"), getMap(req, "destinationProperties"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"destinationId": d.DestinationID})
}

func handleDescribePublishingDestination(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	id := getStr(req, "destinationId")
	if id == "" {
		id = ctx.Params["destinationId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	d, err := store.GetPublishingDestination(detectorID, id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(publishingDestinationMap(d))
}

func handleUpdatePublishingDestination(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	id := getStr(req, "destinationId")
	if id == "" {
		id = ctx.Params["destinationId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	if err := store.UpdatePublishingDestination(detectorID, id, getMap(req, "destinationProperties")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDeletePublishingDestination(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	id := getStr(req, "destinationId")
	if id == "" {
		id = ctx.Params["destinationId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	if err := store.DeletePublishingDestination(detectorID, id); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListPublishingDestinations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	dests, err := store.ListPublishingDestinations(detectorID)
	if err != nil {
		return jsonErr(err)
	}
	out := make([]map[string]any, 0, len(dests))
	for _, d := range dests {
		out = append(out, map[string]any{
			"destinationId":   d.DestinationID,
			"destinationType": d.DestinationType,
			"status":          d.Status,
		})
	}
	return jsonOK(map[string]any{"destinations": out})
}

// ── Malware protection handlers ─────────────────────────────────────────────

func handleCreateMalwareProtectionPlan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	role := getStr(req, "role")
	if role == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"role is required.", http.StatusBadRequest))
	}
	plan := store.CreateMalwareProtectionPlan(
		role,
		getMap(req, "protectedResource"),
		getMap(req, "actions"),
		getStringMap(req, "tags"),
	)
	return jsonOK(map[string]any{"malwareProtectionPlanId": plan.PlanID})
}

func handleGetMalwareProtectionPlan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "malwareProtectionPlanId")
	if id == "" {
		id = ctx.Params["malwareProtectionPlanId"]
	}
	if id == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"malwareProtectionPlanId is required.", http.StatusBadRequest))
	}
	plan, err := store.GetMalwareProtectionPlan(id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(malwareProtectionPlanMap(plan))
}

func handleListMalwareProtectionPlans(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	plans := store.ListMalwareProtectionPlans()
	out := make([]map[string]any, 0, len(plans))
	for _, p := range plans {
		out = append(out, map[string]any{
			"malwareProtectionPlanId": p.PlanID,
		})
	}
	return jsonOK(map[string]any{"malwareProtectionPlans": out})
}

func handleUpdateMalwareProtectionPlan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "malwareProtectionPlanId")
	if id == "" {
		id = ctx.Params["malwareProtectionPlanId"]
	}
	if id == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"malwareProtectionPlanId is required.", http.StatusBadRequest))
	}
	if err := store.UpdateMalwareProtectionPlan(
		id,
		getStr(req, "role"),
		getMap(req, "protectedResource"),
		getMap(req, "actions"),
	); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDeleteMalwareProtectionPlan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "malwareProtectionPlanId")
	if id == "" {
		id = ctx.Params["malwareProtectionPlanId"]
	}
	if id == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"malwareProtectionPlanId is required.", http.StatusBadRequest))
	}
	if err := store.DeleteMalwareProtectionPlan(id); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── Malware scan settings & scans ───────────────────────────────────────────

func handleGetMalwareScanSettings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	cfg, err := store.MalwareScanSettings(detectorID)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"ebsSnapshotPreservation": cfg.EbsSnapshotPreservation,
		"scanResourceCriteria":    cfg.ScanResourceCriteria,
	})
}

func handleUpdateMalwareScanSettings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	if err := store.UpdateMalwareScanSettings(
		detectorID,
		getStr(req, "ebsSnapshotPreservation"),
		getMap(req, "scanResourceCriteria"),
	); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleStartMalwareScan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	scan, err := store.StartMalwareScan(getStr(req, "resourceArn"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"scanId": scan.ScanID})
}

func handleDescribeMalwareScans(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	scans := store.ListMalwareScans()
	out := make([]map[string]any, 0, len(scans))
	for _, s := range scans {
		out = append(out, malwareScanMap(s))
	}
	return jsonOK(map[string]any{"scans": out})
}

func handleGetMalwareScan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	scanID := getStr(req, "scanId")
	if scanID == "" {
		scanID = ctx.Params["scanId"]
	}
	for _, scan := range store.ListMalwareScans() {
		if scan.ScanID == scanID {
			return jsonOK(malwareScanMap(scan))
		}
	}
	return jsonErr(service.NewAWSError("ResourceNotFoundException",
		"Malware scan not found: "+scanID, http.StatusBadRequest))
}

func handleListMalwareScans(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	scans := store.ListMalwareScans()
	out := make([]map[string]any, 0, len(scans))
	for _, s := range scans {
		out = append(out, malwareScanMap(s))
	}
	return jsonOK(map[string]any{"scans": out})
}

func handleSendObjectMalwareScan(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	resourceArn := getStr(req, "resourceArn")
	if resourceArn == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"resourceArn is required.", http.StatusBadRequest))
	}
	scan, err := store.StartMalwareScan(resourceArn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"scanId": scan.ScanID})
}

// ── Coverage / usage / free trial handlers ──────────────────────────────────

func handleListCoverage(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	members, err := store.ListMembers(detectorID)
	if err != nil {
		return jsonErr(err)
	}
	resources := make([]map[string]any, 0, len(members))
	for _, m := range members {
		resources = append(resources, map[string]any{
			"detectorId":   detectorID,
			"resourceId":   m.AccountID,
			"resourceType": "EKS",
			"coverageStatus": "HEALTHY",
			"updatedAt":    rfc3339(m.UpdatedAt),
		})
	}
	return jsonOK(map[string]any{"resources": resources})
}

func handleGetCoverageStatistics(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	members, err := store.ListMembers(detectorID)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"coverageStatistics": map[string]any{
			"countByResourceType": map[string]any{
				"EKS": len(members),
			},
		},
	})
}

func handleGetUsageStatistics(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	if _, err := store.GetDetector(detectorID); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"usageStatistics": map[string]any{
			"sumByAccount": []map[string]any{
				{
					"accountId": store.AccountID(),
					"total": map[string]any{
						"amount": "0.0",
						"unit":   "USD",
					},
				},
			},
		},
	})
}

func handleGetRemainingFreeTrialDays(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	detectorID := getStr(req, "detectorId")
	if detectorID == "" {
		detectorID = ctx.Params["detectorId"]
	}
	if err := validDetectorID(detectorID); err != nil {
		return jsonErr(err)
	}
	ids := getStrList(req, "accountIds")
	if len(ids) == 0 {
		ids = []string{store.AccountID()}
	}
	accounts := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		accounts = append(accounts, map[string]any{
			"accountId": id,
			"features": []map[string]any{
				{"name": "RUNTIME_MONITORING", "freeTrialDaysRemaining": 30},
			},
		})
	}
	return jsonOK(map[string]any{
		"accounts":            accounts,
		"unprocessedAccounts": []map[string]any{},
	})
}

// ── Tag handlers ────────────────────────────────────────────────────────────

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "resourceArn")
	if arn == "" {
		arn = ctx.Params["resourceArn"]
	}
	if arn == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"resourceArn is required.", http.StatusBadRequest))
	}
	tags := getStringMap(req, "tags")
	store.TagResource(arn, tags)
	return jsonOK(map[string]any{})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "resourceArn")
	if arn == "" {
		arn = ctx.Params["resourceArn"]
	}
	if arn == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"resourceArn is required.", http.StatusBadRequest))
	}
	store.UntagResource(arn, getStrList(req, "tagKeys"))
	return jsonOK(map[string]any{})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "resourceArn")
	if arn == "" {
		arn = ctx.Params["resourceArn"]
	}
	if arn == "" {
		return jsonErr(service.NewAWSError("BadRequestException",
			"resourceArn is required.", http.StatusBadRequest))
	}
	tags := store.ListTags(arn)
	return jsonOK(map[string]any{"tags": tags})
}
