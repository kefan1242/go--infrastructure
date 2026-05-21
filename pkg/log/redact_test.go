package log_test

import (
	"net/http"
	"reflect"
	"testing"

	pkglog "github.com/kris/go-infrastructure/pkg/log"
)

func TestRedactHeaders_DefaultsMaskAuthorization(t *testing.T) {
	in := http.Header{
		"Authorization": []string{"Bearer secret-xyz"},
		"Content-Type":  []string{"application/json"},
	}
	got := pkglog.RedactHeaders(in)

	if got.Get("Authorization") != pkglog.RedactionMask {
		t.Errorf("Authorization: want %q, got %q", pkglog.RedactionMask, got.Get("Authorization"))
	}
	if got.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type: want application/json, got %q", got.Get("Content-Type"))
	}
	// Caller's input must not be touched.
	if in.Get("Authorization") != "Bearer secret-xyz" {
		t.Error("input header was mutated")
	}
}

func TestRedactHeaders_CaseInsensitive(t *testing.T) {
	in := http.Header{
		"authorization": []string{"Bearer x"},
		"COOKIE":        []string{"session=abc"},
	}
	got := pkglog.RedactHeaders(in)

	if got.Get("authorization") != pkglog.RedactionMask {
		t.Errorf("lowercase Authorization should mask, got %q", got.Get("authorization"))
	}
	if got.Get("Cookie") != pkglog.RedactionMask {
		t.Errorf("UPPERCASE Cookie should mask, got %q", got.Get("Cookie"))
	}
}

func TestRedactHeaders_MaskMultipleValues(t *testing.T) {
	// Set-Cookie commonly has multiple values per response.
	in := http.Header{
		"Set-Cookie": []string{"a=1; Path=/", "b=2; Path=/"},
	}
	got := pkglog.RedactHeaders(in)
	vs := got.Values("Set-Cookie")
	if len(vs) != 2 {
		t.Fatalf("want 2 masked entries, got %d", len(vs))
	}
	for i, v := range vs {
		if v != pkglog.RedactionMask {
			t.Errorf("Set-Cookie[%d]: want mask, got %q", i, v)
		}
	}
}

func TestRedactHeaders_ExtraKeysMasked(t *testing.T) {
	in := http.Header{
		"X-Tenant-Token": []string{"tnt-abc"},
		"X-Visible":      []string{"safe"},
	}
	got := pkglog.RedactHeaders(in, "X-Tenant-Token")

	if got.Get("X-Tenant-Token") != pkglog.RedactionMask {
		t.Errorf("extra-listed header should mask, got %q", got.Get("X-Tenant-Token"))
	}
	if got.Get("X-Visible") != "safe" {
		t.Errorf("non-listed should pass through, got %q", got.Get("X-Visible"))
	}
}

func TestRedactHeaders_EmptyInputReturnsEmpty(t *testing.T) {
	got := pkglog.RedactHeaders(nil)
	if got == nil {
		t.Fatal("expected non-nil empty header for stable JSON output")
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestRedactHeaders_CopyIsIndependent(t *testing.T) {
	in := http.Header{"Content-Type": []string{"application/json"}}
	got := pkglog.RedactHeaders(in)

	// Mutating the copy must not affect the input.
	got["Content-Type"] = []string{"text/plain"}
	if in.Get("Content-Type") != "application/json" {
		t.Error("mutation on copy leaked to caller's input")
	}
}

func TestRedactValue(t *testing.T) {
	cases := []struct {
		key, val, want string
	}{
		{"Authorization", "Bearer x", pkglog.RedactionMask},
		{"authorization", "Bearer y", pkglog.RedactionMask}, // case-insensitive
		{"Content-Type", "application/json", "application/json"},
		{"X-Trace-Id", "abc-123", "abc-123"},
	}
	for _, tc := range cases {
		if got := pkglog.RedactValue(tc.key, tc.val); got != tc.want {
			t.Errorf("RedactValue(%q,%q): want %q, got %q", tc.key, tc.val, tc.want, got)
		}
	}
}

func TestSensitiveHeaders_ContainsCanonicals(t *testing.T) {
	// Guards against accidental edits to the default list.
	want := []string{"Authorization", "Cookie", "Set-Cookie"}
	have := map[string]bool{}
	for _, k := range pkglog.SensitiveHeaders {
		have[k] = true
	}
	for _, k := range want {
		if !have[k] {
			t.Errorf("SensitiveHeaders missing %q (full list: %v)", k, pkglog.SensitiveHeaders)
		}
	}
	_ = reflect.DeepEqual // imported but kept for future contains-check
}
