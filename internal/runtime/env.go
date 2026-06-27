package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/saltyming/cproxy/internal/config"
	"github.com/saltyming/cproxy/internal/profiles"
	"github.com/saltyming/cproxy/internal/providers"
)

// IsHomebrew reports whether the running binary is managed by Homebrew.
//
// It first checks the HOMEBREW_PREFIX env var (set during `brew install`),
// then falls back to inspecting whether the resolved executable path lives
// inside a Homebrew Cellar directory — which is the case when the user runs
// a Homebrew-installed binary from their normal shell session.
func IsHomebrew() bool {
	if os.Getenv("HOMEBREW_PREFIX") != "" {
		return true
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		resolved = exe
	}
	return strings.Contains(resolved, "/Cellar/")
}

func BuildEnv(target profiles.Target, secrets config.Secrets) ([]string, error) {
	envMap := map[string]string{}
	for _, pair := range os.Environ() {
		key, value, ok := splitEnv(pair)
		if ok {
			envMap[key] = value
		}
	}
	clearAnthropicEnv(envMap)

	if target.BaseURL != "" {
		envMap["ANTHROPIC_BASE_URL"] = target.BaseURL
	}
	// ANTHROPIC_MODEL is intentionally not emitted: tier variables
	// (ANTHROPIC_DEFAULT_{HAIKU,SONNET,OPUS}_MODEL) carry the model
	// identity and override any ANTHROPIC_MODEL the parent shell may
	// have leaked. Overrides in profiles.Resolve rewrite every tier
	// to the same model, so the per-call main-model selection is
	// already covered without ANTHROPIC_MODEL.
	for key, value := range target.ModelTiers {
		cw := target.ModelContextWindows[key]
		annotated := WithContextSuffix(value, cw)
		switch key {
		case "haiku":
			envMap["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = annotated
		case "sonnet":
			envMap["ANTHROPIC_DEFAULT_SONNET_MODEL"] = annotated
		case "opus":
			envMap["ANTHROPIC_DEFAULT_OPUS_MODEL"] = annotated
		case "small":
			envMap["ANTHROPIC_SMALL_FAST_MODEL"] = annotated
		}
	}
	// CLAUDE_CODE_AUTO_COMPACT_WINDOW must be the largest declared
	// window across all tiers (and any default override) so /compact
	// waits for the actual ceiling rather than the smallest tier's.
	var maxTokens int
	for _, cw := range target.ModelContextWindows {
		if t, err := strconv.Atoi(CompactWindowTokens(cw)); err == nil && t > maxTokens {
			maxTokens = t
		}
	}
	if maxTokens > 0 {
		envMap["CLAUDE_CODE_AUTO_COMPACT_WINDOW"] = strconv.Itoa(maxTokens)
	}

	switch target.AuthMode {
	case providers.AuthNone:
	case providers.AuthLiteral:
		envMap["ANTHROPIC_AUTH_TOKEN"] = target.LiteralAuthToken
		envMap["ANTHROPIC_API_KEY"] = ""
	case providers.AuthSecret:
		value := secrets[target.SecretKey]
		if value == "" {
			return nil, fmt.Errorf("%s not configured", target.SecretKey)
		}
		envMap["ANTHROPIC_AUTH_TOKEN"] = value
		if target.Family == providers.FamilyOpenRouter || target.Family == providers.FamilyLocal || target.Family == providers.FamilyCustomUnknown {
			envMap["ANTHROPIC_API_KEY"] = ""
		}
	default:
		return nil, fmt.Errorf("unsupported auth mode %q", target.AuthMode)
	}

	return flattenEnv(envMap), nil
}

// WithContextSuffix appends "[1m]" to modelID when ctxWindow names the
// 1M variant, mirroring Claude Code's own convention for its built-in
// models. The Anthropic-format provider on the other end of the gateway
// is expected to translate or strip the suffix as needed; for non-1M
// windows (or empty values) the model ID is returned unchanged.
func WithContextSuffix(modelID, ctxWindow string) string {
	if modelID == "" {
		return ""
	}
	if strings.EqualFold(strings.TrimSpace(ctxWindow), "1M") {
		return modelID + "[1m]"
	}
	return modelID
}

// CompactWindowTokens converts a catalog context window string like
// "200K", "262K", or "1M" into a raw token count string ("200000",
// "262000", "1000000") suitable for CLAUDE_CODE_AUTO_COMPACT_WINDOW.
// Unrecognized values return "" so the caller can omit the env entry.
func CompactWindowTokens(ctxWindow string) string {
	s := strings.TrimSpace(ctxWindow)
	if s == "" {
		return ""
	}
	upper := strings.ToUpper(s)
	switch {
	case strings.HasSuffix(upper, "M"):
		n, err := strconv.Atoi(strings.TrimSuffix(upper, "M"))
		if err != nil {
			return ""
		}
		return strconv.Itoa(n * 1_000_000)
	case strings.HasSuffix(upper, "K"):
		n, err := strconv.Atoi(strings.TrimSuffix(upper, "K"))
		if err != nil {
			return ""
		}
		return strconv.Itoa(n * 1_000)
	}
	return ""
}

func clearAnthropicEnv(envMap map[string]string) {
	for key := range envMap {
		if strings.HasPrefix(key, "ANTHROPIC_") {
			delete(envMap, key)
		}
	}
}

func splitEnv(pair string) (string, string, bool) {
	for i := 0; i < len(pair); i++ {
		if pair[i] == '=' {
			return pair[:i], pair[i+1:], true
		}
	}
	return "", "", false
}

func flattenEnv(envMap map[string]string) []string {
	env := make([]string, 0, len(envMap))
	for key, value := range envMap {
		env = append(env, key+"="+value)
	}
	return env
}
