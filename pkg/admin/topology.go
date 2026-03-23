package admin

import (
	"fmt"
	"strings"

	"github.com/neureaux/cloudmock/pkg/gateway"
)

// ---- Topology response types ----

// TopologyNodeV2 describes a resource in the topology graph.
type TopologyNodeV2 struct {
	ID             string `json:"id"`                        // "lambda:attendance-handler" or "external:expo-app"
	Label          string `json:"label"`
	Service        string `json:"service"`
	Type           string `json:"type"`                      // "function", "table", "queue", "topic", "bucket", "client", "plugin"
	Group          string `json:"group"`                     // group ID
	RequestService string `json:"requestService,omitempty"` // service name in request log (e.g. "bff" for external:bff-service)
}

// TopologyEdgeV2 describes a connection between resources.
type TopologyEdgeV2 struct {
	Source       string  `json:"source"`
	Target       string  `json:"target"`
	Type         string  `json:"type"`         // "trigger", "read_write", "publish", "subscribe"
	Label        string  `json:"label"`
	Discovered   string  `json:"discovered"`   // "esm", "subscription", "rule", "traffic", "config", "alarm", "cfn"
	AvgLatencyMs float64 `json:"avgLatencyMs"` // average latency in milliseconds (0 = unknown)
	CallCount    int     `json:"callCount"`    // number of observed calls (0 = config-only)
}

// TopologyGroupV2 describes a visual grouping.
type TopologyGroupV2 struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Color string `json:"color"`
}

// TopologyResponseV2 is the dynamic topology response.
type TopologyResponseV2 struct {
	Nodes  []TopologyNodeV2  `json:"nodes"`
	Edges  []TopologyEdgeV2  `json:"edges"`
	Groups []TopologyGroupV2 `json:"groups"`
}

// ---- Static group definitions ----

var topologyGroups = []TopologyGroupV2{
	{ID: "Client", Label: "Client Apps", Color: "#6366F1"},
	{ID: "Plugins", Label: "External Services", Color: "#94A3B8"},
	{ID: "API", Label: "API Layer", Color: "#06B6D4"},
	{ID: "Auth", Label: "Auth & Identity", Color: "#8B5CF6"},
	{ID: "Compute", Label: "Compute", Color: "#3B82F6"},
	{ID: "Core Data", Label: "Core Domain", Color: "#10B981"},
	{ID: "Features", Label: "Features", Color: "#059669"},
	{ID: "Admin", Label: "Admin & Analytics", Color: "#6366F1"},
	{ID: "Integrations", Label: "Integrations", Color: "#A855F7"},
	{ID: "Facilities", Label: "Facilities", Color: "#14B8A6"},
	{ID: "Messaging", Label: "Messaging", Color: "#F97316"},
	{ID: "Storage", Label: "Storage", Color: "#F59E0B"},
	{ID: "Security", Label: "Security & Config", Color: "#6366F1"},
	{ID: "Monitoring", Label: "Monitoring", Color: "#EC4899"},
}

// tableGroups maps DynamoDB table names to their topology group.
var tableGroups = map[string]string{
	"enterprise":          "Core Data",
	"membership":          "Core Data",
	"resource":            "Core Data",
	"resourceMembership":  "Core Data",
	"session":             "Core Data",
	"attendance":          "Core Data",
	"order":               "Core Data",
	"calendar":            "Core Data",
	"userMetadata":        "Core Data",
	"event":               "Core Data",
	"eventInstance":        "Core Data",
	"personalEvent":       "Core Data",
	"featureFlag":         "Features",
	"notification":        "Features",
	"webhook":             "Features",
	"webhookDelivery":     "Features",
	"apiKey":              "Features",
	"attendancePolicy":    "Features",
	"userGroup":           "Features",
	"invitation":          "Features",
	"classTemplate":       "Features",
	"report":              "Features",
	"attendanceOverride":  "Features",
	"colorPreference":     "Features",
	"release":             "Admin",
	"deployment":          "Admin",
	"rolloutStage":        "Admin",
	"healthMetrics":       "Admin",
	"auditLog":            "Admin",
	"approval":            "Admin",
	"analytics":           "Admin",
	"analyticsConsent":    "Admin",
	"integration":         "Integrations",
	"lmsIntegration":      "Integrations",
	"lmsCourseMapping":    "Integrations",
	"lmsSyncLog":          "Integrations",
	"dispute":             "Integrations",
	"dataRequest":         "Integrations",
	"seatingChart":        "Facilities",
	"seatPreferenceRequest": "Facilities",
	"building":            "Facilities",
	"roomBlueprint":       "Facilities",
	"identityProvider":    "Auth",
	"customDomain":        "Auth",
	"tinyUrl":             "Features",
}

