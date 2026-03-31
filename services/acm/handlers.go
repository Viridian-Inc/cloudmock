package acm

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
)

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.NewAWSError("ValidationException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func timeToUnix(t *time.Time) *float64 {
	if t == nil {
		return nil
	}
	v := float64(t.Unix())
	return &v
}

func certToSummary(c *Certificate) map[string]any {
	m := map[string]any{
		"CertificateArn":          c.CertificateArn,
		"DomainName":              c.DomainName,
		"SubjectAlternativeNames": c.SubjectAlternativeNames,
		"Status":                  string(c.Status),
		"Type":                    string(c.Type),
		"KeyAlgorithm":            c.KeyAlgorithm,
		"CreatedAt":               float64(c.CreatedAt.Unix()),
		"InUseBy":                 c.InUseBy,
		"RenewalEligibility":      c.RenewalEligibility,
	}
	if c.IssuedAt != nil {
		m["IssuedAt"] = float64(c.IssuedAt.Unix())
	}
	if c.NotBefore != nil {
		m["NotBefore"] = float64(c.NotBefore.Unix())
	}
	if c.NotAfter != nil {
		m["NotAfter"] = float64(c.NotAfter.Unix())
	}
	return m
}

func certToDetail(c *Certificate) map[string]any {
	m := certToSummary(c)
	m["Serial"] = c.Serial
	m["Subject"] = c.Subject
	m["Issuer"] = c.Issuer
	m["ValidationMethod"] = string(c.ValidationMethod)

	if c.ImportedAt != nil {
		m["ImportedAt"] = float64(c.ImportedAt.Unix())
	}

	dvs := make([]map[string]any, 0, len(c.DomainValidationOptions))
	for _, dv := range c.DomainValidationOptions {
		d := map[string]any{
			"DomainName":       dv.DomainName,
			"ValidationDomain": dv.ValidationDomain,
			"ValidationStatus": dv.ValidationStatus,
			"ValidationMethod": dv.ValidationMethod,
		}
		if dv.ResourceRecord != nil {
			d["ResourceRecord"] = map[string]any{
				"Name":  dv.ResourceRecord.Name,
				"Type":  dv.ResourceRecord.Type,
				"Value": dv.ResourceRecord.Value,
			}
		}
		dvs = append(dvs, d)
	}
	m["DomainValidationOptions"] = dvs
	return m
}

func tagsToMaps(tags []Tag) []map[string]string {
	out := make([]map[string]string, 0, len(tags))
	for _, t := range tags {
		out = append(out, map[string]string{"Key": t.Key, "Value": t.Value})
	}
	return out
}

func parseTags(raw []any) []Tag {
	tags := make([]Tag, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		t := Tag{}
		if v, ok := m["Key"].(string); ok {
			t.Key = v
		}
		if v, ok := m["Value"].(string); ok {
			t.Value = v
		}
		tags = append(tags, t)
	}
	return tags
}

// handleRequestCertificate handles the RequestCertificate action.
func handleRequestCertificate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	domainName, _ := params["DomainName"].(string)
	if domainName == "" {
		return jsonErr(service.ErrValidation("DomainName is required."))
	}

	var sans []string
	if rawSANs, ok := params["SubjectAlternativeNames"].([]any); ok {
		for _, s := range rawSANs {
			if v, ok := s.(string); ok {
				sans = append(sans, v)
			}
		}
	}

	validationMethod := ValidationDNS
	if vm, ok := params["ValidationMethod"].(string); ok && vm == "EMAIL" {
		validationMethod = ValidationEmail
	}

	keyAlgo, _ := params["KeyAlgorithm"].(string)

	var tags []Tag
	if rawTags, ok := params["Tags"].([]any); ok {
		tags = parseTags(rawTags)
	}

	cert, err := store.RequestCertificate(domainName, sans, validationMethod, keyAlgo, tags)
	if err != nil {
		return jsonErr(service.NewAWSError("InternalFailure", err.Error(), http.StatusInternalServerError))
	}

	return jsonOK(map[string]any{"CertificateArn": cert.CertificateArn})
}

// handleDescribeCertificate handles the DescribeCertificate action.
func handleDescribeCertificate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn, _ := params["CertificateArn"].(string)
	if arn == "" {
		return jsonErr(service.ErrValidation("CertificateArn is required."))
	}

	cert, awsErr := store.GetCertificate(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{"Certificate": certToDetail(cert)})
}

