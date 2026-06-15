package admin

import (
	"testing"

	errs "github.com/Viridian-Inc/cloudmock/pkg/errors"
)

func TestDeployMatchesRelease(t *testing.T) {
	d := DeployEvent{ID: "deploy-1", Commit: "abcdef1234567890", Branch: "main"}
	cases := []struct {
		release string
		want    bool
	}{
		{"abcdef1234567890", true}, // exact commit
		{"abcdef1", true},          // short commit prefix (>=7)
		{"main", true},             // branch
		{"deploy-1", true},         // deploy id
		{"abc", false},             // too-short prefix
		{"zzzzzzz", false},         // no match
		{"", false},                // empty release
	}
	for _, c := range cases {
		if got := deployMatchesRelease(d, c.release); got != c.want {
			t.Errorf("deployMatchesRelease(%q) = %v, want %v", c.release, got, c.want)
		}
	}
}

func TestCorrelateErrorDeploys(t *testing.T) {
	groups := []errs.ErrorGroup{
		{ID: "g1", Message: "boom", Release: "abcdef1234567890", Status: "unresolved", Count: 5},
		{ID: "g2", Message: "no release", Release: ""},          // skipped (no release)
		{ID: "g3", Message: "unmatched", Release: "deadbeef99"}, // no matching deploy
	}
	deploys := []DeployEvent{
		{ID: "d1", Service: "bff", Commit: "abcdef1234567890", Branch: "main", Author: "alice"},
		{ID: "d2", Service: "api", Commit: "0000000000000000", Branch: "release"},
	}

	links := correlateErrorDeploys(groups, deploys)
	if len(links) != 1 {
		t.Fatalf("got %d links, want 1: %+v", len(links), links)
	}
	l := links[0]
	if l.GroupID != "g1" || l.Release != "abcdef1234567890" || l.Count != 5 {
		t.Errorf("link = %+v", l)
	}
	if len(l.Deploys) != 1 || l.Deploys[0].ID != "d1" {
		t.Errorf("matched deploys = %+v, want [d1]", l.Deploys)
	}
}
