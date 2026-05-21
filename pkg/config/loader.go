// Package config provides automatic configuration discovery and loading.
// It searches for configuration in multiple locations with priority:
//
//  1. Environment variables (highest)
//  2. Service-specific .env (<projectRoot>/<serviceName>/.env)
//  3. Global .env.global (project root)
//  4. Service-specific .env.example
//  5. Code defaults (lowest)
//
// Project root is detected by walking up the directory tree until a
// `.git` directory or `go.work` file is found.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Loader handles automatic configuration discovery across multiple sources.
type Loader struct {
	serviceName string
	searchPaths []string
}

// NewLoader creates a configuration loader for the given service.
// serviceName is the directory name under the project root that holds the
// service (e.g. "kris-alpha"). The project root is auto-discovered.
func NewLoader(serviceName string) *Loader {
	projectRoot := findProjectRoot()
	serviceDir := filepath.Join(projectRoot, serviceName)

	return &Loader{
		serviceName: serviceName,
		searchPaths: []string{
			serviceDir,
			projectRoot,
			filepath.Dir(serviceDir),
		},
	}
}

// Load loads configuration from all sources in priority order.
// Priority: env vars > service .env > global .env.global > .env.example > defaults
func (l *Loader) Load() error {
	globalEnv := l.findFile(".env.global")
	if globalEnv != "" {
		if err := loadDotEnv(globalEnv); err != nil {
			return fmt.Errorf("load global config: %w", err)
		}
	}

	serviceExample := l.findFile(".env.example")
	if serviceExample != "" {
		if err := loadDotEnv(serviceExample); err != nil {
			return fmt.Errorf("load service example: %w", err)
		}
	}

	serviceEnv := l.findFile(".env")
	if serviceEnv != "" {
		if err := loadDotEnv(serviceEnv); err != nil {
			return fmt.Errorf("load service config: %w", err)
		}
	}

	return nil
}

// findFile searches for a file across all search paths and returns the first hit.
func (l *Loader) findFile(filename string) string {
	for _, dir := range l.searchPaths {
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// findProjectRoot walks up looking for a project-root marker: `.git` or `go.work`.
// Returns "." if nothing is found.
func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}

	markers := []string{".git", "go.work"}
	for {
		for _, marker := range markers {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return dir
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

// loadDotEnv reads KEY=VALUE pairs from a .env file. Existing environment
// variables are preserved (not overwritten) so callers can layer files.
func loadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}

		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		val = strings.Trim(val, `"'`)

		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}

	return nil
}

// GetEnv returns an environment variable with a default value.
func GetEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// GetEnvBool returns a boolean environment variable with a default value.
func GetEnvBool(key string, defaultValue bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultValue
	}
	return b
}

// GetEnvDuration returns a duration environment variable with a default value.
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return defaultValue
	}
	return d
}
