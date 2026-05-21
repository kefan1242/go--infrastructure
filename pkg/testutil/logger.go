// Package testutil offers common helpers for service unit tests:
//
//   - in-memory logger to assert on captured log lines
//   - constructor for a context carrying a trace_id
//   - a fake kratos transport so middlewares can be exercised in tests
//
// Only import from _test.go files; production code must not depend on it.
package testutil

import (
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
)

// MemoryLogger captures log lines into an in-memory slice. Concurrency-safe.
//
// NewMemoryLogger() returns both the kratos log.Logger and the *MemoryLogger
// handle for assertions.
type MemoryLogger struct {
	mu     sync.Mutex
	Lines  []string
	Levels []log.Level
}

// NewMemoryLogger returns (kratos logger, capture sink).
//
//	logger, sink := testutil.NewMemoryLogger()
//	uc := biz.NewFooUsecase(..., logger)
//	uc.Bar(ctx, ...)
//	require.Contains(t, sink.LinesText(), "bar trace=")
func NewMemoryLogger() (log.Logger, *MemoryLogger) {
	m := &MemoryLogger{}
	return m, m
}

// Log implements log.Logger.
func (m *MemoryLogger) Log(level log.Level, keyvals ...any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	line := fmt.Sprintf("[%s]", level)
	for i := 0; i+1 < len(keyvals); i += 2 {
		line += fmt.Sprintf(" %v=%v", keyvals[i], keyvals[i+1])
	}
	m.Lines = append(m.Lines, line)
	m.Levels = append(m.Levels, level)
	return nil
}

// LinesText joins all captured log lines.
func (m *MemoryLogger) LinesText() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := ""
	for _, l := range m.Lines {
		out += l + "\n"
	}
	return out
}

// Reset clears captured log lines.
func (m *MemoryLogger) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Lines = nil
	m.Levels = nil
}
