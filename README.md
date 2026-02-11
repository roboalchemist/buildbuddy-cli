# buildbuddy-cli

Go CLI for the [BuildBuddy Enterprise API](https://www.buildbuddy.io/docs/enterprise-api). Query build invocations, targets, logs, actions, and artifacts from the command line.

Built for both humans and AI agents.

## Installation

```bash
brew tap roboalchemist/tap
brew install buildbuddy-cli
```

Or build from source:

```bash
git clone https://github.com/roboalchemist/buildbuddy-cli.git
cd buildbuddy-cli
make build
make install
```

## Setup

```bash
export BUILDBUDDY_API_KEY="your-api-key"
# Add to ~/.zshrc or ~/.bashrc for persistence
```

## Usage

### Invocations

```bash
buildbuddy-cli invocation list --commit <sha>       # List by commit
buildbuddy-cli invocation get <id>                   # Get details
```

### Targets

```bash
buildbuddy-cli target list <invocation-id>           # All targets
buildbuddy-cli target failed <invocation-id>         # Failed/flaky only
```

### Logs

```bash
buildbuddy-cli log get <invocation-id>               # Full log
buildbuddy-cli log get <id> --grep "ERROR"           # Filter lines
buildbuddy-cli log get <id> --tail 50                # Last N lines
buildbuddy-cli log get <id> -o build.log             # Save to file
```

### Actions & Artifacts

```bash
buildbuddy-cli action list <invocation-id>           # All actions
buildbuddy-cli artifact get <invocation-id>          # Test artifacts
buildbuddy-cli file get <bytestream-uri>             # Download file
```

### Workflows

```bash
buildbuddy-cli workflow run <action-name> --repo <url> --branch <name>
```

## Output Formats

| Flag | Format | Use case |
|------|--------|----------|
| (none) | Colored table | Human terminal |
| `--plaintext` | Tab-separated | Piping, agents |
| `-j` / `--json` | JSON | Programmatic parsing |
| `--jq EXPR` | jq-filtered JSON | Inline filtering |
| `--fields a,b` | Field-selected JSON | Reduce output |
| `--template TPL` | Go template | Custom formatting |
| `-o FILE` | File export | Large responses |

## Common Workflow: Investigate CI Failure

```bash
# 1. List invocations for the commit
buildbuddy-cli invocation list --commit <sha>

# 2. Find the failed invocation ID from the table

# 3. Get failed targets
buildbuddy-cli target failed <invocation-id>

# 4. Get the build log filtered to errors
buildbuddy-cli log get <invocation-id> --grep "FAIL\|ERROR" --tail 50
```

## Embedded Skill

buildbuddy-cli includes an embedded Claude Code skill for AI agent integration:

```bash
buildbuddy-cli skill print    # Print the skill markdown
buildbuddy-cli skill add      # Install to ~/.claude/skills/
```

## Testing

```bash
# Requires env vars:
#   BUILDBUDDY_API_KEY       - API auth
#   BUILDBUDDY_TEST_COMMIT   - a commit SHA with BuildBuddy invocations

# Smoke tests
make test

# Go integration tests
BUILDBUDDY_TEST_COMMIT=<sha> go test ./cmd/ -count=1 -timeout 5m
```

## License

[MIT](LICENSE)
