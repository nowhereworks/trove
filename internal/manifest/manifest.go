package manifest

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	APIVersion = "trove.io/v1"
	Kind       = "AgentArtifactPackage"
)

type Manifest struct {
	APIVersion string   `yaml:"apiVersion" json:"apiVersion"`
	Kind       string   `yaml:"kind" json:"kind"`
	Metadata   Metadata `yaml:"metadata" json:"metadata"`
	Spec       Spec     `yaml:"spec" json:"spec"`
}

type Metadata struct {
	Org         string            `yaml:"org" json:"org"`
	Namespace   string            `yaml:"namespace" json:"namespace"`
	Name        string            `yaml:"name" json:"name"`
	DisplayName string            `yaml:"displayName" json:"displayName"`
	Description string            `yaml:"description" json:"description"`
	Labels      map[string]string `yaml:"labels" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations" json:"annotations,omitempty"`
}

type Spec struct {
	Artifacts     []Artifact        `yaml:"artifacts" json:"artifacts"`
	Dependencies  []string          `yaml:"dependencies" json:"dependencies,omitempty"`
}

type Artifact struct {
	Path       string `yaml:"path" json:"path"`
	Type       string `yaml:"type" json:"type"`
	Required   bool   `yaml:"required" json:"required"`
	TargetPath string `yaml:"targetPath" json:"targetPath,omitempty"`
}

type ValidateOptions struct {
	Org       string
	Namespace string
	Package   string
}

type Error struct {
	Problems []Problem
}

type Problem struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	if len(e.Problems) == 0 {
		return "manifest is invalid"
	}
	return fmt.Sprintf("manifest is invalid: %s %s", e.Problems[0].Field, e.Problems[0].Message)
}

var (
	slugRE      = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$`)
	slugValueRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)
)

var knownArtifactTypes = map[string]struct{}{
	"adapter":            {},
	"agent-instructions": {},
	"command":            {},
	"context-pack":       {},
	"policy":             {},
	"prompt":             {},
	"rubric":             {},
	"skill":              {},
	"subagent":           {},
	"template":           {},
}

func Parse(data []byte) (Manifest, error) {
	var parsed Manifest
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest YAML: %w", err)
	}
	return parsed, nil
}

func Validate(m Manifest, opts ValidateOptions) error {
	var problems []Problem

	requireEqual(&problems, "apiVersion", m.APIVersion, APIVersion)
	requireEqual(&problems, "kind", m.Kind, Kind)
	requireSlug(&problems, "metadata.org", m.Metadata.Org)
	requireSlug(&problems, "metadata.namespace", m.Metadata.Namespace)
	requireSlug(&problems, "metadata.name", m.Metadata.Name)
	requireNonEmpty(&problems, "metadata.displayName", m.Metadata.DisplayName)
	requireNonEmpty(&problems, "metadata.description", m.Metadata.Description)

	if opts.Org != "" && m.Metadata.Org != opts.Org {
		problems = append(problems, Problem{Field: "metadata.org", Message: "must match route org"})
	}
	if opts.Namespace != "" && m.Metadata.Namespace != opts.Namespace {
		problems = append(problems, Problem{Field: "metadata.namespace", Message: "must match route namespace"})
	}
	if opts.Package != "" && m.Metadata.Name != opts.Package {
		problems = append(problems, Problem{Field: "metadata.name", Message: "must match route package"})
	}

	validateArtifacts(&problems, m.Spec.Artifacts)
	validateDependencies(&problems, m.Spec.Dependencies)

	if len(problems) > 0 {
		return &Error{Problems: problems}
	}
	return nil
}

func requireEqual(problems *[]Problem, field string, got string, want string) {
	if got != want {
		*problems = append(*problems, Problem{Field: field, Message: "must be " + want})
	}
}

func requireNonEmpty(problems *[]Problem, field string, value string) {
	if strings.TrimSpace(value) == "" {
		*problems = append(*problems, Problem{Field: field, Message: "is required"})
	}
}

func requireSlug(problems *[]Problem, field string, value string) {
	if !slugRE.MatchString(value) {
		*problems = append(*problems, Problem{Field: field, Message: "must match ^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$"})
	}
}

func validateArtifacts(problems *[]Problem, artifacts []Artifact) {
	if len(artifacts) == 0 {
		*problems = append(*problems, Problem{Field: "spec.artifacts", Message: "must contain at least one artifact"})
		return
	}

	seen := map[string]struct{}{}
	for i, artifact := range artifacts {
		field := fmt.Sprintf("spec.artifacts[%d]", i)
		validatePath(problems, field+".path", artifact.Path)
		if _, ok := seen[artifact.Path]; ok {
			*problems = append(*problems, Problem{Field: field + ".path", Message: "must be unique"})
		}
		seen[artifact.Path] = struct{}{}

		if _, ok := knownArtifactTypes[artifact.Type]; !ok {
			*problems = append(*problems, Problem{Field: field + ".type", Message: "is unknown"})
		}
		if artifact.TargetPath != "" {
			validatePath(problems, field+".targetPath", artifact.TargetPath)
		}
	}
}

func validatePath(problems *[]Problem, field string, value string) {
	if value == "" {
		*problems = append(*problems, Problem{Field: field, Message: "is required"})
		return
	}
	if strings.HasPrefix(value, "/") || strings.Contains(value, "\\") {
		*problems = append(*problems, Problem{Field: field, Message: "must be a relative slash-separated path"})
		return
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.Contains(cleaned, "/../") {
		*problems = append(*problems, Problem{Field: field, Message: "must not escape the package root"})
	}
}

func validateDependencies(problems *[]Problem, dependencies []string) {
	for i, dependency := range dependencies {
		parts := strings.Split(dependency, "/")
		if len(parts) != 3 || !slugRE.MatchString(parts[0]) || !slugRE.MatchString(parts[1]) || !slugRE.MatchString(parts[2]) {
			*problems = append(*problems, Problem{Field: fmt.Sprintf("spec.dependencies[%d]", i), Message: "must use full org/namespace/package reference"})
		}
	}
}
