#!/usr/bin/env bash
# eval_test_pass.sh — Evaluate by running a test command.
#
# Usage: eval_test_pass.sh <repo_dir> <test_cmd>
# Outputs JSON: {"pass": true/false, "exit_code": N, "output": "..."}

set -euo pipefail

REPO_DIR="$1"
TEST_CMD="$2"

OUTPUT=$(cd "$REPO_DIR" && eval "$TEST_CMD" 2>&1) || true
EXIT_CODE=${PIPESTATUS[0]:-$?}

# Capture last 50 lines of output to keep JSON manageable
TRUNCATED=$(echo "$OUTPUT" | tail -50)

jq -n \
    --argjson exit_code "$EXIT_CODE" \
    --arg output "$TRUNCATED" \
    '{
        pass: ($exit_code == 0),
        exit_code: $exit_code,
        output: $output
    }'
