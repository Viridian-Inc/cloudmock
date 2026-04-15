package lexmodels_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/lexmodels"
)

func newGateway(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Default()
	cfg.IAM.Mode = "none"
	reg := routing.NewRegistry()
	reg.Register(svc.New(cfg.AccountID, cfg.Region))
	return gateway.New(cfg, reg)
}

func svcReq(t *testing.T, action string, body any) *http.Request {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
	} else {
		bodyBytes = []byte("{}")
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "lex."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/lex/aws4_request, SignedHeaders=host, Signature=abc123")
	return req
}

func decode(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("decode: %v\nbody: %s", err, w.Body.String())
	}
}

func doCall(t *testing.T, h http.Handler, action string, body any) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, svcReq(t, action, body))
	return w
}

func mustOK(t *testing.T, w *httptest.ResponseRecorder, action string) {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("%s: want 200, got %d: %s", action, w.Code, w.Body.String())
	}
}

// ── Bot lifecycle ────────────────────────────────────────────────────────────

func TestBotLifecycle(t *testing.T) {
	h := newGateway(t)

	// PutBot creates a $LATEST version.
	w := doCall(t, h, "PutBot", map[string]any{
		"name":                    "OrderFlowers",
		"description":             "Order flowers from a Lex bot",
		"locale":                  "en-US",
		"voiceId":                 "Joanna",
		"idleSessionTTLInSeconds": 600,
		"childDirected":           false,
		"intents": []any{
			map[string]any{"intentName": "OrderFlowers", "intentVersion": "$LATEST"},
		},
	})
	mustOK(t, w, "PutBot")
	var put struct {
		Name        string
		Version     string
		Status      string
		Locale      string
		VoiceID     string `json:"voiceId"`
		Checksum    string
		Description string
		Intents     []map[string]any
	}
	decode(t, w, &put)
	if put.Name != "OrderFlowers" {
		t.Fatalf("name: want OrderFlowers, got %q", put.Name)
	}
	if put.Version != "$LATEST" {
		t.Fatalf("version: want $LATEST, got %q", put.Version)
	}
	if put.Status != "READY" {
		t.Fatalf("status: want READY, got %q", put.Status)
	}
	if put.Locale != "en-US" {
		t.Fatalf("locale: want en-US, got %q", put.Locale)
	}
	if put.VoiceID != "Joanna" {
		t.Fatalf("voiceId: want Joanna, got %q", put.VoiceID)
	}
	if put.Checksum == "" {
		t.Fatalf("checksum should be set")
	}
	if len(put.Intents) != 1 {
		t.Fatalf("intents: want 1, got %d", len(put.Intents))
	}

	// GetBot $LATEST returns the same bot.
	w = doCall(t, h, "GetBot", map[string]any{"name": "OrderFlowers", "versionoralias": "$LATEST"})
	mustOK(t, w, "GetBot")
	var got struct {
		Name    string
		Version string
		Locale  string
	}
	decode(t, w, &got)
	if got.Name != "OrderFlowers" || got.Version != "$LATEST" || got.Locale != "en-US" {
		t.Fatalf("GetBot mismatch: %+v", got)
	}

	// CreateBotVersion snapshots $LATEST → "1".
	w = doCall(t, h, "CreateBotVersion", map[string]any{"name": "OrderFlowers"})
	mustOK(t, w, "CreateBotVersion")
	var v1 struct{ Version string }
	decode(t, w, &v1)
	if v1.Version != "1" {
		t.Fatalf("CreateBotVersion: want 1, got %q", v1.Version)
	}

	// GetBotVersions shows both $LATEST and "1".
	w = doCall(t, h, "GetBotVersions", map[string]any{"name": "OrderFlowers"})
	mustOK(t, w, "GetBotVersions")
	var versions struct {
		Bots []struct{ Version string }
	}
	decode(t, w, &versions)
	if len(versions.Bots) != 2 {
		t.Fatalf("GetBotVersions: want 2, got %d", len(versions.Bots))
	}

	// GetBots lists all bots (one per name).
	w = doCall(t, h, "GetBots", nil)
	mustOK(t, w, "GetBots")
	var listed struct {
		Bots []struct{ Name string }
	}
	decode(t, w, &listed)
	if len(listed.Bots) != 1 || listed.Bots[0].Name != "OrderFlowers" {
		t.Fatalf("GetBots: unexpected %+v", listed.Bots)
	}

	// DeleteBotVersion drops "1".
	w = doCall(t, h, "DeleteBotVersion", map[string]any{"name": "OrderFlowers", "version": "1"})
	mustOK(t, w, "DeleteBotVersion")

	w = doCall(t, h, "GetBotVersions", map[string]any{"name": "OrderFlowers"})
	mustOK(t, w, "GetBotVersions after delete")
	decode(t, w, &versions)
	if len(versions.Bots) != 1 {
		t.Fatalf("GetBotVersions after delete: want 1, got %d", len(versions.Bots))
	}

	// DeleteBot removes everything.
	w = doCall(t, h, "DeleteBot", map[string]any{"name": "OrderFlowers"})
	mustOK(t, w, "DeleteBot")

	w = doCall(t, h, "GetBot", map[string]any{"name": "OrderFlowers"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("GetBot after delete: want 404, got %d", w.Code)
	}
}

func TestPutBotRequiresName(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "PutBot", map[string]any{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("PutBot missing name: want 400, got %d", w.Code)
	}
}

func TestDeleteBotVersionRejectsLatest(t *testing.T) {
	h := newGateway(t)
	mustOK(t, doCall(t, h, "PutBot", map[string]any{"name": "B"}), "PutBot")
	w := doCall(t, h, "DeleteBotVersion", map[string]any{"name": "B", "version": "$LATEST"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("DeleteBotVersion $LATEST: want 400, got %d", w.Code)
	}
}

// ── Intent lifecycle ─────────────────────────────────────────────────────────

func TestIntentLifecycle(t *testing.T) {
	h := newGateway(t)

	// PutIntent.
	w := doCall(t, h, "PutIntent", map[string]any{
		"name":        "OrderFlowers",
		"description": "Place a flower order",
		"sampleUtterances": []any{
			"I want to order flowers",
			"Send roses to my friend",
		},
		"slots": []any{
			map[string]any{
				"name":           "FlowerType",
				"slotConstraint": "Required",
				"slotType":       "FlowerTypes",
				"priority":       1,
			},
		},
		"createVersion": true,
	})
	mustOK(t, w, "PutIntent")
	var put struct {
		Name             string
		Version          string
		Slots            []map[string]any
		SampleUtterances []string
		Checksum         string
	}
	decode(t, w, &put)
	if put.Name != "OrderFlowers" || put.Version != "$LATEST" {
		t.Fatalf("PutIntent: %+v", put)
	}
	if len(put.SampleUtterances) != 2 {
		t.Fatalf("sample utterances: want 2, got %d", len(put.SampleUtterances))
	}
	if len(put.Slots) != 1 {
		t.Fatalf("slots: want 1, got %d", len(put.Slots))
	}

	// CreateIntentVersion was already triggered by createVersion=true in PutIntent.
	w = doCall(t, h, "GetIntentVersions", map[string]any{"name": "OrderFlowers"})
	mustOK(t, w, "GetIntentVersions")
	var versions struct {
		Intents []struct{ Version string }
	}
	decode(t, w, &versions)
	if len(versions.Intents) != 2 {
		t.Fatalf("GetIntentVersions after createVersion: want 2, got %d", len(versions.Intents))
	}

	// Explicit CreateIntentVersion bumps to "2".
	w = doCall(t, h, "CreateIntentVersion", map[string]any{"name": "OrderFlowers"})
	mustOK(t, w, "CreateIntentVersion")
	var v2 struct{ Version string }
	decode(t, w, &v2)
	if v2.Version != "2" {
		t.Fatalf("CreateIntentVersion: want 2, got %q", v2.Version)
	}

	// GetIntent specific version.
	w = doCall(t, h, "GetIntent", map[string]any{"name": "OrderFlowers", "version": "1"})
	mustOK(t, w, "GetIntent v1")
	var fetched struct {
		Name    string
		Version string
	}
	decode(t, w, &fetched)
	if fetched.Version != "1" {
		t.Fatalf("GetIntent v1: want version 1, got %q", fetched.Version)
	}

	// GetIntents lists.
	w = doCall(t, h, "GetIntents", nil)
	mustOK(t, w, "GetIntents")
	var listed struct {
		Intents []struct{ Name string }
	}
	decode(t, w, &listed)
	if len(listed.Intents) != 1 {
		t.Fatalf("GetIntents: want 1, got %d", len(listed.Intents))
	}

	// DeleteIntentVersion.
	w = doCall(t, h, "DeleteIntentVersion", map[string]any{"name": "OrderFlowers", "version": "1"})
	mustOK(t, w, "DeleteIntentVersion")

	// DeleteIntent.
	w = doCall(t, h, "DeleteIntent", map[string]any{"name": "OrderFlowers"})
	mustOK(t, w, "DeleteIntent")

	w = doCall(t, h, "GetIntent", map[string]any{"name": "OrderFlowers"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("GetIntent after delete: want 404, got %d", w.Code)
	}
}

// ── Slot type lifecycle ──────────────────────────────────────────────────────

func TestSlotTypeLifecycle(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "PutSlotType", map[string]any{
		"name":        "FlowerTypes",
		"description": "Available flower types",
		"enumerationValues": []any{
			map[string]any{"value": "rose"},
			map[string]any{"value": "lily", "synonyms": []any{"lillies"}},
			map[string]any{"value": "tulip"},
		},
		"valueSelectionStrategy": "ORIGINAL_VALUE",
	})
	mustOK(t, w, "PutSlotType")
	var put struct {
		Name                   string
		Version                string
		EnumerationValues      []map[string]any
		ValueSelectionStrategy string
	}
	decode(t, w, &put)
	if put.Name != "FlowerTypes" || put.Version != "$LATEST" {
		t.Fatalf("PutSlotType: %+v", put)
	}
	if len(put.EnumerationValues) != 3 {
		t.Fatalf("enumerationValues: want 3, got %d", len(put.EnumerationValues))
	}
	if put.ValueSelectionStrategy != "ORIGINAL_VALUE" {
		t.Fatalf("strategy: %q", put.ValueSelectionStrategy)
	}

	w = doCall(t, h, "CreateSlotTypeVersion", map[string]any{"name": "FlowerTypes"})
	mustOK(t, w, "CreateSlotTypeVersion")
	var v1 struct{ Version string }
	decode(t, w, &v1)
	if v1.Version != "1" {
		t.Fatalf("CreateSlotTypeVersion: want 1, got %q", v1.Version)
	}

	w = doCall(t, h, "GetSlotType", map[string]any{"name": "FlowerTypes", "version": "1"})
	mustOK(t, w, "GetSlotType")

	w = doCall(t, h, "GetSlotTypes", nil)
	mustOK(t, w, "GetSlotTypes")
	var listed struct {
		SlotTypes []struct{ Name string }
	}
	decode(t, w, &listed)
	if len(listed.SlotTypes) != 1 {
		t.Fatalf("GetSlotTypes: want 1, got %d", len(listed.SlotTypes))
	}

	w = doCall(t, h, "GetSlotTypeVersions", map[string]any{"name": "FlowerTypes"})
	mustOK(t, w, "GetSlotTypeVersions")
	var vlist struct {
		SlotTypes []struct{ Version string }
	}
	decode(t, w, &vlist)
	if len(vlist.SlotTypes) != 2 {
		t.Fatalf("GetSlotTypeVersions: want 2, got %d", len(vlist.SlotTypes))
	}

	w = doCall(t, h, "DeleteSlotTypeVersion", map[string]any{"name": "FlowerTypes", "version": "1"})
	mustOK(t, w, "DeleteSlotTypeVersion")

	w = doCall(t, h, "DeleteSlotType", map[string]any{"name": "FlowerTypes"})
	mustOK(t, w, "DeleteSlotType")

	w = doCall(t, h, "GetSlotType", map[string]any{"name": "FlowerTypes"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("GetSlotType after delete: want 404, got %d", w.Code)
	}
}

// ── Bot alias lifecycle ──────────────────────────────────────────────────────

func TestBotAliasLifecycle(t *testing.T) {
	h := newGateway(t)
	mustOK(t, doCall(t, h, "PutBot", map[string]any{"name": "ChatBot"}), "PutBot")

	// PutBotAlias requires the bot to exist.
	w := doCall(t, h, "PutBotAlias", map[string]any{
		"name":       "PROD",
		"botName":    "ChatBot",
		"botVersion": "$LATEST",
	})
	mustOK(t, w, "PutBotAlias")
	var put struct {
		Name       string
		BotName    string
		BotVersion string
		Checksum   string
	}
	decode(t, w, &put)
	if put.Name != "PROD" || put.BotName != "ChatBot" {
		t.Fatalf("PutBotAlias: %+v", put)
	}

	// PutBotAlias rejects unknown bot.
	w = doCall(t, h, "PutBotAlias", map[string]any{
		"name":       "X",
		"botName":    "DoesNotExist",
		"botVersion": "$LATEST",
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("PutBotAlias unknown bot: want 404, got %d", w.Code)
	}

	w = doCall(t, h, "GetBotAlias", map[string]any{"name": "PROD", "botName": "ChatBot"})
	mustOK(t, w, "GetBotAlias")

	w = doCall(t, h, "GetBotAliases", map[string]any{"botName": "ChatBot"})
	mustOK(t, w, "GetBotAliases")
	var aliasList struct {
		BotAliases []struct{ Name string }
	}
	decode(t, w, &aliasList)
	if len(aliasList.BotAliases) != 1 {
		t.Fatalf("GetBotAliases: want 1, got %d", len(aliasList.BotAliases))
	}

	w = doCall(t, h, "DeleteBotAlias", map[string]any{"name": "PROD", "botName": "ChatBot"})
	mustOK(t, w, "DeleteBotAlias")

	w = doCall(t, h, "GetBotAlias", map[string]any{"name": "PROD", "botName": "ChatBot"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("GetBotAlias after delete: want 404, got %d", w.Code)
	}
}

// ── Channel association ──────────────────────────────────────────────────────

func TestChannelAssociationLifecycle(t *testing.T) {
	h := newGateway(t)
	mustOK(t, doCall(t, h, "PutBot", map[string]any{"name": "B"}), "PutBot")
	mustOK(t, doCall(t, h, "PutBotAlias", map[string]any{
		"name": "PROD", "botName": "B", "botVersion": "$LATEST",
	}), "PutBotAlias")

	// Channel associations are typically created by AWS console / SDK out of
	// band. There is no PutBotChannelAssociation API in Lex Models V1 — the
	// service returns associations once they exist. We test the read/delete
	// path against an empty store, which should return zero results.
	w := doCall(t, h, "GetBotChannelAssociations", map[string]any{
		"botName": "B", "aliasName": "PROD",
	})
	mustOK(t, w, "GetBotChannelAssociations")
	var list struct {
		BotChannelAssociations []map[string]any
	}
	decode(t, w, &list)
	if len(list.BotChannelAssociations) != 0 {
		t.Fatalf("GetBotChannelAssociations empty: want 0, got %d", len(list.BotChannelAssociations))
	}

	// GetBotChannelAssociation on missing returns 404.
	w = doCall(t, h, "GetBotChannelAssociation", map[string]any{
		"name": "fb", "botName": "B", "aliasName": "PROD",
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("GetBotChannelAssociation missing: want 404, got %d", w.Code)
	}

	// DeleteBotChannelAssociation on missing returns 404.
	w = doCall(t, h, "DeleteBotChannelAssociation", map[string]any{
		"name": "fb", "botName": "B", "aliasName": "PROD",
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("DeleteBotChannelAssociation missing: want 404, got %d", w.Code)
	}
}

// ── Builtin catalogue ────────────────────────────────────────────────────────

func TestBuiltinIntents(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "GetBuiltinIntents", nil)
	mustOK(t, w, "GetBuiltinIntents")
	var all struct {
		Intents []struct{ Signature string }
	}
	decode(t, w, &all)
	if len(all.Intents) == 0 {
		t.Fatalf("expected some builtin intents")
	}

	// Filter by signatureContains.
	w = doCall(t, h, "GetBuiltinIntents", map[string]any{"signatureContains": "Help"})
	mustOK(t, w, "GetBuiltinIntents filter")
	var filtered struct {
		Intents []struct{ Signature string }
	}
	decode(t, w, &filtered)
	if len(filtered.Intents) == 0 {
		t.Fatalf("expected at least one Help match")
	}
	for _, i := range filtered.Intents {
		if i.Signature != "AMAZON.HelpIntent" {
			t.Fatalf("filter: unexpected %q", i.Signature)
		}
	}

	// GetBuiltinIntent fetch one.
	w = doCall(t, h, "GetBuiltinIntent", map[string]any{"signature": "AMAZON.HelpIntent"})
	mustOK(t, w, "GetBuiltinIntent")
	var one struct {
		Signature        string
		SupportedLocales []string
	}
	decode(t, w, &one)
	if one.Signature != "AMAZON.HelpIntent" {
		t.Fatalf("GetBuiltinIntent: %+v", one)
	}

	// Unknown returns 404.
	w = doCall(t, h, "GetBuiltinIntent", map[string]any{"signature": "AMAZON.NopeIntent"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("GetBuiltinIntent unknown: want 404, got %d", w.Code)
	}
}

func TestBuiltinSlotTypes(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "GetBuiltinSlotTypes", nil)
	mustOK(t, w, "GetBuiltinSlotTypes")
	var out struct {
		SlotTypes []struct{ Signature string }
	}
	decode(t, w, &out)
	if len(out.SlotTypes) == 0 {
		t.Fatalf("expected some builtin slot types")
	}
}

// ── Export / import / migration ──────────────────────────────────────────────

func TestExportRequiresExistingResource(t *testing.T) {
	h := newGateway(t)

	// Missing bot returns 404.
	w := doCall(t, h, "GetExport", map[string]any{
		"name":         "Missing",
		"version":      "$LATEST",
		"resourceType": "BOT",
		"exportType":   "LEX",
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("GetExport missing: want 404, got %d", w.Code)
	}

	mustOK(t, doCall(t, h, "PutBot", map[string]any{"name": "B"}), "PutBot")

	w = doCall(t, h, "GetExport", map[string]any{
		"name":         "B",
		"version":      "$LATEST",
		"resourceType": "BOT",
		"exportType":   "LEX",
	})
	mustOK(t, w, "GetExport")
	var exp struct {
		ExportStatus string
		URL          string `json:"url"`
	}
	decode(t, w, &exp)
	if exp.ExportStatus != "READY" {
		t.Fatalf("export status: want READY, got %q", exp.ExportStatus)
	}
	if exp.URL == "" {
		t.Fatalf("export URL should be set")
	}
}

func TestImportLifecycle(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "StartImport", map[string]any{
		"resourceType":  "BOT",
		"mergeStrategy": "OVERWRITE_LATEST",
		"tags":          []any{map[string]any{"key": "owner", "value": "team-x"}},
	})
	mustOK(t, w, "StartImport")
	var start struct {
		ImportID     string `json:"importId"`
		ImportStatus string
		Tags         []map[string]any
	}
	decode(t, w, &start)
	if start.ImportID == "" {
		t.Fatalf("expected import id")
	}
	if start.ImportStatus != "COMPLETE" {
		t.Fatalf("import status: want COMPLETE, got %q", start.ImportStatus)
	}
	if len(start.Tags) != 1 {
		t.Fatalf("tags: want 1, got %d", len(start.Tags))
	}

	w = doCall(t, h, "GetImport", map[string]any{"importId": start.ImportID})
	mustOK(t, w, "GetImport")
	var got struct {
		ImportID     string `json:"importId"`
		ImportStatus string
	}
	decode(t, w, &got)
	if got.ImportID != start.ImportID {
		t.Fatalf("GetImport: id mismatch")
	}

	// Missing import returns 404.
	w = doCall(t, h, "GetImport", map[string]any{"importId": "doesnotexist"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("GetImport missing: want 404, got %d", w.Code)
	}
}

func TestMigrationLifecycle(t *testing.T) {
	h := newGateway(t)
	mustOK(t, doCall(t, h, "PutBot", map[string]any{"name": "Legacy"}), "PutBot")

	w := doCall(t, h, "StartMigration", map[string]any{
		"v1BotName":         "Legacy",
		"v1BotVersion":      "$LATEST",
		"v2BotName":         "LegacyV2",
		"v2BotRole":         "arn:aws:iam::123456789012:role/lex-migrate",
		"migrationStrategy": "CREATE_NEW",
	})
	mustOK(t, w, "StartMigration")
	var start struct {
		MigrationID     string `json:"migrationId"`
		MigrationStatus string
	}
	decode(t, w, &start)
	if start.MigrationID == "" {
		t.Fatalf("expected migration id")
	}
	if start.MigrationStatus != "COMPLETED" {
		t.Fatalf("status: want COMPLETED, got %q", start.MigrationStatus)
	}

	w = doCall(t, h, "GetMigration", map[string]any{"migrationId": start.MigrationID})
	mustOK(t, w, "GetMigration")

	w = doCall(t, h, "GetMigrations", nil)
	mustOK(t, w, "GetMigrations")
	var list struct {
		MigrationSummaries []map[string]any
	}
	decode(t, w, &list)
	if len(list.MigrationSummaries) != 1 {
		t.Fatalf("GetMigrations: want 1, got %d", len(list.MigrationSummaries))
	}

	// StartMigration with unknown v1 bot fails.
	w = doCall(t, h, "StartMigration", map[string]any{
		"v1BotName":         "NopeBot",
		"v1BotVersion":      "$LATEST",
		"v2BotRole":         "arn:role",
		"migrationStrategy": "CREATE_NEW",
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("StartMigration unknown bot: want 404, got %d", w.Code)
	}
}

// ── Utterances ───────────────────────────────────────────────────────────────

func TestUtterancesView(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "GetUtterancesView", map[string]any{
		"botname":      "B",
		"bot_versions": []any{"$LATEST"},
		"status_type":  "Detected",
	})
	mustOK(t, w, "GetUtterancesView")
	var view struct {
		BotName    string
		Utterances []struct {
			BotVersion string
			Utterances []map[string]any
		}
	}
	decode(t, w, &view)
	if view.BotName != "B" {
		t.Fatalf("botName: %q", view.BotName)
	}
	if len(view.Utterances) != 1 {
		t.Fatalf("utterances list: want 1, got %d", len(view.Utterances))
	}

	// DeleteUtterances is a no-op without recorded data; should still return 200.
	w = doCall(t, h, "DeleteUtterances", map[string]any{"botName": "B"})
	mustOK(t, w, "DeleteUtterances")
}

// ── Tagging ──────────────────────────────────────────────────────────────────

func TestTagging(t *testing.T) {
	h := newGateway(t)
	arn := "arn:aws:lex:us-east-1:123456789012:bot:Sample"

	mustOK(t, doCall(t, h, "TagResource", map[string]any{
		"resourceArn": arn,
		"tags": []any{
			map[string]any{"key": "env", "value": "prod"},
			map[string]any{"key": "team", "value": "core"},
		},
	}), "TagResource")

	w := doCall(t, h, "ListTagsForResource", map[string]any{"resourceArn": arn})
	mustOK(t, w, "ListTagsForResource")
	var list struct {
		Tags []struct {
			Key   string
			Value string
		}
	}
	decode(t, w, &list)
	if len(list.Tags) != 2 {
		t.Fatalf("ListTagsForResource: want 2, got %d", len(list.Tags))
	}

	mustOK(t, doCall(t, h, "UntagResource", map[string]any{
		"resourceArn": arn,
		"tagKeys":     []any{"env"},
	}), "UntagResource")

	w = doCall(t, h, "ListTagsForResource", map[string]any{"resourceArn": arn})
	mustOK(t, w, "ListTagsForResource after untag")
	decode(t, w, &list)
	if len(list.Tags) != 1 || list.Tags[0].Key != "team" {
		t.Fatalf("ListTagsForResource after untag: %+v", list.Tags)
	}
}

// ── Validation ───────────────────────────────────────────────────────────────

func TestValidationErrors(t *testing.T) {
	h := newGateway(t)
	cases := []struct {
		action string
		body   map[string]any
	}{
		{"PutBot", map[string]any{}},
		{"PutIntent", map[string]any{}},
		{"PutSlotType", map[string]any{}},
		{"PutBotAlias", map[string]any{}},
		{"GetBot", map[string]any{}},
		{"GetIntent", map[string]any{}},
		{"GetSlotType", map[string]any{}},
		{"DeleteBot", map[string]any{}},
		{"DeleteIntent", map[string]any{}},
		{"DeleteSlotType", map[string]any{}},
		{"ListTagsForResource", map[string]any{}},
		{"TagResource", map[string]any{}},
		{"UntagResource", map[string]any{}},
		{"GetBuiltinIntent", map[string]any{}},
		{"StartImport", map[string]any{}},
		{"GetImport", map[string]any{}},
		{"StartMigration", map[string]any{}},
		{"GetMigration", map[string]any{}},
		{"GetExport", map[string]any{}},
	}
	for _, c := range cases {
		w := doCall(t, h, c.action, c.body)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s: want 400 on missing input, got %d (%s)", c.action, w.Code, w.Body.String())
		}
	}
}

// ── Persistence sanity check ─────────────────────────────────────────────────

func TestPersistsAcrossCalls(t *testing.T) {
	h := newGateway(t)
	for _, name := range []string{"alpha", "beta", "gamma"} {
		mustOK(t, doCall(t, h, "PutBot", map[string]any{"name": name}), "PutBot "+name)
	}
	w := doCall(t, h, "GetBots", nil)
	mustOK(t, w, "GetBots")
	var listed struct {
		Bots []struct{ Name string }
	}
	decode(t, w, &listed)
	if len(listed.Bots) != 3 {
		t.Fatalf("expected 3 bots, got %d", len(listed.Bots))
	}
}
