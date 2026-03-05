#!/usr/bin/env bash
# run_trial.sh — Execute a single benchmark trial.
#
# Usage: run_trial.sh <task_json> <arm> <trial_num> <repo_dir> <output_dir>
#
# arm: "baseline" or "lumen"
# Outputs: <output_dir>/conversation.json, <output_dir>/metrics.json

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=config.env
source "$SCRIPT_DIR/config.env"

TASK_JSON="$1"
ARM="$2"
TRIAL_NUM="$3"
REPO_DIR="$4"
OUTPUT_DIR="$5"

mkdir -p "$OUTPUT_DIR"

# Extract prompt from task JSON
PROMPT=$(jq -r '.prompt' "$TASK_JSON")
TASK_ID=$(jq -r '.id' "$TASK_JSON")

echo "[trial] Task=$TASK_ID Arm=$ARM Trial=$TRIAL_NUM"

# Clean repo state before each trial
git -C "$REPO_DIR" checkout . 2>/dev/null || true
git -C "$REPO_DIR" clean -fd 2>/dev/null || true

# Build claude command
CLAUDE_ARGS=(
    -p "$PROMPT"
    --output-format json
    --max-turns 50
    --cwd "$REPO_DIR"
)

if [[ "$ARM" == "lumen" ]]; then
    # Generate MCP config with correct lumen binary path
    LUMEN_BIN_ABS="$(cd "$(dirname "$LUMEN_BIN")" && pwd)/$(basename "$LUMEN_BIN")"
    MCP_CONFIG="$OUTPUT_DIR/lumen-mcp.json"
    sed "s|__LUMEN_BIN__|${LUMEN_BIN_ABS}|g" "$BENCH_MCP_CONFIG" > "$MCP_CONFIG"
    CLAUDE_ARGS+=(--mcp-config "$MCP_CONFIG")
fi

# Record start time
START_TIME=$(date +%s%N)

# Run claude and capture output
echo "[trial] Running claude ${CLAUDE_ARGS[*]:0:4}..."
CONV_FILE="$OUTPUT_DIR/conversation.json"

timeout "$BENCH_TIMEOUT" claude "${CLAUDE_ARGS[@]}" > "$CONV_FILE" 2>"$OUTPUT_DIR/stderr.log" || {
    EXIT_CODE=$?
    echo "[trial] Claude exited with code $EXIT_CODE"
    echo "{\"error\": \"claude exited with code $EXIT_CODE\"}" > "$CONV_FILE"
}

END_TIME=$(date +%s%N)
ELAPSED_MS=$(( (END_TIME - START_TIME) / 1000000 ))

# Extract metrics
"$SCRIPT_DIR/collect_metrics.sh" "$CONV_FILE" "$TASK_ID" "$ARM" "$TRIAL_NUM" "$ELAPSED_MS" > "$OUTPUT_DIR/metrics.json"

echo "[trial] Done. Metrics: $OUTPUT_DIR/metrics.json"
