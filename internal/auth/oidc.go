package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"trove/internal/config"
)

type OIDCProvider struct {
	cfg                   config.OIDCConfig
	httpClient            *http.Client
	authorizationEndpoint string
	tokenEndpoint         string
	userinfoEndpoint      string
	stateStore            sync.Map
	nonceStore            sync.Map
}

type oidcDiscoveryDocument struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
}

type OIDCTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
}

type OIDCUserInfo struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func NewOIDCProvider(cfg config.OIDCConfig) (*OIDCProvider, error) {
	if cfg.IssuerURL == "" {
		return nil, fmt.Errorf("OIDC issuer URL is required")
	}
	if cfg.ClientID == "" {
		return nil, fmt.Errorf("OIDC client ID is required")
	}
	if cfg.ClientSecret == "" {
		return nil, fmt.Errorf("OIDC client secret is required")
	}
	if cfg.RedirectURL == "" {
		return nil, fmt.Errorf("OIDC redirect URL is required")
	}
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"openid", "profile", "email"}
	}

	provider := &OIDCProvider{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	if err := provider.discoverEndpoints(); err != nil {
		return nil, err
	}
	return provider, nil
}

func (p *OIDCProvider) discoverEndpoints() error {
	resp, err := p.httpClient.Get(strings.TrimRight(p.cfg.IssuerURL, "/") + "/.well-known/openid-configuration")
	if err != nil {
		return fmt.Errorf("OIDC discovery failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OIDC discovery returned %d: %s", resp.StatusCode, string(body))
	}

	var doc oidcDiscoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("OIDC discovery decode failed: %w", err)
	}
	if doc.AuthorizationEndpoint == "" {
		return fmt.Errorf("OIDC discovery missing authorization_endpoint")
	}
	if doc.TokenEndpoint == "" {
		return fmt.Errorf("OIDC discovery missing token_endpoint")
	}
	if doc.UserinfoEndpoint == "" {
		return fmt.Errorf("OIDC discovery missing userinfo_endpoint")
	}

	p.authorizationEndpoint = doc.AuthorizationEndpoint
	p.tokenEndpoint = doc.TokenEndpoint
	p.userinfoEndpoint = doc.UserinfoEndpoint
	return nil
}

func (p *OIDCProvider) AuthURL() string {
	state := randomState()
	nonce := randomState()

	p.stateStore.Store(state, time.Now().Add(10*time.Minute))
	p.nonceStore.Store(nonce, time.Now().Add(10*time.Minute))

	return p.authURL(state, nonce)
}

func (p *OIDCProvider) AuthURLWithState(state, nonce string) string {
	p.stateStore.Store(state, time.Now().Add(10*time.Minute))
	p.nonceStore.Store(nonce, time.Now().Add(10*time.Minute))
	return p.authURL(state, nonce)
}

func (p *OIDCProvider) authURL(state, nonce string) string {
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", p.cfg.ClientID)
	params.Set("redirect_uri", p.cfg.RedirectURL)
	params.Set("scope", strings.Join(p.cfg.Scopes, " "))
	params.Set("state", state)
	params.Set("nonce", nonce)

	return p.authorizationEndpoint + "?" + params.Encode()
}

func (p *OIDCProvider) HandleCallback(code, state string) (*OIDCUserInfo, error) {
	_, ok := p.stateStore.LoadAndDelete(state)
	if !ok {
		return nil, fmt.Errorf("invalid or expired state")
	}

	tokenResp, err := p.exchangeCode(code)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	userInfo, err := p.getUserInfo(tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("user info fetch failed: %w", err)
	}

	return userInfo, nil
}

func (p *OIDCProvider) exchangeCode(code string) (*OIDCTokenResponse, error) {
	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("redirect_uri", p.cfg.RedirectURL)
	params.Set("client_id", p.cfg.ClientID)
	params.Set("client_secret", p.cfg.ClientSecret)

	resp, err := p.httpClient.PostForm(p.tokenEndpoint, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp OIDCTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func (p *OIDCProvider) getUserInfo(accessToken string) (*OIDCUserInfo, error) {
	req, err := http.NewRequest("GET", p.userinfoEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo endpoint returned %d", resp.StatusCode)
	}

	var userInfo OIDCUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

func randomState() string {
	return OidcRandomState()
}

func OidcRandomState() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func CleanupExpiredStates(store *sync.Map) {
	store.Range(func(key, value any) bool {
		if expiry, ok := value.(time.Time); ok && time.Now().After(expiry) {
			store.Delete(key)
		}
		return true
	})
}
