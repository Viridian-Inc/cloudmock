package batch

import (
	"net/http"
	"strings"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func str(params map[string]any, key string) string {
	if params == nil {
		return ""
	}
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

func ceResponse(ce *ComputeEnvironment) map[string]any {
	resp := map[string]any{
		"computeEnvironmentName": ce.ComputeEnvironmentName,
		"computeEnvironmentArn":  ce.ComputeEnvironmentArn,
		"type":                   ce.Type,
		"state":                  ce.State,
		"status":                 ce.Status,
		"statusReason":           ce.StatusReason,
		"serviceRole":            ce.ServiceRole,
	}
	if ce.ComputeResources != nil {
		resp["computeResources"] = map[string]any{
			"type":             ce.ComputeResources.Type,
			"minvCpus":         ce.ComputeResources.MinvCpus,
			"maxvCpus":         ce.ComputeResources.MaxvCpus,
			"desiredvCpus":     ce.ComputeResources.DesiredvCpus,
			"instanceTypes":    ce.ComputeResources.InstanceTypes,
			"subnets":          ce.ComputeResources.Subnets,
			"securityGroupIds": ce.ComputeResources.SecurityGroupIds,
		}
	}
	return resp
}

func handleCreateComputeEnvironment(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "computeEnvironmentName")
	if name == "" {
		return jsonErr(service.ErrValidation("computeEnvironmentName is required"))
	}

	ceType := str(params, "type")
	if ceType == "" {
		ceType = "MANAGED"
	}
	state := str(params, "state")
	if state == "" {
		state = "ENABLED"
	}

	var resources *ComputeResource
	if cr, ok := params["computeResources"].(map[string]any); ok {
		resources = &ComputeResource{
			Type:             str(cr, "type"),
			MinvCpus:         num(cr, "minvCpus", 0),
			MaxvCpus:         num(cr, "maxvCpus", 256),
			DesiredvCpus:     num(cr, "desiredvCpus", 0),
			InstanceTypes:    strSlice(cr, "instanceTypes"),
			Subnets:          strSlice(cr, "subnets"),
			SecurityGroupIds: strSlice(cr, "securityGroupIds"),
			InstanceRole:     str(cr, "instanceRole"),
		}
	}

	ce, err := store.CreateComputeEnvironment(name, ceType, state, str(params, "serviceRole"), resources)
	if err != nil {
		return jsonErr(service.ErrAlreadyExists("ComputeEnvironment", name))
	}
	return jsonOK(map[string]any{
		"computeEnvironmentName": ce.ComputeEnvironmentName,
		"computeEnvironmentArn":  ce.ComputeEnvironmentArn,
	})
}

func handleDescribeComputeEnvironments(params map[string]any, store *Store) (*service.Response, error) {
	envs := store.ListComputeEnvironments()
	names := strSlice(params, "computeEnvironments")
	if len(names) > 0 {
		nameSet := make(map[string]bool)
		for _, n := range names {
			nameSet[n] = true
		}
		filtered := make([]*ComputeEnvironment, 0)
		for _, ce := range envs {
			if nameSet[ce.ComputeEnvironmentName] || nameSet[ce.ComputeEnvironmentArn] {
				filtered = append(filtered, ce)
			}
		}
		envs = filtered
	}

	out := make([]map[string]any, 0, len(envs))
	for _, ce := range envs {
		out = append(out, ceResponse(ce))
	}
	return jsonOK(map[string]any{"computeEnvironments": out})
}

func handleDeleteComputeEnvironment(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "computeEnvironment")
	if name == "" {
		return jsonErr(service.ErrValidation("computeEnvironment is required"))
	}
	if !store.DeleteComputeEnvironment(name) {
		return jsonErr(service.ErrNotFound("ComputeEnvironment", name))
	}
	return jsonOK(map[string]any{})
}

func handleCreateJobQueue(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "jobQueueName")
	if name == "" {
		return jsonErr(service.ErrValidation("jobQueueName is required"))
	}

	state := str(params, "state")
	if state == "" {
		state = "ENABLED"
	}
	priority := num(params, "priority", 1)

	var ceOrder []ComputeEnvironmentOrder
	if ceos, ok := params["computeEnvironmentOrder"].([]any); ok {
		for _, ceo := range ceos {
			if cm, ok := ceo.(map[string]any); ok {
				ceOrder = append(ceOrder, ComputeEnvironmentOrder{
					ComputeEnvironment: str(cm, "computeEnvironment"),
					Order:              num(cm, "order", 0),
				})
			}
		}
	}

	jq, err := store.CreateJobQueue(name, state, priority, ceOrder)
	if err != nil {
		return jsonErr(service.ErrAlreadyExists("JobQueue", name))
	}
	return jsonOK(map[string]any{
		"jobQueueName": jq.JobQueueName,
		"jobQueueArn":  jq.JobQueueArn,
	})
}

