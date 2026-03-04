package main

import (
	"context"
	"log"
)

// App wires every subsystem together.
type App struct {
	cfg    *Config
	logger *log.Logger
}

func NewApp(cfg *Config, logger *log.Logger) *App {
	return &App{cfg: cfg, logger: logger}
}

// Run blocks until ctx is cancelled or a fatal error occurs.
func (a *App) Run(ctx context.Context) error {
	// 1. GPIO output driver
	gpio, err := NewGPIOPin(a.cfg.OutputPin, a.logger)
	if err != nil {
		return err
	}
	defer gpio.Close()

	// 2. Key-press detector (state machine)
	detector := NewDetector(a.cfg, gpio, a.logger)

	// 3. Squelch detector — feeds events into the state machine
	squelch := NewSquelchDetector(a.cfg, detector.HandleEvent, a.logger)

	// 4. RTL-FM source — streams raw PCM into the squelch detector
	sdr := NewSDR(a.cfg, squelch.ProcessChunk, a.logger)

	a.logger.Println("Starting SDR pipeline …")
	if err := sdr.Start(ctx); err != nil {
		return err
	}

	// Block until context cancelled
	<-ctx.Done()
	a.logger.Println("Shutdown signal received, stopping …")
	sdr.Stop()
	detector.Stop()
	return nil
}
