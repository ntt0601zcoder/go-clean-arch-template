// Package fxutil holds small helpers for registering server lifecycles with
// Uber FX (replacing the reference's interface-common fxutil).
package fxutil

import (
	"context"

	"go.uber.org/fx"
)

// Lifecycled is anything with start/stop semantics (http/grpc servers, workers).
type Lifecycled interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Register hooks a Lifecycled component into the FX lifecycle.
func Register(lc fx.Lifecycle, c Lifecycled) {
	lc.Append(fx.Hook{
		OnStart: c.Start,
		OnStop:  c.Stop,
	})
}

// Append hooks raw start/stop closures into the FX lifecycle.
func Append(lc fx.Lifecycle, onStart, onStop func(context.Context) error) {
	lc.Append(fx.Hook{OnStart: onStart, OnStop: onStop})
}
