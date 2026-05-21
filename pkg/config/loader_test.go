package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kris/go-infrastructure/pkg/config"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// withCwd cd's into dir for the duration of t, restoring afterwards.
func withCwd(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

func TestLoader_LoadsFromServiceEnv(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.work"), "")
	writeFile(t, filepath.Join(root, "kris-svc", ".env"), "MY_VAR=from-service\n")

	svc := filepath.Join(root, "kris-svc")
	withCwd(t, svc)
	_ = os.Unsetenv("MY_VAR")
	t.Cleanup(func() { _ = os.Unsetenv("MY_VAR") })

	loader := config.NewLoader("kris-svc")
	if err := loader.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	if v := os.Getenv("MY_VAR"); v != "from-service" {
		t.Errorf("MY_VAR: want from-service, got %q", v)
	}
}

func TestLoader_GlobalThenServiceOverridesNot(t *testing.T) {
	// Verify the "don't overwrite already-set" rule: loading global then
	// service should NOT overwrite the value seeded by global, because
	// loadDotEnv treats existing env vars as authoritative.
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.work"), "")
	writeFile(t, filepath.Join(root, ".env.global"), "SHARED=from-global\n")
	writeFile(t, filepath.Join(root, "kris-svc", ".env"), "SHARED=from-service\n")

	svc := filepath.Join(root, "kris-svc")
	withCwd(t, svc)
	t.Setenv("SHARED", "")
	_ = os.Unsetenv("SHARED")

	loader := config.NewLoader("kris-svc")
	if err := loader.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	// Global is loaded first, sets SHARED=from-global. Service load won't
	// overwrite because loadDotEnv preserves existing env vars.
	if v := os.Getenv("SHARED"); v != "from-global" {
		t.Errorf("SHARED: want from-global (set first wins), got %q", v)
	}
}

func TestLoader_EnvVarBeatsDotEnv(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.work"), "")
	writeFile(t, filepath.Join(root, "kris-svc", ".env"), "PRESET=from-file\n")

	svc := filepath.Join(root, "kris-svc")
	withCwd(t, svc)
	t.Setenv("PRESET", "from-env")

	loader := config.NewLoader("kris-svc")
	if err := loader.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	if v := os.Getenv("PRESET"); v != "from-env" {
		t.Errorf("PRESET: want from-env, got %q", v)
	}
}

func TestLoadDotEnv_HandlesCommentsAndQuotes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	writeFile(t, path, "# a comment\n\nKEY1=\"quoted-val\"\nKEY2=plain\n\n# trailing\n")

	t.Setenv("KEY1", "")
	t.Setenv("KEY2", "")
	_ = os.Unsetenv("KEY1")
	_ = os.Unsetenv("KEY2")

	// Indirectly drive loadDotEnv through Loader.Load by pointing cwd at dir.
	writeFile(t, filepath.Join(dir, "go.work"), "")
	withCwd(t, dir)

	if err := config.NewLoader("svc-nope").Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	if v := os.Getenv("KEY1"); v != "quoted-val" {
		t.Errorf("KEY1: want quoted-val, got %q", v)
	}
	if v := os.Getenv("KEY2"); v != "plain" {
		t.Errorf("KEY2: want plain, got %q", v)
	}
}

func TestGetEnv_Default(t *testing.T) {
	_ = os.Unsetenv("UNSET_VAR_XYZ")
	if got := config.GetEnv("UNSET_VAR_XYZ", "fallback"); got != "fallback" {
		t.Errorf("default: got %s", got)
	}
	t.Setenv("UNSET_VAR_XYZ", "real")
	if got := config.GetEnv("UNSET_VAR_XYZ", "fallback"); got != "real" {
		t.Errorf("set: got %s", got)
	}
}

func TestGetEnvBool(t *testing.T) {
	_ = os.Unsetenv("B_KEY")
	if !config.GetEnvBool("B_KEY", true) {
		t.Error("default true should win")
	}
	t.Setenv("B_KEY", "false")
	if config.GetEnvBool("B_KEY", true) {
		t.Error("explicit false should override")
	}
	t.Setenv("B_KEY", "not-a-bool")
	if !config.GetEnvBool("B_KEY", true) {
		t.Error("unparseable should fall back to default")
	}
}

func TestGetEnvDuration(t *testing.T) {
	_ = os.Unsetenv("D_KEY")
	if got := config.GetEnvDuration("D_KEY", 3*time.Second); got != 3*time.Second {
		t.Errorf("default: got %v", got)
	}
	t.Setenv("D_KEY", "100ms")
	if got := config.GetEnvDuration("D_KEY", time.Second); got != 100*time.Millisecond {
		t.Errorf("parsed: got %v", got)
	}
	t.Setenv("D_KEY", "abc")
	if got := config.GetEnvDuration("D_KEY", time.Second); got != time.Second {
		t.Errorf("invalid: want fallback 1s, got %v", got)
	}
}
