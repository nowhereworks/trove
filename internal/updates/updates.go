package updates

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"trove/internal/manifest"
	"trove/internal/packages"
)

type Target struct {
	Tool          string `json:"tool"`
	ToolVersion   string `json:"toolVersion"`
	Runtime       string `json:"runtime"`
	ModelFamily   string `json:"modelFamily"`
	ContextWindow int    `json:"contextWindow"`
}

type UpdateCheckRequest struct {
	Package             string `json:"package"`
	CurrentVersion      string `json:"currentVersion"`
	CurrentDigest       string `json:"currentDigest"`
	Channel             string `json:"channel"`
	StrictCompatibility bool   `json:"strictCompatibility"`
	Target              Target `json:"target"`
}

type UpdateCheckResponse struct {
	UpdateAvailable      bool   `json:"updateAvailable"`
	LatestVersion        string `json:"latestVersion"`
	LatestDigest         string `json:"latestDigest"`
	Compatibility        string `json:"compatibility"`
	RequiresManualApproval bool `json:"requiresManualApproval"`
	ChangelogURL         string `json:"changelogUrl"`
}

type CompatibilityCheckRequest struct {
	Package             string `json:"package"`
	Version             string `json:"version"`
	StrictCompatibility bool   `json:"strictCompatibility"`
	Target              Target `json:"target"`
}

type CompatibilityCheckResponse struct {
	Compatibility string `json:"compatibility"`
	Details       []CompatibilityDetail `json:"details,omitempty"`
}

type CompatibilityDetail struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Required   string `json:"required,omitempty"`
	Provided   string `json:"provided,omitempty"`
	Compatible bool   `json:"compatible"`
}

type Service struct {
	store packages.Store
}

func NewService(store packages.Store) *Service {
	return &Service{store: store}
}

func (s *Service) CheckUpdate(ctx context.Context, req UpdateCheckRequest) (UpdateCheckResponse, error) {
	parts := strings.SplitN(req.Package, "/", 3)
	if len(parts) != 3 {
		return UpdateCheckResponse{}, fmt.Errorf("package must use org/namespace/package format")
	}
	org, namespace, name := parts[0], parts[1], parts[2]

	channel := req.Channel
	if channel == "" {
		channel = "stable"
	}

	resolved, err := s.store.Resolve(ctx, org, namespace, name, channel)
	if err != nil {
		return UpdateCheckResponse{}, err
	}

	updateAvailable := resolved.ResolvedVersion != req.CurrentVersion

	compatibility := "unknown"
	requiresManualApproval := false

	if updateAvailable {
		compatResult, err := s.checkCompatibilityForVersion(ctx, org, namespace, name, resolved.ResolvedVersion, req.Target, req.StrictCompatibility)
		if err == nil {
			compatibility = compatResult.Compatibility
		}

		manifestResult, merr := s.store.GetManifest(ctx, org, namespace, name, resolved.ResolvedVersion)
		if merr == nil {
			var m manifest.Manifest
			if jerr := json.Unmarshal(manifestResult.Manifest, &m); jerr == nil {
				if policy, ok := m.Spec.UpdatePolicy["breakingChangeRequiresManualApproval"]; ok {
					if b, ok := policy.(bool); ok && b {
						currentMajor, _, _, cErr := packages.ParseStrictSemver(req.CurrentVersion)
						latestMajor, _, _, lErr := packages.ParseStrictSemver(resolved.ResolvedVersion)
						if cErr == nil && lErr == nil && latestMajor > currentMajor {
							requiresManualApproval = true
						}
					}
				}
			}
		}
	} else {
		compatResult, err := s.checkCompatibilityForVersion(ctx, org, namespace, name, req.CurrentVersion, req.Target, req.StrictCompatibility)
		if err == nil {
			compatibility = compatResult.Compatibility
		}
	}

	changelogURL := "/api/v1/packages/" + org + "/" + namespace + "/" + name + "/compare/" + req.CurrentVersion + "..." + resolved.ResolvedVersion

	return UpdateCheckResponse{
		UpdateAvailable:        updateAvailable,
		LatestVersion:          resolved.ResolvedVersion,
		LatestDigest:           resolved.Digest,
		Compatibility:          compatibility,
		RequiresManualApproval: requiresManualApproval,
		ChangelogURL:           changelogURL,
	}, nil
}

func (s *Service) CheckCompatibility(ctx context.Context, req CompatibilityCheckRequest) (CompatibilityCheckResponse, error) {
	parts := strings.SplitN(req.Package, "/", 3)
	if len(parts) != 3 {
		return CompatibilityCheckResponse{}, fmt.Errorf("package must use org/namespace/package format")
	}
	org, namespace, name := parts[0], parts[1], parts[2]

	return s.checkCompatibilityForVersion(ctx, org, namespace, name, req.Version, req.Target, req.StrictCompatibility)
}

