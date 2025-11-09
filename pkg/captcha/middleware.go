package captcha

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	serviceOnce sync.Once
	service     CaptchaService
)

type FailureHandler func(http.ResponseWriter, *http.Request, VerificationResult)

type middlewareConfig struct {
	failureHandler FailureHandler
}

type MiddlewareOption func(*middlewareConfig)

func WithFailureHandler(handler FailureHandler) MiddlewareOption {
	return func(cfg *middlewareConfig) {
		if handler != nil {
			cfg.failureHandler = handler
		}
	}
}

func Middleware(expectedAction string, opts ...MiddlewareOption) func(http.Handler) http.Handler {
	cfg := middlewareConfig{
		failureHandler: JSONFailureHandler(http.StatusBadRequest),
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serviceOnce.Do(func() {
				service = NewCaptchaService()
			})
			svc := service
			token := extractToken(r)
			if token == "" {
				cfg.failureHandler(w, r, VerificationResult{
					Success: false,
					Status:  "token_missing",
					Message: "missing captcha token",
				})
				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), 6*time.Second)
			defer cancel()

			result := svc.Verify(ctx, token, r.RemoteAddr, expectedAction)
			if !result.Success {
				cfg.failureHandler(w, r, result)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func JSONFailureHandler(status int) FailureHandler {
	return func(w http.ResponseWriter, _ *http.Request, result VerificationResult) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(result)
	}
}

func extractToken(r *http.Request) string {
	if t := r.Header.Get("X-Captcha-Token"); t != "" {
		return t
	}
	if err := r.ParseForm(); err == nil {
		if t := r.FormValue("cf-turnstile-response"); t != "" {
			return t
		}
		if t := r.FormValue("g-recaptcha-response"); t != "" {
			return t
		}
		if t := r.FormValue("token"); t != "" {
			return t
		}
	}
	if r.Body != nil {
		b, _ := io.ReadAll(r.Body)
		if len(b) > 0 {
			var m map[string]any
			_ = json.Unmarshal(b, &m)
			r.Body = io.NopCloser(strings.NewReader(string(b)))
			if v, ok := m["token"].(string); ok {
				return v
			}
		}
	}
	return ""
}
