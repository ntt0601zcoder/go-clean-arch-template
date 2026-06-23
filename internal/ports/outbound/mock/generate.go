// Package mock holds generated mocks for the outbound ports. The *_mock.go
// files are produced by `go generate ./internal/ports/outbound/mock/...` (uber
// mockgen) and are NOT committed. Core unit tests can use these mocks or the
// hand-written fakes that ship alongside the tests.
package mock

//go:generate go run go.uber.org/mock/mockgen -destination=account_repository_mock.go -package=mock github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound AccountRepository
//go:generate go run go.uber.org/mock/mockgen -destination=tx_manager_mock.go -package=mock github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound TxManager,Provider
//go:generate go run go.uber.org/mock/mockgen -destination=cache_mock.go -package=mock github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound CacheManager
//go:generate go run go.uber.org/mock/mockgen -destination=lock_mock.go -package=mock github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound DistributedLocker
//go:generate go run go.uber.org/mock/mockgen -destination=limiter_mock.go -package=mock github.com/ntt0601zcoder/go-clean-arch-template/internal/ports/outbound RateLimiter
