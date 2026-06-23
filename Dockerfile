# syntax=docker/dockerfile:1

# ---- builder ----
FROM golang:1.25-alpine AS builder
WORKDIR /src

# Cache modules first.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Build (migrations + generated code are embedded, no extra copy needed).
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/account-service .

# ---- test (optional CI stage: docker build --target test) ----
FROM builder AS test
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go test -race -cover ./...

# ---- runtime ----
FROM alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 app
WORKDIR /app
COPY --from=builder /out/account-service /app/account-service
USER app

# HTTP API, gRPC, gRPC health, worker health
EXPOSE 8080 9090 9091 8081
ENTRYPOINT ["/app/account-service"]
CMD ["server"]
