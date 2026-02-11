#!/usr/bin/env bash
set -euo pipefail

# Smoke tests for buildbuddy-cli
# Requires BUILDBUDDY_API_KEY to be set

BINARY="./buildbuddy-cli"
PASS=0
FAIL=0
COMMIT_SHA="${BUILDBUDDY_TEST_COMMIT:?Set BUILDBUDDY_TEST_COMMIT to a commit SHA with BuildBuddy invocations}"

pass() { echo "  PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "  FAIL: $1"; FAIL=$((FAIL + 1)); }

run_test() {
    local name="$1"
    shift
    if output=$("$@" 2>&1); then
        pass "$name"
    else
        fail "$name (exit=$?): $output"
    fi
}

echo "=== buildbuddy-cli smoke tests ==="
echo ""

# Check binary exists
if [ ! -f "$BINARY" ]; then
    echo "Binary not found. Building..."
    make build
fi

# Version
echo "[version]"
run_test "buildbuddy-cli --version" $BINARY --version

# Help
echo "[help]"
run_test "buildbuddy-cli --help" $BINARY --help
run_test "buildbuddy-cli invocation --help" $BINARY invocation --help
run_test "buildbuddy-cli target --help" $BINARY target --help
run_test "buildbuddy-cli log --help" $BINARY log --help

# Invocation commands
echo "[invocation]"
run_test "invocation list --commit (table)" $BINARY invocation list --commit "$COMMIT_SHA"
run_test "invocation list --commit (json)" $BINARY invocation list --commit "$COMMIT_SHA" -j

# Extract an invocation ID for further tests
INV_ID=$($BINARY invocation list --commit "$COMMIT_SHA" --jq '.invocation[0].id.invocationId' 2>/dev/null | tr -d '"')
if [ -n "$INV_ID" ]; then
    echo "  (using invocation ID: $INV_ID)"
    run_test "invocation get (table)" $BINARY invocation get "$INV_ID"
    run_test "invocation get (json)" $BINARY invocation get "$INV_ID" -j
else
    fail "could not extract invocation ID"
fi

# Target commands
echo "[target]"
if [ -n "$INV_ID" ]; then
    run_test "target list" $BINARY target list "$INV_ID"
    run_test "target list (json)" $BINARY target list "$INV_ID" -j
    run_test "target failed" $BINARY target failed "$INV_ID"
fi

# Log commands
echo "[log]"
if [ -n "$INV_ID" ]; then
    run_test "log get" $BINARY log get "$INV_ID"
    run_test "log get --tail 5" $BINARY log get "$INV_ID" --tail 5
    run_test "log get (json)" $BINARY log get "$INV_ID" -j
fi

# Action commands
echo "[action]"
if [ -n "$INV_ID" ]; then
    run_test "action list" $BINARY action list "$INV_ID"
    run_test "action list (json)" $BINARY action list "$INV_ID" -j
fi

# Artifact commands
echo "[artifact]"
if [ -n "$INV_ID" ]; then
    run_test "artifact get" $BINARY artifact get "$INV_ID"
fi

# JQ filtering
echo "[jq]"
run_test "jq filter" $BINARY invocation list --commit "$COMMIT_SHA" --jq '.invocation | length'

# File export
echo "[file export]"
TMPFILE=$(mktemp)
if $BINARY invocation list --commit "$COMMIT_SHA" -j -o "$TMPFILE" 2>/dev/null; then
    if [ -s "$TMPFILE" ]; then
        pass "file export (-o)"
    else
        fail "file export produced empty file"
    fi
else
    fail "file export"
fi
rm -f "$TMPFILE"

# Plaintext
echo "[plaintext]"
run_test "plaintext output" $BINARY invocation list --commit "$COMMIT_SHA" --plaintext

# Aliases
echo "[aliases]"
run_test "inv alias" $BINARY inv ls --commit "$COMMIT_SHA"
if [ -n "$INV_ID" ]; then
    run_test "invocation show alias" $BINARY invocation show "$INV_ID"
    run_test "target ls alias" $BINARY target ls "$INV_ID"
    run_test "action ls alias" $BINARY action ls "$INV_ID"
fi

# --fields (field selection)
echo "[fields]"
FIELDS_OUT=$($BINARY invocation list --commit "$COMMIT_SHA" --fields "invocation" 2>&1)
if echo "$FIELDS_OUT" | grep -q '"invocation"'; then
    pass "--fields invocation"
else
    fail "--fields invocation: $FIELDS_OUT"
fi
# --fields should exclude keys not listed
FIELDS_OUT2=$($BINARY invocation get "$INV_ID" --fields "invocation" 2>&1)
if echo "$FIELDS_OUT2" | grep -q '"next_page_token"\|"nextPageToken"'; then
    fail "--fields leaks unlisted keys"
else
    pass "--fields excludes unlisted keys"
fi

# --limit
echo "[limit]"
LIMIT_COUNT=$($BINARY invocation list --commit "$COMMIT_SHA" --limit 1 --jq '.invocation | length' 2>&1)
if [ "$LIMIT_COUNT" = "1" ]; then
    pass "--limit 1 returns exactly 1"
else
    fail "--limit 1 returned $LIMIT_COUNT instead of 1"
fi
LIMIT_COUNT2=$($BINARY invocation list --commit "$COMMIT_SHA" --limit 2 --jq '.invocation | length' 2>&1)
if [ "$LIMIT_COUNT2" = "2" ]; then
    pass "--limit 2 returns exactly 2"
else
    fail "--limit 2 returned $LIMIT_COUNT2 instead of 2"
fi

# --log grep
echo "[log grep]"
if [ -n "$INV_ID" ]; then
    run_test "log get --grep" $BINARY log get "$INV_ID" --grep "test"
fi

# Template
echo "[template]"
TMPL_OUT=$($BINARY invocation list --commit "$COMMIT_SHA" --template '{{range .invocation}}{{.command}} {{end}}' 2>&1)
if echo "$TMPL_OUT" | grep -q "test"; then
    pass "--template rendering"
else
    fail "--template: $TMPL_OUT"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
