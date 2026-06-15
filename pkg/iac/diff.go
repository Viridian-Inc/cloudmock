package iac

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
)

// DiffStatus indicates the state of a resource in the IaC vs runtime comparison.
type DiffStatus string

const (
	DiffMissing  DiffStatus = "missing"  // In IaC but not provisioned
	DiffOrphaned DiffStatus = "orphaned" // Provisioned but not in IaC
	DiffDrift    DiffStatus = "drift"    // Provisioned but config differs from IaC
	DiffSynced   DiffStatus = "synced"   // Provisioned and matches IaC
)

// DiffEntry describes one resource's comparison result.
type DiffEntry struct {
	Service string     `json:"service"`           // AWS service (dynamodb, lambda, sqs, etc.)
	Name    string     `json:"name"`              // Resource name
	Type    string     `json:"type"`              // Resource type (table, function, queue, etc.)
	Status  DiffStatus `json:"status"`            // missing, orphaned, drift, synced
	Details string     `json:"details,omitempty"` // Human-readable drift description
}

// DiffResult holds the complete IaC-vs-runtime comparison.
type DiffResult struct {
	Entries []DiffEntry `json:"entries"`
	Summary DiffSummary `json:"summary"`
}

// DiffSummary counts resources by status.
type DiffSummary struct {
	Total    int `json:"total"`
	Synced   int `json:"synced"`
	Missing  int `json:"missing"`
	Orphaned int `json:"orphaned"`
	Drift    int `json:"drift"`
}

// ComputeDiff compares an IaC scan result against what's currently running
// in the CloudMock service registry.
func ComputeDiff(iac *IaCImportResult, registry serviceRegistry, logger *slog.Logger) *DiffResult {
	result := &DiffResult{}

	// DynamoDB tables
	diffDynamo(iac.Tables, registry, result, logger)

	// Lambda functions
	diffLambda(iac.Lambdas, registry, result, logger)

	// SQS queues
	diffSQS(iac.SQSQueues, registry, result, logger)

	// SNS topics
	diffSNS(iac.SNSTopics, registry, result, logger)

	// S3 buckets
	diffS3(iac.S3Buckets, registry, result, logger)

	// Compute summary.
	for _, e := range result.Entries {
		result.Summary.Total++
		switch e.Status {
		case DiffSynced:
			result.Summary.Synced++
		case DiffMissing:
			result.Summary.Missing++
		case DiffOrphaned:
			result.Summary.Orphaned++
		case DiffDrift:
			result.Summary.Drift++
		}
	}

	return result
}

// --- Per-service diff ---

func diffDynamo(iacTables []DynamoTableDef, registry serviceRegistry, result *DiffResult, logger *slog.Logger) {
	svc, err := registry.Lookup("dynamodb")
	if err != nil {
		// Service not running → all IaC tables are "missing".
		for _, t := range iacTables {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "dynamodb", Name: t.Name, Type: "table",
				Status: DiffMissing, Details: "DynamoDB service not registered",
			})
		}
		return
	}

	// Get running table names.
	lister, ok := svc.(interface{ GetTableNames() []string })
	if !ok {
		return
	}
	running := make(map[string]bool)
	for _, name := range lister.GetTableNames() {
		running[name] = true
	}

	// inspector exposes a running table's key schema + GSIs for drift
	// detection. It's matched structurally so this package stays decoupled
	// from the DynamoDB service implementation (mirrors the GetTableNames
	// pattern above). Services that don't implement it fall back to a
	// name-only comparison.
	inspector, hasInspector := svc.(tableSchemaInspector)

	iacSet := make(map[string]bool)
	for _, t := range iacTables {
		iacSet[t.Name] = true
		if !running[t.Name] {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "dynamodb", Name: t.Name, Type: "table",
				Status: DiffMissing, Details: "Table declared in IaC but not provisioned",
			})
			continue
		}
		// Table exists by name — compare key schema and GSIs to detect drift.
		entry := DiffEntry{Service: "dynamodb", Name: t.Name, Type: "table", Status: DiffSynced}
		if hasInspector {
			if drift := dynamoTableDrift(t, inspector); len(drift) > 0 {
				entry.Status = DiffDrift
				entry.Details = strings.Join(drift, "; ")
			}
		}
		result.Entries = append(result.Entries, entry)
	}

	// Orphaned: running but not in IaC.
	for _, name := range lister.GetTableNames() {
		if !iacSet[name] {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "dynamodb", Name: name, Type: "table",
				Status: DiffOrphaned, Details: "Table exists but not declared in IaC",
			})
		}
	}
}

