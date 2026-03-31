package elasticbeanstalk

import (
	"encoding/xml"
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

func xmlOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatXML,
	}, nil
}

func xmlErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatXML}, awsErr
}

type xmlResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

// ---- CreateApplication ----

type xmlCreateApplicationResponse struct {
	XMLName xml.Name            `xml:"CreateApplicationResponse"`
	Result  xmlApplicationResult `xml:"CreateApplicationResult"`
	Meta    xmlResponseMetadata  `xml:"ResponseMetadata"`
}

type xmlApplicationResult struct {
	Application xmlApplication `xml:"Application"`
}

type xmlApplication struct {
	ApplicationName string `xml:"ApplicationName"`
	ApplicationArn  string `xml:"ApplicationArn"`
	Description     string `xml:"Description"`
	DateCreated     string `xml:"DateCreated"`
	DateUpdated     string `xml:"DateUpdated"`
}

func handleCreateApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("ApplicationName")
	if name == "" {
		return xmlErr(service.ErrValidation("ApplicationName is required."))
	}

	app, err := store.CreateApplication(name, form.Get("Description"))
	if err != nil {
		return xmlErr(service.ErrAlreadyExists("Application", name))
	}

	return xmlOK(&xmlCreateApplicationResponse{
		Result: xmlApplicationResult{
			Application: xmlApplication{
				ApplicationName: app.ApplicationName,
				ApplicationArn:  app.ApplicationArn,
				Description:     app.Description,
				DateCreated:     app.DateCreated.Format(time.RFC3339),
				DateUpdated:     app.DateUpdated.Format(time.RFC3339),
			},
		},
		Meta: xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeApplications ----

type xmlDescribeApplicationsResponse struct {
	XMLName xml.Name                   `xml:"DescribeApplicationsResponse"`
	Result  xmlDescribeApplicationsResult `xml:"DescribeApplicationsResult"`
	Meta    xmlResponseMetadata           `xml:"ResponseMetadata"`
}

type xmlDescribeApplicationsResult struct {
	Applications xmlApplicationList `xml:"Applications"`
}

type xmlApplicationList struct {
	Members []xmlApplication `xml:"member"`
}

func handleDescribeApplications(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	apps := store.ListApplications()
	members := make([]xmlApplication, 0, len(apps))
	for _, app := range apps {
		members = append(members, xmlApplication{
			ApplicationName: app.ApplicationName,
			ApplicationArn:  app.ApplicationArn,
			Description:     app.Description,
			DateCreated:     app.DateCreated.Format(time.RFC3339),
			DateUpdated:     app.DateUpdated.Format(time.RFC3339),
		})
	}
	return xmlOK(&xmlDescribeApplicationsResponse{
		Result: xmlDescribeApplicationsResult{
			Applications: xmlApplicationList{Members: members},
		},
		Meta: xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DeleteApplication ----

type xmlDeleteApplicationResponse struct {
	XMLName xml.Name            `xml:"DeleteApplicationResponse"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteApplication(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("ApplicationName")
	if name == "" {
		return xmlErr(service.ErrValidation("ApplicationName is required."))
	}
	store.DeleteApplication(name)
	return xmlOK(&xmlDeleteApplicationResponse{Meta: xmlResponseMetadata{RequestID: newUUID()}})
}

// ---- CreateApplicationVersion ----

type xmlCreateAppVersionResponse struct {
	XMLName xml.Name             `xml:"CreateApplicationVersionResponse"`
	Result  xmlAppVersionResult  `xml:"CreateApplicationVersionResult"`
	Meta    xmlResponseMetadata  `xml:"ResponseMetadata"`
}

type xmlAppVersionResult struct {
	ApplicationVersion xmlAppVersion `xml:"ApplicationVersion"`
}

type xmlAppVersion struct {
	ApplicationName string `xml:"ApplicationName"`
	VersionLabel    string `xml:"VersionLabel"`
	Description     string `xml:"Description"`
	Status          string `xml:"Status"`
	DateCreated     string `xml:"DateCreated"`
	DateUpdated     string `xml:"DateUpdated"`
}

func handleCreateApplicationVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	appName := form.Get("ApplicationName")
	versionLabel := form.Get("VersionLabel")
	if appName == "" || versionLabel == "" {
		return xmlErr(service.ErrValidation("ApplicationName and VersionLabel are required."))
	}

	ver, err := store.CreateApplicationVersion(appName, versionLabel,
		form.Get("Description"),
		form.Get("SourceBundle.S3Bucket"),
		form.Get("SourceBundle.S3Key"))
	if err != nil {
		return xmlErr(service.ErrNotFound("Application", appName))
	}

	return xmlOK(&xmlCreateAppVersionResponse{
		Result: xmlAppVersionResult{
			ApplicationVersion: xmlAppVersion{
				ApplicationName: ver.ApplicationName,
				VersionLabel:    ver.VersionLabel,
				Description:     ver.Description,
				Status:          ver.Status,
				DateCreated:     ver.DateCreated.Format(time.RFC3339),
				DateUpdated:     ver.DateUpdated.Format(time.RFC3339),
			},
		},
		Meta: xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeApplicationVersions ----

type xmlDescribeAppVersionsResponse struct {
	XMLName xml.Name                   `xml:"DescribeApplicationVersionsResponse"`
	Result  xmlDescribeAppVersionsResult `xml:"DescribeApplicationVersionsResult"`
	Meta    xmlResponseMetadata          `xml:"ResponseMetadata"`
}

type xmlDescribeAppVersionsResult struct {
	ApplicationVersions xmlAppVersionList `xml:"ApplicationVersions"`
}

type xmlAppVersionList struct {
	Members []xmlAppVersion `xml:"member"`
}

func handleDescribeApplicationVersions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	appName := form.Get("ApplicationName")

	var versions []*ApplicationVersion
	if appName != "" {
		versions = store.ListApplicationVersions(appName)
	}

	members := make([]xmlAppVersion, 0)
	for _, v := range versions {
		members = append(members, xmlAppVersion{
			ApplicationName: v.ApplicationName,
			VersionLabel:    v.VersionLabel,
			Description:     v.Description,
			Status:          v.Status,
			DateCreated:     v.DateCreated.Format(time.RFC3339),
			DateUpdated:     v.DateUpdated.Format(time.RFC3339),
		})
	}

	return xmlOK(&xmlDescribeAppVersionsResponse{
		Result: xmlDescribeAppVersionsResult{
			ApplicationVersions: xmlAppVersionList{Members: members},
		},
		Meta: xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- CreateEnvironment ----

type xmlCreateEnvironmentResponse struct {
	XMLName xml.Name            `xml:"CreateEnvironmentResponse"`
	Result  xmlEnvironment      `xml:"CreateEnvironmentResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlEnvironment struct {
	EnvironmentId     string `xml:"EnvironmentId"`
	EnvironmentName   string `xml:"EnvironmentName"`
	ApplicationName   string `xml:"ApplicationName"`
	VersionLabel      string `xml:"VersionLabel"`
	Description       string `xml:"Description"`
	EndpointURL       string `xml:"EndpointURL"`
	CNAME             string `xml:"CNAME"`
	Status            string `xml:"Status"`
	Health            string `xml:"Health"`
	HealthStatus      string `xml:"HealthStatus"`
	SolutionStackName string `xml:"SolutionStackName"`
	DateCreated       string `xml:"DateCreated"`
	DateUpdated       string `xml:"DateUpdated"`
}

func envToXML(env *Environment) xmlEnvironment {
	return xmlEnvironment{
		EnvironmentId:     env.EnvironmentID,
		EnvironmentName:   env.EnvironmentName,
		ApplicationName:   env.ApplicationName,
		VersionLabel:      env.VersionLabel,
		Description:       env.Description,
		EndpointURL:       env.EndpointURL,
		CNAME:             env.CNAME,
		Status:            env.Status,
		Health:            env.Health,
		HealthStatus:      env.HealthStatus,
		SolutionStackName: env.SolutionStackName,
		DateCreated:       env.DateCreated.Format(time.RFC3339),
		DateUpdated:       env.DateUpdated.Format(time.RFC3339),
	}
}

func handleCreateEnvironment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	appName := form.Get("ApplicationName")
	envName := form.Get("EnvironmentName")
	if appName == "" || envName == "" {
		return xmlErr(service.ErrValidation("ApplicationName and EnvironmentName are required."))
	}

	tier := EnvironmentTier{
		Name:    form.Get("Tier.Name"),
		Type:    form.Get("Tier.Type"),
		Version: form.Get("Tier.Version"),
	}

	env, err := store.CreateEnvironment(appName, envName,
		form.Get("VersionLabel"), form.Get("Description"),
		form.Get("SolutionStackName"), form.Get("TemplateName"), tier)
	if err != nil {
		return xmlErr(service.NewAWSError("InvalidParameterValue", err.Error(), http.StatusBadRequest))
	}

	return xmlOK(&xmlCreateEnvironmentResponse{
		Result: envToXML(env),
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeEnvironments ----

type xmlDescribeEnvironmentsResponse struct {
	XMLName xml.Name                   `xml:"DescribeEnvironmentsResponse"`
	Result  xmlDescribeEnvironmentsResult `xml:"DescribeEnvironmentsResult"`
	Meta    xmlResponseMetadata           `xml:"ResponseMetadata"`
}

type xmlDescribeEnvironmentsResult struct {
	Environments xmlEnvironmentList `xml:"Environments"`
}

type xmlEnvironmentList struct {
	Members []xmlEnvironment `xml:"member"`
}

func handleDescribeEnvironments(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	appName := form.Get("ApplicationName")

	envs := store.ListEnvironments(appName)
	members := make([]xmlEnvironment, 0, len(envs))
	for _, env := range envs {
		members = append(members, envToXML(env))
	}

	return xmlOK(&xmlDescribeEnvironmentsResponse{
		Result: xmlDescribeEnvironmentsResult{
			Environments: xmlEnvironmentList{Members: members},
		},
		Meta: xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- TerminateEnvironment ----

type xmlTerminateEnvironmentResponse struct {
	XMLName xml.Name            `xml:"TerminateEnvironmentResponse"`
	Result  xmlEnvironment      `xml:"TerminateEnvironmentResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleTerminateEnvironment(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	envName := form.Get("EnvironmentName")
	if envName == "" {
		return xmlErr(service.ErrValidation("EnvironmentName is required."))
	}

	env, err := store.TerminateEnvironment(envName)
	if err != nil {
		return xmlErr(service.ErrNotFound("Environment", envName))
	}

	return xmlOK(&xmlTerminateEnvironmentResponse{
		Result: envToXML(env),
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- CreateConfigurationTemplate ----

type xmlCreateConfigTemplateResponse struct {
	XMLName xml.Name              `xml:"CreateConfigurationTemplateResponse"`
	Result  xmlConfigTemplateResult `xml:"CreateConfigurationTemplateResult"`
	Meta    xmlResponseMetadata     `xml:"ResponseMetadata"`
}

type xmlConfigTemplateResult struct {
	ApplicationName   string `xml:"ApplicationName"`
	TemplateName      string `xml:"TemplateName"`
	Description       string `xml:"Description"`
	SolutionStackName string `xml:"SolutionStackName"`
	DateCreated       string `xml:"DateCreated"`
	DateUpdated       string `xml:"DateUpdated"`
}

func handleCreateConfigurationTemplate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	appName := form.Get("ApplicationName")
	templateName := form.Get("TemplateName")
	if appName == "" || templateName == "" {
		return xmlErr(service.ErrValidation("ApplicationName and TemplateName are required."))
	}

	tmpl, err := store.CreateConfigurationTemplate(appName, templateName,
		form.Get("Description"), form.Get("SolutionStackName"), form.Get("PlatformArn"))
	if err != nil {
		return xmlErr(service.ErrNotFound("Application", appName))
	}

	return xmlOK(&xmlCreateConfigTemplateResponse{
		Result: xmlConfigTemplateResult{
			ApplicationName:   tmpl.ApplicationName,
			TemplateName:      tmpl.TemplateName,
			Description:       tmpl.Description,
			SolutionStackName: tmpl.SolutionStackName,
			DateCreated:       tmpl.DateCreated.Format(time.RFC3339),
			DateUpdated:       tmpl.DateUpdated.Format(time.RFC3339),
		},
		Meta: xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DescribeConfigurationSettings ----

type xmlDescribeConfigSettingsResponse struct {
	XMLName xml.Name            `xml:"DescribeConfigurationSettingsResponse"`
	Result  xmlConfigSettingsResult `xml:"DescribeConfigurationSettingsResult"`
	Meta    xmlResponseMetadata     `xml:"ResponseMetadata"`
}

type xmlConfigSettingsResult struct {
	ConfigurationSettings xmlConfigSettingsList `xml:"ConfigurationSettings"`
}

type xmlConfigSettingsList struct {
	Members []xmlConfigSettings `xml:"member"`
}

type xmlConfigSettings struct {
	ApplicationName   string `xml:"ApplicationName"`
	TemplateName      string `xml:"TemplateName"`
	Description       string `xml:"Description"`
	SolutionStackName string `xml:"SolutionStackName"`
	DateCreated       string `xml:"DateCreated"`
	DateUpdated       string `xml:"DateUpdated"`
}

func handleDescribeConfigurationSettings(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	appName := form.Get("ApplicationName")
	templateName := form.Get("TemplateName")

	var members []xmlConfigSettings
	if appName != "" && templateName != "" {
		tmpl, ok := store.GetConfigurationTemplate(appName, templateName)
		if ok {
			members = append(members, xmlConfigSettings{
				ApplicationName:   tmpl.ApplicationName,
				TemplateName:      tmpl.TemplateName,
				Description:       tmpl.Description,
				SolutionStackName: tmpl.SolutionStackName,
				DateCreated:       tmpl.DateCreated.Format(time.RFC3339),
				DateUpdated:       tmpl.DateUpdated.Format(time.RFC3339),
			})
		}
	}

	return xmlOK(&xmlDescribeConfigSettingsResponse{
		Result: xmlConfigSettingsResult{
			ConfigurationSettings: xmlConfigSettingsList{Members: members},
		},
		Meta: xmlResponseMetadata{RequestID: newUUID()},
	})
}

// ---- DeleteConfigurationTemplate ----

type xmlDeleteConfigTemplateResponse struct {
	XMLName xml.Name            `xml:"DeleteConfigurationTemplateResponse"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteConfigurationTemplate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	appName := form.Get("ApplicationName")
	templateName := form.Get("TemplateName")
	if appName == "" || templateName == "" {
		return xmlErr(service.ErrValidation("ApplicationName and TemplateName are required."))
	}
	store.DeleteConfigurationTemplate(appName, templateName)
	return xmlOK(&xmlDeleteConfigTemplateResponse{Meta: xmlResponseMetadata{RequestID: newUUID()}})
}
