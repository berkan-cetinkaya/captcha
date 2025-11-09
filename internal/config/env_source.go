package config

import (
	"fmt"
	"os"
)

// EnvSource loads values from environment variables (.env in dev).
type EnvSource struct{}

func NewEnvSource() *EnvSource {
	return &EnvSource{}
}

func (e *EnvSource) Name() string {
	return "env"
}

func (e *EnvSource) Get(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("env %s not set", key)
	}
	return val, nil
}
