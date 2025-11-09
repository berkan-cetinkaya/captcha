package captcha

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/berkan-cetinkaya/captcha/internal/policy"
	"github.com/berkan-cetinkaya/captcha/internal/verifier"

	cfg "github.com/berkan-cetinkaya/captcha/internal/config"
)

// ActionMetadata represents the frontend-facing configuration for a policy action.
type ActionMetadata struct {
	Action     string
	SiteKey    string
	Theme      string
	Appearance string
}

type VerificationResult struct {
	Success bool   `json:"success"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type CaptchaService interface {
	Verify(ctx context.Context, token, ip, expectedAction string) VerificationResult
}

type captchaService struct {
	buildVerifier func(secret string) verifier.Verifier
}

func NewCaptchaService() CaptchaService {
	store, err := policy.Current()
	if err != nil {
		panic(fmt.Sprintf("failed to load CAPTCHA config: %v", err))
	}
	provider := store.Provider()
	if provider == "" {
		panic("captcha provider missing in policy config")
	}

	var builder func(secret string) verifier.Verifier
	switch strings.ToLower(provider) {
	case "google":
		builder = func(secret string) verifier.Verifier {
			return verifier.NewGoogle(secret)
		}
	case "turnstile":
		builder = func(secret string) verifier.Verifier {
			return verifier.NewTurnstile(secret)
		}
	default:
		panic("invalid CAPTCHA_PROVIDER (turnstile|google)")
	}

	return &captchaService{
		buildVerifier: builder,
	}
}

func (s *captchaService) Verify(ctx context.Context, token, ip, expectedAction string) VerificationResult {
	store, err := policy.Current()
	if err != nil {
		return VerificationResult{
			Success: false,
			Status:  "policy_error",
			Message: fmt.Sprintf("failed to load policy: %v", err),
		}
	}
	policyValue, ok := store.PolicyFor(expectedAction)
	if !ok {
		log.Printf("[captcha] no policy override for '%s' — using default min_score=%.2f\n", expectedAction, policyValue.MinScore)
	}

	v, err := s.verifierFor(policyValue)
	if err != nil {
		return VerificationResult{
			Success: false,
			Status:  "config_error",
			Message: fmt.Sprintf("captcha secret error: %v", err),
		}
	}

	res, err := v.Verify(ctx, token, ip)
	if err != nil {
		return VerificationResult{
			Success: false,
			Status:  "verify_error",
			Message: fmt.Sprintf("verify error: %v", err),
		}
	}
	if !res.Success {
		return VerificationResult{
			Success: false,
			Status:  "success_failed",
			Message: "captcha provider marked challenge as failed",
		}
	}

	if res.Action != expectedAction {
		return VerificationResult{
			Success: false,
			Status:  "action_mismatch",
			Message: fmt.Sprintf("captcha action mismatch: expected '%s', got '%s'", expectedAction, res.Action),
		}
	}

	if res.Score != nil && *res.Score < policyValue.MinScore {
		return VerificationResult{
			Success: false,
			Status:  "score_too_low",
			Message: fmt.Sprintf("captcha score too low: %.2f < %.2f", *res.Score, policyValue.MinScore),
		}
	}

	return VerificationResult{
		Success: true,
		Status:  "verified",
		Message: "captcha verification passed",
	}
}

func (s *captchaService) verifierFor(p policy.Policy) (verifier.Verifier, error) {
	secretKey := p.SecretKey
	secret, err := cfg.Get(secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load secret '%s': %w", secretKey, err)
	}
	return s.buildVerifier(secret), nil
}

// Metadata returns action metadata using a shared policy store without
// requiring a CaptchaService instance.
func Metadata(action string) (ActionMetadata, error) {
	store, err := policy.Current()
	if err != nil {
		return ActionMetadata{}, err
	}
	policyValue, _ := store.PolicyFor(action)
	return ActionMetadata{
		Action:     action,
		SiteKey:    policyValue.SiteKey,
		Theme:      policyValue.Theme,
		Appearance: policyValue.Appearance,
	}, nil
}
