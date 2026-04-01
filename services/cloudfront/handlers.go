package cloudfront

import (
	"crypto/rand"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/neureaux/cloudmock/pkg/service"
)

// CloudFront uses REST-XML protocol. Requests are XML in body, responses are XML.
// Routes are path-based.

// ---- XML request/response types ----

type xmlDistributionConfig struct {
	XMLName           xml.Name              `xml:"DistributionConfig"`
	CallerReference   string                `xml:"CallerReference"`
	Comment           string                `xml:"Comment"`
	DefaultRootObject string                `xml:"DefaultRootObject"`
	Enabled           bool                  `xml:"Enabled"`
	PriceClass        string                `xml:"PriceClass"`
	Origins           xmlOrigins            `xml:"Origins"`
	DefaultCacheBehavior *xmlCacheBehavior  `xml:"DefaultCacheBehavior"`
	CacheBehaviors    *xmlCacheBehaviorList `xml:"CacheBehaviors"`
}

type xmlOrigins struct {
	Quantity int         `xml:"Quantity"`
	Items    []xmlOrigin `xml:"Items>Origin"`
}

type xmlOrigin struct {
	Id         string              `xml:"Id"`
	DomainName string              `xml:"DomainName"`
	OriginPath string              `xml:"OriginPath"`
	S3Config   *xmlS3OriginConfig  `xml:"S3OriginConfig"`
	CustomConfig *xmlCustomOriginConfig `xml:"CustomOriginConfig"`
}

type xmlS3OriginConfig struct {
	OriginAccessIdentity string `xml:"OriginAccessIdentity"`
}

type xmlCustomOriginConfig struct {
	HTTPPort             int    `xml:"HTTPPort"`
	HTTPSPort            int    `xml:"HTTPSPort"`
	OriginProtocolPolicy string `xml:"OriginProtocolPolicy"`
}

type xmlCacheBehavior struct {
	PathPattern          string            `xml:"PathPattern,omitempty"`
	TargetOriginId       string            `xml:"TargetOriginId"`
	ViewerProtocolPolicy string            `xml:"ViewerProtocolPolicy"`
	Compress             bool              `xml:"Compress"`
	MinTTL               int64             `xml:"MinTTL"`
	MaxTTL               int64             `xml:"MaxTTL"`
	DefaultTTL           int64             `xml:"DefaultTTL"`
	ForwardedValues      *xmlForwardedValues `xml:"ForwardedValues"`
}

type xmlForwardedValues struct {
	QueryString bool     `xml:"QueryString"`
	Cookies     xmlCookies `xml:"Cookies"`
	Headers     *xmlHeaders `xml:"Headers"`
}

type xmlCookies struct {
	Forward string `xml:"Forward"`
}

type xmlHeaders struct {
	Quantity int      `xml:"Quantity"`
	Items    []string `xml:"Items>Name"`
}

type xmlCacheBehaviorList struct {
	Quantity int                `xml:"Quantity"`
	Items    []xmlCacheBehavior `xml:"Items>CacheBehavior"`
}

// ---- Response types ----

type xmlDistribution struct {
	XMLName            xml.Name              `xml:"Distribution"`
	Id                 string                `xml:"Id"`
	ARN                string                `xml:"ARN"`
	Status             string                `xml:"Status"`
	DomainName         string                `xml:"DomainName"`
	LastModifiedTime   string                `xml:"LastModifiedTime"`
	DistributionConfig xmlDistributionConfig `xml:"DistributionConfig"`
}

type xmlCreateDistributionResponse struct {
	XMLName xml.Name        `xml:"CreateDistributionResult"`
	Dist    xmlDistribution `xml:"Distribution"`
}

type xmlDistributionList struct {
	XMLName  xml.Name          `xml:"DistributionList"`
	Quantity int               `xml:"Quantity"`
	Items    []xmlDistSummary  `xml:"Items>DistributionSummary"`
}

type xmlDistSummary struct {
	Id                 string `xml:"Id"`
	ARN                string `xml:"ARN"`
	Status             string `xml:"Status"`
	DomainName         string `xml:"DomainName"`
	Comment            string `xml:"Comment"`
	Enabled            bool   `xml:"Enabled"`
	PriceClass         string `xml:"PriceClass"`
	LastModifiedTime   string `xml:"LastModifiedTime"`
}

