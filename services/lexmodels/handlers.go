package lexmodels

import (
	"net/http"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func parseJSON(body []byte, v any) *service.AWSError {
	if len(body) == 0 {
		return nil
	}
	if err := gojson.Unmarshal(body, v); err != nil {
		return service.NewAWSError("BadRequestException",
			"Request body is not valid JSON.", http.StatusBadRequest)
	}
	return nil
}

func getStr(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

func getFloat(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		}
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if mm, ok := v.(map[string]any); ok {
			return mm
		}
	}
	return nil
}

func getMapList(m map[string]any, key string) []map[string]any {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, x := range arr {
		if xm, ok := x.(map[string]any); ok {
			out = append(out, xm)
		}
	}
	return out
}

func getStrList(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, x := range arr {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// parseTagList accepts either Lex's flat {tagKey,tagValue} or the more common
// {key,value} schema, returning a flat map.
func parseTagList(m map[string]any, key string) map[string]string {
	out := make(map[string]string)
	for _, t := range getMapList(m, key) {
		k := getStr(t, "key")
		if k == "" {
			k = getStr(t, "Key")
		}
		v := getStr(t, "value")
		if v == "" {
			v = getStr(t, "Value")
		}
		if k != "" {
			out[k] = v
		}
	}
	return out
}

func tagListToMaps(tags map[string]string) []map[string]any {
	out := make([]map[string]any, 0, len(tags))
	for k, v := range tags {
		out = append(out, map[string]any{"key": k, "value": v})
	}
	return out
}

// rfc3339 formats a time as Lex's documented createdDate / lastUpdatedDate.
func rfc3339(t time.Time) string { return t.Format(time.RFC3339) }

// pathParam pulls /v1/.../{name} segments from the request URL when present.
// Lex Models V1 uses REST bindings, but real SDK clients usually send JSON.
// We support both shapes by falling back to the body.
func pathParam(ctx *service.RequestContext, key string) string {
	if ctx == nil || ctx.Params == nil {
		return ""
	}
	return ctx.Params[key]
}

// ── Bot response shaping ─────────────────────────────────────────────────────

func botToMap(b *StoredBot) map[string]any {
	out := map[string]any{
		"name":                         b.Name,
		"version":                      b.Version,
		"description":                  b.Description,
		"locale":                       b.Locale,
		"status":                       b.Status,
		"checksum":                     b.Checksum,
		"createdDate":                  rfc3339(b.CreatedAt),
		"lastUpdatedDate":              rfc3339(b.LastUpdatedAt),
		"idleSessionTTLInSeconds":      b.IdleSessionTTLInSeconds,
		"childDirected":                b.ChildDirected,
		"detectSentiment":              b.DetectSentiment,
		"enableModelImprovements":      b.EnableModelImprovements,
		"nluIntentConfidenceThreshold": b.NluIntentConfidenceThreshold,
		"intents":                      b.Intents,
	}
	if b.VoiceID != "" {
		out["voiceId"] = b.VoiceID
	}
	if b.FailureReason != "" {
		out["failureReason"] = b.FailureReason
	}
	if b.ClarificationPrompt != nil {
		out["clarificationPrompt"] = b.ClarificationPrompt
	}
	if b.AbortStatement != nil {
		out["abortStatement"] = b.AbortStatement
	}
	return out
}

func botMetadata(b *StoredBot) map[string]any {
	return map[string]any{
		"name":            b.Name,
		"version":         b.Version,
		"description":     b.Description,
		"status":          b.Status,
		"createdDate":     rfc3339(b.CreatedAt),
		"lastUpdatedDate": rfc3339(b.LastUpdatedAt),
	}
}

func intentToMap(in *StoredIntent) map[string]any {
	out := map[string]any{
		"name":             in.Name,
		"version":          in.Version,
		"description":      in.Description,
		"checksum":         in.Checksum,
		"createdDate":      rfc3339(in.CreatedAt),
		"lastUpdatedDate":  rfc3339(in.LastUpdatedAt),
		"slots":            in.Slots,
		"sampleUtterances": in.SampleUtterances,
	}
	if in.ParentIntentSignature != "" {
		out["parentIntentSignature"] = in.ParentIntentSignature
	}
	if in.ConfirmationPrompt != nil {
		out["confirmationPrompt"] = in.ConfirmationPrompt
	}
	if in.RejectionStatement != nil {
		out["rejectionStatement"] = in.RejectionStatement
	}
	if in.FollowUpPrompt != nil {
		out["followUpPrompt"] = in.FollowUpPrompt
	}
	if in.ConclusionStatement != nil {
		out["conclusionStatement"] = in.ConclusionStatement
	}
	if in.DialogCodeHook != nil {
		out["dialogCodeHook"] = in.DialogCodeHook
	}
	if in.FulfillmentActivity != nil {
		out["fulfillmentActivity"] = in.FulfillmentActivity
	}
	if in.InputContexts != nil {
		out["inputContexts"] = in.InputContexts
	}
	if in.OutputContexts != nil {
		out["outputContexts"] = in.OutputContexts
	}
	if in.KendraConfiguration != nil {
		out["kendraConfiguration"] = in.KendraConfiguration
	}
	return out
}

func intentMetadata(in *StoredIntent) map[string]any {
	return map[string]any{
		"name":            in.Name,
		"version":         in.Version,
		"description":     in.Description,
		"createdDate":     rfc3339(in.CreatedAt),
		"lastUpdatedDate": rfc3339(in.LastUpdatedAt),
	}
}

func slotTypeToMap(st *StoredSlotType) map[string]any {
	out := map[string]any{
		"name":              st.Name,
		"version":           st.Version,
		"description":       st.Description,
		"checksum":          st.Checksum,
		"createdDate":       rfc3339(st.CreatedAt),
		"lastUpdatedDate":   rfc3339(st.LastUpdatedAt),
		"enumerationValues": st.EnumerationValues,
	}
	if st.ValueSelectionStrategy != "" {
		out["valueSelectionStrategy"] = st.ValueSelectionStrategy
	}
	if st.ParentSlotTypeSignature != "" {
		out["parentSlotTypeSignature"] = st.ParentSlotTypeSignature
	}
	if st.SlotTypeConfigurations != nil {
		out["slotTypeConfigurations"] = st.SlotTypeConfigurations
	}
	return out
}

func slotTypeMetadata(st *StoredSlotType) map[string]any {
	return map[string]any{
		"name":            st.Name,
		"version":         st.Version,
		"description":     st.Description,
		"createdDate":     rfc3339(st.CreatedAt),
		"lastUpdatedDate": rfc3339(st.LastUpdatedAt),
	}
}

func botAliasToMap(a *StoredBotAlias) map[string]any {
	out := map[string]any{
		"name":            a.Name,
		"botName":         a.BotName,
		"botVersion":      a.BotVersion,
		"description":     a.Description,
		"checksum":        a.Checksum,
		"createdDate":     rfc3339(a.CreatedAt),
		"lastUpdatedDate": rfc3339(a.LastUpdatedAt),
	}
	if a.ConversationLogs != nil {
		out["conversationLogs"] = a.ConversationLogs
	}
	return out
}

func channelAssocToMap(c *StoredChannelAssoc) map[string]any {
	out := map[string]any{
		"name":             c.Name,
		"botName":          c.BotName,
		"botAlias":         c.BotAlias,
		"description":      c.Description,
		"type":             c.Type,
		"status":           c.Status,
		"botConfiguration": c.BotConfiguration,
		"createdDate":      rfc3339(c.CreatedAt),
	}
	if c.FailureReason != "" {
		out["failureReason"] = c.FailureReason
	}
	return out
}

func importToMap(i *StoredImport) map[string]any {
	return map[string]any{
		"importId":      i.ImportID,
		"name":          i.Name,
		"resourceType":  i.ResourceType,
		"mergeStrategy": i.MergeStrategy,
		"importStatus":  i.Status,
		"failureReason": i.FailureReason,
		"createdDate":   rfc3339(i.CreatedAt),
	}
}

func migrationToMap(m *StoredMigration) map[string]any {
	return map[string]any{
		"migrationId":        m.MigrationID,
		"migrationStatus":    m.Status,
		"migrationStrategy":  m.MigrationStrategy,
		"migrationTimestamp": rfc3339(m.StartedAt),
		"v1BotName":          m.V1BotName,
		"v1BotVersion":       m.V1BotVersion,
		"v1BotLocale":        m.V1BotLocale,
		"v2BotId":            m.V2BotID,
		"v2BotRole":          m.V2BotRole,
		"alerts":             m.Alerts,
	}
}

// ── Builtin catalogue (returned by GetBuiltin*) ─────────────────────────────

var builtinIntents = []map[string]any{
	{"signature": "AMAZON.HelpIntent", "supportedLocales": []string{"en-US", "en-GB", "en-AU"}},
	{"signature": "AMAZON.CancelIntent", "supportedLocales": []string{"en-US", "en-GB", "en-AU"}},
	{"signature": "AMAZON.StopIntent", "supportedLocales": []string{"en-US", "en-GB", "en-AU"}},
	{"signature": "AMAZON.YesIntent", "supportedLocales": []string{"en-US"}},
	{"signature": "AMAZON.NoIntent", "supportedLocales": []string{"en-US"}},
	{"signature": "AMAZON.RepeatIntent", "supportedLocales": []string{"en-US"}},
	{"signature": "AMAZON.StartOverIntent", "supportedLocales": []string{"en-US"}},
	{"signature": "AMAZON.PauseIntent", "supportedLocales": []string{"en-US"}},
	{"signature": "AMAZON.ResumeIntent", "supportedLocales": []string{"en-US"}},
	{"signature": "AMAZON.FallbackIntent", "supportedLocales": []string{"en-US"}},
}

var builtinSlotTypes = []map[string]any{
	{"signature": "AMAZON.DATE", "supportedLocales": []string{"en-US", "en-GB", "en-AU"}},
	{"signature": "AMAZON.TIME", "supportedLocales": []string{"en-US", "en-GB", "en-AU"}},
	{"signature": "AMAZON.NUMBER", "supportedLocales": []string{"en-US", "en-GB", "en-AU"}},
	{"signature": "AMAZON.US_FIRST_NAME", "supportedLocales": []string{"en-US"}},
	{"signature": "AMAZON.US_CITY", "supportedLocales": []string{"en-US"}},
	{"signature": "AMAZON.US_STATE", "supportedLocales": []string{"en-US"}},
	{"signature": "AMAZON.PhoneNumber", "supportedLocales": []string{"en-US"}},
	{"signature": "AMAZON.Country", "supportedLocales": []string{"en-US", "en-GB"}},
	{"signature": "AMAZON.AlphaNumeric", "supportedLocales": []string{"en-US"}},
}

func filterBuiltins(list []map[string]any, locale, contains string) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, item := range list {
		if contains != "" {
			sig := item["signature"].(string)
			if !containsCaseInsensitive(sig, contains) {
				continue
			}
		}
		if locale != "" {
			locales, _ := item["supportedLocales"].([]string)
			if !containsStr(locales, locale) {
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

func containsStr(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func containsCaseInsensitive(haystack, needle string) bool {
	return contains(toLower(haystack), toLower(needle))
}

func toLower(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}

// ── Bot handlers ─────────────────────────────────────────────────────────────

func handlePutBot(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	if name == "" {
		name = pathParam(ctx, "name")
	}
	if name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	bot := &StoredBot{
		Name:                         name,
		Description:                  getStr(req, "description"),
		Locale:                       getStr(req, "locale"),
		VoiceID:                      getStr(req, "voiceId"),
		ProcessBehavior:              getStr(req, "processBehavior"),
		IdleSessionTTLInSeconds:      getInt(req, "idleSessionTTLInSeconds"),
		NluIntentConfidenceThreshold: getFloat(req, "nluIntentConfidenceThreshold"),
		ChildDirected:                getBool(req, "childDirected"),
		DetectSentiment:              getBool(req, "detectSentiment"),
		EnableModelImprovements:      getBool(req, "enableModelImprovements"),
		CreateVersion:                getBool(req, "createVersion"),
		Intents:                      getMapList(req, "intents"),
		ClarificationPrompt:          getMap(req, "clarificationPrompt"),
		AbortStatement:               getMap(req, "abortStatement"),
	}
	saved, err := store.PutBot(bot)
	if err != nil {
		return jsonErr(err)
	}
	if tags := parseTagList(req, "tags"); len(tags) > 0 {
		store.TagResource(store.botArn(saved.Name), tags)
	}
	if saved.CreateVersion {
		if _, vErr := store.CreateBotVersion(saved.Name); vErr != nil {
			return jsonErr(vErr)
		}
	}
	resp := botToMap(saved)
	resp["createVersion"] = saved.CreateVersion
	resp["tags"] = tagListToMaps(store.ListTags(store.botArn(saved.Name)))
	return jsonOK(resp)
}

func handleCreateBotVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	if name == "" {
		name = pathParam(ctx, "name")
	}
	if name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	bot, err := store.CreateBotVersion(name)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(botToMap(bot))
}

func handleGetBot(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	if name == "" {
		name = pathParam(ctx, "name")
	}
	version := getStr(req, "versionoralias")
	if version == "" {
		version = getStr(req, "versionOrAlias")
	}
	if version == "" {
		version = pathParam(ctx, "versionoralias")
	}
	if version == "" {
		version = LatestVersion
	}
	if name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	bot, err := store.GetBot(name, version)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(botToMap(bot))
}

func handleGetBots(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	contains := getStr(req, "nameContains")
	bots := store.ListBots(contains)
	out := make([]map[string]any, 0, len(bots))
	for _, b := range bots {
		out = append(out, botMetadata(b))
	}
	return jsonOK(map[string]any{"bots": out})
}

func handleGetBotVersions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	if name == "" {
		name = pathParam(ctx, "name")
	}
	if name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	versions, err := store.ListBotVersions(name)
	if err != nil {
		return jsonErr(err)
	}
	out := make([]map[string]any, 0, len(versions))
	for _, b := range versions {
		out = append(out, botMetadata(b))
	}
	return jsonOK(map[string]any{"bots": out})
}

func handleDeleteBot(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	if name == "" {
		name = pathParam(ctx, "name")
	}
	if name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	if err := store.DeleteBot(name); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDeleteBotVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	version := getStr(req, "version")
	if name == "" || version == "" {
		return jsonErr(service.ErrValidation("name and version are required."))
	}
	if err := store.DeleteBotVersion(name, version); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── Intent handlers ──────────────────────────────────────────────────────────

func handlePutIntent(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	if name == "" {
		name = pathParam(ctx, "name")
	}
	if name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	in := &StoredIntent{
		Name:                  name,
		Description:           getStr(req, "description"),
		ParentIntentSignature: getStr(req, "parentIntentSignature"),
		Slots:                 getMapList(req, "slots"),
		SampleUtterances:      getStrList(req, "sampleUtterances"),
		ConfirmationPrompt:    getMap(req, "confirmationPrompt"),
		RejectionStatement:    getMap(req, "rejectionStatement"),
		FollowUpPrompt:        getMap(req, "followUpPrompt"),
		ConclusionStatement:   getMap(req, "conclusionStatement"),
		DialogCodeHook:        getMap(req, "dialogCodeHook"),
		FulfillmentActivity:   getMap(req, "fulfillmentActivity"),
		InputContexts:         getMapList(req, "inputContexts"),
		OutputContexts:        getMapList(req, "outputContexts"),
		KendraConfiguration:   getMap(req, "kendraConfiguration"),
	}
	saved, err := store.PutIntent(in)
	if err != nil {
		return jsonErr(err)
	}
	createVersion := getBool(req, "createVersion")
	if createVersion {
		if _, vErr := store.CreateIntentVersion(saved.Name); vErr != nil {
			return jsonErr(vErr)
		}
	}
	resp := intentToMap(saved)
	resp["createVersion"] = createVersion
	return jsonOK(resp)
}

func handleCreateIntentVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	if name == "" {
		name = pathParam(ctx, "name")
	}
	if name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	in, err := store.CreateIntentVersion(name)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(intentToMap(in))
}

func handleGetIntent(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	if name == "" {
		name = pathParam(ctx, "name")
	}
	version := getStr(req, "version")
	if version == "" {
		version = LatestVersion
	}
	if name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	in, err := store.GetIntent(name, version)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(intentToMap(in))
}

func handleGetIntents(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	intents := store.ListIntents(getStr(req, "nameContains"))
	out := make([]map[string]any, 0, len(intents))
	for _, in := range intents {
		out = append(out, intentMetadata(in))
	}
	return jsonOK(map[string]any{"intents": out})
}

func handleGetIntentVersions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	if name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	versions, err := store.ListIntentVersions(name)
	if err != nil {
		return jsonErr(err)
	}
	out := make([]map[string]any, 0, len(versions))
	for _, in := range versions {
		out = append(out, intentMetadata(in))
	}
	return jsonOK(map[string]any{"intents": out})
}

func handleDeleteIntent(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	if name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	if err := store.DeleteIntent(name); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDeleteIntentVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	version := getStr(req, "version")
	if name == "" || version == "" {
		return jsonErr(service.ErrValidation("name and version are required."))
	}
	if err := store.DeleteIntentVersion(name, version); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── Slot type handlers ───────────────────────────────────────────────────────

func handlePutSlotType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	if name == "" {
		name = pathParam(ctx, "name")
	}
	if name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	st := &StoredSlotType{
		Name:                    name,
		Description:             getStr(req, "description"),
		ValueSelectionStrategy:  getStr(req, "valueSelectionStrategy"),
		ParentSlotTypeSignature: getStr(req, "parentSlotTypeSignature"),
		EnumerationValues:       getMapList(req, "enumerationValues"),
		SlotTypeConfigurations:  getMapList(req, "slotTypeConfigurations"),
	}
	saved, err := store.PutSlotType(st)
	if err != nil {
		return jsonErr(err)
	}
	createVersion := getBool(req, "createVersion")
	if createVersion {
		if _, vErr := store.CreateSlotTypeVersion(saved.Name); vErr != nil {
			return jsonErr(vErr)
		}
	}
	resp := slotTypeToMap(saved)
	resp["createVersion"] = createVersion
	return jsonOK(resp)
}

func handleCreateSlotTypeVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	if name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	st, err := store.CreateSlotTypeVersion(name)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(slotTypeToMap(st))
}

func handleGetSlotType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	version := getStr(req, "version")
	if version == "" {
		version = LatestVersion
	}
	if name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	st, err := store.GetSlotType(name, version)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(slotTypeToMap(st))
}

func handleGetSlotTypes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	types := store.ListSlotTypes(getStr(req, "nameContains"))
	out := make([]map[string]any, 0, len(types))
	for _, st := range types {
		out = append(out, slotTypeMetadata(st))
	}
	return jsonOK(map[string]any{"slotTypes": out})
}

