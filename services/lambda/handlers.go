package lambda

import (
	"crypto/sha256"
	"encoding/base64"
	gojson "github.com/goccy/go-json"
	"fmt"
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ---- JSON helpers ----

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonCreated(body any) (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusCreated,
		Body:       body,
		Format:     service.FormatJSON,
	}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func jsonNoContent() (*service.Response, error) {
	return &service.Response{
		StatusCode: http.StatusNoContent,
		Format:     service.FormatJSON,
	}, nil
}

// ---- Response types ----

// functionConfigResponse builds the JSON configuration response for a function.
func functionConfigResponse(fn *Function) map[string]any {
	resp := map[string]any{
		"FunctionName":   fn.FunctionName,
		"FunctionArn":    fn.FunctionArn,
		"Runtime":        fn.Runtime,
		"Role":           fn.Role,
		"Handler":        fn.Handler,
		"CodeSize":       fn.CodeSize,
		"Description":    fn.Description,
		"Timeout":        fn.Timeout,
		"MemorySize":     fn.MemorySize,
		"LastModified":   fn.LastModified,
		"CodeSha256":     fn.CodeSha256,
		"Version":        fn.Version,
		"State":          "Active",
		"LastUpdateStatus": "Successful",
	}
	if fn.Environment != nil && len(fn.Environment.Variables) > 0 {
		resp["Environment"] = fn.Environment
	}
	return resp
}

// ---- S3ObjectGetter ----

// S3ObjectGetter allows retrieving object data from the S3 service.
type S3ObjectGetter interface {
	GetObjectData(bucket, key string) ([]byte, error)
}

// ---- CreateFunction ----

type createFunctionRequest struct {
	FunctionName string `json:"FunctionName"`
	Runtime      string `json:"Runtime"`
	Role         string `json:"Role"`
	Handler      string `json:"Handler"`
	Description  string `json:"Description"`
	Timeout      int    `json:"Timeout"`
	MemorySize   int    `json:"MemorySize"`
	Code         struct {
		ZipFile  string `json:"ZipFile"`
		S3Bucket string `json:"S3Bucket"`
		S3Key    string `json:"S3Key"`
	} `json:"Code"`
	Environment *Environment `json:"Environment"`
}

func handleCreateFunction(ctx *service.RequestContext, store *FunctionStore, executor *Executor, locator ServiceLocator) (*service.Response, error) {
	var req createFunctionRequest
	if err := gojson.Unmarshal(ctx.Body, &req); err != nil {
		return jsonErr(service.ErrValidation("invalid request body: " + err.Error()))
	}

	if req.FunctionName == "" {
		return jsonErr(service.ErrValidation("FunctionName is required"))
	}
	if req.Runtime == "" {
		return jsonErr(service.ErrValidation("Runtime is required"))
	}
	if req.Handler == "" {
		return jsonErr(service.ErrValidation("Handler is required"))
	}
	if req.Role == "" {
		return jsonErr(service.ErrValidation("Role is required"))
	}

	// Resolve code.
	var codeBytes []byte
	var err error

	if req.Code.ZipFile != "" {
		codeBytes, err = base64.StdEncoding.DecodeString(req.Code.ZipFile)
		if err != nil {
			return jsonErr(service.ErrValidation("invalid base64 in Code.ZipFile: " + err.Error()))
		}
	} else if req.Code.S3Bucket != "" && req.Code.S3Key != "" {
		codeBytes, err = getCodeFromS3(locator, req.Code.S3Bucket, req.Code.S3Key)
		if err != nil {
			return jsonErr(service.NewAWSError("InvalidParameterValueException",
				fmt.Sprintf("Error getting object from S3: %v", err), http.StatusBadRequest))
		}
	} else {
		return jsonErr(service.ErrValidation("Code.ZipFile or Code.S3Bucket+S3Key is required"))
	}

	// Defaults.
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 3
	}
	memorySize := req.MemorySize
	if memorySize <= 0 {
		memorySize = 128
	}

	hash := sha256.Sum256(codeBytes)
	codeSha256 := base64.StdEncoding.EncodeToString(hash[:])

	fn := &Function{
		FunctionName: req.FunctionName,
		Runtime:      req.Runtime,
		Role:         req.Role,
		Handler:      req.Handler,
		Description:  req.Description,
		Timeout:      timeout,
		MemorySize:   memorySize,
		CodeSha256:   codeSha256,
		CodeSize:     int64(len(codeBytes)),
		Code:         codeBytes,
		Environment:  req.Environment,
	}

	fn, err = store.Create(fn)
	if err != nil {
		return jsonErr(service.NewAWSError("ResourceConflictException",
			err.Error(), http.StatusConflict))
	}

	return jsonCreated(functionConfigResponse(fn))
}