func handleDescribeJobQueues(params map[string]any, store *Store) (*service.Response, error) {
	queues := store.ListJobQueues()
	out := make([]map[string]any, 0, len(queues))
	for _, jq := range queues {
		ceOrders := make([]map[string]any, 0, len(jq.ComputeEnvironmentOrder))
		for _, ceo := range jq.ComputeEnvironmentOrder {
			ceOrders = append(ceOrders, map[string]any{
				"computeEnvironment": ceo.ComputeEnvironment,
				"order":              ceo.Order,
			})
		}
		out = append(out, map[string]any{
			"jobQueueName":            jq.JobQueueName,
			"jobQueueArn":             jq.JobQueueArn,
			"state":                   jq.State,
			"status":                  jq.Status,
			"priority":                jq.Priority,
			"computeEnvironmentOrder": ceOrders,
		})
	}
	return jsonOK(map[string]any{"jobQueues": out})
}

func handleDeleteJobQueue(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "jobQueue")
	if name == "" {
		return jsonErr(service.ErrValidation("jobQueue is required"))
	}
	if !store.DeleteJobQueue(name) {
		return jsonErr(service.ErrNotFound("JobQueue", name))
	}
	return jsonOK(map[string]any{})
}

func handleRegisterJobDefinition(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "jobDefinitionName")
	if name == "" {
		return jsonErr(service.ErrValidation("jobDefinitionName is required"))
	}
	jobType := str(params, "type")
	if jobType == "" {
		jobType = "container"
	}

	var container *ContainerProperties
	if cp, ok := params["containerProperties"].(map[string]any); ok {
		container = &ContainerProperties{
			Image:      str(cp, "image"),
			Vcpus:      num(cp, "vcpus", 1),
			Memory:     num(cp, "memory", 512),
			Command:    strSlice(cp, "command"),
			JobRoleArn: str(cp, "jobRoleArn"),
		}
		if envVars, ok := cp["environment"].([]any); ok {
			for _, ev := range envVars {
				if em, ok := ev.(map[string]any); ok {
					container.Environment = append(container.Environment, KeyValuePair{
						Name:  str(em, "name"),
						Value: str(em, "value"),
					})
				}
			}
		}
	}

	var retry *RetryStrategy
	if rs, ok := params["retryStrategy"].(map[string]any); ok {
		retry = &RetryStrategy{Attempts: num(rs, "attempts", 1)}
	}

	var timeout *JobTimeout
	if to, ok := params["timeout"].(map[string]any); ok {
		timeout = &JobTimeout{AttemptDurationSeconds: num(to, "attemptDurationSeconds", 0)}
	}

	jd, _ := store.RegisterJobDefinition(name, jobType, container, retry, timeout)
	return jsonOK(map[string]any{
		"jobDefinitionName": jd.JobDefinitionName,
		"jobDefinitionArn":  jd.JobDefinitionArn,
		"revision":          jd.Revision,
	})
}

func handleDescribeJobDefinitions(params map[string]any, store *Store) (*service.Response, error) {
	defs := store.ListJobDefinitions()
	out := make([]map[string]any, 0, len(defs))
	for _, jd := range defs {
		entry := map[string]any{
			"jobDefinitionName": jd.JobDefinitionName,
			"jobDefinitionArn":  jd.JobDefinitionArn,
			"revision":          jd.Revision,
			"type":              jd.Type,
			"status":            jd.Status,
		}
		if jd.ContainerProperties != nil {
			entry["containerProperties"] = map[string]any{
				"image":  jd.ContainerProperties.Image,
				"vcpus":  jd.ContainerProperties.Vcpus,
				"memory": jd.ContainerProperties.Memory,
			}
		}
		out = append(out, entry)
	}
	return jsonOK(map[string]any{"jobDefinitions": out})
}

func handleDeregisterJobDefinition(params map[string]any, store *Store) (*service.Response, error) {
	defArn := str(params, "jobDefinition")
	if defArn == "" {
		return jsonErr(service.ErrValidation("jobDefinition is required"))
	}
	if !store.DeregisterJobDefinition(defArn) {
		return jsonErr(service.ErrNotFound("JobDefinition", defArn))
	}
	return jsonOK(map[string]any{})
}

