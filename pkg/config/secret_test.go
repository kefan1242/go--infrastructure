package config_test

import (
	"os"
	"strings"
	"testing"

	"github.com/kris/go-infrastructure/pkg/config"
)

func TestMustGetSecret_ReturnsValueWhenSet(t *testing.T) {
	const key = "KRIS_TEST_SECRET_FOO"
	t.Setenv(key, "shh")
	got := config.MustGetSecret(key)
	if got != "shh" {
		t.Errorf("want shh, got %q", got)
	}
}

func TestMustGetSecret_PanicsWhenUnset(t *testing.T) {
	const key = "KRIS_TEST_SECRET_UNSET"
	_ = os.Unsetenv(key)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on missing secret")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("panic value should be string, got %T: %v", r, r)
		}
		if !strings.Contains(msg, key) {
			t.Errorf("panic message should mention the key, got %q", msg)
		}
	}()
	_ = config.MustGetSecret(key)
}

func TestMustGetSecret_PanicsWhenEmpty(t *testing.T) {
	const key = "KRIS_TEST_SECRET_EMPTY"
	t.Setenv(key, "")

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on empty secret")
		}
	}()
	_ = config.MustGetSecret(key)
}

func TestGetSecret_ReturnsEmptyOnUnset(t *testing.T) {
	const key = "KRIS_TEST_SECRET_OPTIONAL"
	_ = os.Unsetenv(key)
	if got := config.GetSecret(key); got != "" {
		t.Errorf("want empty, got %q", got)
	}
}
