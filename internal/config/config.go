package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	DefaultListen                  = ":8080"
	DefaultAuthMode                = "dev"
	DefaultStorageMode             = "postgres"
	DefaultMaxArtifactFileBytes    = 10 * 1024 * 1024
	DefaultMaxUnpackedPackageBytes = 100 * 1024 * 1024
	DefaultMaxArtifactsPerVersion  = 1000
)

// Config is the application configuration loaded at process startup.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	OIDC     OIDCConfig
	Orgs     OrgsConfig
	Packages PackagesConfig
	Storage  StorageConfig
	Raw      RawConfig
	Reviews  ReviewsConfig
	Security SecurityConfig
}

type OrgsConfig struct {
	AllowCreateOrg bool
	DefaultOrg     string
}

type PackagesConfig struct {
	CreatePackageOnPush   bool
	CreateNamespaceOnPush bool
}

type ServerConfig struct {
	Listen    string
	PublicURL string
}

type DatabaseConfig struct {
	URL              string
	MigrateOnStartup bool
}

type AuthConfig struct {
	Mode           string
	DevModeEnabled bool
	DevToken       string
	CookieSecure   bool
}

type OIDCConfig struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

type StorageConfig struct {
	Mode   string
	Limits StorageLimitsConfig
}

type StorageLimitsConfig struct {
	MaxArtifactFileBytes    int64
	MaxUnpackedPackageBytes int64
	MaxArtifactsPerVersion  int
}

type RawConfig struct {
	RequireAuthByDefault  bool
	AllowPublicNamespaces bool
	AllowPublicPackages   bool
}

type ReviewsConfig struct {
	RequireApproval   bool
	MinimumApprovals  int
	AllowSelfApproval bool
}

