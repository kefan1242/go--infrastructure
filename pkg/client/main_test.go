package client_test

import (
	"testing"

	"go.uber.org/goleak"
)

// Catch leaked TCP-listener / Accept goroutines from helper-spawned servers.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
