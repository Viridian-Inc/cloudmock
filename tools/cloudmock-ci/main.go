package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		if err := runInit(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "wait":
		if err := runWait(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "run":
		exitCode, err := runRun(os.Args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(exitCode)
	case "report":
		if err := runReport(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`cloudmock-ci — CI helper for cloudmock

Usage:
  cloudmock-ci <command> [flags]

Commands:
  init     Generate CI configuration for your platform
  wait     Wait for cloudmock to become healthy
  run      Run a tool with cloudmock environment
  report   Generate a test report

Flags:
  -h, --help   Show this help message

Use "cloudmock-ci <command> --help" for more information about a command.`)
}
