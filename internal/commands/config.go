package commands

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/saltyming/cproxy/internal/config"
	"github.com/saltyming/cproxy/internal/launchers"
	"github.com/saltyming/cproxy/internal/providers"
	"github.com/saltyming/cproxy/internal/runtime"
)

var (
	validName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
)

func runConfig(_ context.Context, c Context, args []string) (int, error) {
	providerID := ""
	if len(args) > 0 {
		providerID = args[0]
	} else {
		var err error
		providerID, err = chooseProvider(c)
		if err != nil || providerID == "" {
			return 0, err
		}
	}

	switch providerID {
	case "openrouter":
		return configOpenRouter(c)
	case "custom":
		return configCustom(c)
	default:
		if provider, ok := c.Catalog.Get(providerID); ok {
			return configBuiltin(c, provider)
		}
		return 1, fmt.Errorf("unknown provider %q", providerID)
	}
}

func chooseProvider(c Context) (string, error) {
	index := 1
	choices := map[int]string{}
	c.Output.Header("Cproxy Configuration")
	for _, category := range c.Catalog.Categories() {
		fmt.Fprintln(c.Output.Stdout, category)
		for _, provider := range c.Catalog.ProvidersByCategory(category) {
			fmt.Fprintf(c.Output.Stdout, "  %2d. %-14s %s\n", index, provider.ID, provider.Description)
			choices[index] = provider.ID
			index++
		}
	}
	fmt.Fprintf(c.Output.Stdout, "  %2d. %-14s %s\n", index, "openrouter", "100+ models")
	choices[index] = "openrouter"
	index++
	fmt.Fprintf(c.Output.Stdout, "  %2d. %-14s %s\n", index, "custom", "Anthropic-compatible endpoint")
	choices[index] = "custom"

	answer, err := c.Prompt.Prompt("Choose provider number", "")
	if err != nil {
		return "", err
	}
	for number, providerID := range choices {
		if fmt.Sprint(number) == answer {
			return providerID, nil
		}
	}
	return "", fmt.Errorf("invalid choice %q", answer)
}

func configBuiltin(c Context, provider providers.Provider) (int, error) {
	if provider.AuthMode == providers.AuthSecret {
		current := c.Secrets[provider.KeyVar]
		if current != "" {
			fmt.Fprintf(c.Output.Stdout, "Current key: %s\n", config.MaskSecret(current))
		}
		label := "API key"
		if current != "" {
			label = "API key (empty to keep current)"
		}
		value, err := c.Prompt.PromptSecret(label)
		if err != nil {
			return 1, err
		}
		if strings.TrimSpace(value) != "" {
			c.Secrets[provider.KeyVar] = value
		}
	}

	if len(provider.ModelChoices) > 0 {
		override := c.Config.ProviderOverrides[provider.ID]

		promptOne := func(tier, header string, fallback string) error {
			existing := tierValue(override, tier)
			prefill := existing
			if prefill == "" {
				prefill = fallback
			}
			fmt.Fprintf(c.Output.Stdout, "\n%s tier (%s, current: %s):\n", tier, header, prefill)
			for idx, choice := range provider.ModelChoices {
				fmt.Fprintf(c.Output.Stdout, "  %d. %s — %s\n", idx+1, choice.ID, choice.Description)
			}
			answer, err := c.Prompt.Prompt(tier+" tier", prefill)
			if err != nil {
				return err
			}
			answer = resolveModelChoice(answer, provider.ModelChoices)
			if strings.TrimSpace(answer) == "" {
				answer = prefill
			}
			// Only persist when the user actually diverged from the
			// prefill. Accepting the cascade default keeps the on-disk
			// override empty so runtime cascade stays the source of truth.
			if answer != prefill {
				setTierField(&override, tier, answer)
			}
			return nil
		}

		// Opus prefill = provider.DefaultModel. Sonnet defaults to whatever
		// the user picked (or the cascade chain). Haiku cascades from
		// sonnet, small from haiku — every tier unwinds to DefaultModel.
		opusFallback := provider.DefaultModel
		if err := promptOne("opus", "most capable", opusFallback); err != nil {
			return 1, err
		}
		sonnetFallback := tierValue(override, "opus")
		if sonnetFallback == "" {
			sonnetFallback = opusFallback
		}
		if err := promptOne("sonnet", "balanced", sonnetFallback); err != nil {
			return 1, err
		}
		haikuFallback := tierValue(override, "sonnet")
		if haikuFallback == "" {
			haikuFallback = sonnetFallback
		}
		if err := promptOne("haiku", "fast / cheap", haikuFallback); err != nil {
			return 1, err
		}
		smallFallback := tierValue(override, "haiku")
		if smallFallback == "" {
			smallFallback = haikuFallback
		}
		if err := promptOne("small", "background tasks", smallFallback); err != nil {
			return 1, err
		}

		if override.IsEmpty() {
			delete(c.Config.ProviderOverrides, provider.ID)
		} else {
			c.Config.ProviderOverrides[provider.ID] = override
		}
	}
	return persistConfig(c)
}

