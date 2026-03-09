package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := &Config{}

	flag.Float64Var(&cfg.FrequencyMHz,    "freq",            122.725,               "Receive frequency in MHz")
	flag.IntVar(&cfg.SampleRate,          "rate",            12500,                 "RTL-SDR sample rate in Hz")
	flag.StringVar(&cfg.Gain,             "gain",            "40",                  `SDR gain in dB or "auto"`)
	flag.Float64Var(&cfg.SquelchRMS,      "squelch",         200.0,                 "Audio RMS squelch threshold (0–32767)")
	flag.Float64Var(&cfg.Hysteresis,      "hysteresis",      40.0,                  "Hysteresis applied to squelch threshold")
	flag.IntVar(&cfg.RequiredPresses,     "presses",         5,                     "Sequential key presses required")
	flag.DurationVar(&cfg.MinPressDuration,"min-press",      minPressDefault,       "Minimum valid press duration")
	flag.DurationVar(&cfg.MaxPressDuration,"max-press",      maxPressDefault,       "Maximum valid press duration")
	flag.DurationVar(&cfg.SequenceTimeout, "seq-timeout",    seqTimeoutDefault,     "Inter-press timeout before reset")
	flag.StringVar(&cfg.OutputPin,        "pin",             "GPIO8",               "BCM GPIO pin for output (e.g. GPIO8 or 8)")
	flag.DurationVar(&cfg.OutputDuration, "output-duration", outputDurationDefault, "Duration to hold output HIGH")
	flag.BoolVar(&cfg.Calibrate,          "calibrate",       false,                 "Calibration mode: print RMS values, no GPIO")
	flag.BoolVar(&cfg.Verbose,            "verbose",         false,                 "Verbose logging")

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, `
rtlsdr-keypressdetector — detect N sequential PTT presses on a
given frequency and drive a GPIO pin for a configurable duration.

Usage:
  ./keypressdetector [flags]

Flags:`)
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, `
Examples:
  ./keypressdetector                              # defaults
  ./keypressdetector -calibrate                  # tune squelch
  ./keypressdetector -freq 121.500 -gain auto -verbose
  ./keypressdetector -presses 3 -output-duration 2m -pin GPIO17

Cross-compile for Raspberry Pi 4/5:
  GOOS=linux GOARCH=arm64 go build -o keypressdetector .

Cross-compile for Pi Zero / 1:
  GOOS=linux GOARCH=arm GOARM=6 go build -o keypressdetector .`)
	}
	flag.Parse()

	cfg.FrequencyHz = int64(cfg.FrequencyMHz * 1e6)

	logger := log.New(os.Stdout, "", log.LstdFlags)

	if err := cfg.Validate(); err != nil {
		logger.Fatalf("Config error: %v", err)
	}

	logger.Println("==============================================")
	logger.Println("  RTL-SDR Sequential Key-Press Detector")
	logger.Println("==============================================")
	logger.Printf("  Frequency        : %.3f MHz", cfg.FrequencyMHz)
	logger.Printf("  Squelch RMS      : %.0f  (±%.0f)", cfg.SquelchRMS, cfg.Hysteresis)
	logger.Printf("  Required presses : %d", cfg.RequiredPresses)
	logger.Printf("  Press window     : %v – %v", cfg.MinPressDuration, cfg.MaxPressDuration)
	logger.Printf("  Sequence timeout : %v", cfg.SequenceTimeout)
	logger.Printf("  Output pin       : %s  for %v", cfg.OutputPin, cfg.OutputDuration)
	logger.Println("==============================================")

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer stop()

	app := NewApp(cfg, logger)
	if err := app.Run(ctx); err != nil {
		logger.Fatalf("Fatal: %v", err)
	}
	logger.Println("Shutdown complete.")
}
