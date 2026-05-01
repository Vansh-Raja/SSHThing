# SSHThing Agent Skills - Automated Setup

## Quick Start

Copy-paste this prompt into any AI coding assistant (Claude Code, OpenCode, Codex CLI, or any LLM with shell access):

```
Set up SSHThing agent skills for my AI coding assistant by following the instructions at: https://raw.githubusercontent.com/Vansh-Raja/SSHThing/main/skills/SETUP_PROMPT.md
```

That's it. The assistant will read this file and walk you through the rest.

---

## Setup Instructions (for the AI assistant reading this)

You are setting up SSHThing agent skills for a user's AI coding assistant. Follow these steps:

### 1. Ask the user which platform(s) to install

Ask: "Which AI coding assistant(s) do you want to set up SSHThing skills for?"

Options:
- **Claude Code** - installs to `~/.claude/skills/sshthing/`
- **OpenCode** - installs to `~/.config/opencode/skills/sshthing/`
- **Codex** - installs to `~/.codex/skills/sshthing/`
- **All** - installs for all three

### 2. Download and install skill files

For each selected platform, download the SKILL.md from the SSHThing repository and place it in the correct directory.

**Claude Code:**
```bash
mkdir -p ~/.claude/skills/sshthing
curl -fsSL https://raw.githubusercontent.com/Vansh-Raja/SSHThing/main/skills/claude-code/SKILL.md -o ~/.claude/skills/sshthing/SKILL.md
```

**OpenCode:**
```bash
mkdir -p ~/.config/opencode/skills/sshthing
curl -fsSL https://raw.githubusercontent.com/Vansh-Raja/SSHThing/main/skills/opencode/SKILL.md -o ~/.config/opencode/skills/sshthing/SKILL.md
```

**Codex:**
```bash
mkdir -p ~/.codex/skills/sshthing
curl -fsSL https://raw.githubusercontent.com/Vansh-Raja/SSHThing/main/skills/codex/SKILL.md -o ~/.codex/skills/sshthing/SKILL.md
mkdir -p ~/.codex/skills/sshthing/agents
curl -fsSL https://raw.githubusercontent.com/Vansh-Raja/SSHThing/main/skills/codex/agents/openai.yaml -o ~/.codex/skills/sshthing/agents/openai.yaml
```

### 3. Verify installation

After installing, verify the files exist:

```bash
# Check whichever platform(s) were installed:
cat ~/.claude/skills/sshthing/SKILL.md 2>/dev/null && echo "Claude Code: OK" || echo "Claude Code: not installed"
cat ~/.config/opencode/skills/sshthing/SKILL.md 2>/dev/null && echo "OpenCode: OK" || echo "OpenCode: not installed"
cat ~/.codex/skills/sshthing/SKILL.md 2>/dev/null && echo "Codex: OK" || echo "Codex: not installed"
```

### 4. Remind the user about prerequisites

Tell the user:

> Before using SSHThing skills, make sure you have:
>
> 1. **SSHThing installed** - download from [releases](https://github.com/Vansh-Raja/SSHThing/releases) or `brew install vansh-raja/tap/sshthing`
> 2. **At least one host configured** - open `sshthing` and add a server
> 3. **An automation token created** - in the TUI, go to Settings (`,`), open "Manage tokens", press `N` to create one, select hosts, and copy the token
> 4. **Token stored securely** - save it to a file:
>    ```bash
>    echo "stk_..." > ~/.sshthing/token.txt && chmod 600 ~/.sshthing/token.txt
>    ```
>
> Once set up, just ask naturally:
> - "Check disk space on my server"
> - "Restart nginx on production"
> - "Upload this build to the staging server"
> - "Pull the last 1000 lines of app.log from production"
> - "Apply this SQL file to the prod database"
>
> The agent uses `sshthing exec` for commands, `sshthing cp` / `put` / `get` for file transfer, and `sshthing exec --in <file>` for piping local files as remote stdin.

### 5. Done

Confirm to the user that skills are installed and ready to use. No restart is needed - the skills will be picked up automatically on the next conversation.
