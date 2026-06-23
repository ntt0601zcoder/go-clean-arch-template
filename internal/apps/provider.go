package apps

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"

	cacheadapter "github.com/ntt0601zcoder/go-clean-arch-template/internal/adapter/cache"
	limiteradapter "github.com/ntt0601zcoder/go-clean-arch-template/internal/adapter/limiter"
	lockadapter "github.com/ntt0601zcoder/go-clean-arch-template/internal/adapter/lock"
	cachedrepo "github.com/ntt0601zcoder/go-clean-arch-template/internal/adapter/repo/cached"
	gormrepo "github.com/ntt0601zcoder/go-clean-arch-template/internal/adapter/repo/gorm"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/core/services"
	accountusecase "github.com/ntt0601zcoder/go-clean-arch-template/internal/core/usecase/account"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/config"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/db"
	cgorm "github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/gorm"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/logger"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/metrics"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/otelx"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/inbound"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound"
	"github.com/ntt0601zcoder/go-clean-arch-template/migrations"
)

const accountCacheTTL = 5 * time.Minute

// AllModules is the full DI graph shared by every binary.
var AllModules = fx.Options(
	ObservabilityModule,
	RedisModule,
	CacheModule,
	StorageModule,
	ServiceModule,
	UseCaseModule,
	LockLimiterModule,
)

// ---- observability ----

// ObservabilityModule provides the logger + metrics and sets up tracing.
var ObservabilityModule = fx.Module("observability",
	fx.Provide(
		func(cfg *config.Config) *slog.Logger { return logger.NewSlogger("app", cfg.App.LogLevel) },
		func(cfg *config.Config) *metrics.Metrics { return metrics.New("app") },
	),
	fx.Invoke(setupTracing),
)

func setupTracing(lc fx.Lifecycle, cfg *config.Config, log *slog.Logger) {
	var shutdown otelx.ShutdownFunc
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			s, err := otelx.InitTracer(ctx, otelx.Config{
				ServiceName:  cfg.Telemetry.ServiceName,
				Environment:  cfg.App.Env,
				OTLPEndpoint: cfg.Telemetry.OTLPEndpoint,
				SampleRatio:  cfg.Telemetry.SampleRatio,
				Insecure:     cfg.Telemetry.Insecure,
			})
			if err != nil {
				return err
			}
			shutdown = s
			log.Info("tracing initialised", slog.String("otlp_endpoint", cfg.Telemetry.OTLPEndpoint))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if shutdown != nil {
				return shutdown(ctx)
			}
			return nil
		},
	})
}

// ---- redis (shared by cache, lock, limiter) ----

// RedisModule provides the shared Redis client.
var RedisModule = fx.Module("redis",
	fx.Provide(NewRedisClient),
)

// NewRedisClient builds the shared Redis client.
func NewRedisClient(lc fx.Lifecycle, cfg *config.Config) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})
	lc.Append(fx.Hook{OnStop: func(context.Context) error { return client.Close() }})
	return client
}

// ---- cache ----

// CacheModule provides the typed account cache.
//
// The template wires the Redis cache. To use the in-process cache instead, swap
// the body for cacheadapter.NewMemoryManager[*domain.Account](accountCacheTTL).
var CacheModule = fx.Module("cache",
	fx.Provide(
		func(rc *redis.Client) outbound.CacheManager[*domain.Account] {
			return cacheadapter.NewRedisManager[*domain.Account](rc)
		},
	),
)

// ---- storage (single backend) ----

// StorageModule wires ONE persistence backend. The template uses GORM (Postgres);
// to switch backend, replace the providers below with the pgx+sqlc or mongo ones:
//
//	pgx+sqlc: provide NewPgxPool + db.NewPgxTxManager (as outbound.TxManager) and
//	          sqlcrepo.NewAccountRepository(pgxTx).
//	mongo:    provide NewMongoDatabase + db.NewMongoTxManager (as outbound.TxManager)
//	          and mongorepo.NewAccountRepository(mongoDB).
//
// Each backend's repo + tx-manager already implement the same ports, so only this
// module changes — the core and transports are untouched.
var StorageModule = fx.Module("storage",
	fx.Provide(
		NewGormDB,
		db.NewGormTxManager,
		// The gorm tx manager satisfies both the repo's Provider and the TxManager port.
		func(m *db.GormTxManager) outbound.Provider { return m },
		func(m *db.GormTxManager) outbound.TxManager { return m },
		NewAccountRepository,
	),
)

// NewGormDB opens the GORM (PostgreSQL) connection and, when configured, applies
// the embedded migrations on boot.
func NewGormDB(cfg *config.Config, log *slog.Logger) (*gorm.DB, error) {
	gormDB, err := gorm.Open(gormpostgres.Open(cfg.Postgres.DSN), &gorm.Config{
		Logger:         cgorm.NewGormLogger(log, "warn"),
		TranslateError: true,
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.Postgres.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Postgres.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Postgres.ConnMaxLifetime)

	if cfg.Postgres.Migrate {
		if err := db.RunMigrations(sqlDB, migrations.FS, cfg.Postgres.MigrateVersion); err != nil {
			return nil, err
		}
	}
	return gormDB, nil
}

// NewAccountRepository builds the active account repository: the GORM backend
// wrapped by the read-through cache decorator.
func NewAccountRepository(provider outbound.Provider, cache outbound.CacheManager[*domain.Account], log *slog.Logger) outbound.AccountRepository {
	return cachedrepo.NewAccountRepository(gormrepo.NewAccountRepository(provider), cache, accountCacheTTL, log)
}

// ---- services ----

// ServiceModule provides the domain services bound to their inbound ports.
var ServiceModule = fx.Module("services",
	fx.Provide(
		fx.Annotate(services.NewPasswordService, fx.As(new(inbound.PasswordService))),
	),
)

// ---- use cases ----

// UseCaseModule provides the account use case bound to its inbound port.
var UseCaseModule = fx.Module("usecases",
	fx.Provide(
		fx.Annotate(accountusecase.NewAccountUseCase, fx.As(new(inbound.AccountUseCase))),
	),
)

// ---- infra adapters (lock, limiter) ----

// LockLimiterModule provides the distributed lock and rate limiter.
var LockLimiterModule = fx.Module("lock-limiter",
	fx.Provide(
		fx.Annotate(lockadapter.NewRedisLocker, fx.As(new(outbound.DistributedLocker))),
		fx.Annotate(limiteradapter.NewRedisLimiter, fx.As(new(outbound.RateLimiter))),
	),
)
