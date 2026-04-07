package gateway

import (
	"net/http"
	"net/url"
	"sync"

	gojson "github.com/goccy/go-json"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// stdReqPool pools *http.Request objects for the fasthttp→net/http adapter.
var stdReqPool = sync.Pool{
	New: func() any {
		return &http.Request{
			Header: make(http.Header, 8),
			URL:    &url.URL{},
		}
	},
}

// Ensure fasthttpadaptor is available (used if we need full compatibility).
var _ = fasthttpadaptor.NewFastHTTPHandler

// fastCtxPool reuses RequestContext structs for fasthttp.
var fastCtxPool = sync.Pool{
	New: func() any {
		return &service.RequestContext{}
	},
}

// FastTestModeServer returns a fasthttp.Server configured for maximum throughput.
// It bypasses net/http entirely and handles AWS requests with minimal allocation.
func FastTestModeServer(identity *service.CallerIdentity, region, accountID string, registry *routing.Registry) *fasthttp.Server {
	// Pre-resolve every service at startup — no locks on the hot path.
	all := registry.All()
	services := make(map[string]service.Service, len(all))
	for _, svc := range all {
		services[svc.Name()] = svc
	}

	handler := func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())

		// Health check fast path.
		if path == "/_cloudmock/health" {
			ctx.SetStatusCode(200)
			return
		}

		// Detect service from Authorization header credential scope.
		auth := string(ctx.Request.Header.Peek("Authorization"))
		svcName := serviceFromAuthFast(auth)
		if svcName == "" {
			// Try X-Amz-Target
			target := string(ctx.Request.Header.Peek("X-Amz-Target"))
			if target != "" {
				svcName = serviceFromTargetFast(target)
			}
		}

		if svcName == "" {
			ctx.SetStatusCode(400)
			ctx.SetContentType("text/xml")
			ctx.WriteString(`<ErrorResponse><Error><Code>MissingAuthenticationToken</Code><Message>No service detected</Message></Error></ErrorResponse>`)
			return
		}

		svc, ok := services[svcName]
		if !ok {
			ctx.SetStatusCode(503)
			ctx.SetContentType("text/xml")
			ctx.WriteString(`<ErrorResponse><Error><Code>ServiceUnavailable</Code><Message>Service not registered</Message></Error></ErrorResponse>`)
			return
		}

		// Get request body — fasthttp gives us a []byte directly, no allocation.
		body := ctx.Request.Body()

		// Detect action.
		action := ""
		if target := ctx.Request.Header.Peek("X-Amz-Target"); len(target) > 0 {
			for i := len(target) - 1; i >= 0; i-- {
				if target[i] == '.' {
					action = string(target[i+1:])
					break
				}
			}
		}
		if action == "" {
			action = string(ctx.QueryArgs().Peek("Action"))
		}

		// Build params from query string.
		var params map[string]string
		if ctx.QueryArgs().Len() > 0 {
			params = make(map[string]string, ctx.QueryArgs().Len())
			ctx.QueryArgs().VisitAll(func(key, value []byte) {
				params[string(key)] = string(value)
			})
		}

		// Build a lightweight *http.Request for services that need RawRequest.
		stdReq := stdReqPool.Get().(*http.Request)
		stdReq.Method = string(ctx.Method())
		stdReq.URL.Path = path
		stdReq.URL.RawQuery = string(ctx.QueryArgs().QueryString())
		stdReq.Host = string(ctx.Host())
		// Copy essential headers.
		for k := range stdReq.Header {
			delete(stdReq.Header, k)
		}
		ctx.Request.Header.VisitAll(func(key, value []byte) {
			stdReq.Header.Set(string(key), string(value))
		})

		// Use pooled RequestContext.
		reqCtx := fastCtxPool.Get().(*service.RequestContext)
		reqCtx.Action = action
		reqCtx.Region = region
		reqCtx.AccountID = accountID
		reqCtx.Identity = identity
		reqCtx.RawRequest = stdReq
		reqCtx.Body = body
		reqCtx.Params = params
		reqCtx.Service = svcName

		resp, svcErr := svc.HandleRequest(reqCtx)

		// Reset and return to pool.
		reqCtx.Action = ""
		reqCtx.Region = ""
		reqCtx.AccountID = ""
		reqCtx.Identity = nil
		reqCtx.RawRequest = nil
		reqCtx.Body = nil
		reqCtx.Params = nil
		reqCtx.Service = ""
		fastCtxPool.Put(reqCtx)
		stdReqPool.Put(stdReq)

		if svcErr != nil {
			if awsErr, ok := svcErr.(*service.AWSError); ok {
				ct := string(ctx.Request.Header.ContentType())
				if ct == "application/x-amz-json-1.0" || ct == "application/x-amz-json-1.1" || ct == "application/json" {
					ctx.SetContentType("application/x-amz-json-1.1")
					ctx.SetStatusCode(awsErr.StatusCode())
					data, _ := gojson.Marshal(awsErr)
					ctx.Write(data)
				} else {
					ctx.SetContentType("text/xml")
					ctx.SetStatusCode(awsErr.StatusCode())
					ctx.WriteString(`<ErrorResponse><Error><Code>`)
					ctx.WriteString(awsErr.Code)
					ctx.WriteString(`</Code><Message>`)
					ctx.WriteString(awsErr.Message)
					ctx.WriteString(`</Message></Error></ErrorResponse>`)
				}
			} else {
				ctx.SetStatusCode(500)
				ctx.WriteString(svcErr.Error())
			}
			return
		}

		if resp == nil {
			ctx.SetStatusCode(200)
			return
		}

		for k, v := range resp.Headers {
			ctx.Response.Header.Set(k, v)
		}

		if resp.RawBody != nil {
			if resp.RawContentType != "" {
				ctx.SetContentType(resp.RawContentType)
			}
			ctx.SetStatusCode(resp.StatusCode)
			ctx.Write(resp.RawBody)
			return
		}

		if resp.Body == nil {
			ctx.SetStatusCode(resp.StatusCode)
			return
		}

		data, err := gojson.Marshal(resp.Body)
		if err != nil {
			ctx.SetStatusCode(500)
			return
		}

		switch resp.Format {
		case service.FormatJSON:
			ctx.SetContentType("application/x-amz-json-1.1")
		default:
			ctx.SetContentType("text/xml")
		}
		ctx.SetStatusCode(resp.StatusCode)
		ctx.Write(data)
	}

	return &fasthttp.Server{
		Handler:               handler,
		Name:                  "cloudmock",
		ReduceMemoryUsage: false,
	}
}

