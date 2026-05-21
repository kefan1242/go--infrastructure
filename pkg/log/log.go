// Package log provides a standard kratos logger seeded with service metadata.
package log

import (
	"os"

	"github.com/go-kratos/kratos/v2/log"
)

// New returns a kratos logger annotated with the service name, version and
// instance id. Wrap or replace at the project boundary when wiring in a
// centralized logging backend (remote shipper / structured fields / etc).
func New(serviceName, version, instanceID string) log.Logger {
	return log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", instanceID,
		"service.name", serviceName,
		"service.version", version,
	)
}
