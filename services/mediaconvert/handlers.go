package mediaconvert

import (
	"net/http"
	"strings"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonCreated(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusCreated, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func jsonNoContent() (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusNoContent, Format: service.FormatJSON}, nil
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

func jobResponse(j *Job) map[string]any {
	resp := map[string]any{
		"id":          j.ID,
		"arn":         j.Arn,
		"queue":       j.Queue,
		"role":        j.Role,
		"status":      j.Status,
		"createdAt":   j.CreatedAt.Unix(),
		"timing": map[string]any{
			"submitTime": j.SubmittedAt.Unix(),
		},
	}
	if j.FinishedAt != nil {
		resp["timing"].(map[string]any)["finishTime"] = j.FinishedAt.Unix()
	}
	return resp
}

func validateInputURIs(settings map[string]any) *service.AWSError {
	if settings == nil {
		return nil
	}
	inputs, ok := settings["inputs"].([]any)
	if !ok {
		return nil
	}
	for _, input := range inputs {
		if im, ok := input.(map[string]any); ok {
			fileInput := str(im, "fileInput")
			if fileInput != "" && !strings.HasPrefix(fileInput, "s3://") {
				return service.ErrValidation("Input file URI must start with s3://: " + fileInput)
			}
		}
	}
	return nil
}

func handleCreateJob(params map[string]any, store *Store) (*service.Response, error) {
	role := str(params, "role")
	queue := str(params, "queue")
	if queue == "" {
		queue = "Default"
	}
	settings, _ := params["settings"].(map[string]any)

	// Validate queue exists
	if _, ok := store.GetQueue(queue); !ok {
		return jsonErr(service.NewAWSError("NotFoundException",
			"Queue not found: "+queue, http.StatusNotFound))
	}

	// Validate input file URIs
	if awsErr := validateInputURIs(settings); awsErr != nil {
		return jsonErr(awsErr)
	}

	job, _ := store.CreateJob(queue, role, settings)
	return jsonCreated(map[string]any{"job": jobResponse(job)})
}

func handleGetJob(id string, store *Store) (*service.Response, error) {
	job, ok := store.GetJob(id)
	if !ok {
		return jsonErr(service.ErrNotFound("Job", id))
	}
	return jsonOK(map[string]any{"job": jobResponse(job)})
}

func handleListJobs(store *Store) (*service.Response, error) {
	jobs := store.ListJobs()
	out := make([]map[string]any, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, jobResponse(j))
	}
	return jsonOK(map[string]any{"jobs": out})
}

func handleCancelJob(id string, store *Store) (*service.Response, error) {
	if err := store.CancelJob(id); err != nil {
		return jsonErr(service.NewAWSError("ConflictException", err.Error(), http.StatusConflict))
	}
	return jsonNoContent()
}

func handleCreateJobTemplate(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "name")
	if name == "" {
		return jsonErr(service.ErrValidation("name is required"))
	}
	settings, _ := params["settings"].(map[string]any)
	tmpl, err := store.CreateJobTemplate(name, str(params, "description"), str(params, "category"), settings)
	if err != nil {
		return jsonErr(service.ErrAlreadyExists("JobTemplate", name))
	}
	return jsonCreated(map[string]any{
		"jobTemplate": map[string]any{
			"name":        tmpl.Name,
			"arn":         tmpl.Arn,
			"description": tmpl.Description,
			"category":    tmpl.Category,
			"type":        tmpl.Type,
			"createdAt":   tmpl.CreatedAt.Format(time.RFC3339),
		},
	})
}

func handleGetJobTemplate(name string, store *Store) (*service.Response, error) {
	tmpl, ok := store.GetJobTemplate(name)
	if !ok {
		return jsonErr(service.ErrNotFound("JobTemplate", name))
	}
	return jsonOK(map[string]any{
		"jobTemplate": map[string]any{
			"name":        tmpl.Name,
			"arn":         tmpl.Arn,
			"description": tmpl.Description,
			"category":    tmpl.Category,
			"type":        tmpl.Type,
		},
	})
}

func handleListJobTemplates(store *Store) (*service.Response, error) {
	templates := store.ListJobTemplates()
	out := make([]map[string]any, 0, len(templates))
	for _, t := range templates {
		out = append(out, map[string]any{
			"name":        t.Name,
			"arn":         t.Arn,
			"description": t.Description,
			"type":        t.Type,
		})
	}
	return jsonOK(map[string]any{"jobTemplates": out})
}