// ---- Builder ----

// buildDynamicTopology merges IaC-defined topology (nodes + edges pushed from
// Pulumi/Terraform) with dynamically-discovered resources from cloudmock services
// and traffic-observed edges from the request log.
func (a *API) buildDynamicTopology() TopologyResponseV2 {
	nodes := make([]TopologyNodeV2, 0, 64)
	edges := make([]TopologyEdgeV2, 0, 64)
	edgeSet := make(map[string]bool)

	addNode := func(id, label, svc, typ, group string) {
		nodes = append(nodes, TopologyNodeV2{
			ID:      id,
			Label:   label,
			Service: svc,
			Type:    typ,
			Group:   group,
		})
	}

	addEdge := func(source, target, typ, label, discovered string) {
		key := source + "|" + target + "|" + discovered
		if edgeSet[key] {
			return
		}
		edgeSet[key] = true
		edges = append(edges, TopologyEdgeV2{
			Source:     source,
			Target:     target,
			Type:       typ,
			Label:      label,
			Discovered: discovered,
		})
	}

	// 1. Load IaC-defined nodes and edges (pushed from Pulumi via /api/topology/config)
	a.iacTopologyMu.RLock()
	iacCfg := a.iacTopology
	a.iacTopologyMu.RUnlock()

	if iacCfg != nil {
		for _, n := range iacCfg.Nodes {
			nodes = append(nodes, n) // preserve all fields including requestService
		}
		for _, e := range iacCfg.Edges {
			addEdge(e.Source, e.Target, e.Type, e.Label, e.Discovered)
		}
	}

	// 4. Query each service for resources
	svcs := a.registry.List()

	// Collect resource node IDs for cross-reference
	lambdaFunctions := make(map[string]bool)
	dynamoTables := make(map[string]bool)
	sqsQueues := make(map[string]bool)
	snsTopics := make(map[string]bool)

	for _, svc := range svcs {
		switch svc.Name() {
		case "lambda":
			if lsvc, ok := svc.(interface{ GetFunctionNames() []string }); ok {
				for _, fn := range lsvc.GetFunctionNames() {
					addNode("lambda:"+fn, fn, "lambda", "function", "Compute")
					lambdaFunctions[fn] = true
				}
			}

		case "dynamodb":
			if dsvc, ok := svc.(interface{ GetTableNames() []string }); ok {
				for _, t := range dsvc.GetTableNames() {
					group := tableGroups[t]
					if group == "" {
						group = "Core Data"
					}
					addNode("dynamodb:"+t, t, "dynamodb", "table", group)
					dynamoTables[t] = true
				}
			}

		case "sqs":
			if qsvc, ok := svc.(interface{ GetQueueNames() []string }); ok {
				for _, q := range qsvc.GetQueueNames() {
					addNode("sqs:"+q, q, "sqs", "queue", "Messaging")
					sqsQueues[q] = true
				}
			}

		case "sns":
			if ssvc, ok := svc.(interface{ GetAllTopics() []string }); ok {
				for _, arn := range ssvc.GetAllTopics() {
					name := arnLastPart(arn)
					addNode("sns:"+name, name, "sns", "topic", "Messaging")
					snsTopics[name] = true
				}
			}

		case "s3":
			if bsvc, ok := svc.(interface{ GetBucketNames() []string }); ok {
				for _, b := range bsvc.GetBucketNames() {
					addNode("s3:"+b, b, "s3", "bucket", "Storage")
				}
			}

		case "cognito-idp":
			addNode("cognito:user-pool", "Cognito User Pool", "cognito-idp", "userpool", "Auth")

		case "events":
			if ebsvc, ok := svc.(interface{ GetAllEventBuses() []string }); ok {
				for _, bus := range ebsvc.GetAllEventBuses() {
					addNode("eventbridge:"+bus, bus+" bus", "events", "eventbus", "Messaging")
				}
			}

		case "monitoring":
			addNode("cloudwatch:alarms", "CloudWatch Alarms", "monitoring", "alarm", "Monitoring")

		case "logs":
			addNode("logs:log-groups", "Log Groups", "logs", "loggroup", "Monitoring")

		case "kms":
			addNode("kms:keys", "KMS Keys", "kms", "key", "Security")

		case "secretsmanager":
			addNode("secrets:store", "Secrets Manager", "secretsmanager", "secret", "Security")

		case "ssm":
			addNode("ssm:params", "SSM Parameters", "ssm", "parameter", "Security")

		case "iam":
			addNode("iam:roles", "IAM Roles", "iam", "role", "Auth")

		case "sts":
			addNode("sts:identity", "STS", "sts", "identity", "Auth")

		case "ses":
			addNode("ses:email", "SES Email", "ses", "email", "Messaging")

		case "rds":
			addNode("rds:databases", "RDS Databases", "rds", "database", "Storage")

		case "cloudformation":
			addNode("cfn:stacks", "CloudFormation", "cloudformation", "stack", "Security")

		case "apigateway":
			addNode("apigw:apis", "API Gateway", "apigateway", "api", "API")
		}
	}

	// 5. Lambda event source mappings -> trigger edges
	for _, svc := range svcs {
		if svc.Name() != "lambda" {
			continue
		}
		type esmGetter interface {
			GetEventSourceMappings() []*struct {
				UUID           string
				EventSourceArn string
				FunctionArn    string
				FunctionName   string
			}
		}
		// Use a more flexible type assertion
		if lsvc, ok := svc.(interface {
			GetEventSourceMappings() interface{}
		}); ok {
			_ = lsvc
		}
		// Try the concrete interface
		if lsvc, ok := svc.(interface {
			GetFunctionNames() []string
			GetEventSourceMappingsForTopology() ([]string, []string, []string)
		}); ok {
			_ = lsvc
		}
		// Simplest approach: use the lambda package types via registry + type assertion on raw slice
		break
	}

	// Use a separate helper to query ESMs without importing the lambda package
	a.addLambdaESMEdges(addEdge)

	// 6. SNS subscriptions -> subscribe edges
	a.addSNSSubscriptionEdges(addEdge, lambdaFunctions, sqsQueues)

	// 7. EventBridge rules -> rule edges
	a.addEventBridgeEdges(addEdge, lambdaFunctions, sqsQueues, snsTopics)

	// 8. CloudWatch alarm actions -> alarm edges
	a.addCloudWatchAlarmEdges(addEdge, snsTopics)

	// 9. CloudFormation stack dependencies
	a.addCloudFormationEdges(addEdge)

	// 10. Traffic-based edges from request log
	a.addTrafficEdges(addEdge, lambdaFunctions, dynamoTables, sqsQueues)

	// 11. All service relationship edges come from IaC config (pushed via /api/topology/config).
	// No hardcoded edges — the topology is derived from Pulumi/Terraform definitions.

	// Enrich edges with latency stats from request log
	if a.log != nil {
		enrichEdgesWithLatency(edges, a.log)
	}

	return TopologyResponseV2{
		Nodes:  nodes,
		Edges:  edges,
		Groups: topologyGroups,
	}
}