func handleGetSlotTypeVersions(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	if name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	versions, err := store.ListSlotTypeVersions(name)
	if err != nil {
		return jsonErr(err)
	}
	out := make([]map[string]any, 0, len(versions))
	for _, st := range versions {
		out = append(out, slotTypeMetadata(st))
	}
	return jsonOK(map[string]any{"slotTypes": out})
}

func handleDeleteSlotType(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	if name == "" {
		return jsonErr(service.ErrValidation("name is required."))
	}
	if err := store.DeleteSlotType(name); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

func handleDeleteSlotTypeVersion(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	version := getStr(req, "version")
	if name == "" || version == "" {
		return jsonErr(service.ErrValidation("name and version are required."))
	}
	if err := store.DeleteSlotTypeVersion(name, version); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── Bot alias handlers ───────────────────────────────────────────────────────

func handlePutBotAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	botName := getStr(req, "botName")
	if name == "" || botName == "" {
		return jsonErr(service.ErrValidation("name and botName are required."))
	}
	alias := &StoredBotAlias{
		Name:             name,
		BotName:          botName,
		BotVersion:       getStr(req, "botVersion"),
		Description:      getStr(req, "description"),
		ConversationLogs: getMap(req, "conversationLogs"),
	}
	saved, err := store.PutBotAlias(alias)
	if err != nil {
		return jsonErr(err)
	}
	if tags := parseTagList(req, "tags"); len(tags) > 0 {
		store.TagResource(store.botAliasArn(botName, name), tags)
	}
	resp := botAliasToMap(saved)
	resp["tags"] = tagListToMaps(store.ListTags(store.botAliasArn(botName, name)))
	return jsonOK(resp)
}

func handleGetBotAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	botName := getStr(req, "botName")
	if name == "" || botName == "" {
		return jsonErr(service.ErrValidation("name and botName are required."))
	}
	alias, err := store.GetBotAlias(botName, name)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(botAliasToMap(alias))
}

