package codeconnections

import (
	gojson "github.com/goccy/go-json"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("InvalidInputException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func getStr(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getStrSlice(m map[string]any, key string) []string {
	arr, ok := m[key].([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func parseTagsList(tags []any) map[string]string {
	m := make(map[string]string)
	for _, t := range tags {
		if tm, ok := t.(map[string]any); ok {
			k := getStr(tm, "Key")
			v := getStr(tm, "Value")
			if k != "" {
				m[k] = v
			}
		}
	}
	return m
}

func tagsToList(m map[string]string) []map[string]any {
	out := make([]map[string]any, 0, len(m))
	for k, v := range m {
		out = append(out, map[string]any{"Key": k, "Value": v})
	}
	return out
}

// ---- Connection handlers ----

func handleCreateConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getStr(req, "ConnectionName")
	providerType := getStr(req, "ProviderType")
	hostARN := getStr(req, "HostArn")

	var tags map[string]string
	if tagList, ok := req["Tags"].([]any); ok {
		tags = parseTagsList(tagList)
	}

	conn, awsErr := store.CreateConnection(name, providerType, hostARN, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"ConnectionArn": conn.ARN,
		"Tags":          tagsToList(conn.Tags),
	})
}

func handleGetConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn := getStr(req, "ConnectionArn")
	conn, awsErr := store.GetConnection(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Connection": connectionToMap(conn)})
}

func handleListConnections(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	providerType := getStr(req, "ProviderTypeFilter")
	hostARN := getStr(req, "HostArnFilter")

	conns := store.ListConnections(providerType, hostARN)
	result := make([]map[string]any, len(conns))
	for i, c := range conns {
		result[i] = connectionToMap(c)
	}
	return jsonOK(map[string]any{"Connections": result})
}

func handleDeleteConnection(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn := getStr(req, "ConnectionArn")
	if awsErr := store.DeleteConnection(arn); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

func handleUpdateConnectionStatus(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn := getStr(req, "ConnectionArn")
	status := getStr(req, "ConnectionStatus")

	conn, awsErr := store.UpdateConnectionStatus(arn, status)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"ConnectionArn":    conn.ARN,
		"ConnectionStatus": conn.ConnectionStatus,
	})
}

// ---- Host handlers ----

func handleCreateHost(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	name := getStr(req, "Name")
	providerType := getStr(req, "ProviderType")
	providerEndpoint := getStr(req, "ProviderEndpoint")

	var vpcConfig *VPCConfiguration
	if vpc, ok := req["VpcConfiguration"].(map[string]any); ok {
		vpcConfig = &VPCConfiguration{
			VpcID:            getStr(vpc, "VpcId"),
			SubnetIDs:        getStrSlice(vpc, "SubnetIds"),
			SecurityGroupIDs: getStrSlice(vpc, "SecurityGroupIds"),
			TLSCertificate:   getStr(vpc, "TlsCertificate"),
		}
	}

	var tags map[string]string
	if tagList, ok := req["Tags"].([]any); ok {
		tags = parseTagsList(tagList)
	}

	host, awsErr := store.CreateHost(name, providerType, providerEndpoint, vpcConfig, tags)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"HostArn": host.ARN,
		"Tags":    tagsToList(host.Tags),
	})
}

func handleGetHost(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn := getStr(req, "HostArn")
	host, awsErr := store.GetHost(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(hostToMap(host))
}

func handleListHosts(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	hosts := store.ListHosts()
	result := make([]map[string]any, len(hosts))
	for i, h := range hosts {
		result[i] = hostToMap(h)
	}
	return jsonOK(map[string]any{"Hosts": result})
}

func handleDeleteHost(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn := getStr(req, "HostArn")
	if awsErr := store.DeleteHost(arn); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

// ---- Tag handlers ----

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn := getStr(req, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}

	var tags map[string]string
	if tagList, ok := req["Tags"].([]any); ok {
		tags = parseTagsList(tagList)
	}

	if awsErr := store.TagResource(arn, tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn := getStr(req, "ResourceArn")
	keys := getStrSlice(req, "TagKeys")

	if awsErr := store.UntagResource(arn, keys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn := getStr(req, "ResourceArn")
	tags := store.ListTagsForResource(arn)
	return jsonOK(map[string]any{"Tags": tagsToList(tags)})
}

// ---- conversion helpers ----

func connectionToMap(c *Connection) map[string]any {
	return map[string]any{
		"ConnectionName":   c.Name,
		"ConnectionArn":    c.ARN,
		"ProviderType":     c.ProviderType,
		"HostArn":          c.HostARN,
		"ConnectionStatus": c.ConnectionStatus,
		"OwnerAccountId":   c.OwnerAccountID,
	}
}

func hostToMap(h *Host) map[string]any {
	m := map[string]any{
		"Name":             h.Name,
		"HostArn":          h.ARN,
		"ProviderType":     h.ProviderType,
		"ProviderEndpoint": h.ProviderEndpoint,
		"Status":           h.Status,
	}
	if h.VPCConfiguration != nil {
		m["VpcConfiguration"] = map[string]any{
			"VpcId":            h.VPCConfiguration.VpcID,
			"SubnetIds":        h.VPCConfiguration.SubnetIDs,
			"SecurityGroupIds": h.VPCConfiguration.SecurityGroupIDs,
			"TlsCertificate":   h.VPCConfiguration.TLSCertificate,
		}
	}
	return m
}
