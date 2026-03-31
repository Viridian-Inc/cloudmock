package account

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
	info := &ContactInformation{
		FullName:         str(ci, "FullName"),
		AddressLine1:     str(ci, "AddressLine1"),
		AddressLine2:     str(ci, "AddressLine2"),
		AddressLine3:     str(ci, "AddressLine3"),
		City:             str(ci, "City"),
		StateOrRegion:    str(ci, "StateOrRegion"),
		PostalCode:       str(ci, "PostalCode"),
		CountryCode:      str(ci, "CountryCode"),
		PhoneNumber:      str(ci, "PhoneNumber"),
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
	contact := &AlternateContact{
		AlternateContactType: contactType,
		Name:                 str(params, "Name"),
		Title:                str(params, "Title"),
		EmailAddress:         str(params, "EmailAddress"),
		PhoneNumber:          str(params, "PhoneNumber"),
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
		out = append(out, map[string]any{
			"RegionName":      r.RegionName,
			"RegionOptStatus": r.RegionOptStatus,
		})
	}
	return jsonOK(map[string]any{"Regions": out})
}

func handleEnableRegion(params map[string]any, store *Store) (*service.Response, error) {
	regionName := str(params, "RegionName")
	if regionName == "" {
		return jsonErr(service.ErrValidation("RegionName is required"))
	}
	store.EnableRegion(regionName)
	return jsonOK(map[string]any{})
}

func handleDisableRegion(params map[string]any, store *Store) (*service.Response, error) {
	regionName := str(params, "RegionName")
	if regionName == "" {
		return jsonErr(service.ErrValidation("RegionName is required"))
	}
	store.DisableRegion(regionName)
	return jsonOK(map[string]any{})
}
