// Package worker is the composition root for the worker process. It shows two
// common background patterns, decoupled from the synchronous API:
//
//   - a scheduled job guarded by a distributed lock, so exactly one replica runs
//     it per tick (here: a tiny "housekeeping" job that counts accounts);
//   - a typed Kafka consumer of generic domain.AccountEvent messages
//     (at-least-once, commit-after-handle).
//
// Both are illustrative scaffolding — replace the job/handler bodies with real
// work for your service.
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"

	kafkaadapter "github.com/ntt0601zcoder/go-clean-arch-template/internal/adapter/kafka"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/apps"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/domain"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/config"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/ginx"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/httpserver"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/metrics"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/inbound"
	"github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound"
)

// Start boots and runs the worker process until interrupted.
func Start() {
	apps.New(run).Run()
}

type params struct {
	fx.In

	LC        fx.Lifecycle
	Cfg       *config.Config
	Log       *slog.Logger
	Metrics   *metrics.Metrics
	AccountUC inbound.AccountUseCase
	Locker    outbound.DistributedLocker
	Redis     *redis.Client
}

func run(p params) {
	p.startProbeServer()
	p.startEventConsumer()
	p.startScheduledJob()
}

// startProbeServer exposes liveness/readiness/metrics for the worker process.
func (p params) startProbeServer() {
	checks := map[string]ginx.HealthChecker{
		"redis": func(ctx context.Context) error { return p.Redis.Ping(ctx).Err() },
	}
	probe := httpserver.New(p.Cfg.App.WorkerHealthAddr, p.Log)
	ginx.RegisterHealth(probe.Engine(), checks)
	ginx.RegisterMetrics(probe.Engine(), p.Metrics)
	p.LC.Append(fx.Hook{OnStart: probe.Start, OnStop: probe.Stop})
}

// startEventConsumer consumes generic account events and processes them. Here it
// just logs and counts them; swap in real handling (projections, notifications…).
func (p params) startEventConsumer() {
	reader := kafkaadapter.NewReader(p.Cfg.Kafka.Brokers, p.Cfg.Kafka.Topic, p.Cfg.Kafka.GroupID)
	consumer := kafkaadapter.NewConsumer(reader,
		func(ctx context.Context, evt domain.AccountEvent) error {
			p.Log.InfoContext(ctx, "account event received",
				slog.String("type", string(evt.Type)),
				slog.String("account_id", evt.AccountID))
			p.Metrics.WorkerRuns.WithLabelValues("event", string(evt.Type)).Inc()
			return nil
		}, p.Log)

	ctx, cancel := context.WithCancel(context.Background())
	p.LC.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := consumer.Run(ctx); err != nil {
					p.Log.Error("kafka consumer stopped", slog.Any("err", err))
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			cancel()
			return consumer.Close()
		},
	})
}

// startScheduledJob runs a periodic job guarded by the distributed lock so only
// one worker instance executes it per tick.
func (p params) startScheduledJob() {
	ctx, cancel := context.WithCancel(context.Background())
	ticker := time.NewTicker(p.Cfg.Worker.Interval)

	p.LC.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						p.runHousekeeping(ctx)
					}
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			ticker.Stop()
			cancel()
			return nil
		},
	})
}

// runHousekeeping is the lock-guarded periodic task. As a sample it counts active
// accounts via the use case; replace with real maintenance work.
func (p params) runHousekeeping(ctx context.Context) {
	release, err := p.Locker.Acquire(ctx, p.Cfg.Worker.LockKey, p.Cfg.Worker.LockTTL)
	if err != nil {
		// Another replica holds the lock — skip this tick.
		p.Metrics.WorkerRuns.WithLabelValues("housekeeping", "skipped").Inc()
		return
	}
	defer func() { _ = release(ctx) }()

	active := domain.AccountStatusActive
	_, total, err := p.AccountUC.List(ctx, domain.ListAccountFilter{Status: &active, Limit: 1000})
	if err != nil {
		p.Log.ErrorContext(ctx, "housekeeping failed", slog.Any("err", err))
		p.Metrics.WorkerRuns.WithLabelValues("housekeeping", "error").Inc()
		return
	}

	p.Log.InfoContext(ctx, "housekeeping complete (lock held)", slog.Int("active_accounts", total))
	p.Metrics.WorkerRuns.WithLabelValues("housekeeping", "ok").Inc()
}
