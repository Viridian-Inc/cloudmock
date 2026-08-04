package resources

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/azurearm"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func (s *ResourcesService) handleDeployments(ctx *service.RequestContext, subscriptionID string, parts []string, providerIndex int) (*service.Response, error) {
	if len(parts) < 4 || !strings.EqualFold(parts[2], "resourceGroups") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "Deployment routes require a resource group scope.")
	}
	resourceGroup := parts[3]
	if len(parts) == providerIndex+3 && ctx.RawRequest.Method == http.MethodGet {
		return s.listDeployments(subscriptionID, resourceGroup)
	}
	if len(parts) != providerIndex+4 {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The deployment route is not implemented.")
	}

	name := parts[providerIndex+3]
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateDeployment(subscriptionID, resourceGroup, name, ctx.Body)
	case http.MethodGet:
		if ctx.RawRequest.URL.Query().Get("$operationStatus") == "delete" {
			return s.pollDeploymentDelete(ctx)
		}
		return s.getDeployment(subscriptionID, resourceGroup, name)
	case http.MethodDelete:
		return s.deleteDeployment(ctx, subscriptionID, resourceGroup, name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ResourcesService) createOrUpdateDeployment(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var raw map[string]any
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &raw); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	props := asMap(raw["properties"])
	mode := stringValue(props["mode"])
	if mode == "" {
		mode = "Incremental"
	}
	if !strings.EqualFold(mode, "Incremental") {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidDeploymentMode", "Only Incremental deployment mode is supported.")
	}

	template := asMap(props["template"])
	parameters := deploymentParameters(subscriptionID, resourceGroup, asMap(props["parameters"]), asMap(template["parameters"]))
	templateCtx := newTemplateContext(subscriptionID, resourceGroup, parameters, asMap(template["variables"]))
	resources := orderedTemplateResources(asSlice(template["resources"]), templateCtx)

	outputResources := make([]any, 0, len(resources))
	for _, resource := range resources {
		resolved := resolveTemplateValue(resource, templateCtx).(map[string]any)
		provisioner := s.templateProvisionerFor(resolved)
		if provisioner == nil {
			outputResources = append(outputResources, resolved)
			templateCtx.registerReference(resolved, resolved, nil)
			continue
		}
		provisioned, err := provisioner.ProvisionTemplateResource(subscriptionID, resourceGroup, resolved)
		if err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "DeploymentFailed", err.Error())
		}
		outputResources = append(outputResources, provisioned)
		templateCtx.registerReference(resolved, provisioned, provisioner)
	}

	deployment := Deployment{
		ID:   deploymentID(subscriptionID, resourceGroup, name),
		Name: name,
		Type: "Microsoft.Resources/deployments",
		Properties: DeploymentProperties{
			ProvisioningState: "Succeeded",
			Mode:              "Incremental",
			Timestamp:         time.Now().UTC().Format(time.RFC3339Nano),
			Parameters:        parameters,
			Outputs:           deploymentOutputs(asMap(template["outputs"]), templateCtx),
			OutputResources:   outputResources,
		},
	}

	s.mu.Lock()
	if s.deployments[deploymentScopeKey(subscriptionID, resourceGroup)] == nil {
		s.deployments[deploymentScopeKey(subscriptionID, resourceGroup)] = make(map[string]Deployment)
	}
	s.deployments[deploymentScopeKey(subscriptionID, resourceGroup)][strings.ToLower(name)] = deployment
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusCreated, deployment)
}

func (s *ResourcesService) templateProvisionerFor(resource map[string]any) TemplateProvisioner {
	for _, provisioner := range s.provisioners {
		if provisioner.SupportsTemplateResource(resource) {
			return provisioner
		}
	}
	return nil
}

