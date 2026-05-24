package cli

import (
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
		{[]string{"fetch"}, true},
		{[]string{"install"}, true},
		{[]string{"check"}, true},
		{[]string{"update"}, true},
		{[]string{"help"}, true},
		{[]string{"--help"}, true},
		{[]string{"-h"}, true},
		{[]string{"version"}, true},
		{[]string{"--version"}, true},
		{[]string{"-v"}, true},
		{[]string{}, false},
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
