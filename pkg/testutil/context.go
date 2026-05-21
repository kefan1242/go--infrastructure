package testutil

import (
	"context"

	"github.com/kris/go-infrastructure/pkg/middleware/logid"
)

// NewContextWithTraceID returns a test ctx pre-populated with a trace_id.
// Production code reading logid.FromContext(ctx) will see this id.
func NewContextWithTraceID(id string) context.Context {
	return logid.NewContext(context.Background(), id)
}
