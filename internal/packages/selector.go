package packages

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type SelectorKind int

const (
	SelectorExact SelectorKind = iota
	SelectorChannel
	SelectorMajor
	SelectorMinor
	SelectorDigest
)

type ParsedSelector struct {
	Kind    SelectorKind
	Value   string
	Major   int
	Minor   int
	Version string
}

func ParseStrictSemver(version string) (major int, minor int, patch int, err error) {
	if !strictSemverRE.MatchString(version) {
		return 0, 0, 0, ErrInvalidSelector
	}
	parts := strings.Split(version, ".")
	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, ErrInvalidSelector
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, ErrInvalidSelector
	}
	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, ErrInvalidSelector
	}
	return major, minor, patch, nil
}

var strictSemverRE = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

func SplitPackageSelector(segment string) (string, string, error) {
	idx := strings.LastIndex(segment, "@")
	if idx <= 0 || idx == len(segment)-1 {
		return "", "", ErrInvalidSelector
	}
	return segment[:idx], segment[idx+1:], nil
}

func ParseSelector(selector string) (ParsedSelector, error) {
	selector = strings.TrimPrefix(selector, "@")
	if selector == "" {
		return ParsedSelector{}, ErrInvalidSelector
	}

	if strings.HasPrefix(selector, "sha256:") {
		return ParsedSelector{Kind: SelectorDigest, Value: selector}, nil
	}
	if selector == "latest" {
		return ParsedSelector{Kind: SelectorChannel, Value: selector}, nil
	}

	versionSelector := strings.TrimPrefix(selector, "v")
	if strictSemverRE.MatchString(versionSelector) {
		return ParsedSelector{Kind: SelectorExact, Value: versionSelector, Version: versionSelector}, nil
	}

	parts := strings.Split(versionSelector, ".")
	if len(parts) == 1 {
		major, err := parseNonNegativeInt(parts[0])
		if err != nil {
			return ParsedSelector{}, ErrInvalidSelector
		}
		return ParsedSelector{Kind: SelectorMajor, Value: versionSelector, Major: major}, nil
	}
	if len(parts) == 2 {
		major, err := parseNonNegativeInt(parts[0])
		if err != nil {
			return ParsedSelector{}, ErrInvalidSelector
		}
		minor, err := parseNonNegativeInt(parts[1])
		if err != nil {
			return ParsedSelector{}, ErrInvalidSelector
		}
		return ParsedSelector{Kind: SelectorMinor, Value: versionSelector, Major: major, Minor: minor}, nil
	}

	return ParsedSelector{}, ErrInvalidSelector
}

func parseNonNegativeInt(value string) (int, error) {
	if value == "" {
		return 0, fmt.Errorf("empty integer")
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, ErrInvalidSelector
	}
	return parsed, nil
}