func handleSubmitJob(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "jobName")
	queue := str(params, "jobQueue")
	jobDef := str(params, "jobDefinition")
	if name == "" || queue == "" || jobDef == "" {
		return jsonErr(service.ErrValidation("jobName, jobQueue, and jobDefinition are required"))
	}

	job, _ := store.SubmitJob(name, queue, jobDef, nil, nil, nil)
	return jsonOK(map[string]any{
		"jobName": job.JobName,
		"jobId":   job.JobID,
		"jobArn":  job.JobArn,
	})
}

func handleDescribeJobs(params map[string]any, store *Store) (*service.Response, error) {
	jobIDs := strSlice(params, "jobs")
	out := make([]map[string]any, 0)
	for _, id := range jobIDs {
		job, ok := store.GetJob(id)
		if !ok {
			continue
		}
		entry := map[string]any{
			"jobName":       job.JobName,
			"jobId":         job.JobID,
			"jobArn":        job.JobArn,
			"jobQueue":      job.JobQueue,
			"jobDefinition": job.JobDefinition,
			"status":        job.Status,
			"statusReason":  job.StatusReason,
			"createdAt":     job.CreatedAt.Unix(),
		}
		if job.StartedAt != nil {
			entry["startedAt"] = job.StartedAt.Unix()
		}
		if job.StoppedAt != nil {
			entry["stoppedAt"] = job.StoppedAt.Unix()
		}
		out = append(out, entry)
	}
	return jsonOK(map[string]any{"jobs": out})
}

func handleListJobs(params map[string]any, store *Store) (*service.Response, error) {
	queue := str(params, "jobQueue")
	status := str(params, "jobStatus")

	jobs := store.ListJobs(queue, status)
	out := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		entry := map[string]any{
			"jobId":      job.JobID,
			"jobName":    job.JobName,
			"status":     job.Status,
			"createdAt":  job.CreatedAt.Unix(),
		}
		if job.StartedAt != nil {
			entry["startedAt"] = job.StartedAt.Unix()
		}
		if job.StoppedAt != nil {
			entry["stoppedAt"] = job.StoppedAt.Unix()
		}
		out = append(out, entry)
	}
	return jsonOK(map[string]any{"jobSummaryList": out})
}

func handleCancelJob(params map[string]any, store *Store) (*service.Response, error) {
	jobID := str(params, "jobId")
	reason := str(params, "reason")
	if jobID == "" {
		return jsonErr(service.ErrValidation("jobId is required"))
	}
	if err := store.CancelJob(jobID, reason); err != nil {
		return jsonErr(service.NewAWSError("ClientException", err.Error(), http.StatusBadRequest))
	}
	return jsonOK(map[string]any{})
}

func handleTerminateJob(params map[string]any, store *Store) (*service.Response, error) {
	jobID := str(params, "jobId")
	reason := str(params, "reason")
	if jobID == "" {
		return jsonErr(service.ErrValidation("jobId is required"))
	}
	if err := store.TerminateJob(jobID, reason); err != nil {
		return jsonErr(service.NewAWSError("ClientException", err.Error(), http.StatusBadRequest))
	}
	return jsonOK(map[string]any{})
}

func handleUpdateComputeEnvironment(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "computeEnvironment")
	if name == "" {
		return jsonErr(service.ErrValidation("computeEnvironment is required"))
	}
	ce, ok := store.UpdateComputeEnvironment(name, str(params, "state"), str(params, "serviceRole"))
	if !ok {
		return jsonErr(service.ErrNotFound("ComputeEnvironment", name))
	}
	return jsonOK(map[string]any{
		"computeEnvironmentName": ce.ComputeEnvironmentName,
		"computeEnvironmentArn":  ce.ComputeEnvironmentArn,
	})
}

func handleUpdateJobQueue(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "jobQueue")
	if name == "" {
		return jsonErr(service.ErrValidation("jobQueue is required"))
	}
	var ceOrder []ComputeEnvironmentOrder
	if ceos, ok := params["computeEnvironmentOrder"].([]any); ok {
		for _, ceo := range ceos {
			if cm, ok := ceo.(map[string]any); ok {
				ceOrder = append(ceOrder, ComputeEnvironmentOrder{
					ComputeEnvironment: str(cm, "computeEnvironment"),
					Order:              num(cm, "order", 0),
				})
			}
		}
	}
	jq, ok := store.UpdateJobQueue(name, str(params, "state"), num(params, "priority", 0), ceOrder)
	if !ok {
		return jsonErr(service.ErrNotFound("JobQueue", name))
	}
	return jsonOK(map[string]any{
		"jobQueueName": jq.JobQueueName,
		"jobQueueArn":  jq.JobQueueArn,
	})
}

