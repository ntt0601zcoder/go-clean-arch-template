// Package server is the composition root for the API process: it runs the Gin
// HTTP server, the gRPC server (with health + reflection) and a sibling HTTP
// port exposing gRPC liveness/readiness for k8s probes.
//
//	@title			Account Service API
//	@version		1.0
//	@description	Sample clean-architecture Go service (account domain).
//	@BasePath		/api/v1
package server

import (
	"context"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/fx"
	"gorm.io/gorm"

	grpchandler "github.com/ntt0601zcoder/go-clean-arch-template/internal/adapter/grpc"
	httphandler "github.com/ntt0601zcoder/go-clean-arch-template/internal/adapter/http"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/apps"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/config"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/ginx"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/grpcserver"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/grpcx"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/httpserver"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/metrics"

	_ "github.com/ntt0601zcoder/go-clean-arch-template/docs/swagger" // swag-generated spec (regenerate: make swagger)
)

// Start boots and runs the API process until interrupted.
func Start() {
	apps.New(
		run,
		apps.WithProvides(
			httphandler.NewAccountHandler,
			grpchandler.NewAccountHandler,
		),
	).Run()
}

type params struct {
	fx.In

	LC          fx.Lifecycle
	Cfg         *config.Config
	Log         *slog.Logger
	Metrics     *metrics.Metrics
	AccountHTTP *httphandler.AccountHandler
	AccountGRPC *grpchandler.AccountHandler

	// readiness dependencies
	Redis  *redis.Client
	GormDB *gorm.DB
}

func run(p params) error {
	checks := map[string]ginx.HealthChecker{
		"redis": func(ctx context.Context) error { return p.Redis.Ping(ctx).Err() },
		"postgres": func(ctx context.Context) error {
			sqlDB, err := p.GormDB.DB()
			if err != nil {
				return err
			}
			return sqlDB.PingContext(ctx)
		},
	}

	if err := p.registerHTTP(checks); err != nil {
		return err
	}
	return p.registerGRPC(checks)
}

func (p params) registerHTTP(checks map[string]ginx.HealthChecker) error {
	api := httpserver.New(p.Cfg.App.HTTPAddr, p.Log,
		ginx.Recovery(p.Log),
		ginx.CorrelationMiddleware(),
		ginx.TracingMiddleware(p.Cfg.Telemetry.ServiceName),
		ginx.LoggerMiddleware(p.Log),
		ginx.MetricsMiddleware(p.Metrics),
		ginx.ErrorMiddleware(),
	)

	engine := api.Engine()
	ginx.RegisterHealth(engine, checks)
	ginx.RegisterMetrics(engine, p.Metrics)
	if p.Cfg.App.PprofEnabled {
		ginx.RegisterPprof(engine)
	}
	if p.Cfg.App.SwaggerEnabled {
		ginx.RegisterSwagger(engine)
	}

	p.AccountHTTP.RegisterRoutes(engine.Group("/api/v1"))

	fxRegister(p.LC, api)
	return nil
}

func (p params) registerGRPC(checks map[string]ginx.HealthChecker) error {
	validation, err := grpcx.NewValidationInterceptor()
	if err != nil {
		return err
	}

	srv := grpcserver.New(p.Cfg.App.GRPCAddr, p.Log,
		grpcserver.WithReflection(),
		grpcserver.WithStatsHandler(otelgrpc.NewServerHandler()),
		grpcserver.WithUnaryInterceptors(grpcx.NewErrorInterceptor(), validation),
	)
	p.AccountGRPC.RegisterServer(srv.GRPC())
	fxRegister(p.LC, srv)

	// Sibling HTTP port so HTTP-based probes can watch the gRPC process.
	health := httpserver.New(p.Cfg.App.GRPCHealthAddr, p.Log)
	ginx.RegisterHealth(health.Engine(), checks)
	ginx.RegisterMetrics(health.Engine(), p.Metrics)
	fxRegister(p.LC, health)
	return nil
}

// lifecycled is the start/stop surface shared by the http and grpc servers.
type lifecycled interface {
	Start(context.Context) error
	Stop(context.Context) error
}

func fxRegister(lc fx.Lifecycle, c lifecycled) {
	lc.Append(fx.Hook{OnStart: c.Start, OnStop: c.Stop})
}