func (s *Service) checkCompatibilityForVersion(ctx context.Context, org, namespace, name, version string, target Target, strict bool) (CompatibilityCheckResponse, error) {
	manifestResult, err := s.store.GetManifest(ctx, org, namespace, name, version)
	if err != nil {
		return CompatibilityCheckResponse{}, err
	}

	var m manifest.Manifest
	if err := json.Unmarshal(manifestResult.Manifest, &m); err != nil {
		return CompatibilityCheckResponse{Compatibility: "unknown"}, nil
	}

	compat := m.Spec.Compatibility
	hasAnyCompat := len(compat.Tools) > 0 || len(compat.Runtimes) > 0 || len(compat.Models) > 0

	if !hasAnyCompat {
		return CompatibilityCheckResponse{Compatibility: "unknown"}, nil
	}

	response := CompatibilityCheckResponse{Compatibility: "compatible"}
	overallCompatible := true

	for _, tool := range compat.Tools {
		detail := CompatibilityDetail{Kind: "tool", Name: tool.Name, Compatible: true}

		if target.Tool != "" && !strings.EqualFold(target.Tool, tool.Name) {
			continue
		}

		if target.ToolVersion != "" {
			constraint := tool.Version
			if constraint == "" {
				constraint = tool.MinVersion
			}
			if constraint != "" {
				satisfied, serr := satisfiesSemVer(target.ToolVersion, constraint)
				if serr != nil {
					detail.Compatible = false
					overallCompatible = false
				} else if !satisfied {
					detail.Compatible = false
					overallCompatible = false
				}
			}
		}

		detail.Required = tool.Version
		if detail.Required == "" {
			detail.Required = tool.MinVersion
		}
		detail.Provided = target.ToolVersion
		response.Details = append(response.Details, detail)
	}

	for _, runtime := range compat.Runtimes {
		detail := CompatibilityDetail{Kind: "runtime", Name: runtime, Required: runtime, Provided: target.Runtime, Compatible: true}

		if target.Runtime != "" && !strings.EqualFold(target.Runtime, runtime) {
			detail.Compatible = false
			overallCompatible = false
		}

		response.Details = append(response.Details, detail)
	}

	for _, model := range compat.Models {
		detail := CompatibilityDetail{Kind: "model", Name: model.Family, Compatible: true}

		if target.ModelFamily != "" && !strings.EqualFold(target.ModelFamily, model.Family) {
			continue
		}

		if target.ModelFamily != "" {
			if model.MinContextWindow > 0 && target.ContextWindow > 0 {
				if target.ContextWindow < model.MinContextWindow {
					detail.Compatible = false
					overallCompatible = false
				}
			}
		}

		detail.Required = model.Family
		if model.MinContextWindow > 0 {
			detail.Required += " (min " + strconv.Itoa(model.MinContextWindow) + " context)"
		}
		detail.Provided = target.ModelFamily
		if target.ContextWindow > 0 {
			detail.Provided += " (" + strconv.Itoa(target.ContextWindow) + " context)"
		}
		response.Details = append(response.Details, detail)
	}

	if !overallCompatible {
		response.Compatibility = "incompatible"
	}

	return response, nil
}

var semverRE = regexp.MustCompile(`^([><=!]+)?\s*([0-9]+)\.([0-9]+)\.([0-9]+)$`)

func satisfiesSemVer(version string, constraint string) (bool, error) {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" {
		return true, nil
	}

	if idx := strings.Index(constraint, " - "); idx > 0 && !strings.Contains(constraint[:idx], ",") {
		low := strings.TrimSpace(constraint[:idx])
		high := strings.TrimSpace(constraint[idx+3:])
		gte, err := satisfiesSemVer(version, ">="+low)
		if err != nil {
			return false, err
		}
		lte, err := satisfiesSemVer(version, "<="+high)
		if err != nil {
			return false, err
		}
		return gte && lte, nil
	}

	constraints := splitConstraints(constraint)
	for _, c := range constraints {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		satisfied, err := matchesSingleConstraint(version, c)
		if err != nil {
			return false, err
		}
		if !satisfied {
			return false, nil
		}
	}
	return true, nil
}

