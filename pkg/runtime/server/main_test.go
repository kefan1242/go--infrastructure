package server_test

import (
	"testing"

	"go.uber.org/goleak"
)

// Verify kratos HTTP servers + accept loops shut down cleanly via t.Cleanup.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
