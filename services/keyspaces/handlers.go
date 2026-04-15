package keyspaces

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
		return service.NewAWSError("ValidationException",
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
		k := getStr(t, "key")
		if k == "" {
			k = getStr(t, "Key")
		}
		v := getStr(t, "value")
		if v == "" {
			v = getStr(t, "Value")
		}
		if k != "" {
			out[k] = v
		}
	}
	return out
}

func rfc3339(t time.Time) string { return t.Format(time.RFC3339) }

// ── Keyspace handlers ───────────────────────────────────────────────────────

func handleCreateKeyspace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "keyspaceName")
	if name == "" {
		return jsonErr(service.ErrValidation("keyspaceName is required."))
	}
	replicationType := "SINGLE_REGION"
	var regions []string
	if rs := getMap(req, "replicationSpecification"); rs != nil {
		replicationType = getStr(rs, "replicationStrategy")
		if replicationType == "" {
			replicationType = "SINGLE_REGION"
		}
		regions = getStrList(rs, "regionList")
	}

	ks, err := store.CreateKeyspace(name, replicationType, regions, parseTagList(req, "tags"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"resourceArn": ks.Arn})
}

func handleDeleteKeyspace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "keyspaceName")
	if name == "" {
		return jsonErr(service.ErrValidation("keyspaceName is required."))
	}
	if err := store.DeleteKeyspace(name); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleGetKeyspace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "keyspaceName")
	if name == "" {
		return jsonErr(service.ErrValidation("keyspaceName is required."))
	}
	ks, err := store.GetKeyspace(name)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(keyspaceSummary(ks))
}

func handleListKeyspaces(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	list := store.ListKeyspaces()
	out := make([]map[string]any, 0, len(list))
	for _, ks := range list {
		out = append(out, keyspaceSummary(ks))
	}
	return jsonOK(map[string]any{"keyspaces": out})
}

func handleUpdateKeyspace(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "keyspaceName")
	if name == "" {
		return jsonErr(service.ErrValidation("keyspaceName is required."))
	}
	var regions []string
	if rs := getMap(req, "replicationSpecification"); rs != nil {
		regions = getStrList(rs, "regionList")
	}
	ks, err := store.UpdateKeyspace(name, regions)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"resourceArn": ks.Arn})
}

func keyspaceSummary(ks *StoredKeyspace) map[string]any {
	return map[string]any{
		"keyspaceName":       ks.Name,
		"resourceArn":        ks.Arn,
		"replicationStrategy": ks.ReplicationType,
		"replicationRegions":  ks.ReplicationRegions,
	}
}

// ── Table handlers ──────────────────────────────────────────────────────────

func handleCreateTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	ksName := getStr(req, "keyspaceName")
	tableName := getStr(req, "tableName")
	if ksName == "" {
		return jsonErr(service.ErrValidation("keyspaceName is required."))
	}
	if tableName == "" {
		return jsonErr(service.ErrValidation("tableName is required."))
	}
	schema := getMap(req, "schemaDefinition")
	if schema == nil {
		return jsonErr(service.ErrValidation("schemaDefinition is required."))
	}

	t, err := store.CreateTable(
		ksName, tableName,
		schema,
		getMap(req, "capacitySpecification"),
		getMap(req, "encryptionSpecification"),
		getMap(req, "pointInTimeRecovery"),
		getMap(req, "ttl"),
		getInt(req, "defaultTimeToLive"),
		getMap(req, "comment"),
		parseTagList(req, "tags"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"resourceArn": t.Arn})
}

func handleDeleteTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.DeleteTable(getStr(req, "keyspaceName"), getStr(req, "tableName")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleGetTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	t, err := store.GetTable(getStr(req, "keyspaceName"), getStr(req, "tableName"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(tableSummary(t))
}

func handleGetTableAutoScalingSettings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	t, err := store.GetTable(getStr(req, "keyspaceName"), getStr(req, "tableName"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"keyspaceName": t.KeyspaceName,
		"tableName":    t.TableName,
		"resourceArn":  t.Arn,
		"autoScalingSpecification": map[string]any{
			"readCapacityAutoScaling": map[string]any{
				"autoScalingDisabled": true,
			},
			"writeCapacityAutoScaling": map[string]any{
				"autoScalingDisabled": true,
			},
		},
	})
}

func handleListTables(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListTables(getStr(req, "keyspaceName"))
	out := make([]map[string]any, 0, len(list))
	for _, t := range list {
		out = append(out, map[string]any{
			"keyspaceName": t.KeyspaceName,
			"tableName":    t.TableName,
			"resourceArn":  t.Arn,
		})
	}
	return jsonOK(map[string]any{"tables": out})
}

func handleUpdateTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	var ttl *int
	if _, ok := req["defaultTimeToLive"]; ok {
		v := getInt(req, "defaultTimeToLive")
		ttl = &v
	}
	t, err := store.UpdateTable(
		getStr(req, "keyspaceName"),
		getStr(req, "tableName"),
		getMap(req, "addColumns"),
		getMap(req, "capacitySpecification"),
		getMap(req, "pointInTimeRecovery"),
		ttl,
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"resourceArn": t.Arn})
}

func handleRestoreTable(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	t, err := store.RestoreTable(
		getStr(req, "sourceKeyspaceName"),
		getStr(req, "sourceTableName"),
		getStr(req, "targetKeyspaceName"),
		getStr(req, "targetTableName"),
	)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{"restoredTableARN": t.Arn})
}

func tableSummary(t *StoredTable) map[string]any {
	return map[string]any{
		"keyspaceName":       t.KeyspaceName,
		"tableName":          t.TableName,
		"resourceArn":        t.Arn,
		"status":             t.Status,
		"schemaDefinition":   t.SchemaDefinition,
		"capacitySpecification": t.CapacitySpec,
		"encryptionSpecification": t.EncryptionSpec,
		"pointInTimeRecovery": t.PITR,
		"ttl":                t.TTL,
		"defaultTimeToLive":  t.DefaultTimeToLive,
		"comment":            t.Comment,
		"creationTimestamp":  rfc3339(t.CreatedAt),
	}
}

// ── Type handlers ───────────────────────────────────────────────────────────

func handleCreateType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	ksName := getStr(req, "keyspaceName")
	typeName := getStr(req, "typeName")
	if ksName == "" || typeName == "" {
		return jsonErr(service.ErrValidation("keyspaceName and typeName are required."))
	}
	fields := getMapList(req, "fieldDefinitions")
	if len(fields) == 0 {
		return jsonErr(service.ErrValidation("fieldDefinitions is required."))
	}
	t, err := store.CreateType(ksName, typeName, fields)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"keyspaceArn": t.KeyspaceArn,
		"typeName":    t.TypeName,
	})
}

func handleDeleteType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if err := store.DeleteType(getStr(req, "keyspaceName"), getStr(req, "typeName")); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"keyspaceArn": "arn:aws:cassandra::/keyspace/" + getStr(req, "keyspaceName"),
		"typeName":    getStr(req, "typeName"),
	})
}

func handleGetType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	t, err := store.GetType(getStr(req, "keyspaceName"), getStr(req, "typeName"))
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{
		"keyspaceName":     t.KeyspaceName,
		"typeName":         t.TypeName,
		"keyspaceArn":      t.KeyspaceArn,
		"fieldDefinitions": t.Fields,
		"directReferringTables": t.DirectReferringTables,
		"directParentTypes":     t.DirectParentTypes,
		"maxNestingDepth":       t.MaxNestingDepth,
		"status":                "ACTIVE",
		"creationTimestamp":     rfc3339(t.CreatedAt),
	})
}

func handleListTypes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	list := store.ListTypes(getStr(req, "keyspaceName"))
	names := make([]string, 0, len(list))
	for _, t := range list {
		names = append(names, t.TypeName)
	}
	return jsonOK(map[string]any{"types": names})
}

// ── Tags ────────────────────────────────────────────────────────────────────

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	store.TagResource(arn, parseTagList(req, "tags"))
	return jsonOK(map[string]any{})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	keys := make([]string, 0)
	for _, t := range getMapList(req, "tags") {
		if k := getStr(t, "key"); k != "" {
			keys = append(keys, k)
		}
	}
	store.UntagResource(arn, keys)
	return jsonOK(map[string]any{})
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	tags := store.ListTags(arn)
	out := make([]map[string]any, 0, len(tags))
	for k, v := range tags {
		out = append(out, map[string]any{"key": k, "value": v})
	}
	return jsonOK(map[string]any{"tags": out})
}
