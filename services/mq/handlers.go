package mq

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

func emptyOK() (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func getStr(p map[string]any, k string) string {
	if v, ok := p[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBool(p map[string]any, k string) bool {
	if v, ok := p[k]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getBoolPtr(p map[string]any, k string) *bool {
	if v, ok := p[k]; ok {
		if b, ok := v.(bool); ok {
			return &b
		}
	}
	return nil
}

func getStrMap(p map[string]any, k string) map[string]string {
	if v, ok := p[k]; ok {
		if m, ok := v.(map[string]any); ok {
			r := make(map[string]string, len(m))
			for key, val := range m {
				if s, ok := val.(string); ok {
					r[key] = s
				}
			}
			return r
		}
	}
	return nil
}

func getStringSlice(p map[string]any, k string) []string {
	if v, ok := p[k]; ok {
		if arr, ok := v.([]any); ok {
			r := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					r = append(r, s)
				}
			}
			return r
		}
	}
	return nil
}

func getSliceOfMaps(p map[string]any, k string) []map[string]any {
	if v, ok := p[k]; ok {
		if arr, ok := v.([]any); ok {
			r := make([]map[string]any, 0, len(arr))
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					r = append(r, m)
				}
			}
			return r
		}
	}
	return nil
}

// ---- Broker handlers ----

func handleCreateBroker(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "brokerName")
	if name == "" {
		return jsonErr(service.ErrValidation("brokerName is required."))
	}
	b, awsErr := store.CreateBroker(
		name,
		getStr(p, "engineType"),
		getStr(p, "engineVersion"),
		getStr(p, "hostInstanceType"),
		getStr(p, "deploymentMode"),
		getBool(p, "autoMinorVersionUpgrade"),
		getBool(p, "publiclyAccessible"),
		getStringSlice(p, "subnetIds"),
		getStringSlice(p, "securityGroups"),
		getSliceOfMaps(p, "users"),
		getStrMap(p, "tags"),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"brokerId":  b.BrokerId,
		"brokerArn": b.BrokerArn,
	})
}

func handleDescribeBroker(p map[string]any, store *Store) (*service.Response, error) {
	id := getStr(p, "brokerId")
	if id == "" {
		return jsonErr(service.ErrValidation("brokerId is required."))
	}
	b, awsErr := store.DescribeBroker(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(brokerToMap(b))
}

func handleListBrokers(store *Store) (*service.Response, error) {
	brokers := store.ListBrokers()
	summaries := make([]map[string]any, 0, len(brokers))
	for _, b := range brokers {
		summaries = append(summaries, map[string]any{
			"brokerId":    b.BrokerId,
			"brokerArn":   b.BrokerArn,
			"brokerName":  b.BrokerName,
			"brokerState": string(b.BrokerState),
			"engineType":  b.EngineType,
			"createdAt":   b.CreationTime.Format(time.RFC3339),
			"deploymentMode": b.DeploymentMode,
			"hostInstanceType": b.HostInstanceType,
		})
	}
	return jsonOK(map[string]any{"brokerSummaries": summaries})
}

func handleDeleteBroker(p map[string]any, store *Store) (*service.Response, error) {
	id := getStr(p, "brokerId")
	if id == "" {
		return jsonErr(service.ErrValidation("brokerId is required."))
	}
	if awsErr := store.DeleteBroker(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"brokerId": id})
}

func handleUpdateBroker(p map[string]any, store *Store) (*service.Response, error) {
	id := getStr(p, "brokerId")
	if id == "" {
		return jsonErr(service.ErrValidation("brokerId is required."))
	}
	b, awsErr := store.UpdateBroker(id, getStr(p, "hostInstanceType"), getStr(p, "engineVersion"), getBoolPtr(p, "autoMinorVersionUpgrade"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"brokerId": b.BrokerId})
}

