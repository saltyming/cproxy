package runtime

import (
	"strings"
	"testing"

	"github.com/saltyming/cproxy/internal/config"
	"github.com/saltyming/cproxy/internal/profiles"
	"github.com/saltyming/cproxy/internal/providers"
)

func TestBuildEnvForOpenRouter(t *testing.T) {
	t.Parallel()

	target := profiles.Target{
		Profile:   "or-kimi",
		Family:    providers.FamilyOpenRouter,
		BaseURL:   "https://openrouter.ai/api",
		AuthMode:  providers.AuthSecret,
		SecretKey: "OPENROUTER_API_KEY",
		ModelTiers: map[string]string{
			"haiku":  "moonshotai/kimi-k2.5",
			"sonnet": "moonshotai/kimi-k2.5",
			"opus":   "moonshotai/kimi-k2.5",
		},
	}

	env, err := BuildEnv(target, config.Secrets{"OPENROUTER_API_KEY": "sk-openrouter"})
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(env, "\n")
	for _, expected := range []string{
		"ANTHROPIC_BASE_URL=https://openrouter.ai/api",
		"ANTHROPIC_AUTH_TOKEN=sk-openrouter",
		"ANTHROPIC_API_KEY=",
		"ANTHROPIC_DEFAULT_OPUS_MODEL=moonshotai/kimi-k2.5",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("env missing %q:\n%s", expected, text)
		}
	}
}

func TestBuildEnvCustomProviderClearsAPIKey(t *testing.T) {
	t.Parallel()

	target := profiles.Target{
		Profile:   "myprovider",
		Family:    providers.FamilyCustomUnknown,
		BaseURL:   "https://api.example.com/anthropic",
		AuthMode:  providers.AuthSecret,
		SecretKey: "MYPROVIDER_API_KEY",
	}

	env, err := BuildEnv(target, config.Secrets{"MYPROVIDER_API_KEY": "sk-custom"})
	if err != nil {
		t.Fatal(err)
	}
	got := envToMap(env)
	if got["ANTHROPIC_AUTH_TOKEN"] != "sk-custom" {
		t.Fatalf("ANTHROPIC_AUTH_TOKEN = %q, want sk-custom", got["ANTHROPIC_AUTH_TOKEN"])
	}
	if v, ok := got["ANTHROPIC_API_KEY"]; !ok || v != "" {
		t.Fatalf("ANTHROPIC_API_KEY should be cleared for custom providers, got %q (present=%v)", v, ok)
	}
}

func TestBuildEnvFailsWhenSecretMissing(t *testing.T) {
	t.Parallel()

	target := profiles.Target{
		Profile:   "zai",
		Family:    providers.FamilyAnthropicCompatibleNonClaude,
		AuthMode:  providers.AuthSecret,
		SecretKey: "ZAI_API_KEY",
		BaseURL:   "https://api.z.ai/api/anthropic",
	}
	if _, err := BuildEnv(target, config.Secrets{}); err == nil {
		t.Fatal("expected missing secret error")
	}
}

func TestBuildEnvNativeClearsInheritedAnthropic(t *testing.T) {
	t.Setenv("ANTHROPIC_BASE_URL", "https://evil.example")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "bad-token")
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "bad-model")

	target := profiles.Target{
		Profile:  "native",
		Family:   providers.FamilyClaudeStrict,
		AuthMode: providers.AuthNone,
	}

	env, err := BuildEnv(target, config.Secrets{})
	if err != nil {
		t.Fatal(err)
	}
	for key := range envToMap(env) {
		if strings.HasPrefix(key, "ANTHROPIC_") {
			t.Fatalf("native launcher leaked %s from parent environment", key)
		}
	}
}

