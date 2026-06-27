package commands

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saltyming/cproxy/internal/config"
	"github.com/saltyming/cproxy/internal/providers"
	"github.com/saltyming/cproxy/internal/ui"
)

func TestResolveModelChoiceMapsNumericSelections(t *testing.T) {
	t.Parallel()

	choices := []providers.ModelChoice{
		{ID: "glm-5.2"},
		{ID: "glm-4.7"},
	}

	if got := resolveModelChoice("1", choices); got != "glm-5.2" {
		t.Fatalf("resolveModelChoice(1) = %q, want glm-5.2", got)
	}
	if got := resolveModelChoice("glm-4.7", choices); got != "glm-4.7" {
		t.Fatalf("resolveModelChoice(glm-4.7) = %q", got)
	}
}

func TestResolveModelChoiceKeepsCustomModelID(t *testing.T) {
	t.Parallel()

	choices := []providers.ModelChoice{
		{ID: "MiniMax-M3"},
		{ID: "MiniMax-M2.5"},
	}

	if got := resolveModelChoice("MiniMax-M3-pro", choices); got != "MiniMax-M3-pro" {
		t.Fatalf("resolveModelChoice(MiniMax-M3-pro) = %q, want MiniMax-M3-pro", got)
	}
}

func TestConfigBuiltinPromptsFourTiers(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, ".local", "share"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, ".cache"))
	t.Setenv("CPROXY_BIN", filepath.Join(root, "bin"))

	paths, err := config.Detect("")
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := providers.Load()
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.File{
		Version:           1,
		ProviderOverrides: map[string]config.ProviderOverride{},
		OpenRouterAliases: map[string]string{},
		CustomProviders:   map[string]config.CustomProvider{},
	}

	// Use the real catalog provider so ModelChoices are populated.
	// minimax exposes MiniMax-M3 (1st), MiniMax-M2.7 (2nd),
	// MiniMax-M2.7-highspeed (3rd), MiniMax-M2.5 (4th), ... in
	// ModelChoices. The reader picks index 2 (M2.7) for opus, then
	// enters (cascade prefill) for the rest. Only the explicit pick
	// is persisted; the cascade chain is a runtime concern, so the
	// override stores just the opus tier.
	provider, ok := catalog.Get("minimax")
	if !ok {
		t.Fatal("minimax not in catalog")
	}

	ctx := Context{
		Paths:   paths,
		Config:  cfg,
		Secrets: config.Secrets{},
		Catalog: catalog,
		Output:  &ui.Output{Stdout: io.Discard, Stderr: io.Discard, Format: ui.FormatHuman},
		Prompt:  ui.NewPrompter(strings.NewReader("\n2\n\n\n\n"), io.Discard),
	}

	code, err := configBuiltin(ctx, provider)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("configBuiltin() code = %d, want 0", code)
	}
	override := cfg.ProviderOverrides["minimax"]
	if override.Opus != "MiniMax-M2.7" {
		t.Fatalf("override.Opus = %q, want MiniMax-M2.7 (user picked index 2)", override.Opus)
	}
	// Cascade tiers must NOT be persisted — the runtime cascade chain
	// (Sonnet → Opus, Haiku → Sonnet, Small → Haiku) is the source of
	// truth at launch time. Override fields stay empty.
	if override.Sonnet != "" {
		t.Fatalf("override.Sonnet = %q, want empty (cascade from opus)", override.Sonnet)
	}
	if override.Haiku != "" {
		t.Fatalf("override.Haiku = %q, want empty (cascade from sonnet)", override.Haiku)
	}
	if override.Small != "" {
		t.Fatalf("override.Small = %q, want empty (cascade from haiku)", override.Small)
	}
}

func TestConfigBuiltinAcceptsCatalogDefaultAcrossAllTiers(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, ".local", "share"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, ".cache"))
	t.Setenv("CPROXY_BIN", filepath.Join(root, "bin"))

	paths, err := config.Detect("")
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := providers.Load()
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.File{
		Version:           1,
		ProviderOverrides: map[string]config.ProviderOverride{},
		OpenRouterAliases: map[string]string{},
		CustomProviders:   map[string]config.CustomProvider{},
	}

	// "zai" has catalog tiers all == glm-5.2; pressing enter on every
	// prompt must cascade through the defaults and the resulting
	// override should match the catalog exactly, which Normalize
	// then drops. Empty reader (no answers at all) exercises the
	// press-enter-on-everything path.
	provider, _ := catalog.Get("zai")
	ctx := Context{
		Paths:   paths,
		Config:  cfg,
		Secrets: config.Secrets{},
		Catalog: catalog,
		Output:  &ui.Output{Stdout: io.Discard, Stderr: io.Discard, Format: ui.FormatHuman},
		Prompt:  ui.NewPrompter(strings.NewReader("\n\n\n\n"), io.Discard),
	}

	code, err := configBuiltin(ctx, provider)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("configBuiltin() code = %d, want 0", code)
	}
	if _, ok := cfg.ProviderOverrides["zai"]; ok {
		t.Fatalf("expected no override after accepting all defaults, got %+v", cfg.ProviderOverrides["zai"])
	}
}
