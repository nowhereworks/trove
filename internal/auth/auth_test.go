package auth

import (
	"net/http"
	"strings"
	"testing"

	"trove/internal/config"
)

func TestHashTokenProducesConsistentHash(t *testing.T) {
	token := "test-token-123"
	hash1 := HashToken(token)
	hash2 := HashToken(token)

	if hash1 != hash2 {
		t.Fatalf("hash1 = %q, hash2 = %q, want equal", hash1, hash2)
	}
	if len(hash1) != 64 {
		t.Fatalf("hash length = %d, want 64", len(hash1))
	}
}

func TestHashTokenDiffersForDifferentTokens(t *testing.T) {
	hash1 := HashToken("token-a")
	hash2 := HashToken("token-b")

	if hash1 == hash2 {
		t.Fatal("different tokens produced same hash")
	}
}

func TestHasScope(t *testing.T) {
	tests := []struct {
		scopes   []string
		required string
		want     bool
	}{
		{[]string{"package:read"}, "package:read", true},
		{[]string{"package:read"}, "package:write", false},
		{[]string{"*:*"}, "package:read", true},
		{[]string{"*:*"}, "anything:here", true},
		{[]string{"package:read", "package:write"}, "package:write", true},
		{[]string{}, "package:read", false},
	}

	for _, tt := range tests {
		got := hasScope(tt.scopes, tt.required)
		if got != tt.want {
			t.Errorf("hasScope(%v, %q) = %v, want %v", tt.scopes, tt.required, got, tt.want)
		}
	}
}

func TestCheckVisibility(t *testing.T) {
	authUser := User{IsAuthenticated: true}
	unauthUser := User{IsAuthenticated: false}

	tests := []struct {
		effective    string
		user         User
		isPublicRead bool
		want         bool
	}{
		{"public", authUser, true, true},
		{"private", authUser, true, true},
		{"public", unauthUser, true, true},
		{"public", unauthUser, false, false},
		{"private", unauthUser, true, false},
		{"private", unauthUser, false, false},
		{"internal", unauthUser, true, false},
	}

	for _, tt := range tests {
		got := CheckVisibility(tt.effective, tt.user, tt.isPublicRead)
		if got != tt.want {
			t.Errorf("CheckVisibility(%q, %+v, %v) = %v, want %v",
				tt.effective, tt.user, tt.isPublicRead, got, tt.want)
		}
	}
}

func TestAuthenticatorDevMode(t *testing.T) {
	cfg := devConfig()
	auth, err := NewAuthenticator(cfg, nil)
	if err != nil {
		t.Fatalf("NewAuthenticator error: %v", err)
	}

	req := httptestRequest("Bearer dev-token-local-only")
	user := auth.authenticateRequest(req)

	if !user.IsAuthenticated {
		t.Fatal("dev mode user should be authenticated")
	}
	if !user.IsDev {
		t.Fatal("dev mode user should have IsDev=true")
	}
	if user.ID != DevUserID {
		t.Fatalf("dev user ID = %q, want %s", user.ID, DevUserID)
	}
}

func TestAuthenticatorDevModeRejectsWrongToken(t *testing.T) {
	cfg := devConfig()
	auth, err := NewAuthenticator(cfg, nil)
	if err != nil {
		t.Fatalf("NewAuthenticator error: %v", err)
	}

	req := httptestRequest("Bearer wrong-token")
	user := auth.authenticateRequest(req)

	if user.IsAuthenticated {
		t.Fatal("wrong token should not authenticate")
	}
}

func TestAuthenticatorRejectsInvalidAuthHeader(t *testing.T) {
	cfg := devConfig()
	auth, err := NewAuthenticator(cfg, nil)
	if err != nil {
		t.Fatalf("NewAuthenticator error: %v", err)
	}

	req := httptestRequest("Basic abc123")
	user := auth.authenticateRequest(req)

	if user.IsAuthenticated {
		t.Fatal("Basic auth should not authenticate")
	}
}