func TestBuildEnvClearsUnusedTierVariables(t *testing.T) {
	t.Setenv("ANTHROPIC_DEFAULT_HAIKU_MODEL", "stale-haiku")
	t.Setenv("ANTHROPIC_DEFAULT_SONNET_MODEL", "stale-sonnet")
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "stale-opus")

	target := profiles.Target{
		Profile:   "zai",
		Family:    providers.FamilyAnthropicCompatibleNonClaude,
		BaseURL:   "https://api.z.ai/api/anthropic",
		AuthMode:  providers.AuthSecret,
		SecretKey: "ZAI_API_KEY",
		ModelTiers: map[string]string{
			"opus": "glm-5.2",
		},
	}

	env, err := BuildEnv(target, config.Secrets{"ZAI_API_KEY": "sk-zai"})
	if err != nil {
		t.Fatal(err)
	}
	got := envToMap(env)
	if got["ANTHROPIC_DEFAULT_OPUS_MODEL"] != "glm-5.2" {
		t.Fatalf("unexpected opus model: %q", got["ANTHROPIC_DEFAULT_OPUS_MODEL"])
	}
	if _, ok := got["ANTHROPIC_DEFAULT_HAIKU_MODEL"]; ok {
		t.Fatal("haiku tier leaked from inherited environment")
	}
	if _, ok := got["ANTHROPIC_DEFAULT_SONNET_MODEL"]; ok {
		t.Fatal("sonnet tier leaked from inherited environment")
	}
}

func envToMap(env []string) map[string]string {
	out := map[string]string{}
	for _, pair := range env {
		key, value, ok := splitEnv(pair)
		if ok {
			out[key] = value
		}
	}
	return out
}

func TestWithContextSuffix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		modelID  string
		window   string
		expected string
	}{
		{"1M gets suffix", "glm-5.2", "1M", "glm-5.2[1m]"},
		{"1M case-insensitive", "glm-5.2", "1m", "glm-5.2[1m]"},
		{"1M with whitespace", "glm-5.2", " 1M ", "glm-5.2[1m]"},
		{"200K unchanged", "glm-5", "200K", "glm-5"},
		{"empty window unchanged", "glm-5.2", "", "glm-5.2"},
		{"empty modelID stays empty", "", "1M", ""},
		{"unknown window unchanged", "foo", "100G", "foo"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := WithContextSuffix(tc.modelID, tc.window); got != tc.expected {
				t.Fatalf("WithContextSuffix(%q, %q) = %q, want %q", tc.modelID, tc.window, got, tc.expected)
			}
		})
	}
}

func TestCompactWindowTokens(t *testing.T) {
	t.Parallel()

	cases := []struct {
		window   string
		expected string
	}{
		{"1M", "1000000"},
		{"1m", "1000000"},
		{" 1M ", "1000000"},
		{"200K", "200000"},
		{"262K", "262000"},
		{"256K", "256000"},
		{"200k", "200000"},
		{"2M", "2000000"},
		{"", ""},
		{"100G", ""},
		{"abc", ""},
		{"1MB", ""},
		{"KM", ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.window+"->"+tc.expected, func(t *testing.T) {
			t.Parallel()
			if got := CompactWindowTokens(tc.window); got != tc.expected {
				t.Fatalf("CompactWindowTokens(%q) = %q, want %q", tc.window, got, tc.expected)
			}
		})
	}
}

func TestBuildEnvAnnotatesOneMContext(t *testing.T) {
	t.Parallel()

	target := profiles.Target{
		Profile:  "minimax",
		Family:   providers.FamilyAnthropicCompatibleNonClaude,
		BaseURL:  "https://api.minimax.io/anthropic",
		AuthMode: providers.AuthSecret,
		SecretKey: "MINIMAX_API_KEY",
		ModelTiers: map[string]string{
			"haiku":  "MiniMax-M3",
			"sonnet": "MiniMax-M3",
			"opus":   "MiniMax-M3",
		},
		ModelContextWindows: map[string]string{
			"haiku":  "1M",
			"sonnet": "1M",
			"opus":   "1M",
		},
	}

	env, err := BuildEnv(target, config.Secrets{"MINIMAX_API_KEY": "sk-m"})
	if err != nil {
		t.Fatal(err)
	}
	got := envToMap(env)
	if got["ANTHROPIC_DEFAULT_OPUS_MODEL"] != "MiniMax-M3[1m]" {
		t.Fatalf("opus should carry [1m] suffix, got %q", got["ANTHROPIC_DEFAULT_OPUS_MODEL"])
	}
	if got["ANTHROPIC_DEFAULT_SONNET_MODEL"] != "MiniMax-M3[1m]" {
		t.Fatalf("sonnet should carry [1m] suffix, got %q", got["ANTHROPIC_DEFAULT_SONNET_MODEL"])
	}
	if got["ANTHROPIC_DEFAULT_HAIKU_MODEL"] != "MiniMax-M3[1m]" {
		t.Fatalf("haiku should carry [1m] suffix, got %q", got["ANTHROPIC_DEFAULT_HAIKU_MODEL"])
	}
	if got["CLAUDE_CODE_AUTO_COMPACT_WINDOW"] != "1000000" {
		t.Fatalf("CLAUDE_CODE_AUTO_COMPACT_WINDOW = %q, want 1000000", got["CLAUDE_CODE_AUTO_COMPACT_WINDOW"])
	}
}

