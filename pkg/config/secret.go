package config

import (
	"fmt"
	"os"
)

// MustGetSecret reads `key` from the environment and panics if it's missing
// or empty. Use for credentials whose absence means "this deploy is
// misconfigured, do not start" — much better than silently using an empty
// password and authentication-failing on every request after boot.
//
// Recommended usage in main:
//
//	cfg := data.MySQLConfig{
//	    DSN: config.MustGetSecret("MYSQL_DSN"),
//	}
//
// The k8s deployment must populate MYSQL_DSN via envFrom: secretRef.
// If the secret is missing, the pod CrashLoopBackOffs immediately —
// surfacing a clear "secret not mounted" signal instead of a confusing
// runtime auth error.
func MustGetSecret(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("config: required secret %q is unset — refusing to start", key))
	}
	return v
}

// GetSecret is the non-panicking variant. Returns the empty string when
// the env var is unset; prefer MustGetSecret unless the caller can run
// degraded without the secret.
func GetSecret(key string) string {
	return os.Getenv(key)
}
