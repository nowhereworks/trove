package cli

import (
	"testing"
)

func TestParsePackageRef(t *testing.T) {
	tests := []struct {
		input    string
		want     PackageRef
		wantErr  bool
	}{
		{
			input: "companyx/platform/agent-backend@stable",
			want: PackageRef{
				Org:       "companyx",
				Namespace: "platform",
				Name:      "agent-backend",
				Selector:  "stable",
			},
		},
		{
			input: "my-org/my-ns/my-pkg@1.2.3",
			want: PackageRef{
				Org:       "my-org",
				Namespace: "my-ns",
				Name:      "my-pkg",
				Selector:  "1.2.3",
			},
		},
		{
			input: "a/b/c@latest",
			want: PackageRef{
				Org:       "a",
				Namespace: "b",
				Name:      "c",
				Selector:  "latest",
			},
		},
		{
			input:   "invalid",
			wantErr: true,
		},
		{
			input:   "org/ns@selector",
			wantErr: true,
		},
		{
			input:   "org/ns/pkg",
			wantErr: true,
		},
		{
			input:   "org/ns/pkg@",
			wantErr: true,
		},
		{
			input:   "ORG/ns/pkg@selector",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParsePackageRef(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParsePackageRef(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParsePackageRef(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ParsePackageRef(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPackageRefString(t *testing.T) {
	ref := PackageRef{
		Org:       "companyx",
		Namespace: "platform",
		Name:      "agent-backend",
		Selector:  "stable",
	}
	want := "companyx/platform/agent-backend@stable"
	if got := ref.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestPackageRefPackagePath(t *testing.T) {
	ref := PackageRef{
		Org:       "companyx",
		Namespace: "platform",
		Name:      "agent-backend",
		Selector:  "stable",
	}
	want := "companyx/platform/agent-backend"
	if got := ref.PackagePath(); got != want {
		t.Errorf("PackagePath() = %q, want %q", got, want)
	}
}
