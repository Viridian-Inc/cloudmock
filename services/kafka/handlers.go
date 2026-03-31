package kafka

import (
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonCreated(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusCreated, Body: body, Format: service.FormatJSON}, nil
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

func getMap(p map[string]any, k string) map[string]any {
	if v, ok := p[k]; ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return nil
}

func getInt(p map[string]any, k string) int {
	if v, ok := p[k]; ok {
		if n, ok := v.(float64); ok {
			return int(n)
		}
	}
	return 0
}

func getInt64(p map[string]any, k string) int64 {
	if v, ok := p[k]; ok {
		if n, ok := v.(float64); ok {
			return int64(n)
		}
	}
	return 0
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

// ---- Cluster handlers ----

func handleCreateCluster(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "clusterName")
	if name == "" {
		return jsonErr(service.ErrValidation("clusterName is required."))
	}
	c, awsErr := store.CreateCluster(
		name,
		getStr(p, "kafkaVersion"),
		getStr(p, "clusterType"),
		getInt(p, "numberOfBrokerNodes"),
		getMap(p, "brokerNodeGroupInfo"),
		getMap(p, "encryptionInfo"),
		getMap(p, "loggingInfo"),
		getStr(p, "enhancedMonitoring"),
		getStrMap(p, "tags"),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"clusterArn":  c.ClusterArn,
		"clusterName": c.ClusterName,
		"state":       string(c.State),
	})
}

func handleDescribeCluster(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "clusterArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("clusterArn is required."))
	}
	c, awsErr := store.DescribeCluster(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"clusterInfo": clusterToMap(c)})
}

func handleListClusters(store *Store) (*service.Response, error) {
	clusters := store.ListClusters()
	infos := make([]map[string]any, 0, len(clusters))
	for _, c := range clusters {
		infos = append(infos, clusterToMap(c))
	}
	return jsonOK(map[string]any{"clusterInfoList": infos})
}

func handleDeleteCluster(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "clusterArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("clusterArn is required."))
	}
	if awsErr := store.DeleteCluster(arn); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"clusterArn": arn, "state": "DELETING"})
}

func handleUpdateBrokerCount(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "clusterArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("clusterArn is required."))
	}
	op, awsErr := store.UpdateBrokerCount(arn, getInt(p, "targetNumberOfBrokerNodes"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"clusterArn": arn, "clusterOperationArn": op.OperationArn})
}

func handleUpdateBrokerStorage(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "clusterArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("clusterArn is required."))
	}
	op, awsErr := store.UpdateBrokerStorage(arn, getInt(p, "targetBrokerEBSVolumeInfo"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"clusterArn": arn, "clusterOperationArn": op.OperationArn})
}

func handleUpdateClusterConfiguration(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "clusterArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("clusterArn is required."))
	}
	configInfo := getMap(p, "configurationInfo")
	configArn := ""
	var configRevision int64
	if configInfo != nil {
		configArn = getStr(configInfo, "arn")
		configRevision = getInt64(configInfo, "revision")
	}
	op, awsErr := store.UpdateClusterConfiguration(arn, configArn, configRevision)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"clusterArn": arn, "clusterOperationArn": op.OperationArn})
}

func handleRebootBroker(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "clusterArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("clusterArn is required."))
	}
	op, awsErr := store.RebootBroker(arn, getStringSlice(p, "brokerIds"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"clusterArn": arn, "clusterOperationArn": op.OperationArn})
}

func handleGetBootstrapBrokers(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "clusterArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("clusterArn is required."))
	}
	brokers, awsErr := store.GetBootstrapBrokers(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"bootstrapBrokerString":    brokers,
		"bootstrapBrokerStringTls": brokers,
	})
}

func handleListNodes(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "clusterArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("clusterArn is required."))
	}
	nodes, awsErr := store.ListNodes(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"nodeInfoList": nodes})
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
		getStr(p, "kafkaVersion"),
		getStr(p, "serverProperties"),
		getStrMap(p, "tags"),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"arn":            cfg.Arn,
		"name":           cfg.Name,
		"state":          cfg.State,
		"creationTime":   cfg.CreationTime.Format(time.RFC3339),
		"latestRevision": revisionToMap(cfg.LatestRevision),
	})
}