func handleGetBotAliases(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	botName := getStr(req, "botName")
	if botName == "" {
		return jsonErr(service.ErrValidation("botName is required."))
	}
	aliases := store.ListBotAliases(botName, getStr(req, "nameContains"))
	out := make([]map[string]any, 0, len(aliases))
	for _, a := range aliases {
		out = append(out, botAliasToMap(a))
	}
	return jsonOK(map[string]any{"BotAliases": out})
}

func handleDeleteBotAlias(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	botName := getStr(req, "botName")
	if name == "" || botName == "" {
		return jsonErr(service.ErrValidation("name and botName are required."))
	}
	if err := store.DeleteBotAlias(botName, name); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── Channel association handlers ─────────────────────────────────────────────

func handleGetBotChannelAssociation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	botName := getStr(req, "botName")
	alias := getStr(req, "aliasName")
	if alias == "" {
		alias = getStr(req, "botAlias")
	}
	if name == "" || botName == "" || alias == "" {
		return jsonErr(service.ErrValidation("name, botName, and aliasName are required."))
	}
	a, err := store.GetChannelAssoc(botName, alias, name)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(channelAssocToMap(a))
}

func handleGetBotChannelAssociations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	botName := getStr(req, "botName")
	alias := getStr(req, "aliasName")
	if alias == "" {
		alias = getStr(req, "botAlias")
	}
	if botName == "" || alias == "" {
		return jsonErr(service.ErrValidation("botName and aliasName are required."))
	}
	list := store.ListChannelAssocs(botName, alias, getStr(req, "nameContains"))
	out := make([]map[string]any, 0, len(list))
	for _, a := range list {
		out = append(out, channelAssocToMap(a))
	}
	return jsonOK(map[string]any{"botChannelAssociations": out})
}

