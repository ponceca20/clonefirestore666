package config

import (
	"errors"

	"github.com/caarlos0/env/v6"
)

// RealtimeConfig holds configuration specific to real-time functionalities.
type RealtimeConfig struct {
	// WebSocketPath is the endpoint path for WebSocket connections.
	// Example: "/ws/v1/listen"
	WebSocketPath string `env:"WEBSOCKET_PATH" envDefault:"/ws/v1/listen" mapstructure:"websocket_path" json:"websocket_path"`

	// ClientSendChannelBuffer is the buffer size for channels sending events to WebSocket clients.
	// Helps in preventing blocking when broadcasting events if a client is slow.
	ClientSendChannelBuffer int `env:"CLIENT_SEND_CHANNEL_BUFFER" envDefault:"10" mapstructure:"client_send_channel_buffer" json:"client_send_channel_buffer"`

	// Example: HandshakeTimeout for WebSocket connections
	// HandshakeTimeout time.Duration `env:"HANDSHAKE_TIMEOUT" envDefault:"5s" mapstructure:"handshake_timeout" json:"handshake_timeout"`

	// Example: MaxMessageSize for WebSocket messages
	// MaxMessageSize int64 `env:"MAX_MESSAGE_SIZE" envDefault:"1048576" mapstructure:"max_message_size" json:"max_message_size"`
}

// CORSConfig holds configuration for CORS middleware
type CORSConfig struct {
	AllowOrigins     string `env:"CORS_ALLOW_ORIGINS" envDefault:"http://localhost:3000,https://tudominio.com"`
	AllowMethods     string `env:"CORS_ALLOW_METHODS" envDefault:"GET,POST,PUT,DELETE,PATCH,OPTIONS"`
	AllowHeaders     string `env:"CORS_ALLOW_HEADERS" envDefault:"Origin,Content-Type,Accept,Authorization,X-Requested-With"`
	AllowCredentials bool   `env:"CORS_ALLOW_CREDENTIALS" envDefault:"true"`
}

// RedisConfig holds configuration for Redis connection and event storage
type RedisConfig struct {
	// Host is the Redis server hostname
	Host string `env:"REDIS_HOST" envDefault:"localhost" mapstructure:"host" json:"host"`

	// Port is the Redis server port
	Port string `env:"REDIS_PORT" envDefault:"6379" mapstructure:"port" json:"port"`

	// Password for Redis authentication (if required)
	Password string `env:"REDIS_PASSWORD" envDefault:"" mapstructure:"password" json:"password"`
	// Database number to use (0-15 for standard Redis)
	Database int `env:"REDIS_DB" envDefault:"0" mapstructure:"database" json:"database"`

	// MaxRetries for Redis operations
	MaxRetries int `env:"REDIS_MAX_RETRIES" envDefault:"3" mapstructure:"max_retries" json:"max_retries"`

	// PoolSize is the maximum number of socket connections
	PoolSize int `env:"REDIS_POOL_SIZE" envDefault:"10" mapstructure:"pool_size" json:"pool_size"`

	// MinIdleConns is the minimum number of idle connections
	MinIdleConns int `env:"REDIS_MIN_IDLE_CONNS" envDefault:"2" mapstructure:"min_idle_conns" json:"min_idle_conns"`

	// EnableTLS enables TLS connection to Redis
	EnableTLS bool `env:"REDIS_ENABLE_TLS" envDefault:"false" mapstructure:"enable_tls" json:"enable_tls"`

	// ConnMaxIdleTime is the maximum amount of time a connection may be idle
	ConnMaxIdleTime string `env:"REDIS_CONN_MAX_IDLE_TIME" envDefault:"30m" mapstructure:"conn_max_idle_time" json:"conn_max_idle_time"`

	// ConnMaxLifetime is the maximum amount of time a connection may be reused
	ConnMaxLifetime string `env:"REDIS_CONN_MAX_LIFETIME" envDefault:"1h" mapstructure:"conn_max_lifetime" json:"conn_max_lifetime"`

	// StreamMaxLength is the maximum length for Redis Streams (for event retention)
	StreamMaxLength int64 `env:"REDIS_STREAM_MAX_LENGTH" envDefault:"10000" mapstructure:"stream_max_length" json:"stream_max_length"`
}

