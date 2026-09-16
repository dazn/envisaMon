package main

import (
	"errors"
	"fmt"
	"io"
	"log"
)

type logDestination struct {
	name   string
	writer io.Writer
}

// logFanout attempts each destination in order, even after a write fails.
// Diagnostics use a separate logger so Println callers see failures without
// sending diagnostic messages back through the event logging pipeline.
type logFanout struct {
	destinations []logDestination
	diagnostics  *log.Logger
}

func newLogFanout(stream string, diagnosticWriter io.Writer, destinations ...logDestination) *logFanout {
	return &logFanout{
		destinations: destinations,
		diagnostics:  log.New(diagnosticWriter, "ERROR: "+stream+" logging: ", log.LstdFlags|log.Lmsgprefix),
	}
}

func (w *logFanout) Write(p []byte) (int, error) {
	var failures []error
	for _, destination := range w.destinations {
		n, err := destination.writer.Write(p)
		if err == nil && n != len(p) {
			err = io.ErrShortWrite
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", destination.name, err))
		}
	}
	// Attempt all event destinations before writing best-effort diagnostics.
	// Never include the original event or retry a failed diagnostic write.
	for _, failure := range failures {
		w.diagnostics.Println(failure)
	}
	if err := errors.Join(failures...); err != nil {
		return 0, err
	}
	return len(p), nil
}
