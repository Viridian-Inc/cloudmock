package dynamodb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── PartiQL: ExecuteStatement ────────────────────────────────────────────────

// ExecuteStatement provides minimal PartiQL support for the two common forms:
//
//	INSERT INTO "table" VALUE {'attr': value, ...}
//	SELECT * FROM "table" [WHERE attr = value [AND attr = value ...]]
//
// Values are string ('x'), number (42), boolean (true/false), null, or the ?
// placeholder bound positionally to Parameters (typed AttributeValues). INSERT
// delegates to PutItem; SELECT scans the table and filters by the equality
// conditions, returning the matching Items. Anything else (UPDATE/DELETE or an
// unparseable statement) is rejected with ValidationException rather than
// silently returning an empty result.
func handleExecuteStatement(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req struct {
		Statement  string `json:"Statement"`
		Parameters []any  `json:"Parameters"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return ddbErr("ValidationException", "Invalid request body.")
	}
	stmt := strings.TrimSpace(req.Statement)
	if stmt == "" {
		return ddbErr("ValidationException", "Statement is required.")
	}

	switch strings.ToUpper(strings.Fields(stmt)[0]) {
	case "INSERT":
		return execPartiQLInsert(store, stmt, req.Parameters)
	case "SELECT":
		return execPartiQLSelect(store, stmt, req.Parameters)
	default:
		return ddbErr("ValidationException",
			"Unsupported PartiQL statement; CloudMock supports INSERT and SELECT.")
	}
}

// execPartiQLInsert handles: INSERT INTO "table" VALUE {'k': v, ...}
func execPartiQLInsert(store *TableStore, stmt string, params []any) (*service.Response, error) {
	table, ok := partiqlTableName(stmt, "INTO")
	if !ok {
		return ddbErr("ValidationException", `INSERT must be of the form: INSERT INTO "table" VALUE {...}`)
	}
	open := strings.Index(stmt, "{")
	closeIdx := strings.LastIndex(stmt, "}")
	if open == -1 || closeIdx == -1 || closeIdx < open {
		return ddbErr("ValidationException", "INSERT requires a VALUE { ... } map.")
	}
	idx := 0
	item, perr := parsePartiQLMap(stmt[open+1:closeIdx], params, &idx)
	if perr != "" {
		return ddbErr("ValidationException", perr)
	}
	if len(item) == 0 {
		return ddbErr("ValidationException", "INSERT VALUE map is empty.")
	}
	if awsErr := store.PutItem(table, item); awsErr != nil {
		return jsonErr(awsErr)
	}
	// AWS returns no Items for a successful INSERT.
	return ddbOK(map[string]any{})
}

// execPartiQLSelect handles: SELECT * FROM "table" [WHERE attr = v [AND ...]]
func execPartiQLSelect(store *TableStore, stmt string, params []any) (*service.Response, error) {
	table, ok := partiqlTableName(stmt, "FROM")
	if !ok {
		return ddbErr("ValidationException", `SELECT must be of the form: SELECT ... FROM "table" [WHERE ...]`)
	}
	conds, perr := parsePartiQLWhere(stmt, params)
	if perr != "" {
		return ddbErr("ValidationException", perr)
	}
	all, _, _, awsErr := store.Scan(table, "", "", nil, nil, 0)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	items := make([]Item, 0, len(all))
	for _, it := range all {
		match := true
		for _, c := range conds {
			if !avEqual(it[c.attr], c.val) {
				match = false
				break
			}
		}
		if match {
			items = append(items, it)
		}
	}
	return ddbOK(map[string]any{"Items": items})
}

type partiqlCond struct {
	attr string
	val  AttributeValue
}

// partiqlTableName returns the double-quoted table name that follows the given
// keyword (INTO or FROM), case-insensitively.
func partiqlTableName(stmt, keyword string) (string, bool) {
	up := strings.ToUpper(stmt)
	ki := strings.Index(up, keyword)
	if ki == -1 {
		return "", false
	}
	rest := stmt[ki+len(keyword):]
	q1 := strings.Index(rest, `"`)
	if q1 == -1 {
		return "", false
	}
	rest = rest[q1+1:]
	q2 := strings.Index(rest, `"`)
	if q2 == -1 {
		return "", false
	}
	name := rest[:q2]
	if name == "" {
		return "", false
	}
	return name, true
}

// parsePartiQLWhere parses the optional WHERE clause into equality conditions.
// No WHERE clause means "match everything" (full scan).
func parsePartiQLWhere(stmt string, params []any) ([]partiqlCond, string) {
	up := strings.ToUpper(stmt)
	wi := strings.Index(up, " WHERE ")
	if wi == -1 {
		return nil, ""
	}
	clause := strings.TrimSpace(stmt[wi+len(" WHERE "):])
	idx := 0
	var conds []partiqlCond
	for _, part := range splitPartiQL(clause, " AND ") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		eq := strings.Index(part, "=")
		if eq == -1 {
			return nil, "WHERE supports only '=' equality conditions: " + part
		}
		attr := strings.Trim(strings.TrimSpace(part[:eq]), `"'`)
		if attr == "" {
			return nil, "empty attribute name in WHERE"
		}
		av, perr := parsePartiQLValue(strings.TrimSpace(part[eq+1:]), params, &idx)
		if perr != "" {
			return nil, perr
		}
		conds = append(conds, partiqlCond{attr: attr, val: av})
	}
	return conds, ""
}

