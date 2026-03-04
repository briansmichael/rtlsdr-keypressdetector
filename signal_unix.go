//go:build !windows

package main

import (
	"context"
	"os/signal"
	"syscall"
)

func signalContext() context.Context {
	ctx, _ := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	return ctx
}
