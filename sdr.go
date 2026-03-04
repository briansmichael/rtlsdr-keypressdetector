package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
)

const chunkSamples = 2048 // samples per RMS calculation window

// ChunkHandler is called with each raw PCM chunk (signed 16-bit LE stereo/mono).
type ChunkHandler func(samples []int16)

// SDR manages the rtl_fm subprocess and feeds raw PCM chunks to a handler.
type SDR struct {
	cfg     *Config
	handler ChunkHandler
	logger  *log.Logger
	cmd     *exec.Cmd
}

func NewSDR(cfg *Config, handler ChunkHandler, logger *log.Logger) *SDR {
	return &SDR{cfg: cfg, handler: handler, logger: logger}
}

// Start launches rtl_fm and begins reading PCM in a background goroutine.
func (s *SDR) Start(ctx context.Context) error {
	args := []string{
		"-f", fmt.Sprintf("%d", s.cfg.FrequencyHz),
		"-M", "fm",
		"-s", fmt.Sprintf("%d", s.cfg.SampleRate),
		"-g", s.cfg.Gain,
		"-", // output raw signed-16 PCM to stdout
	}

	s.cmd = exec.CommandContext(ctx, "rtl_fm", args...)
	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("rtl_fm stdout pipe: %w", err)
	}

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("starting rtl_fm: %w – is rtl-sdr installed? (sudo apt install rtl-sdr)", err)
	}
	s.logger.Printf("[SDR] rtl_fm started (PID %d)  freq=%.3f MHz  rate=%d Hz  gain=%s",
		s.cmd.Process.Pid, s.cfg.FrequencyMHz, s.cfg.SampleRate, s.cfg.Gain)

	go s.readLoop(stdout)
	return nil
}

func (s *SDR) Stop() {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
}

func (s *SDR) readLoop(r io.Reader) {
	buf := make([]byte, chunkSamples*2) // 2 bytes per int16 sample
	samples := make([]int16, chunkSamples)

	for {
		_, err := io.ReadFull(r, buf)
		if err != nil {
			// Normal EOF when context cancels rtl_fm
			return
		}
		// Convert little-endian bytes → int16 samples
		for i := 0; i < chunkSamples; i++ {
			samples[i] = int16(buf[i*2]) | int16(buf[i*2+1])<<8
		}
		s.handler(samples)
	}
}