// parsePartiQLMap parses the body of a VALUE {...} map into an Item.
func parsePartiQLMap(body string, params []any, idx *int) (Item, string) {
	item := Item{}
	for _, pair := range splitPartiQL(body, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		ci := strings.Index(pair, ":")
		if ci == -1 {
			return nil, "malformed VALUE entry: " + pair
		}
		key := strings.Trim(strings.TrimSpace(pair[:ci]), `"'`)
		if key == "" {
			return nil, "empty attribute name in VALUE"
		}
		av, perr := parsePartiQLValue(strings.TrimSpace(pair[ci+1:]), params, idx)
		if perr != "" {
			return nil, perr
		}
		item[key] = av
	}
	return item, ""
}

// parsePartiQLValue converts a single PartiQL literal (or ? placeholder) to a
// typed AttributeValue. idx tracks positional Parameter consumption.
func parsePartiQLValue(tok string, params []any, idx *int) (AttributeValue, string) {
	switch {
	case tok == "?":
		if *idx >= len(params) {
			return nil, "not enough Parameters for ? placeholders"
		}
		m, ok := params[*idx].(map[string]any)
		*idx++
		if !ok {
			return nil, "Parameter must be a typed AttributeValue (e.g. {\"S\":\"x\"})"
		}
		return AttributeValue(m), ""
	case len(tok) >= 2 && tok[0] == '\'' && tok[len(tok)-1] == '\'':
		return AttributeValue{"S": tok[1 : len(tok)-1]}, ""
	case strings.EqualFold(tok, "true"), strings.EqualFold(tok, "false"):
		return AttributeValue{"BOOL": strings.EqualFold(tok, "true")}, ""
	case strings.EqualFold(tok, "null"):
		return AttributeValue{"NULL": true}, ""
	default:
		if _, err := strconv.ParseFloat(tok, 64); err == nil {
			return AttributeValue{"N": tok}, ""
		}
		return nil, "unrecognized value: " + tok
	}
}

