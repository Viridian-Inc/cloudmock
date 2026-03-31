package dms

import (
	"net/http"

	"github.com/neureaux/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func str(params map[string]any, key string) string {
	if v, ok := params[key].(string); ok {
		return v
	}
	return ""
}

func num(params map[string]any, key string, def int) int {
	if v, ok := params[key].(float64); ok {
		return int(v)
	}
	return def
}

func boolVal(params map[string]any, key string) bool {
	if v, ok := params[key].(bool); ok {
		return v
	}
	return false
}

func strSlice(params map[string]any, key string) []string {
	if v, ok := params[key].([]any); ok {
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func instanceResponse(inst *ReplicationInstance) map[string]any {
	return map[string]any{
		"ReplicationInstanceIdentifier": inst.ReplicationInstanceIdentifier,
		"ReplicationInstanceArn":        inst.ReplicationInstanceArn,
		"ReplicationInstanceClass":      inst.ReplicationInstanceClass,
		"AllocatedStorage":              inst.AllocatedStorage,
		"ReplicationInstanceStatus":     inst.ReplicationInstanceStatus,
		"EngineVersion":                 inst.EngineVersion,
		"AutoMinorVersionUpgrade":       inst.AutoMinorVersionUpgrade,
		"AvailabilityZone":              inst.AvailabilityZone,
		"MultiAZ":                       inst.MultiAZ,
		"PubliclyAccessible":            inst.PubliclyAccessible,
		"InstanceCreateTime":            inst.InstanceCreateTime,
	}
}

func endpointResponse(ep *Endpoint) map[string]any {
	return map[string]any{
		"EndpointIdentifier": ep.EndpointIdentifier,
		"EndpointArn":        ep.EndpointArn,
		"EndpointType":       ep.EndpointType,
		"EngineName":         ep.EngineName,
		"ServerName":         ep.ServerName,
		"Port":               ep.Port,
		"DatabaseName":       ep.DatabaseName,
		"Username":           ep.Username,
		"Status":             ep.Status,
	}
}

func taskResponse(task *ReplicationTask) map[string]any {
	resp := map[string]any{
		"ReplicationTaskIdentifier": task.ReplicationTaskIdentifier,
		"ReplicationTaskArn":        task.ReplicationTaskArn,
		"SourceEndpointArn":         task.SourceEndpointArn,
		"TargetEndpointArn":         task.TargetEndpointArn,
		"ReplicationInstanceArn":    task.ReplicationInstanceArn,
		"MigrationType":             task.MigrationType,
		"TableMappings":             task.TableMappings,
		"Status":                    task.Status,
	}
	if task.StartedAt != nil {
		resp["ReplicationTaskStartDate"] = task.StartedAt
	}
	return resp
}

func subscriptionResponse(sub *EventSubscription) map[string]any {
	return map[string]any{
		"CustSubscriptionId": sub.CustSubscriptionId,
		"SnsTopicArn":        sub.SnsTopicArn,
		"SourceType":         sub.SourceType,
		"SourceIdsList":      sub.SourceIds,
		"EventCategoriesList": sub.EventCategories,
		"Status":             sub.Status,
	}
}

// ---- Replication Instances ----

func handleCreateReplicationInstance(params map[string]any, store *Store) (*service.Response, error) {
	id := str(params, "ReplicationInstanceIdentifier")
	if id == "" {
		return jsonErr(service.ErrValidation("ReplicationInstanceIdentifier is required"))
	}
	class := str(params, "ReplicationInstanceClass")
	if class == "" {
		class = "dms.t3.medium"
	}
	storage := num(params, "AllocatedStorage", 50)
	engineVersion := str(params, "EngineVersion")
	if engineVersion == "" {
		engineVersion = "3.5.1"
	}
	az := str(params, "AvailabilityZone")
	multiAZ := boolVal(params, "MultiAZ")
	publicAccess := boolVal(params, "PubliclyAccessible")

	inst, err := store.CreateReplicationInstance(id, class, storage, engineVersion, az, multiAZ, publicAccess)
	if err != nil {
		return jsonErr(service.ErrAlreadyExists("ReplicationInstance", id))
	}

	return jsonOK(map[string]any{"ReplicationInstance": instanceResponse(inst)})
}

func handleDescribeReplicationInstances(store *Store) (*service.Response, error) {
	instances := store.ListReplicationInstances()
	out := make([]map[string]any, 0, len(instances))
	for _, inst := range instances {
		out = append(out, instanceResponse(inst))
	}
	return jsonOK(map[string]any{"ReplicationInstances": out})
}

func handleDeleteReplicationInstance(params map[string]any, store *Store) (*service.Response, error) {
	id := str(params, "ReplicationInstanceArn")
	// Extract identifier from ARN or use directly
	if id == "" {
		return jsonErr(service.ErrValidation("ReplicationInstanceArn is required"))
	}
	// Try looking up by ARN - check all instances
	instances := store.ListReplicationInstances()
	for _, inst := range instances {
		if inst.ReplicationInstanceArn == id {
			deleted, _ := store.DeleteReplicationInstance(inst.ReplicationInstanceIdentifier)
			if deleted != nil {
				return jsonOK(map[string]any{"ReplicationInstance": instanceResponse(deleted)})
			}
		}
	}
	return jsonErr(service.ErrNotFound("ReplicationInstance", id))
}

// ---- Endpoints ----

func handleCreateEndpoint(params map[string]any, store *Store) (*service.Response, error) {
	id := str(params, "EndpointIdentifier")
	if id == "" {
		return jsonErr(service.ErrValidation("EndpointIdentifier is required"))
	}
	endpointType := str(params, "EndpointType")
	if endpointType == "" {
		return jsonErr(service.ErrValidation("EndpointType is required"))
	}
	engine := str(params, "EngineName")
	if engine == "" {
		return jsonErr(service.ErrValidation("EngineName is required"))
	}

	ep, err := store.CreateEndpoint(id, endpointType, engine,
		str(params, "ServerName"), num(params, "Port", 3306),
		str(params, "DatabaseName"), str(params, "Username"))
	if err != nil {
		return jsonErr(service.ErrAlreadyExists("Endpoint", id))
	}
	return jsonOK(map[string]any{"Endpoint": endpointResponse(ep)})
}

func handleDescribeEndpoints(store *Store) (*service.Response, error) {
	endpoints := store.ListEndpoints()
	out := make([]map[string]any, 0, len(endpoints))
	for _, ep := range endpoints {
		out = append(out, endpointResponse(ep))
	}
	return jsonOK(map[string]any{"Endpoints": out})
}

func handleDeleteEndpoint(params map[string]any, store *Store) (*service.Response, error) {
	arn := str(params, "EndpointArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("EndpointArn is required"))
	}
	endpoints := store.ListEndpoints()
	for _, ep := range endpoints {
		if ep.EndpointArn == arn {
			store.DeleteEndpoint(ep.EndpointIdentifier)
			return jsonOK(map[string]any{"Endpoint": endpointResponse(ep)})
		}
	}
	return jsonErr(service.ErrNotFound("Endpoint", arn))
}

// ---- Replication Tasks ----

func handleCreateReplicationTask(params map[string]any, store *Store) (*service.Response, error) {
	id := str(params, "ReplicationTaskIdentifier")
	if id == "" {
		return jsonErr(service.ErrValidation("ReplicationTaskIdentifier is required"))
	}

	task, err := store.CreateReplicationTask(id,
		str(params, "SourceEndpointArn"),
		str(params, "TargetEndpointArn"),
		str(params, "ReplicationInstanceArn"),
		str(params, "MigrationType"),
		str(params, "TableMappings"))
	if err != nil {
		return jsonErr(service.ErrAlreadyExists("ReplicationTask", id))
	}
	return jsonOK(map[string]any{"ReplicationTask": taskResponse(task)})
}

func handleDescribeReplicationTasks(store *Store) (*service.Response, error) {
	tasks := store.ListReplicationTasks()
	out := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, taskResponse(task))
	}
	return jsonOK(map[string]any{"ReplicationTasks": out})
}

