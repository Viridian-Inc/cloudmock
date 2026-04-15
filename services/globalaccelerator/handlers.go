package globalaccelerator

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
		return service.NewAWSError("InvalidParameterException",
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

func getIntPtr(m map[string]any, key string) *int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			i := int(n)
			return &i
		case int:
			return &n
		}
	}
	return nil
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

func parseTagList(m map[string]any, key string) map[string]string {
	out := make(map[string]string)
	for _, t := range getMapList(m, key) {
		k := getStr(t, "Key")
		if k == "" {
			k = getStr(t, "key")
		}
		v := getStr(t, "Value")
		if v == "" {
			v = getStr(t, "value")
		}
		if k != "" {
			out[k] = v
		}
	}
	return out
}

func rfc3339(t time.Time) string { return t.Format(time.RFC3339) }

// ── Response shapers ─────────────────────────────────────────────────────────

func acceleratorToMap(a *StoredAccelerator) map[string]any {
	return map[string]any{
		"AcceleratorArn":   a.Arn,
		"Name":             a.Name,
		"IpAddressType":    a.IpAddressType,
		"Enabled":          a.Enabled,
		"DnsName":          a.DnsName,
		"DualStackDnsName": a.DualStackDnsName,
		"Status":           a.Status,
		"IpSets":           a.IpSets,
		"Events":           a.Events,
		"CreatedTime":      rfc3339(a.CreatedTime),
		"LastModifiedTime": rfc3339(a.LastModifiedTime),
	}
}

func customAcceleratorToMap(a *StoredCustomRoutingAccelerator) map[string]any {
	return map[string]any{
		"AcceleratorArn":   a.Arn,
		"Name":             a.Name,
		"IpAddressType":    a.IpAddressType,
		"Enabled":          a.Enabled,
		"DnsName":          a.DnsName,
		"Status":           a.Status,
		"IpSets":           a.IpSets,
		"CreatedTime":      rfc3339(a.CreatedTime),
		"LastModifiedTime": rfc3339(a.LastModifiedTime),
	}
}

func acceleratorAttributesToMap(a *StoredAcceleratorAttributes) map[string]any {
	return map[string]any{
		"FlowLogsEnabled":  a.FlowLogsEnabled,
		"FlowLogsS3Bucket": a.FlowLogsS3Bucket,
		"FlowLogsS3Prefix": a.FlowLogsS3Prefix,
	}
}

func customAcceleratorAttributesToMap(a *StoredCustomRoutingAcceleratorAttributes) map[string]any {
	return map[string]any{
		"FlowLogsEnabled":  a.FlowLogsEnabled,
		"FlowLogsS3Bucket": a.FlowLogsS3Bucket,
		"FlowLogsS3Prefix": a.FlowLogsS3Prefix,
	}
}

func listenerToMap(l *StoredListener) map[string]any {
	return map[string]any{
		"ListenerArn":    l.Arn,
		"Protocol":       l.Protocol,
		"ClientAffinity": l.ClientAffinity,
		"PortRanges":     l.PortRanges,
	}
}

func customListenerToMap(l *StoredCustomRoutingListener) map[string]any {
	return map[string]any{
		"ListenerArn": l.Arn,
		"PortRanges":  l.PortRanges,
	}
}

func endpointGroupToMap(g *StoredEndpointGroup) map[string]any {
	return map[string]any{
		"EndpointGroupArn":           g.Arn,
		"EndpointGroupRegion":        g.EndpointGroupRegion,
		"HealthCheckPort":            g.HealthCheckPort,
		"HealthCheckProtocol":        g.HealthCheckProtocol,
		"HealthCheckPath":            g.HealthCheckPath,
		"HealthCheckIntervalSeconds": g.HealthCheckIntervalSeconds,
		"ThresholdCount":             g.ThresholdCount,
		"TrafficDialPercentage":      g.TrafficDialPercentage,
		"PortOverrides":              g.PortOverrides,
		"EndpointDescriptions":       g.EndpointDescriptions,
	}
}