func (s *ResourcesService) getDeployment(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	deployment, ok := s.deployments[deploymentScopeKey(subscriptionID, resourceGroup)][strings.ToLower(name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "DeploymentNotFound", fmt.Sprintf("Deployment %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, deployment)
}

func (s *ResourcesService) listDeployments(subscriptionID, resourceGroup string) (*service.Response, error) {
	s.mu.RLock()
	values := make([]Deployment, 0, len(s.deployments[deploymentScopeKey(subscriptionID, resourceGroup)]))
	for _, deployment := range s.deployments[deploymentScopeKey(subscriptionID, resourceGroup)] {
		values = append(values, deployment)
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ResourcesService) deleteDeployment(ctx *service.RequestContext, subscriptionID, resourceGroup, name string) (*service.Response, error) {
	scopeKey := deploymentScopeKey(subscriptionID, resourceGroup)
	deploymentKey := strings.ToLower(name)

	s.mu.Lock()
	deployments := s.deployments[scopeKey]
	if deployments == nil {
		s.mu.Unlock()
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	if _, ok := deployments[deploymentKey]; !ok {
		s.mu.Unlock()
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	delete(deployments, deploymentKey)
	if len(deployments) == 0 {
		delete(s.deployments, scopeKey)
	}
	s.mu.Unlock()

	location := *ctx.RawRequest.URL
	query := location.Query()
	query.Set("$operationStatus", "delete")
	location.RawQuery = query.Encode()

	s.mu.Lock()
	s.deploymentOps[location.String()] = 1
	s.mu.Unlock()

	return &service.Response{
		StatusCode: http.StatusAccepted,
		Headers: map[string]string{
			"Location":    location.String(),
			"Retry-After": "1",
		},
	}, nil
}

func (s *ResourcesService) pollDeploymentDelete(ctx *service.RequestContext) (*service.Response, error) {
	operationKey := ctx.RawRequest.URL.String()

	s.mu.Lock()
	remaining := s.deploymentOps[operationKey]
	if remaining > 0 {
		s.deploymentOps[operationKey] = remaining - 1
		s.mu.Unlock()
		return &service.Response{
			StatusCode: http.StatusAccepted,
			Headers: map[string]string{
				"Retry-After": "1",
			},
		}, nil
	}
	delete(s.deploymentOps, operationKey)
	s.mu.Unlock()

	return &service.Response{StatusCode: http.StatusNoContent}, nil
}

func deploymentParameters(subscriptionID, resourceGroup string, provided, definitions map[string]any) map[string]any {
	out := make(map[string]any, len(provided)+len(definitions))
	for key, definition := range definitions {
		if definitionMap := asMap(definition); definitionMap != nil {
			if defaultValue, ok := definitionMap["defaultValue"]; ok {
				out[key] = defaultValue
			}
		}
	}
	for key, value := range provided {
		if valueMap := asMap(value); valueMap != nil {
			out[key] = valueMap["value"]
			continue
		}
		out[key] = value
	}
	ctx := newTemplateContext(subscriptionID, resourceGroup, out, nil)
	for key := range out {
		out[key] = ctx.parameter(key)
	}
	return out
}

func deploymentOutputs(raw map[string]any, ctx templateContext) map[string]DeploymentOutput {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]DeploymentOutput, len(raw))
	for key, value := range raw {
		valueMap := asMap(value)
		if copySpec := asMap(valueMap["copy"]); copySpec != nil {
			out[key] = DeploymentOutput{
				Type:  stringValue(valueMap["type"]),
				Value: expandTemplateOutputCopy(copySpec, ctx),
			}
			continue
		}
		out[key] = DeploymentOutput{
			Type:  stringValue(valueMap["type"]),
			Value: resolveTemplateValue(valueMap["value"], ctx),
		}
	}
	return out
}

func expandTemplateOutputCopy(copySpec map[string]any, ctx templateContext) []any {
	count := templateIntValue(resolveTemplateValue(copySpec["count"], ctx))
	if count <= 0 {
		return []any{}
	}

	out := make([]any, 0, count)
	for index := 0; index < count; index++ {
		copyCtx := ctx.withCopyIndex("", index)
		out = append(out, resolveTemplateValue(copySpec["input"], copyCtx))
	}
	return out
}

func expandTemplatePropertyCopies(out map[string]any, copySpecs []any, ctx templateContext) {
	for _, item := range copySpecs {
		copySpec := asMap(item)
		if copySpec == nil {
			continue
		}

		name := stringValue(resolveTemplateValue(copySpec["name"], ctx))
		if name == "" {
			continue
		}

		count := templateIntValue(resolveTemplateValue(copySpec["count"], ctx))
		values := make([]any, 0, positiveCount(count))
		for index := 0; index < count; index++ {
			copyCtx := ctx.withCopyIndex(name, index)
			values = append(values, resolveTemplateValue(copySpec["input"], copyCtx))
		}
		out[name] = values
	}
}

func expandTemplateVariableCopies(variables map[string]any, copySpecs []any, ctx templateContext) {
	for _, item := range copySpecs {
		copySpec := asMap(item)
		if copySpec == nil {
			continue
		}

		name := stringValue(resolveTemplateValue(copySpec["name"], ctx))
		if name == "" {
			continue
		}

		count := templateIntValue(resolveTemplateValue(copySpec["count"], ctx))
		values := make([]any, 0, positiveCount(count))
		for index := 0; index < count; index++ {
			copyCtx := ctx.withCopyIndex(name, index)
			values = append(values, resolveTemplateValue(copySpec["input"], copyCtx))
		}
		variables[name] = values
	}
}

func positiveCount(count int) int {
	if count > 0 {
		return count
	}
	return 0
}

func orderedTemplateResources(raw []any, ctx templateContext) []map[string]any {
	remaining := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if resource := asMap(item); resource != nil {
			remaining = append(remaining, expandTemplateResource(resource, ctx)...)
		}
	}

	ordered := make([]map[string]any, 0, len(remaining))
	done := make(map[string]bool)
	for len(remaining) > 0 {
		progress := false
		nextRemaining := remaining[:0]
		for _, resource := range remaining {
			resolvedType := fmt.Sprint(resolveTemplateValue(resource["type"], ctx))
			resolvedName := fmt.Sprint(resolveTemplateValue(resource["name"], ctx))
			resourceID := resolvedType + "/" + resolvedName
			fullResourceID := templateResourceID(ctx, resolvedType, resolvedName)
			if dependenciesMet(asSlice(resource["dependsOn"]), done, ctx) {
				ordered = append(ordered, resource)
				done[resourceID] = true
				done[fullResourceID] = true
				progress = true
				continue
			}
			nextRemaining = append(nextRemaining, resource)
		}
		if !progress {
			ordered = append(ordered, nextRemaining...)
			break
		}
		remaining = nextRemaining
	}
	return ordered
}

func expandTemplateResource(resource map[string]any, ctx templateContext) []map[string]any {
	if condition, hasCondition := resource["condition"]; hasCondition && !templateBoolValue(resolveTemplateValue(condition, ctx)) {
		return nil
	}

	copySpec := asMap(resource["copy"])
	if copySpec == nil {
		resolved, ok := resolveTemplateValue(resource, ctx).(map[string]any)
		if !ok {
			return nil
		}
		return []map[string]any{resolved}
	}

	count := templateIntValue(resolveTemplateValue(copySpec["count"], ctx))
	if count <= 0 {
		return nil
	}

	loopName := stringValue(resolveTemplateValue(copySpec["name"], ctx))
	out := make([]map[string]any, 0, count)
	for index := 0; index < count; index++ {
		copyCtx := ctx.withCopyIndex(loopName, index)
		resolved, ok := resolveTemplateValue(resource, copyCtx).(map[string]any)
		if !ok {
			continue
		}
		delete(resolved, "copy")
		out = append(out, resolved)
	}
	return out
}

func dependenciesMet(raw []any, done map[string]bool, ctx templateContext) bool {
	for _, item := range raw {
		dep := fmt.Sprint(resolveTemplateValue(item, ctx))
		if dep == "" {
			continue
		}
		if !done[dep] {
			return false
		}
	}
	return true
}

type templateContext struct {
	subscriptionID string
	resourceGroup  string
	parameters     map[string]any
	variables      map[string]any
	resolving      map[string]bool
	copyIndexes    map[string]int
	references     map[string]templateReference
	listResults    map[string]any
}

type templateReference struct {
	properties any
	full       any
}

func newTemplateContext(subscriptionID, resourceGroup string, parameters, variables map[string]any) templateContext {
	if parameters == nil {
		parameters = map[string]any{}
	}
	if variables == nil {
		variables = map[string]any{}
	}
	ctx := templateContext{
		subscriptionID: subscriptionID,
		resourceGroup:  resourceGroup,
		parameters:     parameters,
		variables:      variables,
		resolving:      make(map[string]bool),
		copyIndexes:    make(map[string]int),
		references:     make(map[string]templateReference),
		listResults:    make(map[string]any),
	}
	expandTemplateVariableCopies(ctx.variables, asSlice(ctx.variables["copy"]), ctx)
	delete(ctx.variables, "copy")
	return ctx
}

func (ctx templateContext) registerReference(resource map[string]any, runtime any, provisioner TemplateProvisioner) {
	resourceType := fmt.Sprint(resource["type"])
	resourceName := fmt.Sprint(resource["name"])
	ref := templateReference{
		properties: selectTemplateMapValue(runtime, "properties"),
		full:       runtime,
	}
	if ref.properties == nil {
		ref.properties = runtime
	}

	ctx.references[strings.ToLower(resourceName)] = ref
	ctx.references[strings.ToLower(templateResourceID(ctx, resourceType, resourceName))] = ref
	if resourceType != "" && resourceName != "" {
		ctx.references[strings.ToLower(resourceType+"/"+resourceName)] = ref
	}
	if runtimeMap := asMap(runtime); runtimeMap != nil {
		if id := stringValue(runtimeMap["id"]); id != "" {
			ctx.references[strings.ToLower(id)] = ref
		}
		if name := stringValue(runtimeMap["name"]); name != "" {
			ctx.references[strings.ToLower(name)] = ref
		}
	}
	ctx.registerStorageListKeys(resource, runtime)
	ctx.registerRedisListKeys(resource, runtime)
	ctx.registerProviderListOperations(resource, runtime, provisioner)
}

func (ctx templateContext) registerStorageListKeys(resource map[string]any, runtime any) {
	resourceType := fmt.Sprint(resource["type"])
	if !strings.EqualFold(resourceType, "Microsoft.Storage/storageAccounts") {
		return
	}

	resourceName := fmt.Sprint(resource["name"])
	if runtimeMap := asMap(runtime); runtimeMap != nil {
		if name := stringValue(runtimeMap["name"]); name != "" {
			resourceName = name
		}
	}
	if resourceName == "" {
		return
	}

	accountKey := strings.ToLower(ctx.subscriptionID) + "/" + strings.ToLower(ctx.resourceGroup) + "/" + strings.ToLower(resourceName)
	result := map[string]any{
		"keys": []any{
			map[string]any{"keyName": "key1", "permissions": "Full", "value": templateStorageAccountKeyValue(accountKey, "key1", "0")},
			map[string]any{"keyName": "key2", "permissions": "Full", "value": templateStorageAccountKeyValue(accountKey, "key2", "0")},
		},
	}

	ctx.registerListResultForResource("listKeys", resourceType, resourceName, resource, runtime, result)
}

func (ctx templateContext) registerProviderListOperations(resource map[string]any, runtime any, provisioner TemplateProvisioner) {
	provider, ok := provisioner.(TemplateListOperationProvider)
	if !ok {
		return
	}

	resourceType := fmt.Sprint(resource["type"])
	resourceName := resolvedTemplateResourceName(resource, runtime)
	for _, operation := range []string{"listKeys", "listCredentials"} {
		result, ok := provider.TemplateListOperationResult(ctx.subscriptionID, ctx.resourceGroup, resource, operation)
		if ok {
			ctx.registerListResultForResource(operation, resourceType, resourceName, resource, runtime, result)
		}
	}
}

func templateStorageAccountKeyValue(accountKey, keyName, generation string) string {
	return base64.StdEncoding.EncodeToString([]byte("cloudmock:" + accountKey + ":" + keyName + ":" + generation))
}

func (ctx templateContext) registerRedisListKeys(resource map[string]any, runtime any) {
	resourceType := fmt.Sprint(resource["type"])
	if !strings.EqualFold(resourceType, "Microsoft.Cache/Redis") {
		return
	}

	resourceName := fmt.Sprint(resource["name"])
	if runtimeMap := asMap(runtime); runtimeMap != nil {
		if name := stringValue(runtimeMap["name"]); name != "" {
			resourceName = name
		}
	}
	if resourceName == "" {
		return
	}

	result := map[string]any{
		"primaryKey":   "cloudmock-" + resourceName + "-primary",
		"secondaryKey": "cloudmock-" + resourceName + "-secondary",
	}
	ctx.registerListResultForResource("listKeys", resourceType, resourceName, resource, runtime, result)
}

func resolvedTemplateResourceName(resource map[string]any, runtime any) string {
	resourceName := fmt.Sprint(resource["name"])
	if runtimeMap := asMap(runtime); runtimeMap != nil {
		if name := stringValue(runtimeMap["name"]); name != "" {
			resourceName = name
		}
	}
	return resourceName
}

func (ctx templateContext) registerListResultForResource(function, resourceType, resourceName string, resource map[string]any, runtime any, result any) {
	ctx.registerListResult(function, resourceName, result)
	if resourceType != "" && resourceName != "" {
		ctx.registerListResult(function, templateResourceID(ctx, resourceType, resourceName), result)
	}
	if originalName := fmt.Sprint(resource["name"]); originalName != "" && originalName != resourceName {
		ctx.registerListResult(function, originalName, result)
	}
	if runtimeMap := asMap(runtime); runtimeMap != nil {
		if id := stringValue(runtimeMap["id"]); id != "" {
			ctx.registerListResult(function, id, result)
		}
	}
}

func (ctx templateContext) registerListResult(function, target string, result any) {
	target = strings.TrimSpace(target)
	if target == "" {
		return
	}
	ctx.listResults[strings.ToLower(target)+"/"+strings.ToLower(function)] = result
}

func (ctx templateContext) withCopyIndex(loopName string, index int) templateContext {
	next := ctx
	next.copyIndexes = make(map[string]int, len(ctx.copyIndexes)+2)
	for key, value := range ctx.copyIndexes {
		next.copyIndexes[key] = value
	}
	next.copyIndexes[""] = index
	if loopName != "" {
		next.copyIndexes[strings.ToLower(loopName)] = index
	}
	return next
}

func (ctx templateContext) variable(name string) any {
	resolvingKey := "variable:" + name
	if ctx.resolving[resolvingKey] {
		return nil
	}
	value, ok := ctx.variables[name]
	if !ok {
		return nil
	}
	ctx.resolving[resolvingKey] = true
	resolved := resolveTemplateValue(value, ctx)
	delete(ctx.resolving, resolvingKey)
	ctx.variables[name] = resolved
	return resolved
}

func (ctx templateContext) parameter(name string) any {
	resolvingKey := "parameter:" + name
	if ctx.resolving[resolvingKey] {
		return nil
	}
	value, ok := ctx.parameters[name]
	if !ok {
		return nil
	}
	ctx.resolving[resolvingKey] = true
	resolved := resolveTemplateValue(value, ctx)
	delete(ctx.resolving, resolvingKey)
	ctx.parameters[name] = resolved
	return resolved
}

func resolveTemplateValue(value any, ctx templateContext) any {
	switch typed := value.(type) {
	case string:
		return resolveTemplateString(typed, ctx)
	case map[string]any:
		out := make(map[string]any, len(typed))
		copySpecs := asSlice(typed["copy"])
		for key, child := range typed {
			if key == "copy" && copySpecs != nil {
				continue
			}
			out[key] = resolveTemplateValue(child, ctx)
		}
		expandTemplatePropertyCopies(out, copySpecs, ctx)
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = resolveTemplateValue(child, ctx)
		}
		return out
	default:
		return value
	}
}

func resolveTemplateString(value string, ctx templateContext) any {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		return resolveTemplateExpression(strings.TrimSpace(trimmed[1:len(trimmed)-1]), ctx, value)
	}
	return value
}

func resolveTemplateExpression(expr string, ctx templateContext, original string) any {
	if name, selector, ok := templateFunctionStringArgAndSelector(expr, "parameters"); ok {
		return selectTemplateValue(ctx.parameter(name), selector, ctx)
	}
	if name, selector, ok := templateFunctionStringArgAndSelector(expr, "variables"); ok {
		return selectTemplateValue(ctx.variable(name), selector, ctx)
	}
	lower := strings.ToLower(expr)
	if strings.HasPrefix(lower, "concat(") && strings.HasSuffix(expr, ")") {
		return resolveTemplateConcat(expr[len("concat("):len(expr)-1], ctx)
	}
	if strings.HasPrefix(lower, "format(") && strings.HasSuffix(expr, ")") {
		return resolveTemplateFormat(expr[len("format("):len(expr)-1], ctx)
	}
	if strings.HasPrefix(lower, "length(") && strings.HasSuffix(expr, ")") {
		return resolveTemplateLength(expr[len("length("):len(expr)-1], ctx)
	}
	if args, selector, ok := templateFunctionArgsAndSelector(expr, "range"); ok {
		return selectTemplateValue(resolveTemplateRange(args, ctx), selector, ctx)
	}
	if args, selector, ok := templateFunctionArgsAndSelector(expr, "uniqueString"); ok {
		return selectTemplateValue(resolveTemplateUniqueString(args, ctx), selector, ctx)
	}
	if args, selector, ok := templateFunctionArgsAndSelector(expr, "reference"); ok {
		return selectTemplateValue(resolveTemplateReference(args, ctx), selector, ctx)
	}
	if args, selector, ok := templateFunctionArgsAndSelector(expr, "listKeys"); ok {
		return selectTemplateValue(resolveTemplateListOperation(args, ctx, "listKeys"), selector, ctx)
	}
	if args, selector, ok := templateFunctionArgsAndSelector(expr, "listCredentials"); ok {
		return selectTemplateValue(resolveTemplateListOperation(args, ctx, "listCredentials"), selector, ctx)
	}
	if strings.HasPrefix(lower, "resourceid(") && strings.HasSuffix(expr, ")") {
		return resolveTemplateResourceID(expr[len("resourceId("):len(expr)-1], ctx)
	}
	if strings.HasPrefix(lower, "copyindex(") && strings.HasSuffix(expr, ")") {
		return resolveTemplateCopyIndex(expr[len("copyIndex("):len(expr)-1], ctx)
	}
	if strings.HasPrefix(lower, "equals(") && strings.HasSuffix(expr, ")") {
		return resolveTemplateEquals(expr[len("equals("):len(expr)-1], ctx)
	}
	if strings.EqualFold(expr, "resourceGroup().id") {
		return templateResourceGroupID(ctx)
	}
	if strings.EqualFold(expr, "resourceGroup().name") {
		return ctx.resourceGroup
	}
	if strings.EqualFold(expr, "resourceGroup().location") {
		return "eastus"
	}
	return original
}

func resolveTemplateConcat(args string, ctx templateContext) string {
	var out strings.Builder
	for _, arg := range splitTemplateArgs(args) {
		value := resolveTemplateArg(arg, ctx)
		if value == nil {
			continue
		}
		out.WriteString(fmt.Sprint(value))
	}
	return out.String()
}

func resolveTemplateFormat(args string, ctx templateContext) string {
	parts := splitTemplateArgs(args)
	if len(parts) == 0 {
		return ""
	}

	out := fmt.Sprint(resolveTemplateArg(parts[0], ctx))
	for index, arg := range parts[1:] {
		placeholder := "{" + strconv.Itoa(index) + "}"
		out = strings.ReplaceAll(out, placeholder, fmt.Sprint(resolveTemplateArg(arg, ctx)))
	}
	return out
}

func resolveTemplateLength(args string, ctx templateContext) int {
	parts := splitTemplateArgs(args)
	if len(parts) != 1 {
		return 0
	}

	switch typed := resolveTemplateArg(parts[0], ctx).(type) {
	case []any:
		return len(typed)
	case map[string]any:
		return len(typed)
	case string:
		return len(typed)
	default:
		return 0
	}
}

func resolveTemplateRange(args string, ctx templateContext) []any {
	parts := splitTemplateArgs(args)
	if len(parts) != 2 {
		return []any{}
	}

	start := templateIntValue(resolveTemplateArg(parts[0], ctx))
	count := templateIntValue(resolveTemplateArg(parts[1], ctx))
	if count <= 0 {
		return []any{}
	}

	out := make([]any, count)
	for i := 0; i < count; i++ {
		out[i] = start + i
	}
	return out
}

func resolveTemplateUniqueString(args string, ctx templateContext) string {
	parts := splitTemplateArgs(args)
	if len(parts) == 0 {
		return ""
	}

	values := make([]string, 0, len(parts))
	for _, part := range parts {
		values = append(values, fmt.Sprint(resolveTemplateArg(part, ctx)))
	}
	sum := sha256.Sum256([]byte(strings.Join(values, "\x00")))
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:])
	return strings.ToLower(encoded[:13])
}

