package verifier

import "context"

// Result is the unified verification response model for all CAPTCHA providers.
type Result struct {
	Success    bool
	Action     string
	Score      *float64
	ErrorCodes []string
}

// Verifier is the generic interface implemented by each provider.
type Verifier interface {
	Verify(ctx context.Context, token, ip string) (Result, error)
}
