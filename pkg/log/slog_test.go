package log_test

import (
	"log/slog"
	"strings"
	"testing"

	pkglog "github.com/kris/go-infrastructure/pkg/log"

	"github.com/go-kratos/kratos/v2/log"
)

// memSink captures records by level + flattened keyvals.
type memSink struct{ lines []string }

func (m *memSink) Log(level log.Level, keyvals ...any) error {
	parts := []string{level.String()}
	for i := 0; i+1 < len(keyvals); i += 2 {
		parts = append(parts, asString(keyvals[i])+"="+asString(keyvals[i+1]))
	}
	m.lines = append(m.lines, strings.Join(parts, " "))
	return nil
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return slog.AnyValue(v).String()
}

func TestSlogHandler_ForwardsMessageAndAttrs(t *testing.T) {
	sink := &memSink{}
	logger := slog.New(pkglog.SlogHandler(sink))

	logger.Info("hello", "tenant", "acme", "n", 42)

	if len(sink.lines) != 1 {
		t.Fatalf("want 1 line, got %d: %v", len(sink.lines), sink.lines)
	}
	line := sink.lines[0]
	for _, want := range []string{"INFO", "msg=hello", "tenant=acme", "n=42"} {
		if !strings.Contains(line, want) {
			t.Errorf("missing %q in %q", want, line)
		}
	}
}

func TestSlogHandler_LevelMapping(t *testing.T) {
	cases := []struct {
		fn   func(*slog.Logger, string, ...any)
		want string
	}{
		{(*slog.Logger).Debug, "DEBUG"},
		{(*slog.Logger).Info, "INFO"},
		{(*slog.Logger).Warn, "WARN"},
		{(*slog.Logger).Error, "ERROR"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			sink := &memSink{}
			h := pkglog.SlogHandler(sink)
			// DEBUG is below default slog threshold; force-enable by routing
			// through a handler that doesn't filter.
			logger := slog.New(h)
			tc.fn(logger, "msg")
			if len(sink.lines) != 1 {
				t.Fatalf("want 1 line, got %d", len(sink.lines))
			}
			if !strings.HasPrefix(sink.lines[0], tc.want) {
				t.Errorf("level mismatch: want prefix %s, got %q", tc.want, sink.lines[0])
			}
		})
	}
}

func TestSlogHandler_WithAttrsPersists(t *testing.T) {
	sink := &memSink{}
	base := slog.New(pkglog.SlogHandler(sink))
	child := base.With("service", "kris-x", "build", "v1")

	child.Info("event")
	child.Info("other")

	if len(sink.lines) != 2 {
		t.Fatalf("want 2 lines, got %d", len(sink.lines))
	}
	for i, line := range sink.lines {
		if !strings.Contains(line, "service=kris-x") {
			t.Errorf("line %d missing service=kris-x: %q", i, line)
		}
		if !strings.Contains(line, "build=v1") {
			t.Errorf("line %d missing build=v1: %q", i, line)
		}
	}
}

func TestSlogHandler_WithGroupPrefixesKeys(t *testing.T) {
	sink := &memSink{}
	base := slog.New(pkglog.SlogHandler(sink))
	grouped := base.WithGroup("req")

	grouped.Info("hit", "path", "/v1/foo", "method", "GET")

	line := sink.lines[0]
	if !strings.Contains(line, "req.path=/v1/foo") {
		t.Errorf("expected grouped key req.path, got %q", line)
	}
	if !strings.Contains(line, "req.method=GET") {
		t.Errorf("expected grouped key req.method, got %q", line)
	}
}

func TestSlogHandler_NestedGroups(t *testing.T) {
	sink := &memSink{}
	base := slog.New(pkglog.SlogHandler(sink))
	deep := base.WithGroup("outer").WithGroup("inner")

	deep.Info("event", "k", "v")

	if !strings.Contains(sink.lines[0], "outer.inner.k=v") {
		t.Errorf("expected nested prefix outer.inner.k, got %q", sink.lines[0])
	}
}

func TestSlogHandler_EmptyWithGroupIsNoop(t *testing.T) {
	sink := &memSink{}
	base := slog.New(pkglog.SlogHandler(sink))
	same := base.WithGroup("")

	same.Info("event", "k", "v")

	if !strings.Contains(sink.lines[0], "k=v") || strings.Contains(sink.lines[0], ".k=") {
		t.Errorf("empty group should not add prefix, got %q", sink.lines[0])
	}
}
