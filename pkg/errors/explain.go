package errors

import (
	"fmt"
	"strings"
	"time"
)

// AutoExplainThreshold is the occurrence count at which an error group is
// automatically given an explanation.
const AutoExplainThreshold = 5

// BuildAutoExplanation produces a short, deterministic explanation for an error
// group that has become significant (crossed AutoExplainThreshold occurrences).
// It is generated from the group's own data — no external LLM, no cross-package
// dependency — so it can be computed inline during ingest.
func BuildAutoExplanation(g *ErrorGroup) string {
	if g == nil {
		return ""
	}
	kind := g.Type
	if kind == "" {
		kind = "error"
	}

	var b strings.Builder
	msg := g.Message
	if msg == "" {
		msg = "(no message)"
	}
	fmt.Fprintf(&b, "%q has occurred %d times", msg, g.Count)
	if g.Sessions > 0 {
		fmt.Fprintf(&b, " across %d session(s)", g.Sessions)
	}
	if !g.FirstSeen.IsZero() {
		fmt.Fprintf(&b, " since %s", g.FirstSeen.UTC().Format(time.RFC3339))
	}
	b.WriteString(".")
	if g.Source != "" {
		fmt.Fprintf(&b, " Source: %s.", g.Source)
	}
	if g.Release != "" {
		fmt.Fprintf(&b, " First seen in release %s.", g.Release)
	}
	fmt.Fprintf(&b, " It crossed the %d-occurrence threshold, so this %s is a recurring issue worth investigating.",
		AutoExplainThreshold, kind)
	return b.String()
}
