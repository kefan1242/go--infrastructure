package log_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	pkglog "github.com/kris/go-infrastructure/pkg/log"

	"github.com/go-kratos/kratos/v2/log"
)

func TestNewJSON_StdoutVariantReturnsNonNil(t *testing.T) {
	// NewJSON writes to os.Stdout; just verify construction.
	if l := pkglog.NewJSON("x", "v1", "id"); l == nil {
		t.Fatal("NewJSON returned nil")
	}
}

func TestNew_StdoutVariantReturnsNonNil(t *testing.T) {
	if l := pkglog.New("x", "v1", "id"); l == nil {
		t.Fatal("New returned nil")
	}
}

func TestNewJSON_EmitsValidJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := pkglog.NewJSONTo(&buf, "kris-x", "v0.1", "host-1")

	log.NewHelper(logger).Infow("event", "hello", "tenant", "acme")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d:\n%s", len(lines), buf.String())
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("not valid JSON: %v\nbody=%s", err, lines[0])
	}

	for _, want := range []string{"ts", "level", "service.id", "service.name", "service.version", "event"} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing field %q in:\n%+v", want, got)
		}
	}
	if got["service.name"] != "kris-x" {
		t.Errorf("service.name: want kris-x, got %v", got["service.name"])
	}
	if got["tenant"] != "acme" {
		t.Errorf("tenant: want acme, got %v", got["tenant"])
	}
	if got["level"] != "INFO" {
		t.Errorf("level: want INFO, got %v", got["level"])
	}
}

func TestNewJSON_OddKeyvalsPadsMissing(t *testing.T) {
	var buf bytes.Buffer
	logger := pkglog.NewJSONTo(&buf, "x", "v1", "id")

	// Intentionally odd: 3 keyvals — kratos passes them through, jsonSink pads.
	_ = logger.Log(log.LevelWarn, "k1", "v1", "k2_dangling")

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, buf.String())
	}
	if got["k2_dangling"] != "MISSING_VALUE" {
		t.Errorf("expected padded MISSING_VALUE, got %v", got["k2_dangling"])
	}
}

func TestNewJSON_ConcurrentLines(t *testing.T) {
	var buf bytes.Buffer
	logger := pkglog.NewJSONTo(&buf, "x", "v1", "id")
	helper := log.NewHelper(logger)

	const N = 50
	done := make(chan struct{})
	for i := 0; i < N; i++ {
		go func(i int) {
			helper.Infow("i", i)
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < N; i++ {
		<-done
	}

	scanner := bytes.NewReader(buf.Bytes())
	dec := json.NewDecoder(scanner)
	count := 0
	for {
		var obj map[string]any
		if err := dec.Decode(&obj); err != nil {
			break
		}
		count++
	}
	if count != N {
		t.Errorf("want %d JSON lines, got %d (output corrupted under contention?)", N, count)
	}
}
