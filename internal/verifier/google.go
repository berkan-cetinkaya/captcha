package verifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const googleEndpoint = "https://www.google.com/recaptcha/api/siteverify"

type Google struct {
	Secret string
	Client *http.Client
}

func NewGoogle(secret string) *Google {
	return &Google{
		Secret: secret,
		Client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (g *Google) Verify(ctx context.Context, token, ip string) (Result, error) {
	form := url.Values{}
	form.Set("secret", g.Secret)
	form.Set("response", token)
	if ip != "" {
		form.Set("remoteip", ip)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleEndpoint, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.Client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	var raw struct {
		Success    bool     `json:"success"`
		Score      float64  `json:"score,omitempty"`
		Action     string   `json:"action,omitempty"`
		ErrorCodes []string `json:"error-codes,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return Result{}, fmt.Errorf("google decode error: %w", err)
	}

	return Result{
		Success:    raw.Success,
		Action:     raw.Action,
		Score:      &raw.Score,
		ErrorCodes: raw.ErrorCodes,
	}, nil
}
