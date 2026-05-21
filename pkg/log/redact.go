package log

import (
	"net/http"
	"strings"
)

// SensitiveHeaders is the canonical default set of HTTP header names that
// must be masked before any log line or telemetry event can carry their
// values. Names are compared case-insensitively.
//
// Extend per-service via RedactHeaders(h, extra...) — do not mutate this
// slice at runtime (it's the shared default; mutation is a data race).
var SensitiveHeaders = []string{
	"Authorization",
	"Proxy-Authorization",
	"Cookie",
	"Set-Cookie",
	"X-Auth-Token",
	"X-Api-Key",
	"X-Csrf-Token",
}

// RedactionMask replaces sensitive values in log output.
const RedactionMask = "***"

// RedactHeaders returns a copy of h with every value belonging to a sensitive
// key (the union of SensitiveHeaders and extra) replaced by RedactionMask.
// The input is never mutated, so it's safe to pass an *http.Request's headers
// directly.
//
//	masked := pkglog.RedactHeaders(r.Header, "X-Tenant-Token")
//	helper.Infow("inbound", "headers", masked)
//
// Comparison is case-insensitive — matches Go's http.Header semantics.
func RedactHeaders(h http.Header, extra ...string) http.Header {
	if len(h) == 0 {
		return http.Header{}
	}
	sensitive := make(map[string]struct{}, len(SensitiveHeaders)+len(extra))
	for _, k := range SensitiveHeaders {
		sensitive[http.CanonicalHeaderKey(k)] = struct{}{}
	}
	for _, k := range extra {
		sensitive[http.CanonicalHeaderKey(k)] = struct{}{}
	}

	out := make(http.Header, len(h))
	for k, v := range h {
		canon := http.CanonicalHeaderKey(k)
		if _, hit := sensitive[canon]; hit {
			masked := make([]string, len(v))
			for i := range v {
				masked[i] = RedactionMask
			}
			out[canon] = masked
			continue
		}
		// Copy slice (so caller mutations to v don't leak into the result)
		// and canonicalize the key — stable log/JSON shape regardless of how
		// the input map was constructed.
		out[canon] = append([]string(nil), v...)
	}
	return out
}

// RedactValue masks `value` when `key` is sensitive, otherwise returns
// `value` unchanged. Use this when you're emitting a single key/value pair
// directly instead of forwarding the whole header map.
//
//	helper.Infow(pkglog.RedactValue(name, val))
//	// or:
//	logger.Log(log.LevelInfo, "header", name, "value", pkglog.RedactValue(name, val))
func RedactValue(key, value string) string {
	for _, s := range SensitiveHeaders {
		if strings.EqualFold(s, key) {
			return RedactionMask
		}
	}
	return value
}
