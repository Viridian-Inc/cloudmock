package s3

import (
	"encoding/xml"
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- XML types for versioning ----

type versioningConfiguration struct {
	XMLName xml.Name `xml:"VersioningConfiguration"`
	Xmlns   string   `xml:"xmlns,attr,omitempty"`
	Status  string   `xml:"Status,omitempty"`
}

type xmlObjectVersion struct {
	Key          string `xml:"Key"`
	VersionId    string `xml:"VersionId"`
	IsLatest     bool   `xml:"IsLatest"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag,omitempty"`
	Size         int64  `xml:"Size"`
	StorageClass string `xml:"StorageClass,omitempty"`
}

type xmlDeleteMarker struct {
	Key          string `xml:"Key"`
	VersionId    string `xml:"VersionId"`
	IsLatest     bool   `xml:"IsLatest"`
	LastModified string `xml:"LastModified"`
}

type listVersionsResult struct {
	XMLName       xml.Name           `xml:"ListVersionsResult"`
	Xmlns         string             `xml:"xmlns,attr"`
	Name          string             `xml:"Name"`
	Prefix        string             `xml:"Prefix,omitempty"`
	Versions      []xmlObjectVersion `xml:"Version"`
	DeleteMarkers []xmlDeleteMarker  `xml:"DeleteMarker"`
}

// ---- handlers ----

func handlePutBucketVersioning(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)

	var vc versioningConfiguration
	if err := xml.Unmarshal(ctx.Body, &vc); err != nil {
		awsErr := service.NewAWSError("MalformedXML",
			"The XML you provided was not well-formed.", http.StatusBadRequest)
		return &service.Response{Format: service.FormatXML}, awsErr
	}

	if vc.Status != "Enabled" && vc.Status != "Suspended" {
		awsErr := service.NewAWSError("MalformedXML",
			"The XML you provided was not well-formed.", http.StatusBadRequest)
		return &service.Response{Format: service.FormatXML}, awsErr
	}

	if err := store.SetVersioning(bucket, vc.Status); err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	return &service.Response{
		StatusCode: http.StatusOK,
		Format:     service.FormatXML,
	}, nil
}

func handleGetBucketVersioning(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)

	status, err := store.GetVersioning(bucket)
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	result := &versioningConfiguration{
		Xmlns:  "http://s3.amazonaws.com/doc/2006-03-01/",
		Status: status,
	}

	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       result,
		Format:     service.FormatXML,
	}, nil
}

func handleListObjectVersions(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)

	objs, err := store.bucketObjects(bucket)
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	prefix := ctx.RawRequest.URL.Query().Get("prefix")
	versions := objs.ListObjectVersions(prefix)

	var xmlVersions []xmlObjectVersion
	var xmlDeleteMarkers []xmlDeleteMarker

	for _, v := range versions {
		if v.IsDeleteMarker {
			xmlDeleteMarkers = append(xmlDeleteMarkers, xmlDeleteMarker{
				Key:          v.Key,
				VersionId:    v.VersionId,
				IsLatest:     v.IsLatest,
				LastModified: v.LastModified.Format(time.RFC3339),
			})
		} else {
			xmlVersions = append(xmlVersions, xmlObjectVersion{
				Key:          v.Key,
				VersionId:    v.VersionId,
				IsLatest:     v.IsLatest,
				LastModified: v.LastModified.Format(time.RFC3339),
				ETag:         v.ETag,
				Size:         v.Size,
				StorageClass: "STANDARD",
			})
		}
	}

	result := &listVersionsResult{
		Xmlns:         "http://s3.amazonaws.com/doc/2006-03-01/",
		Name:          bucket,
		Prefix:        prefix,
		Versions:      xmlVersions,
		DeleteMarkers: xmlDeleteMarkers,
	}

	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       result,
		Format:     service.FormatXML,
	}, nil
}

// ---- Bucket Policy handlers ----

func handlePutBucketPolicy(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)

	if len(ctx.Body) == 0 {
		awsErr := service.NewAWSError("MissingRequestBodyError",
			"Request body is empty.", http.StatusBadRequest)
		return &service.Response{Format: service.FormatXML}, awsErr
	}

	if err := store.SetBucketPolicy(bucket, ctx.Body); err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	return &service.Response{
		StatusCode: http.StatusNoContent,
		Format:     service.FormatXML,
	}, nil
}

func handleGetBucketPolicy(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)

	policy, err := store.GetBucketPolicy(bucket)
	if err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        policy,
		RawContentType: "application/json",
	}, nil
}

func handleDeleteBucketPolicy(store *Store, ctx *service.RequestContext) (*service.Response, error) {
	bucket := extractBucketName(ctx)

	if err := store.DeleteBucketPolicy(bucket); err != nil {
		return &service.Response{Format: service.FormatXML}, err
	}

	return &service.Response{
		StatusCode: http.StatusNoContent,
		Format:     service.FormatXML,
	}, nil
}
