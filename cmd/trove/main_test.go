package main

import "testing"

func TestCommandMode(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want mode
	}{
		{name: "empty args show help", args: nil, want: modeHelp},
		{name: "serve starts server", args: []string{"serve"}, want: modeServer},
		{name: "server starts server", args: []string{"server"}, want: modeServer},
		{name: "client command uses cli", args: []string{"resolve", "nwks/platform/agent-backend@latest"}, want: modeCLI},
		{name: "unknown command uses cli error path", args: []string{"typo"}, want: modeCLI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := commandMode(tt.args); got != tt.want {
				t.Fatalf("commandMode(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