// serviceFromAuthFast extracts the service from an Authorization header
// without allocating a []string via strings.Split.
func serviceFromAuthFast(auth string) string {
	const prefix = "Credential="
	idx := -1
	for i := 0; i <= len(auth)-len(prefix); i++ {
		if auth[i:i+len(prefix)] == prefix {
			idx = i + len(prefix)
			break
		}
	}
	if idx < 0 {
		return ""
	}

	// Walk to the 4th slash: AKID/date/region/SERVICE/aws4_request
	slashes := 0
	start := 0
	for i := idx; i < len(auth); i++ {
		c := auth[i]
		if c == '/' {
			slashes++
			if slashes == 3 {
				start = i + 1
			}
			if slashes == 4 {
				svc := auth[start:i]
				// Check if alias
				if svc == "s3control" {
					return "s3"
				}
				return svc
			}
		}
		if c == ',' || c == ' ' {
			break
		}
	}
	return ""
}

// serviceFromTargetFast extracts the service from an X-Amz-Target header
// without allocating.
func serviceFromTargetFast(target string) string {
	// Take part before the dot.
	svc := target
	for i := 0; i < len(target); i++ {
		if target[i] == '.' {
			svc = target[:i]
			break
		}
	}

	// Strip version suffix (after first underscore).
	for i := 0; i < len(svc); i++ {
		if svc[i] == '_' {
			svc = svc[:i]
			break
		}
	}

	// Lowercase in-place (avoid allocation when already lowercase).
	needsLower := false
	for i := 0; i < len(svc); i++ {
		if svc[i] >= 'A' && svc[i] <= 'Z' {
			needsLower = true
			break
		}
	}

	var lower string
	if needsLower {
		b := make([]byte, len(svc))
		for i := 0; i < len(svc); i++ {
			c := svc[i]
			if c >= 'A' && c <= 'Z' {
				c += 'a' - 'A'
			}
			b[i] = c
		}
		lower = string(b)
	} else {
		lower = svc
	}

	// Check targetToService map (defined in router.go).
	if mapped, ok := routing.TargetToService[lower]; ok {
		return mapped
	}
	return lower
}
