package config

import (
	"context"
	"fmt"
	"os"

	vault "github.com/hashicorp/vault/api"
)

// VaultSource fetches values from HashiCorp Vault KV v2 backend.
type VaultSource struct {
	client    *vault.Client
	mountPath string
}

func NewVaultSource() (*VaultSource, error) {
	addr := os.Getenv("VAULT_ADDR")
	token := os.Getenv("VAULT_TOKEN")
	mount := os.Getenv("VAULT_PATH")
	if mount == "" {
		mount = "secret"
	}
	if addr == "" || token == "" {
		return nil, fmt.Errorf("vault config requires VAULT_ADDR and VAULT_TOKEN")
	}

	client, err := vault.NewClient(&vault.Config{Address: addr})
	if err != nil {
		return nil, fmt.Errorf("vault client init error: %w", err)
	}
	client.SetToken(token)
	return &VaultSource{
		client:    client,
		mountPath: mount,
	}, nil
}

func (v *VaultSource) Name() string {
	return "vault"
}

// Get tries environment variables first, then fetches from Vault at path "<VAULT_PATH>/data/{key}".
func (v *VaultSource) Get(key string) (string, error) {
	if val := os.Getenv(key); val != "" {
		return val, nil
	}

	secret, err := v.client.KVv2(v.mountPath).Get(context.Background(), key)
	if err != nil {
		return "", fmt.Errorf("vault read error: %w", err)
	}
	if val, ok := secret.Data["value"].(string); ok && val != "" {
		return val, nil
	}
	return "", fmt.Errorf("no 'value' field found in vault secret: %s", key)
}