func resolveTemplateReference(args string, ctx templateContext) any {
	parts := splitTemplateArgs(args)
	if len(parts) == 0 {
		return nil
	}

	target := fmt.Sprint(resolveTemplateArg(parts[0], ctx))
	ref, ok := ctx.references[strings.ToLower(target)]
	if !ok {
		return nil
	}

	for _, part := range parts[1:] {
		if value, ok := trimTemplateQuotedString(part); ok && strings.EqualFold(value, "Full") {
			return ref.full
		}
	}
	return ref.properties
}

func resolveTemplateListOperation(args string, ctx templateContext, operation string) any {
	parts := splitTemplateArgs(args)
	if len(parts) == 0 {
		return nil
	}

	target := fmt.Sprint(resolveTemplateArg(parts[0], ctx))
	return ctx.listResults[strings.ToLower(target)+"/"+strings.ToLower(operation)]
}

func resolveTemplateEquals(args string, ctx templateContext) bool {
	parts := splitTemplateArgs(args)
	if len(parts) != 2 {
		return false
	}
	left := resolveTemplateArg(parts[0], ctx)
	right := resolveTemplateArg(parts[1], ctx)
	return fmt.Sprint(left) == fmt.Sprint(right)
}

func resolveTemplateCopyIndex(args string, ctx templateContext) int {
	parts := splitTemplateArgs(args)
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return ctx.copyIndexes[""]
	}

	if loopName, ok := trimTemplateQuotedString(parts[0]); ok {
		index := ctx.copyIndexes[strings.ToLower(loopName)]
		if len(parts) > 1 {
			index += templateIntValue(resolveTemplateArg(parts[1], ctx))
		}
		return index
	}

	return ctx.copyIndexes[""] + templateIntValue(resolveTemplateArg(parts[0], ctx))
}