type xmlGetDistributionResponse struct {
	XMLName xml.Name        `xml:"GetDistributionResult"`
	Dist    xmlDistribution `xml:"Distribution"`
}

type xmlInvalidation struct {
	XMLName         xml.Name         `xml:"Invalidation"`
	Id              string           `xml:"Id"`
	Status          string           `xml:"Status"`
	CreateTime      string           `xml:"CreateTime"`
	InvalidationBatch xmlInvalidationBatch `xml:"InvalidationBatch"`
}

type xmlInvalidationBatch struct {
	CallerReference string   `xml:"CallerReference"`
	Paths           xmlPaths `xml:"Paths"`
}

type xmlPaths struct {
	Quantity int      `xml:"Quantity"`
	Items    []string `xml:"Items>Path"`
}

type xmlInvalidationList struct {
	XMLName  xml.Name               `xml:"InvalidationList"`
	Quantity int                    `xml:"Quantity"`
	Items    []xmlInvalidationSummary `xml:"Items>InvalidationSummary"`
}

type xmlInvalidationSummary struct {
	Id         string `xml:"Id"`
	Status     string `xml:"Status"`
	CreateTime string `xml:"CreateTime"`
}

type xmlTagsPayload struct {
	XMLName xml.Name `xml:"Tags"`
	Items   []xmlTag `xml:"Items>Tag"`
}

type xmlTag struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type xmlTagKeysPayload struct {
	XMLName xml.Name `xml:"TagKeys"`
	Items   []string `xml:"Items>Key"`
}

type xmlTagListResponse struct {
	XMLName xml.Name `xml:"Tags"`
	Items   []xmlTag `xml:"Items>Tag"`
}

// ---- Conversion helpers ----

func distToXML(d *Distribution) xmlDistribution {
	cfg := xmlDistributionConfig{
		CallerReference:   d.CallerReference,
		Comment:           d.Comment,
		DefaultRootObject: d.DefaultRootObject,
		Enabled:           d.Enabled,
		PriceClass:        d.PriceClass,
	}

	origins := make([]xmlOrigin, 0, len(d.Origins))
	for _, o := range d.Origins {
		xo := xmlOrigin{
			Id:         o.ID,
			DomainName: o.DomainName,
			OriginPath: o.OriginPath,
		}
		if o.S3Config != nil {
			xo.S3Config = &xmlS3OriginConfig{OriginAccessIdentity: o.S3Config.OriginAccessIdentity}
		}
		if o.CustomConfig != nil {
			xo.CustomConfig = &xmlCustomOriginConfig{
				HTTPPort:             o.CustomConfig.HTTPPort,
				HTTPSPort:            o.CustomConfig.HTTPSPort,
				OriginProtocolPolicy: o.CustomConfig.OriginProtocolPolicy,
			}
		}
		origins = append(origins, xo)
	}
	cfg.Origins = xmlOrigins{Quantity: len(origins), Items: origins}

	if d.DefaultCacheBehavior != nil {
		cfg.DefaultCacheBehavior = cacheBehaviorToXML(d.DefaultCacheBehavior)
	}

	if len(d.CacheBehaviors) > 0 {
		items := make([]xmlCacheBehavior, 0, len(d.CacheBehaviors))
		for _, b := range d.CacheBehaviors {
			items = append(items, *cacheBehaviorToXML(&b))
		}
		cfg.CacheBehaviors = &xmlCacheBehaviorList{Quantity: len(items), Items: items}
	}

	return xmlDistribution{
		Id:                 d.ID,
		ARN:                d.ARN,
		Status:             d.Status,
		DomainName:         d.DomainName,
		LastModifiedTime:   d.LastModified.Format("2006-01-02T15:04:05Z"),
		DistributionConfig: cfg,
	}
}

func cacheBehaviorToXML(b *CacheBehavior) *xmlCacheBehavior {
	xb := &xmlCacheBehavior{
		PathPattern:          b.PathPattern,
		TargetOriginId:       b.TargetOriginID,
		ViewerProtocolPolicy: b.ViewerProtocolPolicy,
		Compress:             b.Compress,
		MinTTL:               b.MinTTL,
		MaxTTL:               b.MaxTTL,
		DefaultTTL:           b.DefaultTTL,
	}
	if b.ForwardedValues != nil {
		fv := &xmlForwardedValues{
			QueryString: b.ForwardedValues.QueryString,
			Cookies:     xmlCookies{Forward: b.ForwardedValues.Cookies},
		}
		if len(b.ForwardedValues.Headers) > 0 {
			fv.Headers = &xmlHeaders{
				Quantity: len(b.ForwardedValues.Headers),
				Items:    b.ForwardedValues.Headers,
			}
		}
		xb.ForwardedValues = fv
	}
	return xb
}