// GetAddr returns the Redis address in host:port format
func (r *RedisConfig) GetAddr() string {
	return r.Host + ":" + r.Port
}

// FirestoreConfig holds all configuration for the Firestore module.
type FirestoreConfig struct {
	MongoDBURI          string         `env:"MONGODB_URI"`
	DefaultDatabaseName string         `env:"MONGODB_DEFAULT_DATABASE" envDefault:"firestore_default"`
	Realtime            RealtimeConfig `mapstructure:"realtime" json:"realtime"`
	CORS                CORSConfig     `mapstructure:"cors" json:"cors"`
	Redis               RedisConfig    `mapstructure:"redis" json:"redis"`
	// Other configurations for persistence, security rules, etc.
}

// LoadConfig loads configuration from environment variables and applies defaults.
func LoadConfig() (*FirestoreConfig, error) {
	cfg := &FirestoreConfig{}

	// Load root FirestoreConfig fields from environment variables
	if err := env.Parse(cfg); err != nil {
		return nil, errors.New("failed to load root firestore configuration from environment: " + err.Error())
	}

	// Load nested RealtimeConfig from environment variables
	if err := env.Parse(&cfg.Realtime); err != nil {
		return nil, errors.New("failed to load firestore realtime configuration from environment: " + err.Error())
	}

	// Load nested CORSConfig from environment variables
	if err := env.Parse(&cfg.CORS); err != nil {
		return nil, errors.New("failed to load firestore CORS configuration from environment: " + err.Error())
	}

	// Load nested RedisConfig from environment variables
	if err := env.Parse(&cfg.Redis); err != nil {
		return nil, errors.New("failed to load firestore Redis configuration from environment: " + err.Error())
	}

	// Validate configuration
	if cfg.MongoDBURI == "" {
		// MONGODB_URI is critical, return an error if not set
		return nil, errors.New("MONGODB_URI environment variable is not set")
	}
	if cfg.Realtime.WebSocketPath == "" {
		cfg.Realtime.WebSocketPath = "/ws/v1/listen"
	}
	if cfg.Realtime.ClientSendChannelBuffer <= 0 {
		cfg.Realtime.ClientSendChannelBuffer = 10
	}

	return cfg, nil
}

// DefaultFirestoreConfig returns a FirestoreConfig with default values.
func DefaultFirestoreConfig() *FirestoreConfig {
	return &FirestoreConfig{
		MongoDBURI:          "mongodb://localhost:27017", // Default for local development
		DefaultDatabaseName: "firestore_default",
		Realtime: RealtimeConfig{
			WebSocketPath:           "/ws/v1/listen", // Default path
			ClientSendChannelBuffer: 10,              // Default buffer size
			// HandshakeTimeout: time.Second * 5,
			// MaxMessageSize: 1024 * 1024, // 1MB
		},
		CORS: CORSConfig{
			AllowOrigins:     "http://localhost:3000,https://tudominio.com",
			AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
			AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Requested-With",
			AllowCredentials: true,
		},
		Redis: RedisConfig{
			Host:            "localhost",
			Port:            "6379",
			Password:        "",
			Database:        0,
			MaxRetries:      3,
			PoolSize:        10,
			MinIdleConns:    2,
			EnableTLS:       false,
			StreamMaxLength: 10000,
		},
		// Initialize other defaults here
	}
}

// LoadConfig would typically load configuration from a file or environment variables.
// For now, it can just return default or be a placeholder.
// func LoadConfig(path string) (*FirestoreConfig, error) {
// 	 viper.SetConfigFile(path)
// 	 if err := viper.ReadInConfig(); err != nil {
// 	 	return nil, err
// 	 }
// 	 var cfg FirestoreConfig
// 	 if err := viper.Unmarshal(&cfg); err != nil {
// 	 	return nil, err
// 	 }
// 	 return &cfg, nil
// }