// enrichEdgesWithLatency computes average latency and call count per service
// from the request log and attaches them to matching edges.
func enrichEdgesWithLatency(edges []TopologyEdgeV2, log *gateway.RequestLog) {
	entries := log.Recent("", 1000)

	// Compute per-service stats: average latency and call count
	type svcStats struct {
		totalLatency float64
		count        int
	}
	stats := make(map[string]*svcStats)
	for _, e := range entries {
		svc := e.Service
		if svc == "" {
			continue
		}
		s, ok := stats[svc]
		if !ok {
			s = &svcStats{}
			stats[svc] = s
		}
		s.totalLatency += float64(e.Latency.Milliseconds())
		s.count++
	}

	// Apply stats to edges whose target matches a service
	for i := range edges {
		edge := &edges[i]
		// Extract service name from target node ID (e.g., "dynamodb:enterprise" → "dynamodb")
		targetSvc := ""
		if idx := strings.Index(edge.Target, ":"); idx > 0 {
			targetSvc = edge.Target[:idx]
		}

		if s, ok := stats[targetSvc]; ok && s.count > 0 {
			edge.AvgLatencyMs = s.totalLatency / float64(s.count)
			edge.CallCount = s.count
		}
	}
}

// addLambdaESMEdges queries Lambda ESMs and creates trigger edges.
func (a *API) addLambdaESMEdges(addEdge func(string, string, string, string, string)) {
	svc, err := a.registry.Lookup("lambda")
	if err != nil {
		return
	}

	type esmData struct {
		EventSourceArn string
		FunctionName   string
	}

	type esmProvider interface {
		GetEventSourceMappingsData() []esmData
	}

	// The lambda.LambdaService exposes GetEventSourceMappings() []*EventSourceMapping
	// We use a generic interface to avoid importing the lambda package.
	type rawESMProvider interface {
		GetEventSourceMappingsRaw() []map[string]string
	}

	// Try a reflection-free approach: type assert to get a method that returns
	// something we can iterate. The lambda service has:
	//   GetEventSourceMappings() []*EventSourceMapping
	// We need EventSourceArn and FunctionName from each.

	// Use the most generic possible assertion.
	type esmItem interface {
		GetESMFields() (eventSourceArn, functionName string)
	}

	// Since we can't import lambda, use the registry + known method pattern.
	// The cleanest way: call the method via interface assertion to get raw data.

	// Interface that lambda.LambdaService now satisfies:
	type lambdaESMAccess interface {
		GetEventSourceMappingsSummary() (arns []string, funcNames []string)
	}

	if lsvc, ok := svc.(lambdaESMAccess); ok {
		arns, funcNames := lsvc.GetEventSourceMappingsSummary()
		for i := range arns {
			if i >= len(funcNames) {
				break
			}
			arn := arns[i]
			fn := funcNames[i]

			// Determine source service and name from ARN
			sourceID := arnToNodeID(arn)
			if sourceID == "" {
				continue
			}

			addEdge(sourceID, "lambda:"+fn, "trigger", "event source mapping", "esm")
		}
	}
}

