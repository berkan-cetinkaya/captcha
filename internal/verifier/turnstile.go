package verifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

const turnstileEndpoint = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

type Turnstile struct {
	Secret string
	Client *http.Client
}

func NewTurnstile(secret string) *Turnstile {
	return &Turnstile{
		Secret: secret,
		Client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (t *Turnstile) Verify(ctx context.Context, token, ip string) (Result, error) {
	form := url.Values{}
	form.Set("secret", t.Secret)
	form.Set("response", token)
	if ip != "" {
		form.Set("remoteip", ip)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, turnstileEndpoint, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.Client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	var raw struct {
		Success    bool     `json:"success"`
		Action     string   `json:"action,omitempty"`
		ErrorCodes []string `json:"error-codes,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return Result{}, fmt.Errorf("turnstile decode error: %w", err)
	}
	if err != nil {
		log.Println("error reading body:", err)
	}

	return Result{
		Success:    raw.Success,
		Action:     raw.Action,
		Score:      nil,
		ErrorCodes: raw.ErrorCodes,
	}, nil
}