func splitConstraints(constraint string) []string {
	var result []string
	var current strings.Builder
	depth := 0
	for _, ch := range constraint {
		switch ch {
		case ' ', ',':
			if depth == 0 && current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		case '(':
			depth++
			current.WriteRune(ch)
		case ')':
			depth--
			current.WriteRune(ch)
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result
}

func matchesSingleConstraint(version string, constraint string) (bool, error) {
	constraint = strings.TrimSpace(constraint)

	if strings.ContainsRune(constraint, 'x') || strings.ContainsRune(constraint, 'X') || strings.ContainsRune(constraint, '*') {
		return matchesWildcardConstraint(version, constraint)
	}

	if strings.HasPrefix(constraint, "^") {
		return matchesCaretConstraint(version, strings.TrimPrefix(constraint, "^"))
	}

	if strings.HasPrefix(constraint, "~") {
		return matchesTildeConstraint(version, strings.TrimPrefix(constraint, "~"))
	}

	return matchesComparatorConstraint(version, constraint)
}

func matchesComparatorConstraint(version string, constraint string) (bool, error) {
	m := semverRE.FindStringSubmatch(constraint)
	if m == nil {
		return false, fmt.Errorf("invalid constraint: %s", constraint)
	}

	op := strings.TrimSpace(m[1])
	if op == "" {
		op = "="
	}

	cMajor, cMinor, cPatch := parseInt(m[2]), parseInt(m[3]), parseInt(m[4])
	vMajor, vMinor, vPatch, err := packages.ParseStrictSemver(version)
	if err != nil {
		return false, err
	}

	switch op {
	case "=":
		return vMajor == cMajor && vMinor == cMinor && vPatch == cPatch, nil
	case "!=":
		return vMajor != cMajor || vMinor != cMinor || vPatch != cPatch, nil
	case ">":
		return compareSemVer(vMajor, vMinor, vPatch, cMajor, cMinor, cPatch) > 0, nil
	case ">=":
		return compareSemVer(vMajor, vMinor, vPatch, cMajor, cMinor, cPatch) >= 0, nil
	case "<":
		return compareSemVer(vMajor, vMinor, vPatch, cMajor, cMinor, cPatch) < 0, nil
	case "<=":
		return compareSemVer(vMajor, vMinor, vPatch, cMajor, cMinor, cPatch) <= 0, nil
	default:
		return false, fmt.Errorf("unknown operator: %s", op)
	}
}

func matchesCaretConstraint(version string, base string) (bool, error) {
	base = strings.TrimSpace(base)
	major, minor, patch, err := packages.ParseStrictSemver(base)
	if err != nil {
		return false, err
	}

	vMajor, vMinor, vPatch, err := packages.ParseStrictSemver(version)
	if err != nil {
		return false, err
	}

	if major != 0 {
		return vMajor == major && compareSemVer(vMajor, vMinor, vPatch, major, minor, patch) >= 0, nil
	}
	if minor != 0 {
		return vMajor == 0 && vMinor == minor && vPatch >= patch, nil
	}
	return vMajor == 0 && vMinor == 0 && vPatch == patch, nil
}

func matchesTildeConstraint(version string, base string) (bool, error) {
	base = strings.TrimSpace(base)
	major, minor, patch, err := packages.ParseStrictSemver(base)
	if err != nil {
		return false, err
	}

	vMajor, vMinor, vPatch, err := packages.ParseStrictSemver(version)
	if err != nil {
		return false, err
	}

	return vMajor == major && vMinor == minor && vPatch >= patch, nil
}

func matchesWildcardConstraint(version string, constraint string) (bool, error) {
	parts := strings.FieldsFunc(constraint, func(r rune) bool { return r == '.' })
	if len(parts) == 0 {
		return false, fmt.Errorf("invalid wildcard constraint: %s", constraint)
	}

	vMajor, vMinor, vPatch, err := packages.ParseStrictSemver(version)
	if err != nil {
		return false, err
	}

	if len(parts) >= 1 && !isWildcard(parts[0]) {
		cMajor := parseInt(parts[0])
		if vMajor != cMajor {
			return false, nil
		}
	}
	if len(parts) >= 2 && !isWildcard(parts[1]) {
		cMinor := parseInt(parts[1])
		if vMinor != cMinor {
			return false, nil
		}
	}
	if len(parts) >= 3 && !isWildcard(parts[2]) {
		cPatch := parseInt(parts[2])
		if vPatch != cPatch {
			return false, nil
		}
	}
	return true, nil
}

func isWildcard(s string) bool {
	return s == "x" || s == "X" || s == "*"
}

func compareSemVer(aMajor, aMinor, aPatch, bMajor, bMinor, bPatch int) int {
	if aMajor != bMajor {
		if aMajor > bMajor {
			return 1
		}
		return -1
	}
	if aMinor != bMinor {
		if aMinor > bMinor {
			return 1
		}
		return -1
	}
	if aPatch != bPatch {
		if aPatch > bPatch {
			return 1
		}
		return -1
	}
	return 0
}

func parseInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}
