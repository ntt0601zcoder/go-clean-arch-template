// Package config loads all runtime configuration from environment variables
// (with a .env file auto-loaded in dev via godotenv) using caarlos0/env, then
// exposes a process-wide singleton via GetConfig(). Each layer is handed only
// the slice of config it needs.
package config

import (
	"fmt"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"
	_ "github.com/joho/godotenv/autoload" // load .env in dev; no-op if absent
)

// Config is the fully resolved configuration.
type Config struct {
	App       AppConfig       `envPrefix:"APP_"`
	Postgres  PostgresConfig  `envPrefix:"POSTGRES_"`
	Mongo     MongoConfig     `envPrefix:"MONGO_"`
	Redis     RedisConfig     `envPrefix:"REDIS_"`
	Kafka     KafkaConfig     `envPrefix:"KAFKA_"`
	Telemetry TelemetryConfig `envPrefix:"OTEL_"`
	Worker    WorkerConfig    `envPrefix:"WORKER_"`
}

// AppConfig holds process-wide knobs and listen addresses.
type AppConfig struct {
	Env              string        `env:"ENV" envDefault:"local"`
	LogLevel         string        `env:"LOG_LEVEL" envDefault:"info"`
	StartTimeout     time.Duration `env:"START_TIMEOUT" envDefault:"30s"`
	StopTimeout      time.Duration `env:"STOP_TIMEOUT" envDefault:"30s"`
	HTTPAddr         string        `env:"HTTP_ADDR" envDefault:":8080"`
	GRPCAddr         string        `env:"GRPC_ADDR" envDefault:":9090"`
	GRPCHealthAddr   string        `env:"GRPC_HEALTH_ADDR" envDefault:":9091"`
	WorkerHealthAddr string        `env:"WORKER_HEALTH_ADDR" envDefault:":8081"`
	SwaggerEnabled   bool          `env:"SWAGGER_ENABLED" envDefault:"true"`
	PprofEnabled     bool          `env:"PPROF_ENABLED" envDefault:"true"`
}

// PostgresConfig backs both the gorm and pgx+sqlc repositories.
type PostgresConfig struct {
	DSN             string        `env:"DSN" envDefault:"postgres://postgres:postgres@localhost:5432/app?sslmode=disable"`
	MaxOpenConns    int           `env:"MAX_OPEN_CONNS" envDefault:"25"`
	MaxIdleConns    int           `env:"MAX_IDLE_CONNS" envDefault:"5"`
	ConnMaxLifetime time.Duration `env:"CONN_MAX_LIFETIME" envDefault:"30m"`
	Migrate         bool          `env:"MIGRATE" envDefault:"true"`
	MigrateVersion  uint          `env:"MIGRATE_VERSION" envDefault:"0"` // 0 = latest
}

// MongoConfig is the MongoDB repository.
type MongoConfig struct {
	URI      string `env:"URI" envDefault:"mongodb://localhost:27017/?replicaSet=rs0"`
	Database string `env:"DATABASE" envDefault:"app"`
}

// RedisConfig backs cache, lock and rate limiter (shared client).
type RedisConfig struct {
	Addr     string `env:"ADDR" envDefault:"localhost:6379"`
	Password string `env:"PASSWORD" envDefault:""`
	DB       int    `env:"DB" envDefault:"0"`
	PoolSize int    `env:"POOL_SIZE" envDefault:"10"`
}

// KafkaConfig drives the worker consumer.
type KafkaConfig struct {
	Brokers []string `env:"BROKERS" envDefault:"localhost:9092" envSeparator:","`
	Topic   string   `env:"TOPIC" envDefault:"account.events"`
	GroupID string   `env:"GROUP_ID" envDefault:"account-worker"`
}

// TelemetryConfig configures OpenTelemetry tracing. Empty endpoint => tracing
// falls back to a no-op exporter.
type TelemetryConfig struct {
	ServiceName  string  `env:"SERVICE_NAME" envDefault:"go-clean-arch-template"`
	OTLPEndpoint string  `env:"EXPORTER_OTLP_ENDPOINT" envDefault:""`
	SampleRatio  float64 `env:"TRACES_SAMPLE_RATIO" envDefault:"1.0"`
	Insecure     bool    `env:"EXPORTER_INSECURE" envDefault:"true"`
}

// WorkerConfig governs the background worker loop and its locking.
type WorkerConfig struct {
	Interval time.Duration `env:"INTERVAL" envDefault:"30s"`
	LockTTL  time.Duration `env:"LOCK_TTL" envDefault:"1m"`
	LockKey  string        `env:"LOCK_KEY" envDefault:"worker:housekeeping"`
}

var (
	once   sync.Once
	cached *Config
)

// Load parses configuration from the environment.
func Load() (*Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &c, nil
}

// GetConfig returns the process-wide singleton, parsing once. A parse failure at
// boot is fatal.
func GetConfig() *Config {
	once.Do(func() {
		c, err := Load()
		if err != nil {
			panic(err)
		}
		cached = c
	})
	return cached
}