// ---- GetFunction ----

func handleGetFunction(ctx *service.RequestContext, store *FunctionStore, name string) (*service.Response, error) {
	fn, ok := store.Get(name)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Function not found: %s", name), http.StatusNotFound))
	}

	resp := map[string]any{
		"Configuration": functionConfigResponse(fn),
		"Code": map[string]any{
			"Location": fmt.Sprintf("https://awslambda-%s-tasks.s3.%s.amazonaws.com/snapshots/%s",
				ctx.Region, ctx.Region, fn.FunctionName),
		},
	}
	return jsonOK(resp)
}

// ---- ListFunctions ----

func handleListFunctions(ctx *service.RequestContext, store *FunctionStore) (*service.Response, error) {
	functions := store.List()
	configs := make([]map[string]any, 0, len(functions))
	for _, fn := range functions {
		configs = append(configs, functionConfigResponse(fn))
	}

	resp := map[string]any{
		"Functions": configs,
	}
	return jsonOK(resp)
}

// ---- DeleteFunction ----

func handleDeleteFunction(ctx *service.RequestContext, store *FunctionStore, executor *Executor, name string) (*service.Response, error) {
	if !store.Delete(name) {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Function not found: %s", name), http.StatusNotFound))
	}
	executor.InvalidateCache(name)
	return jsonNoContent()
}

// ---- UpdateFunctionCode ----

type updateFunctionCodeRequest struct {
	ZipFile  string `json:"ZipFile"`
	S3Bucket string `json:"S3Bucket"`
	S3Key    string `json:"S3Key"`
}

func handleUpdateFunctionCode(ctx *service.RequestContext, store *FunctionStore, executor *Executor, locator ServiceLocator, name string) (*service.Response, error) {
	var req updateFunctionCodeRequest
	if err := gojson.Unmarshal(ctx.Body, &req); err != nil {
		return jsonErr(service.ErrValidation("invalid request body: " + err.Error()))
	}

	var codeBytes []byte
	var err error

	if req.ZipFile != "" {
		codeBytes, err = base64.StdEncoding.DecodeString(req.ZipFile)
		if err != nil {
			return jsonErr(service.ErrValidation("invalid base64 in ZipFile: " + err.Error()))
		}
	} else if req.S3Bucket != "" && req.S3Key != "" {
		codeBytes, err = getCodeFromS3(locator, req.S3Bucket, req.S3Key)
		if err != nil {
			return jsonErr(service.NewAWSError("InvalidParameterValueException",
				fmt.Sprintf("Error getting object from S3: %v", err), http.StatusBadRequest))
		}
	} else {
		return jsonErr(service.ErrValidation("ZipFile or S3Bucket+S3Key is required"))
	}

	hash := sha256.Sum256(codeBytes)
	codeSha256 := base64.StdEncoding.EncodeToString(hash[:])

	executor.InvalidateCache(name)

	fn, err := store.UpdateCode(name, codeBytes, codeSha256, int64(len(codeBytes)))
	if err != nil {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Function not found: %s", name), http.StatusNotFound))
	}

	return jsonOK(functionConfigResponse(fn))
}

