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
	// In calibration mode, skip GPIO entirely and just run the SDR pipeline
	var gpio OutputDriver
	if !a.cfg.Calibrate {
		pin, err := NewGPIOPin(a.cfg.OutputPin, a.logger)
		if err != nil {
			return err
		}
		defer pin.Close()
		gpio = pin
	} else {
		gpio = &noopOutput{}
		a.logger.Println("[GPIO] Calibration mode – GPIO disabled")
	}

	detector := NewDetector(a.cfg, gpio, a.logger)
	squelch := NewSquelchDetector(a.cfg, detector.HandleEvent, a.logger)
	sdr := NewSDR(a.cfg, squelch.ProcessChunk, a.logger)

	a.logger.Println("Starting SDR pipeline …")
	if err := sdr.Start(ctx); err != nil {
		return err
	}

	<-ctx.Done()
	a.logger.Println("Shutdown signal received, stopping …")
	sdr.Stop()
	detector.Stop()
	return nil
}

// noopOutput satisfies OutputDriver without touching any hardware.
type noopOutput struct{}
func (n *noopOutput) SetHigh() error { return nil }
func (n *noopOutput) SetLow() error  { return nil }
