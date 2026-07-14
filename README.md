# gh-codex-skillset

A GitHub CLI extension that controls user-scoped Codex skills per Git repository.

It reads skills from `$HOME/.agents/skills`, stores project-specific state in `.codex/user-skills.toml`, and launches Codex with session-only `skills.config` overrides. It never modifies `~/.codex/config.toml`.

## Install

````bash
gh extension install f4ah6o/gh-codex-skillset
````

## Usage

Initialize project-local configuration:

````bash
gh codex-skillset init
````

List user-scoped skills and their state in the current repository:

````bash
gh codex-skillset list
````

Disable or enable skills:

````bash
gh codex-skillset disable pdfs slides spreadsheets
gh codex-skillset enable spreadsheets
````

Launch Codex with the project configuration:

````bash
gh codex-skillset run
````

Forward arguments to Codex after `--`:

````bash
gh codex-skillset run -- exec "review this repository"
````

Inspect the generated command without starting Codex:

````bash
gh codex-skillset run --dry-run
````

Validate the environment:

````bash
gh codex-skillset doctor
````

## Configuration

`.codex/user-skills.toml` uses a denylist:

````toml
version = 1

disabled = [
  "pdfs",
  "slides",
]
````

Skill names are directory names directly under `$HOME/.agents/skills`.

At launch, the extension passes an explicit `enabled=true` or `enabled=false` session override for every discovered user-scoped skill. This lets the repository configuration override matching entries in the user's global Codex configuration without changing that file.

## Commands

| Command | Purpose |
|---|---|
| `init` | Create `.codex/user-skills.toml` |
| `list` | Show user skills and project state |
| `enable` | Remove skills from the project denylist |
| `disable` | Add skills to the project denylist |
| `run` | Launch Codex with session overrides |
| `doctor` | Validate repository, config, skills, and Codex |
| `version` | Print the extension version |

Run `gh codex-skillset <command> --help` for command-specific options.

## Scope

This extension only manages user-scoped skills in `$HOME/.agents/skills`. It does not install, update, remove, or modify skills, and it does not manage repository, admin, system, or plugin-provided skills.

See [`docs/initial-plan.md`](docs/initial-plan.md) for the original specification.