// ---- GetFunctionConfiguration ----

func handleGetFunctionConfiguration(ctx *service.RequestContext, store *FunctionStore, name string) (*service.Response, error) {
	fn, ok := store.Get(name)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Function not found: %s", name), http.StatusNotFound))
	}
	return jsonOK(functionConfigResponse(fn))
}

// ---- UpdateFunctionConfiguration ----

func handleUpdateFunctionConfiguration(ctx *service.RequestContext, store *FunctionStore, name string) (*service.Response, error) {
	var updates map[string]any
	if err := gojson.Unmarshal(ctx.Body, &updates); err != nil {
		return jsonErr(service.ErrValidation("invalid request body: " + err.Error()))
	}

	fn, err := store.UpdateConfiguration(name, updates)
	if err != nil {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Function not found: %s", name), http.StatusNotFound))
	}

	return jsonOK(functionConfigResponse(fn))
}

// ---- Invoke ----

func handleInvoke(ctx *service.RequestContext, store *FunctionStore, executor *Executor, name string) (*service.Response, error) {
	fn, ok := store.Get(name)
	if !ok {
		return jsonErr(service.NewAWSError("ResourceNotFoundException",
			fmt.Sprintf("Function not found: %s", name), http.StatusNotFound))
	}

	invocationType := ctx.RawRequest.Header.Get("X-Amz-Invocation-Type")
	if invocationType == "" {
		invocationType = "RequestResponse"
	}

	// DryRun: validate and return 204.
	if invocationType == "DryRun" {
		return &service.Response{
			StatusCode: http.StatusNoContent,
			Format:     service.FormatJSON,
			Headers: map[string]string{
				"X-Amz-Function-Error": "",
			},
		}, nil
	}

	event := ctx.Body
	if event == nil {
		event = []byte("{}")
	}

	result, err := executor.Invoke(fn, event)
	if err != nil {
		return jsonErr(service.NewAWSError("ServiceException",
			fmt.Sprintf("Function execution error: %v", err), http.StatusInternalServerError))
	}

	headers := map[string]string{
		"X-Amz-Executed-Version": fn.Version,
	}

	// Check if result contains an error.
	var resultMap map[string]any
	if gojson.Unmarshal(result, &resultMap) == nil {
		if _, hasErr := resultMap["errorMessage"]; hasErr {
			headers["X-Amz-Function-Error"] = "Unhandled"
		}
	}

	// For Event (async) invocation, return 202.
	if invocationType == "Event" {
		return &service.Response{
			StatusCode: http.StatusAccepted,
			Format:     service.FormatJSON,
			Headers:    headers,
		}, nil
	}

	// RequestResponse: return the result directly.
	logResult := ctx.RawRequest.Header.Get("X-Amz-Log-Type")
	if logResult == "Tail" {
		// Return empty log in the header (base64-encoded).
		headers["X-Amz-Log-Result"] = base64.StdEncoding.EncodeToString([]byte("cloudmock: logs not captured\n"))
	}

	return &service.Response{
		StatusCode:     http.StatusOK,
		RawBody:        result,
		RawContentType: "application/json",
		Headers:        headers,
		Format:         service.FormatJSON,
	}, nil
}

// ---- helpers ----

func getCodeFromS3(locator ServiceLocator, bucket, key string) ([]byte, error) {
	if locator == nil {
		return nil, fmt.Errorf("S3 code source not supported: no service locator configured")
	}

	svc, err := locator.Lookup("s3")
	if err != nil {
		return nil, fmt.Errorf("S3 service not available: %w", err)
	}

	getter, ok := svc.(S3ObjectGetter)
	if !ok {
		return nil, fmt.Errorf("S3 service does not support GetObjectData")
	}

	return getter.GetObjectData(bucket, key)
}
