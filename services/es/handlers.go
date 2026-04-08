package es

import (
	"crypto/rand"
	gojson "github.com/goccy/go-json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

const esXmlns = "http://es.amazonaws.com/doc/2015-01-01/"

type xmlResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

type xmlTag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

// ---- helpers ----

func parseForm(ctx *service.RequestContext) url.Values {
	form := make(url.Values)
	for k, v := range ctx.Params {
		form.Set(k, v)
	}
	if len(ctx.Body) > 0 {
		if bodyVals, err := url.ParseQuery(string(ctx.Body)); err == nil {
			for k, vs := range bodyVals {
				for _, v := range vs {
					form.Add(k, v)
				}
			}
		}
	}
	return form
}

func parseTags(form url.Values) map[string]string {
	tags := make(map[string]string)
	for i := 1; ; i++ {
		key := form.Get(fmt.Sprintf("Tags.member.%d.Key", i))
		if key == "" {
			break
		}
		val := form.Get(fmt.Sprintf("Tags.member.%d.Value", i))
		tags[key] = val
	}
	return tags
}

func parseTagKeys(form url.Values) []string {
	keys := make([]string, 0)
	for i := 1; ; i++ {
		k := form.Get(fmt.Sprintf("TagKeys.member.%d", i))
		if k == "" {
			break
		}
		keys = append(keys, k)
	}
	return keys
}

func xmlOK(body any) (*service.Response, error) {
	data, err := xml.Marshal(body)
	if err != nil {
		return nil, err
	}
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        data,
		RawContentType: "text/xml",
	}, nil
}

func xmlErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatXML}, awsErr
}

func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ---- Domain XML types ----

type xmlDomainStatus struct {
	DomainName           string `xml:"DomainName"`
	ARN                  string `xml:"ARN"`
	DomainId             string `xml:"DomainId"`
	ElasticsearchVersion string `xml:"ElasticsearchVersion"`
	Endpoint             string `xml:"Endpoint"`
	Processing           bool   `xml:"Processing"`
	Created              bool   `xml:"Created"`
	Deleted              bool   `xml:"Deleted"`
}

func toXMLDomainStatus(d *Domain) xmlDomainStatus {
	return xmlDomainStatus{
		DomainName: d.DomainName, ARN: d.ARN, DomainId: d.DomainId,
		ElasticsearchVersion: d.ElasticsearchVersion, Endpoint: d.Endpoint,
		Processing: d.Processing, Created: d.Created, Deleted: d.Deleted,
	}
}

// ---- CreateElasticsearchDomain ----

type xmlCreateElasticsearchDomainResponse struct {
	XMLName xml.Name            `xml:"CreateElasticsearchDomainResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct{ DomainStatus xmlDomainStatus } `xml:"CreateElasticsearchDomainResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleCreateElasticsearchDomain(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("DomainName")
	if name == "" {
		return xmlErr(service.ErrValidation("DomainName is required."))
	}
	version := form.Get("ElasticsearchVersion")
	instanceType := form.Get("ElasticsearchClusterConfig.InstanceType")
	instanceCount := 1
	if s := form.Get("ElasticsearchClusterConfig.InstanceCount"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			instanceCount = v
		}
	}
	tags := parseTags(form)
	d, ok := store.CreateDomain(name, version, instanceType, instanceCount, tags)
	if !ok {
		return xmlErr(service.ErrAlreadyExists("Domain", name))
	}
	resp := &xmlCreateElasticsearchDomainResponse{Xmlns: esXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.DomainStatus = toXMLDomainStatus(d)
	return xmlOK(resp)
}

// ---- DescribeElasticsearchDomain ----