// addSNSSubscriptionEdges queries SNS subscriptions and creates edges.
func (a *API) addSNSSubscriptionEdges(addEdge func(string, string, string, string, string), lambdaFns, sqsQueues map[string]bool) {
	svc, err := a.registry.Lookup("sns")
	if err != nil {
		return
	}

	type subData struct {
		TopicArn string
		Protocol string
		Endpoint string
	}

	type snsSubProvider interface {
		GetSubscriptionsSummary() (topicArns, protocols, endpoints []string)
	}

	if ssvc, ok := svc.(snsSubProvider); ok {
		topicArns, protocols, endpoints := ssvc.GetSubscriptionsSummary()
		for i := range topicArns {
			topicName := arnLastPart(topicArns[i])
			sourceID := "sns:" + topicName
			proto := protocols[i]
			endpoint := endpoints[i]

			switch proto {
			case "sqs":
				qName := arnLastPart(endpoint)
				addEdge(sourceID, "sqs:"+qName, "subscribe", "subscription", "subscription")
			case "lambda":
				fnName := arnLastPart(endpoint)
				addEdge(sourceID, "lambda:"+fnName, "subscribe", "subscription", "subscription")
			case "http", "https":
				// External endpoint
				addEdge(sourceID, "external:bff-service", "subscribe", "webhook", "subscription")
			}
		}
	}
}

