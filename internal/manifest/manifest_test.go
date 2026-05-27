package manifest

import (
	"errors"
	"testing"
)

func TestParseAndValidateValidManifest(t *testing.T) {
	m, err := Parse([]byte(validManifestYAML))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if err := Validate(m, ValidateOptions{Org: "companyx", Namespace: "platform", Package: "agent-backend"}); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateReportsRequiredFields(t *testing.T) {
	err := Validate(Manifest{}, ValidateOptions{})
	problems := validationProblems(t, err)

	for _, field := range []string{
		"apiVersion",
		"kind",
		"metadata.org",
		"metadata.namespace",
		"metadata.name",
		"metadata.displayName",
		"metadata.description",
		"spec.artifacts",
	} {
		assertProblem(t, problems, field)
	}
}

func TestValidateReportsRouteMismatches(t *testing.T) {
	m := mustParse(t, validManifestYAML)
	err := Validate(m, ValidateOptions{Org: "otherorg", Namespace: "otherns", Package: "otherpkg"})
	problems := validationProblems(t, err)

	for _, field := range []string{"metadata.org", "metadata.namespace", "metadata.name"} {
		assertProblem(t, problems, field)
	}
}

func TestValidateRejectsInvalidSlugsAndSemver(t *testing.T) {
	m := mustParse(t, validManifestYAML)
	m.Metadata.Org = "CompanyX"
	m.Metadata.Namespace = "p"
	m.Metadata.Name = "bad_name"

	err := Validate(m, ValidateOptions{})
	problems := validationProblems(t, err)

	for _, field := range []string{"metadata.org", "metadata.namespace", "metadata.name"} {
		assertProblem(t, problems, field)
	}
}

func TestValidateRejectsArtifactPathProblems(t *testing.T) {
	m := mustParse(t, validManifestYAML)
	m.Spec.Artifacts = []Artifact{
		{Path: "/absolute", Type: "agent-instructions"},
		{Path: "../escape", Type: "skill"},
		{Path: "AGENTS.md", Type: "unknown"},
		{Path: "AGENTS.md", Type: "agent-instructions"},
		{Path: "commands/review.md", Type: "command", TargetPath: "../review.md"},
	}

	err := Validate(m, ValidateOptions{})
	problems := validationProblems(t, err)

	for _, field := range []string{
		"spec.artifacts[0].path",
		"spec.artifacts[1].path",
		"spec.artifacts[2].type",
		"spec.artifacts[3].path",
		"spec.artifacts[4].targetPath",
	} {
		assertProblem(t, problems, field)
	}
}

func TestValidateRejectsMalformedCompatibilityAndDependencies(t *testing.T) {
	m := mustParse(t, validManifestYAML)
	m.Spec.Compatibility.Tools = []ToolCompatibility{{Name: "OpenCode"}}
	m.Spec.Compatibility.Models = []ModelCompatibility{{Family: "GPT"}}
	m.Spec.Compatibility.Runtimes = []string{"Linux"}
	m.Spec.Dependencies = []string{"platform/agent-backend"}

	err := Validate(m, ValidateOptions{})
	problems := validationProblems(t, err)

	for _, field := range []string{
		"spec.compatibility.tools[0].name",
		"spec.compatibility.tools[0].version",
		"spec.compatibility.models[0].family",
		"spec.compatibility.runtimes[0]",
		"spec.dependencies[0]",
	} {
		assertProblem(t, problems, field)
	}
}

func mustParse(t *testing.T, data string) Manifest {
	t.Helper()
	m, err := Parse([]byte(data))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	return m
}

func validationProblems(t *testing.T, err error) []Problem {
	t.Helper()
	var validationErr *Error
	if !errors.As(err, &validationErr) {
		t.Fatalf("Validate() error = %v, want *Error", err)
	}
	return validationErr.Problems
}

func assertProblem(t *testing.T, problems []Problem, field string) {
	t.Helper()
	for _, problem := range problems {
		if problem.Field == field {
			return
		}
	}
	t.Fatalf("missing problem for %s in %+v", field, problems)
}

const validManifestYAML = `apiVersion: trove.io/v1
kind: AgentArtifactPackage
metadata:
  org: companyx
  namespace: platform
  name: agent-backend
  displayName: Backend Agent Defaults
  description: Default agent instructions, skills, and commands for backend services.
  labels:
    language: golang
    framework: chi
spec:
  visibility: public
  compatibility:
    tools:
      - name: opencode
        version: ">=0.6.0 <2.0.0"
    models:
      - family: gpt
        minContextWindow: 128000
    runtimes:
      - linux
  artifacts:
    - path: AGENTS.md
      type: agent-instructions
      required: true
      targetPath: AGENTS.md
  dependencies:
    - companyx/platform/base-agent
`
