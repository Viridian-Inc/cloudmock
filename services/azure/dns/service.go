package dns

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/azurearm"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

const dnsAPIVersion = "2018-05-01"

// DNSService implements first-slice Azure DNS control-plane APIs.
type DNSService struct {
	mu         sync.RWMutex
	zones      map[string]Zone
	recordSets map[string]RecordSet
}

func New() *DNSService {
	return &DNSService{
		zones:      make(map[string]Zone),
		recordSets: make(map[string]RecordSet),
	}
}

func (s *DNSService) Name() string { return "Microsoft.Network.Dns" }

func (s *DNSService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdateZone", Method: http.MethodPut, IAMAction: "azure:Microsoft.Network/dnsZones/write"},
		{Name: "GetZone", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/dnsZones/read"},
		{Name: "ListZones", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/dnsZones/read"},
		{Name: "DeleteZone", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Network/dnsZones/delete"},
		{Name: "CreateOrUpdateRecordSet", Method: http.MethodPut, IAMAction: "azure:Microsoft.Network/dnsZones/recordsets/write"},
		{Name: "GetRecordSet", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/dnsZones/recordsets/read"},
		{Name: "ListRecordSets", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/dnsZones/recordsets/read"},
		{Name: "DeleteRecordSet", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Network/dnsZones/recordsets/delete"},
	}
}

func (s *DNSService) HealthCheck() error { return nil }

func (s *DNSService) ServiceKeys() []routing.ServiceKey {
	return []routing.ServiceKey{
		{Provider: routing.ProviderAzure, Service: "Microsoft.Network/dnsZones", APIVersion: dnsAPIVersion},
	}
}

func (s *DNSService) SupportsTemplateResource(resource map[string]any) bool {
	resourceType := stringValue(resource["type"])
	if strings.EqualFold(resourceType, "Microsoft.Network/dnsZones") {
		return true
	}
	recordType, ok := dnsRecordTypeFromResourceType(resourceType)
	return ok && recordType != ""
}

func (s *DNSService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported DNS template resource type %q", stringValue(resource["type"]))
	}
	resourceType := stringValue(resource["type"])
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("DNS template resource is missing name")
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

	var resp *service.Response
	switch {
	case strings.EqualFold(resourceType, "Microsoft.Network/dnsZones"):
		resp, err = s.createOrUpdateZone(subscriptionID, resourceGroup, name, data)
	default:
		recordType, ok := dnsRecordTypeFromResourceType(resourceType)
		if !ok {
			return nil, fmt.Errorf("unsupported DNS template resource type %q", resourceType)
		}
		zoneName, recordName, ok := splitNestedName(name)
		if !ok {
			return nil, fmt.Errorf("DNS record set template resource name must be {zone}/{recordSet}")
		}
		resp, err = s.createOrUpdateRecordSet(subscriptionID, resourceGroup, zoneName, recordType, recordName, data)
	}
	if err != nil {
		return nil, err
	}
	var out any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *DNSService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}

	route, ok := parseRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok || !strings.EqualFold(route.ResourceType, "dnsZones") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The DNS route is not implemented.")
	}
	if route.RecordCollection {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listRecordSets(route.SubscriptionID, route.ResourceGroup, route.ZoneName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if route.RecordType != "" {
		return s.handleRecordSetRequest(ctx, route)
	}
	return s.handleZoneRequest(ctx, route)
}

func (s *DNSService) handleZoneRequest(ctx *service.RequestContext, route dnsRoute) (*service.Response, error) {
	if route.ZoneName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listZones(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateZone(route.SubscriptionID, route.ResourceGroup, route.ZoneName, ctx.Body)
	case http.MethodGet:
		return s.getZone(route.SubscriptionID, route.ResourceGroup, route.ZoneName)
	case http.MethodDelete:
		return s.deleteZone(route.SubscriptionID, route.ResourceGroup, route.ZoneName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *DNSService) handleRecordSetRequest(ctx *service.RequestContext, route dnsRoute) (*service.Response, error) {
	if route.RecordName == "" {
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateRecordSet(route.SubscriptionID, route.ResourceGroup, route.ZoneName, route.RecordType, route.RecordName, ctx.Body)
	case http.MethodGet:
		return s.getRecordSet(route.SubscriptionID, route.ResourceGroup, route.ZoneName, route.RecordType, route.RecordName)
	case http.MethodDelete:
		return s.deleteRecordSet(route.SubscriptionID, route.ResourceGroup, route.ZoneName, route.RecordType, route.RecordName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *DNSService) createOrUpdateZone(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		Tags       map[string]any `json:"tags"`
		Properties map[string]any `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Location == "" {
		input.Location = "global"
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}
	input.Properties["provisioningState"] = "Succeeded"
	if _, ok := input.Properties["zoneType"]; !ok {
		input.Properties["zoneType"] = "Public"
	}

	zone := Zone{
		ID:         zoneID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.Network/dnsZones",
		Location:   input.Location,
		Tags:       stringifyTags(input.Tags),
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := zoneKey(subscriptionID, resourceGroup, name)
	_, existed := s.zones[key]
	s.zones[key] = zone
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, zone)
}

func (s *DNSService) getZone(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	zone, ok := s.zones[zoneKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("DNS zone %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, zone)
}

func (s *DNSService) listZones(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"

	s.mu.RLock()
	values := make([]Zone, 0)
	for key, zone := range s.zones {
		if strings.HasPrefix(key, prefix) {
			values = append(values, zone)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *DNSService) deleteZone(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := zoneKey(subscriptionID, resourceGroup, name)
	if _, ok := s.zones[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("DNS zone %q could not be found.", name))
	}
	delete(s.zones, key)
	childPrefix := key + "/"
	for recordSetKey := range s.recordSets {
		if strings.HasPrefix(recordSetKey, childPrefix) {
			delete(s.recordSets, recordSetKey)
		}
	}
	return azurearm.JSONResponse(http.StatusOK, map[string]string{"status": "Succeeded"})
}

func (s *DNSService) createOrUpdateRecordSet(subscriptionID, resourceGroup, zoneName, recordType, recordName string, body []byte) (*service.Response, error) {
	var input struct {
		Properties map[string]any `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}
	input.Properties["provisioningState"] = "Succeeded"

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.zones[zoneKey(subscriptionID, resourceGroup, zoneName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("DNS zone %q could not be found.", zoneName))
	}

	recordSet := RecordSet{
		ID:         recordSetID(subscriptionID, resourceGroup, zoneName, recordType, recordName),
		Name:       zoneName + "/" + recordName,
		Type:       "Microsoft.Network/dnsZones/" + normalizeRecordType(recordType),
		Properties: input.Properties,
	}
	key := recordSetKey(subscriptionID, resourceGroup, zoneName, recordType, recordName)
	_, existed := s.recordSets[key]
	s.recordSets[key] = recordSet

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, recordSet)
}

func (s *DNSService) getRecordSet(subscriptionID, resourceGroup, zoneName, recordType, recordName string) (*service.Response, error) {
	s.mu.RLock()
	recordSet, ok := s.recordSets[recordSetKey(subscriptionID, resourceGroup, zoneName, recordType, recordName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("DNS record set %q could not be found.", recordName))
	}
	return azurearm.JSONResponse(http.StatusOK, recordSet)
}

func (s *DNSService) listRecordSets(subscriptionID, resourceGroup, zoneName string) (*service.Response, error) {
	parentKey := zoneKey(subscriptionID, resourceGroup, zoneName)

	s.mu.RLock()
	_, zoneExists := s.zones[parentKey]
	values := make([]RecordSet, 0)
	prefix := parentKey + "/"
	for key, recordSet := range s.recordSets {
		if strings.HasPrefix(key, prefix) {
			values = append(values, recordSet)
		}
	}
	s.mu.RUnlock()

	if !zoneExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("DNS zone %q could not be found.", zoneName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *DNSService) deleteRecordSet(subscriptionID, resourceGroup, zoneName, recordType, recordName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := recordSetKey(subscriptionID, resourceGroup, zoneName, recordType, recordName)
	if _, ok := s.recordSets[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("DNS record set %q could not be found.", recordName))
	}
	delete(s.recordSets, key)
	return azurearm.JSONResponse(http.StatusOK, map[string]string{"status": "Succeeded"})
}

type dnsRoute struct {
	SubscriptionID   string
	ResourceGroup    string
	ResourceType     string
	ZoneName         string
	RecordCollection bool
	RecordType       string
	RecordName       string
}

func parseRoute(escapedPath string) (dnsRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) < 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.Network") {
		return dnsRoute{}, false
	}
	route := dnsRoute{
		SubscriptionID: parts[1],
		ResourceGroup:  parts[3],
		ResourceType:   parts[6],
	}
	switch len(parts) {
	case 7:
		return route, true
	case 8:
		route.ZoneName = parts[7]
		return route, true
	case 9:
		route.ZoneName = parts[7]
		if strings.EqualFold(parts[8], "recordsets") {
			route.RecordCollection = true
			return route, true
		}
		route.RecordType = parts[8]
		return route, true
	case 10:
		route.ZoneName = parts[7]
		route.RecordType = parts[8]
		route.RecordName = parts[9]
		return route, true
	default:
		return dnsRoute{}, false
	}
}

func zoneID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/dnsZones/" + name
}

func recordSetID(subscriptionID, resourceGroup, zoneName, recordType, recordName string) string {
	return zoneID(subscriptionID, resourceGroup, zoneName) + "/" + normalizeRecordType(recordType) + "/" + recordName
}

func zoneKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func recordSetKey(subscriptionID, resourceGroup, zoneName, recordType, recordName string) string {
	return zoneKey(subscriptionID, resourceGroup, zoneName) + "/" + strings.ToLower(recordType) + "/" + strings.ToLower(recordName)
}

func normalizeRecordType(recordType string) string {
	switch strings.ToLower(recordType) {
	case "a":
		return "A"
	case "aaaa":
		return "AAAA"
	case "caa":
		return "CAA"
	case "cname":
		return "CNAME"
	case "mx":
		return "MX"
	case "ns":
		return "NS"
	case "ptr":
		return "PTR"
	case "soa":
		return "SOA"
	case "srv":
		return "SRV"
	case "txt":
		return "TXT"
	default:
		return recordType
	}
}

func dnsRecordTypeFromResourceType(resourceType string) (string, bool) {
	prefix := "microsoft.network/dnszones/"
	lower := strings.ToLower(resourceType)
	if !strings.HasPrefix(lower, prefix) || len(resourceType) <= len(prefix) {
		return "", false
	}
	return normalizeRecordType(resourceType[len(prefix):]), true
}

func splitNestedName(name string) (string, string, bool) {
	parts := strings.Split(name, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func splitPath(escapedPath string) []string {
	trimmed := strings.Trim(escapedPath, "/")
	if trimmed == "" {
		return nil
	}
	rawParts := strings.Split(trimmed, "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		if part == "" {
			continue
		}
		decoded, err := url.PathUnescape(part)
		if err != nil {
			decoded = part
		}
		parts = append(parts, decoded)
	}
	return parts
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
