// Package apps is the composition root. container.go provides a small wrapper
// around Uber FX so each binary (server, worker, migrate) boots the
// shared module graph plus its own providers/invokes with one call to New().
package apps

import (
	"go.uber.org/fx"

	"github.com/ntt0601zcoder/go-clean-arch-template/internal/infra/config"
)

type dependency struct {
	provides []any
	supplies []any
	invokes  []any
}

// Option mutates the per-binary dependency set.
type Option func(*dependency)

// WithProvides registers constructors for this binary.
func WithProvides(provides ...any) Option {
	return func(d *dependency) { d.provides = append(d.provides, provides...) }
}

// WithSupplies registers already-built values for this binary.
func WithSupplies(supplies ...any) Option {
	return func(d *dependency) { d.supplies = append(d.supplies, supplies...) }
}

// WithInvokes registers functions to run after the graph is built.
func WithInvokes(invokes ...any) Option {
	return func(d *dependency) { d.invokes = append(d.invokes, invokes...) }
}

// New builds the fx.App: shared AllModules + the binary's starter (an invoke)
// and options. The config singleton is supplied to the graph.
func New(starter any, opts ...Option) *fx.App {
	cfg := config.GetConfig()

	d := &dependency{}
	for _, opt := range opts {
		opt(d)
	}
	d.supplies = append(d.supplies, cfg)
	d.invokes = append(d.invokes, starter)

	return fx.New(
		fx.StartTimeout(cfg.App.StartTimeout),
		fx.StopTimeout(cfg.App.StopTimeout),
		AllModules,
		fx.Provide(d.provides...),
		fx.Supply(d.supplies...),
		fx.Invoke(d.invokes...),
	)
}
