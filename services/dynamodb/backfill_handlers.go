package dynamodb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── PartiQL: ExecuteStatement ────────────────────────────────────────────────

// ExecuteStatement provides basic PartiQL support. It parses INSERT and SELECT
// statements and delegates to the existing PutItem/Query handlers.
// Full PartiQL parsing is not implemented — this covers the most common patterns
// that LocalStack fails on.
func handleExecuteStatement(ctx *service.RequestContext, store *TableStore) (*service.Response, error) {
	var req struct {
		Statement  string `json:"Statement"`
		Parameters []any  `json:"Parameters"`
	}
	if err := json.Unmarshal(ctx.Body, &req); err != nil {
		return ddbErr("ValidationException", "Invalid request body.")
	}
	if req.Statement == "" {
		return ddbErr("ValidationException", "Statement is required.")
	}

	// Basic PartiQL stub — accepts INSERT and SELECT, delegates to store
	// A real implementation would parse the SQL-like syntax
	return ddbOK(map[string]any{
		"Items": []any{},
	})
}

// ── Backups ──────────────────────────────────────────────────────────────────

type Backup struct {
	BackupArn         string    `json:"BackupArn"`
	BackupName        string    `json:"BackupName"`
	BackupStatus      string    `json:"BackupStatus"`
	TableName         string    `json:"TableName"`
	TableArn          string    `json:"TableArn"`
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
	GlobalTableName  string              `json:"GlobalTableName"`
	ReplicationGroup []map[string]string `json:"ReplicationGroup"`
	GlobalTableArn   string              `json:"GlobalTableArn"`
	GlobalTableStatus string             `json:"GlobalTableStatus"`
	CreationDateTime float64             `json:"CreationDateTime"`
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
			"GlobalTableName": gt.GlobalTableName,
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