func customEndpointGroupToMap(g *StoredCustomRoutingEndpointGroup) map[string]any {
	return map[string]any{
		"EndpointGroupArn":        g.Arn,
		"EndpointGroupRegion":     g.EndpointGroupRegion,
		"DestinationDescriptions": g.DestinationDescriptions,
		"EndpointDescriptions":    g.EndpointDescriptions,
	}
}

func byoipCidrToMap(c *StoredByoipCidr) map[string]any {
	return map[string]any{
		"Cidr":   c.Cidr,
		"State":  c.State,
		"Events": c.Events,
	}
}

func attachmentToMap(a *StoredCrossAccountAttachment) map[string]any {
	return map[string]any{
		"AttachmentArn":    a.Arn,
		"Name":             a.Name,
		"Principals":       a.Principals,
		"Resources":        a.Resources,
		"CreatedTime":      rfc3339(a.CreatedTime),
		"LastModifiedTime": rfc3339(a.LastModifiedTime),
	}
}

// ── Accelerator handlers ─────────────────────────────────────────────────────

func handleCreateAccelerator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	enabled := true
	if v := getBoolPtr(req, "Enabled"); v != nil {
		enabled = *v
	}
	a := store.CreateAccelerator(name, getStr(req, "IpAddressType"), getStrList(req, "IpAddresses"), enabled)
	tags := parseTagList(req, "Tags")
	if len(tags) > 0 {
		store.TagResource(a.Arn, tags)
	}
	return jsonOK(map[string]any{"Accelerator": acceleratorToMap(a)})
}

func handleDescribeAccelerator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "AcceleratorArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("AcceleratorArn is required."))
	}
	a, err := store.GetAccelerator(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"Accelerator": acceleratorToMap(a)})
}

func handleUpdateAccelerator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "AcceleratorArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("AcceleratorArn is required."))
	}
	a, err := store.UpdateAccelerator(arn,
		getStrPtr(req, "Name"),
		getStrPtr(req, "IpAddressType"),
		getStrList(req, "IpAddresses"),
		getBoolPtr(req, "Enabled"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"Accelerator": acceleratorToMap(a)})
}

func handleDeleteAccelerator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "AcceleratorArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("AcceleratorArn is required."))
	}
	if err := store.DeleteAccelerator(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListAccelerators(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListAccelerators()
	out := make([]map[string]any, 0, len(list))
	for _, a := range list {
		out = append(out, acceleratorToMap(a))
	}
	return jsonOK(map[string]any{"Accelerators": out})
}

func handleDescribeAcceleratorAttributes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "AcceleratorArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("AcceleratorArn is required."))
	}
	attrs, err := store.GetAcceleratorAttributes(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"AcceleratorAttributes": acceleratorAttributesToMap(attrs)})
}

func handleUpdateAcceleratorAttributes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "AcceleratorArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("AcceleratorArn is required."))
	}
	attrs, err := store.UpdateAcceleratorAttributes(arn,
		getBoolPtr(req, "FlowLogsEnabled"),
		getStrPtr(req, "FlowLogsS3Bucket"),
		getStrPtr(req, "FlowLogsS3Prefix"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"AcceleratorAttributes": acceleratorAttributesToMap(attrs)})
}

// ── Custom Routing Accelerator handlers ──────────────────────────────────────

func handleCreateCustomRoutingAccelerator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	enabled := true
	if v := getBoolPtr(req, "Enabled"); v != nil {
		enabled = *v
	}
	a := store.CreateCustomRoutingAccelerator(name, getStr(req, "IpAddressType"), getStrList(req, "IpAddresses"), enabled)
	tags := parseTagList(req, "Tags")
	if len(tags) > 0 {
		store.TagResource(a.Arn, tags)
	}
	return jsonOK(map[string]any{"Accelerator": customAcceleratorToMap(a)})
}

