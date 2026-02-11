# buildbuddy-cli

Go CLI for the BuildBuddy Enterprise API. Binary name: `buildbuddy-cli`.

## Quick Start

```bash
export BUILDBUDDY_API_KEY="your-api-key"  # or add to ~/.zshrc
make build
./buildbuddy-cli invocation list --commit <sha>
```

## Project Structure

```
main.go          → entry point, version injection
cmd/root.go      → cobra root, global flags (--json, --plaintext, --jq, --output)
cmd/helpers.go   → newClient(), formatDuration(), truncateStr()
cmd/invocation.go → invocation get/list
cmd/target.go    → target list/failed
cmd/log.go       → log get (--grep, --tail, --head)
cmd/action.go    → action list
cmd/artifact.go  → test artifact retrieval
cmd/file.go      → file get/delete
cmd/workflow.go  → workflow run
pkg/api/client.go → HTTP POST client with x-buildbuddy-api-key header
pkg/api/types.go  → request/response structs (proto3 camelCase JSON)
pkg/auth/auth.go  → BUILDBUDDY_API_KEY env var loading
pkg/output/       → JSON/table/plaintext rendering, jq, prune, errors
```

## API Notes

- All endpoints are POST to `https://app.buildbuddy.io/api/v1/<EndpointName>`
- Auth header: `x-buildbuddy-api-key`
- Response uses proto3 JSON (camelCase fields, int64 as strings)
- Available endpoints: GetInvocation, GetTarget, GetLog, GetAction, GetFile, DeleteFile, ExecuteWorkflow, Run
- No SearchInvocation endpoint; use GetInvocation with commit_sha selector

## Building

```bash
make build      # builds ./buildbuddy-cli
make test       # runs smoke_test.sh
make install    # installs to /usr/local/bin
```

## Testing

```bash
# Requires both env vars:
#   BUILDBUDDY_API_KEY       - API auth
#   BUILDBUDDY_TEST_COMMIT   - a commit SHA that has BuildBuddy invocations

# Smoke tests (bash)
BUILDBUDDY_TEST_COMMIT=<sha> ./smoke_test.sh

# Go integration tests
BUILDBUDDY_TEST_COMMIT=<sha> go test ./cmd/ -count=1 -timeout 5m
```