// handleListCertificates handles the ListCertificates action.
func handleListCertificates(_ *service.RequestContext, store *Store) (*service.Response, error) {
	certs := store.ListCertificates()
	summaries := make([]map[string]any, 0, len(certs))
	for _, c := range certs {
		summaries = append(summaries, certToSummary(c))
	}
	return jsonOK(map[string]any{
		"CertificateSummaryList": summaries,
	})
}

// handleDeleteCertificate handles the DeleteCertificate action.
func handleDeleteCertificate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn, _ := params["CertificateArn"].(string)
	if arn == "" {
		return jsonErr(service.ErrValidation("CertificateArn is required."))
	}

	if awsErr := store.DeleteCertificate(arn); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

// handleImportCertificate handles the ImportCertificate action.
func handleImportCertificate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	certBody, _ := params["Certificate"].(string)
	if certBody == "" {
		return jsonErr(service.ErrValidation("Certificate is required."))
	}
	privateKey, _ := params["PrivateKey"].(string)
	if privateKey == "" {
		return jsonErr(service.ErrValidation("PrivateKey is required."))
	}
	certChain, _ := params["CertificateChain"].(string)
	existingARN, _ := params["CertificateArn"].(string)

	var tags []Tag
	if rawTags, ok := params["Tags"].([]any); ok {
		tags = parseTags(rawTags)
	}

	cert, err := store.ImportCertificate(certBody, certChain, privateKey, tags, existingARN)
	if err != nil {
		return jsonErr(service.NewAWSError("InternalFailure", err.Error(), http.StatusInternalServerError))
	}

	return jsonOK(map[string]any{"CertificateArn": cert.CertificateArn})
}

// handleRenewCertificate handles the RenewCertificate action.
func handleRenewCertificate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn, _ := params["CertificateArn"].(string)
	if arn == "" {
		return jsonErr(service.ErrValidation("CertificateArn is required."))
	}

	if awsErr := store.RenewCertificate(arn); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

// handleExportCertificate handles the ExportCertificate action.
func handleExportCertificate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn, _ := params["CertificateArn"].(string)
	if arn == "" {
		return jsonErr(service.ErrValidation("CertificateArn is required."))
	}

	cert, awsErr := store.GetCertificate(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	if cert.Type != CertTypeImported {
		return jsonErr(service.NewAWSError("ValidationException",
			"Only imported certificates can be exported.", http.StatusBadRequest))
	}

	return jsonOK(map[string]any{
		"Certificate":      cert.CertificateBody,
		"CertificateChain": cert.CertificateChain,
		"PrivateKey":       cert.PrivateKey,
	})
}

// handleGetCertificate handles the GetCertificate action.
func handleGetCertificate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn, _ := params["CertificateArn"].(string)
	if arn == "" {
		return jsonErr(service.ErrValidation("CertificateArn is required."))
	}

	cert, awsErr := store.GetCertificate(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	if cert.Status != StatusIssued {
		return jsonErr(service.NewAWSError("RequestInProgressException",
			"Certificate is not yet issued.", http.StatusBadRequest))
	}

	result := map[string]any{
		"Certificate": cert.CertificateBody,
	}
	if cert.CertificateChain != "" {
		result["CertificateChain"] = cert.CertificateChain
	}
	return jsonOK(result)
}

// handleAddTagsToCertificate handles the AddTagsToCertificate action.
func handleAddTagsToCertificate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn, _ := params["CertificateArn"].(string)
	if arn == "" {
		return jsonErr(service.ErrValidation("CertificateArn is required."))
	}

	rawTags, ok := params["Tags"].([]any)
	if !ok || len(rawTags) == 0 {
		return jsonErr(service.ErrValidation("Tags are required."))
	}

	if awsErr := store.AddTags(arn, parseTags(rawTags)); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

// handleRemoveTagsFromCertificate handles the RemoveTagsFromCertificate action.
func handleRemoveTagsFromCertificate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn, _ := params["CertificateArn"].(string)
	if arn == "" {
		return jsonErr(service.ErrValidation("CertificateArn is required."))
	}

	rawTags, ok := params["Tags"].([]any)
	if !ok || len(rawTags) == 0 {
		return jsonErr(service.ErrValidation("Tags are required."))
	}

	if awsErr := store.RemoveTags(arn, parseTags(rawTags)); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

// handleListTagsForCertificate handles the ListTagsForCertificate action.
func handleListTagsForCertificate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn, _ := params["CertificateArn"].(string)
	if arn == "" {
		return jsonErr(service.ErrValidation("CertificateArn is required."))
	}

	cert, awsErr := store.GetCertificate(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{"Tags": tagsToMaps(cert.Tags)})
}