func handleDescribeCustomRoutingAccelerator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "AcceleratorArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("AcceleratorArn is required."))
	}
	a, err := store.GetCustomRoutingAccelerator(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"Accelerator": customAcceleratorToMap(a)})
}

func handleUpdateCustomRoutingAccelerator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "AcceleratorArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("AcceleratorArn is required."))
	}
	a, err := store.UpdateCustomRoutingAccelerator(arn,
		getStrPtr(req, "Name"),
		getStrPtr(req, "IpAddressType"),
		getStrList(req, "IpAddresses"),
		getBoolPtr(req, "Enabled"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"Accelerator": customAcceleratorToMap(a)})
}

func handleDeleteCustomRoutingAccelerator(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "AcceleratorArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("AcceleratorArn is required."))
	}
	if err := store.DeleteCustomRoutingAccelerator(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListCustomRoutingAccelerators(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListCustomRoutingAccelerators()
	out := make([]map[string]any, 0, len(list))
	for _, a := range list {
		out = append(out, customAcceleratorToMap(a))
	}
	return jsonOK(map[string]any{"Accelerators": out})
}

func handleDescribeCustomRoutingAcceleratorAttributes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "AcceleratorArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("AcceleratorArn is required."))
	}
	attrs, err := store.GetCustomRoutingAcceleratorAttributes(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"AcceleratorAttributes": customAcceleratorAttributesToMap(attrs)})
}

func handleUpdateCustomRoutingAcceleratorAttributes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "AcceleratorArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("AcceleratorArn is required."))
	}
	attrs, err := store.UpdateCustomRoutingAcceleratorAttributes(arn,
		getBoolPtr(req, "FlowLogsEnabled"),
		getStrPtr(req, "FlowLogsS3Bucket"),
		getStrPtr(req, "FlowLogsS3Prefix"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"AcceleratorAttributes": customAcceleratorAttributesToMap(attrs)})
}

// ── Listener handlers ────────────────────────────────────────────────────────

func handleCreateListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	acceleratorArn := getStr(req, "AcceleratorArn")
	if acceleratorArn == "" {
		return jsonErr(service.ErrValidation("AcceleratorArn is required."))
	}
	protocol := getStr(req, "Protocol")
	if protocol == "" {
		return jsonErr(service.ErrValidation("Protocol is required."))
	}
	portRanges := getMapList(req, "PortRanges")
	if len(portRanges) == 0 {
		return jsonErr(service.ErrValidation("PortRanges is required."))
	}
	l, err := store.CreateListener(acceleratorArn, protocol, getStr(req, "ClientAffinity"), portRanges)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"Listener": listenerToMap(l)})
}

func handleDescribeListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ListenerArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ListenerArn is required."))
	}
	l, err := store.GetListener(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"Listener": listenerToMap(l)})
}

func handleUpdateListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ListenerArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ListenerArn is required."))
	}
	l, err := store.UpdateListener(arn,
		getStrPtr(req, "Protocol"),
		getStrPtr(req, "ClientAffinity"),
		getMapList(req, "PortRanges"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"Listener": listenerToMap(l)})
}

func handleDeleteListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ListenerArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ListenerArn is required."))
	}
	if err := store.DeleteListener(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListListeners(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "AcceleratorArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("AcceleratorArn is required."))
	}
	list := store.ListListeners(arn)
	out := make([]map[string]any, 0, len(list))
	for _, l := range list {
		out = append(out, listenerToMap(l))
	}
	return jsonOK(map[string]any{"Listeners": out})
}

// ── Custom Routing Listener handlers ─────────────────────────────────────────

func handleCreateCustomRoutingListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	acceleratorArn := getStr(req, "AcceleratorArn")
	if acceleratorArn == "" {
		return jsonErr(service.ErrValidation("AcceleratorArn is required."))
	}
	portRanges := getMapList(req, "PortRanges")
	if len(portRanges) == 0 {
		return jsonErr(service.ErrValidation("PortRanges is required."))
	}
	l, err := store.CreateCustomRoutingListener(acceleratorArn, portRanges)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"Listener": customListenerToMap(l)})
}

func handleDescribeCustomRoutingListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ListenerArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ListenerArn is required."))
	}
	l, err := store.GetCustomRoutingListener(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"Listener": customListenerToMap(l)})
}

func handleUpdateCustomRoutingListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ListenerArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ListenerArn is required."))
	}
	l, err := store.UpdateCustomRoutingListener(arn, getMapList(req, "PortRanges"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"Listener": customListenerToMap(l)})
}

func handleDeleteCustomRoutingListener(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ListenerArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ListenerArn is required."))
	}
	if err := store.DeleteCustomRoutingListener(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListCustomRoutingListeners(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "AcceleratorArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("AcceleratorArn is required."))
	}
	list := store.ListCustomRoutingListeners(arn)
	out := make([]map[string]any, 0, len(list))
	for _, l := range list {
		out = append(out, customListenerToMap(l))
	}
	return jsonOK(map[string]any{"Listeners": out})
}

// ── Endpoint Group handlers ──────────────────────────────────────────────────

func handleCreateEndpointGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	listenerArn := getStr(req, "ListenerArn")
	if listenerArn == "" {
		return jsonErr(service.ErrValidation("ListenerArn is required."))
	}
	region := getStr(req, "EndpointGroupRegion")
	if region == "" {
		return jsonErr(service.ErrValidation("EndpointGroupRegion is required."))
	}
	g, err := store.CreateEndpointGroup(
		listenerArn,
		region,
		getInt(req, "HealthCheckPort"),
		getStr(req, "HealthCheckProtocol"),
		getStr(req, "HealthCheckPath"),
		getInt(req, "HealthCheckIntervalSeconds"),
		getInt(req, "ThresholdCount"),
		getFloat(req, "TrafficDialPercentage"),
		getMapList(req, "PortOverrides"),
		getMapList(req, "EndpointConfigurations"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"EndpointGroup": endpointGroupToMap(g)})
}

func handleDescribeEndpointGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "EndpointGroupArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("EndpointGroupArn is required."))
	}
	g, err := store.GetEndpointGroup(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"EndpointGroup": endpointGroupToMap(g)})
}

func handleUpdateEndpointGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "EndpointGroupArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("EndpointGroupArn is required."))
	}
	g, err := store.UpdateEndpointGroup(arn,
		getIntPtr(req, "HealthCheckPort"),
		getStrPtr(req, "HealthCheckProtocol"),
		getStrPtr(req, "HealthCheckPath"),
		getIntPtr(req, "HealthCheckIntervalSeconds"),
		getIntPtr(req, "ThresholdCount"),
		getFloatPtr(req, "TrafficDialPercentage"),
		getMapList(req, "PortOverrides"),
		getMapList(req, "EndpointConfigurations"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"EndpointGroup": endpointGroupToMap(g)})
}

func handleDeleteEndpointGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "EndpointGroupArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("EndpointGroupArn is required."))
	}
	if err := store.DeleteEndpointGroup(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListEndpointGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ListenerArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ListenerArn is required."))
	}
	list := store.ListEndpointGroups(arn)
	out := make([]map[string]any, 0, len(list))
	for _, g := range list {
		out = append(out, endpointGroupToMap(g))
	}
	return jsonOK(map[string]any{"EndpointGroups": out})
}

func handleAddEndpoints(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "EndpointGroupArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("EndpointGroupArn is required."))
	}
	g, err := store.AddEndpoints(arn, getMapList(req, "EndpointConfigurations"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"EndpointGroupArn":     arn,
		"EndpointDescriptions": g.EndpointDescriptions,
	})
}

