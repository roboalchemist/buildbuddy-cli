# buildbuddy-cli - Command Reference

**Version**: 0.1.0 | **Binary**: `buildbuddy-cli` | **API**: BuildBuddy Enterprise REST API

## Contents
- [invocation](#invocation) - get, list
- [target](#target) - list, failed
- [log](#log) - get
- [action](#action) - list
- [artifact](#artifact) - get
- [file](#file) - get, delete
- [workflow](#workflow) - run
- [Global Flags](#global-flags)
- [API Endpoints](#api-endpoints)

## invocation

### invocation get <id>

Get a specific invocation by ID. Accepts full BuildBuddy URLs (strips prefix automatically).

```bash
buildbuddy-cli invocation get 5da90d0f-054d-4842-952a-2d9cf50ad26a
buildbuddy-cli invocation get https://app.buildbuddy.io/invocation/5da90d0f-054d-4842-952a-2d9cf50ad26a
```

Aliases: `show`

### invocation list --commit <sha>

List invocations for a commit SHA (short or full).

```bash
buildbuddy-cli invocation list --commit a3930f89299aae864e6b50625d45f0bef6401a32
buildbuddy-cli inv ls --commit a3930f8 --limit 5
```

Aliases: `ls`

| Flag | Short | Description |
|------|-------|-------------|
| `--commit` | | Commit SHA (required) |
| `--limit` | `-n` | Max results (default: all) |

## target

### target list <invocation-id>

List all targets for an invocation.

```bash
buildbuddy-cli target list 5da90d0f-054d-4842-952a-2d9cf50ad26a
```

Aliases: `ls`

### target failed <invocation-id>

Show only failed, flaky, timed-out, or build-failed targets.

```bash
buildbuddy-cli target failed 5da90d0f-054d-4842-952a-2d9cf50ad26a
```

Status values: `PASSED`, `FAILED`, `FLAKY`, `TIMED_OUT`, `BUILT`, `BUILD_FAILED`, `SKIPPED`, `INCOMPLETE`

## log

### log get <invocation-id>

Get build log for an invocation. Supports filtering and file export.

```bash
buildbuddy-cli log get <id>
buildbuddy-cli log get <id> --grep "ERROR"
buildbuddy-cli log get <id> --tail 50
buildbuddy-cli log get <id> --head 20
buildbuddy-cli log get <id> -o build.log
```

| Flag | Description |
|------|-------------|
| `--grep` | Filter lines containing pattern |
| `--tail N` | Show last N lines |
| `--head N` | Show first N lines |

## action

### action list <invocation-id>

List all build actions for an invocation.

```bash
buildbuddy-cli action list <id>
buildbuddy-cli action ls <id> --json
```

Aliases: `ls`

Table columns: TARGET, MNEMONIC, DURATION, CACHE (HIT/MISS/UPLOAD), EXIT

## artifact

### artifact get <invocation-id>

Get test targets for an invocation (targets with test-related status).

```bash
buildbuddy-cli artifact get <id>
```

## file

### file get <bytestream-uri>

Download a file by bytestream URI.

```bash
buildbuddy-cli file get <uri>
buildbuddy-cli file get <uri> -o output.bin
```

### file delete <bytestream-uri>

Delete a cached file.

```bash
buildbuddy-cli file delete <uri>
```

## workflow

### workflow run <action-name>

Execute a workflow action.

```bash
buildbuddy-cli workflow run "Build" --repo https://github.com/org/repo --branch main
```

| Flag | Description |
|------|-------------|
| `--repo` | Repository URL |
| `--branch` | Branch name |
| `--commit` | Commit SHA |
| `--visibility` | Visibility (e.g., PUBLIC) |
| `--async` | Run asynchronously |

## Global Flags

Available on all commands:

| Flag | Short | Description |
|------|-------|-------------|
| `--json` | `-j` | JSON output |
| `--plaintext` | | Tab-separated for piping |
| `--jq EXPR` | | Inline jq filter (implies --json) |
| `--fields LIST` | | Comma-separated field selection (implies --json) |
| `--template TPL` | `-t` | Go template formatting |
| `--output FILE` | `-o` | Write to file instead of stdout |
| `--no-color` | | Disable ANSI colors |
| `--debug` | | Verbose logging to stderr |
| `--version` | `-v` | Print version |

## API Endpoints

All requests are POST to `https://app.buildbuddy.io/api/v1/<endpoint>` with `x-buildbuddy-api-key` header.

| CLI Command | API Endpoint |
|-------------|-------------|
| `invocation get/list` | `GetInvocation` |
| `target list/failed` | `GetTarget` |
| `log get` | `GetLog` |
| `action list` | `GetAction` |
| `file get` | `GetFile` |
| `file delete` | `DeleteFile` |
| `workflow run` | `ExecuteWorkflow` |
