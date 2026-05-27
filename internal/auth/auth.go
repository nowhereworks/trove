package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"trove/internal/config"
	"trove/internal/db/sqlc"
)

type contextKey struct{}

const DevUserID = "0198f006-0000-7000-8000-00000000de00"
const SessionCookieName = "trove_session"

type User struct {
	ID                string
	Email             string
	DisplayName       string
	IsAuthenticated   bool
	IsDev             bool
	TokenScopes       []string
	TokenResourceID   string
	TokenResourceType string
}

type AuthResult struct {
	User       User
	Allowed    bool
	StatusCode int
	Code       string
	Message    string
}

func UserFromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(contextKey{}).(User)
	return u, ok
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFromContext(r.Context())
		if !ok {
			next(w, r)
			return
		}
		if !user.IsAuthenticated && !user.IsDev {
			writeAuthError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
			return
		}
		next(w, r)
	}
}

func RequireScope(scope string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok {
				next(w, r)
				return
			}
			if !user.IsAuthenticated && !user.IsDev {
				writeAuthError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
				return
			}
			if !hasScope(user.TokenScopes, scope) && !user.IsDev {
				writeAuthError(w, r, http.StatusForbidden, "INSUFFICIENT_SCOPE", "Token does not have required scope: "+scope)
				return
			}
			next(w, r)
		}
	}
}

func hasScope(scopes []string, required string) bool {
	for _, s := range scopes {
		if s == required || s == "*:*" {
			return true
		}
	}
	return false
}

func writeAuthError(w http.ResponseWriter, r *http.Request, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	reqID := r.Header.Get("X-Request-Id")
	_, _ = w.Write([]byte(`{"error":{"code":"` + code + `","message":"` + message + `","requestId":"` + reqID + `"}}`))
}

type Authenticator struct {
	cfg      config.Config
	queries  *sqlc.Queries
	oidc     *OIDCProvider
	devToken string
}

func NewAuthenticator(cfg config.Config, queries *sqlc.Queries) (*Authenticator, error) {
	var oidc *OIDCProvider
	if cfg.Auth.Mode == "oidc" {
		var err error
		oidc, err = NewOIDCProvider(cfg.OIDC)
		if err != nil {
			return nil, err
		}
	}

	devToken := ""
	if cfg.Auth.DevModeEnabled && cfg.Auth.Mode == "dev" {
		devToken = cfg.Auth.DevToken
		if devToken == "" {
			devToken = "dev-token-local-only"
		}
	}

	return &Authenticator{
		cfg:      cfg,
		queries:  queries,
		oidc:     oidc,
		devToken: devToken,
	}, nil
}

func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := a.authenticateRequest(r)
		ctx := context.WithValue(r.Context(), contextKey{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Authenticator) authenticateRequest(r *http.Request) User {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if a.cfg.Auth.Mode == "dev" && a.devToken != "" {
			if token == a.devToken {
				return a.devUser(r.Context())
			}
			return User{IsAuthenticated: false}
		}
		if a.queries != nil {
			return a.authenticateToken(r.Context(), token)
		}
	}

	if a.queries != nil {
		cookie, err := r.Cookie(SessionCookieName)
		if err == nil && cookie.Value != "" {
			return a.authenticateToken(r.Context(), cookie.Value)
		}
	}

	return User{IsAuthenticated: false}
}

func (a *Authenticator) devUser(ctx context.Context) User {
	if a.queries != nil {
		id := mustParsePgUUID(DevUserID)
		if _, err := a.queries.GetUserByID(ctx, id); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				_, _ = a.queries.CreateUser(ctx, sqlc.CreateUserParams{
					ID:          id,
					Email:       "dev@local.trove",
					DisplayName: "Dev User",
				})
			}
		}
	}

	return User{
		ID:              DevUserID,
		Email:           "dev@local.trove",
		DisplayName:     "Dev User",
		IsAuthenticated: true,
		IsDev:           true,
		TokenScopes:     []string{"*:*"},
	}
}