type SecurityConfig struct {
	SecretScanning            bool
	UnsafeInstructionScanning bool
}

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$`)

// Load reads process environment variables and overlays them onto defaults.
func Load() (Config, error) {
	return LoadEnv(os.LookupEnv)
}

// LoadEnv reads configuration using lookup, which keeps tests independent from
// the process environment.
func LoadEnv(lookup func(string) (string, bool)) (Config, error) {
	cfg := Defaults()
	var errs []error
	var cookieSecureRaw string

	assignString(lookup, "TROVE_SERVER_LISTEN", &cfg.Server.Listen)
	assignString(lookup, "TROVE_PUBLIC_URL", &cfg.Server.PublicURL)
	assignString(lookup, "TROVE_DATABASE_URL", &cfg.Database.URL)
	assignString(lookup, "TROVE_AUTH_MODE", &cfg.Auth.Mode)
	assignString(lookup, "TROVE_AUTH_DEV_TOKEN", &cfg.Auth.DevToken)
	assignString(lookup, "TROVE_ORG", &cfg.Orgs.DefaultOrg)
	assignString(lookup, "TROVE_STORAGE_MODE", &cfg.Storage.Mode)
	assignString(lookup, "TROVE_OIDC_ISSUER_URL", &cfg.OIDC.IssuerURL)
	assignString(lookup, "TROVE_OIDC_CLIENT_ID", &cfg.OIDC.ClientID)
	assignString(lookup, "TROVE_OIDC_CLIENT_SECRET", &cfg.OIDC.ClientSecret)
	assignString(lookup, "TROVE_OIDC_REDIRECT_URL", &cfg.OIDC.RedirectURL)
	assignString(lookup, "TROVE_COOKIE_SECURE", &cookieSecureRaw)

	assignBool(lookup, "TROVE_DATABASE_MIGRATE_ON_STARTUP", &cfg.Database.MigrateOnStartup, &errs)
	assignBool(lookup, "TROVE_AUTH_DEV_MODE_ENABLED", &cfg.Auth.DevModeEnabled, &errs)
	assignBool(lookup, "TROVE_ALLOW_CREATE_ORG", &cfg.Orgs.AllowCreateOrg, &errs)
	assignBool(lookup, "TROVE_CREATE_PACKAGE_ON_PUSH", &cfg.Packages.CreatePackageOnPush, &errs)
	assignBool(lookup, "TROVE_CREATE_NAMESPACE_ON_PUSH", &cfg.Packages.CreateNamespaceOnPush, &errs)
	assignBool(lookup, "TROVE_RAW_REQUIRE_AUTH_BY_DEFAULT", &cfg.Raw.RequireAuthByDefault, &errs)
	assignBool(lookup, "TROVE_RAW_ALLOW_PUBLIC_NAMESPACES", &cfg.Raw.AllowPublicNamespaces, &errs)
	assignBool(lookup, "TROVE_RAW_ALLOW_PUBLIC_PACKAGES", &cfg.Raw.AllowPublicPackages, &errs)
	assignBool(lookup, "TROVE_REVIEWS_REQUIRE_APPROVAL", &cfg.Reviews.RequireApproval, &errs)
	assignBool(lookup, "TROVE_REVIEWS_ALLOW_SELF_APPROVAL", &cfg.Reviews.AllowSelfApproval, &errs)
	assignBool(lookup, "TROVE_SECURITY_SECRET_SCANNING", &cfg.Security.SecretScanning, &errs)
	assignBool(lookup, "TROVE_SECURITY_UNSAFE_INSTRUCTION_SCANNING", &cfg.Security.UnsafeInstructionScanning, &errs)

	assignInt64(lookup, "TROVE_STORAGE_MAX_ARTIFACT_FILE_BYTES", &cfg.Storage.Limits.MaxArtifactFileBytes, &errs)
	assignInt64(lookup, "TROVE_STORAGE_MAX_UNPACKED_PACKAGE_BYTES", &cfg.Storage.Limits.MaxUnpackedPackageBytes, &errs)
	assignInt(lookup, "TROVE_STORAGE_MAX_ARTIFACTS_PER_VERSION", &cfg.Storage.Limits.MaxArtifactsPerVersion, &errs)
	assignInt(lookup, "TROVE_REVIEWS_MINIMUM_APPROVALS", &cfg.Reviews.MinimumApprovals, &errs)

	if cookieSecureRaw != "" {
		parsed, err := strconv.ParseBool(cookieSecureRaw)
		if err != nil {
			errs = append(errs, fmt.Errorf("TROVE_COOKIE_SECURE: parse bool: %w", err))
		} else {
			cfg.Auth.CookieSecure = parsed
		}
	}
	if cfg.Server.PublicURL != "" && len(errs) == 0 {
		if strings.HasPrefix(cfg.Server.PublicURL, "https://") {
			cfg.Auth.CookieSecure = true
		}
	}

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}
	if !cfg.Orgs.AllowCreateOrg && cfg.Orgs.DefaultOrg == "" {
		return Config{}, errors.New("TROVE_ORG is required when TROVE_ALLOW_CREATE_ORG=false")
	}
	if cfg.Orgs.DefaultOrg != "" && !slugPattern.MatchString(cfg.Orgs.DefaultOrg) {
		return Config{}, fmt.Errorf("TROVE_ORG must match %s", slugPattern.String())
	}

	return cfg, nil
}

func Defaults() Config {
	return Config{
		Server: ServerConfig{
			Listen: DefaultListen,
		},
		Auth: AuthConfig{
			Mode:           DefaultAuthMode,
			DevModeEnabled: true,
			CookieSecure:   false,
		},
		Orgs: OrgsConfig{
			AllowCreateOrg: true,
		},
		Packages: PackagesConfig{
			CreatePackageOnPush:   true,
			CreateNamespaceOnPush: true,
		},
		Storage: StorageConfig{
			Mode: DefaultStorageMode,
			Limits: StorageLimitsConfig{
				MaxArtifactFileBytes:    DefaultMaxArtifactFileBytes,
				MaxUnpackedPackageBytes: DefaultMaxUnpackedPackageBytes,
				MaxArtifactsPerVersion:  DefaultMaxArtifactsPerVersion,
			},
		},
		Raw: RawConfig{
			RequireAuthByDefault:  true,
			AllowPublicNamespaces: true,
			AllowPublicPackages:   true,
		},
		Reviews: ReviewsConfig{
			RequireApproval:  true,
			MinimumApprovals: 1,
		},
		Security: SecurityConfig{
			SecretScanning:            true,
			UnsafeInstructionScanning: true,
		},
	}
}

func assignString(lookup func(string) (string, bool), key string, dst *string) {
	if value, ok := lookup(key); ok {
		*dst = value
	}
}

func assignBool(lookup func(string) (string, bool), key string, dst *bool, errs *[]error) {
	value, ok := lookup(key)
	if !ok {
		return
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s: parse bool: %w", key, err))
		return
	}
	*dst = parsed
}

func assignInt64(lookup func(string) (string, bool), key string, dst *int64, errs *[]error) {
	value, ok := lookup(key)
	if !ok {
		return
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s: parse int64: %w", key, err))
		return
	}
	*dst = parsed
}

func assignInt(lookup func(string) (string, bool), key string, dst *int, errs *[]error) {
	value, ok := lookup(key)
	if !ok {
		return
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s: parse int: %w", key, err))
		return
	}
	*dst = parsed
}
