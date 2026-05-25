package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestHasFlag(t *testing.T) {
	tests := []struct {
		args []string
		flag string
		want bool
	}{
		{[]string{"--json", "--apply"}, "--json", true},
		{[]string{"--json", "--apply"}, "--apply", true},
		{[]string{"--json", "--apply"}, "--output", false},
		{[]string{}, "--json", false},
		{[]string{"pkg@1.0"}, "--json", false},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			if got := hasFlag(tt.args, tt.flag); got != tt.want {
				t.Errorf("hasFlag(%v, %q) = %v, want %v", tt.args, tt.flag, got, tt.want)
			}
		})
	}
}

func TestFlagValue(t *testing.T) {
	tests := []struct {
		args []string
		flag string
		want string
	}{
		{[]string{"--output", "dir", "--json"}, "--output", "dir"},
		{[]string{"--output=dir", "--json"}, "--output", "dir"},
		{[]string{"--json"}, "--output", ""},
		{[]string{}, "--output", ""},
		{[]string{"--output"}, "--output", ""},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			if got := flagValue(tt.args, tt.flag); got != tt.want {
				t.Errorf("flagValue(%v, %q) = %q, want %q", tt.args, tt.flag, got, tt.want)
			}
		})
	}
}

func TestFilterFlags(t *testing.T) {
	tests := []struct {
		args []string
		want []string
	}{
		{[]string{"pkg@1.0", "--json"}, []string{"pkg@1.0"}},
		{[]string{"--json", "pkg@1.0"}, []string{"pkg@1.0"}},
		{[]string{"--output", "dir", "pkg@1.0"}, []string{"pkg@1.0"}},
		{[]string{"pkg@1.0"}, []string{"pkg@1.0"}},
		{[]string{}, []string{}},
		{[]string{"--json", "--apply"}, []string{}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := filterFlags(tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterFlags(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestIsCLICommand(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"resolve"}, true},
		{[]string{"download"}, true},
		{[]string{"fetch"}, false},
		{[]string{"install"}, true},
		{[]string{"check"}, true},
		{[]string{"update"}, true},
		{[]string{"skills"}, true},
		{[]string{"help"}, true},
		{[]string{"--help"}, true},
		{[]string{"-h"}, true},
		{[]string{"version"}, true},
		{[]string{"--version"}, true},
		{[]string{"-v"}, true},
		{[]string{}, false},
		{[]string{"serve"}, false},
		{[]string{"server"}, false},
		{[]string{"unknown"}, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := IsCLICommand(tt.args); got != tt.want {
				t.Errorf("IsCLICommand(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestFetchIsUnknownSubcommand(t *testing.T) {
	err := Run([]string{"fetch", "nwks/platform/agent-backend", "AGENTS.md"})
	if err == nil || !strings.Contains(err.Error(), "unknown subcommand: fetch") {
		t.Fatalf("Run(fetch) error = %v, want unknown subcommand", err)
	}
}

func TestSkillsBogusIsUnknownSubcommand(t *testing.T) {
	err := Run([]string{"skills", "bogus"})
	if err == nil || !strings.Contains(err.Error(), "unknown skills subcommand: bogus") {
		t.Fatalf("Run(skills bogus) error = %v, want unknown skills subcommand", err)
	}
}

func TestSkillsFindPrintsHumanResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search/packages" || r.URL.Query().Get("q") != "react performance" || r.URL.Query().Get("artifactType") != "skill" {
			t.Fatalf("unexpected search request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(SearchPackagesResponse{Items: []PackageSummary{{
			Org:           "nwks",
			Namespace:     "platform",
			Name:          "react-best-practices",
			Description:   "React and Next.js performance optimization guidelines.",
			StableVersion: "1.0.0",
		}}})
	}))
	t.Cleanup(server.Close)
	t.Setenv("TROVE_SERVER_URL", server.URL)

	stdout := captureStdout(t, func() error {
		return Run([]string{"skills", "find", "react", "performance"})
	})
	if !strings.Contains(stdout, "Skills matching \"react performance\":") || !strings.Contains(stdout, "nwks/platform/react-best-practices@stable") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestSkillsFindPrintsJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "react" || r.URL.Query().Get("artifactType") != "skill" {
			t.Fatalf("unexpected search query: %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(SearchPackagesResponse{Items: []PackageSummary{{
			Org:       "nwks",
			Namespace: "platform",
			Name:      "react-best-practices",
		}}})
	}))
	t.Cleanup(server.Close)
	t.Setenv("TROVE_SERVER_URL", server.URL)

	stdout := captureStdout(t, func() error {
		return Run([]string{"skills", "find", "react", "--json"})
	})
	var out SearchPackagesResponse
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("decode JSON stdout %q: %v", stdout, err)
	}
	if len(out.Items) != 1 || out.Items[0].Name != "react-best-practices" {
		t.Fatalf("JSON stdout = %+v", out)
	}
}