func (a *Authenticator) authenticateToken(ctx context.Context, token string) User {
	hash := hashToken(token)
	apiToken, err := a.queries.GetAPITokenByHash(ctx, hash)
	if err != nil {
		return User{IsAuthenticated: false}
	}

	if apiToken.RevokedAt.Valid && !apiToken.RevokedAt.Time.IsZero() {
		return User{IsAuthenticated: false}
	}
	if apiToken.ExpiresAt.Valid && !apiToken.ExpiresAt.Time.IsZero() && apiToken.ExpiresAt.Time.Before(time.Now()) {
		return User{IsAuthenticated: false}
	}

	_ = a.queries.UpdateTokenLastUsed(ctx, apiToken.ID)

	userID := ""
	email := ""
	displayName := ""
	if apiToken.ActorUserID.Valid {
		userID = uuid.UUID(apiToken.ActorUserID.Bytes).String()
		if user, err := a.queries.GetUserByID(ctx, apiToken.ActorUserID); err == nil {
			email = user.Email
			displayName = user.DisplayName
		}
	}

	resourceID := ""
	resourceType := ""
	if apiToken.PackageID.Valid {
		resourceID = uuid.UUID(apiToken.PackageID.Bytes).String()
		resourceType = "package"
	} else if apiToken.NamespaceID.Valid {
		resourceID = uuid.UUID(apiToken.NamespaceID.Bytes).String()
		resourceType = "namespace"
	} else if apiToken.OrgID.Valid {
		resourceID = uuid.UUID(apiToken.OrgID.Bytes).String()
		resourceType = "org"
	}

	return User{
		ID:                userID,
		Email:             email,
		DisplayName:       displayName,
		IsAuthenticated:   true,
		TokenScopes:       apiToken.Scopes,
		TokenResourceID:   resourceID,
		TokenResourceType: resourceType,
	}
}

func (a *Authenticator) CanAccessPackage(user User, org, namespace, pkg string, action string) bool {
	if !user.IsAuthenticated {
		return false
	}
	if user.IsDev {
		return true
	}
	if hasScope(user.TokenScopes, "*:*") {
		return true
	}

	if user.TokenResourceType == "org" && action == "read" {
		return true
	}
	if user.TokenResourceType == "namespace" && action == "read" {
		return true
	}
	if user.TokenResourceType == "package" && action == "read" {
		return true
	}

	requiredScope := "package:read"
	if action == "write" {
		requiredScope = "package:write"
	}
	if action == "publish" {
		requiredScope = "version:publish"
	}
	return hasScope(user.TokenScopes, requiredScope)
}

type MaintainerCheckFunc func(ctx context.Context, userID string) (bool, error)

func RequireMaintainer(resolveAndCheck MaintainerCheckFunc) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok || (!user.IsAuthenticated && !user.IsDev) {
				writeAuthError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication is required.")
				return
			}
			if user.IsDev {
				next(w, r)
				return
			}
			if hasScope(user.TokenScopes, "*:*") {
				next(w, r)
				return
			}
			ok, err := resolveAndCheck(r.Context(), user.ID)
			if err != nil {
				writeAuthError(w, r, http.StatusInternalServerError, "MAINTAINER_CHECK_FAILED", "Could not verify maintainer status.")
				return
			}
			if !ok {
				writeAuthError(w, r, http.StatusForbidden, "NOT_MAINTAINER", "You must be a package maintainer to perform this action.")
				return
			}
			next(w, r)
		}
	}
}