func originsFromXML(xos []xmlOrigin) []Origin {
	origins := make([]Origin, 0, len(xos))
	for _, xo := range xos {
		o := Origin{
			ID:         xo.Id,
			DomainName: xo.DomainName,
			OriginPath: xo.OriginPath,
		}
		if xo.S3Config != nil {
			o.S3Config = &S3OriginConfig{OriginAccessIdentity: xo.S3Config.OriginAccessIdentity}
		}
		if xo.CustomConfig != nil {
			o.CustomConfig = &CustomOriginConfig{
				HTTPPort:             xo.CustomConfig.HTTPPort,
				HTTPSPort:            xo.CustomConfig.HTTPSPort,
				OriginProtocolPolicy: xo.CustomConfig.OriginProtocolPolicy,
			}
		}
		origins = append(origins, o)
	}
	return origins
}

func cacheBehaviorFromXML(xb *xmlCacheBehavior) *CacheBehavior {
	if xb == nil {
		return nil
	}
	b := &CacheBehavior{
		PathPattern:          xb.PathPattern,
		TargetOriginID:       xb.TargetOriginId,
		ViewerProtocolPolicy: xb.ViewerProtocolPolicy,
		Compress:             xb.Compress,
		MinTTL:               xb.MinTTL,
		MaxTTL:               xb.MaxTTL,
		DefaultTTL:           xb.DefaultTTL,
	}
	if xb.ForwardedValues != nil {
		b.ForwardedValues = &ForwardedValues{
			QueryString: xb.ForwardedValues.QueryString,
			Cookies:     xb.ForwardedValues.Cookies.Forward,
		}
		if xb.ForwardedValues.Headers != nil {
			b.ForwardedValues.Headers = xb.ForwardedValues.Headers.Items
		}
	}
	return b
}

func cacheBehaviorsFromXML(xbl *xmlCacheBehaviorList) []CacheBehavior {
	if xbl == nil || len(xbl.Items) == 0 {
		return nil
	}
	behaviors := make([]CacheBehavior, 0, len(xbl.Items))
	for _, xb := range xbl.Items {
		b := cacheBehaviorFromXML(&xb)
		if b != nil {
			behaviors = append(behaviors, *b)
		}
	}
	return behaviors
}

// ---- Request routing ----

// HandleRESTRequest routes CloudFront REST-XML requests based on path and method.
func HandleRESTRequest(ctx *service.RequestContext, store *Store, locator ServiceLocator) (*service.Response, error) {
	r := ctx.RawRequest
	method := r.Method
	path := strings.TrimRight(r.URL.Path, "/")

	// Distribution routes
	const distPrefix = "/2020-05-31/distribution"
	const tagPrefix = "/2020-05-31/tagging"

	// Tagging
	if strings.HasPrefix(path, tagPrefix) {
		resource := r.URL.Query().Get("Resource")
		switch method {
		case http.MethodPost:
			operation := r.URL.Query().Get("Operation")
			if operation == "Untag" {
				return handleUntagResource(ctx, store, resource)
			}
			return handleTagResource(ctx, store, resource)
		case http.MethodGet:
			return handleListTagsForResource(ctx, store, resource)
		}
		return cfNotImplemented()
	}

	if !strings.HasPrefix(path, distPrefix) {
		return cfNotImplemented()
	}

	rest := path[len(distPrefix):]

	// POST /2020-05-31/distribution -> CreateDistribution
	// GET  /2020-05-31/distribution -> ListDistributions
	if rest == "" {
		switch method {
		case http.MethodPost:
			return handleCreateDistribution(ctx, store, locator)
		case http.MethodGet:
			return handleListDistributions(ctx, store)
		}
		return cfNotImplemented()
	}

	parts := strings.SplitN(strings.TrimPrefix(rest, "/"), "/", 2)
	distID := parts[0]

	// GET    /2020-05-31/distribution/{id} -> GetDistribution
	// PUT    /2020-05-31/distribution/{id}/config -> UpdateDistribution
	// DELETE /2020-05-31/distribution/{id} -> DeleteDistribution
	if len(parts) == 1 {
		switch method {
		case http.MethodGet:
			return handleGetDistribution(ctx, store, distID)
		case http.MethodDelete:
			return handleDeleteDistribution(ctx, store, distID)
		}
		return cfNotImplemented()
	}

	subPath := parts[1]

	if subPath == "config" && method == http.MethodPut {
		return handleUpdateDistribution(ctx, store, distID)
	}

	// Invalidation routes: /2020-05-31/distribution/{id}/invalidation
	if strings.HasPrefix(subPath, "invalidation") {
		invRest := subPath[len("invalidation"):]

		if invRest == "" {
			switch method {
			case http.MethodPost:
				return handleCreateInvalidation(ctx, store, distID)
			case http.MethodGet:
				return handleListInvalidations(ctx, store, distID)
			}
			return cfNotImplemented()
		}

		invID := strings.TrimPrefix(invRest, "/")
		if method == http.MethodGet {
			return handleGetInvalidation(ctx, store, distID, invID)
		}
	}

	return cfNotImplemented()
}