func handleRemoveEndpoints(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "EndpointGroupArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("EndpointGroupArn is required."))
	}
	if err := store.RemoveEndpoints(arn, getMapList(req, "EndpointIdentifiers")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── Custom Routing Endpoint Group handlers ───────────────────────────────────

func handleCreateCustomRoutingEndpointGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	listenerArn := getStr(req, "ListenerArn")
	if listenerArn == "" {
		return jsonErr(service.ErrValidation("ListenerArn is required."))
	}
	region := getStr(req, "EndpointGroupRegion")
	if region == "" {
		return jsonErr(service.ErrValidation("EndpointGroupRegion is required."))
	}
	destinations := getMapList(req, "DestinationConfigurations")
	if len(destinations) == 0 {
		return jsonErr(service.ErrValidation("DestinationConfigurations is required."))
	}
	g, err := store.CreateCustomRoutingEndpointGroup(listenerArn, region, destinations)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"EndpointGroup": customEndpointGroupToMap(g)})
}

func handleDescribeCustomRoutingEndpointGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "EndpointGroupArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("EndpointGroupArn is required."))
	}
	g, err := store.GetCustomRoutingEndpointGroup(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"EndpointGroup": customEndpointGroupToMap(g)})
}

func handleDeleteCustomRoutingEndpointGroup(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "EndpointGroupArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("EndpointGroupArn is required."))
	}
	if err := store.DeleteCustomRoutingEndpointGroup(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListCustomRoutingEndpointGroups(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ListenerArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ListenerArn is required."))
	}
	list := store.ListCustomRoutingEndpointGroups(arn)
	out := make([]map[string]any, 0, len(list))
	for _, g := range list {
		out = append(out, customEndpointGroupToMap(g))
	}
	return jsonOK(map[string]any{"EndpointGroups": out})
}

func handleAddCustomRoutingEndpoints(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "EndpointGroupArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("EndpointGroupArn is required."))
	}
	g, err := store.AddCustomRoutingEndpoints(arn, getMapList(req, "EndpointConfigurations"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"EndpointGroupArn":     arn,
		"EndpointDescriptions": g.EndpointDescriptions,
	})
}

func handleRemoveCustomRoutingEndpoints(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "EndpointGroupArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("EndpointGroupArn is required."))
	}
	if err := store.RemoveCustomRoutingEndpoints(arn, getStrList(req, "EndpointIds")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── Custom Routing Port Mappings ─────────────────────────────────────────────

func handleListCustomRoutingPortMappings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "AcceleratorArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("AcceleratorArn is required."))
	}
	if _, err := store.GetCustomRoutingAccelerator(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"PortMappings": []map[string]any{}})
}

func handleListCustomRoutingPortMappingsByDestination(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if getStr(req, "EndpointId") == "" {
		return jsonErr(service.ErrValidation("EndpointId is required."))
	}
	if getStr(req, "DestinationAddress") == "" {
		return jsonErr(service.ErrValidation("DestinationAddress is required."))
	}
	return jsonOK(map[string]any{"DestinationPortMappings": []map[string]any{}})
}

// ── Allow / Deny Custom Routing Traffic ──────────────────────────────────────

func handleAllowCustomRoutingTraffic(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "EndpointGroupArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("EndpointGroupArn is required."))
	}
	if getStr(req, "EndpointId") == "" {
		return jsonErr(service.ErrValidation("EndpointId is required."))
	}
	if _, err := store.GetCustomRoutingEndpointGroup(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDenyCustomRoutingTraffic(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "EndpointGroupArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("EndpointGroupArn is required."))
	}
	if getStr(req, "EndpointId") == "" {
		return jsonErr(service.ErrValidation("EndpointId is required."))
	}
	if _, err := store.GetCustomRoutingEndpointGroup(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── BYOIP CIDR handlers ──────────────────────────────────────────────────────

func handleProvisionByoipCidr(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	cidr := getStr(req, "Cidr")
	if cidr == "" {
		return jsonErr(service.ErrValidation("Cidr is required."))
	}
	c, err := store.ProvisionByoipCidr(cidr)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"ByoipCidr": byoipCidrToMap(c)})
}

