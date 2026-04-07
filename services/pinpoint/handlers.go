package pinpoint

import (
	"net/http"

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

func tagsMap(params map[string]any) map[string]string {
	tags := make(map[string]string)
	if v, ok := params["tags"].(map[string]any); ok {
		for k, val := range v {
			if sv, ok := val.(string); ok {
				tags[k] = sv
			}
		}
	}
	return tags
}

func handleCreateApp(params map[string]any, store *Store) (*service.Response, error) {
	cReq, _ := params["CreateApplicationRequest"].(map[string]any)
	name := str(cReq, "Name")
	if name == "" {
		name = str(params, "Name")
	}
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required"))
	}
	app := store.CreateApp(name, tagsMap(params))
	return &service.Response{
		StatusCode: http.StatusCreated,
		Body: map[string]any{
			"ApplicationResponse": map[string]any{
				"Id":   app.ApplicationID,
				"Arn":  app.Arn,
				"Name": app.Name,
			},
		},
		Format: service.FormatJSON,
	}, nil
}

func handleGetApp(appID string, store *Store) (*service.Response, error) {
	app, ok := store.GetApp(appID)
	if !ok {
		return jsonErr(service.ErrNotFound("Application", appID))
	}
	return jsonOK(map[string]any{
		"ApplicationResponse": map[string]any{
			"Id":   app.ApplicationID,
			"Arn":  app.Arn,
			"Name": app.Name,
			"tags": app.Tags,
		},
	})
}

func handleGetApps(store *Store) (*service.Response, error) {
	apps := store.ListApps()
	out := make([]map[string]any, 0, len(apps))
	for _, a := range apps {
		out = append(out, map[string]any{
			"Id":   a.ApplicationID,
			"Arn":  a.Arn,
			"Name": a.Name,
		})
	}
	return jsonOK(map[string]any{
		"ApplicationsResponse": map[string]any{"Item": out},
	})
}

func handleDeleteApp(appID string, store *Store) (*service.Response, error) {
	app, ok := store.GetApp(appID)
	if !ok {
		return jsonErr(service.ErrNotFound("Application", appID))
	}
	store.DeleteApp(appID)
	return jsonOK(map[string]any{
		"ApplicationResponse": map[string]any{
			"Id":   app.ApplicationID,
			"Arn":  app.Arn,
			"Name": app.Name,
		},
	})
}

func handleCreateSegment(appID string, params map[string]any, store *Store) (*service.Response, error) {
	req, _ := params["WriteSegmentRequest"].(map[string]any)
	if req == nil {
		req = params
	}
	name := str(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required"))
	}
	dims, _ := req["Dimensions"].(map[string]any)
	seg, err := store.CreateSegment(appID, name, "DIMENSIONAL", dims)
	if err != nil {
		return jsonErr(service.ErrNotFound("Application", appID))
	}
	return &service.Response{
		StatusCode: http.StatusCreated,
		Body: map[string]any{
			"SegmentResponse": map[string]any{
				"Id":            seg.SegmentID,
				"ApplicationId": seg.ApplicationID,
				"Name":          seg.Name,
				"SegmentType":   seg.SegmentType,
				"Version":       seg.Version,
			},
		},
		Format: service.FormatJSON,
	}, nil
}

func handleGetSegment(appID, segID string, store *Store) (*service.Response, error) {
	seg, ok := store.GetSegment(appID, segID)
	if !ok {
		return jsonErr(service.ErrNotFound("Segment", segID))
	}
	return jsonOK(map[string]any{
		"SegmentResponse": map[string]any{
			"Id":            seg.SegmentID,
			"ApplicationId": seg.ApplicationID,
			"Name":          seg.Name,
			"SegmentType":   seg.SegmentType,
			"Version":       seg.Version,
		},
	})
}

func handleGetSegments(appID string, store *Store) (*service.Response, error) {
	segs := store.ListSegments(appID)
	out := make([]map[string]any, 0, len(segs))
	for _, s := range segs {
		out = append(out, map[string]any{
			"Id":            s.SegmentID,
			"ApplicationId": s.ApplicationID,
			"Name":          s.Name,
			"SegmentType":   s.SegmentType,
		})
	}
	return jsonOK(map[string]any{
		"SegmentsResponse": map[string]any{"Item": out},
	})
}

func handleDeleteSegment(appID, segID string, store *Store) (*service.Response, error) {
	if !store.DeleteSegment(appID, segID) {
		return jsonErr(service.ErrNotFound("Segment", segID))
	}
	return jsonOK(map[string]any{
		"SegmentResponse": map[string]any{"Id": segID, "ApplicationId": appID},
	})
}

