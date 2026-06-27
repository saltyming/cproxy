package profiles

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/saltyming/cproxy/internal/config"
	"github.com/saltyming/cproxy/internal/providers"
)

type Target struct {
	Profile             string
	DisplayName         string
	Description         string
	Category            string
	Family              providers.Family
	BaseURL             string
	Model               string
	ModelTiers          map[string]string
	ModelContextWindows map[string]string
	AuthMode            providers.AuthMode
	SecretKey           string
	LiteralAuthToken    string
	TestURL             string
}

func Invocation(argv0 string) (string, bool) {
	base := filepath.Base(argv0)
	if base == "cproxy" || base == "cproxy.sh" {
		return "", false
	}
	if strings.HasPrefix(base, "cproxy-") {
		return strings.TrimPrefix(base, "cproxy-"), true
	}
	return "", false
}

func Resolve(profile string, catalog providers.Catalog, cfg *config.File) (Target, error) {
	if provider, ok := catalog.Get(profile); ok {
		override := cfg.ProviderOverrides[profile]
		opus := override.Opus
		if opus == "" {
			opus = provider.DefaultModel
		}
		sonnet := override.Sonnet
		if sonnet == "" {
			sonnet = opus
		}
		haiku := override.Haiku
		if haiku == "" {
			haiku = sonnet
		}
		small := override.Small
		if small == "" {
			small = haiku
		}
		tiers := map[string]string{
			"opus":   opus,
			"sonnet": sonnet,
			"haiku":  haiku,
			"small":  small,
		}
		// Pick the strongest tier the user explicitly set as the
		// "main" model for /info display. Claude Code itself reads
		// the per-tier variables, so this is purely cosmetic.
		model := provider.DefaultModel
		switch {
		case override.Opus != "":
			model = override.Opus
		case override.Sonnet != "":
			model = override.Sonnet
		case override.Haiku != "":
			model = override.Haiku
		case override.Small != "":
			model = override.Small
		}
		return Target{
			Profile:             profile,
			DisplayName:         provider.DisplayName,
			Description:         provider.Description,
			Category:            provider.Category,
			Family:              provider.Family,
			BaseURL:             provider.BaseURL,
			Model:               model,
			ModelTiers:          tiers,
			ModelContextWindows: collectContextWindows(catalog, tiers),
			AuthMode:            provider.AuthMode,
			SecretKey:           provider.KeyVar,
			LiteralAuthToken:    provider.LiteralAuthToken,
			TestURL:             provider.TestURL,
		}, nil
	}
	if strings.HasPrefix(profile, "or-") {
		name := strings.TrimPrefix(profile, "or-")
		model := cfg.OpenRouterAliases[name]
		if model == "" {
			return Target{}, fmt.Errorf("unknown OpenRouter alias %q", name)
		}
		return Target{
			Profile:     profile,
			DisplayName: "OpenRouter: " + name,
			Description: "OpenRouter alias",
			Category:    "advanced",
			Family:      providers.FamilyOpenRouter,
			BaseURL:     "https://openrouter.ai/api",
			ModelTiers: map[string]string{
				"haiku":  model,
				"sonnet": model,
				"opus":   model,
				"small":  model,
			},
			AuthMode:  providers.AuthSecret,
			SecretKey: "OPENROUTER_API_KEY",
			TestURL:   "https://openrouter.ai/api",
		}, nil
	}
	if custom, ok := cfg.CustomProviders[profile]; ok {
		return Target{
			Profile:     profile,
			DisplayName: custom.DisplayName,
			Description: "Custom provider",
			Category:    "advanced",
			Family:      providers.FamilyCustomUnknown,
			BaseURL:     custom.BaseURL,
			Model:       custom.DefaultModel,
			ModelTiers: map[string]string{
				"opus":   custom.DefaultModel,
				"sonnet": custom.DefaultModel,
				"haiku":  custom.DefaultModel,
				"small":  custom.DefaultModel,
			},
			AuthMode:  providers.AuthSecret,
			SecretKey: custom.APIKeyEnv,
			TestURL:   custom.BaseURL,
		}, nil
	}
	return Target{}, fmt.Errorf("unknown profile %q", profile)
}

func All(catalog providers.Catalog, cfg *config.File) []Target {
	var out []Target
	for _, provider := range catalog.All() {
		target, _ := Resolve(provider.ID, catalog, cfg)
		out = append(out, target)
	}
	for _, name := range cfg.OpenRouterNames() {
		target, _ := Resolve("or-"+name, catalog, cfg)
		out = append(out, target)
	}
	for _, name := range cfg.CustomProviderNames() {
		target, _ := Resolve(name, catalog, cfg)
		out = append(out, target)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Profile < out[j].Profile
	})
	return out
}

func copyMap(input map[string]string) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

// collectContextWindows maps each tier ("haiku"/"sonnet"/"opus"/"small")
// to the catalog-declared context window of the model that tier resolves
// to. Models without a catalogued context window are omitted so callers
// can tell "unknown" from "explicit empty".
func collectContextWindows(catalog providers.Catalog, tiers map[string]string) map[string]string {
	out := map[string]string{}
	for tier, modelID := range tiers {
		if modelID == "" {
			continue
		}
		if cw := catalog.ContextWindowFor(modelID); cw != "" {
			out[tier] = cw
		}
	}
	return out
}
