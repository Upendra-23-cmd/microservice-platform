package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration loaded from environment variables.
// All fields are tagged with mapstructure for automatic binding via Viper.
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	GRPC     GRPCConfig     `mapstructure:"grpc"`
	HTTP     HTTPConfig     `mapstructure:"http"`
	Postgres PostgresConfig `mapstructure:"postgres"`
	MongoDB  MongoDBConfig  `mapstructure:"mongodb"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	TLS      TLSConfig      `mapstructure:"tls"`
	Otel     OtelConfig     `mapstructure:"otel"`
	Rate     RateLimitConfig `mapstructure:"rate_limit"`
	CORS     CORSConfig     `mapstructure:"cors"`
}

type AppConfig struct {
	Env      string `mapstructure:"env"`
	Name     string `mapstructure:"name"`
	Version  string `mapstructure:"version"`
	LogLevel string `mapstructure:"log_level"`
}

type GRPCConfig struct {
	Host               string        `mapstructure:"host"`
	Port               int           `mapstructure:"port"`
	MaxRecvMsgSize     int           `mapstructure:"max_recv_msg_size"`
	MaxSendMsgSize     int           `mapstructure:"max_send_msg_size"`
	KeepaliveTime      time.Duration `mapstructure:"keepalive_time"`
	KeepaliveTimeout   time.Duration `mapstructure:"keepalive_timeout"`
	ReflectionEnabled  bool          `mapstructure:"reflection_enabled"`
}

func (g GRPCConfig) Addr() string {
	return fmt.Sprintf("%s:%d", g.Host, g.Port)
}

type HTTPConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

func (h HTTPConfig) Addr() string {
	return fmt.Sprintf("%s:%d", h.Host, h.Port)
}

type PostgresConfig struct {
	Host             string        `mapstructure:"host"`
	Port             int           `mapstructure:"port"`
	DB               string        `mapstructure:"db"`
	User             string        `mapstructure:"user"`
	Password         string        `mapstructure:"password"`
	SSLMode          string        `mapstructure:"ssl_mode"`
	MaxOpenConns     int32         `mapstructure:"max_open_conns"`
	MaxIdleConns     int32         `mapstructure:"max_idle_conns"`
	ConnMaxLifetime  time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime  time.Duration `mapstructure:"conn_max_idle_time"`
	MigrationDir     string        `mapstructure:"migration_dir"`
}

func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		p.Host, p.Port, p.DB, p.User, p.Password, p.SSLMode,
	)
}

type MongoDBConfig struct {
	URI                    string        `mapstructure:"uri"`
	DB                     string        `mapstructure:"db"`
	MinPoolSize            uint64        `mapstructure:"min_pool_size"`
	MaxPoolSize            uint64        `mapstructure:"max_pool_size"`
	MaxConnIdleTime        time.Duration `mapstructure:"max_conn_idle_time"`
	ServerSelectionTimeout time.Duration `mapstructure:"server_selection_timeout"`
	ConnectTimeout         time.Duration `mapstructure:"connect_timeout"`
}

type RedisConfig struct {
	Host          string        `mapstructure:"host"`
	Port          int           `mapstructure:"port"`
	Password      string        `mapstructure:"password"`
	DB            int           `mapstructure:"db"`
	PoolSize      int           `mapstructure:"pool_size"`
	MinIdleConns  int           `mapstructure:"min_idle_conns"`
	DialTimeout   time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout   time.Duration `mapstructure:"read_timeout"`
	WriteTimeout  time.Duration `mapstructure:"write_timeout"`
	DefaultTTL    time.Duration `mapstructure:"default_ttl"`
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type JWTConfig struct {
	AccessSecret  string        `mapstructure:"access_secret"`
	RefreshSecret string        `mapstructure:"refresh_secret"`
	AccessExpiry  time.Duration `mapstructure:"access_expiry"`
	RefreshExpiry time.Duration `mapstructure:"refresh_expiry"`
	Issuer        string        `mapstructure:"issuer"`
}

type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
	CAFile   string `mapstructure:"ca_file"`
}

type OtelConfig struct {
	ExporterEndpoint    string `mapstructure:"exporter_otlp_endpoint"`
	PrometheusPort      int    `mapstructure:"prometheus_metrics_port"`
	PrometheusPath      string `mapstructure:"prometheus_metrics_path"`
	HealthCheckPort     int    `mapstructure:"health_check_port"`
}

type RateLimitConfig struct {
	Enabled bool    `mapstructure:"enabled"`
	RPS     float64 `mapstructure:"rps"`
	Burst   int     `mapstructure:"burst"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
	MaxAge         int      `mapstructure:"max_age"`
}