func resolveTemplateResourceID(args string, ctx templateContext) string {
	parts := splitTemplateArgs(args)
	if len(parts) < 2 {
		return ""
	}
	resourceType := fmt.Sprint(resolveTemplateArg(parts[0], ctx))
	names := make([]string, 0, len(parts)-1)
	for _, part := range parts[1:] {
		names = append(names, fmt.Sprint(resolveTemplateArg(part, ctx)))
	}
	return templateResourceID(ctx, resourceType, strings.Join(names, "/"))
}

func templateResourceID(ctx templateContext, resourceType, resourceName string) string {
	typeParts := strings.Split(resourceType, "/")
	nameParts := strings.Split(resourceName, "/")
	var id strings.Builder
	id.WriteString("/subscriptions/")
	id.WriteString(ctx.subscriptionID)
	id.WriteString("/resourceGroups/")
	id.WriteString(ctx.resourceGroup)
	id.WriteString("/providers/")
	if len(typeParts) == 0 {
		return id.String()
	}
	id.WriteString(typeParts[0])
	for i := 1; i < len(typeParts); i++ {
		id.WriteString("/")
		id.WriteString(typeParts[i])
		if i-1 < len(nameParts) && nameParts[i-1] != "" {
			id.WriteString("/")
			id.WriteString(nameParts[i-1])
		}
	}
	for i := len(typeParts) - 1; i < len(nameParts); i++ {
		if i >= 0 && nameParts[i] != "" {
			id.WriteString("/")
			id.WriteString(nameParts[i])
		}
	}
	return id.String()
}

