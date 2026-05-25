package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func handleSkills(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: trove skills find [query] [--json]")
	}
	subcommand := args[0]
	remaining := args[1:]
	switch subcommand {
	case "find":
		jsonOutput := hasFlag(remaining, "--json")
		return RunSkillsFind(filterFlags(remaining), jsonOutput)
	default:
		return fmt.Errorf("unknown skills subcommand: %s", subcommand)
	}
}

func RunSkillsFind(args []string, jsonOutput bool) error {
	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		query = "skill"
	}

	result, err := NewClient().SearchPackages(SearchPackagesParams{Query: query, ArtifactType: "skill"})
	if err != nil {
		return err
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	if len(result.Items) == 0 {
		fmt.Printf("No Trove skills found for %q.\n", query)
		return nil
	}

	fmt.Printf("Skills matching %q:\n", query)
	for _, item := range result.Items {
		fmt.Printf("- %s\n", item.Ref())
		if item.Description != "" {
			fmt.Printf("  %s\n", item.Description)
		}
	}
	return nil
}

func (p PackageSummary) Ref() string {
	selector := p.StableVersion
	if selector != "" {
		selector = "stable"
	} else if p.LatestVersion != "" {
		selector = "latest"
	}
	ref := p.Org + "/" + p.Namespace + "/" + p.Name
	if selector != "" {
		ref += "@" + selector
	}
	return ref
}