// tableSchemaInspector is implemented by the DynamoDB service to report a
// running table's key schema and global secondary indexes. The signature uses
// only built-in types so the service satisfies it structurally, keeping pkg/iac
// and the service package free of an import dependency on one another.
type tableSchemaInspector interface {
	// TableKeySchema returns the hash key, optional range key ("" when the
	// table has no sort key), and a map of GSI name → [hashKey, rangeKey] for
	// the named table. ok is false if the table is not present.
	TableKeySchema(name string) (hashKey, rangeKey string, gsis map[string][2]string, ok bool)
}

// dynamoTableDrift compares an IaC table definition against the running table's
// key schema and GSIs, returning sorted, human-readable drift descriptions. An
// empty result means the running table matches the declared configuration.
//
// Comparisons are skipped where the IaC side declares no value (e.g. an empty
// hash key from incomplete parsing) so partial IaC definitions don't produce
// false-positive drift.
func dynamoTableDrift(decl DynamoTableDef, inspector tableSchemaInspector) []string {
	hashKey, rangeKey, gsis, ok := inspector.TableKeySchema(decl.Name)
	if !ok {
		// Table vanished between the name listing and this lookup; the
		// missing/orphaned passes account for it, so report no drift.
		return nil
	}

	var drift []string

	// Partition (HASH) key — flagged only when IaC declares one.
	if decl.HashKey != "" && decl.HashKey != hashKey {
		drift = append(drift, fmt.Sprintf("hash key %q in IaC, %q running", decl.HashKey, hashKey))
	}

	// Sort (RANGE) key.
	if d := rangeKeyDrift("", decl.RangeKey, rangeKey); d != "" {
		drift = append(drift, d)
	}

	// Global secondary indexes, compared by index name.
	declGSIs := make(map[string][2]string, len(decl.GSIs))
	for _, g := range decl.GSIs {
		declGSIs[g.Name] = [2]string{g.HashKey, g.RangeKey}
	}
	for name, want := range declGSIs {
		got, exists := gsis[name]
		if !exists {
			drift = append(drift, fmt.Sprintf("GSI %q declared in IaC but not running", name))
			continue
		}
		if want[0] != "" && want[0] != got[0] {
			drift = append(drift, fmt.Sprintf("GSI %q hash key %q in IaC, %q running", name, want[0], got[0]))
		}
		if d := rangeKeyDrift(fmt.Sprintf("GSI %q ", name), want[1], got[1]); d != "" {
			drift = append(drift, d)
		}
	}
	for name := range gsis {
		if _, declared := declGSIs[name]; !declared {
			drift = append(drift, fmt.Sprintf("GSI %q running but not declared in IaC", name))
		}
	}

	sort.Strings(drift) // deterministic ordering for stable Details output
	return drift
}

// rangeKeyDrift formats a sort-key mismatch between a declared and running
// value, or returns "" when they match. label prefixes the message (e.g.
// "GSI \"foo\" ") so the same wording serves both tables and indexes.
func rangeKeyDrift(label, declared, running string) string {
	switch {
	case declared == running:
		return ""
	case declared == "":
		return fmt.Sprintf("%srange key %q running but absent in IaC", label, running)
	case running == "":
		return fmt.Sprintf("%srange key %q in IaC but absent running", label, declared)
	default:
		return fmt.Sprintf("%srange key %q in IaC, %q running", label, declared, running)
	}
}

