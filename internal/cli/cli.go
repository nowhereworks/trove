package cli

import (
	"fmt"
	"os"
	"strings"
)

func Run(args []string) error {
	if len(args) == 0 {
		return nil
	}

	subcommand := args[0]
	remaining := args[1:]

	switch subcommand {
	case "resolve":
		return handleResolve(remaining)
	case "fetch":
		return handleFetch(remaining)
	case "install":
		return handleInstall(remaining)
	case "check":
		return handleCheck(remaining)
	case "update":
		return handleUpdate(remaining)
	case "help", "--help", "-h":
		printUsage()
		return nil
	case "version", "--version", "-v":
		fmt.Println("trove 0.1.0")
		return nil
	default:
		return fmt.Errorf("unknown subcommand: %s\nRun 'trove help' for usage", subcommand)
	}
}

func handleResolve(args []string) error {
	jsonOutput := hasFlag(args, "--json")
	filtered := filterFlags(args)
	return RunResolve(filtered, jsonOutput)
}

func handleFetch(args []string) error {
	jsonOutput := hasFlag(args, "--json")
	outputDir := flagValue(args, "--output")
	filtered := filterFlags(args)
	return RunFetch(filtered, outputDir, jsonOutput)
}

func handleInstall(args []string) error {
	jsonOutput := hasFlag(args, "--json")
	optional := hasFlag(args, "--optional")
	overwrite := hasFlag(args, "--overwrite")
	outputDir := flagValue(args, "--output")
	filtered := filterFlags(args)
	return RunInstall(filtered, outputDir, optional, overwrite, jsonOutput)
}

func handleCheck(args []string) error {
	jsonOutput := hasFlag(args, "--json")
	return RunCheck(nil, jsonOutput)
}

func handleUpdate(args []string) error {
	jsonOutput := hasFlag(args, "--json")
	apply := hasFlag(args, "--apply")
	return RunUpdate(nil, apply, jsonOutput)
}

func hasFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

func flagValue(args []string, flag string) string {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(arg, flag+"=") {
			return strings.TrimPrefix(arg, flag+"=")
		}
	}
	return ""
}

func filterFlags(args []string) []string {
	valueFlags := map[string]bool{
		"--output": true,
	}

	var result []string
	skipNext := false
	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if strings.HasPrefix(arg, "--") {
			if valueFlags[arg] && i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				skipNext = true
			}
			continue
		}
		result = append(result, arg)
	}
	if result == nil {
		return []string{}
	}
	return result
}

func printUsage() {
	fmt.Println(`trove - Agent artifact registry CLI

Usage:
  trove <subcommand> [flags]

Subcommands:
  resolve <org/namespace/package@selector>
    Resolve a selector to an exact version

  fetch <org/namespace/package@selector> [--output dir] [--json]
    Fetch and extract a package archive to a directory

  install <org/namespace/package@selector> [--output dir] [--optional] [--overwrite] [--json]
    Install package artifacts to their target paths

  check [--json]
    Check installed packages for updates (reads .trove.lock.yaml)

  update [--apply] [--json]
    Check for updates and optionally apply them (dry-run by default)

Flags:
  --json        Output in JSON format for agents/CI
  --output dir  Output directory (default: current directory or package name)
  --optional    Include optional artifacts during install
  --overwrite   Overwrite existing files during install
  --apply       Apply updates and write lockfile

Environment:
  TROVE_SERVER_URL  Server URL (default: http://localhost:8080)
  TROVE_TOKEN       Bearer token for authentication`)
}

func IsCLICommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "resolve", "fetch", "install", "check", "update", "help", "--help", "-h", "version", "--version", "-v":
		return true
	default:
		return false
	}
}

func ExitWithUsage(code int, msg string) {
	fmt.Fprintln(os.Stderr, msg)
	fmt.Fprintln(os.Stderr)
	printUsage()
	os.Exit(code)
}