func handleDeleteJobTemplate(name string, store *Store) (*service.Response, error) {
	if !store.DeleteJobTemplate(name) {
		return jsonErr(service.ErrNotFound("JobTemplate", name))
	}
	return jsonNoContent()
}

func handleCreatePreset(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "name")
	if name == "" {
		return jsonErr(service.ErrValidation("name is required"))
	}
	settings, _ := params["settings"].(map[string]any)
	preset, err := store.CreatePreset(name, str(params, "description"), str(params, "category"), settings)
	if err != nil {
		return jsonErr(service.ErrAlreadyExists("Preset", name))
	}
	return jsonCreated(map[string]any{
		"preset": map[string]any{
			"name":        preset.Name,
			"arn":         preset.Arn,
			"description": preset.Description,
			"type":        preset.Type,
		},
	})
}

func handleGetPreset(name string, store *Store) (*service.Response, error) {
	preset, ok := store.GetPreset(name)
	if !ok {
		return jsonErr(service.ErrNotFound("Preset", name))
	}
	return jsonOK(map[string]any{
		"preset": map[string]any{
			"name":        preset.Name,
			"arn":         preset.Arn,
			"description": preset.Description,
			"type":        preset.Type,
		},
	})
}

func handleListPresets(store *Store) (*service.Response, error) {
	presets := store.ListPresets()
	out := make([]map[string]any, 0, len(presets))
	for _, p := range presets {
		out = append(out, map[string]any{
			"name":        p.Name,
			"arn":         p.Arn,
			"description": p.Description,
			"type":        p.Type,
		})
	}
	return jsonOK(map[string]any{"presets": out})
}

func handleDeletePreset(name string, store *Store) (*service.Response, error) {
	if !store.DeletePreset(name) {
		return jsonErr(service.ErrNotFound("Preset", name))
	}
	return jsonNoContent()
}

func handleCreateQueue(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "name")
	if name == "" {
		return jsonErr(service.ErrValidation("name is required"))
	}
	q, err := store.CreateQueue(name, str(params, "description"), str(params, "pricingPlan"))
	if err != nil {
		return jsonErr(service.ErrAlreadyExists("Queue", name))
	}
	return jsonCreated(map[string]any{
		"queue": map[string]any{
			"name":        q.Name,
			"arn":         q.Arn,
			"description": q.Description,
			"status":      q.Status,
			"pricingPlan": q.PricingPlan,
			"type":        q.Type,
		},
	})
}

func handleGetQueue(name string, store *Store) (*service.Response, error) {
	q, ok := store.GetQueue(name)
	if !ok {
		return jsonErr(service.ErrNotFound("Queue", name))
	}
	return jsonOK(map[string]any{
		"queue": map[string]any{
			"name":                 q.Name,
			"arn":                  q.Arn,
			"description":         q.Description,
			"status":              q.Status,
			"pricingPlan":         q.PricingPlan,
			"type":                q.Type,
			"submittedJobsCount":  q.SubmittedJobsCount,
			"progressingJobsCount": q.ProgressingJobsCount,
		},
	})
}

func handleListQueues(store *Store) (*service.Response, error) {
	queues := store.ListQueues()
	out := make([]map[string]any, 0, len(queues))
	for _, q := range queues {
		out = append(out, map[string]any{
			"name":   q.Name,
			"arn":    q.Arn,
			"status": q.Status,
			"type":   q.Type,
		})
	}
	return jsonOK(map[string]any{"queues": out})
}

func handleUpdateQueue(name string, params map[string]any, store *Store) (*service.Response, error) {
	description := str(params, "description")
	q, ok := store.UpdateQueue(name, description)
	if !ok {
		return jsonErr(service.ErrNotFound("Queue", name))
	}
	return jsonOK(map[string]any{
		"queue": map[string]any{
			"name":        q.Name,
			"arn":         q.Arn,
			"description": q.Description,
			"status":      q.Status,
			"type":        q.Type,
		},
	})
}

func handleDeleteQueue(name string, store *Store) (*service.Response, error) {
	if !store.DeleteQueue(name) {
		return jsonErr(service.ErrNotFound("Queue", name))
	}
	return jsonNoContent()
}
