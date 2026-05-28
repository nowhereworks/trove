package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
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

// FileConfig mirrors Config with yaml tags for config file loading.
type FileConfig struct {
	Server   *struct {
		Listen    string `yaml:"listen"`
		PublicURL string `yaml:"publicURL"`
	} `yaml:"server"`
	Database *struct {
		URL              string `yaml:"url"`
		MigrateOnStartup *bool  `yaml:"migrateOnStartup"`
	} `yaml:"database"`
	Auth *struct {
		Mode           string `yaml:"mode"`
		DevModeEnabled *bool  `yaml:"devModeEnabled"`
		DevToken       string `yaml:"devToken"`
		CookieSecure   *bool  `yaml:"cookieSecure"`
	} `yaml:"auth"`
	OIDC *struct {
		IssuerURL    string   `yaml:"issuerURL"`
		ClientID     string   `yaml:"clientID"`
		ClientSecret string   `yaml:"clientSecret"`
		RedirectURL  string   `yaml:"redirectURL"`
		Scopes       []string `yaml:"scopes"`
	} `yaml:"oidc"`
	Orgs *struct {
		AllowCreateOrg *bool  `yaml:"allowCreateOrg"`
		DefaultOrg     string `yaml:"defaultOrg"`
	} `yaml:"orgs"`
	Packages *struct {
		CreatePackageOnPush   *bool `yaml:"createPackageOnPush"`
		CreateNamespaceOnPush *bool `yaml:"createNamespaceOnPush"`
	} `yaml:"packages"`
	Storage *struct {
		Mode   string `yaml:"mode"`
		Limits *struct {
			MaxArtifactFileBytes    *int64 `yaml:"maxArtifactFileBytes"`
			MaxUnpackedPackageBytes *int64 `yaml:"maxUnpackedPackageBytes"`
			MaxArtifactsPerVersion  *int   `yaml:"maxArtifactsPerVersion"`
		} `yaml:"limits"`
	} `yaml:"storage"`
	Raw *struct {
		RequireAuthByDefault  *bool `yaml:"requireAuthByDefault"`
		AllowPublicNamespaces *bool `yaml:"allowPublicNamespaces"`
		AllowPublicPackages   *bool `yaml:"allowPublicPackages"`
	} `yaml:"raw"`
	Reviews *struct {
		RequireApproval   *bool `yaml:"requireApproval"`
		MinimumApprovals  *int  `yaml:"minimumApprovals"`
		AllowSelfApproval *bool `yaml:"allowSelfApproval"`
	} `yaml:"reviews"`
	Security *struct {
		SecretScanning            *bool `yaml:"secretScanning"`
		UnsafeInstructionScanning *bool `yaml:"unsafeInstructionScanning"`
	} `yaml:"security"`
}

// LoadFile reads a YAML config file and returns a FileConfig.
func LoadFile(path string) (FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FileConfig{}, fmt.Errorf("read config file %s: %w", path, err)
	}

	var fc FileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return FileConfig{}, fmt.Errorf("parse config file %s: %w", path, err)
	}

	return fc, nil
}

