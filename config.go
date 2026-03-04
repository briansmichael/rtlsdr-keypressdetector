package main

import (
	"fmt"
	"time"
)

const (
	minPressDefault      = 80 * time.Millisecond
	maxPressDefault      = 3000 * time.Millisecond
	seqTimeoutDefault    = 5 * time.Second
	outputDurationDefault = 10 * time.Minute
)

// Config holds every tunable parameter.
type Config struct {
	// SDR
	FrequencyMHz float64
	FrequencyHz  int64
	SampleRate   int
	Gain         string

	// Squelch
	SquelchRMS float64
	Hysteresis float64

	// Key-press detection
	RequiredPresses  int
	MinPressDuration time.Duration
	MaxPressDuration time.Duration
	SequenceTimeout  time.Duration

	// Output
	OutputPin      string
	OutputDuration time.Duration

	// Misc
	Calibrate bool
	Verbose   bool
}

func (c *Config) Validate() error {
	if c.FrequencyMHz <= 0 {
		return fmt.Errorf("frequency must be > 0 MHz")
	}
	if c.SampleRate <= 0 {
		return fmt.Errorf("sample rate must be > 0")
	}
	if c.SquelchRMS < 0 {
		return fmt.Errorf("squelch RMS must be >= 0")
	}
	if c.RequiredPresses < 1 {
		return fmt.Errorf("required presses must be >= 1")
	}
	if c.MinPressDuration >= c.MaxPressDuration {
		return fmt.Errorf("min-press (%v) must be less than max-press (%v)",
			c.MinPressDuration, c.MaxPressDuration)
	}
	if c.OutputPin == "" {
		return fmt.Errorf("output pin must be specified")
	}
	if c.OutputDuration <= 0 {
		return fmt.Errorf("output duration must be > 0")
	}
	return nil
}
