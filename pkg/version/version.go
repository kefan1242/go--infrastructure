// Package version exposes a small Info struct services populate at build
// time, plus an HTTP handler that serves it as JSON on the sidecar listener.
//
// Recommended -ldflags (set in your service Makefile or `go build` invocation):
//
//	-ldflags "-X main.Name=kris-alpha \
//	          -X main.Version=$(git describe --tags --always) \
//	          -X main.Commit=$(git rev-parse HEAD) \
//	          -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
//
// Usage in main.go:
//
//	var (
//	    Name      = "kris-alpha"
//	    Version   = "dev"
//	    Commit    = ""
//	    BuildTime = ""
//	)
//
//	info := version.Info{Name: Name, Version: Version, Commit: Commit, BuildTime: BuildTime}
//	oh.S.HandleFunc("/version", version.Handler(info))
package version

import (
	"encoding/json"
	"net/http"
	"runtime"
)

// Info is the service identity payload. `Go` is filled automatically.
type Info struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	BuildTime string `json:"build_time,omitempty"`
	Go        string `json:"go"`
}

// WithGo returns a copy of i with the Go runtime version filled in when empty.
func (i Info) WithGo() Info {
	if i.Go == "" {
		i.Go = runtime.Version()
	}
	return i
}

// Handler returns an http.HandlerFunc that responds with the Info as JSON.
// The payload is serialized once at construction time, not on every request.
func Handler(i Info) http.HandlerFunc {
	body, _ := json.Marshal(i.WithGo())
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}
}
