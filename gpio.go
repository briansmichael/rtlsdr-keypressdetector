package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// GPIOPin drives a single GPIO pin via the Linux sysfs interface.
// This works on every Raspberry Pi without any external library.
//
// The pin identifier is the BCM GPIO number, e.g. "8" or "GPIO8".
//
// For production use you can swap this implementation for
// github.com/warthog618/gpiod (character-device API) without
// changing any other file – just keep the OutputDriver interface.

type GPIOPin struct {
	number    int
	logger    *log.Logger
	valueFile *os.File
}

func NewGPIOPin(pinName string, logger *log.Logger) (*GPIOPin, error) {
	num, err := parsePinNumber(pinName)
	if err != nil {
		return nil, fmt.Errorf("invalid pin %q: %w", pinName, err)
	}

	g := &GPIOPin{number: num, logger: logger}

	if err := g.export(); err != nil {
		return nil, err
	}
	if err := g.setDirection("out"); err != nil {
		return nil, err
	}
	// Open the value file once and keep it open for fast writes
	vf, err := os.OpenFile(
		fmt.Sprintf("/sys/class/gpio/gpio%d/value", num),
		os.O_WRONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("opening gpio%d value: %w", num, err)
	}
	g.valueFile = vf

	// Ensure pin starts LOW
	if err := g.SetLow(); err != nil {
		return nil, err
	}

	logger.Printf("[GPIO] Pin GPIO%d initialised (sysfs)", num)
	return g, nil
}

func (g *GPIOPin) SetHigh() error {
	return g.write("1")
}

func (g *GPIOPin) SetLow() error {
	return g.write("0")
}

func (g *GPIOPin) Close() {
	_ = g.SetLow()
	if g.valueFile != nil {
		_ = g.valueFile.Close()
	}
	_ = g.unexport()
}

// ── helpers ──────────────────────────────────────────────────

func (g *GPIOPin) write(val string) error {
	if _, err := g.valueFile.WriteAt([]byte(val), 0); err != nil {
		return fmt.Errorf("gpio%d write %s: %w", g.number, val, err)
	}
	return nil
}

func (g *GPIOPin) export() error {
	// Unexport first in case it was left exported by a previous run
	_ = writeFile("/sys/class/gpio/unexport", strconv.Itoa(g.number))
	return writeFile("/sys/class/gpio/export", strconv.Itoa(g.number))
}

func (g *GPIOPin) unexport() error {
	return writeFile("/sys/class/gpio/unexport",
		strconv.Itoa(g.number))
}

func (g *GPIOPin) setDirection(dir string) error {
	return writeFile(
		fmt.Sprintf("/sys/class/gpio/gpio%d/direction", g.number),
		dir)
}

func writeFile(path, value string) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	_, err = f.WriteString(value)
	return err
}

// parsePinNumber accepts "8", "GPIO8", "gpio8".
func parsePinNumber(s string) (int, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(strings.ToLower(s), "gpio")
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("expected integer GPIO number, got %q", s)
	}
	if n < 0 || n > 53 {
		return 0, fmt.Errorf("GPIO number %d out of range (0–53)", n)
	}
	return n, nil
}
