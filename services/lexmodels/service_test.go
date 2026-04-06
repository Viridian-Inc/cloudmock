package lexmodels_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/neureaux/cloudmock/pkg/gateway"
	"github.com/neureaux/cloudmock/pkg/routing"
	svc "github.com/neureaux/cloudmock/services/lexmodels"
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


func TestCreateBotVersion(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateBotVersion", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateBotVersion: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateIntentVersion(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateIntentVersion", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateIntentVersion: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreateSlotTypeVersion(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "CreateSlotTypeVersion", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("CreateSlotTypeVersion: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteBot(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteBot", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteBot: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteBotAlias(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteBotAlias", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteBotAlias: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteBotChannelAssociation(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteBotChannelAssociation", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteBotChannelAssociation: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteBotVersion(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteBotVersion", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteBotVersion: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteIntent(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteIntent", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteIntent: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteIntentVersion(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteIntentVersion", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteIntentVersion: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteSlotType(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteSlotType", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteSlotType: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteSlotTypeVersion(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteSlotTypeVersion", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteSlotTypeVersion: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestDeleteUtterances(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "DeleteUtterances", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteUtterances: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetBot(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetBot", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetBot: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetBotAlias(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetBotAlias", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetBotAlias: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetBotAliases(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetBotAliases", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetBotAliases: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetBotChannelAssociation(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetBotChannelAssociation", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetBotChannelAssociation: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetBotChannelAssociations(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetBotChannelAssociations", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetBotChannelAssociations: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetBotVersions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetBotVersions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetBotVersions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetBots(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetBots", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetBots: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetBuiltinIntent(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetBuiltinIntent", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetBuiltinIntent: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetBuiltinIntents(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetBuiltinIntents", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetBuiltinIntents: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetBuiltinSlotTypes(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetBuiltinSlotTypes", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetBuiltinSlotTypes: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetExport(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetExport", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetExport: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetImport(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetImport", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetImport: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetIntent(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetIntent", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetIntent: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetIntentVersions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetIntentVersions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetIntentVersions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetIntents(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetIntents", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetIntents: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetMigration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetMigration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetMigration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetMigrations(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetMigrations", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetMigrations: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetSlotType(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetSlotType", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetSlotType: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetSlotTypeVersions(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetSlotTypeVersions", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetSlotTypeVersions: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetSlotTypes(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetSlotTypes", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetSlotTypes: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetUtterancesView(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "GetUtterancesView", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GetUtterancesView: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListTagsForResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "ListTagsForResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ListTagsForResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestPutBot(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "PutBot", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("PutBot: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestPutBotAlias(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "PutBotAlias", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("PutBotAlias: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestPutIntent(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "PutIntent", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("PutIntent: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestPutSlotType(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "PutSlotType", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("PutSlotType: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartImport(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartImport", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartImport: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestStartMigration(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "StartMigration", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartMigration: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestTagResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "TagResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("TagResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUntagResource(t *testing.T) {
	handler := newGateway(t)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, svcReq(t, "UntagResource", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("UntagResource: expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

