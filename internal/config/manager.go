package config

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// Source describes a backend that can provide configuration values.
type Source interface {
	Get(key string) (string, error)
	Name() string
}

// Manager stores a single source (selected via CONFIG_PROVIDER) and proxies calls.
type Manager struct {
	source Source
}

func (m *Manager) Get(key string) (string, error) {
	return m.source.Get(key)
}

var (
	defaultManager *Manager
	managerOnce    sync.Once
	managerErr     error
)

// Get returns the value for a given key from the configured source.
func Get(key string) (string, error) {
	mgr, err := getDefaultManager()
	if err != nil {
		return "", err
	}
	return mgr.Get(key)
}

// MustGet returns the value or panics if it does not exist.
func MustGet(key string) string {
	val, err := Get(key)
	if err != nil {
		panic(err)
	}
	return val
}

// GetDefault returns the value if available, otherwise falls back to defaultVal.
func GetDefault(key, defaultVal string) string {
	val, err := Get(key)
	if err != nil || val == "" {
		return defaultVal
	}
	return val
}

func getDefaultManager() (*Manager, error) {
	managerOnce.Do(func() {
		sourceName := strings.ToLower(strings.TrimSpace(os.Getenv("CONFIG_PROVIDER")))
		if sourceName == "" {
			sourceName = "env"
		}

		source, err := newSource(sourceName)
		if err != nil {
			managerErr = err
			return
		}

		defaultManager = &Manager{source: source}
	})

	return defaultManager, managerErr
}

func newSource(name string) (Source, error) {
	switch name {
	case "env":
		return NewEnvSource(), nil
	case "vault":
		return NewVaultSource()
	default:
		return nil, fmt.Errorf("unknown config provider: %s", name)
	}
}
