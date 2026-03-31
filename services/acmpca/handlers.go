package acmpca

import (
	"encoding/json"
	"net/http"

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

func caToMap(ca *CertificateAuthority) map[string]any {
	m := map[string]any{
		"Arn":              ca.Arn,
		"Type":             ca.Type,
		"Status":           string(ca.Status),
		"KeyAlgorithm":     ca.KeyAlgorithm,
		"SigningAlgorithm":  ca.SigningAlgorithm,
		"Serial":           ca.Serial,
		"CreatedAt":        float64(ca.CreatedAt.Unix()),
		"LastStateChangeAt": float64(ca.LastStateChangeAt.Unix()),
		"CertificateAuthorityConfiguration": map[string]any{
			"KeyAlgorithm":     ca.KeyAlgorithm,
			"SigningAlgorithm": ca.SigningAlgorithm,
			"Subject": map[string]any{
				"Country":            ca.Subject.Country,
				"Organization":       ca.Subject.Organization,
				"OrganizationalUnit": ca.Subject.OrganizationalUnit,
				"State":              ca.Subject.State,
				"Locality":           ca.Subject.Locality,
				"CommonName":         ca.Subject.CommonName,
			},
		},
	}
	if ca.NotBefore != nil {
		m["NotBefore"] = float64(ca.NotBefore.Unix())
	}
	if ca.NotAfter != nil {
		m["NotAfter"] = float64(ca.NotAfter.Unix())
	}
	if ca.RevocationConfiguration != nil {
		m["RevocationConfiguration"] = ca.RevocationConfiguration
	}
	return m
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

func tagsToMaps(tags []Tag) []map[string]string {
	out := make([]map[string]string, 0, len(tags))
	for _, t := range tags {
		out = append(out, map[string]string{"Key": t.Key, "Value": t.Value})
	}
	return out
}

func parseSubject(raw map[string]any) CASubject {
	s := CASubject{}
	if v, ok := raw["Country"].(string); ok {
		s.Country = v
	}
	if v, ok := raw["Organization"].(string); ok {
		s.Organization = v
	}
	if v, ok := raw["OrganizationalUnit"].(string); ok {
		s.OrganizationalUnit = v
	}
	if v, ok := raw["State"].(string); ok {
		s.State = v
	}
	if v, ok := raw["Locality"].(string); ok {
		s.Locality = v
	}
	if v, ok := raw["CommonName"].(string); ok {
		s.CommonName = v
	}
	return s
}

func handleCreateCertificateAuthority(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	caType, _ := params["CertificateAuthorityType"].(string)

	var subject CASubject
	if config, ok := params["CertificateAuthorityConfiguration"].(map[string]any); ok {
		if subj, ok := config["Subject"].(map[string]any); ok {
			subject = parseSubject(subj)
		}
	}

	keyAlgo, _ := params["KeyAlgorithm"].(string)
	sigAlgo, _ := params["SigningAlgorithm"].(string)

	// Try to get key/signing algo from config
	if config, ok := params["CertificateAuthorityConfiguration"].(map[string]any); ok {
		if v, ok := config["KeyAlgorithm"].(string); ok && keyAlgo == "" {
			keyAlgo = v
		}
		if v, ok := config["SigningAlgorithm"].(string); ok && sigAlgo == "" {
			sigAlgo = v
		}
	}

	var revConfig map[string]any
	if rc, ok := params["RevocationConfiguration"].(map[string]any); ok {
		revConfig = rc
	}

	var tags []Tag
	if rawTags, ok := params["Tags"].([]any); ok {
		tags = parseTags(rawTags)
	}

	ca, err := store.CreateCA(caType, keyAlgo, sigAlgo, subject, revConfig, tags)
	if err != nil {
		return jsonErr(service.NewAWSError("InternalFailure", err.Error(), http.StatusInternalServerError))
	}

	return jsonOK(map[string]any{"CertificateAuthorityArn": ca.Arn})
}

func handleDescribeCertificateAuthority(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn, _ := params["CertificateAuthorityArn"].(string)
	if arn == "" {
		return jsonErr(service.ErrValidation("CertificateAuthorityArn is required."))
	}

	ca, awsErr := store.GetCA(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{"CertificateAuthority": caToMap(ca)})
}

func handleListCertificateAuthorities(_ *service.RequestContext, store *Store) (*service.Response, error) {
	cas := store.ListCAs()
	items := make([]map[string]any, 0, len(cas))
	for _, ca := range cas {
		items = append(items, caToMap(ca))
	}
	return jsonOK(map[string]any{"CertificateAuthorities": items})
}

func handleDeleteCertificateAuthority(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn, _ := params["CertificateAuthorityArn"].(string)
	if arn == "" {
		return jsonErr(service.ErrValidation("CertificateAuthorityArn is required."))
	}

	if awsErr := store.DeleteCA(arn); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleUpdateCertificateAuthority(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn, _ := params["CertificateAuthorityArn"].(string)
	if arn == "" {
		return jsonErr(service.ErrValidation("CertificateAuthorityArn is required."))
	}

	status, _ := params["Status"].(string)
	var revConfig map[string]any
	if rc, ok := params["RevocationConfiguration"].(map[string]any); ok {
		revConfig = rc
	}

	if awsErr := store.UpdateCA(arn, status, revConfig); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleIssueCertificate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	caArn, _ := params["CertificateAuthorityArn"].(string)
	if caArn == "" {
		return jsonErr(service.ErrValidation("CertificateAuthorityArn is required."))
	}

	cert, awsErr := store.IssueCertificate(caArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{"CertificateArn": cert.CertificateArn})
}

func handleGetCertificate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	caArn, _ := params["CertificateAuthorityArn"].(string)
	certArn, _ := params["CertificateArn"].(string)
	if caArn == "" || certArn == "" {
		return jsonErr(service.ErrValidation("CertificateAuthorityArn and CertificateArn are required."))
	}

	cert, awsErr := store.GetCertificate(caArn, certArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{
		"Certificate":      cert.CertificateBody,
		"CertificateChain": cert.CertificateChain,
	})
}

func handleRevokeCertificate(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	caArn, _ := params["CertificateAuthorityArn"].(string)
	serial, _ := params["CertificateSerial"].(string)
	reason, _ := params["RevocationReason"].(string)

	if caArn == "" || serial == "" {
		return jsonErr(service.ErrValidation("CertificateAuthorityArn and CertificateSerial are required."))
	}

	if reason == "" {
		reason = string(ReasonUnspecified)
	}

	if awsErr := store.RevokeCertificate(caArn, serial, RevocationReason(reason)); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleTagCertificateAuthority(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn, _ := params["CertificateAuthorityArn"].(string)
	if arn == "" {
		return jsonErr(service.ErrValidation("CertificateAuthorityArn is required."))
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

func handleUntagCertificateAuthority(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn, _ := params["CertificateAuthorityArn"].(string)
	if arn == "" {
		return jsonErr(service.ErrValidation("CertificateAuthorityArn is required."))
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

func handleListTags(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	arn, _ := params["CertificateAuthorityArn"].(string)
	if arn == "" {
		return jsonErr(service.ErrValidation("CertificateAuthorityArn is required."))
	}

	ca, awsErr := store.GetCA(arn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	return jsonOK(map[string]any{"Tags": tagsToMaps(ca.Tags)})
}

func handleCreatePermission(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	caArn, _ := params["CertificateAuthorityArn"].(string)
	principal, _ := params["Principal"].(string)
	sourceAccount, _ := params["SourceAccount"].(string)

	if caArn == "" || principal == "" {
		return jsonErr(service.ErrValidation("CertificateAuthorityArn and Principal are required."))
	}

	var actions []string
	if rawActions, ok := params["Actions"].([]any); ok {
		for _, a := range rawActions {
			if s, ok := a.(string); ok {
				actions = append(actions, s)
			}
		}
	}

	if awsErr := store.CreatePermission(caArn, principal, sourceAccount, actions); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}

func handleListPermissions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	caArn, _ := params["CertificateAuthorityArn"].(string)
	if caArn == "" {
		return jsonErr(service.ErrValidation("CertificateAuthorityArn is required."))
	}

	perms, awsErr := store.ListPermissions(caArn)
	if awsErr != nil {
		return jsonErr(awsErr)
	}

	items := make([]map[string]any, 0, len(perms))
	for _, p := range perms {
		items = append(items, map[string]any{
			"CertificateAuthorityArn": p.CertificateAuthorityArn,
			"Principal":               p.Principal,
			"SourceAccount":           p.SourceAccount,
			"Actions":                 p.Actions,
			"CreatedAt":               float64(p.CreatedAt.Unix()),
		})
	}

	return jsonOK(map[string]any{"Permissions": items})
}

func handleDeletePermission(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var params map[string]any
	if awsErr := parseJSON(ctx.Body, &params); awsErr != nil {
		return jsonErr(awsErr)
	}

	caArn, _ := params["CertificateAuthorityArn"].(string)
	principal, _ := params["Principal"].(string)
	sourceAccount, _ := params["SourceAccount"].(string)

	if caArn == "" || principal == "" {
		return jsonErr(service.ErrValidation("CertificateAuthorityArn and Principal are required."))
	}

	if awsErr := store.DeletePermission(caArn, principal, sourceAccount); awsErr != nil {
		return jsonErr(awsErr)
	}

	return &service.Response{StatusCode: http.StatusOK, Body: struct{}{}, Format: service.FormatJSON}, nil
}