// applyFileConfig overlays file config values onto a Config struct.
// Only non-nil/non-empty values from the file config are applied.
func applyFileConfig(cfg *Config, fc FileConfig) {
	if fc.Server != nil {
		if fc.Server.Listen != "" {
			cfg.Server.Listen = fc.Server.Listen
		}
		if fc.Server.PublicURL != "" {
			cfg.Server.PublicURL = fc.Server.PublicURL
		}
	}
	if fc.Database != nil {
		if fc.Database.URL != "" {
			cfg.Database.URL = fc.Database.URL
		}
		if fc.Database.MigrateOnStartup != nil {
			cfg.Database.MigrateOnStartup = *fc.Database.MigrateOnStartup
		}
	}
	if fc.Auth != nil {
		if fc.Auth.Mode != "" {
			cfg.Auth.Mode = fc.Auth.Mode
		}
		if fc.Auth.DevModeEnabled != nil {
			cfg.Auth.DevModeEnabled = *fc.Auth.DevModeEnabled
		}
		if fc.Auth.DevToken != "" {
			cfg.Auth.DevToken = fc.Auth.DevToken
		}
		if fc.Auth.CookieSecure != nil {
			cfg.Auth.CookieSecure = *fc.Auth.CookieSecure
		}
	}
	if fc.OIDC != nil {
		if fc.OIDC.IssuerURL != "" {
			cfg.OIDC.IssuerURL = fc.OIDC.IssuerURL
		}
		if fc.OIDC.ClientID != "" {
			cfg.OIDC.ClientID = fc.OIDC.ClientID
		}
		if fc.OIDC.ClientSecret != "" {
			cfg.OIDC.ClientSecret = fc.OIDC.ClientSecret
		}
		if fc.OIDC.RedirectURL != "" {
			cfg.OIDC.RedirectURL = fc.OIDC.RedirectURL
		}
		if fc.OIDC.Scopes != nil {
			cfg.OIDC.Scopes = fc.OIDC.Scopes
		}
	}
	if fc.Orgs != nil {
		if fc.Orgs.DefaultOrg != "" {
			cfg.Orgs.DefaultOrg = fc.Orgs.DefaultOrg
		}
		if fc.Orgs.AllowCreateOrg != nil {
			cfg.Orgs.AllowCreateOrg = *fc.Orgs.AllowCreateOrg
		}
	}
	if fc.Packages != nil {
		if fc.Packages.CreatePackageOnPush != nil {
			cfg.Packages.CreatePackageOnPush = *fc.Packages.CreatePackageOnPush
		}
		if fc.Packages.CreateNamespaceOnPush != nil {
			cfg.Packages.CreateNamespaceOnPush = *fc.Packages.CreateNamespaceOnPush
		}
	}
	if fc.Storage != nil {
		if fc.Storage.Mode != "" {
			cfg.Storage.Mode = fc.Storage.Mode
		}
		if fc.Storage.Limits != nil {
			if fc.Storage.Limits.MaxArtifactFileBytes != nil {
				cfg.Storage.Limits.MaxArtifactFileBytes = *fc.Storage.Limits.MaxArtifactFileBytes
			}
			if fc.Storage.Limits.MaxUnpackedPackageBytes != nil {
				cfg.Storage.Limits.MaxUnpackedPackageBytes = *fc.Storage.Limits.MaxUnpackedPackageBytes
			}
			if fc.Storage.Limits.MaxArtifactsPerVersion != nil {
				cfg.Storage.Limits.MaxArtifactsPerVersion = *fc.Storage.Limits.MaxArtifactsPerVersion
			}
		}
	}
	if fc.Raw != nil {
		if fc.Raw.RequireAuthByDefault != nil {
			cfg.Raw.RequireAuthByDefault = *fc.Raw.RequireAuthByDefault
		}
		if fc.Raw.AllowPublicNamespaces != nil {
			cfg.Raw.AllowPublicNamespaces = *fc.Raw.AllowPublicNamespaces
		}
		if fc.Raw.AllowPublicPackages != nil {
			cfg.Raw.AllowPublicPackages = *fc.Raw.AllowPublicPackages
		}
	}
	if fc.Reviews != nil {
		if fc.Reviews.RequireApproval != nil {
			cfg.Reviews.RequireApproval = *fc.Reviews.RequireApproval
		}
		if fc.Reviews.MinimumApprovals != nil {
			cfg.Reviews.MinimumApprovals = *fc.Reviews.MinimumApprovals
		}
		if fc.Reviews.AllowSelfApproval != nil {
			cfg.Reviews.AllowSelfApproval = *fc.Reviews.AllowSelfApproval
		}
	}
	if fc.Security != nil {
		if fc.Security.SecretScanning != nil {
			cfg.Security.SecretScanning = *fc.Security.SecretScanning
		}
		if fc.Security.UnsafeInstructionScanning != nil {
			cfg.Security.UnsafeInstructionScanning = *fc.Security.UnsafeInstructionScanning
		}
	}
}

// Load reads process environment variables and overlays them onto defaults.
func Load() (Config, error) {
	return LoadWithFile("")
}

// LoadWithFile loads config from a YAML file (if path is non-empty), then
// overlays environment variables. Env vars take precedence over file values.
func LoadWithFile(path string) (Config, error) {
	cfg := Defaults()

	if path != "" {
		fc, err := LoadFile(path)
		if err != nil {
			return Config{}, err
		}
		applyFileConfig(&cfg, fc)
	}

	return LoadEnv(os.LookupEnv, cfg)
}

// LoadEnv reads configuration using lookup, which keeps tests independent from
// the process environment. The base config is used as the starting point;
// environment variables overlay on top of it.
func LoadEnv(lookup func(string) (string, bool), base Config) (Config, error) {
	cfg := base
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
