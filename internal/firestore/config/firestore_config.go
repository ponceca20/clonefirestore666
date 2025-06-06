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

// FirestoreConfig holds all configuration for the Firestore module.
type FirestoreConfig struct {
	MongoDBURI          string `env:"MONGODB_URI"`
	DefaultDatabaseName string `env:"MONGODB_DEFAULT_DATABASE" envDefault:"firestore_default"`
	// DatabaseURL string `env:"DATABASE_URL" mapstructure:"database_url" json:"database_url"` // Example
	// Port string `env:"PORT" envDefault:"8080" mapstructure:"port" json:"port"` // Example
	Realtime RealtimeConfig `mapstructure:"realtime" json:"realtime"`
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
