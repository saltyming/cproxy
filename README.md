<div align="center">
  <img src="docs/logo.png" alt="Cproxy logo" width="220" />
  <h1>Cproxy</h1>
  <p><strong>One CLI to switch between Claude Code providers instantly.</strong></p>
  <p>
    <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="MIT License" /></a>
    <a href="https://go.dev/"><img src="https://img.shields.io/badge/Language-Go-00ADD8.svg" alt="Go" /></a>
    <a href="#platform-support"><img src="https://img.shields.io/badge/Platform-macOS%20%7C%20Linux-lightgrey.svg" alt="Platform macOS and Linux" /></a>
    <a href="https://github.com/saltyming/cproxy/stargazers"><img src="https://img.shields.io/github/stars/saltyming/cproxy?style=social" alt="GitHub stars" /></a>
  </p>
</div>

<br/>

> **Fork notice:** Cproxy is a personal fork of [`jolehuit/clother`](https://github.com/jolehuit/clother) (upstream v3.0.9), maintained by [@saltyming](https://github.com/saltyming). It tracks the upstream while shipping current-generation model defaults (MiniMax-M3, glm-5.2, kimi-k2.7-code, deepseek-v4-flash) and independent fixes. Bug reports and PRs are welcome on this repository; upstream sync notes may be opened upstream when relevant.

<br/>

<div align="center">
  <img src="docs/demo-fast.gif" alt="Cproxy terminal demo" width="900" />
</div>

## Why Cproxy?

Switching Claude Code providers usually means changing env vars, endpoints, models, and launcher scripts by hand.
Cproxy gives you one install and one command pattern across Claude, Z.AI, Kimi, Alibaba, OpenRouter, local backends, China endpoints, and many other Anthropic-compatible providers.

## Table of Contents

- [Installation](#installation)
- [Core Usage](#core-usage)
- [Provider Reference](#provider-reference)
- [Troubleshooting](#troubleshooting)
- [VS Code Integration](#vs-code-integration)
- [Platform Support](#platform-support)
- [Under the Hood](#under-the-hood)
- [Contributors](#contributors)
- [Star History](#star-history)
- [License](#license)

## Installation

### Homebrew (macOS recommended)

```bash
# 1. Install Claude Code CLI
curl -fsSL https://claude.ai/install.sh | bash

# 2. Install Cproxy via tap
brew tap saltyming/tap
brew install cproxy

# 3. Start using it — all launchers are ready immediately
cproxy-native                          # Use your Claude Pro/Max/Team subscription
cproxy-zai                             # Z.AI (GLM-5)
cproxy-zai --yolo                      # Skip permission prompts
cproxy-kimi                            # Kimi (kimi-k2.5)
cproxy config                          # Configure providers
```

All `cproxy-*` provider launchers are installed directly into `$(brew --prefix)/bin` by the formula — no extra setup needed. `brew upgrade cproxy` keeps everything up to date.

**Update:**
```bash
cproxy update          # routes to brew upgrade under Homebrew
# or equivalently:
brew upgrade cproxy
```

### curl (macOS / Linux)

```bash
# 1. Install Claude Code CLI
curl -fsSL https://claude.ai/install.sh | bash

# 2. Install Cproxy
curl -fsSL https://raw.githubusercontent.com/saltyming/cproxy/main/scripts/install.sh | bash

# 3. Start using it
cproxy-native                          # Use your Claude Pro/Max/Team subscription
cproxy-zai                             # Z.AI (GLM-5)
cproxy-zai --yolo                      # Skip permission prompts
cproxy-kimi                            # Kimi (kimi-k2.5)
cproxy-ollama --model qwen3-coder      # Local with Ollama
cproxy config                          # Configure providers
```

**Update:**
```bash
cproxy update          # downloads and installs latest release
```

This installs:
- `cproxy`
- `cproxy-*` provider launchers
- resume compatibility for `claude --resume ...`

### Install Options

By default, Cproxy installs launchers to:
- the same directory as your existing `claude` binary, when `claude` is already on `PATH`
- otherwise **macOS**: `~/bin`
- otherwise **Linux**: `~/.local/bin` (XDG standard)

If the chosen bin directory is not on `PATH`, `cproxy install` prints a warning with the exact directory to add.

You can override this with `--bin-dir` or the `CPROXY_BIN` environment variable:

```bash
# Using --bin-dir flag
curl -fsSL https://raw.githubusercontent.com/saltyming/cproxy/main/scripts/install.sh | bash -s -- --bin-dir ~/.local/bin

# Using environment variable
export CPROXY_BIN="$HOME/.local/bin"
curl -fsSL https://raw.githubusercontent.com/saltyming/cproxy/main/scripts/install.sh | bash
```

Cproxy keeps `claude --resume ...` working with Cproxy features after install.

## Core Usage

### Commands

| Command | Description |
|---------|-------------|
| `cproxy config [provider]` | Configure provider |
| `cproxy list` | List profiles |
| `cproxy info <provider>` | Show provider details |
| `cproxy test` | Test connectivity |
| `cproxy status` | Installation status |
| `cproxy install` | Install/update Cproxy (create/refresh symlinks) |
| `cproxy update` | Update to latest version |
| `cproxy uninstall` | Remove everything |

### Update

```bash
cproxy update
```

Routes to `brew upgrade cproxy` under Homebrew, or downloads the latest release for curl installs. Also refreshes provider symlinks.

### Changing the Default Model

Each provider launcher comes with a default model (for example `glm-5` for Z.AI). You can override it in two ways:

```bash
# One-time: pass --model through to Claude CLI
cproxy-zai --model glm-4.7

# Permanent: configure the provider and pick a different default
cproxy config zai
```

Use `cproxy info <provider>` to inspect the resolved model.

### Resume

Cproxy keeps the resume command printed by Claude Code working across providers.

After a provider-launched session, Cproxy also prints a provider-aware reopen
command such as:

```bash
cproxy-kimi --resume <session-id>
```

When resuming a non-Claude session into native Claude, Cproxy temporarily
sanitizes incompatible non-Claude thinking blocks for the duration of that
single launch, then restores the original session file afterwards.

## Provider Reference

### Cloud

| Command | Provider | Model | API Key |
|---------|----------|-------|---------|
| `cproxy-native` | Anthropic | Claude | Your subscription |
| `cproxy-zai` | Z.AI | GLM-5 | [z.ai](https://z.ai) |
| `cproxy-minimax` | MiniMax | MiniMax-M2.7 | [minimax.io](https://minimax.io) |
| `cproxy-kimi` | Kimi | kimi-k2.5 | [kimi.com](https://kimi.com) |
| `cproxy-moonshot` | Moonshot AI | kimi-k2.5 | [moonshot.ai](https://moonshot.ai) |
| `cproxy-deepseek` | DeepSeek | deepseek-chat | [deepseek.com](https://platform.deepseek.com) |
| `cproxy-mimo` | Xiaomi MiMo | mimo-v2-pro | [xiaomimimo.com](https://platform.xiaomimimo.com) |
| `cproxy-alibaba` | Alibaba Coding Plan | qwen3.5-plus | [modelstudio](https://modelstudio.console.alibabacloud.com) |
| `cproxy-alibaba-us` | Alibaba Coding Plan (US) | qwen3.5-plus | [modelstudio](https://modelstudio.console.alibabacloud.com) |

### OpenRouter (100+ Models)

OpenRouter launchers follow the `cproxy-or-<alias>` naming pattern.
For example, if you alias `moonshotai/kimi-k2.5` to `kimi-k25`, the launcher becomes `cproxy-or-kimi-k25`.

```bash
cproxy config openrouter               # Set API key + add models
# Example: alias moonshotai/kimi-k2.5 as kimi-k25
cproxy-or-kimi-k25                     # Use it
```

> **Tip**: Find model IDs on [openrouter.ai/models](https://openrouter.ai/models) — click the copy icon next to any model name.

> If a model doesn't work as expected, try the `:exacto` variant (e.g. `moonshotai/kimi-k2-0905:exacto`) which provides better tool calling support.

### China Endpoints

| Command | Provider | Endpoint |
|---------|----------|----------|
| `cproxy-zai-cn` | Z.AI China | open.bigmodel.cn |
| `cproxy-minimax-cn` | MiniMax China | api.minimaxi.com |
| `cproxy-ve` | Volcengine | ark.cn-beijing.volces.com |
| `cproxy-alibaba-cn` | Alibaba China | coding.dashscope.aliyuncs.com |

### Local (No API Key)

| Command | Provider | Port | Setup |
|---------|----------|------|-------|
| `cproxy-ollama` | Ollama | 11434 | [ollama.com](https://ollama.com) |
| `cproxy-lmstudio` | LM Studio | 1234 | [lmstudio.ai](https://lmstudio.ai) |
| `cproxy-llamacpp` | llama.cpp | 8000 | [github.com/ggml-org/llama.cpp](https://github.com/ggml-org/llama.cpp) |

```bash
# Ollama
ollama pull qwen3-coder && ollama serve
cproxy-ollama --model qwen3-coder

# LM Studio
cproxy-lmstudio --model <model>

# llama.cpp
./llama-server --model model.gguf --port 8000 --jinja
cproxy-llamacpp --model <model>
```

### Custom

```bash
cproxy config custom
cproxy-myprovider                      # Ready
```

### Alibaba Coding Plan Models

All Alibaba variants (`alibaba`, `alibaba-us`, `alibaba-cn`) share the same API key and support these models:

| Model |
|-------|
| `qwen3.5-plus` (default) |
| `kimi-k2.5` |
| `glm-5` |
| `MiniMax-M2.5` |
| `qwen3-coder-next` |
| `qwen3-coder-plus` |
| `qwen3-max-2026-01-23` |
| `glm-4.7` |

Switch models with `--model`:

```bash
cproxy-alibaba --model kimi-k2.5
cproxy-alibaba --model glm-5
cproxy-alibaba-cn --model qwen3-coder-next
```

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `claude: command not found` | Install Claude CLI first |
| `cproxy: command not found` | Run `cproxy status` to see the installed bin dir, then add that directory to `PATH` and restart your shell |
| `claude --resume ...` does not behave like Cproxy | Restart your shell, then run `cproxy install` again |
| `--yolo` is not recognized | Restart your shell, then run `cproxy install` again |
| `API key not set` | Run `cproxy config` |

## VS Code Integration

Cproxy works with the official **Claude Code** extension.
Use Claude Code extension `2.6+`.

To configure it:

1. Open VS Code Settings (`Cmd+,` or `Ctrl+,`).
2. Search for **"Claude Process Wrapper"** (`claudeProcessWrapper`).
3. Set it to the **full path** of your chosen launcher:
   - macOS: `/Users/yourname/bin/cproxy-zai`
   - Linux: `/home/yourname/.local/bin/cproxy-zai`
4. Reload VS Code.

> **Note**: Requires Cproxy v2.6+ (which handles non-interactive shell output correctly).

## Platform Support

macOS (zsh/bash) • Linux (zsh/bash) • Windows (WSL)

## Under the Hood

### How It Works

Cproxy is a single Go binary. The installer downloads the release artifact,
installs `cproxy` into your bin directory, then creates:
- `cproxy-*` symlinks for providers
- a `claude` shim symlink for resume compatibility

At runtime, the binary resolves the selected profile from its own invocation
name, loads config and secrets, sets the required Anthropic-compatible
environment variables, then launches the real Claude binary outside the Cproxy
bin directory.

Example for `cproxy-zai`:

```bash
export ANTHROPIC_BASE_URL="https://api.z.ai/api/anthropic"
export ANTHROPIC_AUTH_TOKEN="$ZAI_API_KEY"
exec /path/to/the/real/claude "$@"
```

API keys stored in `~/.local/share/cproxy/secrets.env` (chmod 600).

`--yolo` is accepted by Cproxy launchers and by the Cproxy `claude` shim as
shorthand for `--dangerously-skip-permissions`.

### Local Release Testing

Test the binary installer locally against a local directory or server:

```bash
CPROXY_RELEASE_BASE_URL=http://127.0.0.1:8000 \
  ./scripts/install.sh install
```

## Contributors

- [@darkokoa](https://github.com/darkokoa) — China endpoints
- [@RawToast](https://github.com/RawToast) — Kimi endpoint fix
- [@sammcj](https://github.com/sammcj) — Security hardening
- [@aprakasa](https://github.com/aprakasa) — Linux compatibility fixes in `load_secrets()`
- [@luciano-fiandesio](https://github.com/luciano-fiandesio) — Install directory improvement (issue)

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=saltyming/cproxy&type=Date)](https://www.star-history.com/#saltyming/cproxy&Date)

## License

MIT © [saltyming](https://github.com/saltyming) — forked from [jolehuit/clother](https://github.com/jolehuit/clother)