func templateResourceGroupID(ctx templateContext) string {
	return "/subscriptions/" + ctx.subscriptionID + "/resourceGroups/" + ctx.resourceGroup
}

func resolveTemplateArg(arg string, ctx templateContext) any {
	arg = strings.TrimSpace(arg)
	if unquoted, ok := trimTemplateQuotedString(arg); ok {
		return unquoted
	}
	return resolveTemplateExpression(arg, ctx, arg)
}

func splitTemplateArgs(args string) []string {
	parts := []string{}
	start := 0
	depth := 0
	var quote rune
	for index, char := range args {
		switch {
		case quote != 0:
			if char == quote {
				quote = 0
			}
		case char == '\'' || char == '"':
			quote = char
		case char == '(':
			depth++
		case char == ')' && depth > 0:
			depth--
		case char == ',' && depth == 0:
			parts = append(parts, strings.TrimSpace(args[start:index]))
			start = index + len(string(char))
		}
	}
	if start <= len(args) {
		parts = append(parts, strings.TrimSpace(args[start:]))
	}
	return parts
}

func templateFunctionStringArg(expr, function string) (string, bool) {
	name, selector, ok := templateFunctionStringArgAndSelector(expr, function)
	return name, ok && selector == ""
}

func templateFunctionStringArgAndSelector(expr, function string) (string, string, bool) {
	expr = strings.TrimSpace(expr)
	prefix := function + "("
	if !strings.HasPrefix(strings.ToLower(expr), strings.ToLower(prefix)) {
		return "", "", false
	}
	closeIndex := findTemplateCallEnd(expr, len(function))
	if closeIndex < 0 {
		return "", "", false
	}
	name, ok := trimTemplateQuotedString(expr[len(prefix):closeIndex])
	if !ok {
		return "", "", false
	}
	return name, strings.TrimSpace(expr[closeIndex+1:]), true
}

