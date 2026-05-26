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
	case "init":
		return handleInit(remaining)
	case "remote":
		return handleRemote(remaining)
	case "status":
		return handleStatus(remaining)
	case "push":
		return handlePush(remaining)
	case "clone":
		return handleClone(remaining)
	case "pull":
		return handlePull(remaining)
	case "resolve":
		return handleResolve(remaining)
	case "download":
		return handleDownload(remaining)
	case "install":
		return handleInstall(remaining)
	case "check":
		return handleCheck(remaining)
	case "update":
		return handleUpdate(remaining)
	case "skills":
		return handleSkills(remaining)
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

func handleInit(args []string) error {
	jsonOutput := hasFlag(args, "--json")
	return RunInit(args, jsonOutput)
}

func handleRemote(args []string) error {
	jsonOutput := hasFlag(args, "--json")
	force := hasFlag(args, "--force")
	filtered := filterFlags(args)
	return RunRemote(filtered, jsonOutput, force)
}

func handleStatus(args []string) error {
	jsonOutput := hasFlag(args, "--json")
	return RunStatus(args, jsonOutput)
}

func handlePush(args []string) error {
	jsonOutput := hasFlag(args, "--json")
	return RunPush(args, jsonOutput)
}

func handleClone(args []string) error {
	jsonOutput := hasFlag(args, "--json")
	filtered := filterFlags(args)
	return RunClone(filtered, jsonOutput)
}

func handlePull(args []string) error {
	jsonOutput := hasFlag(args, "--json")
	return RunPull(args, jsonOutput)
}

func handleResolve(args []string) error {
	jsonOutput := hasFlag(args, "--json")
	filtered := filterFlags(args)
	return RunResolve(filtered, jsonOutput)
}

func handleDownload(args []string) error {
	jsonOutput := hasFlag(args, "--json")
	metadataOnly := hasFlag(args, "--metadata-only")
	overwrite := hasFlag(args, "--overwrite")
	outputPath := flagValue(args, "--output")
	filtered := filterFlags(args)
	return RunDownload(filtered, outputPath, overwrite, metadataOnly, jsonOutput)
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
		"--output":          true,
		"--remote":          true,
		"--package":         true,
		"--display-name":    true,
		"--description":     true,
		"--visibility":      true,
		"--maintainer-team": true,
		"--maintainer-user": true,
		"--version":         true,
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
	serve|server
	  Start the Trove registry API and web UI server

	init agents-md [--remote url-or-package-ref]
	  Initialize an editable AGENTS.md package worktree

	remote add|list|remove
	  Manage publishing remotes in .trove/config.yaml

	status [--json]
	  Show local AGENTS.md publishing status

	push [--patch|--minor|--major|--version x.y.z] [--force] [--json]
	  Upload and publish or submit an AGENTS.md package

	clone <org/namespace/package[@selector]> [dir]
	  Create an editable package worktree

	pull [--json]
	  Refresh an editable package worktree from its remote

	resolve <org/namespace/package@selector>
	  Resolve a selector to an exact version

  download <org/namespace/package[@selector]> <artifact-path> [--output file] [--overwrite] [--json]
  download core/skills/<name>/SKILL.md [--output file] [--overwrite] [--json]
    Download one artifact file or bundled core skill

  install <org/namespace/package@selector> [--output dir] [--optional] [--overwrite] [--json]
    Install package artifacts to their target paths

  check [--json]
    Check installed packages for updates (reads .trove.lock.yaml)

  update [--apply] [--json]
    Check for updates and optionally apply them (dry-run by default)

  skills find [query] [--json]
    Find Trove-hosted agent skills

Flags:
  --json        Output in JSON format for agents/CI
  --output path Output path for install/download
  --optional    Include optional artifacts during install
	--overwrite   Overwrite existing files during install
	--apply       Apply updates and write lockfile
	--force       Reset unpublished review versions during push

Environment:
  TROVE_SERVER_URL  Server URL (default: http://localhost:8080)
  TROVE_TOKEN       Bearer token for authentication`)
}

func IsCLICommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "init", "remote", "status", "push", "clone", "pull", "resolve", "download", "install", "check", "update", "skills", "help", "--help", "-h", "version", "--version", "-v":
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