func handleCreateCampaign(appID string, params map[string]any, store *Store) (*service.Response, error) {
	req, _ := params["WriteCampaignRequest"].(map[string]any)
	if req == nil {
		req = params
	}
	name := str(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required"))
	}
	schedule, _ := req["Schedule"].(map[string]any)
	camp, err := store.CreateCampaign(appID, name, str(req, "SegmentId"), str(req, "Description"), schedule)
	if err != nil {
		return jsonErr(service.ErrNotFound("Application", appID))
	}
	return &service.Response{
		StatusCode: http.StatusCreated,
		Body: map[string]any{
			"CampaignResponse": map[string]any{
				"Id":            camp.CampaignID,
				"ApplicationId": camp.ApplicationID,
				"Name":          camp.Name,
				"SegmentId":     camp.SegmentID,
				"State":         map[string]any{"CampaignStatus": camp.State},
			},
		},
		Format: service.FormatJSON,
	}, nil
}

func handleGetCampaign(appID, campID string, store *Store) (*service.Response, error) {
	camp, ok := store.GetCampaign(appID, campID)
	if !ok {
		return jsonErr(service.ErrNotFound("Campaign", campID))
	}
	return jsonOK(map[string]any{
		"CampaignResponse": map[string]any{
			"Id":            camp.CampaignID,
			"ApplicationId": camp.ApplicationID,
			"Name":          camp.Name,
			"SegmentId":     camp.SegmentID,
			"State":         map[string]any{"CampaignStatus": camp.State},
			"Description":   camp.Description,
		},
	})
}

func handleGetCampaigns(appID string, store *Store) (*service.Response, error) {
	camps := store.ListCampaigns(appID)
	out := make([]map[string]any, 0, len(camps))
	for _, c := range camps {
		out = append(out, map[string]any{
			"Id":            c.CampaignID,
			"ApplicationId": c.ApplicationID,
			"Name":          c.Name,
			"State":         map[string]any{"CampaignStatus": c.State},
		})
	}
	return jsonOK(map[string]any{
		"CampaignsResponse": map[string]any{"Item": out},
	})
}

func handleDeleteCampaign(appID, campID string, store *Store) (*service.Response, error) {
	if !store.DeleteCampaign(appID, campID) {
		return jsonErr(service.ErrNotFound("Campaign", campID))
	}
	return jsonOK(map[string]any{
		"CampaignResponse": map[string]any{"Id": campID, "ApplicationId": appID},
	})
}

func handleCreateJourney(appID string, params map[string]any, store *Store) (*service.Response, error) {
	req, _ := params["WriteJourneyRequest"].(map[string]any)
	if req == nil {
		req = params
	}
	name := str(req, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required"))
	}
	j, err := store.CreateJourney(appID, name)
	if err != nil {
		return jsonErr(service.ErrNotFound("Application", appID))
	}
	return &service.Response{
		StatusCode: http.StatusCreated,
		Body: map[string]any{
			"JourneyResponse": map[string]any{
				"Id":            j.JourneyID,
				"ApplicationId": j.ApplicationID,
				"Name":          j.Name,
				"State":         j.State,
			},
		},
		Format: service.FormatJSON,
	}, nil
}

func handleGetJourney(appID, journeyID string, store *Store) (*service.Response, error) {
	j, ok := store.GetJourney(appID, journeyID)
	if !ok {
		return jsonErr(service.ErrNotFound("Journey", journeyID))
	}
	return jsonOK(map[string]any{
		"JourneyResponse": map[string]any{
			"Id":            j.JourneyID,
			"ApplicationId": j.ApplicationID,
			"Name":          j.Name,
			"State":         j.State,
		},
	})
}

func handleListJourneys(appID string, store *Store) (*service.Response, error) {
	journeys := store.ListJourneys(appID)
	out := make([]map[string]any, 0, len(journeys))
	for _, j := range journeys {
		out = append(out, map[string]any{
			"Id":            j.JourneyID,
			"ApplicationId": j.ApplicationID,
			"Name":          j.Name,
			"State":         j.State,
		})
	}
	return jsonOK(map[string]any{
		"JourneysResponse": map[string]any{"Item": out},
	})
}

func handleUpdateEndpoint(appID, endpointID string, params map[string]any, store *Store) (*service.Response, error) {
	req, _ := params["EndpointRequest"].(map[string]any)
	if req == nil {
		req = params
	}
	channelType := str(req, "ChannelType")
	address := str(req, "Address")
	user, _ := req["User"].(map[string]any)

	ep, err := store.UpdateEndpoint(appID, endpointID, channelType, address, user, nil)
	if err != nil {
		return jsonErr(service.ErrNotFound("Application", appID))
	}
	return &service.Response{
		StatusCode: http.StatusAccepted,
		Body:       map[string]any{"MessageBody": map[string]any{"Message": "Accepted", "RequestID": ep.EndpointID}},
		Format:     service.FormatJSON,
	}, nil
}

func handleGetEndpoint(appID, endpointID string, store *Store) (*service.Response, error) {
	ep, ok := store.GetEndpoint(appID, endpointID)
	if !ok {
		return jsonErr(service.ErrNotFound("Endpoint", endpointID))
	}
	return jsonOK(map[string]any{
		"EndpointResponse": map[string]any{
			"Id":            ep.EndpointID,
			"ApplicationId": ep.ApplicationID,
			"ChannelType":   ep.ChannelType,
			"Address":       ep.Address,
		},
	})
}
