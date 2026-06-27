package commands

import (
	"github.com/saltyming/cproxy/internal/cli"
	"github.com/saltyming/cproxy/internal/config"
	"github.com/saltyming/cproxy/internal/providers"
	"github.com/saltyming/cproxy/internal/ui"
)

type Context struct {
	Paths   config.Paths
	Config  *config.File
	Secrets config.Secrets
	Catalog providers.Catalog
	Output  *ui.Output
	Prompt  *ui.Prompter
	Options cli.Options
}