type xmlDescribeElasticsearchDomainResponse struct {
	XMLName xml.Name            `xml:"DescribeElasticsearchDomainResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct{ DomainStatus xmlDomainStatus } `xml:"DescribeElasticsearchDomainResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDescribeElasticsearchDomain(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("DomainName")
	if name == "" {
		return xmlErr(service.ErrValidation("DomainName is required."))
	}
	d, ok := store.GetDomain(name)
	if !ok {
		return xmlErr(service.NewAWSError("ResourceNotFoundException", "Domain "+name+" not found.", http.StatusNotFound))
	}
	resp := &xmlDescribeElasticsearchDomainResponse{Xmlns: esXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.DomainStatus = toXMLDomainStatus(d)
	return xmlOK(resp)
}

// ---- ListDomainNames ----

type xmlDomainInfo struct {
	DomainName string `xml:"DomainName"`
}

type xmlListDomainNamesResponse struct {
	XMLName xml.Name            `xml:"ListDomainNamesResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct{ DomainNames []xmlDomainInfo `xml:"DomainNames>member"` } `xml:"ListDomainNamesResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleListDomainNames(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	domains := store.ListDomainNames()
	list := make([]xmlDomainInfo, 0, len(domains))
	for _, d := range domains {
		list = append(list, xmlDomainInfo{DomainName: d.DomainName})
	}
	resp := &xmlListDomainNamesResponse{Xmlns: esXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.DomainNames = list
	return xmlOK(resp)
}

// ---- DeleteElasticsearchDomain ----

type xmlDeleteElasticsearchDomainResponse struct {
	XMLName xml.Name            `xml:"DeleteElasticsearchDomainResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct{ DomainStatus xmlDomainStatus } `xml:"DeleteElasticsearchDomainResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteElasticsearchDomain(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("DomainName")
	if name == "" {
		return xmlErr(service.ErrValidation("DomainName is required."))
	}
	d, ok := store.DeleteDomain(name)
	if !ok {
		return xmlErr(service.NewAWSError("ResourceNotFoundException", "Domain "+name+" not found.", http.StatusNotFound))
	}
	resp := &xmlDeleteElasticsearchDomainResponse{Xmlns: esXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.DomainStatus = toXMLDomainStatus(d)
	return xmlOK(resp)
}

// ---- UpdateElasticsearchDomainConfig ----

type xmlUpdateElasticsearchDomainConfigResponse struct {
	XMLName xml.Name            `xml:"UpdateElasticsearchDomainConfigResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct{ DomainConfig xmlDomainStatus } `xml:"UpdateElasticsearchDomainConfigResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleUpdateElasticsearchDomainConfig(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("DomainName")
	if name == "" {
		return xmlErr(service.ErrValidation("DomainName is required."))
	}
	version := form.Get("ElasticsearchVersion")
	instanceType := form.Get("ElasticsearchClusterConfig.InstanceType")
	instanceCount := 0
	if s := form.Get("ElasticsearchClusterConfig.InstanceCount"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			instanceCount = v
		}
	}
	d, ok := store.UpdateDomainConfig(name, version, instanceType, instanceCount)
	if !ok {
		return xmlErr(service.NewAWSError("ResourceNotFoundException", "Domain "+name+" not found.", http.StatusNotFound))
	}
	resp := &xmlUpdateElasticsearchDomainConfigResponse{Xmlns: esXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.DomainConfig = toXMLDomainStatus(d)
	return xmlOK(resp)
}

// ---- DescribeElasticsearchDomainConfig ----

type xmlDescribeElasticsearchDomainConfigResponse struct {
	XMLName xml.Name            `xml:"DescribeElasticsearchDomainConfigResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct{ DomainConfig xmlDomainStatus } `xml:"DescribeElasticsearchDomainConfigResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDescribeElasticsearchDomainConfig(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	name := form.Get("DomainName")
	if name == "" {
		return xmlErr(service.ErrValidation("DomainName is required."))
	}
	d, ok := store.GetDomain(name)
	if !ok {
		return xmlErr(service.NewAWSError("ResourceNotFoundException", "Domain "+name+" not found.", http.StatusNotFound))
	}
	resp := &xmlDescribeElasticsearchDomainConfigResponse{Xmlns: esXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.DomainConfig = toXMLDomainStatus(d)
	return xmlOK(resp)
}

// ---- Tag handlers ----

type xmlAddTagsResponse struct {
	XMLName xml.Name            `xml:"AddTagsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleAddTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("ARN")
	tags := parseTags(form)
	store.AddTags(arn, tags)
	return xmlOK(&xmlAddTagsResponse{Xmlns: esXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}})
}

type xmlRemoveTagsResponse struct {
	XMLName xml.Name            `xml:"RemoveTagsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleRemoveTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("ARN")
	keys := parseTagKeys(form)
	store.RemoveTags(arn, keys)
	return xmlOK(&xmlRemoveTagsResponse{Xmlns: esXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}})
}

type xmlListTagsResponse struct {
	XMLName xml.Name            `xml:"ListTagsResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct{ TagList []xmlTag `xml:"TagList>Tag"` } `xml:"ListTagsResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleListTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	arn := form.Get("ARN")
	tags, _ := store.ListTags(arn)
	if tags == nil {
		tags = make(map[string]string)
	}
	xmlTags := make([]xmlTag, 0, len(tags))
	for k, v := range tags {
		xmlTags = append(xmlTags, xmlTag{Key: k, Value: v})
	}
	resp := &xmlListTagsResponse{Xmlns: esXmlns, Meta: xmlResponseMetadata{RequestID: newRequestID()}}
	resp.Result.TagList = xmlTags
	return xmlOK(resp)
}

// ---- JSON-based document API handlers ----

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSONBody(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("ValidationException", "Invalid JSON.", http.StatusBadRequest)
	}
	return nil
}

type indexDocumentRequest struct {
	DomainName string         `json:"DomainName"`
	Index      string         `json:"Index"`
	DocumentId string         `json:"DocumentId"`
	Document   map[string]any `json:"Document"`
}

func handleIndexDocument(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req indexDocumentRequest
	if awsErr := parseJSONBody(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DomainName == "" || req.Index == "" {
		return jsonErr(service.ErrValidation("DomainName and Index are required."))
	}
	docID, ok := store.IndexDocument(req.DomainName, req.Index, req.DocumentId, req.Document)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Domain "+req.DomainName+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{
		"_index": req.Index, "_id": docID, "result": "created", "_version": 1,
	})
}

type searchRequest struct {
	DomainName string         `json:"DomainName"`
	Index      string         `json:"Index"`
	Query      map[string]any `json:"Query"`
}

func handleSearch(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req searchRequest
	if awsErr := parseJSONBody(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	if req.DomainName == "" || req.Index == "" {
		return jsonErr(service.ErrValidation("DomainName and Index are required."))
	}
	docs, ok := store.SearchDocuments(req.DomainName, req.Index, req.Query)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Domain "+req.DomainName+" not found.", http.StatusNotFound))
	}
	hits := make([]map[string]any, len(docs))
	for i, d := range docs {
		hits[i] = map[string]any{"_index": d.Index, "_id": d.ID, "_source": d.Source}
	}
	return jsonOK(map[string]any{
		"hits": map[string]any{
			"total": map[string]any{"value": len(docs), "relation": "eq"},
			"hits":  hits,
		},
	})
}

type clusterHealthRequest struct {
	DomainName string `json:"DomainName"`
}

func handleClusterHealth(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req clusterHealthRequest
	if awsErr := parseJSONBody(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	health, ok := store.ClusterHealth(req.DomainName)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException", "Domain "+req.DomainName+" not found.", http.StatusNotFound))
	}
	return jsonOK(map[string]any{
		"cluster_name": req.DomainName, "status": health, "number_of_nodes": 1,
	})
}
