package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed templates/*.yml
var templateFS embed.FS

var ciFileMapping = map[string]struct {
	template string
	output   string
}{
	"github":    {template: "templates/github.yml", output: ".github/workflows/cloudmock.yml"},
	"gitlab":    {template: "templates/gitlab.yml", output: ".gitlab-ci.yml"},
	"circleci":  {template: "templates/circleci.yml", output: ".circleci/config.yml"},
	"bitbucket": {template: "templates/bitbucket.yml", output: "bitbucket-pipelines.yml"},
	"buildkite": {template: "templates/buildkite.yml", output: ".buildkite/pipeline.yml"},
	"codebuild": {template: "templates/codebuild.yml", output: "buildspec.yml"},
	"travis":    {template: "templates/travis.yml", output: ".travis.yml"},
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	ci := fs.String("ci", "", "CI system: github|gitlab|circleci|bitbucket|buildkite|codebuild|travis")
	fs.Parse(args)

	if *ci == "" {
		return fmt.Errorf("--ci flag is required (github|gitlab|circleci|bitbucket|buildkite|codebuild|travis)")
	}

	mapping, ok := ciFileMapping[*ci]
	if !ok {
		return fmt.Errorf("unsupported CI system: %s", *ci)
	}

	data, err := templateFS.ReadFile(mapping.template)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	outputDir := filepath.Dir(mapping.output)
	if outputDir != "." {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
		}
	}

	if err := os.WriteFile(mapping.output, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", mapping.output, err)
	}

	fmt.Printf("Generated %s CI configuration: %s\n", *ci, mapping.output)
	return nil
}