func handleRebootBroker(p map[string]any, store *Store) (*service.Response, error) {
	id := getStr(p, "brokerId")
	if id == "" {
		return jsonErr(service.ErrValidation("brokerId is required."))
	}
	if awsErr := store.RebootBroker(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Configuration handlers ----

func handleCreateConfiguration(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "name")
	if name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	cfg, awsErr := store.CreateConfiguration(
		name,
		getStr(p, "description"),
		getStr(p, "engineType"),
		getStr(p, "engineVersion"),
		getStr(p, "data"),
		getStrMap(p, "tags"),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"id":   cfg.Id,
		"arn":  cfg.Arn,
		"name": cfg.Name,
		"latestRevision": map[string]any{
			"revisionId":  cfg.LatestRevisionId,
			"description": cfg.Description,
			"created":     cfg.CreationTime.Format(time.RFC3339),
		},
		"created": cfg.CreationTime.Format(time.RFC3339),
	})
}

func handleDescribeConfiguration(p map[string]any, store *Store) (*service.Response, error) {
	id := getStr(p, "configurationId")
	if id == "" {
		return jsonErr(service.ErrValidation("configurationId is required."))
	}
	cfg, awsErr := store.DescribeConfiguration(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"id":            cfg.Id,
		"arn":           cfg.Arn,
		"name":          cfg.Name,
		"description":   cfg.Description,
		"engineType":    cfg.EngineType,
		"engineVersion": cfg.EngineVersion,
		"latestRevision": map[string]any{
			"revisionId":  cfg.LatestRevisionId,
			"description": cfg.Description,
			"created":     cfg.CreationTime.Format(time.RFC3339),
		},
		"created": cfg.CreationTime.Format(time.RFC3339),
	})
}

func handleListConfigurations(store *Store) (*service.Response, error) {
	cfgs := store.ListConfigurations()
	entries := make([]map[string]any, 0, len(cfgs))
	for _, cfg := range cfgs {
		entries = append(entries, map[string]any{
			"id":            cfg.Id,
			"arn":           cfg.Arn,
			"name":          cfg.Name,
			"description":   cfg.Description,
			"engineType":    cfg.EngineType,
			"engineVersion": cfg.EngineVersion,
			"created":       cfg.CreationTime.Format(time.RFC3339),
		})
	}
	return jsonOK(map[string]any{"configurations": entries})
}

func handleUpdateConfiguration(p map[string]any, store *Store) (*service.Response, error) {
	id := getStr(p, "configurationId")
	if id == "" {
		return jsonErr(service.ErrValidation("configurationId is required."))
	}
	cfg, awsErr := store.UpdateConfiguration(id, getStr(p, "description"), getStr(p, "data"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"id":  cfg.Id,
		"arn": cfg.Arn,
		"latestRevision": map[string]any{
			"revisionId":  cfg.LatestRevisionId,
			"description": cfg.Description,
		},
	})
}

// ---- User handlers ----

func handleDescribeConfigurationRevision(p map[string]any, store *Store) (*service.Response, error) {
	id := getStr(p, "configurationId")
	if id == "" {
		return jsonErr(service.ErrValidation("configurationId is required."))
	}
	revID := 0
	if v, ok := p["configurationRevision"]; ok {
		switch rv := v.(type) {
		case float64:
			revID = int(rv)
		case int:
			revID = rv
		}
	}
	if revID == 0 {
		return jsonErr(service.ErrValidation("configurationRevision is required."))
	}
	cfg, rev, awsErr := store.DescribeConfigurationRevision(id, revID)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"configurationId":       cfg.Id,
		"configurationRevision": rev.RevisionId,
		"description":           rev.Description,
		"data":                  rev.Data,
		"created":               rev.CreationTime.Format(time.RFC3339),
	})
}