func TestAuthenticatorNoAuthHeader(t *testing.T) {
	cfg := devConfig()
	auth, err := NewAuthenticator(cfg, nil)
	if err != nil {
		t.Fatalf("NewAuthenticator error: %v", err)
	}

	req := httptestRequest("")
	user := auth.authenticateRequest(req)

	if user.IsAuthenticated {
		t.Fatal("no auth header should not authenticate")
	}
}

func TestAuthenticatorCanAccessPackageDevMode(t *testing.T) {
	cfg := devConfig()
	auth, err := NewAuthenticator(cfg, nil)
	if err != nil {
		t.Fatalf("NewAuthenticator error: %v", err)
	}

	devUser := User{IsAuthenticated: true, IsDev: true}

	if !auth.CanAccessPackage(devUser, "org", "ns", "pkg", "read") {
		t.Fatal("dev user should have read access")
	}
	if !auth.CanAccessPackage(devUser, "org", "ns", "pkg", "write") {
		t.Fatal("dev user should have write access")
	}
}

func TestAuthenticatorCanAccessPackageUnauthenticated(t *testing.T) {
	cfg := devConfig()
	auth, err := NewAuthenticator(cfg, nil)
	if err != nil {
		t.Fatalf("NewAuthenticator error: %v", err)
	}

	unauthUser := User{IsAuthenticated: false}

	if auth.CanAccessPackage(unauthUser, "org", "ns", "pkg", "read") {
		t.Fatal("unauthenticated user should not have access")
	}
}

func TestOIDCProviderValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.OIDCConfig
		wantErr bool
	}{
		{"missing issuer", config.OIDCConfig{ClientID: "x", ClientSecret: "x", RedirectURL: "x"}, true},
		{"missing client ID", config.OIDCConfig{IssuerURL: "x", ClientSecret: "x", RedirectURL: "x"}, true},
		{"missing client secret", config.OIDCConfig{IssuerURL: "x", ClientID: "x", RedirectURL: "x"}, true},
		{"missing redirect", config.OIDCConfig{IssuerURL: "x", ClientID: "x", ClientSecret: "x"}, true},
		{"valid config", config.OIDCConfig{IssuerURL: "https://auth.example.com", ClientID: "abc", ClientSecret: "secret", RedirectURL: "http://localhost/callback", Scopes: []string{"openid", "email"}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewOIDCProvider(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOIDCProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOIDCProviderAuthURL(t *testing.T) {
	cfg := config.OIDCConfig{
		IssuerURL:    "https://auth.example.com",
		ClientID:     "test-client",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost/callback",
		Scopes:       []string{"openid", "email"},
	}
	provider, err := NewOIDCProvider(cfg)
	if err != nil {
		t.Fatalf("NewOIDCProvider error: %v", err)
	}

	authURL := provider.AuthURL()

	if !strings.HasPrefix(authURL, "https://auth.example.com/authorize?") {
		t.Fatalf("auth URL = %q, want prefix https://auth.example.com/authorize?", authURL)
	}
	if !strings.Contains(authURL, "client_id=test-client") {
		t.Fatal("auth URL missing client_id")
	}
	if !strings.Contains(authURL, "state=") {
		t.Fatal("auth URL missing state")
	}
	if !strings.Contains(authURL, "nonce=") {
		t.Fatal("auth URL missing nonce")
	}
}

func devConfig() config.Config {
	return config.Config{
		Auth: config.AuthConfig{
			Mode:           "dev",
			DevModeEnabled: true,
			DevToken:       "dev-token-local-only",
		},
		Raw: config.RawConfig{
			RequireAuthByDefault:  true,
			AllowPublicNamespaces: true,
			AllowPublicPackages:   true,
		},
	}
}

func httptestRequest(authHeader string) *http.Request {
	req, _ := http.NewRequest("GET", "/healthz", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	return req
}
