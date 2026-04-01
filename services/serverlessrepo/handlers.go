package serverlessrepo

import (
	"net/http"
	"strings"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
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

func handleCreateApplication(params map[string]any, store *Store) (*service.Response, error) {
	name := str(params, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required"))
	}
	author := str(params, "Author")
	if author == "" {
		return jsonErr(service.ErrValidation("Author is required"))
	}

	app, _ := store.CreateApplication(name, str(params, "Description"), author,
		str(params, "SpdxLicenseId"), str(params, "HomePageUrl"),
		str(params, "SemanticVersion"), strSlice(params, "Labels"))

	return jsonCreated(map[string]any{
		"ApplicationId":   app.ApplicationID,
		"Name":            app.Name,
		"Description":     app.Description,
		"Author":          app.Author,
		"SpdxLicenseId":   app.SpdxLicenseID,
		"Labels":          app.Labels,
		"HomePageUrl":     app.HomePageURL,
		"SemanticVersion": app.SemanticVersion,
		"CreationTime":    app.CreationTime.Format(time.RFC3339),
	})
}

func handleGetApplication(appID string, store *Store) (*service.Response, error) {
	app, ok := store.GetApplication(appID)
	if !ok {
		return jsonErr(service.ErrNotFound("Application", appID))
	}
	return jsonOK(map[string]any{
		"ApplicationId":   app.ApplicationID,
		"Name":            app.Name,
		"Description":     app.Description,
		"Author":          app.Author,
		"SpdxLicenseId":   app.SpdxLicenseID,
		"Labels":          app.Labels,
		"HomePageUrl":     app.HomePageURL,
		"SemanticVersion": app.SemanticVersion,
		"CreationTime":    app.CreationTime.Format(time.RFC3339),
	})
}

func handleListApplications(store *Store) (*service.Response, error) {
	apps := store.ListApplications()
	out := make([]map[string]any, 0, len(apps))
	for _, app := range apps {
		out = append(out, map[string]any{
			"ApplicationId": app.ApplicationID,
			"Name":          app.Name,
			"Description":   app.Description,
			"Author":        app.Author,
		})
	}
	return jsonOK(map[string]any{"Applications": out})
}

func handleDeleteApplication(appID string, store *Store) (*service.Response, error) {
	if !store.DeleteApplication(appID) {
		return jsonErr(service.ErrNotFound("Application", appID))
	}
	return jsonNoContent()
}

func handleCreateApplicationVersion(appID, version string, params map[string]any, store *Store) (*service.Response, error) {
	ver, err := store.CreateApplicationVersion(appID, version, str(params, "SourceCodeUrl"))
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "invalid semantic version") {
			return jsonErr(service.ErrValidation(errMsg))
		}
		if strings.Contains(errMsg, "already exists") {
			return jsonErr(service.ErrAlreadyExists("ApplicationVersion", version))
		}
		return jsonErr(service.ErrNotFound("Application", appID))
	}
	return jsonCreated(map[string]any{
		"ApplicationId":   ver.ApplicationID,
		"SemanticVersion": ver.SemanticVersion,
		"TemplateUrl":     ver.TemplateURL,
		"SourceCodeUrl":   ver.SourceCodeURL,
		"CreationTime":    ver.CreationTime.Format(time.RFC3339),
	})
}

func handleListApplicationVersions(appID string, store *Store) (*service.Response, error) {
	versions := store.ListApplicationVersions(appID)
	out := make([]map[string]any, 0, len(versions))
	for _, v := range versions {
		out = append(out, map[string]any{
			"ApplicationId":   v.ApplicationID,
			"SemanticVersion": v.SemanticVersion,
			"CreationTime":    v.CreationTime.Format(time.RFC3339),
		})
	}
	return jsonOK(map[string]any{"Versions": out})
}

func handleCreateChangeSet(appID string, params map[string]any, store *Store) (*service.Response, error) {
	stackName := str(params, "StackName")
	if stackName == "" {
		stackName = "serverlessrepo-" + appID
	}
	cs, err := store.CreateCloudFormationChangeSet(appID, str(params, "SemanticVersion"), stackName)
	if err != nil {
		return jsonErr(service.ErrNotFound("Application", appID))
	}
	return jsonCreated(map[string]any{
		"ApplicationId":   cs.ApplicationID,
		"ChangeSetId":     cs.ChangeSetID,
		"SemanticVersion": cs.SemanticVersion,
		"StackId":         cs.StackID,
	})
}
