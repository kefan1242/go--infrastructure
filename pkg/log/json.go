package log

import (
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
)

// NewJSON returns a kratos logger that emits one JSON object per line,
// seeded with service identity. Use this in production when a log shipper
// expects structured input; use New for human-friendly key=value output.
//
// Fields baked in: ts (RFC3339), level, caller, service.{id,name,version}.
// Additional key/value pairs passed via Helper.Infow / Warnw become siblings.
func NewJSON(serviceName, version, instanceID string) log.Logger {
	return log.With(&jsonSink{w: os.Stdout},
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", instanceID,
		"service.name", serviceName,
		"service.version", version,
	)
}

// NewJSONTo is like NewJSON but lets callers swap the writer (handy in tests).
func NewJSONTo(w io.Writer, serviceName, version, instanceID string) log.Logger {
	return log.With(&jsonSink{w: w},
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", instanceID,
		"service.name", serviceName,
		"service.version", version,
	)
}

type jsonSink struct {
	mu sync.Mutex
	w  io.Writer
}

// Log implements log.Logger. Valuers in keyvals are already resolved by the
// kratos `log.With` wrapper before it dispatches here.
func (s *jsonSink) Log(level log.Level, keyvals ...any) error {
	if len(keyvals) == 0 {
		return nil
	}
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, "MISSING_VALUE")
	}
	obj := make(map[string]any, len(keyvals)/2+1)
	obj["level"] = level.String()
	for i := 0; i < len(keyvals); i += 2 {
		k, ok := keyvals[i].(string)
		if !ok {
			continue
		}
		obj[k] = keyvals[i+1]
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.NewEncoder(s.w).Encode(obj)
}