// Load reads configuration from environment variables and/or a .env file.
// Environment variables always take precedence over file values.
func Load() (*Config, error) {
	v := viper.New()

	// Automatically read environment variables
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	setDefaults(v)

	// Optionally read .env file (non-fatal if missing in production)
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	_ = v.ReadInConfig() // Intentionally ignore missing .env in production

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: failed to unmarshal: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config: validation failed: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.env", "production")
	v.SetDefault("app.name", "microservice-platform")
	v.SetDefault("app.version", "1.0.0")
	v.SetDefault("app.log_level", "info")

	v.SetDefault("grpc.host", "0.0.0.0")
	v.SetDefault("grpc.port", 50051)
	v.SetDefault("grpc.max_recv_msg_size", 10*1024*1024) // 10MB
	v.SetDefault("grpc.max_send_msg_size", 10*1024*1024)
	v.SetDefault("grpc.keepalive_time", 30*time.Second)
	v.SetDefault("grpc.keepalive_timeout", 10*time.Second)

	v.SetDefault("http.host", "0.0.0.0")
	v.SetDefault("http.port", 8080)
	v.SetDefault("http.read_timeout", 15*time.Second)
	v.SetDefault("http.write_timeout", 15*time.Second)
	v.SetDefault("http.idle_timeout", 60*time.Second)
	v.SetDefault("http.shutdown_timeout", 30*time.Second)

	v.SetDefault("postgres.ssl_mode", "require")
	v.SetDefault("postgres.max_open_conns", 25)
	v.SetDefault("postgres.max_idle_conns", 10)
	v.SetDefault("postgres.conn_max_lifetime", 5*time.Minute)
	v.SetDefault("postgres.conn_max_idle_time", time.Minute)

	v.SetDefault("mongodb.min_pool_size", 5)
	v.SetDefault("mongodb.max_pool_size", 50)
	v.SetDefault("mongodb.max_conn_idle_time", 30*time.Second)
	v.SetDefault("mongodb.server_selection_timeout", 5*time.Second)
	v.SetDefault("mongodb.connect_timeout", 10*time.Second)

	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 20)
	v.SetDefault("redis.min_idle_conns", 5)
	v.SetDefault("redis.dial_timeout", 5*time.Second)
	v.SetDefault("redis.read_timeout", 3*time.Second)
	v.SetDefault("redis.write_timeout", 3*time.Second)
	v.SetDefault("redis.default_ttl", time.Hour)

	v.SetDefault("jwt.access_expiry", 15*time.Minute)
	v.SetDefault("jwt.refresh_expiry", 7*24*time.Hour)
	v.SetDefault("jwt.issuer", "microservice-platform")

	v.SetDefault("otel.prometheus_metrics_port", 9090)
	v.SetDefault("otel.prometheus_metrics_path", "/metrics")
	v.SetDefault("otel.health_check_port", 8081)

	v.SetDefault("rate_limit.enabled", true)
	v.SetDefault("rate_limit.rps", 100)
	v.SetDefault("rate_limit.burst", 200)

	v.SetDefault("cors.max_age", 3600)
}

func validate(cfg *Config) error {
	if cfg.JWT.AccessSecret == "" {
		return fmt.Errorf("JWT_ACCESS_SECRET must be set")
	}
	if len(cfg.JWT.AccessSecret) < 32 {
		return fmt.Errorf("JWT_ACCESS_SECRET must be at least 32 characters")
	}
	if cfg.JWT.RefreshSecret == "" {
		return fmt.Errorf("JWT_REFRESH_SECRET must be set")
	}
	if cfg.Postgres.Host == "" {
		return fmt.Errorf("POSTGRES_HOST must be set")
	}
	if cfg.MongoDB.URI == "" {
		return fmt.Errorf("MONGODB_URI must be set")
	}
	return nil
}
