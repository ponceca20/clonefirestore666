package config

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v6"
)

// Config holds all configuration for the auth module.
type Config struct {
	// Database configuration
	MongoDBURI   string `env:"MONGODB_URI,required"`
	DatabaseName string `env:"DATABASE_NAME" envDefault:"firestore_auth_db"`

	// JWT configuration
	JWTSecretKey    string        `env:"JWT_SECRET_KEY,required"`
	JWTIssuer       string        `env:"JWT_ISSUER" envDefault:"firestore-clone-auth-service"`
	AccessTokenTTL  time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m"`
	RefreshTokenTTL time.Duration `env:"REFRESH_TOKEN_TTL" envDefault:"168h"`
	// Cookie configuration
	CookieName     string `env:"COOKIE_NAME" envDefault:"fs_auth_token"`
	CookiePath     string `env:"COOKIE_PATH" envDefault:"/"`
	CookieDomain   string `env:"COOKIE_DOMAIN"`
	CookieSecure   bool   `env:"COOKIE_SECURE" envDefault:"false"`
	CookieHTTPOnly bool   `env:"COOKIE_HTTP_ONLY" envDefault:"true"`
	CookieSameSite string `env:"COOKIE_SAME_SITE" envDefault:"Lax"`

	// Server configuration
	ReadTimeout  time.Duration `env:"READ_TIMEOUT" envDefault:"30s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" envDefault:"30s"`
	IdleTimeout  time.Duration `env:"IDLE_TIMEOUT" envDefault:"120s"`

	// Security configuration
	BCryptCost int `env:"BCRYPT_COST" envDefault:"12"`

	// Rate limiting
	RateLimitEnabled bool `env:"RATE_LIMIT_ENABLED" envDefault:"true"`
	RateLimitRPS     int  `env:"RATE_LIMIT_RPS" envDefault:"10"`

	// CORS settings
	CORSEnabled      bool     `env:"CORS_ENABLED" envDefault:"true"`
	CORSAllowOrigins []string `env:"CORS_ALLOW_ORIGINS" envSeparator:"," envDefault:"*"`
	CORSAllowMethods []string `env:"CORS_ALLOW_METHODS" envSeparator:"," envDefault:"GET,POST,PUT,DELETE,OPTIONS"`
	CORSAllowHeaders []string `env:"CORS_ALLOW_HEADERS" envSeparator:"," envDefault:"*"`

	// Multitenant configuration
	TenantIsolationEnabled bool   `env:"TENANT_ISOLATION_ENABLED" envDefault:"true"`
	DefaultTenantID        string `env:"DEFAULT_TENANT_ID" envDefault:"default"`

	// Organization/Tenant database settings
	TenantDBPrefix       string        `env:"TENANT_DB_PREFIX" envDefault:"firestore_org_"`
	MaxTenantConnections int           `env:"MAX_TENANT_CONNECTIONS" envDefault:"100"`
	TenantConnectionTTL  time.Duration `env:"TENANT_CONNECTION_TTL" envDefault:"1h"`
}

// LoadConfig loads configuration from environment variables and applies defaults.
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	// Validate required fields
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	// Apply defaults and transformations
	cfg.applyDefaults()

	return cfg, nil
}

// validate performs validation on the loaded configuration
func (c *Config) validate() error {
	if c.MongoDBURI == "" {
		return errors.New("mongodb URI is required")
	}

	if c.JWTSecretKey == "" {
		return errors.New("JWT secret key is required")
	}

	if len(c.JWTSecretKey) < 32 {
		return errors.New("JWT secret key must be at least 32 characters long")
	}

	if c.AccessTokenTTL <= 0 {
		return errors.New("access token TTL must be positive")
	}

	if c.RefreshTokenTTL <= 0 {
		return errors.New("refresh token TTL must be positive")
	}

	if c.BCryptCost < 4 || c.BCryptCost > 31 {
		return errors.New("bcrypt cost must be between 4 and 31")
	}

	return nil
}

// applyDefaults applies defaults and transformations to the configuration
func (c *Config) applyDefaults() {
	// Ensure database name doesn't have special characters
	c.DatabaseName = strings.ReplaceAll(c.DatabaseName, "-", "_")
	c.DatabaseName = strings.ReplaceAll(c.DatabaseName, ".", "_")

	// Ensure tenant DB prefix ends with underscore
	if c.TenantDBPrefix != "" && !strings.HasSuffix(c.TenantDBPrefix, "_") {
		c.TenantDBPrefix += "_"
	}

	// Default CORS origins
	if len(c.CORSAllowOrigins) == 0 {
		c.CORSAllowOrigins = []string{"*"}
	}

	// Resolve environment variables in CORS origins
	for i, origin := range c.CORSAllowOrigins {
		if strings.HasPrefix(origin, "$") {
			envVar := strings.TrimPrefix(origin, "$")
			if value := os.Getenv(envVar); value != "" {
				c.CORSAllowOrigins[i] = value
			}
		}
	}
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	env := os.Getenv("GO_ENV")
	return env == "development" || env == "dev" || env == ""
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	env := os.Getenv("GO_ENV")
	return env == "production" || env == "prod"
}

// GetTenantDatabaseName returns the database name for a specific tenant/organization
func (c *Config) GetTenantDatabaseName(organizationID string) string {
	// Sanitize organization ID for database name
	sanitized := strings.ToLower(organizationID)
	sanitized = strings.ReplaceAll(sanitized, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	sanitized = strings.ReplaceAll(sanitized, " ", "_")

	return c.TenantDBPrefix + sanitized
}