func handleDeleteBotChannelAssociation(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	botName := getStr(req, "botName")
	alias := getStr(req, "aliasName")
	if alias == "" {
		alias = getStr(req, "botAlias")
	}
	if name == "" || botName == "" || alias == "" {
		return jsonErr(service.ErrValidation("name, botName, and aliasName are required."))
	}
	if err := store.DeleteChannelAssoc(botName, alias, name); err != nil {
		return jsonErr(err)
	}
	return jsonOK(map[string]any{})
}

// ── Builtin / catalog handlers ───────────────────────────────────────────────

func handleGetBuiltinIntent(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	signature := getStr(req, "signature")
	if signature == "" {
		return jsonErr(service.ErrValidation("signature is required."))
	}
	for _, b := range builtinIntents {
		if b["signature"] == signature {
			return jsonOK(map[string]any{
				"signature":        signature,
				"supportedLocales": b["supportedLocales"],
				"slots":            []any{},
			})
		}
	}
	return jsonErr(service.NewAWSError("NotFoundException",
		"Builtin intent not found: "+signature, 404))
}

func handleGetBuiltinIntents(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	out := filterBuiltins(builtinIntents, getStr(req, "locale"), getStr(req, "signatureContains"))
	return jsonOK(map[string]any{"intents": out})
}