// addEventBridgeEdges queries EventBridge rules and creates edges.
func (a *API) addEventBridgeEdges(addEdge func(string, string, string, string, string), lambdaFns, sqsQueues, snsTopics map[string]bool) {
	svc, err := a.registry.Lookup("events")
	if err != nil {
		return
	}

	type ruleTargetProvider interface {
		GetRuleTargetsSummary() (ruleNames, targetArns []string)
	}

	if ebsvc, ok := svc.(ruleTargetProvider); ok {
		ruleNames, targetArns := ebsvc.GetRuleTargetsSummary()
		for i := range ruleNames {
			targetArn := targetArns[i]
			targetID := arnToNodeID(targetArn)
			if targetID == "" {
				continue
			}
			// Rules originate from the default event bus for simplicity
			addEdge("eventbridge:default", targetID, "trigger", "rule: "+ruleNames[i], "rule")
		}
	}
}

// addCloudWatchAlarmEdges queries CloudWatch alarms and creates alarm -> SNS edges.
func (a *API) addCloudWatchAlarmEdges(addEdge func(string, string, string, string, string), snsTopics map[string]bool) {
	svc, err := a.registry.Lookup("monitoring")
	if err != nil {
		return
	}

	type alarmProvider interface {
		GetAlarmActionsSummary() (alarmNames, actionArns []string)
	}

	if cwsvc, ok := svc.(alarmProvider); ok {
		alarmNames, actionArns := cwsvc.GetAlarmActionsSummary()
		for i := range alarmNames {
			targetID := arnToNodeID(actionArns[i])
			if targetID == "" {
				continue
			}
			addEdge("cloudwatch:alarms", targetID, "publish", "alarm: "+alarmNames[i], "alarm")
		}
	}
}

// addCloudFormationEdges queries CloudFormation for stack resource dependencies.
func (a *API) addCloudFormationEdges(addEdge func(string, string, string, string, string)) {
	svc, err := a.registry.Lookup("cloudformation")
	if err != nil {
		return
	}

	type cfnProvider interface {
		GetStackResourcesSummary() (stackNames []string, resourceTypes [][]string, logicalIDs [][]string)
	}

	if cfnSvc, ok := svc.(cfnProvider); ok {
		stackNames, resourceTypes, logicalIDs := cfnSvc.GetStackResourcesSummary()
		for i, stackName := range stackNames {
			if i >= len(resourceTypes) {
				break
			}
			for j, resType := range resourceTypes[i] {
				logicalID := ""
				if j < len(logicalIDs[i]) {
					logicalID = logicalIDs[i][j]
				}
				targetID := cfnResourceToNodeID(resType, logicalID)
				if targetID == "" {
					continue
				}
				addEdge("cfn:stacks", targetID, "provision", "stack: "+stackName, "cfn")
			}
		}
	}
}

