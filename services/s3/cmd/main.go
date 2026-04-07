package main

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	s3svc "github.com/Viridian-Inc/cloudmock/services/s3"
)

func main() {
	s3 := s3svc.New()
	mux := http.NewServeMux()

	mux.HandleFunc("/_health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"service": s3.Name(),
			"status":  "ok",
		})
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := service.ParseRequestBody(r)
		ctx := &service.RequestContext{
			RawRequest: r,
			Body:       body,
			Service:    "s3",
			Region:     "us-east-1",
			AccountID:  "000000000000",
			Params:     make(map[string]string),
			Identity: &service.CallerIdentity{
				AccountID: "000000000000",
				IsRoot:    true,
			},
		}
		for key, values := range r.URL.Query() {
			if len(values) > 0 {
				ctx.Params[key] = values[0]
			}
		}

		resp, err := s3.HandleRequest(ctx)
		if err != nil {
			if awsErr, ok := err.(*service.AWSError); ok {
				_ = service.WriteErrorResponse(w, awsErr, service.FormatXML)
				return
			}
			_ = service.WriteErrorResponse(w, service.NewAWSError("InternalError", err.Error(), 500), service.FormatXML)
			return
		}

		if resp.Body == nil {
			w.WriteHeader(resp.StatusCode)
			return
		}

		switch resp.Format {
		case service.FormatJSON:
			_ = service.WriteJSONResponse(w, resp.StatusCode, resp.Body)
		default:
			_ = service.WriteXMLResponse(w, resp.StatusCode, resp.Body)
		}
	})

	slog.Info("S3 service starting", "port", 8080)
	if err := http.ListenAndServe(":8080", mux); err != nil {
		slog.Error("S3 service failed", "error", err)
	}
}