func handleStartReplicationTask(params map[string]any, store *Store) (*service.Response, error) {
	arn := str(params, "ReplicationTaskArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ReplicationTaskArn is required"))
	}
	task, err := store.StartReplicationTask(arn)
	if err != nil {
		return jsonErr(service.NewAWSError("InvalidResourceStateFault", err.Error(), http.StatusBadRequest))
	}
	return jsonOK(map[string]any{"ReplicationTask": taskResponse(task)})
}

func handleStopReplicationTask(params map[string]any, store *Store) (*service.Response, error) {
	arn := str(params, "ReplicationTaskArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ReplicationTaskArn is required"))
	}
	task, err := store.StopReplicationTask(arn)
	if err != nil {
		return jsonErr(service.NewAWSError("InvalidResourceStateFault", err.Error(), http.StatusBadRequest))
	}
	return jsonOK(map[string]any{"ReplicationTask": taskResponse(task)})
}

func handleDeleteReplicationTask(params map[string]any, store *Store) (*service.Response, error) {
	arn := str(params, "ReplicationTaskArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ReplicationTaskArn is required"))
	}
	if !store.DeleteReplicationTask(arn) {
		return jsonErr(service.ErrNotFound("ReplicationTask", arn))
	}
	return jsonOK(map[string]any{"ReplicationTask": map[string]any{"Status": "deleting"}})
}

// ---- Event Subscriptions ----

func handleCreateEventSubscription(params map[string]any, store *Store) (*service.Response, error) {
	id := str(params, "SubscriptionName")
	if id == "" {
		return jsonErr(service.ErrValidation("SubscriptionName is required"))
	}

	sub, err := store.CreateEventSubscription(id,
		str(params, "SnsTopicArn"),
		str(params, "SourceType"),
		strSlice(params, "SourceIds"),
		strSlice(params, "EventCategories"))
	if err != nil {
		return jsonErr(service.ErrAlreadyExists("EventSubscription", id))
	}
	return jsonOK(map[string]any{"EventSubscription": subscriptionResponse(sub)})
}

func handleDescribeEventSubscriptions(store *Store) (*service.Response, error) {
	subs := store.ListEventSubscriptions()
	out := make([]map[string]any, 0, len(subs))
	for _, sub := range subs {
		out = append(out, subscriptionResponse(sub))
	}
	return jsonOK(map[string]any{"EventSubscriptionsList": out})
}

func handleDeleteEventSubscription(params map[string]any, store *Store) (*service.Response, error) {
	id := str(params, "SubscriptionName")
	if id == "" {
		return jsonErr(service.ErrValidation("SubscriptionName is required"))
	}
	sub, ok := store.GetEventSubscription(id)
	if !ok {
		return jsonErr(service.ErrNotFound("EventSubscription", id))
	}
	store.DeleteEventSubscription(id)
	return jsonOK(map[string]any{"EventSubscription": subscriptionResponse(sub)})
}