// ---- Distribution handlers ----

func handleCreateDistribution(ctx *service.RequestContext, store *Store, locator ServiceLocator) (*service.Response, error) {
	var cfg xmlDistributionConfig
	if err := xml.Unmarshal(ctx.Body, &cfg); err != nil {
		return xmlErr(service.ErrValidation("Invalid XML request body."))
	}

	origins := originsFromXML(cfg.Origins.Items)

	// Validate origins via locator if available.
	if locator != nil {
		for _, origin := range origins {
			if err := validateOrigin(origin, locator); err != nil {
				return xmlErr(err)
			}
		}
	}

	defaultBehavior := cacheBehaviorFromXML(cfg.DefaultCacheBehavior)
	behaviors := cacheBehaviorsFromXML(cfg.CacheBehaviors)

	dist := store.CreateDistribution(cfg.CallerReference, cfg.Comment, cfg.DefaultRootObject,
		cfg.PriceClass, cfg.Enabled, origins, defaultBehavior, behaviors)

	resp := distToXML(dist)
	return &service.Response{
		StatusCode: http.StatusCreated,
		Body:       resp,
		Format:     service.FormatXML,
		Headers:    map[string]string{"ETag": dist.ETag, "Location": fmt.Sprintf("/2020-05-31/distribution/%s", dist.ID)},
	}, nil
}

func handleGetDistribution(ctx *service.RequestContext, store *Store, id string) (*service.Response, error) {
	dist, ok := store.GetDistribution(id)
	if !ok {
		return xmlErr(service.NewAWSError("NoSuchDistribution",
			"The specified distribution does not exist.", http.StatusNotFound))
	}

	resp := distToXML(dist)
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       resp,
		Format:     service.FormatXML,
		Headers:    map[string]string{"ETag": dist.ETag},
	}, nil
}

func handleListDistributions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	dists := store.ListDistributions()

	summaries := make([]xmlDistSummary, 0, len(dists))
	for _, d := range dists {
		summaries = append(summaries, xmlDistSummary{
			Id:               d.ID,
			ARN:              d.ARN,
			Status:           d.Status,
			DomainName:       d.DomainName,
			Comment:          d.Comment,
			Enabled:          d.Enabled,
			PriceClass:       d.PriceClass,
			LastModifiedTime: d.LastModified.Format("2006-01-02T15:04:05Z"),
		})
	}

	return xmlOK(&xmlDistributionList{Quantity: len(summaries), Items: summaries})
}

func handleUpdateDistribution(ctx *service.RequestContext, store *Store, id string) (*service.Response, error) {
	var cfg xmlDistributionConfig
	if err := xml.Unmarshal(ctx.Body, &cfg); err != nil {
		return xmlErr(service.ErrValidation("Invalid XML request body."))
	}

	origins := originsFromXML(cfg.Origins.Items)
	defaultBehavior := cacheBehaviorFromXML(cfg.DefaultCacheBehavior)
	behaviors := cacheBehaviorsFromXML(cfg.CacheBehaviors)
	enabled := &cfg.Enabled

	dist, ok := store.UpdateDistribution(id, cfg.Comment, cfg.DefaultRootObject, cfg.PriceClass,
		enabled, origins, defaultBehavior, behaviors)
	if !ok {
		return xmlErr(service.NewAWSError("NoSuchDistribution",
			"The specified distribution does not exist.", http.StatusNotFound))
	}

	resp := distToXML(dist)
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       resp,
		Format:     service.FormatXML,
		Headers:    map[string]string{"ETag": dist.ETag},
	}, nil
}

