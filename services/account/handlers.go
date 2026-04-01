package account

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/neureaux/cloudmock/pkg/service"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
var phoneRegex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

// knownRegions is the set of all known AWS region names.
var knownRegions = map[string]bool{
	"us-east-1": true, "us-east-2": true, "us-west-1": true, "us-west-2": true,
	"eu-west-1": true, "eu-west-2": true, "eu-west-3": true, "eu-central-1": true, "eu-north-1": true,
	"eu-south-1": true, "eu-south-2": true, "eu-central-2": true,
	"ap-southeast-1": true, "ap-southeast-2": true, "ap-southeast-3": true,
	"ap-northeast-1": true, "ap-northeast-2": true, "ap-south-1": true, "ap-south-2": true, "ap-east-1": true,
	"sa-east-1": true, "ca-central-1": true, "af-south-1": true,
	"me-south-1": true, "me-central-1": true, "il-central-1": true,
}

func validateEmail(email string) *service.AWSError {
	if email == "" {
		return nil
	}
	if !emailRegex.MatchString(email) {
		return service.ErrValidation("EmailAddress is not a valid email format")
	}
	return nil
}

func validatePhone(phone string) *service.AWSError {
	if phone == "" {
		return nil
	}
	// Accept either E.164 or dashed formats like +1-555-0100
	cleaned := strings.ReplaceAll(phone, "-", "")
	if !phoneRegex.MatchString(cleaned) {
		return service.ErrValidation("PhoneNumber is not a valid phone format (expected E.164)")
	}
	return nil
}

func validateRegionName(regionName string) *service.AWSError {
	if !knownRegions[regionName] {
		return service.NewAWSError("ValidationException",
			"The provided region name is not a valid AWS region: "+regionName,
			http.StatusBadRequest)
	}
	return nil
}

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

func handleGetContactInformation(store *Store) (*service.Response, error) {
	info := store.GetContactInformation()
	return jsonOK(map[string]any{
		"ContactInformation": map[string]any{
			"FullName":         info.FullName,
			"AddressLine1":     info.AddressLine1,
			"AddressLine2":     info.AddressLine2,
			"AddressLine3":     info.AddressLine3,
			"City":             info.City,
			"StateOrRegion":    info.StateOrRegion,
			"PostalCode":       info.PostalCode,
			"CountryCode":      info.CountryCode,
			"PhoneNumber":      info.PhoneNumber,
			"CompanyName":      info.CompanyName,
			"DistrictOrCounty": info.DistrictOrCounty,
			"WebsiteUrl":       info.WebsiteURL,
		},
	})
}

func handlePutContactInformation(params map[string]any, store *Store) (*service.Response, error) {
	ci, _ := params["ContactInformation"].(map[string]any)
	if ci == nil {
		return jsonErr(service.ErrValidation("ContactInformation is required"))
	}
	fullName := str(ci, "FullName")
	if fullName == "" {
		return jsonErr(service.ErrValidation("ContactInformation.FullName is required"))
	}
	phone := str(ci, "PhoneNumber")
	if awsErr := validatePhone(phone); awsErr != nil {
		return jsonErr(awsErr)
	}
	info := &ContactInformation{
		FullName:         fullName,
		AddressLine1:     str(ci, "AddressLine1"),
		AddressLine2:     str(ci, "AddressLine2"),
		AddressLine3:     str(ci, "AddressLine3"),
		City:             str(ci, "City"),
		StateOrRegion:    str(ci, "StateOrRegion"),
		PostalCode:       str(ci, "PostalCode"),
		CountryCode:      str(ci, "CountryCode"),
		PhoneNumber:      phone,
		CompanyName:      str(ci, "CompanyName"),
		DistrictOrCounty: str(ci, "DistrictOrCounty"),
		WebsiteURL:       str(ci, "WebsiteUrl"),
	}
	store.PutContactInformation(info)
	return jsonOK(map[string]any{})
}

