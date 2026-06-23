# go-clean-arch-template

A production-ready Go project template following **Clean / Hexagonal Architecture**.

It ships a runnable sample `account` domain wired through every layer, three swappable
persistence backends behind one interface, FX-based dependency injection, and the
batteries you need for a real service: cobra-free multi-process entrypoint, HTTP + gRPC
APIs, a background worker, observability, and code generation.

## Features

- **Hexagonal layers** — `domain → ports → core → adapter`, wired in `internal/apps` with [Uber FX](https://github.com/uber-go/fx).
- **One repository interface, your choice of backend** — the port is defined once; the template wires GORM (Postgres) and ships pgx + [sqlc](https://sqlc.dev) and MongoDB implementations as drop-in examples. Swap the backend in the composition root — the core never changes. Reads go through a cache decorator.
- **Transaction manager** abstraction over [avito-tech/go-transaction-manager](https://github.com/avito-tech/go-transaction-manager) (ctx-propagated, uniform across all three backends).
- **Caching** behind one interface ([eko/gocache](https://github.com/eko/gocache)) with **Redis** and **in-memory** implementations.
- **Distributed lock** (Redis) and **business-level rate limiting** ([redis_rate](https://github.com/go-redis/redis_rate)).
- **Three processes** from one binary: `server` (HTTP + gRPC), `worker` (Kafka consumer + lock-guarded scheduled job), `migrate`. Each exposes **liveness/readiness**.
- **Observability**: structured logging (slog), Prometheus metrics, OpenTelemetry tracing, pprof.
- **gRPC** with protobuf + [buf](https://buf.build) validation, reflection and health; **REST** with Gin + **Swagger** (swaggo).
- **Tests** (table-driven, `-race`), `golangci-lint` config, multi-stage **Dockerfile**, and a **docker-compose** with Postgres + Mongo (replica set) + Redis.

## Quick start

```bash
cp .env.example .env
make docker-up            # Postgres + Mongo(rs0) + Redis
make migrate              # or: go run main.go migrate
make run-server           # or: go run main.go server
```

Then:

- REST:    `http://localhost:8080/api/v1/accounts`
- Swagger: `http://localhost:8080/swagger/index.html`
- Metrics: `http://localhost:8080/metrics`
- Health:  `http://localhost:8080/liveness`, `/readiness`
- gRPC:    `localhost:9090` (reflection + `grpc.health.v1`)

Run the worker in another shell:

```bash
make run-worker           # or: go run main.go worker
```

## Swapping the backend

The app uses one storage backend, wired in code (no env switch). GORM (Postgres)
is wired by default in `internal/apps/provider.go` (`StorageModule`). To use a
different store, change that one module to provide the pgx+sqlc or mongo
implementation instead — both already implement the same `AccountRepository` /
`TxManager` ports, so nothing else changes. The cache decorator likewise wraps
the Redis cache; swap it for the in-memory one in `CacheModule`.

## Project layout

```text
main.go                      # go run main.go <server|worker|migrate>
internal/
  domain/                    # entities, DTOs, constants (stdlib only)
  ports/{inbound,outbound}/  # interfaces (use cases / repositories & infra)
  core/{services,usecase}/   # business logic
  adapter/                   # http, grpc, repo/{gorm,sqlc,mongo,cached}, cache, lock, limiter, kafka
  infra/                     # config, logger, apperr, db, ginx, grpcx, httpx, metrics, otelx, servers
  apps/                      # FX composition roots (appserver, appworker, appmigrate)
  gen/                       # generated protobuf/gRPC (committed)
api/proto/                   # .proto sources
migrations/                  # embedded golang-migrate SQL
```

## Requirements

Go 1.25+, Docker (for local infra). Generators (`make tools`): buf, protoc-gen-go,
protoc-gen-go-grpc, sqlc, swag, mockgen.
