package airflow

import (
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
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

func getInt(p map[string]any, k string) int {
	if v, ok := p[k]; ok {
		if n, ok := v.(float64); ok {
			return int(n)
		}
	}
	return 0
}

func getMap(p map[string]any, k string) map[string]any {
	if v, ok := p[k]; ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return nil
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

func handleCreateEnvironment(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	env, awsErr := store.CreateEnvironment(
		name,
		getStr(p, "AirflowVersion"),
		getStr(p, "SourceBucketArn"),
		getStr(p, "DagS3Path"),
		getStr(p, "EnvironmentClass"),
		getInt(p, "MaxWorkers"),
		getInt(p, "MinWorkers"),
		getInt(p, "Schedulers"),
		getStr(p, "ExecutionRoleArn"),
		getStr(p, "WebserverAccessMode"),
		getMap(p, "NetworkConfiguration"),
		getMap(p, "LoggingConfiguration"),
		getStrMap(p, "AirflowConfigurationOptions"),
		getStr(p, "WeeklyMaintenanceWindowStart"),
		getStrMap(p, "Tags"),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Arn": env.Arn})
}

func handleGetEnvironment(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	env, awsErr := store.GetEnvironment(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Environment": envToMap(env)})
}

func handleListEnvironments(store *Store) (*service.Response, error) {
	names := store.ListEnvironments()
	return jsonOK(map[string]any{"Environments": names})
}

func handleUpdateEnvironment(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	env, awsErr := store.UpdateEnvironment(
		name,
		getStr(p, "AirflowVersion"),
		getStr(p, "SourceBucketArn"),
		getStr(p, "DagS3Path"),
		getStr(p, "EnvironmentClass"),
		getInt(p, "MaxWorkers"),
		getInt(p, "MinWorkers"),
		getInt(p, "Schedulers"),
		getStr(p, "ExecutionRoleArn"),
		getStr(p, "WebserverAccessMode"),
		getMap(p, "NetworkConfiguration"),
		getMap(p, "LoggingConfiguration"),
		getStrMap(p, "AirflowConfigurationOptions"),
		getStr(p, "WeeklyMaintenanceWindowStart"),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Arn": env.Arn})
}

func handleDeleteEnvironment(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	if awsErr := store.DeleteEnvironment(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleCreateCliToken(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	token, webserverUrl, awsErr := store.CreateCliToken(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"CliToken":          token,
		"WebServerHostname": webserverUrl,
	})
}

func handleCreateWebLoginToken(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	token, webserverUrl, awsErr := store.CreateWebLoginToken(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"WebToken":          token,
		"WebServerHostname": webserverUrl,
	})
}

func handleTagResource(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	tags := getStrMap(p, "Tags")
	if awsErr := store.TagResource(arn, tags); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleUntagResource(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	tagKeys := getStringSlice(p, "tagKeys")
	if awsErr := store.UntagResource(arn, tagKeys); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleListTagsForResource(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	tags, awsErr := store.ListTagsForResource(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{"Tags": tags})
}

func envToMap(env *Environment) map[string]any {
	return map[string]any{
		"Name":                           env.Name,
		"Arn":                            env.Arn,
		"Status":                         string(env.Status),
		"AirflowVersion":                 env.AirflowVersion,
		"SourceBucketArn":                env.SourceBucketArn,
		"DagS3Path":                      env.DagS3Path,
		"EnvironmentClass":               env.EnvironmentClass,
		"MaxWorkers":                     env.MaxWorkers,
		"MinWorkers":                     env.MinWorkers,
		"Schedulers":                     env.Schedulers,
		"ExecutionRoleArn":               env.ExecutionRoleArn,
		"WebserverAccessMode":            env.WebserverAccessMode,
		"WebserverUrl":                   env.WebserverUrl,
		"NetworkConfiguration":           env.NetworkConfiguration,
		"LoggingConfiguration":           env.LoggingConfiguration,
		"AirflowConfigurationOptions":    env.AirflowConfigurationOptions,
		"WeeklyMaintenanceWindowStart":   env.WeeklyMaintenanceWindowStart,
		"ServiceRoleArn":                 env.ServiceRoleArn,
		"CreatedAt":                      env.CreatedAt.Format(time.RFC3339),
		"LastUpdate":                     map[string]any{"CreatedAt": env.LastUpdate.Format(time.RFC3339), "Status": string(env.Status)},
		"Tags":                           env.Tags,
	}
}
