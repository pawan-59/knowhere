package config

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration, loaded from environment / .env file.
type Config struct {
	Port        string
	AllowOrigin string

	// DBPath is the SQLite database file path. On Kubernetes this points at a
	// PersistentVolume mount (e.g. /data/central.db) so data survives pod
	// rotation.
	DBPath string

	Zoho    ZohoConfig
	Devtron DevtronConfig
	Auth    AuthConfig

	// CacheTTL controls how long upstream (Zoho/Devtron) responses are cached.
	CacheTTL time.Duration
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	// Secret signs session tokens (HMAC-SHA256). Auto-generated if unset, which
	// means tokens are invalidated on restart — set it in production.
	Secret []byte
	// CookieSecure marks the session cookie Secure (HTTPS-only). Enable in prod.
	CookieSecure bool
	// TokenTTL is how long a session lasts before re-login is required.
	TokenTTL time.Duration
	// AdminEmail / AdminPassword seed the initial admin user on startup.
	AdminEmail    string
	AdminPassword string
}

// ZohoConfig holds Zoho Desk OAuth + org details. Data center is India (.in).
type ZohoConfig struct {
	// AccountsBase is the OAuth token endpoint host, e.g. https://accounts.zoho.in
	AccountsBase string
	// APIBase is the Desk API host, e.g. https://desk.zoho.in
	APIBase      string
	ClientID     string
	ClientSecret string
	RefreshToken string
	OrgID        string
}

// DevtronConfig holds Devtron dashboard API access.
type DevtronConfig struct {
	BaseURL  string // e.g. https://devtron.example.com
	APIToken string // Devtron API token (sent as `token` header)
}

// Load reads configuration from the environment. A .env file, if present in the
// working directory, is loaded first (existing env vars take precedence).
func Load() (*Config, error) {
	_ = godotenv.Load() // best-effort; missing .env is fine

	cfg := &Config{
		Port:        getenv("PORT", "8080"),
		AllowOrigin: getenv("ALLOW_ORIGIN", "http://localhost:5173"),
		DBPath:      getenv("DB_PATH", "central.db"),
		Zoho: ZohoConfig{
			AccountsBase: getenv("ZOHO_ACCOUNTS_BASE", "https://accounts.zoho.in"),
			APIBase:      getenv("ZOHO_API_BASE", "https://desk.zoho.in"),
			ClientID:     os.Getenv("ZOHO_CLIENT_ID"),
			ClientSecret: os.Getenv("ZOHO_CLIENT_SECRET"),
			RefreshToken: os.Getenv("ZOHO_REFRESH_TOKEN"),
			OrgID:        os.Getenv("ZOHO_ORG_ID"),
		},
		Devtron: DevtronConfig{
			BaseURL:  os.Getenv("DEVTRON_BASE_URL"),
			APIToken: os.Getenv("DEVTRON_API_TOKEN"),
		},
		Auth: AuthConfig{
			Secret:        authSecret(),
			CookieSecure:  getbool("COOKIE_SECURE", false),
			TokenTTL:      getdur("AUTH_TOKEN_TTL_SECONDS", 12*time.Hour),
			AdminEmail:    os.Getenv("ADMIN_EMAIL"),
			AdminPassword: os.Getenv("ADMIN_PASSWORD"),
		},
		CacheTTL: getdur("CACHE_TTL_SECONDS", 60*time.Second),
	}

	return cfg, nil
}

// authSecret reads AUTH_SECRET, or generates a random ephemeral one.
func authSecret() []byte {
	if s := os.Getenv("AUTH_SECRET"); s != "" {
		return []byte(s)
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic("auth: cannot generate secret: " + err.Error())
	}
	log.Printf("warning: AUTH_SECRET unset — generated an ephemeral secret; sessions will not survive restart")
	return buf
}

func getbool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return fallback
}

// Validate reports which integrations are unconfigured. It never fails hard so
// the dashboard can boot with a subset of modules live.
func (c *Config) MissingIntegrations() []string {
	var missing []string
	if c.Zoho.ClientID == "" || c.Zoho.RefreshToken == "" || c.Zoho.OrgID == "" {
		missing = append(missing, "zoho")
	}
	if c.Devtron.BaseURL == "" || c.Devtron.APIToken == "" {
		missing = append(missing, "devtron")
	}
	return missing
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getdur(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return fallback
}

func (c *Config) String() string {
	return fmt.Sprintf("port=%s db=%s zoho=%s devtron=%s", c.Port, c.DBPath, c.Zoho.APIBase, c.Devtron.BaseURL)
}