func (a *Authenticator) IsPublicRead(org, namespace, pkg string) bool {
	if !a.cfg.Raw.AllowPublicPackages {
		return false
	}
	if !a.cfg.Raw.AllowPublicNamespaces {
		return false
	}
	return true
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func HashToken(token string) string {
	return hashToken(token)
}

func CheckVisibility(effective string, user User, isPublicRead bool) bool {
	if user.IsAuthenticated {
		return true
	}
	if effective == "public" && isPublicRead {
		return true
	}
	return false
}

func (a *Authenticator) GenerateToken(rawToken string) (string, error) {
	return HashToken(rawToken), nil
}

func (a *Authenticator) RevokeAPIToken(ctx context.Context, tokenID string) error {
	id, err := uuid.Parse(tokenID)
	if err != nil {
		return err
	}
	return a.queries.RevokeAPIToken(ctx, pgtype.UUID{Bytes: id, Valid: true})
}

func (a *Authenticator) OIDCAuthURL() (string, string, error) {
	if a.oidc == nil {
		return "", "", fmt.Errorf("OIDC provider is not configured")
	}
	state := OidcRandomState()
	nonce := OidcRandomState()

	return a.oidc.AuthURLWithState(state, nonce), state, nil
}

func (a *Authenticator) OIDCCallback(ctx context.Context, code string) (User, error) {
	if a.oidc == nil {
		return User{}, fmt.Errorf("OIDC provider is not configured")
	}

	tokenResp, err := a.oidc.exchangeCode(code)
	if err != nil {
		return User{}, fmt.Errorf("token exchange failed: %w", err)
	}

	userInfo, err := a.oidc.getUserInfo(tokenResp.AccessToken)
	if err != nil {
		return User{}, fmt.Errorf("user info fetch failed: %w", err)
	}

	user, err := a.queries.GetUserByOIDC(ctx, sqlc.GetUserByOIDCParams{
		OidcIssuer:  pgtype.Text{String: a.oidc.cfg.IssuerURL, Valid: true},
		OidcSubject: pgtype.Text{String: userInfo.Subject, Valid: true},
	})
	if err != nil {
		id := uuid.New()
		user, err = a.queries.CreateUser(ctx, sqlc.CreateUserParams{
			ID:          pgtype.UUID{Bytes: id, Valid: true},
			Email:       userInfo.Email,
			DisplayName: userInfo.Name,
			OidcIssuer:  pgtype.Text{String: a.oidc.cfg.IssuerURL, Valid: true},
			OidcSubject: pgtype.Text{String: userInfo.Subject, Valid: true},
		})
		if err != nil {
			return User{}, fmt.Errorf("failed to create user: %w", err)
		}
	}

	return User{
		ID:              uuid.UUID(user.ID.Bytes).String(),
		Email:           user.Email,
		DisplayName:     user.DisplayName,
		IsAuthenticated: true,
	}, nil
}

func (a *Authenticator) CreateAPIToken(ctx context.Context, req CreateTokenRequest) (sqlc.ApiToken, string, error) {
	rawToken := uuid.New().String()
	hash := HashToken(rawToken)

	id, _ := uuid.NewV7()
	var actorUserID pgtype.UUID
	var actorServiceAccount pgtype.Text

	if req.ActorUserID != "" {
		uid, err := uuid.Parse(req.ActorUserID)
		if err != nil {
			return sqlc.ApiToken{}, "", err
		}
		actorUserID = pgtype.UUID{Bytes: uid, Valid: true}
	} else {
		actorServiceAccount = pgtype.Text{String: req.ActorServiceAccount, Valid: true}
	}

	var orgID, namespaceID, packageID pgtype.UUID
	if req.OrgID != "" {
		uid, _ := uuid.Parse(req.OrgID)
		orgID = pgtype.UUID{Bytes: uid, Valid: true}
	}
	if req.NamespaceID != "" {
		uid, _ := uuid.Parse(req.NamespaceID)
		namespaceID = pgtype.UUID{Bytes: uid, Valid: true}
	}
	if req.PackageID != "" {
		uid, _ := uuid.Parse(req.PackageID)
		packageID = pgtype.UUID{Bytes: uid, Valid: true}
	}

	var expiresAt pgtype.Timestamptz
	if !req.ExpiresAt.IsZero() {
		expiresAt = pgtype.Timestamptz{Time: req.ExpiresAt, Valid: true}
	}

	token, err := a.queries.CreateAPIToken(ctx, sqlc.CreateAPITokenParams{
		ID:                  pgtype.UUID{Bytes: id, Valid: true},
		TokenHash:           hash,
		DisplayName:         req.DisplayName,
		ActorUserID:         actorUserID,
		ActorServiceAccount: actorServiceAccount,
		Scopes:              req.Scopes,
		OrgID:               orgID,
		NamespaceID:         namespaceID,
		PackageID:           packageID,
		ExpiresAt:           expiresAt,
	})
	if err != nil {
		return sqlc.ApiToken{}, "", err
	}

	return token, rawToken, nil
}

type CreateTokenRequest struct {
	DisplayName         string
	ActorUserID         string
	ActorServiceAccount string
	Scopes              []string
	OrgID               string
	NamespaceID         string
	PackageID           string
	ExpiresAt           time.Time
}

func mustParseUUID(s string) [16]byte {
	uid, _ := uuid.Parse(s)
	return uid
}

func mustParsePgUUID(s string) pgtype.UUID {
	uid, _ := uuid.Parse(s)
	return pgtype.UUID{Bytes: uid, Valid: true}
}

const LocalUserEmail = "local@trove.internal"

func (a *Authenticator) AuthMode() string {
	return a.cfg.Auth.Mode
}

func (a *Authenticator) CookieSecure() bool {
	return a.cfg.Auth.CookieSecure
}

func (a *Authenticator) DevModeEnabled() bool {
	return a.cfg.Auth.DevModeEnabled && a.cfg.Auth.Mode == "dev"
}

func (a *Authenticator) DevToken() string {
	return a.devToken
}

func (a *Authenticator) SetSessionCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cfg.Auth.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})
}

