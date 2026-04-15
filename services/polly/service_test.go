package polly_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/config"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	svc "github.com/Viridian-Inc/cloudmock/services/polly"
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
	req.Header.Set("X-Amz-Target", "polly."+action)
	req.Header.Set("Authorization",
		"AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/polly/aws4_request, SignedHeaders=host, Signature=abc123")
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

func TestDescribeVoices(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "DescribeVoices", map[string]any{})
	if w.Code != http.StatusOK {
		t.Fatalf("DescribeVoices: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var out struct {
		Voices []struct {
			ID           string `json:"Id"`
			LanguageCode string
		}
	}
	decode(t, w, &out)
	if len(out.Voices) == 0 {
		t.Fatalf("expected static voices, got 0")
	}
}

func TestDescribeVoicesFilteredByLanguage(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "DescribeVoices", map[string]any{"LanguageCode": "en-GB"})
	var out struct {
		Voices []struct {
			LanguageCode string
		}
	}
	decode(t, w, &out)
	if len(out.Voices) == 0 {
		t.Fatalf("expected en-GB voices, got 0")
	}
	for _, v := range out.Voices {
		if v.LanguageCode != "en-GB" {
			t.Fatalf("unexpected language code: %s", v.LanguageCode)
		}
	}
}

func TestLexiconLifecycle(t *testing.T) {
	h := newGateway(t)

	lex := `<lexicon version="1.0" xml:lang="en-US"><lexeme><grapheme>IPA</grapheme><phoneme>aɪ pi eɪ</phoneme></lexeme></lexicon>`
	if w := doCall(t, h, "PutLexicon", map[string]any{"LexiconName": "test", "Content": lex}); w.Code != http.StatusOK {
		t.Fatalf("PutLexicon: want 200, got %d: %s", w.Code, w.Body.String())
	}

	w := doCall(t, h, "ListLexicons", nil)
	var listed struct {
		Lexicons []struct {
			Name       string
			Attributes struct {
				LexemesCount int
			}
		}
	}
	decode(t, w, &listed)
	if len(listed.Lexicons) != 1 {
		t.Fatalf("expected 1 lexicon, got %d", len(listed.Lexicons))
	}
	if listed.Lexicons[0].Name != "test" {
		t.Fatalf("expected name=test, got %q", listed.Lexicons[0].Name)
	}
	if listed.Lexicons[0].Attributes.LexemesCount != 1 {
		t.Fatalf("expected 1 lexeme, got %d", listed.Lexicons[0].Attributes.LexemesCount)
	}

	w = doCall(t, h, "GetLexicon", map[string]any{"LexiconName": "test"})
	var got struct {
		Lexicon struct {
			Name    string
			Content string
		}
	}
	decode(t, w, &got)
	if got.Lexicon.Content != lex {
		t.Fatalf("lexicon content roundtrip broken")
	}

	if w := doCall(t, h, "DeleteLexicon", map[string]any{"LexiconName": "test"}); w.Code != http.StatusOK {
		t.Fatalf("DeleteLexicon: want 200, got %d", w.Code)
	}
	w = doCall(t, h, "GetLexicon", map[string]any{"LexiconName": "test"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("GetLexicon after delete: want 404, got %d", w.Code)
	}
}

func TestSynthesizeSpeechValidates(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "SynthesizeSpeech", map[string]any{"Text": "hello", "OutputFormat": "mp3"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing voice: want 400, got %d", w.Code)
	}

	w = doCall(t, h, "SynthesizeSpeech", map[string]any{
		"Text": "hello", "OutputFormat": "mp3", "VoiceId": "Joanna",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("valid synth: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var out struct {
		AudioStream       []byte  `json:"AudioStream"`
		ContentType       string  `json:"Content-Type"`
		RequestCharacters int     `json:"x-amzn-RequestCharacters"`
	}
	decode(t, w, &out)
	if out.ContentType != "audio/mpeg" {
		t.Fatalf("unexpected content type: %s", out.ContentType)
	}
	if out.RequestCharacters != 5 {
		t.Fatalf("expected 5 chars, got %d", out.RequestCharacters)
	}
}

func TestSynthesisTaskLifecycle(t *testing.T) {
	h := newGateway(t)

	w := doCall(t, h, "StartSpeechSynthesisTask", map[string]any{
		"Text":               "hello world",
		"OutputFormat":       "mp3",
		"OutputS3BucketName": "my-bucket",
		"VoiceId":            "Matthew",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("StartSpeechSynthesisTask: want 200, got %d: %s", w.Code, w.Body.String())
	}
	var start struct {
		SynthesisTask struct {
			TaskID     string `json:"TaskId"`
			OutputURI  string `json:"OutputUri"`
			TaskStatus string
		}
	}
	decode(t, w, &start)
	if start.SynthesisTask.TaskID == "" {
		t.Fatalf("expected task id")
	}
	if start.SynthesisTask.TaskStatus != "completed" {
		t.Fatalf("expected completed, got %s", start.SynthesisTask.TaskStatus)
	}

	w = doCall(t, h, "GetSpeechSynthesisTask", map[string]any{"TaskId": start.SynthesisTask.TaskID})
	if w.Code != http.StatusOK {
		t.Fatalf("GetSpeechSynthesisTask: want 200, got %d", w.Code)
	}

	w = doCall(t, h, "ListSpeechSynthesisTasks", nil)
	var list struct {
		SynthesisTasks []struct {
			TaskID string `json:"TaskId"`
		}
	}
	decode(t, w, &list)
	if len(list.SynthesisTasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(list.SynthesisTasks))
	}
}

func TestStartSpeechSynthesisTaskValidatesVoice(t *testing.T) {
	h := newGateway(t)
	w := doCall(t, h, "StartSpeechSynthesisTask", map[string]any{
		"Text":               "hi",
		"OutputFormat":       "mp3",
		"OutputS3BucketName": "b",
		"VoiceId":            "NotARealVoice",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid voice: want 400, got %d", w.Code)
	}
}
