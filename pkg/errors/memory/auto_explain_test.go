package memory

import (
	"strings"
	"testing"

	errs "github.com/Viridian-Inc/cloudmock/pkg/errors"
)

func TestIngestError_AutoExplainOnThreshold(t *testing.T) {
	s := NewStore(100)
	ev := errs.ErrorEvent{Message: "boom", Stack: "at handler.js:10", Service: "api", SessionID: "s1", Release: "v1.2.3"}

	// Below the threshold: no explanation yet.
	for i := 0; i < errs.AutoExplainThreshold-1; i++ {
		if err := s.IngestError(ev); err != nil {
			t.Fatal(err)
		}
	}
	groups, err := s.GetGroups("", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(groups))
	}
	if groups[0].AutoExplanation != "" {
		t.Errorf("explanation set before threshold: %q", groups[0].AutoExplanation)
	}

	// The threshold-th occurrence triggers the explanation.
	if err := s.IngestError(ev); err != nil {
		t.Fatal(err)
	}
	groups, _ = s.GetGroups("", 10)
	if groups[0].Count != errs.AutoExplainThreshold {
		t.Fatalf("count = %d, want %d", groups[0].Count, errs.AutoExplainThreshold)
	}
	exp := groups[0].AutoExplanation
	if exp == "" {
		t.Fatal("explanation not generated at threshold")
	}
	for _, want := range []string{"boom", "recurring", "v1.2.3"} {
		if !strings.Contains(exp, want) {
			t.Errorf("explanation %q missing %q", exp, want)
		}
	}

	// Further occurrences must not overwrite/regenerate it.
	before := groups[0].AutoExplanation
	_ = s.IngestError(ev)
	groups, _ = s.GetGroups("", 10)
	if groups[0].AutoExplanation != before {
		t.Errorf("explanation regenerated after threshold")
	}
}