func handleListConfigurationRevisions(p map[string]any, store *Store) (*service.Response, error) {
	id := getStr(p, "configurationId")
	if id == "" {
		return jsonErr(service.ErrValidation("configurationId is required."))
	}
	cfg, revisions, awsErr := store.ListConfigurationRevisions(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	revList := make([]map[string]any, 0, len(revisions))
	for _, rev := range revisions {
		revList = append(revList, map[string]any{
			"revision":    rev.RevisionId,
			"description": rev.Description,
			"created":     rev.CreationTime.Format(time.RFC3339),
		})
	}
	return jsonOK(map[string]any{
		"configurationId": cfg.Id,
		"maxResults":      len(revList),
		"revisions":       revList,
	})
}

func handleCreateUser(p map[string]any, store *Store) (*service.Response, error) {
	brokerId := getStr(p, "brokerId")
	username := getStr(p, "username")
	if brokerId == "" || username == "" {
		return jsonErr(service.ErrValidation("brokerId and username are required."))
	}
	if awsErr := store.CreateUser(brokerId, username, getBool(p, "consoleAccess"), getStringSlice(p, "groups")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDescribeUser(p map[string]any, store *Store) (*service.Response, error) {
	brokerId := getStr(p, "brokerId")
	username := getStr(p, "username")
	if brokerId == "" || username == "" {
		return jsonErr(service.ErrValidation("brokerId and username are required."))
	}
	u, awsErr := store.DescribeUser(brokerId, username)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"brokerId":      u.BrokerId,
		"username":      u.Username,
		"consoleAccess": u.ConsoleAccess,
		"groups":        u.Groups,
	})
}

func handleListUsers(p map[string]any, store *Store) (*service.Response, error) {
	brokerId := getStr(p, "brokerId")
	if brokerId == "" {
		return jsonErr(service.ErrValidation("brokerId is required."))
	}
	users, awsErr := store.ListUsers(brokerId)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	entries := make([]map[string]any, 0, len(users))
	for _, u := range users {
		entries = append(entries, map[string]any{
			"username": u.Username,
		})
	}
	return jsonOK(map[string]any{"brokerId": brokerId, "users": entries})
}

func handleUpdateUser(p map[string]any, store *Store) (*service.Response, error) {
	brokerId := getStr(p, "brokerId")
	username := getStr(p, "username")
	if brokerId == "" || username == "" {
		return jsonErr(service.ErrValidation("brokerId and username are required."))
	}
	if awsErr := store.UpdateUser(brokerId, username, getBoolPtr(p, "consoleAccess"), getStringSlice(p, "groups")); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDeleteUser(p map[string]any, store *Store) (*service.Response, error) {
	brokerId := getStr(p, "brokerId")
	username := getStr(p, "username")
	if brokerId == "" || username == "" {
		return jsonErr(service.ErrValidation("brokerId and username are required."))
	}
	if awsErr := store.DeleteUser(brokerId, username); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Tag handlers ----

func handleCreateTags(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	tags := getStrMap(p, "tags")
	if awsErr := store.CreateTags(arn, tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDeleteTags(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	tagKeys := getStringSlice(p, "tagKeys")
	if awsErr := store.DeleteTags(arn, tagKeys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleListTags(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	tags, awsErr := store.ListTags(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"tags": tags})
}

// ---- helpers ----

func brokerToMap(b *Broker) map[string]any {
	return map[string]any{
		"brokerId":                b.BrokerId,
		"brokerArn":              b.BrokerArn,
		"brokerName":             b.BrokerName,
		"brokerState":            string(b.BrokerState),
		"engineType":             b.EngineType,
		"engineVersion":          b.EngineVersion,
		"hostInstanceType":       b.HostInstanceType,
		"deploymentMode":         b.DeploymentMode,
		"autoMinorVersionUpgrade": b.AutoMinorVersionUpgrade,
		"publiclyAccessible":     b.PubliclyAccessible,
		"subnetIds":              b.SubnetIds,
		"securityGroups":         b.SecurityGroups,
		"brokerInstances":        b.BrokerInstances,
		"created":                b.CreationTime.Format(time.RFC3339),
		"tags":                   b.Tags,
	}
}
