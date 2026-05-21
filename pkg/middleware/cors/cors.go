// Package cors provides a CORS filter for the kratos HTTP transport.
//
// CORS is HTTP-only and needs direct access to the ResponseWriter (for
// preflight short-circuit) and Request.Method, so it ships as a
// `khttp.FilterFunc` rather than a `middleware.Middleware`. Wire it via the
// `Filters` field on `pkg/runtime/server.HTTPConfig`.
//
// Usage:
//
//	hs := pkgserver.NewBizHTTPServer(pkgserver.HTTPConfig{
//	    Network: "tcp",
//	    Addr:    ":8080",
//	    Filters: []khttp.FilterFunc{
//	        cors.New(
//	            cors.WithAllowedOrigins("https://app.example.com"),
//	            cors.WithAllowCredentials(true),
//	        ),
//	    },
//	}, logger, registerFn)
package cors

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// Option configures the CORS filter.
type Option func(*options)

type options struct {
	allowAnyOrigin   bool
	allowedOrigins   map[string]struct{}
	allowedMethods   []string
	allowedHeaders   []string
	exposedHeaders   []string
	allowCredentials bool
	maxAge           time.Duration
}

func defaultOptions() *options {
	return &options{
		allowedOrigins:   map[string]struct{}{},
		allowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions},
		allowedHeaders:   []string{"Content-Type", "Authorization", "X-Trace-Id"},
		exposedHeaders:   []string{"X-Trace-Id"},
		allowCredentials: false,
		maxAge:           10 * time.Minute,
	}
}

// WithAllowedOrigins permits the given origins. Pass "*" to allow any origin.
// "*" cannot be combined with WithAllowCredentials(true) — browsers will
// reject the response. Multiple calls accumulate.
func WithAllowedOrigins(origins ...string) Option {
	return func(o *options) {
		for _, origin := range origins {
			if origin == "*" {
				o.allowAnyOrigin = true
				continue
			}
			o.allowedOrigins[origin] = struct{}{}
		}
	}
}

// WithAllowedMethods overrides the default method list.
func WithAllowedMethods(methods ...string) Option {
	return func(o *options) { o.allowedMethods = append([]string{}, methods...) }
}

// WithAllowedHeaders overrides the default request-header list.
func WithAllowedHeaders(headers ...string) Option {
	return func(o *options) { o.allowedHeaders = append([]string{}, headers...) }
}

// WithExposedHeaders sets the Access-Control-Expose-Headers list.
func WithExposedHeaders(headers ...string) Option {
	return func(o *options) { o.exposedHeaders = append([]string{}, headers...) }
}

// WithAllowCredentials sets Access-Control-Allow-Credentials.
func WithAllowCredentials(b bool) Option { return func(o *options) { o.allowCredentials = b } }

// WithMaxAge sets the preflight cache lifetime.
func WithMaxAge(d time.Duration) Option { return func(o *options) { o.maxAge = d } }

// New returns a khttp.FilterFunc enforcing the configured CORS policy.
//
// Behavior:
//   - Requests without an Origin header pass through unchanged (same-origin
//     or non-browser caller).
//   - Disallowed origins pass through without CORS headers (browsers reject).
//   - OPTIONS preflight short-circuits with 204 + CORS headers; the wrapped
//     handler never runs. This is why CORS belongs at the filter layer:
//     it runs before middlewares like auth that would 401 the OPTIONS.
func New(opts ...Option) khttp.FilterFunc {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	methodsHdr := strings.Join(o.allowedMethods, ", ")
	headersHdr := strings.Join(o.allowedHeaders, ", ")
	exposedHdr := strings.Join(o.exposedHeaders, ", ")
	maxAgeHdr := strconv.Itoa(int(o.maxAge.Seconds()))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}
			if !o.originAllowed(origin) {
				next.ServeHTTP(w, r)
				return
			}

			h := w.Header()
			if o.allowAnyOrigin && !o.allowCredentials {
				h.Set("Access-Control-Allow-Origin", "*")
			} else {
				h.Set("Access-Control-Allow-Origin", origin)
				h.Add("Vary", "Origin")
			}
			if o.allowCredentials {
				h.Set("Access-Control-Allow-Credentials", "true")
			}
			if exposedHdr != "" {
				h.Set("Access-Control-Expose-Headers", exposedHdr)
			}

			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				h.Set("Access-Control-Allow-Methods", methodsHdr)
				h.Set("Access-Control-Allow-Headers", headersHdr)
				if o.maxAge > 0 {
					h.Set("Access-Control-Max-Age", maxAgeHdr)
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (o *options) originAllowed(origin string) bool {
	if o.allowAnyOrigin {
		return true
	}
	_, ok := o.allowedOrigins[origin]
	return ok
}