func tierValue(o config.ProviderOverride, tier string) string {
	switch tier {
	case "haiku":
		return o.Haiku
	case "sonnet":
		return o.Sonnet
	case "opus":
		return o.Opus
	case "small":
		return o.Small
	}
	return ""
}

func setTierField(o *config.ProviderOverride, tier, value string) {
	switch tier {
	case "haiku":
		o.Haiku = value
	case "sonnet":
		o.Sonnet = value
	case "opus":
		o.Opus = value
	case "small":
		o.Small = value
	}
}

func configOpenRouter(c Context) (int, error) {
	current := c.Secrets["OPENROUTER_API_KEY"]
	if current != "" {
		fmt.Fprintf(c.Output.Stdout, "Current key: %s\n", config.MaskSecret(current))
	}
	value, err := c.Prompt.PromptSecret("OpenRouter API key (empty to keep current)")
	if err != nil {
		return 1, err
	}
	if strings.TrimSpace(value) != "" {
		c.Secrets["OPENROUTER_API_KEY"] = value
	}
	for {
		model, err := c.Prompt.Prompt("Model ID (empty to stop)", "")
		if err != nil {
			return 1, err
		}
		if strings.TrimSpace(model) == "" {
			break
		}
		name, err := c.Prompt.Prompt("Alias", defaultAliasName(model))
		if err != nil {
			return 1, err
		}
		if !validName.MatchString(name) {
			return 1, fmt.Errorf("invalid alias %q", name)
		}
		c.Config.OpenRouterAliases[name] = model
	}
	return persistConfig(c)
}

func configCustom(c Context) (int, error) {
	name, err := c.Prompt.Prompt("Provider name", "")
	if err != nil {
		return 1, err
	}
	if !validName.MatchString(name) {
		return 1, fmt.Errorf("invalid provider name %q", name)
	}

	existing := c.Config.CustomProviders[name]

	urlLabel := "Base URL"
	if existing.BaseURL != "" {
		fmt.Fprintf(c.Output.Stdout, "Current URL: %s\n", existing.BaseURL)
		urlLabel = "Base URL (empty to keep current)"
	}
	baseURL, err := c.Prompt.Prompt(urlLabel, "")
	if err != nil {
		return 1, err
	}
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = existing.BaseURL
	}
	if baseURL == "" {
		return 1, fmt.Errorf("base URL is required")
	}

	modelLabel := "Default model (optional)"
	if existing.DefaultModel != "" {
		modelLabel = fmt.Sprintf("Default model (empty to keep %q)", existing.DefaultModel)
	}
	defaultModel, err := c.Prompt.Prompt(modelLabel, "")
	if err != nil {
		return 1, err
	}
	defaultModel = strings.TrimSpace(defaultModel)
	if defaultModel == "" {
		defaultModel = existing.DefaultModel
	}

	keyVar := strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_API_KEY"
	current := c.Secrets[keyVar]
	keyLabel := "API key"
	if current != "" {
		fmt.Fprintf(c.Output.Stdout, "Current key: %s\n", config.MaskSecret(current))
		keyLabel = "API key (empty to keep current)"
	}
	apiKey, err := c.Prompt.PromptSecret(keyLabel)
	if err != nil {
		return 1, err
	}
	if strings.TrimSpace(apiKey) != "" {
		c.Secrets[keyVar] = apiKey
	}

	c.Config.CustomProviders[name] = config.CustomProvider{
		Name:         name,
		DisplayName:  name,
		BaseURL:      baseURL,
		APIKeyEnv:    keyVar,
		DefaultModel: defaultModel,
	}
	return persistConfig(c)
}

func persistConfig(c Context) (int, error) {
	config.NormalizeLegacySecrets(c.Secrets, c.Catalog)
	c.Config.Normalize(c.Catalog)
	if err := config.SaveConfig(c.Paths.ConfigFile, c.Config); err != nil {
		return 1, err
	}
	if err := config.SaveSecrets(c.Paths.SecretsFile, c.Secrets); err != nil {
		return 1, err
	}
	execPath, execErr := os.Executable()
	if execErr != nil {
		return 1, execErr
	}
	if err := launchers.Sync(execPath, c.Paths, c.Catalog, c.Config, runtime.IsHomebrew()); err != nil {
		return 1, err
	}
	c.Output.Success("configuration saved")
	return 0, nil
}

func defaultAliasName(model string) string {
	model = strings.ToLower(model)
	if slash := strings.LastIndex(model, "/"); slash >= 0 {
		model = model[slash+1:]
	}
	model = strings.ReplaceAll(model, ".", "-")
	model = strings.ReplaceAll(model, "_", "-")
	return model
}

func resolveModelChoice(answer string, choices []providers.ModelChoice) string {
	answer = strings.TrimSpace(answer)
	if idx, err := strconv.Atoi(answer); err == nil {
		if idx >= 1 && idx <= len(choices) {
			return choices[idx-1].ID
		}
	}
	return answer
}
