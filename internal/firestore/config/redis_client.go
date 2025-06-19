package config

import (
	"crypto/tls"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates a new Redis client using the provided configuration
// This function follows Firestore's connection patterns and includes proper error handling
func NewRedisClient(cfg *RedisConfig) *redis.Client {
	// Parse duration strings
	connMaxIdleTime, _ := time.ParseDuration(cfg.ConnMaxIdleTime)
	if connMaxIdleTime == 0 {
		connMaxIdleTime = 30 * time.Minute // default
	}

	connMaxLifetime, _ := time.ParseDuration(cfg.ConnMaxLifetime)
	if connMaxLifetime == 0 {
		connMaxLifetime = 1 * time.Hour // default
	}

	options := &redis.Options{
		Addr:         cfg.GetAddr(),
		Password:     cfg.Password,
		DB:           cfg.Database,
		MaxRetries:   cfg.MaxRetries,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,

		// Connection timeouts following Firestore patterns
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,

		// Connection lifecycle management
		ConnMaxIdleTime: connMaxIdleTime,
		ConnMaxLifetime: connMaxLifetime,
	}

	// Enable TLS if configured
	if cfg.EnableTLS {
		options.TLSConfig = &tls.Config{
			ServerName: cfg.Host,
		}
	}

	return redis.NewClient(options)
}

// NewRedisClientWithDefaults creates a Redis client with default configuration
// Useful for testing and development environments
func NewRedisClientWithDefaults() *redis.Client {
	defaultConfig := &RedisConfig{
		Host:            "localhost",
		Port:            "6379",
		Password:        "",
		Database:        0,
		MaxRetries:      3,
		PoolSize:        10,
		MinIdleConns:    2,
		EnableTLS:       false,
		ConnMaxIdleTime: "30m",
		ConnMaxLifetime: "1h",
		StreamMaxLength: 10000,
	}

	return NewRedisClient(defaultConfig)
}
