package main

import (
	"fmt"
	"log"
	"math"
	"time"
)

// SquelchEvent signals key-down or key-up.
type SquelchEvent int

const (
	EventKeyDown SquelchEvent = iota // squelch opened
	EventKeyUp                       // squelch closed
)

// EventHandler receives squelch transitions with a timestamp.
type EventHandler func(event SquelchEvent, at time.Time)

// SquelchDetector converts a stream of audio RMS values into open/close events.
type SquelchDetector struct {
	cfg            *Config
	handler        EventHandler
	logger         *log.Logger
	squelchOpen    bool
	thresholdOpen  float64
	thresholdClose float64
}

func NewSquelchDetector(cfg *Config, handler EventHandler, logger *log.Logger) *SquelchDetector {
	return &SquelchDetector{
		cfg:            cfg,
		handler:        handler,
		logger:         logger,
		thresholdOpen:  cfg.SquelchRMS + cfg.Hysteresis,
		thresholdClose: cfg.SquelchRMS - cfg.Hysteresis,
	}
}

// ProcessChunk is called by the SDR read loop for every PCM chunk.
func (sq *SquelchDetector) ProcessChunk(samples []int16) {
	rms := calcRMS(samples)

	if sq.cfg.Calibrate {
		bar := ""
		n := int(rms / 20)
		for i := 0; i < n && i < 60; i++ {
			bar += "#"
		}
		fmt.Printf("\rRMS=%7.1f  %-60s", rms, bar)
		return
	}

	now := time.Now()

	if !sq.squelchOpen && rms > sq.thresholdOpen {
		sq.squelchOpen = true
		if sq.cfg.Verbose {
			sq.logger.Printf("[SQUELCH] OPEN   rms=%.0f", rms)
		}
		sq.handler(EventKeyDown, now)

	} else if sq.squelchOpen && rms < sq.thresholdClose {
		sq.squelchOpen = false
		if sq.cfg.Verbose {
			sq.logger.Printf("[SQUELCH] CLOSED rms=%.0f", rms)
		}
		sq.handler(EventKeyUp, now)
	}
}

// calcRMS computes the root-mean-square of a signed 16-bit sample slice.
func calcRMS(samples []int16) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		f := float64(s)
		sum += f * f
	}
	return math.Sqrt(sum / float64(len(samples)))
}