func handleDescribeConfiguration(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "arn")
	if arn == "" {
		return jsonErr(service.ErrValidation("arn is required."))
	}
	cfg, awsErr := store.DescribeConfiguration(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"arn":            cfg.Arn,
		"name":           cfg.Name,
		"description":    cfg.Description,
		"kafkaVersions":  cfg.KafkaVersions,
		"state":          cfg.State,
		"creationTime":   cfg.CreationTime.Format(time.RFC3339),
		"latestRevision": revisionToMap(cfg.LatestRevision),
	})
}

func handleListConfigurations(store *Store) (*service.Response, error) {
	cfgs := store.ListConfigurations()
	entries := make([]map[string]any, 0, len(cfgs))
	for _, cfg := range cfgs {
		entries = append(entries, map[string]any{
			"arn":            cfg.Arn,
			"name":           cfg.Name,
			"description":    cfg.Description,
			"state":          cfg.State,
			"creationTime":   cfg.CreationTime.Format(time.RFC3339),
			"latestRevision": revisionToMap(cfg.LatestRevision),
		})
	}
	return jsonOK(map[string]any{"configurations": entries})
}

func handleUpdateConfiguration(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "arn")
	if arn == "" {
		return jsonErr(service.ErrValidation("arn is required."))
	}
	cfg, awsErr := store.UpdateConfiguration(arn, getStr(p, "description"), getStr(p, "serverProperties"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"arn":            cfg.Arn,
		"latestRevision": revisionToMap(cfg.LatestRevision),
	})
}

func handleDeleteConfiguration(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "arn")
	if arn == "" {
		return jsonErr(service.ErrValidation("arn is required."))
	}
	if awsErr := store.DeleteConfiguration(arn); awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"arn": arn, "state": "DELETING"})
}

// ---- Operation handlers ----

func handleListClusterOperations(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "clusterArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("clusterArn is required."))
	}
	ops := store.ListClusterOperations(arn)
	entries := make([]map[string]any, 0, len(ops))
	for _, op := range ops {
		entries = append(entries, operationToMap(op))
	}
	return jsonOK(map[string]any{"clusterOperationInfoList": entries})
}

func handleDescribeClusterOperation(p map[string]any, store *Store) (*service.Response, error) {
	opArn := getStr(p, "clusterOperationArn")
	if opArn == "" {
		return jsonErr(service.ErrValidation("clusterOperationArn is required."))
	}
	op, awsErr := store.DescribeClusterOperation(opArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"clusterOperationInfo": operationToMap(op)})
}

// ---- Tag handlers ----

func handleTagResource(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	tags := getStrMap(p, "tags")
	if awsErr := store.TagResource(arn, tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleUntagResource(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	tagKeys := getStringSlice(p, "tagKeys")
	if awsErr := store.UntagResource(arn, tagKeys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleListTagsForResource(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	tags, awsErr := store.ListTagsForResource(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"tags": tags})
}

// ---- helpers ----

func clusterToMap(c *Cluster) map[string]any {
	m := map[string]any{
		"clusterArn":          c.ClusterArn,
		"clusterName":         c.ClusterName,
		"state":               string(c.State),
		"clusterType":         c.ClusterType,
		"creationTime":        c.CreationTime.Format(time.RFC3339),
		"numberOfBrokerNodes": c.NumberOfBrokerNodes,
		"tags":                c.Tags,
	}
	if c.KafkaVersion != "" {
		m["currentBrokerSoftwareInfo"] = map[string]any{"kafkaVersion": c.KafkaVersion}
	}
	if c.BrokerNodeGroupInfo != nil {
		m["brokerNodeGroupInfo"] = c.BrokerNodeGroupInfo
	}
	if c.EncryptionInfo != nil {
		m["encryptionInfo"] = c.EncryptionInfo
	}
	if c.EnhancedMonitoring != "" {
		m["enhancedMonitoring"] = c.EnhancedMonitoring
	}
	if c.LoggingInfo != nil {
		m["loggingInfo"] = c.LoggingInfo
	}
	return m
}

func revisionToMap(r ConfigurationRevision) map[string]any {
	return map[string]any{
		"revision":     r.Revision,
		"description":  r.Description,
		"creationTime": r.CreationTime.Format(time.RFC3339),
	}
}

func operationToMap(op *ClusterOperation) map[string]any {
	m := map[string]any{
		"operationArn":   op.OperationArn,
		"clusterArn":     op.ClusterArn,
		"operationType":  op.OperationType,
		"operationState": string(op.OperationState),
		"creationTime":   op.CreationTime.Format(time.RFC3339),
	}
	if op.EndTime != nil {
		m["endTime"] = op.EndTime.Format(time.RFC3339)
	}
	return m
}
