package iotwireless

import (
	"net/http"
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

type tagEntry struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

func parseTags(p map[string]any) map[string]string {
	if v, ok := p["Tags"]; ok {
		if arr, ok := v.([]any); ok {
			m := make(map[string]string, len(arr))
			for _, item := range arr {
				if t, ok := item.(map[string]any); ok {
					m[getStr(t, "Key")] = getStr(t, "Value")
				}
			}
			return m
		}
	}
	return nil
}

func tagsToEntries(m map[string]string) []tagEntry {
	entries := make([]tagEntry, 0, len(m))
	for k, v := range m {
		entries = append(entries, tagEntry{Key: k, Value: v})
	}
	return entries
}

// ---- Wireless device handlers ----

func handleCreateWirelessDevice(p map[string]any, store *Store) (*service.Response, error) {
	d, awsErr := store.CreateWirelessDevice(
		getStr(p, "Name"),
		getStr(p, "Type"),
		getStr(p, "DestinationName"),
		getStr(p, "Description"),
		getMap(p, "LoRaWAN"),
		getMap(p, "Sidewalk"),
		parseTags(p),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonCreated(map[string]any{"Arn": d.Arn, "Id": d.Id})
}

func handleGetWirelessDevice(p map[string]any, store *Store) (*service.Response, error) {
	id := getStr(p, "Identifier")
	if id == "" {
		id = getStr(p, "Id")
	}
	if id == "" {
		return jsonErr(service.ErrValidation("Identifier is required."))
	}
	d, awsErr := store.GetWirelessDevice(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{
		"Id":              d.Id,
		"Arn":             d.Arn,
		"Name":            d.Name,
		"Type":            d.Type,
		"DestinationName": d.DestinationName,
		"Description":     d.Description,
	}
	if d.LoRaWAN != nil {
		resp["LoRaWAN"] = d.LoRaWAN
	}
	if d.Sidewalk != nil {
		resp["Sidewalk"] = d.Sidewalk
	}
	if d.ThingArn != "" {
		resp["ThingArn"] = d.ThingArn
	}
	return jsonOK(resp)
}

func handleListWirelessDevices(store *Store) (*service.Response, error) {
	devices := store.ListWirelessDevices()
	entries := make([]map[string]any, 0, len(devices))
	for _, d := range devices {
		entries = append(entries, map[string]any{
			"Id":              d.Id,
			"Arn":             d.Arn,
			"Name":            d.Name,
			"Type":            d.Type,
			"DestinationName": d.DestinationName,
		})
	}
	return jsonOK(map[string]any{"WirelessDeviceList": entries})
}

func handleDeleteWirelessDevice(p map[string]any, store *Store) (*service.Response, error) {
	id := getStr(p, "Id")
	if id == "" {
		return jsonErr(service.ErrValidation("Id is required."))
	}
	if awsErr := store.DeleteWirelessDevice(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleUpdateWirelessDevice(p map[string]any, store *Store) (*service.Response, error) {
	id := getStr(p, "Id")
	if id == "" {
		return jsonErr(service.ErrValidation("Id is required."))
	}
	_, awsErr := store.UpdateWirelessDevice(id, getStr(p, "Name"), getStr(p, "DestinationName"), getStr(p, "Description"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Wireless gateway handlers ----

func handleCreateWirelessGateway(p map[string]any, store *Store) (*service.Response, error) {
	gw, awsErr := store.CreateWirelessGateway(
		getStr(p, "Name"),
		getStr(p, "Description"),
		getMap(p, "LoRaWAN"),
		parseTags(p),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonCreated(map[string]any{"Arn": gw.Arn, "Id": gw.Id})
}

func handleGetWirelessGateway(p map[string]any, store *Store) (*service.Response, error) {
	id := getStr(p, "Identifier")
	if id == "" {
		id = getStr(p, "Id")
	}
	if id == "" {
		return jsonErr(service.ErrValidation("Identifier is required."))
	}
	gw, awsErr := store.GetWirelessGateway(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{
		"Id":          gw.Id,
		"Arn":         gw.Arn,
		"Name":        gw.Name,
		"Description": gw.Description,
	}
	if gw.LoRaWAN != nil {
		resp["LoRaWAN"] = gw.LoRaWAN
	}
	if gw.ThingArn != "" {
		resp["ThingArn"] = gw.ThingArn
	}
	return jsonOK(resp)
}

func handleListWirelessGateways(store *Store) (*service.Response, error) {
	gateways := store.ListWirelessGateways()
	entries := make([]map[string]any, 0, len(gateways))
	for _, gw := range gateways {
		entries = append(entries, map[string]any{
			"Id":          gw.Id,
			"Arn":         gw.Arn,
			"Name":        gw.Name,
			"Description": gw.Description,
		})
	}
	return jsonOK(map[string]any{"WirelessGatewayList": entries})
}

func handleDeleteWirelessGateway(p map[string]any, store *Store) (*service.Response, error) {
	id := getStr(p, "Id")
	if id == "" {
		return jsonErr(service.ErrValidation("Id is required."))
	}
	if awsErr := store.DeleteWirelessGateway(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleUpdateWirelessGateway(p map[string]any, store *Store) (*service.Response, error) {
	id := getStr(p, "Id")
	if id == "" {
		return jsonErr(service.ErrValidation("Id is required."))
	}
	_, awsErr := store.UpdateWirelessGateway(id, getStr(p, "Name"), getStr(p, "Description"))
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Device profile handlers ----

func handleCreateDeviceProfile(p map[string]any, store *Store) (*service.Response, error) {
	dp, awsErr := store.CreateDeviceProfile(
		getStr(p, "Name"),
		getMap(p, "LoRaWAN"),
		getMap(p, "Sidewalk"),
		parseTags(p),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonCreated(map[string]any{"Arn": dp.Arn, "Id": dp.Id})
}

func handleGetDeviceProfile(p map[string]any, store *Store) (*service.Response, error) {
	id := getStr(p, "Id")
	if id == "" {
		return jsonErr(service.ErrValidation("Id is required."))
	}
	dp, awsErr := store.GetDeviceProfile(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{
		"Id":   dp.Id,
		"Arn":  dp.Arn,
		"Name": dp.Name,
	}
	if dp.LoRaWAN != nil {
		resp["LoRaWAN"] = dp.LoRaWAN
	}
	if dp.Sidewalk != nil {
		resp["Sidewalk"] = dp.Sidewalk
	}
	return jsonOK(resp)
}

func handleListDeviceProfiles(store *Store) (*service.Response, error) {
	profiles := store.ListDeviceProfiles()
	entries := make([]map[string]any, 0, len(profiles))
	for _, dp := range profiles {
		entries = append(entries, map[string]any{"Id": dp.Id, "Arn": dp.Arn, "Name": dp.Name})
	}
	return jsonOK(map[string]any{"DeviceProfileList": entries})
}

func handleDeleteDeviceProfile(p map[string]any, store *Store) (*service.Response, error) {
	id := getStr(p, "Id")
	if id == "" {
		return jsonErr(service.ErrValidation("Id is required."))
	}
	if awsErr := store.DeleteDeviceProfile(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Service profile handlers ----

func handleCreateServiceProfile(p map[string]any, store *Store) (*service.Response, error) {
	sp, awsErr := store.CreateServiceProfile(
		getStr(p, "Name"),
		getMap(p, "LoRaWAN"),
		parseTags(p),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonCreated(map[string]any{"Arn": sp.Arn, "Id": sp.Id})
}

func handleGetServiceProfile(p map[string]any, store *Store) (*service.Response, error) {
	id := getStr(p, "Id")
	if id == "" {
		return jsonErr(service.ErrValidation("Id is required."))
	}
	sp, awsErr := store.GetServiceProfile(id)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	resp := map[string]any{
		"Id":   sp.Id,
		"Arn":  sp.Arn,
		"Name": sp.Name,
	}
	if sp.LoRaWAN != nil {
		resp["LoRaWAN"] = sp.LoRaWAN
	}
	return jsonOK(resp)
}

func handleListServiceProfiles(store *Store) (*service.Response, error) {
	profiles := store.ListServiceProfiles()
	entries := make([]map[string]any, 0, len(profiles))
	for _, sp := range profiles {
		entries = append(entries, map[string]any{"Id": sp.Id, "Arn": sp.Arn, "Name": sp.Name})
	}
	return jsonOK(map[string]any{"ServiceProfileList": entries})
}

func handleDeleteServiceProfile(p map[string]any, store *Store) (*service.Response, error) {
	id := getStr(p, "Id")
	if id == "" {
		return jsonErr(service.ErrValidation("Id is required."))
	}
	if awsErr := store.DeleteServiceProfile(id); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Destination handlers ----

func handleCreateDestination(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	d, awsErr := store.CreateDestination(
		name,
		getStr(p, "Expression"),
		getStr(p, "ExpressionType"),
		getStr(p, "Description"),
		getStr(p, "RoleArn"),
		parseTags(p),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonCreated(map[string]any{"Arn": d.Arn, "Name": d.Name})
}

func handleGetDestination(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	d, awsErr := store.GetDestination(name)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return jsonOK(map[string]any{
		"Name":           d.Name,
		"Arn":            d.Arn,
		"Expression":     d.Expression,
		"ExpressionType": d.ExpressionType,
		"Description":    d.Description,
		"RoleArn":        d.RoleArn,
		"CreatedAt":      d.CreationTime.Format(time.RFC3339),
	})
}

func handleListDestinations(store *Store) (*service.Response, error) {
	dests := store.ListDestinations()
	entries := make([]map[string]any, 0, len(dests))
	for _, d := range dests {
		entries = append(entries, map[string]any{
			"Name":           d.Name,
			"Arn":            d.Arn,
			"Expression":     d.Expression,
			"ExpressionType": d.ExpressionType,
			"Description":    d.Description,
			"RoleArn":        d.RoleArn,
		})
	}
	return jsonOK(map[string]any{"DestinationList": entries})
}

func handleUpdateDestination(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	_, awsErr := store.UpdateDestination(
		name,
		getStr(p, "Expression"),
		getStr(p, "ExpressionType"),
		getStr(p, "Description"),
		getStr(p, "RoleArn"),
	)
	if awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

func handleDeleteDestination(p map[string]any, store *Store) (*service.Response, error) {
	name := getStr(p, "Name")
	if name == "" {
		return jsonErr(service.ErrValidation("Name is required."))
	}
	if awsErr := store.DeleteDestination(name); awsErr != nil {
		return jsonErr(awsErr)
	}
	return emptyOK()
}

// ---- Tag handlers ----

func handleTagResource(p map[string]any, store *Store) (*service.Response, error) {
	arn := getStr(p, "ResourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("ResourceArn is required."))
	}
	tags := parseTags(p)
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
	tagKeys := getStringSlice(p, "TagKeys")
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
	return jsonOK(map[string]any{"Tags": tagsToEntries(tags)})
}
