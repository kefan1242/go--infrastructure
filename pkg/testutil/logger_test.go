package testutil_test

import (
	"strings"
	"testing"

	"github.com/kris/go-infrastructure/pkg/testutil"

	"github.com/go-kratos/kratos/v2/log"
)

func TestMemoryLogger_Capture(t *testing.T) {
	logger, sink := testutil.NewMemoryLogger()

	helper := log.NewHelper(log.With(logger, "module", "demo"))
	helper.Infof("hello %s", "world")
	helper.Warnw("k", "v")

	if got := sink.LinesText(); !strings.Contains(got, "hello world") {
		t.Fatalf("expected hello world; got:\n%s", got)
	}
	if len(sink.Lines) != 2 {
		t.Fatalf("expected 2 lines; got %d", len(sink.Lines))
	}

	sink.Reset()
	if len(sink.Lines) != 0 {
		t.Fatalf("expected reset; got %d lines", len(sink.Lines))
	}
}
