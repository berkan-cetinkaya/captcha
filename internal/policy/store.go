package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/berkan-cetinkaya/captcha/internal/config"
)

const defaultMinScore = 0.5

type Policy struct {
	MinScore   float64
	SiteKey    string
	SecretKey  string
	Theme      string
	Appearance string
}

type rawPolicy struct {
	MinScore   *float64 `json:"min_score,omitempty"`
	SiteKey    string   `json:"site_key,omitempty"`
	SecretKey  string   `json:"secret_key,omitempty"`
	Theme      string   `json:"theme,omitempty"`
	Appearance string   `json:"appearance,omitempty"`
}

type rawPolicyConfig struct {
	Provider string               `json:"provider"`
	Global   rawPolicy            `json:"global"`
	Actions  map[string]rawPolicy `json:"actions"`
}

type Store struct {
	global   Policy
	actions  map[string]Policy
	mu       sync.RWMutex
	provider string
}

var (
	policies      *Store
	policyPath    string
	policyModTime time.Time
	policyMu      sync.Mutex
)

// Current returns the latest policy store, reloading from disk when the file changes.
func Current() (*Store, error) {
	path, err := resolvePolicyPath()
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("could not stat captcha policy config: %w", err)
	}

	policyMu.Lock()
	defer policyMu.Unlock()

	if policies != nil && path == policyPath && info.ModTime().Equal(policyModTime) {
		return policies, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not open captcha policy config: %w", err)
	}

	store, err := parseAndBuild(data)
	if err != nil {
		return nil, err
	}

	policies = store
	policyPath = path
	policyModTime = info.ModTime()
	return policies, nil
}

func parseAndBuild(data []byte) (*Store, error) {
	var cfg rawPolicyConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse captcha policy config: %w", err)
	}
	if strings.TrimSpace(cfg.Provider) == "" {
		return nil, fmt.Errorf("captcha policy requires provider")
	}
	if len(cfg.Actions) == 0 {
		return nil, fmt.Errorf("captcha policy requires at least one action")
	}

	base := Policy{
		MinScore:   defaultMinScore,
		SiteKey:    strings.TrimSpace(cfg.Global.SiteKey),
		SecretKey:  strings.TrimSpace(cfg.Global.SecretKey),
		Theme:      strings.TrimSpace(cfg.Global.Theme),
		Appearance: strings.TrimSpace(cfg.Global.Appearance),
	}
	if cfg.Global.MinScore != nil {
		base.MinScore = *cfg.Global.MinScore
	}
	actions := make(map[string]Policy, len(cfg.Actions))
	missingSiteKey := base.SiteKey == ""
	for name, raw := range cfg.Actions {
		if strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("captcha policy action name cannot be empty")
		}
		p := base
		if raw.MinScore != nil {
			p.MinScore = *raw.MinScore
		}
		if s := strings.TrimSpace(raw.SiteKey); s != "" {
			p.SiteKey = s
		} else if p.SiteKey == "" {
			missingSiteKey = true
		}
		if s := strings.TrimSpace(raw.SecretKey); s != "" {
			p.SecretKey = s
		}
		if s := strings.TrimSpace(raw.Theme); s != "" {
			p.Theme = s
		}
		if s := strings.TrimSpace(raw.Appearance); s != "" {
			p.Appearance = s
		}
		actions[name] = p
	}
	if missingSiteKey {
		return nil, fmt.Errorf("captcha policy requires a site_key in global or for every action")
	}

	return &Store{
		global:   base,
		actions:  actions,
		provider: strings.TrimSpace(cfg.Provider),
	}, nil
}

func (ps *Store) PolicyFor(action string) (Policy, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	if policy, ok := ps.actions[action]; ok {
		return policy, true
	}
	return ps.global, false
}

func (ps *Store) Provider() string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.provider
}

func resolvePolicyPath() (string, error) {
	val, err := config.Get("CAPTCHA_CONFIG")
	if err != nil {
		return "", fmt.Errorf("CAPTCHA_CONFIG must be set")
	}

	env := strings.TrimSpace(val)
	if env == "" {
		return "", fmt.Errorf("CAPTCHA_CONFIG must be set")
	}
	if _, err := os.Stat(env); err != nil {
		return "", fmt.Errorf("could not stat captcha policy config (%s): %w", env, err)
	}
	return env, nil
}
