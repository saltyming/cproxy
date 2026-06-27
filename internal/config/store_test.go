package config

import (
	"testing"

	"github.com/saltyming/cproxy/internal/providers"
)

func TestNormalizeRepairsLegacyProviderOverridesAndOpenRouterAliases(t *testing.T) {
	t.Parallel()

	catalog, err := providers.Load()
	if err != nil {
		t.Fatal(err)
	}

	cfg := &File{
		Version: 1,
		ProviderOverrides: map[string]ProviderOverride{
			"zai": {Opus: "1"},
		},
		OpenRouterAliases: map[string]string{
			"cproxy-or-kimi-k25": "cproxy-or-kimi-k25",
			"kimi-k25":            "moonshotai/kimi-k2.5",
		},
		CustomProviders: map[string]CustomProvider{},
	}

	cfg.Normalize(catalog)

	override, ok := cfg.ProviderOverrides["zai"]
	if !ok {
		t.Fatalf("expected zai override to remain after index normalization, got %+v", cfg.ProviderOverrides)
	}
	if override.Opus != "glm-5.2" {
		t.Fatalf("expected index 1 to resolve to first model choice, got %q", override.Opus)
	}
	if len(cfg.OpenRouterAliases) != 1 {
		t.Fatalf("expected one valid OpenRouter alias, got %+v", cfg.OpenRouterAliases)
	}
	if cfg.OpenRouterAliases["kimi-k25"] != "moonshotai/kimi-k2.5" {
		t.Fatalf("expected normalized alias to keep valid model, got %+v", cfg.OpenRouterAliases)
	}
}

func TestNormalizeDropsEmptyProviderOverrides(t *testing.T) {
	t.Parallel()

	catalog, err := providers.Load()
	if err != nil {
		t.Fatal(err)
	}

	cfg := &File{
		Version: 1,
		ProviderOverrides: map[string]ProviderOverride{
			"minimax": {},
		},
		OpenRouterAliases: map[string]string{},
		CustomProviders:   map[string]CustomProvider{},
	}

	cfg.Normalize(catalog)

	if _, ok := cfg.ProviderOverrides["minimax"]; ok {
		t.Fatalf("expected empty override to be dropped, got %+v", cfg.ProviderOverrides)
	}
}

func TestApplyLegacySecretsIgnoresLauncherShapedOpenRouterValues(t *testing.T) {
	t.Parallel()

	catalog, err := providers.Load()
	if err != nil {
		t.Fatal(err)
	}

	cfg := &File{
		Version:           1,
		ProviderOverrides: map[string]ProviderOverride{},
		OpenRouterAliases: map[string]string{},
		CustomProviders:   map[string]CustomProvider{},
	}

	cfg.ApplyLegacySecrets(Secrets{
		"OPENROUTER_MODEL_CPROXY_OR_KIMI_K25": "cproxy-or-kimi-k25",
		"OPENROUTER_MODEL_KIMI_K25":            "moonshotai/kimi-k2.5",
	}, catalog)

	if len(cfg.OpenRouterAliases) != 1 {
		t.Fatalf("expected one migrated alias, got %+v", cfg.OpenRouterAliases)
	}
	if cfg.OpenRouterAliases["kimi-k25"] != "moonshotai/kimi-k2.5" {
		t.Fatalf("unexpected aliases: %+v", cfg.OpenRouterAliases)
	}
}