// splitPartiQL splits s on sep, ignoring occurrences inside single-quoted
// string literals so values like 'a,b' or 'x AND y' aren't split mid-string.
// sep is matched case-insensitively.
func splitPartiQL(s, sep string) []string {
	var parts []string
	var b strings.Builder
	inQuote := false
	for i := 0; i < len(s); {
		if s[i] == '\'' {
			inQuote = !inQuote
			b.WriteByte(s[i])
			i++
			continue
		}
		if !inQuote && i+len(sep) <= len(s) && strings.EqualFold(s[i:i+len(sep)], sep) {
			parts = append(parts, b.String())
			b.Reset()
			i += len(sep)
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	parts = append(parts, b.String())
	return parts
}

// ── Backups ──────────────────────────────────────────────────────────────────

type Backup struct {
	BackupArn              string  `json:"BackupArn"`
	BackupName             string  `json:"BackupName"`
	BackupStatus           string  `json:"BackupStatus"`
	TableName              string  `json:"TableName"`
	TableArn               string  `json:"TableArn"`
	BackupCreationDateTime float64 `json:"BackupCreationDateTime"`
}

var (
	backupsMu sync.RWMutex
	backups   = make(map[string]*Backup) // backupArn -> backup
)

func handleCreateBackup(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req struct {
		TableName  string `json:"TableName"`
		BackupName string `json:"BackupName"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return ddbErr("ValidationException", "Invalid request body.")
	}
	if req.TableName == "" || req.BackupName == "" {
		return ddbErr("ValidationException", "TableName and BackupName are required.")
	}

	// Verify table exists
	if _, awsErr := store.DescribeTable(req.TableName); awsErr != nil {
		return &service.Response{Format: service.FormatJSON}, awsErr
	}

	backupArn := fmt.Sprintf("arn:aws:dynamodb:us-east-1:000000000000:table/%s/backup/%s",
		req.TableName, req.BackupName)
	tableArn := fmt.Sprintf("arn:aws:dynamodb:us-east-1:000000000000:table/%s", req.TableName)

	backup := &Backup{
		BackupArn:              backupArn,
		BackupName:             req.BackupName,
		BackupStatus:           "AVAILABLE",
		TableName:              req.TableName,
		TableArn:               tableArn,
		BackupCreationDateTime: float64(time.Now().Unix()),
	}

	backupsMu.Lock()
	backups[backupArn] = backup
	backupsMu.Unlock()

	return ddbOK(map[string]any{
		"BackupDetails": backup,
	})
}

func handleListBackups(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req struct {
		TableName string `json:"TableName"`
	}
	_ = json.Unmarshal(ctx.Body, &req)

	backupsMu.RLock()
	defer backupsMu.RUnlock()

	var summaries []map[string]any
	for _, b := range backups {
		if req.TableName != "" && b.TableName != req.TableName {
			continue
		}
		summaries = append(summaries, map[string]any{
			"BackupArn":              b.BackupArn,
			"BackupName":             b.BackupName,
			"BackupStatus":           b.BackupStatus,
			"TableName":              b.TableName,
			"TableArn":               b.TableArn,
			"BackupCreationDateTime": b.BackupCreationDateTime,
		})
	}
	if summaries == nil {
		summaries = []map[string]any{}
	}

	return ddbOK(map[string]any{
		"BackupSummaries": summaries,
	})
}

func handleDescribeBackup(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req struct {
		BackupArn string `json:"BackupArn"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return ddbErr("ValidationException", "Invalid request body.")
	}

	backupsMu.RLock()
	b, ok := backups[req.BackupArn]
	backupsMu.RUnlock()

	if !ok {
		return ddbErr("BackupNotFoundException", "Backup not found.")
	}
	return ddbOK(map[string]any{"BackupDescription": b})
}

// ── Global Tables ────────────────────────────────────────────────────────────

type GlobalTable struct {
	GlobalTableName   string              `json:"GlobalTableName"`
	ReplicationGroup  []map[string]string `json:"ReplicationGroup"`
	GlobalTableArn    string              `json:"GlobalTableArn"`
	GlobalTableStatus string              `json:"GlobalTableStatus"`
	CreationDateTime  float64             `json:"CreationDateTime"`
}

var (
	globalTablesMu sync.RWMutex
	globalTables   = make(map[string]*GlobalTable)
)

func handleCreateGlobalTable(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req struct {
		GlobalTableName  string              `json:"GlobalTableName"`
		ReplicationGroup []map[string]string `json:"ReplicationGroup"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return ddbErr("ValidationException", "Invalid request body.")
	}
	if req.GlobalTableName == "" {
		return ddbErr("ValidationException", "GlobalTableName is required.")
	}

	gt := &GlobalTable{
		GlobalTableName:   req.GlobalTableName,
		ReplicationGroup:  req.ReplicationGroup,
		GlobalTableArn:    fmt.Sprintf("arn:aws:dynamodb::000000000000:global-table/%s", req.GlobalTableName),
		GlobalTableStatus: "ACTIVE",
		CreationDateTime:  float64(time.Now().Unix()),
	}

	globalTablesMu.Lock()
	globalTables[req.GlobalTableName] = gt
	globalTablesMu.Unlock()

	return ddbOK(map[string]any{
		"GlobalTableDescription": gt,
	})
}

func handleDescribeGlobalTable(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req struct {
		GlobalTableName string `json:"GlobalTableName"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return ddbErr("ValidationException", "Invalid request body.")
	}

	globalTablesMu.RLock()
	gt, ok := globalTables[req.GlobalTableName]
	globalTablesMu.RUnlock()

	if !ok {
		return ddbErr("GlobalTableNotFoundException", "Global table not found.")
	}
	return ddbOK(map[string]any{"GlobalTableDescription": gt})
}

func handleListGlobalTables(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	globalTablesMu.RLock()
	defer globalTablesMu.RUnlock()

	var items []map[string]any
	for _, gt := range globalTables {
		items = append(items, map[string]any{
			"GlobalTableName":  gt.GlobalTableName,
			"ReplicationGroup": gt.ReplicationGroup,
		})
	}
	if items == nil {
		items = []map[string]any{}
	}
	return ddbOK(map[string]any{"GlobalTables": items})
}

// ── Exports ──────────────────────────────────────────────────────────────────

type Export struct {
	ExportArn    string  `json:"ExportArn"`
	ExportStatus string  `json:"ExportStatus"`
	TableArn     string  `json:"TableArn"`
	S3Bucket     string  `json:"S3Bucket"`
	ExportFormat string  `json:"ExportFormat"`
	ExportTime   float64 `json:"ExportTime"`
}

var (
	exportsMu sync.RWMutex
	exports   = make(map[string]*Export)
)

func handleExportTableToPointInTime(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req struct {
		TableArn     string `json:"TableArn"`
		S3Bucket     string `json:"S3Bucket"`
		ExportFormat string `json:"ExportFormat"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return ddbErr("ValidationException", "Invalid request body.")
	}
	if req.TableArn == "" {
		return ddbErr("ValidationException", "TableArn is required.")
	}

	exportArn := fmt.Sprintf("%s/export/%d", req.TableArn, time.Now().UnixNano())
	if req.ExportFormat == "" {
		req.ExportFormat = "DYNAMODB_JSON"
	}

	exp := &Export{
		ExportArn:    exportArn,
		ExportStatus: "COMPLETED",
		TableArn:     req.TableArn,
		S3Bucket:     req.S3Bucket,
		ExportFormat: req.ExportFormat,
		ExportTime:   float64(time.Now().Unix()),
	}

	exportsMu.Lock()
	exports[exportArn] = exp
	exportsMu.Unlock()

	return ddbOK(map[string]any{
		"ExportDescription": exp,
	})
}

func handleDescribeExport(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req struct {
		ExportArn string `json:"ExportArn"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return ddbErr("ValidationException", "Invalid request body.")
	}

	exportsMu.RLock()
	exp, ok := exports[req.ExportArn]
	exportsMu.RUnlock()

	if !ok {
		return ddbErr("ExportNotFoundException", "Export not found.")
	}
	return ddbOK(map[string]any{"ExportDescription": exp})
}

func handleListExports(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	exportsMu.RLock()
	defer exportsMu.RUnlock()

	var summaries []map[string]any
	for _, exp := range exports {
		summaries = append(summaries, map[string]any{
			"ExportArn":    exp.ExportArn,
			"ExportStatus": exp.ExportStatus,
			"TableArn":     exp.TableArn,
		})
	}
	if summaries == nil {
		summaries = []map[string]any{}
	}
	return ddbOK(map[string]any{"ExportSummaries": summaries})
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func ddbOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func ddbErr(code, msg string) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON},
		service.NewAWSError(code, msg, http.StatusBadRequest)
}
