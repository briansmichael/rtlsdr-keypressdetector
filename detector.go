package main

import (
	"log"
	"time"
)

// OutputDriver abstracts GPIO so the detector doesn't depend on hardware directly.
type OutputDriver interface {
	SetHigh() error
	SetLow() error
}

type detectorState int

const (
	stateIdle      detectorState = iota // waiting for first press
	stateKeyDown                        // squelch is open; measuring press duration
	stateKeyUpWait                      // squelch closed; waiting for next press
)

// Detector is the state machine ported 1-to-1 from the Arduino sketch.
type Detector struct {
	cfg    *Config
	output OutputDriver
	logger *log.Logger

	state        detectorState
	pressCount   int
	pressStart   time.Time
	lastPressEnd time.Time

	outputActive bool
	outputEnd    time.Time

	stopCh chan struct{}
}

func NewDetector(cfg *Config, output OutputDriver, logger *log.Logger) *Detector {
	d := &Detector{
		cfg:    cfg,
		output: output,
		logger: logger,
		stopCh: make(chan struct{}),
	}
	go d.timerLoop()
	return d
}

// HandleEvent is called by the SquelchDetector on each state transition.
func (d *Detector) HandleEvent(event SquelchEvent, at time.Time) {
	switch event {
	case EventKeyDown:
		d.onKeyDown(at)
	case EventKeyUp:
		d.onKeyUp(at)
	}
}

func (d *Detector) onKeyDown(at time.Time) {
	if d.state == stateIdle || d.state == stateKeyUpWait {
		d.state = stateKeyDown
		d.pressStart = at
		if d.cfg.Verbose {
			d.logger.Println("[DETECTOR] Key DOWN")
		}
	}
}

func (d *Detector) onKeyUp(at time.Time) {
	if d.state != stateKeyDown {
		return
	}
	duration := at.Sub(d.pressStart)

	if duration < d.cfg.MinPressDuration || duration > d.cfg.MaxPressDuration {
		d.logger.Printf("[DETECTOR] Key UP – ignored (duration %v out of range)", duration)
		if d.pressCount > 0 {
			d.state = stateKeyUpWait
		} else {
			d.state = stateIdle
		}
		return
	}

	// Valid press
	d.pressCount++
	d.lastPressEnd = at
	d.state = stateKeyUpWait

	d.logger.Printf("[DETECTOR] Valid press #%d  (duration %v)",
		d.pressCount, duration.Round(time.Millisecond))

	if d.pressCount >= d.cfg.RequiredPresses {
		d.triggerOutput()
		d.pressCount = 0
		d.state = stateIdle
	}
}

func (d *Detector) triggerOutput() {
	d.logger.Println("")
	d.logger.Printf("**** %d KEY PRESSES DETECTED ****", d.cfg.RequiredPresses)
	d.logger.Printf("**** OUTPUT PIN %s ACTIVE for %v ****",
		d.cfg.OutputPin, d.cfg.OutputDuration)
	d.logger.Println("")

	if err := d.output.SetHigh(); err != nil {
		d.logger.Printf("[DETECTOR] ERROR setting output HIGH: %v", err)
		return
	}
	d.outputActive = true
	d.outputEnd = time.Now().Add(d.cfg.OutputDuration)
}

// timerLoop runs in its own goroutine and handles two periodic checks:
//   - sequence timeout (no press within SequenceTimeout → reset counter)
//   - output timer    (deactivate output after OutputDuration)
func (d *Detector) timerLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-d.stopCh:
			_ = d.output.SetLow()
			return

		case now := <-ticker.C:
			// Sequence timeout
			if d.pressCount > 0 && d.state != stateKeyDown {
				if now.Sub(d.lastPressEnd) > d.cfg.SequenceTimeout {
					d.logger.Printf("[DETECTOR] Sequence timeout – reset (had %d press(es))",
						d.pressCount)
					d.pressCount = 0
					d.state = stateIdle
				}
			}
			// Output timer
			if d.outputActive && now.After(d.outputEnd) {
				if err := d.output.SetLow(); err != nil {
					d.logger.Printf("[DETECTOR] ERROR setting output LOW: %v", err)
				}
				d.outputActive = false
				d.logger.Println("[DETECTOR] Output deactivated – timer expired")
			}
		}
	}
}

func (d *Detector) Stop() {
	close(d.stopCh)
}