func handleGetAlternateContact(params map[string]any, store *Store) (*service.Response, error) {
	contactType := str(params, "AlternateContactType")
	if contactType == "" {
		return jsonErr(service.ErrValidation("AlternateContactType is required"))
	}
	contact, ok := store.GetAlternateContact(contactType)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"No alternate contact found for type: "+contactType, http.StatusNotFound))
	}
	return jsonOK(map[string]any{
		"AlternateContact": map[string]any{
			"AlternateContactType": contact.AlternateContactType,
			"Name":                 contact.Name,
			"Title":                contact.Title,
			"EmailAddress":         contact.EmailAddress,
			"PhoneNumber":          contact.PhoneNumber,
		},
	})
}

func handlePutAlternateContact(params map[string]any, store *Store) (*service.Response, error) {
	contactType := str(params, "AlternateContactType")
	if contactType == "" {
		return jsonErr(service.ErrValidation("AlternateContactType is required"))
	}
	validTypes := map[string]bool{"BILLING": true, "OPERATIONS": true, "SECURITY": true}
	if !validTypes[contactType] {
		return jsonErr(service.ErrValidation("AlternateContactType must be one of: BILLING, OPERATIONS, SECURITY"))
	}
	email := str(params, "EmailAddress")
	if awsErr := validateEmail(email); awsErr != nil {
		return jsonErr(awsErr)
	}
	phone := str(params, "PhoneNumber")
	if awsErr := validatePhone(phone); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := str(params, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required for alternate contact"))
	}
	contact := &AlternateContact{
		AlternateContactType: contactType,
		Name:                 name,
		Title:                str(params, "Title"),
		EmailAddress:         email,
		PhoneNumber:          phone,
	}
	store.PutAlternateContact(contact)
	return jsonOK(map[string]any{})
}

func handleDeleteAlternateContact(params map[string]any, store *Store) (*service.Response, error) {
	contactType := str(params, "AlternateContactType")
	if contactType == "" {
		return jsonErr(service.ErrValidation("AlternateContactType is required"))
	}
	if !store.DeleteAlternateContact(contactType) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			"No alternate contact found for type: "+contactType, http.StatusNotFound))
	}
	return jsonOK(map[string]any{})
}

func handleGetRegionOptStatus(params map[string]any, store *Store) (*service.Response, error) {
	regionName := str(params, "RegionName")
	if regionName == "" {
		return jsonErr(service.ErrValidation("RegionName is required"))
	}
	if awsErr := validateRegionName(regionName); awsErr != nil {
		return jsonErr(awsErr)
	}
	info, ok := store.GetRegionOptStatus(regionName)
	if !ok {
		return jsonErr(service.ErrNotFound("Region", regionName))
	}
	return jsonOK(map[string]any{
		"RegionName":      info.RegionName,
		"RegionOptStatus": info.RegionOptStatus,
	})
}

func handleListRegions(store *Store) (*service.Response, error) {
	regions := store.ListRegions()
	out := make([]map[string]any, 0, len(regions))
	for _, r := range regions {
		entry := map[string]any{
			"RegionName":      r.RegionName,
			"RegionOptStatus": r.RegionOptStatus,
		}
		if r.Description != "" {
			entry["RegionDescription"] = r.Description
		}
		out = append(out, entry)
	}
	return jsonOK(map[string]any{"Regions": out})
}

func handleEnableRegion(params map[string]any, store *Store) (*service.Response, error) {
	regionName := str(params, "RegionName")
	if regionName == "" {
		return jsonErr(service.ErrValidation("RegionName is required"))
	}
	if awsErr := validateRegionName(regionName); awsErr != nil {
		return jsonErr(awsErr)
	}
	store.EnableRegion(regionName)
	return jsonOK(map[string]any{})
}

func handleDisableRegion(params map[string]any, store *Store) (*service.Response, error) {
	regionName := str(params, "RegionName")
	if regionName == "" {
		return jsonErr(service.ErrValidation("RegionName is required"))
	}
	if awsErr := validateRegionName(regionName); awsErr != nil {
		return jsonErr(awsErr)
	}
	store.DisableRegion(regionName)
	return jsonOK(map[string]any{})
}