func handleAdvertiseByoipCidr(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	cidr := getStr(req, "Cidr")
	if cidr == "" {
		return jsonErr(service.ErrValidation("Cidr is required."))
	}
	c, err := store.SetByoipCidrState(cidr, "ADVERTISING")
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"ByoipCidr": byoipCidrToMap(c)})
}

func handleWithdrawByoipCidr(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	cidr := getStr(req, "Cidr")
	if cidr == "" {
		return jsonErr(service.ErrValidation("Cidr is required."))
	}
	c, err := store.SetByoipCidrState(cidr, "READY")
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"ByoipCidr": byoipCidrToMap(c)})
}

func handleDeprovisionByoipCidr(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	cidr := getStr(req, "Cidr")
	if cidr == "" {
		return jsonErr(service.ErrValidation("Cidr is required."))
	}
	c, err := store.DeleteByoipCidr(cidr)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"ByoipCidr": byoipCidrToMap(c)})
}

func handleListByoipCidrs(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListByoipCidrs()
	out := make([]map[string]any, 0, len(list))
	for _, c := range list {
		out = append(out, byoipCidrToMap(c))
	}
	return jsonOK(map[string]any{"ByoipCidrs": out})
}

// ── Cross Account Attachment handlers ────────────────────────────────────────

func handleCreateCrossAccountAttachment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	a := store.CreateCrossAccountAttachment(name, getStrList(req, "Principals"), getMapList(req, "Resources"))
	tags := parseTagList(req, "Tags")
	if len(tags) > 0 {
		store.TagResource(a.Arn, tags)
	}
	return jsonOK(map[string]any{"CrossAccountAttachment": attachmentToMap(a)})
}

func handleDescribeCrossAccountAttachment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "AttachmentArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("AttachmentArn is required."))
	}
	a, err := store.GetCrossAccountAttachment(arn)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"CrossAccountAttachment": attachmentToMap(a)})
}

func handleUpdateCrossAccountAttachment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "AttachmentArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("AttachmentArn is required."))
	}
	a, err := store.UpdateCrossAccountAttachment(arn,
		getStrPtr(req, "Name"),
		getStrList(req, "AddPrincipals"),
		getStrList(req, "RemovePrincipals"),
		getMapList(req, "AddResources"),
		getMapList(req, "RemoveResources"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"CrossAccountAttachment": attachmentToMap(a)})
}

func handleDeleteCrossAccountAttachment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "AttachmentArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("AttachmentArn is required."))
	}
	if err := store.DeleteCrossAccountAttachment(arn); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleListCrossAccountAttachments(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListCrossAccountAttachments()
	out := make([]map[string]any, 0, len(list))
	for _, a := range list {
		out = append(out, attachmentToMap(a))
	}
	return jsonOK(map[string]any{"CrossAccountAttachments": out})
}

func handleListCrossAccountResourceAccounts(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	accounts := store.ListCrossAccountResourceAccounts()
	return jsonOK(map[string]any{"ResourceOwnerAwsAccountIds": accounts})
}

func handleListCrossAccountResources(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	owner := getStr(req, "ResourceOwnerAwsAccountId")
	if owner == "" {
		return jsonErr(service.ErrValidation("ResourceOwnerAwsAccountId is required."))
	}
	list := store.ListCrossAccountResources(owner)
	return jsonOK(map[string]any{"CrossAccountResources": list})
}

// ── Tag handlers ─────────────────────────────────────────────────────────────

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	store.TagResource(arn, parseTagList(req, "Tags"))
	return jsonOK(map[string]any{})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	store.UntagResource(arn, getStrList(req, "TagKeys"))
	return jsonOK(map[string]any{})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	tags := store.ListTags(arn)
	out := make([]map[string]any, 0, len(tags))
	for k, v := range tags {
		out = append(out, map[string]any{"Key": k, "Value": v})
	}
	return jsonOK(map[string]any{"Tags": out})
}
