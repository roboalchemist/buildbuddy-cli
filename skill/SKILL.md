---
name: buildbuddy-cli
description: "CLI for the BuildBuddy Enterprise REST API. Query build invocations, targets, logs, actions, and artifacts. Use when looking up CI builds, debugging failed targets, fetching build logs, or analyzing invocations by commit SHA."
---

# buildbuddy-cli

CLI for the [BuildBuddy Enterprise API](https://www.buildbuddy.io/docs/enterprise-api) -- invocations, targets, logs, actions, artifacts, and files. Built for both humans and AI agents.

**Version**: 0.1.0 | **Binary**: `buildbuddy-cli` | **Auth**: `BUILDBUDDY_API_KEY` env var

## Setup

```bash
brew install buildbuddy-cli
export BUILDBUDDY_API_KEY="your-api-key"
# Add to ~/.zshrc for persistence
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

<examples>
<example>
Task: List CI invocations for a commit

```bash
buildbuddy-cli invocation list --commit abc123def456
```

Output:
```
ID                                     STATUS   DURATION   COMMAND        BRANCH              USER
5da90d0f-054d-4842-952a-2d9cf50ad26a   PASS     11s        test           feature/my-branch   buildbuddy
e3c1e54c-addf-4075-aa86-311ebb025f6a   FAIL     2m15s      build          feature/my-branch   buildbuddy
```
</example>

<example>
Task: Get invocation details as JSON for parsing

```bash
buildbuddy-cli invocation get 5da90d0f-054d-4842-952a-2d9cf50ad26a --json
```

Output (truncated):
```json
{
  "invocation": [{
    "id": {"invocationId": "5da90d0f-054d-4842-952a-2d9cf50ad26a"},
    "success": true,
    "command": "test",
    "durationUsec": 11647000,
    "branchName": "feature/my-branch"
  }]
}
```
</example>

<example>
Task: Find failed targets in a build

```bash
buildbuddy-cli target failed <invocation-id>
```

Output:
```
LABEL                                STATUS
//src/test/unit:test_auth            FAILED
//src/test/integration:test_api      TIMED_OUT
```
</example>

<example>
Task: Get last 20 lines of a build log

```bash
buildbuddy-cli log get <invocation-id> --tail 20
```

Output:
```
INFO: Build completed successfully, 7 total actions
Executed 0 out of 1 test: 1 test passes.
```
</example>

<example>
Task: Extract invocation ID with jq

```bash
buildbuddy-cli invocation list --commit <sha> --jq '.invocation[0].id.invocationId'
```

Output:
```
"5da90d0f-054d-4842-952a-2d9cf50ad26a"
```
</example>

<example>
Task: Save full build log to file

```bash
buildbuddy-cli log get <invocation-id> -o build.log
```

Output (stderr):
```
Wrote build.log (3.2MB)
```
</example>
</examples>

## Quick Reference

### Invocations

```bash
buildbuddy-cli invocation list --commit <sha>     # List by commit (alias: inv ls)
buildbuddy-cli invocation list --commit <sha> --limit 5  # Cap results
buildbuddy-cli invocation get <id>                 # Get details (alias: inv show)
```

### Targets

```bash
buildbuddy-cli target list <invocation-id>         # All targets (alias: target ls)
buildbuddy-cli target failed <invocation-id>       # Failed/flaky only
```

### Logs

```bash
buildbuddy-cli log get <invocation-id>             # Full log
buildbuddy-cli log get <id> --grep "ERROR"         # Filter lines
buildbuddy-cli log get <id> --tail 50              # Last N lines
buildbuddy-cli log get <id> --head 20              # First N lines
buildbuddy-cli log get <id> -o build.log           # Save to file
```

### Actions

```bash
buildbuddy-cli action list <invocation-id>         # All actions (alias: action ls)
```

### Artifacts & Files

```bash
buildbuddy-cli artifact get <invocation-id>        # Test artifacts
buildbuddy-cli file get <bytestream-uri>           # Download file
buildbuddy-cli file get <uri> -o output.bin        # Save to file
buildbuddy-cli file delete <bytestream-uri>        # Delete cached file
```

### Workflows

```bash
buildbuddy-cli workflow run <action-name> --repo <url> --branch <name>
```

### Skill Management

```bash
buildbuddy-cli skill print                         # Print embedded SKILL.md
buildbuddy-cli skill add                           # Install to ~/.claude/skills/
```

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--json` | `-j` | JSON output |
| `--plaintext` | | Tab-separated |
| `--jq` | | Inline jq filter (implies --json) |
| `--fields` | | Comma-separated field selection (implies --json) |
| `--template` | `-t` | Go template |
| `--output` | `-o` | Write to file |
| `--no-color` | | Disable ANSI |
| `--debug` | | Verbose stderr |

## Common Workflow: Investigate CI Failure

1. List invocations for the commit: `buildbuddy-cli inv ls --commit <sha>`
2. Find the failed invocation ID from the table
3. Get failed targets: `buildbuddy-cli target failed <id>`
4. Get the build log: `buildbuddy-cli log get <id> --grep "FAIL\|ERROR" --tail 50`
5. For full log: `buildbuddy-cli log get <id> -o /tmp/build.log`

## Verify Setup

```bash
buildbuddy-cli --version                    # Should print version
buildbuddy-cli invocation list --commit abc123 --json  # Should return {} (empty, no error)
```

If you get `BUILDBUDDY_API_KEY not set`, the env var is missing.

## Agent Best Practices

1. **Always use `--json`** for programmatic parsing
2. **Use `--jq`** to extract specific values without external tools
3. **Use `--limit`** to cap results and reduce tokens
4. **Use `-o`** for large outputs (logs) to avoid flooding context
5. **Invocation IDs** are UUIDs; commit SHAs can be short or full
6. **URL stripping**: `invocation get` accepts full BuildBuddy URLs

## Gotchas

- **API uses proto3 JSON**: field names are camelCase (`invocationId`, `durationUsec`)
- **int64 as strings**: Duration and timestamp fields are string-encoded in JSON
- **No SearchInvocation endpoint**: Use `invocation list --commit` (wraps GetInvocation with commit_sha selector)
- **Invocation ID is nested**: Access via `.invocation[0].id.invocationId` not `.invocation[0].invocationId`
- **Target status is string**: `"PASSED"`, `"FAILED"`, `"FLAKY"`, `"BUILT"` (not numeric codes)

See [reference/commands.md](reference/commands.md) for complete command details.