func diffLambda(iacLambdas []LambdaDef, registry serviceRegistry, result *DiffResult, logger *slog.Logger) {
	svc, err := registry.Lookup("lambda")
	if err != nil {
		for _, l := range iacLambdas {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "lambda", Name: l.Name, Type: "function",
				Status: DiffMissing, Details: "Lambda service not registered",
			})
		}
		return
	}

	lister, ok := svc.(interface{ GetFunctionNames() []string })
	if !ok {
		return
	}
	running := make(map[string]bool)
	for _, name := range lister.GetFunctionNames() {
		running[name] = true
	}

	iacSet := make(map[string]bool)
	for _, l := range iacLambdas {
		iacSet[l.Name] = true
		if running[l.Name] {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "lambda", Name: l.Name, Type: "function",
				Status: DiffSynced,
			})
		} else {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "lambda", Name: l.Name, Type: "function",
				Status: DiffMissing, Details: "Function declared in IaC but not provisioned",
			})
		}
	}

	for _, name := range lister.GetFunctionNames() {
		if !iacSet[name] {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "lambda", Name: name, Type: "function",
				Status: DiffOrphaned, Details: "Function exists but not declared in IaC",
			})
		}
	}
}

func diffSQS(iacQueues []SQSQueueDef, registry serviceRegistry, result *DiffResult, logger *slog.Logger) {
	svc, err := registry.Lookup("sqs")
	if err != nil {
		for _, q := range iacQueues {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "sqs", Name: q.Name, Type: "queue",
				Status: DiffMissing,
			})
		}
		return
	}

	lister, ok := svc.(interface{ GetQueueNames() []string })
	if !ok {
		return
	}
	running := make(map[string]bool)
	for _, name := range lister.GetQueueNames() {
		running[name] = true
	}

	iacSet := make(map[string]bool)
	for _, q := range iacQueues {
		iacSet[q.Name] = true
		if running[q.Name] {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "sqs", Name: q.Name, Type: "queue", Status: DiffSynced,
			})
		} else {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "sqs", Name: q.Name, Type: "queue",
				Status: DiffMissing, Details: "Queue declared in IaC but not provisioned",
			})
		}
	}

	for _, name := range lister.GetQueueNames() {
		if !iacSet[name] {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "sqs", Name: name, Type: "queue",
				Status: DiffOrphaned,
			})
		}
	}
}

func diffSNS(iacTopics []SNSTopicDef, registry serviceRegistry, result *DiffResult, logger *slog.Logger) {
	svc, err := registry.Lookup("sns")
	if err != nil {
		for _, t := range iacTopics {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "sns", Name: t.Name, Type: "topic",
				Status: DiffMissing,
			})
		}
		return
	}

	lister, ok := svc.(interface{ GetTopicNames() []string })
	if !ok {
		return
	}
	running := make(map[string]bool)
	for _, name := range lister.GetTopicNames() {
		running[name] = true
	}

	iacSet := make(map[string]bool)
	for _, t := range iacTopics {
		iacSet[t.Name] = true
		if running[t.Name] {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "sns", Name: t.Name, Type: "topic", Status: DiffSynced,
			})
		} else {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "sns", Name: t.Name, Type: "topic",
				Status: DiffMissing,
			})
		}
	}

	for _, name := range lister.GetTopicNames() {
		if !iacSet[name] {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "sns", Name: name, Type: "topic",
				Status: DiffOrphaned,
			})
		}
	}
}

func diffS3(iacBuckets []S3BucketDef, registry serviceRegistry, result *DiffResult, logger *slog.Logger) {
	svc, err := registry.Lookup("s3")
	if err != nil {
		for _, b := range iacBuckets {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "s3", Name: b.Name, Type: "bucket",
				Status: DiffMissing,
			})
		}
		return
	}

	lister, ok := svc.(interface{ GetBucketNames() []string })
	if !ok {
		return
	}
	running := make(map[string]bool)
	for _, name := range lister.GetBucketNames() {
		running[name] = true
	}

	iacSet := make(map[string]bool)
	for _, b := range iacBuckets {
		iacSet[b.Name] = true
		if running[b.Name] {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "s3", Name: b.Name, Type: "bucket", Status: DiffSynced,
			})
		} else {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "s3", Name: b.Name, Type: "bucket",
				Status: DiffMissing,
			})
		}
	}

	for _, name := range lister.GetBucketNames() {
		if !iacSet[name] {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "s3", Name: name, Type: "bucket",
				Status: DiffOrphaned,
			})
		}
	}
}
