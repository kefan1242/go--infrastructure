package version_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/kris/go-infrastructure/pkg/version"
)

func TestInfo_WithGoFillsRuntime(t *testing.T) {
	got := version.Info{Name: "x", Version: "v1"}.WithGo()
	if got.Go != runtime.Version() {
		t.Errorf("Go: want %s, got %s", runtime.Version(), got.Go)
	}
}

func TestInfo_WithGoPreservesExplicit(t *testing.T) {
	got := version.Info{Go: "custom"}.WithGo()
	if got.Go != "custom" {
		t.Errorf("Go: want custom, got %s", got.Go)
	}
}

func TestHandler_ReturnsJSON(t *testing.T) {
	h := version.Handler(version.Info{
		Name:      "kris-x",
		Version:   "v0.1.0",
		Commit:    "abc",
		BuildTime: "2026-01-01T00:00:00Z",
	})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/version", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type: want application/json, got %q", ct)
	}
	var got version.Info
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Name != "kris-x" || got.Version != "v0.1.0" || got.Commit != "abc" {
		t.Errorf("payload: %+v", got)
	}
	if got.Go == "" {
		t.Error("Go runtime should be filled")
	}
}

func TestHandler_OmitsEmptyCommitAndBuildTime(t *testing.T) {
	h := version.Handler(version.Info{Name: "x", Version: "dev"})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/version", nil))

	body := w.Body.String()
	if strings.Contains(body, `"commit"`) {
		t.Errorf("commit should be omitempty, got %s", body)
	}
	if strings.Contains(body, `"build_time"`) {
		t.Errorf("build_time should be omitempty, got %s", body)
	}
}