func TestBuildEnvKeeps200KWindowBare(t *testing.T) {
	t.Parallel()

	target := profiles.Target{
		Profile:  "zai",
		Family:   providers.FamilyAnthropicCompatibleNonClaude,
		BaseURL:  "https://api.z.ai/api/anthropic",
		AuthMode: providers.AuthSecret,
		SecretKey: "ZAI_API_KEY",
		ModelTiers: map[string]string{
			"opus": "glm-5",
		},
		ModelContextWindows: map[string]string{
			"opus": "200K",
		},
	}

	env, err := BuildEnv(target, config.Secrets{"ZAI_API_KEY": "sk-zai"})
	if err != nil {
		t.Fatal(err)
	}
	got := envToMap(env)
	if got["ANTHROPIC_DEFAULT_OPUS_MODEL"] != "glm-5" {
		t.Fatalf("200K model should not get [1m] suffix, got %q", got["ANTHROPIC_DEFAULT_OPUS_MODEL"])
	}
	if got["CLAUDE_CODE_AUTO_COMPACT_WINDOW"] != "200000" {
		t.Fatalf("CLAUDE_CODE_AUTO_COMPACT_WINDOW = %q, want 200000", got["CLAUDE_CODE_AUTO_COMPACT_WINDOW"])
	}
}

func TestBuildEnvSkipsAutoCompactWhenWindowUnknown(t *testing.T) {
	t.Parallel()

	target := profiles.Target{
		Profile:  "minimax",
		Family:   providers.FamilyAnthropicCompatibleNonClaude,
		BaseURL:  "https://api.minimax.io/anthropic",
		AuthMode: providers.AuthSecret,
		SecretKey: "MINIMAX_API_KEY",
		ModelTiers: map[string]string{
			"opus": "MiniMax-M3",
		},
		// No ModelContextWindows entry — catalog has no context_window for
		// this model, so we should not emit a misleading compact threshold.
	}

	env, err := BuildEnv(target, config.Secrets{"MINIMAX_API_KEY": "sk-m"})
	if err != nil {
		t.Fatal(err)
	}
	got := envToMap(env)
	if _, ok := got["CLAUDE_CODE_AUTO_COMPACT_WINDOW"]; ok {
		t.Fatalf("CLAUDE_CODE_AUTO_COMPACT_WINDOW should be absent when window unknown, got %q",
			got["CLAUDE_CODE_AUTO_COMPACT_WINDOW"])
	}
	if got["ANTHROPIC_DEFAULT_OPUS_MODEL"] != "MiniMax-M3" {
		t.Fatalf("model should be unchanged without window info, got %q", got["ANTHROPIC_DEFAULT_OPUS_MODEL"])
	}
}

func TestBuildEnvAutoCompactHonoursOverriddenModel(t *testing.T) {
	t.Parallel()

	target := profiles.Target{
		Profile:  "minimax",
		Family:   providers.FamilyAnthropicCompatibleNonClaude,
		BaseURL:  "https://api.minimax.io/anthropic",
		AuthMode: providers.AuthSecret,
		SecretKey: "MINIMAX_API_KEY",
		// profiles.Resolve rewrites every tier to the override model,
		// so by the time BuildEnv sees this Target ANTHROPIC_MODEL is
		// intentionally absent and the override reaches Claude Code via
		// the per-tier variables.
		Model: "MiniMax-M3",
		ModelTiers: map[string]string{
			"haiku":  "MiniMax-M3",
			"sonnet": "MiniMax-M3",
			"opus":   "MiniMax-M3",
			"small":  "MiniMax-M3",
		},
		ModelContextWindows: map[string]string{
			"haiku":  "1M",
			"sonnet": "1M",
			"opus":   "1M",
			"small":  "1M",
		},
	}

	env, err := BuildEnv(target, config.Secrets{"MINIMAX_API_KEY": "sk-m"})
	if err != nil {
		t.Fatal(err)
	}
	got := envToMap(env)
	if _, ok := got["ANTHROPIC_MODEL"]; ok {
		t.Fatalf("ANTHROPIC_MODEL must not be emitted, got %q", got["ANTHROPIC_MODEL"])
	}
	if got["ANTHROPIC_DEFAULT_OPUS_MODEL"] != "MiniMax-M3[1m]" {
		t.Fatalf("override tier should carry [1m] suffix, got %q", got["ANTHROPIC_DEFAULT_OPUS_MODEL"])
	}
	if got["CLAUDE_CODE_AUTO_COMPACT_WINDOW"] != "1000000" {
		t.Fatalf("CLAUDE_CODE_AUTO_COMPACT_WINDOW = %q, want 1000000", got["CLAUDE_CODE_AUTO_COMPACT_WINDOW"])
	}
}