// addTrafficEdges discovers service-to-resource edges from observed request
// traffic. Uses two strategies:
//  1. Trace correlation: requests sharing a TraceID are linked as caller→callee
//  2. Request analysis: extracts the specific resource (table, queue, bucket)
//     from each request's action/path to build precise edges
//
// This automatically discovers edges like:
//   lambda:attendance-handler → dynamodb:attendance (from Invoke → Query trace)
//   lambda:order-handler → dynamodb:order (from PutItem request body)
//   bff-service → dynamodb:featureFlag (from Query with TableName)
func (a *API) addTrafficEdges(addEdge func(string, string, string, string, string), lambdaFns, dynamoTables, sqsQueues map[string]bool) {
	if a.log == nil {
		return
	}

	entries := a.log.Recent("", 1000)

	// Group requests by TraceID to find caller→callee relationships
	traceGroups := make(map[string][]gateway.RequestEntry)
	for _, e := range entries {
		if e.TraceID != "" {
			traceGroups[e.TraceID] = append(traceGroups[e.TraceID], e)
		}
	}

	type edgeKey struct{ from, to string }
	seen := make(map[edgeKey]int) // count of observations

	// Strategy 1: Trace-correlated edges
	// Within a trace, the first request is the caller, subsequent requests are callees
	for _, group := range traceGroups {
		if len(group) < 2 {
			continue
		}
		// Sort by timestamp (entries are already newest-first, reverse for chronological)
		sorted := make([]gateway.RequestEntry, len(group))
		copy(sorted, group)
		for i, j := 0, len(sorted)-1; i < j; i, j = i+1, j-1 {
			sorted[i], sorted[j] = sorted[j], sorted[i]
		}

		callerID := requestToNodeID(sorted[0], lambdaFns, dynamoTables, sqsQueues)
		for _, callee := range sorted[1:] {
			calleeID := requestToNodeID(callee, lambdaFns, dynamoTables, sqsQueues)
			if callerID != "" && calleeID != "" && callerID != calleeID {
				k := edgeKey{callerID, calleeID}
				seen[k]++
			}
		}
	}

	// Strategy 2: Per-request resource extraction
	// Each request to DynamoDB/SQS/S3/RDS includes the specific resource name
	for _, e := range entries {
		resourceID := extractResourceNodeID(e, dynamoTables, sqsQueues)
		if resourceID == "" {
			continue
		}
		// The caller is either a Lambda function (if CallerID matches) or the BFF
		callerID := ""
		if e.CallerID != "" {
			// Try to match caller to a known Lambda function name
			for fn := range lambdaFns {
				if strings.Contains(e.CallerID, fn) || strings.Contains(fn, e.CallerID) {
					callerID = "lambda:" + fn
					break
				}
			}
		}
		if callerID == "" {
			callerID = "external:bff-service" // default caller
		}
		if callerID != resourceID {
			k := edgeKey{callerID, resourceID}
			seen[k]++
		}
	}

	// Emit edges with call counts
	for k, count := range seen {
		label := "observed"
		if count > 1 {
			label = fmt.Sprintf("%d calls", count)
		}
		addEdge(k.from, k.to, "read_write", label, "traffic")
	}
}

// requestToNodeID maps a request entry to its topology node ID.
func requestToNodeID(e gateway.RequestEntry, lambdaFns, dynamoTables, sqsQueues map[string]bool) string {
	switch e.Service {
	case "lambda":
		// Extract function name from action or path
		if strings.Contains(e.Action, "Invoke") {
			name := extractLambdaName(e)
			if name != "" && lambdaFns[name] {
				return "lambda:" + name
			}
		}
		return "lambda:" + e.Action
	case "dynamodb":
		return extractResourceNodeID(e, dynamoTables, sqsQueues)
	case "sqs":
		return extractResourceNodeID(e, dynamoTables, sqsQueues)
	case "s3":
		return extractResourceNodeID(e, dynamoTables, sqsQueues)
	case "cognito-idp":
		return "cognito:user-pool"
	case "rds":
		return "rds:databases"
	case "secretsmanager":
		return "secrets:store"
	case "ses":
		return "ses:email"
	default:
		return ""
	}
}