func handleGetBuiltinSlotTypes(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	out := filterBuiltins(builtinSlotTypes, getStr(req, "locale"), getStr(req, "signatureContains"))
	return jsonOK(map[string]any{"slotTypes": out})
}

// ── Export / import / migration handlers ─────────────────────────────────────

func handleGetExport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	name := getStr(req, "name")
	resourceType := getStr(req, "resourceType")
	version := getStr(req, "version")
	exportType := getStr(req, "exportType")
	if name == "" || resourceType == "" || exportType == "" {
		return jsonErr(service.ErrValidation("name, resourceType, and exportType are required."))
	}
	if version == "" {
		version = LatestVersion
	}
	// Verify the resource exists. This mirrors real Lex which rejects exports
	// for unknown resources rather than producing an empty payload.
	switch resourceType {
	case "BOT":
		if _, err := store.GetBot(name, version); err != nil {
			return jsonErr(err)
		}
	case "INTENT":
		if _, err := store.GetIntent(name, version); err != nil {
			return jsonErr(err)
		}
	case "SLOT_TYPE":
		if _, err := store.GetSlotType(name, version); err != nil {
			return jsonErr(err)
		}
	}
	return jsonOK(map[string]any{
		"name":         name,
		"version":      version,
		"resourceType": resourceType,
		"exportType":   exportType,
		"exportStatus": "READY",
		"url":          "https://cloudmock.local/lex/exports/" + name + "/" + version,
	})
}

