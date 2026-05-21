package log

import (
	"context"
	"log/slog"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
)

// SlogHandler returns an slog.Handler that forwards records to the given
// kratos logger. Useful when a third-party library accepts a slog.Logger /
// slog.Handler (Go 1.21+ stdlib pattern) but you want the output to flow
// through the kratos pipeline (and thus into pkg/log's JSON / pretty sinks).
//
//	logger := pkglog.NewJSON("kris-x", "v0.1", id)
//	slogLogger := slog.New(pkglog.SlogHandler(logger))
//	// pass slogLogger to libraries that take *slog.Logger
func SlogHandler(logger log.Logger) slog.Handler {
	return &slogHandler{logger: logger}
}

type slogHandler struct {
	logger log.Logger
	attrs  []any    // pre-bound key/value pairs from WithAttrs
	groups []string // active group stack for key prefixing
}

// Enabled defers to kratos's own level filtering — kratos's Logger interface
// doesn't expose a query method, so we forward every record.
func (h *slogHandler) Enabled(context.Context, slog.Level) bool { return true }

// Handle converts an slog.Record into a kratos Log call. The kratos level is
// derived from the slog level; msg, attrs, and any pre-bound attrs from
// WithAttrs are flattened into keyvals.
func (h *slogHandler) Handle(_ context.Context, r slog.Record) error {
	level := slogToKratos(r.Level)
	kv := make([]any, 0, 2+len(h.attrs)+r.NumAttrs()*2)
	kv = append(kv, "msg", r.Message)
	kv = append(kv, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		kv = append(kv, h.prefix(a.Key), a.Value.Any())
		return true
	})
	return h.logger.Log(level, kv...)
}

// WithAttrs returns a child handler with the attrs pre-bound to every record.
func (h *slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cp := *h
	cp.attrs = append([]any{}, h.attrs...)
	for _, a := range attrs {
		cp.attrs = append(cp.attrs, h.prefix(a.Key), a.Value.Any())
	}
	return &cp
}

// WithGroup returns a child handler that prefixes subsequent keys with
// `<group>.`.
func (h *slogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	cp := *h
	cp.groups = append(append([]string{}, h.groups...), name)
	return &cp
}

func (h *slogHandler) prefix(key string) string {
	if len(h.groups) == 0 {
		return key
	}
	return strings.Join(h.groups, ".") + "." + key
}

// slogToKratos maps slog levels to the closest kratos level.
//
//	slog DEBUG (-4) -> kratos DEBUG
//	slog INFO  (0)  -> kratos INFO
//	slog WARN  (4)  -> kratos WARN
//	slog ERROR (8)  -> kratos ERROR
//
// Custom levels between thresholds round down to the lower kratos level.
func slogToKratos(l slog.Level) log.Level {
	switch {
	case l < slog.LevelInfo:
		return log.LevelDebug
	case l < slog.LevelWarn:
		return log.LevelInfo
	case l < slog.LevelError:
		return log.LevelWarn
	default:
		return log.LevelError
	}
}
