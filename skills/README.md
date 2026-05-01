# SSHThing Agent Skills

Agent skills that let AI coding assistants run commands on your remote servers via SSHThing.

## What are skills?

Skills are instruction files that teach AI agents how to use tools. Once installed, your agent (Claude Code, OpenCode, Codex) can execute commands on your SSH servers using SSHThing tokens.

## Choose your platform

| Platform | Skill directory | Install path |
|---|---|---|
| **Claude Code** | `skills/claude-code/` | `~/.claude/skills/sshthing/` |
| **OpenCode** | `skills/opencode/` | `~/.config/opencode/skills/sshthing/` or `~/.claude/skills/sshthing/` |
| **Codex** | `skills/codex/` | `~/.codex/skills/sshthing/` or `.agents/skills/sshthing/` |

## Installation

### Option A: Let an LLM do it

Paste this into any AI coding assistant (Claude Code, OpenCode, Codex CLI, or any LLM with shell access):

```
Set up SSHThing agent skills for my AI coding assistant by following the instructions at: https://raw.githubusercontent.com/Vansh-Raja/SSHThing/main/skills/SETUP_PROMPT.md
```

The assistant will ask which platform(s) you want, download the right files, and verify the installation.

### Option B: Manual install

#### Claude Code

```bash
mkdir -p ~/.claude/skills/sshthing
cp skills/claude-code/SKILL.md ~/.claude/skills/sshthing/SKILL.md
```

#### OpenCode

```bash
mkdir -p ~/.config/opencode/skills/sshthing
cp skills/opencode/SKILL.md ~/.config/opencode/skills/sshthing/SKILL.md
```

#### Codex

```bash
mkdir -p ~/.codex/skills/sshthing
cp skills/codex/SKILL.md ~/.codex/skills/sshthing/SKILL.md
cp -r skills/codex/agents ~/.codex/skills/sshthing/agents
```

## Prerequisites

Before using the skill, you need:

1. **SSHThing installed** — download from [releases](https://github.com/Vansh-Raja/SSHThing/releases) or build with `go build -o sshthing ./cmd/sshthing`
2. **At least one host configured** — open `sshthing` and add a server
3. **A token created** — in the TUI, go to Tokens page (`Shift+Tab`), press `a`, name it, select hosts, copy the token
4. **Token stored in a file** — `echo "stk_..." > ~/.sshthing/token.txt && chmod 600 ~/.sshthing/token.txt`

## Usage

Once installed, just ask your agent naturally:

- "Check the disk space on my GPU Server"
- "Deploy the latest code to production"
- "Show me the nginx logs on the web server"
- "Restart the API service on the deployment server"
- "Upload this build artifact to the staging server"
- "Pull the last 1000 lines of `app.log` from production"
- "Run this SQL file against the prod database"

The agent will use `sshthing exec` to run remote commands, `sshthing cp` /
`put` / `get` to transfer files, and `sshthing exec --in <file>` to pipe a
local file as the remote command's stdin.

## Platform differences

| Feature | Claude Code | OpenCode | Codex |
|---|---|---|---|
| Auto-invocation control | `disable-model-invocation` field | Not supported | `agents/openai.yaml` |
| Tool pre-approval | `allowed-tools: Bash(sshthing *)` | Not enforced | Not supported |
| Manual invocation | `/sshthing` | `/sshthing` | `$sshthing` |
| Skill discovery | `~/.claude/skills/` | Multiple paths | `~/.codex/skills/`, `.agents/skills/` |