func TestBuildEnvNeverEmitsAnthropicModel(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name    string
		target  profiles.Target
		secrets config.Secrets
	}{
		{
			name: "third-party with tiers",
			target: profiles.Target{
				Family:   providers.FamilyAnthropicCompatibleNonClaude,
				BaseURL:  "https://api.example.com/anthropic",
				AuthMode: providers.AuthSecret,
				SecretKey: "X_API_KEY",
				ModelTiers: map[string]string{
					"haiku": "m1",
					"opus":  "m1",
				},
			},
			secrets: config.Secrets{"X_API_KEY": "sk-x"},
		},
		{
			name: "native has no model context",
			target: profiles.Target{
				Family:   providers.FamilyClaudeStrict,
				AuthMode: providers.AuthNone,
			},
			secrets: config.Secrets{},
		},
		{
			name: "override model still no ANTHROPIC_MODEL",
			target: profiles.Target{
				Family:   providers.FamilyAnthropicCompatibleNonClaude,
				BaseURL:  "https://api.example.com/anthropic",
				AuthMode: providers.AuthSecret,
				SecretKey: "X_API_KEY",
				Model:    "override-m",
				ModelTiers: map[string]string{
					"haiku": "override-m",
					"opus":  "override-m",
				},
			},
			secrets: config.Secrets{"X_API_KEY": "sk-x"},
		},
	}

	for _, sc := range scenarios {
		sc := sc
		t.Run(sc.name, func(t *testing.T) {
			t.Parallel()
			env, err := BuildEnv(sc.target, sc.secrets)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := envToMap(env)["ANTHROPIC_MODEL"]; ok {
				t.Fatal("ANTHROPIC_MODEL must never be emitted; tier variables carry model identity")
			}
		})
	}
}

func TestBuildEnvPicksMaxContextAcrossTiers(t *testing.T) {
	t.Parallel()

	target := profiles.Target{
		Family:   providers.FamilyAnthropicCompatibleNonClaude,
		BaseURL:  "https://api.example.com/anthropic",
		AuthMode: providers.AuthSecret,
		SecretKey: "X_API_KEY",
		ModelTiers: map[string]string{
			"haiku":  "small-model",
			"sonnet": "big-model",
			"opus":   "big-model",
		},
		ModelContextWindows: map[string]string{
			"haiku":  "200K",
			"sonnet": "1M",
			"opus":   "1M",
		},
	}

	env, err := BuildEnv(target, config.Secrets{"X_API_KEY": "sk-x"})
	if err != nil {
		t.Fatal(err)
	}
	got := envToMap(env)
	if got["CLAUDE_CODE_AUTO_COMPACT_WINDOW"] != "1000000" {
		t.Fatalf("CLAUDE_CODE_AUTO_COMPACT_WINDOW = %q, want 1000000 (max of 200K, 1M, 1M)",
			got["CLAUDE_CODE_AUTO_COMPACT_WINDOW"])
	}
	if got["ANTHROPIC_DEFAULT_HAIKU_MODEL"] != "small-model" {
		t.Fatalf("haiku 200K should not carry [1m], got %q", got["ANTHROPIC_DEFAULT_HAIKU_MODEL"])
	}
	if got["ANTHROPIC_DEFAULT_SONNET_MODEL"] != "big-model[1m]" {
		t.Fatalf("sonnet 1M should carry [1m], got %q", got["ANTHROPIC_DEFAULT_SONNET_MODEL"])
	}
}
