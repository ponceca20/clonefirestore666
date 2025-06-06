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
	// MongoDB Configuration
	MongoDBURI   string `env:"MONGODB_URI,required"`
	DatabaseName string `env:"DATABASE_NAME" envDefault:"firestore_auth_db"`

	// JWT Configuration
	JWTSecretKey   string        `env:"JWT_SECRET_KEY,required"`
	JWTIssuer      string        `env:"JWT_ISSUER" envDefault:"firestore-clone-auth-service"`
	AccessTokenTTL time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m"`
	// RefreshTokenTTL is included as it was already present, useful for a complete auth system
	RefreshTokenTTL time.Duration `env:"REFRESH_TOKEN_TTL" envDefault:"168h"` // 7 days

	// Cookie Configuration
	CookieName     string `env:"COOKIE_NAME" envDefault:"fs_auth_token"`
	CookiePath     string `env:"COOKIE_PATH" envDefault:"/"`
	CookieDomain   string `env:"COOKIE_DOMAIN" envDefault:""`      // Defaults to host, empty is fine
	CookieSecure   bool   `env:"COOKIE_SECURE" envDefault:"false"` // Set to true in production
	CookieHTTPOnly bool   `env:"COOKIE_HTTP_ONLY" envDefault:"true"`
	CookieSameSite string `env:"COOKIE_SAME_SITE" envDefault:"Lax"` // "Lax", "Strict", "None"
}

// LoadConfig loads configuration from environment variables and applies defaults.
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	// Attempt to load from environment variables using github.com/caarlos0/env/v6
	if err := env.Parse(cfg); err != nil {
		// This block is for handling cases where env.Parse itself fails or
		// for complex fallback logic not covered by envDefault.
		// For simple defaults, envDefault is preferred.
		// The existing fallback logic for JWTSecretKey, MongoDBURI, DatabaseName
		// if they are not set by environment variables and are required.
		// This part is mostly for local development convenience if env vars aren't set.
		// In production, all required envs should be set.

		// Example of handling a required field if not set by env and no default:
		if cfg.JWTSecretKey == "" && os.Getenv("JWT_SECRET_KEY") == "" {
			// This indicates 'required' tag failed or was bypassed by partial load.
			// Setting a development-only default here if truly needed for local runs.
			// However, caarlos0/env should handle 'required' by returning an error.
			// This explicit set is more of a safeguard or for non-env loading scenarios.
			// cfg.JWTSecretKey = "verysecretkeylocaltodevelopmentonly_fallback"
			// log.Println("Warning: JWT_SECRET_KEY not set, using insecure development fallback key.")
		}
		if cfg.MongoDBURI == "" && os.Getenv("MONGODB_URI") == "" {
			// cfg.MongoDBURI = "mongodb://localhost:27017/dev_auth_fallback"
			// log.Println("Warning: MONGODB_URI not set, using local development fallback.")
		}
		// It's better to let env.Parse fail on required fields if they are not set.
		// The code below for re-parsing is generally not needed if envDefault and required tags are used correctly.
		// For this exercise, we simplify and assume env.Parse handles most cases.
		// If after Parse, critical fields are still missing, it implies misconfiguration or missing envDefault.
		// The initial error from env.Parse(cfg) should be the primary indicator of missing required envs.
		// If we want to set OS env vars and re-parse, that's an option:
		// os.Setenv("JWT_SECRET_KEY", "...") // if for some reason needed
		// if errReParse := env.Parse(cfg); errReParse != nil {
		//    return nil, errors.New("failed to load configuration after attempting fallback: " + errReParse.Error())
		// }
		// For now, we'll rely on the first parse and then validate.
		// The original code had some os.Setenv calls, which can be useful for ensuring
		// the cfg struct accurately reflects what would be used, even if defaults were applied manually.
		// Let's keep it simple: env.Parse does its job, then we validate.
		// The error `err` from `env.Parse(cfg)` should be returned if critical.
		// The current structure of the original code tries to be too clever with fallbacks
		// that might hide actual missing configuration issues.
		// We will return the error from env.Parse if any required field is missing.
		return nil, errors.New("failed to load configuration from environment: " + err.Error() +
			". Please ensure all required environment variables are set.")
	}

	// Validations after attempting to load from environment
	if cfg.JWTSecretKey == "" {
		// This should ideally be caught by `env:",required"`
		return nil, errors.New("jwt_secret_key is required")
	}
	if len(cfg.JWTSecretKey) < 32 && cfg.JWTIssuer != "dev-issuer-for-testing-only" { // Example length check
		// Allow short keys only for a specific dev issuer to avoid accidental weak prod keys
		// return nil, errors.New("jwt_secret_key must be at least 32 characters long for production")
		// For now, we'll just ensure it's not empty, length check can be added if policy dictates.
	}
	if cfg.MongoDBURI == "" {
		// This should also be caught by `env:",required"`
		return nil, errors.New("mongodb_uri is required")
	}

	// Normalize and validate CookieSameSite
	cfg.CookieSameSite = strings.Title(strings.ToLower(cfg.CookieSameSite))
	if !(cfg.CookieSameSite == "Lax" || cfg.CookieSameSite == "Strict" || cfg.CookieSameSite == "None") {
		return nil, errors.New("cookie_same_site must be one of 'Lax', 'Strict', or 'None'")
	}

	// Ensure DatabaseName has a default if not set by env and envDefault didn't catch it (should not happen with caarlos0/env)
	if cfg.DatabaseName == "" {
		cfg.DatabaseName = "firestore_auth_db_fallback" // Fallback, though envDefault should handle
	}
	if cfg.JWTIssuer == "" {
		cfg.JWTIssuer = "firestore-clone-auth-service-fallback" // Fallback
	}
	if cfg.AccessTokenTTL == 0 {
		cfg.AccessTokenTTL = 15 * time.Minute // Fallback
	}
	if cfg.CookieName == "" {
		cfg.CookieName = "fs_auth_token_fallback" // Fallback
	}

	return cfg, nil
}
