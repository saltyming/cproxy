package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/saltyming/cproxy/internal/profiles"
)

func runInfo(_ context.Context, c Context, args []string) (int, error) {
	if len(args) == 0 {
		return 1, fmt.Errorf("usage: cproxy info <provider>")
	}
	target, err := profiles.Resolve(args[0], c.Catalog, c.Config)
	if err != nil {
		return 1, err
	}
	if c.Options.Format == "json" {
		data, _ := json.MarshalIndent(target, "", "  ")
		fmt.Fprintln(c.Output.Stdout, string(data))
		return 0, nil
	}
	c.Output.Header("Provider Info")
	fmt.Fprintf(c.Output.Stdout, "Profile:     %s\n", target.Profile)
	fmt.Fprintf(c.Output.Stdout, "Name:        %s\n", target.DisplayName)
	fmt.Fprintf(c.Output.Stdout, "Family:      %s\n", target.Family)
	fmt.Fprintf(c.Output.Stdout, "Base URL:    %s\n", target.BaseURL)
	if target.Model != "" {
		line := target.Model
		if cw, ok := target.ModelContextWindows["default"]; ok && cw != "" {
			line = fmt.Sprintf("%s (%s context)", target.Model, cw)
		}
		fmt.Fprintf(c.Output.Stdout, "Model:       %s\n", line)
	}
	if len(target.ModelTiers) > 0 {
		fmt.Fprintln(c.Output.Stdout, "Tiers:")
		tierOrder := []string{"haiku", "sonnet", "opus", "small"}
		seen := map[string]bool{}
		for _, tier := range tierOrder {
			modelID, ok := target.ModelTiers[tier]
			if !ok {
				continue
			}
			seen[tier] = true
			cw := target.ModelContextWindows[tier]
			if cw != "" {
				fmt.Fprintf(c.Output.Stdout, "  %-7s  %s (%s)\n", tier+":", modelID, cw)
			} else {
				fmt.Fprintf(c.Output.Stdout, "  %-7s  %s\n", tier+":", modelID)
			}
		}
		// Any tiers outside the standard order
		for tier, modelID := range target.ModelTiers {
			if seen[tier] {
				continue
			}
			cw := target.ModelContextWindows[tier]
			if cw != "" {
				fmt.Fprintf(c.Output.Stdout, "  %-7s  %s (%s)\n", tier+":", modelID, cw)
			} else {
				fmt.Fprintf(c.Output.Stdout, "  %-7s  %s\n", tier+":", modelID)
			}
		}
	}
	if target.SecretKey != "" {
		status := "configured"
		if c.Secrets[target.SecretKey] == "" {
			status = "not configured"
		}
		fmt.Fprintf(c.Output.Stdout, "Credential:  %s (%s)\n", target.SecretKey, status)
	}
	return 0, nil
}
