package ses

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"

	"github.com/neureaux/cloudmock/pkg/service"
)

// ---- shared XML types ----

type xmlResponseMetadata struct {
	RequestID string `xml:"RequestId"`
}

const sesXmlns = "http://ses.amazonaws.com/doc/2010-12-01/"

// ---- SendEmail ----

type xmlSendEmailResponse struct {
	XMLName xml.Name            `xml:"SendEmailResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  xmlSendEmailResult  `xml:"SendEmailResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

type xmlSendEmailResult struct {
	MessageId string `xml:"MessageId"`
}

func handleSendEmail(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)

	source := form.Get("Source")
	if source == "" {
		return xmlErr(service.ErrValidation("Source is required."))
	}

	subject := form.Get("Message.Subject.Data")
	textBody := form.Get("Message.Body.Text.Data")
	htmlBody := form.Get("Message.Body.Html.Data")

	// Collect To/Cc/Bcc from indexed member lists.
	toAddrs := collectMembers(form, "Destination.ToAddresses.member.")
	ccAddrs := collectMembers(form, "Destination.CcAddresses.member.")
	bccAddrs := collectMembers(form, "Destination.BccAddresses.member.")

	if len(toAddrs) == 0 && len(ccAddrs) == 0 && len(bccAddrs) == 0 {
		return xmlErr(service.ErrValidation("At least one recipient address is required."))
	}

	email := &Email{
		Source:       source,
		ToAddresses:  toAddrs,
		CcAddresses:  ccAddrs,
		BccAddresses: bccAddrs,
		Subject:      subject,
		TextBody:     textBody,
		HtmlBody:     htmlBody,
	}

	msgID := store.StoreEmail(email)

	resp := &xmlSendEmailResponse{
		Xmlns:  sesXmlns,
		Result: xmlSendEmailResult{MessageId: msgID},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- SendRawEmail ----

type xmlSendRawEmailResponse struct {
	XMLName xml.Name               `xml:"SendRawEmailResponse"`
	Xmlns   string                 `xml:"xmlns,attr"`
	Result  xmlSendRawEmailResult  `xml:"SendRawEmailResult"`
	Meta    xmlResponseMetadata    `xml:"ResponseMetadata"`
}

type xmlSendRawEmailResult struct {
	MessageId string `xml:"MessageId"`
}

func handleSendRawEmail(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)

	rawData := form.Get("RawMessage.Data")
	if rawData == "" {
		return xmlErr(service.ErrValidation("RawMessage.Data is required."))
	}

	// Decode to verify it is valid base64; store the decoded content.
	decoded, err := base64.StdEncoding.DecodeString(rawData)
	if err != nil {
		// Try URL-safe base64 as some SDKs use it.
		decoded, err = base64.URLEncoding.DecodeString(rawData)
		if err != nil {
			return xmlErr(service.ErrValidation("RawMessage.Data is not valid base64."))
		}
	}

	email := &Email{
		RawMessage: string(decoded),
	}
	msgID := store.StoreEmail(email)

	resp := &xmlSendRawEmailResponse{
		Xmlns:  sesXmlns,
		Result: xmlSendRawEmailResult{MessageId: msgID},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- VerifyEmailIdentity ----

type xmlVerifyEmailIdentityResponse struct {
	XMLName xml.Name            `xml:"VerifyEmailIdentityResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct{}            `xml:"VerifyEmailIdentityResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleVerifyEmailIdentity(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	email := form.Get("EmailAddress")
	if email == "" {
		return xmlErr(service.ErrValidation("EmailAddress is required."))
	}

	store.VerifyIdentity(email)

	resp := &xmlVerifyEmailIdentityResponse{
		Xmlns: sesXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- ListIdentities ----

type xmlListIdentitiesResponse struct {
	XMLName xml.Name               `xml:"ListIdentitiesResponse"`
	Xmlns   string                 `xml:"xmlns,attr"`
	Result  xmlListIdentitiesResult `xml:"ListIdentitiesResult"`
	Meta    xmlResponseMetadata    `xml:"ResponseMetadata"`
}

type xmlListIdentitiesResult struct {
	Identities []string `xml:"Identities>member"`
}

func handleListIdentities(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	ids := store.ListIdentities()

	resp := &xmlListIdentitiesResponse{
		Xmlns:  sesXmlns,
		Result: xmlListIdentitiesResult{Identities: ids},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- DeleteIdentity ----

type xmlDeleteIdentityResponse struct {
	XMLName xml.Name            `xml:"DeleteIdentityResponse"`
	Xmlns   string              `xml:"xmlns,attr"`
	Result  struct{}            `xml:"DeleteIdentityResult"`
	Meta    xmlResponseMetadata `xml:"ResponseMetadata"`
}

func handleDeleteIdentity(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)
	identity := form.Get("Identity")
	if identity == "" {
		return xmlErr(service.ErrValidation("Identity is required."))
	}

	store.DeleteIdentity(identity)

	resp := &xmlDeleteIdentityResponse{
		Xmlns: sesXmlns,
		Meta:  xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- GetIdentityVerificationAttributes ----

type xmlGetIdentityVerificationAttributesResponse struct {
	XMLName xml.Name                                    `xml:"GetIdentityVerificationAttributesResponse"`
	Xmlns   string                                      `xml:"xmlns,attr"`
	Result  xmlGetIdentityVerificationAttributesResult  `xml:"GetIdentityVerificationAttributesResult"`
	Meta    xmlResponseMetadata                         `xml:"ResponseMetadata"`
}

type xmlGetIdentityVerificationAttributesResult struct {
	VerificationAttributes []xmlVerificationEntry `xml:"VerificationAttributes>entry"`
}

type xmlVerificationEntry struct {
	Key   string                `xml:"key"`
	Value xmlVerificationAttrs `xml:"value"`
}

type xmlVerificationAttrs struct {
	VerificationStatus string `xml:"VerificationStatus"`
}

func handleGetIdentityVerificationAttributes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	form := parseForm(ctx)

	// Identities.member.1, Identities.member.2, ...
	identities := collectMembers(form, "Identities.member.")

	entries := make([]xmlVerificationEntry, 0, len(identities))
	for _, id := range identities {
		entries = append(entries, xmlVerificationEntry{
			Key:   id,
			Value: xmlVerificationAttrs{VerificationStatus: "Success"},
		})
	}

	resp := &xmlGetIdentityVerificationAttributesResponse{
		Xmlns: sesXmlns,
		Result: xmlGetIdentityVerificationAttributesResult{
			VerificationAttributes: entries,
		},
		Meta: xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- ListVerifiedEmailAddresses ----

type xmlListVerifiedEmailAddressesResponse struct {
	XMLName xml.Name                              `xml:"ListVerifiedEmailAddressesResponse"`
	Xmlns   string                                `xml:"xmlns,attr"`
	Result  xmlListVerifiedEmailAddressesResult   `xml:"ListVerifiedEmailAddressesResult"`
	Meta    xmlResponseMetadata                   `xml:"ResponseMetadata"`
}

type xmlListVerifiedEmailAddressesResult struct {
	VerifiedEmailAddresses []string `xml:"VerifiedEmailAddresses>member"`
}

func handleListVerifiedEmailAddresses(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	addrs := store.ListVerifiedEmailAddresses()

	resp := &xmlListVerifiedEmailAddressesResponse{
		Xmlns:  sesXmlns,
		Result: xmlListVerifiedEmailAddressesResult{VerifiedEmailAddresses: addrs},
		Meta:   xmlResponseMetadata{RequestID: newUUID()},
	}
	return xmlOK(resp)
}

// ---- helper functions ----

// parseForm merges query-string params and form-encoded body into url.Values.
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

// collectMembers gathers indexed list values with a given prefix (e.g. "Destination.ToAddresses.member.").
func collectMembers(form url.Values, prefix string) []string {
	out := make([]string, 0)
	for i := 1; ; i++ {
		v := form.Get(fmt.Sprintf("%s%d", prefix, i))
		if v == "" {
			break
		}
		out = append(out, v)
	}
	return out
}

// xmlOK wraps a response body in a 200 XML response.
func xmlOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatXML,
	}, nil
}

// xmlErr wraps an AWSError in an XML error response.
func xmlErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatXML}, awsErr
}

// newUUID returns a random UUID-shaped identifier.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