func handleStartImport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	resourceType := getStr(req, "resourceType")
	mergeStrategy := getStr(req, "mergeStrategy")
	if resourceType == "" || mergeStrategy == "" {
		return jsonErr(service.ErrValidation("resourceType and mergeStrategy are required."))
	}
	imp := store.StartImport("imported-"+resourceType, resourceType, mergeStrategy, parseTagList(req, "tags"))
	resp := importToMap(imp)
	resp["tags"] = tagListToMaps(imp.Tags)
	return jsonOK(resp)
}

func handleGetImport(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "importId")
	if id == "" {
		return jsonErr(service.ErrValidation("importId is required."))
	}
	imp, err := store.GetImport(id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(importToMap(imp))
}

func handleStartMigration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	v1Name := getStr(req, "v1BotName")
	v1Version := getStr(req, "v1BotVersion")
	v2Role := getStr(req, "v2BotRole")
	strategy := getStr(req, "migrationStrategy")
	if v1Name == "" || v1Version == "" || v2Role == "" || strategy == "" {
		return jsonErr(service.ErrValidation(
			"v1BotName, v1BotVersion, v2BotRole, and migrationStrategy are required."))
	}
	if _, err := store.GetBot(v1Name, v1Version); err != nil {
		return jsonErr(err)
	}
	m := store.StartMigration(strategy, v1Name, v1Version, getStr(req, "v2BotName"), v2Role)
	return jsonOK(migrationToMap(m))
}