func (a *Authenticator) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cfg.Auth.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (a *Authenticator) GetOrCreateLocalUser(ctx context.Context) (User, sqlc.ApiToken, string, error) {
	var emptyToken sqlc.ApiToken
	user, err := a.queries.GetUserByEmail(ctx, LocalUserEmail)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return User{}, emptyToken, "", fmt.Errorf("lookup local user: %w", err)
		}
		id := uuid.New()
		_, err = a.queries.CreateUser(ctx, sqlc.CreateUserParams{
			ID:          pgtype.UUID{Bytes: id, Valid: true},
			Email:       LocalUserEmail,
			DisplayName: "Local User",
		})
		if err != nil {
			return User{}, emptyToken, "", fmt.Errorf("create local user: %w", err)
		}
		user, err = a.queries.GetUserByEmail(ctx, LocalUserEmail)
		if err != nil {
			return User{}, emptyToken, "", fmt.Errorf("lookup created local user: %w", err)
		}
	}

	tokens, err := a.queries.ListAPITokensByUser(ctx, user.ID)
	if err != nil {
		return User{}, emptyToken, "", fmt.Errorf("list local user tokens: %w", err)
	}
	for _, t := range tokens {
		if !t.RevokedAt.Valid || t.RevokedAt.Time.IsZero() {
			_ = a.queries.RevokeAPIToken(ctx, t.ID)
		}
	}

	token, rawToken, err := a.CreateAPIToken(ctx, CreateTokenRequest{
		DisplayName: "Local Session",
		ActorUserID: uuid.UUID(user.ID.Bytes).String(),
		Scopes:      []string{"*:*"},
	})
	if err != nil {
		return User{}, emptyToken, "", fmt.Errorf("create local session token: %w", err)
	}
	return User{
		ID:              uuid.UUID(user.ID.Bytes).String(),
		Email:           user.Email,
		DisplayName:     user.DisplayName,
		IsAuthenticated: true,
	}, token, rawToken, nil
}