func handleDeleteDistribution(ctx *service.RequestContext, store *Store, id string) (*service.Response, error) {
	if !store.DeleteDistribution(id) {
		return xmlErr(service.NewAWSError("NoSuchDistribution",
			"The specified distribution does not exist.", http.StatusNotFound))
	}

	return &service.Response{
		StatusCode: http.StatusNoContent,
		Format:     service.FormatXML,
	}, nil
}

// ---- Invalidation handlers ----

func handleCreateInvalidation(ctx *service.RequestContext, store *Store, distID string) (*service.Response, error) {
	var batch xmlInvalidationBatch
	if err := xml.Unmarshal(ctx.Body, &batch); err != nil {
		return xmlErr(service.ErrValidation("Invalid XML request body."))
	}

	inv, ok := store.CreateInvalidation(distID, batch.CallerReference, batch.Paths.Items)
	if !ok {
		return xmlErr(service.NewAWSError("NoSuchDistribution",
			"The specified distribution does not exist.", http.StatusNotFound))
	}

	resp := xmlInvalidation{
		Id:         inv.ID,
		Status:     inv.Status,
		CreateTime: inv.CreateTime.Format("2006-01-02T15:04:05Z"),
		InvalidationBatch: xmlInvalidationBatch{
			CallerReference: inv.CallerReference,
			Paths:           xmlPaths{Quantity: len(inv.Paths), Items: inv.Paths},
		},
	}

	return &service.Response{
		StatusCode: http.StatusCreated,
		Body:       resp,
		Format:     service.FormatXML,
		Headers:    map[string]string{"Location": fmt.Sprintf("/2020-05-31/distribution/%s/invalidation/%s", distID, inv.ID)},
	}, nil
}

func handleGetInvalidation(ctx *service.RequestContext, store *Store, distID, invID string) (*service.Response, error) {
	inv, ok := store.GetInvalidation(distID, invID)
	if !ok {
		return xmlErr(service.NewAWSError("NoSuchInvalidation",
			"The specified invalidation does not exist.", http.StatusNotFound))
	}

	resp := xmlInvalidation{
		Id:         inv.ID,
		Status:     inv.Status,
		CreateTime: inv.CreateTime.Format("2006-01-02T15:04:05Z"),
		InvalidationBatch: xmlInvalidationBatch{
			CallerReference: inv.CallerReference,
			Paths:           xmlPaths{Quantity: len(inv.Paths), Items: inv.Paths},
		},
	}

	return xmlOK(&resp)
}

func handleListInvalidations(ctx *service.RequestContext, store *Store, distID string) (*service.Response, error) {
	invs, ok := store.ListInvalidations(distID)
	if !ok {
		return xmlErr(service.NewAWSError("NoSuchDistribution",
			"The specified distribution does not exist.", http.StatusNotFound))
	}

	summaries := make([]xmlInvalidationSummary, 0, len(invs))
	for _, inv := range invs {
		summaries = append(summaries, xmlInvalidationSummary{
			Id:         inv.ID,
			Status:     inv.Status,
			CreateTime: inv.CreateTime.Format("2006-01-02T15:04:05Z"),
		})
	}

	return xmlOK(&xmlInvalidationList{Quantity: len(summaries), Items: summaries})
}

// ---- Tag handlers ----

func handleTagResource(ctx *service.RequestContext, store *Store, arn string) (*service.Response, error) {
	if arn == "" {
		return xmlErr(service.ErrValidation("Resource ARN is required."))
	}

	var payload xmlTagsPayload
	if err := xml.Unmarshal(ctx.Body, &payload); err != nil {
		return xmlErr(service.ErrValidation("Invalid XML request body."))
	}

	tags := make(map[string]string, len(payload.Items))
	for _, t := range payload.Items {
		tags[t.Key] = t.Value
	}

	if !store.TagResource(arn, tags) {
		return xmlErr(service.NewAWSError("NoSuchResource",
			"The specified resource does not exist.", http.StatusNotFound))
	}

	return &service.Response{
		StatusCode: http.StatusNoContent,
		Format:     service.FormatXML,
	}, nil
}

