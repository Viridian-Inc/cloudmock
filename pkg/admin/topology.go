package admin

import (
	"fmt"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/iac"
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

// TopologyTreeResponse is the response for GET /api/topology/tree.
type TopologyTreeResponse struct {
	Nodes           []iac.DependencyNode `json:"nodes"`
	Hierarchy       map[string][]string  `json:"hierarchy"`
	DependencyEdges []iac.DependencyEdge `json:"dependencyEdges"`
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

	nodeIDs := make(map[string]bool)
	addNode := func(id, label, svc, typ, group string) {
		if nodeIDs[id] {
			return
		}
		nodeIDs[id] = true
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
			if nodeIDs[n.ID] {
				continue
			}
			nodeIDs[n.ID] = true
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

		// Skip CloudFormation placeholder (no IaC resources provisioned)
		case "cloudformation":
			// Will appear when traffic hits it

		case "apigateway":
			addNode("apigw:apis", "API Gateway", "apigateway", "api", "API")

		default:
			// Only add services that have actual resources or traffic.
			// Skip idle services to reduce visual clutter and reflect real usage.
			if a.serviceHasActivity(svc.Name()) {
				name := svc.Name()
				label := serviceDisplayName(name)
				group := serviceGroup(name)
				nodeType := serviceNodeType(name)
				addNode(name+":service", label, name, nodeType, group)
			}
		}
	}

	// 4b. IaC-extracted microservices (Lambda endpoints with routes + table dependencies)
	for _, ms := range a.iacMicroservices {
		nodeID := "microservice:" + ms.Name
		// Skip if a Lambda node with the same name already exists
		if nodeIDs["lambda:"+ms.Name] {
			nodeID = "lambda:" + ms.Name // use existing Lambda node ID
		} else {
			addNode(nodeID, ms.Name, "lambda", "function", "Compute")
		}
		lambdaFunctions[ms.Name] = true

		// Create edges from microservice to its DynamoDB tables
		for _, table := range ms.Tables {
			tableID := "dynamodb:" + table
			if dynamoTables[table] {
				addEdge(nodeID, tableID, "read_write", "table access", "config")
			}
		}

		// Create edge from API Gateway to microservice
		addEdge("apigw:apis", nodeID, "trigger", fmt.Sprintf("%d routes", len(ms.Routes)), "config")

	}

	// 4c. Infrastructure edges — key architectural relationships only.
	// Avoid noisy edges (every function → IAM, every function → Logs) that clutter the graph.

	// Cognito → Lambda triggers (auth hooks)
	for fn := range lambdaFunctions {
		if strings.Contains(fn, "cognito") || strings.Contains(fn, "auth") || strings.Contains(fn, "authorizer") {
			// Use the node ID that actually exists (lambda: for discovered, microservice: for IaC)
			targetID := "lambda:" + fn
			if nodeIDs["microservice:"+fn] {
				targetID = "microservice:" + fn
			}
			addEdge("cognito:user-pool", targetID, "trigger", "Cognito trigger", "config")
		}
	}

	// API Gateway → Cognito authorizer
	addEdge("apigw:apis", "cognito:user-pool", "read_write", "authorizer", "config")

	// Monitoring chain: CloudWatch → SNS alerts
	for topic := range snsTopics {
		if strings.Contains(topic, "alert") {
			addEdge("cloudwatch:alarms", "sns:"+topic, "publish", "alarm action", "config")
		}
	}

	// Auth chain: IAM ↔ STS ↔ KMS
	addEdge("iam:roles", "sts:identity", "read_write", "assume role", "config")
	addEdge("kms:keys", "iam:roles", "read_write", "key policy", "config")

	// Secrets Manager — only services that actually use secrets
	for _, ms := range a.iacMicroservices {
		if ms.Name == "sso" || ms.Name == "billing" || ms.Name == "stripeWebhook" || ms.Name == "stripe_webhook" {
			addEdge("microservice:"+ms.Name, "secrets:store", "read_write", "get secret", "config")
		}
	}

	// EventBridge — connect event-driven services and standalone Lambdas
	addEdge("eventbridge:default", "logs:log-groups", "trigger", "event logging", "config")
	// Standalone Lambda functions (stream processors, scheduled tasks) triggered by EventBridge/DynamoDB Streams
	for fn := range lambdaFunctions {
		if strings.Contains(fn, "stream") || strings.Contains(fn, "sync") {
			addEdge("eventbridge:default", "lambda:"+fn, "trigger", "stream trigger", "config")
			// Stream processors write to DynamoDB
			for table := range dynamoTables {
				if strings.Contains(fn, "dynamodb") {
					addEdge("lambda:"+fn, "dynamodb:"+table, "read_write", "stream sync", "config")
					break // just one representative edge
				}
			}
		}
		if strings.Contains(fn, "expire") || strings.Contains(fn, "scheduled") || strings.Contains(fn, "cron") {
			addEdge("eventbridge:default", "lambda:"+fn, "trigger", "scheduled", "config")
		}
	}

	// Connect webhook/subscription tables to their services by matching table base names.
	// Table names may have environment suffixes (e.g. webhook-dev, webhook-prod).
	tableServiceMap := map[string]string{
		"webhook":         "webhook",
		"webhookDelivery": "webhook",
		"subscription":    "billing",
		"notification":    "notification",
	}
	for table := range dynamoTables {
		baseName := strings.TrimSuffix(table, "-"+a.environment())
		if svcName, ok := tableServiceMap[baseName]; ok {
			msID := "microservice:" + svcName
			if nodeIDs[msID] {
				addEdge(msID, "dynamodb:"+table, "read_write", "table access", "config")
			}
		}
	}

	// SES — notification service sends email
	for _, ms := range a.iacMicroservices {
		if ms.Name == "notification" {
			addEdge("microservice:"+ms.Name, "ses:email", "publish", "send email", "config")
		}
	}

	// Connect isolated DynamoDB tables to microservices by name matching
	for table := range dynamoTables {
		tableID := "dynamodb:" + table
		hasEdge := false
		for _, e := range edges {
			if e.Target == tableID || e.Source == tableID {
				hasEdge = true
				break
			}
		}
		if !hasEdge {
			// Try matching table name to a microservice
			baseName := strings.TrimSuffix(table, "-dev")
			baseName = strings.TrimSuffix(baseName, "-local")
			for _, ms := range a.iacMicroservices {
				if strings.EqualFold(ms.Name, baseName) || strings.Contains(strings.ToLower(ms.Name), strings.ToLower(baseName)) {
					addEdge("microservice:"+ms.Name, tableID, "read_write", "table access", "config")
					hasEdge = true
					break
				}
			}
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
			GetEventSourceMappings() any
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

	// Clean labels: strip project prefixes and env suffixes for display
	env := a.environment()
	for i := range nodes {
		nodes[i].Label = cleanNodeLabel(nodes[i].Label, env)
	}

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

// cleanNodeLabel strips common IaC prefixes and environment suffixes from node labels.
func cleanNodeLabel(label, env string) string {
	cleaned := label
	for _, prefix := range []string{"autotend-", "autotend_"} {
		cleaned = strings.TrimPrefix(cleaned, prefix)
	}
	if env != "" {
		cleaned = strings.TrimSuffix(cleaned, "-"+env)
		cleaned = strings.TrimSuffix(cleaned, "_"+env)
	}
	if len(cleaned) <= 1 {
		return label
	}
	return cleaned
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

// environment returns the configured IaC environment name, defaulting to "dev".
func (a *API) environment() string {
	if a.cfg != nil && a.cfg.IaCEnv != "" {
		return a.cfg.IaCEnv
	}
	return "dev"
}

// serviceHasActivity checks if a service has received any traffic in the request log.
func (a *API) serviceHasActivity(name string) bool {
	if a.log == nil {
		return false
	}
	entries := a.log.Recent(name, 1)
	return len(entries) > 0
}

// ---- Service metadata helpers ----

// serviceDisplayName returns a human-readable label for an AWS service.
var serviceDisplayNames = map[string]string{
	"acm": "ACM", "acm-pca": "ACM PCA", "appconfig": "AppConfig",
	"application-autoscaling": "App AutoScaling", "appsync": "AppSync",
	"athena": "Athena", "autoscaling": "Auto Scaling", "backup": "Backup",
	"batch": "Batch", "bedrock": "Bedrock", "ce": "Cost Explorer",
	"cloudcontrol": "Cloud Control", "cloudfront": "CloudFront",
	"cloudtrail": "CloudTrail", "codebuild": "CodeBuild",
	"codecommit": "CodeCommit", "codeconnections": "CodeConnections",
	"codedeploy": "CodeDeploy", "codepipeline": "CodePipeline",
	"codeartifact": "CodeArtifact", "config": "Config",
	"dms": "DMS", "docdb": "DocumentDB", "ec2": "EC2", "ecr": "ECR",
	"ecs": "ECS", "eks": "EKS", "elasticache": "ElastiCache",
	"elasticbeanstalk": "Elastic Beanstalk", "elasticloadbalancing": "ELB",
	"elasticmapreduce": "EMR", "es": "Elasticsearch", "fis": "FIS",
	"glacier": "Glacier", "glue": "Glue", "identitystore": "Identity Store",
	"iot": "IoT", "iot-data": "IoT Data", "iot-wireless": "IoT Wireless",
	"kafka": "MSK", "kinesis": "Kinesis", "kinesisanalytics": "Kinesis Analytics",
	"lakeformation": "Lake Formation", "managedblockchain": "Blockchain",
	"mediaconvert": "MediaConvert", "memorydb": "MemoryDB", "mq": "MQ",
	"neptune": "Neptune", "opensearch": "OpenSearch", "organizations": "Organizations",
	"pinpoint": "Pinpoint", "pipes": "EventBridge Pipes", "ram": "RAM",
	"redshift": "Redshift", "resource-groups": "Resource Groups",
	"route53": "Route 53", "route53resolver": "Route 53 Resolver",
	"s3tables": "S3 Tables", "sagemaker": "SageMaker", "scheduler": "Scheduler",
	"secretsmanager": "Secrets Manager", "serverlessrepo": "SAR",
	"servicediscovery": "Cloud Map", "shield": "Shield",
	"sso-admin": "SSO Admin", "support": "Support", "swf": "SWF",
	"tagging": "Resource Tagging", "textract": "Textract",
	"timestream-write": "Timestream", "transcribe": "Transcribe",
	"transfer": "Transfer", "verifiedpermissions": "Verified Permissions",
	"waf-regional": "WAF Regional", "wafv2": "WAFv2",
	"firehose": "Firehose", "stepfunctions": "Step Functions",
	"airflow": "MWAA",
}

func serviceDisplayName(name string) string {
	if label, ok := serviceDisplayNames[name]; ok {
		return label
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

// serviceGroup returns the topology group for a service.
var serviceGroupMap = map[string]string{
	// Compute
	"ec2": "Compute", "lambda": "Compute", "ecs": "Compute", "eks": "Compute",
	"batch": "Compute", "elasticbeanstalk": "Compute", "lightsail": "Compute",
	// Storage
	"s3": "Storage", "glacier": "Storage", "s3tables": "Storage", "backup": "Storage",
	"rds": "Storage", "dynamodb": "Storage", "docdb": "Storage", "neptune": "Storage",
	"elasticache": "Storage", "memorydb": "Storage", "redshift": "Storage",
	"timestream-write": "Storage", "dms": "Storage",
	// Messaging
	"sqs": "Messaging", "sns": "Messaging", "events": "Messaging", "pipes": "Messaging",
	"kinesis": "Messaging", "firehose": "Messaging", "kafka": "Messaging",
	"mq": "Messaging", "ses": "Messaging", "scheduler": "Messaging",
	// Auth
	"iam": "Auth", "sts": "Auth", "cognito-idp": "Auth", "sso-admin": "Auth",
	"identitystore": "Auth", "verifiedpermissions": "Auth", "organizations": "Auth",
	// Security
	"kms": "Security", "secretsmanager": "Security", "ssm": "Security",
	"acm": "Security", "acm-pca": "Security", "shield": "Security",
	"wafv2": "Security", "waf-regional": "Security", "ram": "Security",
	"cloudtrail": "Security", "config": "Security", "guardduty": "Security",
	// Monitoring
	"monitoring": "Monitoring", "logs": "Monitoring", "xray": "Monitoring",
	// API
	"apigateway": "API", "appsync": "API", "cloudfront": "API",
	"elasticloadbalancing": "API", "route53": "API", "route53resolver": "API",
	// CI/CD
	"codebuild": "Integrations", "codepipeline": "Integrations", "codedeploy": "Integrations",
	"codecommit": "Integrations", "codeconnections": "Integrations", "codeartifact": "Integrations",
	"cloudformation": "Integrations", "cloudcontrol": "Integrations",
	// ML/AI
	"sagemaker": "Admin", "bedrock": "Admin", "textract": "Admin",
	"transcribe": "Admin", "mediaconvert": "Admin",
	// App Services
	"appconfig": "Features", "servicediscovery": "Features",
	"swf": "Features", "stepfunctions": "Features", "airflow": "Features",
	// Other
	"iot": "Integrations", "iot-data": "Integrations", "iot-wireless": "Integrations",
	"glue": "Admin", "athena": "Admin", "lakeformation": "Admin",
	"opensearch": "Admin", "es": "Admin", "kinesisanalytics": "Admin",
	"fis": "Monitoring", "support": "Admin", "ce": "Admin",
	"tagging": "Security", "resource-groups": "Security",
	"pinpoint": "Features", "managedblockchain": "Integrations",
	"serverlessrepo": "Integrations", "transfer": "Integrations",
	"account": "Auth",
}

func serviceGroup(name string) string {
	if g, ok := serviceGroupMap[name]; ok {
		return g
	}
	return "Features"
}

// serviceNodeType returns the visual node type for a service.
func serviceNodeType(name string) string {
	switch {
	case strings.Contains(name, "lambda") || strings.Contains(name, "batch") || strings.Contains(name, "ecs") || strings.Contains(name, "eks"):
		return "compute"
	case strings.Contains(name, "sqs") || strings.Contains(name, "sns") || strings.Contains(name, "kinesis") || strings.Contains(name, "kafka"):
		return "queue"
	case strings.Contains(name, "s3") || strings.Contains(name, "glacier"):
		return "bucket"
	case strings.Contains(name, "rds") || strings.Contains(name, "dynamo") || strings.Contains(name, "docdb") || strings.Contains(name, "neptune") || strings.Contains(name, "redis") || strings.Contains(name, "elasticache"):
		return "database"
	case strings.Contains(name, "iam") || strings.Contains(name, "sts") || strings.Contains(name, "cognito") || strings.Contains(name, "sso"):
		return "identity"
	case strings.Contains(name, "waf") || strings.Contains(name, "shield") || strings.Contains(name, "kms") || strings.Contains(name, "acm"):
		return "security"
	case strings.Contains(name, "cloudwatch") || strings.Contains(name, "logs") || strings.Contains(name, "monitoring"):
		return "monitoring"
	case strings.Contains(name, "code") || strings.Contains(name, "pipeline"):
		return "cicd"
	default:
		return "service"
	}
}

// ---- ARN Helpers ----

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

