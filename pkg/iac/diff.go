package iac

import "log/slog"

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
	Service  string     `json:"service"`  // AWS service (dynamodb, lambda, sqs, etc.)
	Name     string     `json:"name"`     // Resource name
	Type     string     `json:"type"`     // Resource type (table, function, queue, etc.)
	Status   DiffStatus `json:"status"`   // missing, orphaned, drift, synced
	Details  string     `json:"details,omitempty"` // Human-readable drift description
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

	iacSet := make(map[string]bool)
	for _, t := range iacTables {
		iacSet[t.Name] = true
		if running[t.Name] {
			// TODO: compare key schema, GSIs for drift detection.
			result.Entries = append(result.Entries, DiffEntry{
				Service: "dynamodb", Name: t.Name, Type: "table",
				Status: DiffSynced,
			})
		} else {
			result.Entries = append(result.Entries, DiffEntry{
				Service: "dynamodb", Name: t.Name, Type: "table",
				Status: DiffMissing, Details: "Table declared in IaC but not provisioned",
			})
		}
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