func handleUntagResource(ctx *service.RequestContext, store *Store, arn string) (*service.Response, error) {
	if arn == "" {
		return xmlErr(service.ErrValidation("Resource ARN is required."))
	}

	var payload xmlTagKeysPayload
	if err := xml.Unmarshal(ctx.Body, &payload); err != nil {
		return xmlErr(service.ErrValidation("Invalid XML request body."))
	}

	if !store.UntagResource(arn, payload.Items) {
		return xmlErr(service.NewAWSError("NoSuchResource",
			"The specified resource does not exist.", http.StatusNotFound))
	}

	return &service.Response{
		StatusCode: http.StatusNoContent,
		Format:     service.FormatXML,
	}, nil
}

func handleListTagsForResource(ctx *service.RequestContext, store *Store, arn string) (*service.Response, error) {
	if arn == "" {
		return xmlErr(service.ErrValidation("Resource ARN is required."))
	}

	tags, ok := store.ListTagsForResource(arn)
	if !ok {
		return xmlErr(service.NewAWSError("NoSuchResource",
			"The specified resource does not exist.", http.StatusNotFound))
	}

	xmlTags := make([]xmlTag, 0, len(tags))
	for k, v := range tags {
		xmlTags = append(xmlTags, xmlTag{Key: k, Value: v})
	}

	return xmlOK(&xmlTagListResponse{Items: xmlTags})
}

// ---- helpers ----

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

func cfNotImplemented() (*service.Response, error) {
	return xmlErr(service.NewAWSError("NotImplemented",
		"This method and path combination is not implemented by cloudmock.", http.StatusNotImplemented))
}

// parseJSON is used for tag operations which may also accept JSON in some SDK paths.
func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return service.ErrValidation("Invalid JSON request body.")
	}
	return nil
}

// validateOrigin checks that S3 bucket and ELB origins exist via the ServiceLocator.
// Returns an AWSError if the origin cannot be found.
func validateOrigin(origin Origin, locator ServiceLocator) *service.AWSError {
	domainName := origin.DomainName

	// Check S3 bucket origins (*.s3.amazonaws.com or *.s3.*.amazonaws.com)
	if strings.HasSuffix(domainName, ".s3.amazonaws.com") || strings.Contains(domainName, ".s3.") {
		// Extract bucket name from domain
		bucketName := strings.Split(domainName, ".s3")[0]
		if bucketName != "" {
			s3Svc, err := locator.Lookup("s3")
			if err == nil {
				body, _ := json.Marshal(map[string]any{
					"Bucket": bucketName,
				})
				_, err := s3Svc.HandleRequest(&service.RequestContext{
					Action:     "HeadBucket",
					Body:       body,
					RawRequest: httptest.NewRequest(http.MethodHead, "/"+bucketName, nil),
				})
				if err != nil {
					return service.NewAWSError("InvalidOrigin",
						fmt.Sprintf("The specified origin server does not exist or is not valid. S3 bucket %q not found.", bucketName),
						http.StatusBadRequest)
				}
			}
			// If S3 service not available, degrade gracefully
		}
	}

	// Check ELB origins (*.elb.amazonaws.com or *.elb.*.amazonaws.com)
	if strings.Contains(domainName, ".elb.") && strings.HasSuffix(domainName, ".amazonaws.com") {
		elbSvc, err := locator.Lookup("elasticloadbalancing")
		if err == nil {
			// Try to describe load balancers; if the ELB service is up but
			// no matching LB is found, treat as invalid origin.
			body, _ := json.Marshal(map[string]any{})
			_, err := elbSvc.HandleRequest(&service.RequestContext{
				Action:     "DescribeLoadBalancers",
				Body:       body,
				RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
			})
			if err != nil {
				return service.NewAWSError("InvalidOrigin",
					fmt.Sprintf("The specified origin server does not exist or is not valid. ELB origin %q not found.", domainName),
					http.StatusBadRequest)
			}
		}
		// If ELB service not available, degrade gracefully
	}

	return nil
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