func templateFunctionArgsAndSelector(expr, function string) (string, string, bool) {
	expr = strings.TrimSpace(expr)
	prefix := function + "("
	if !strings.HasPrefix(strings.ToLower(expr), strings.ToLower(prefix)) {
		return "", "", false
	}
	closeIndex := findTemplateCallEnd(expr, len(function))
	if closeIndex < 0 {
		return "", "", false
	}
	return expr[len(prefix):closeIndex], strings.TrimSpace(expr[closeIndex+1:]), true
}

func findTemplateCallEnd(expr string, openIndex int) int {
	depth := 0
	var quote rune
	for index, char := range expr[openIndex:] {
		absolute := openIndex + index
		switch {
		case quote != 0:
			if char == quote {
				quote = 0
			}
		case char == '\'' || char == '"':
			quote = char
		case char == '(':
			depth++
		case char == ')':
			depth--
			if depth == 0 {
				return absolute
			}
		}
	}
	return -1
}

func selectTemplateValue(value any, selector string, ctx templateContext) any {
	for selector != "" {
		selector = strings.TrimSpace(selector)
		switch {
		case strings.HasPrefix(selector, "."):
			end := 1
			for end < len(selector) && selector[end] != '.' && selector[end] != '[' {
				end++
			}
			value = selectTemplateMapValue(value, selector[1:end])
			selector = selector[end:]
		case strings.HasPrefix(selector, "["):
			end := strings.Index(selector, "]")
			if end < 0 {
				return nil
			}
			token := strings.TrimSpace(selector[1:end])
			if key, ok := trimTemplateQuotedString(token); ok {
				value = selectTemplateMapValue(value, key)
			} else {
				index, err := strconv.Atoi(token)
				if err != nil {
					index = templateIntValue(resolveTemplateArg(token, ctx))
				}
				value = selectTemplateSliceValue(value, index)
			}
			selector = selector[end+1:]
		default:
			return nil
		}
	}
	return value
}

func selectTemplateMapValue(value any, key string) any {
	if typed, ok := value.(map[string]any); ok {
		return typed[key]
	}
	return nil
}

func selectTemplateSliceValue(value any, index int) any {
	if typed, ok := value.([]any); ok && index >= 0 && index < len(typed) {
		return typed[index]
	}
	return nil
}

func trimTemplateQuotedString(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if len(value) < 2 {
		return "", false
	}
	if (value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"') {
		return value[1 : len(value)-1], true
	}
	return "", false
}

func asMap(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func asSlice(value any) []any {
	if typed, ok := value.([]any); ok {
		return typed
	}
	return nil
}

func stringValue(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func templateIntValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(typed))
		return parsed
	default:
		return 0
	}
}

func templateBoolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		parsed, _ := strconv.ParseBool(strings.TrimSpace(typed))
		return parsed
	default:
		return false
	}
}

func deploymentScopeKey(subscriptionID, resourceGroup string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup)
}

func deploymentID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Resources/deployments/" + name
}