// extractResourceNodeID extracts the specific resource (table/queue/bucket) from
// a request's body or path.
func extractResourceNodeID(e gateway.RequestEntry, dynamoTables, sqsQueues map[string]bool) string {
	switch e.Service {
	case "dynamodb":
		// DynamoDB requests include TableName in the JSON body
		tableName := extractJSONField(e.RequestBody, "TableName")
		if tableName != "" && dynamoTables[tableName] {
			return "dynamodb:" + tableName
		}
		// Fallback: try action name (e.g., "Query" doesn't help, but "CreateTable" might)
		return ""
	case "sqs":
		// SQS queue name from URL path or QueueUrl parameter
		for q := range sqsQueues {
			if strings.Contains(e.Path, q) || strings.Contains(e.RequestBody, q) {
				return "sqs:" + q
			}
		}
		return ""
	case "s3":
		// S3 bucket from path: /{bucket}/{key}
		path := strings.TrimPrefix(e.Path, "/")
		if idx := strings.Index(path, "/"); idx > 0 {
			return "s3:" + path[:idx]
		}
		if path != "" {
			return "s3:" + path
		}
		return ""
	case "rds":
		return "rds:databases"
	default:
		return ""
	}
}

// extractLambdaName tries to extract the Lambda function name from a request.
func extractLambdaName(e gateway.RequestEntry) string {
	// Path format: /2015-03-31/functions/{name}/invocations
	path := e.Path
	if strings.Contains(path, "/functions/") {
		parts := strings.Split(path, "/functions/")
		if len(parts) > 1 {
			name := strings.Split(parts[1], "/")[0]
			return name
		}
	}
	return ""
}

// extractJSONField extracts a simple string field from a JSON body.
// Uses simple string scanning to avoid json.Unmarshal overhead.
func extractJSONField(body, field string) string {
	key := `"` + field + `"`
	idx := strings.Index(body, key)
	if idx < 0 {
		return ""
	}
	// Find the value after the key
	rest := body[idx+len(key):]
	// Skip whitespace and colon
	rest = strings.TrimLeft(rest, " \t\n\r:")
	if len(rest) == 0 || rest[0] != '"' {
		return ""
	}
	rest = rest[1:]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// ---- Helpers ----

// arnLastPart extracts the last segment of an ARN (after the last : or /).
func arnLastPart(arn string) string {
	// Try slash first (for ARNs like arn:aws:events:...:event-bus/default)
	if idx := strings.LastIndex(arn, "/"); idx >= 0 {
		return arn[idx+1:]
	}
	// Fall back to colon
	if idx := strings.LastIndex(arn, ":"); idx >= 0 {
		return arn[idx+1:]
	}
	return arn
}

// arnToNodeID converts an ARN to a topology node ID.
func arnToNodeID(arn string) string {
	if arn == "" {
		return ""
	}
	parts := strings.SplitN(arn, ":", 6)
	if len(parts) < 6 {
		return ""
	}
	svcName := parts[2] // e.g. "sqs", "lambda", "sns", "events"
	resource := parts[5] // e.g. "queue-name" or "function:name"

	switch svcName {
	case "sqs":
		return "sqs:" + resource
	case "lambda":
		// arn:aws:lambda:region:account:function:name
		if strings.HasPrefix(resource, "function:") {
			return "lambda:" + resource[len("function:"):]
		}
		return "lambda:" + resource
	case "sns":
		return "sns:" + resource
	case "dynamodb":
		// arn:aws:dynamodb:region:account:table/name
		if strings.HasPrefix(resource, "table/") {
			return "dynamodb:" + resource[len("table/"):]
		}
		return "dynamodb:" + resource
	case "events":
		return "eventbridge:" + arnLastPart(arn)
	case "logs":
		return "logs:log-groups"
	case "s3":
		return "s3:" + resource
	default:
		return ""
	}
}

// cfnResourceToNodeID maps a CloudFormation resource type to a topology node ID.
func cfnResourceToNodeID(resType, logicalID string) string {
	switch resType {
	case "AWS::DynamoDB::Table":
		return "dynamodb:" + logicalID
	case "AWS::SQS::Queue":
		return "sqs:" + logicalID
	case "AWS::SNS::Topic":
		return "sns:" + logicalID
	case "AWS::Lambda::Function":
		return "lambda:" + logicalID
	case "AWS::S3::Bucket":
		return "s3:" + logicalID
	default:
		return ""
	}
}