func handleCreateSchedulingPolicy(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "name")
	if name == "" {
		return jsonErr(service.ErrValidation("name is required"))
	}
	var fsp *FairsharePolicy
	if fp, ok := params["fairsharePolicy"].(map[string]any); ok {
		fsp = &FairsharePolicy{
			ComputeReservation: num(fp, "computeReservation", 0),
			ShareDecaySeconds:  num(fp, "shareDecaySeconds", 0),
		}
		if sds, ok := fp["shareDistributions"].([]any); ok {
			for _, sd := range sds {
				if sdm, ok := sd.(map[string]any); ok {
					wf := 0.0
					if v, ok := sdm["weightFactor"].(float64); ok {
						wf = v
					}
					fsp.ShareDistributions = append(fsp.ShareDistributions, ShareDistribution{
						ShareIdentifier: str(sdm, "shareIdentifier"),
						WeightFactor:    wf,
					})
				}
			}
		}
	}
	var tags map[string]string
	if t, ok := params["tags"].(map[string]any); ok {
		tags = make(map[string]string)
		for k, v := range t {
			if sv, ok := v.(string); ok {
				tags[k] = sv
			}
		}
	}
	sp, err := store.CreateSchedulingPolicy(name, fsp, tags)
	if err != nil {
		return jsonErr(service.ErrAlreadyExists("SchedulingPolicy", name))
	}
	return jsonOK(map[string]any{"arn": sp.Arn})
}

func handleDescribeSchedulingPolicies(params map[string]any, store *Store) (*service.Response, error) {
	arns := strSlice(params, "arns")
	policies := store.DescribeSchedulingPolicies(arns)
	out := make([]map[string]any, 0, len(policies))
	for _, sp := range policies {
		out = append(out, map[string]any{"arn": sp.Arn, "name": sp.Name})
	}
	return jsonOK(map[string]any{"schedulingPolicies": out})
}

func handleUpdateSchedulingPolicy(params map[string]any, store *Store) (*service.Response, error) {
	arn := str(params, "arn")
	if arn == "" {
		return jsonErr(service.ErrValidation("arn is required"))
	}
	var fsp *FairsharePolicy
	if fp, ok := params["fairsharePolicy"].(map[string]any); ok {
		fsp = &FairsharePolicy{
			ComputeReservation: num(fp, "computeReservation", 0),
			ShareDecaySeconds:  num(fp, "shareDecaySeconds", 0),
		}
	}
	if !store.UpdateSchedulingPolicy(arn, fsp) {
		return jsonErr(service.ErrNotFound("SchedulingPolicy", arn))
	}
	return jsonOK(map[string]any{})
}

func handleDeleteSchedulingPolicy(params map[string]any, store *Store) (*service.Response, error) {
	arn := str(params, "arn")
	if arn == "" {
		return jsonErr(service.ErrValidation("arn is required"))
	}
	if !store.DeleteSchedulingPolicy(arn) {
		return jsonErr(service.ErrNotFound("SchedulingPolicy", arn))
	}
	return jsonOK(map[string]any{})
}

func handleTagResource(path string, params map[string]any, store *Store) (*service.Response, error) {
	// path: /v1/tags/{resourceArn}
	resourceARN := strings.TrimPrefix(path, "/v1/tags/")
	if resourceARN == "" {
		return jsonErr(service.ErrValidation("resourceArn is required"))
	}
	tags := make(map[string]string)
	if t, ok := params["tags"].(map[string]any); ok {
		for k, v := range t {
			if sv, ok := v.(string); ok {
				tags[k] = sv
			}
		}
	}
	store.TagResource(resourceARN, tags)
	return jsonOK(map[string]any{})
}

func handleUntagResource(path string, params map[string]any, store *Store) (*service.Response, error) {
	resourceARN := strings.TrimPrefix(path, "/v1/tags/")
	if resourceARN == "" {
		return jsonErr(service.ErrValidation("resourceArn is required"))
	}
	tagKeys := strSlice(params, "tagKeys")
	store.UntagResource(resourceARN, tagKeys)
	return jsonOK(map[string]any{})
}

func handleListTagsForResource(path string, store *Store) (*service.Response, error) {
	resourceARN := strings.TrimPrefix(path, "/v1/tags/")
	if resourceARN == "" {
		return jsonErr(service.ErrValidation("resourceArn is required"))
	}
	tags := store.ListTagsForResource(resourceARN)
	return jsonOK(map[string]any{"tags": tags})
}

// unused but required to satisfy the import for time package in build
var _ = time.Now
