package log_test

import (
	"testing"

	"go.uber.org/goleak"
)

// Catch forgotten goroutines from the concurrent json sink test.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