func handleGetMigration(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	id := getStr(req, "migrationId")
	if id == "" {
		return jsonErr(service.ErrValidation("migrationId is required."))
	}
	m, err := store.GetMigration(id)
	if err != nil {
		return jsonErr(err)
	}
	return jsonOK(migrationToMap(m))
}

func handleGetMigrations(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	migrations := store.ListMigrations()
	out := make([]map[string]any, 0, len(migrations))
	for _, m := range migrations {
		out = append(out, migrationToMap(m))
	}
	return jsonOK(map[string]any{"migrationSummaries": out})
}

// ── Utterance handlers ───────────────────────────────────────────────────────

func handleGetUtterancesView(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	botName := getStr(req, "botname")
	if botName == "" {
		botName = getStr(req, "botName")
	}
	if botName == "" {
		botName = pathParam(ctx, "botname")
	}
	if botName == "" {
		return jsonErr(service.ErrValidation("botName is required."))
	}
	versions := getStrList(req, "bot_versions")
	if len(versions) == 0 {
		versions = []string{LatestVersion}
	}
	utterances := store.GetUtterances(botName)
	utteranceData := make([]map[string]any, 0, len(utterances))
	now := time.Now().UTC()
	for _, u := range utterances {
		utteranceData = append(utteranceData, map[string]any{
			"utteranceString":  u,
			"count":            1,
			"distinctUsers":    1,
			"firstUtteredDate": rfc3339(now),
			"lastUtteredDate":  rfc3339(now),
		})
	}
	out := make([]map[string]any, 0, len(versions))
	for _, v := range versions {
		out = append(out, map[string]any{
			"botVersion": v,
			"utterances": utteranceData,
		})
	}
	return jsonOK(map[string]any{
		"botName":    botName,
		"utterances": out,
	})
}

func handleDeleteUtterances(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	botName := getStr(req, "botName")
	if botName == "" {
		return jsonErr(service.ErrValidation("botName is required."))
	}
	store.DeleteUtterances(botName)
	return jsonOK(map[string]any{})
}

// ── Tag handlers ─────────────────────────────────────────────────────────────

func handleListTagsForResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	return jsonOK(map[string]any{"tags": tagListToMaps(store.ListTags(arn))})
}

func handleTagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	tags := parseTagList(req, "tags")
	if len(tags) == 0 {
		return jsonErr(service.ErrValidation("tags is required."))
	}
	store.TagResource(arn, tags)
	return jsonOK(map[string]any{})
}

func handleUntagResource(ctx *service.RequestContext, store *Store) (*service.Response, error) {
	var req map[string]any
	if awsErr := parseJSON(ctx.Body, &req); awsErr != nil {
		return jsonErr(awsErr)
	}
	arn := getStr(req, "resourceArn")
	if arn == "" {
		return jsonErr(service.ErrValidation("resourceArn is required."))
	}
	keys := getStrList(req, "tagKeys")
	store.UntagResource(arn, keys)
	return jsonOK(map[string]any{})
}
