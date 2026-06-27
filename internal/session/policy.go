package session

import "github.com/saltyming/cproxy/internal/providers"

func RequiresClaudeSanitization(family providers.Family) bool {
	return family == providers.FamilyClaudeStrict
}
