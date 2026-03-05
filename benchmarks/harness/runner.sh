#!/usr/bin/env bash
# runner.sh — Main benchmark orchestrator.
#
# Usage:
#   ./runner.sh                              # Run all tasks
#   ./runner.sh --task bug_fix/django-*.json # Run specific task(s)
#   ./runner.sh --arm baseline               # Run only one arm
#   ./runner.sh --trials 3                   # Override trial count
#
# Results are written to benchmarks/results/<timestamp>/

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BENCH_DIR="$(dirname "$SCRIPT_DIR")"

# shellcheck source=config.env
source "$SCRIPT_DIR/config.env"

# Parse arguments
TASK_GLOB=""
ARM_FILTER=""
TRIAL_OVERRIDE=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --task)   TASK_GLOB="$2"; shift 2 ;;
        --arm)    ARM_FILTER="$2"; shift 2 ;;
        --trials) TRIAL_OVERRIDE="$2"; shift 2 ;;
        --help)
            echo "Usage: runner.sh [--task <glob>] [--arm baseline|lumen] [--trials N]"
            exit 0
            ;;
        *)
            echo "Unknown argument: $1" >&2
            exit 1
            ;;
    esac
done

TRIALS="${TRIAL_OVERRIDE:-$BENCH_TRIALS}"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
RESULTS_DIR="$BENCH_DIR/results/$TIMESTAMP"
mkdir -p "$RESULTS_DIR"

echo "============================================="
echo " Lumen Benchmark Suite"
echo " Trials per arm: $TRIALS"
echo " Results: $RESULTS_DIR"
echo "============================================="

# Discover tasks
if [[ -n "$TASK_GLOB" ]]; then
    TASK_FILES=("$BENCH_DIR/tasks/$TASK_GLOB")
else
    TASK_FILES=()
    while IFS= read -r -d '' f; do
        TASK_FILES+=("$f")
    done < <(find "$BENCH_DIR/tasks" -name "*.json" -not -name "task_schema.json" -print0 | sort -z)
fi

echo "Found ${#TASK_FILES[@]} task(s)"
echo ""

# Determine arms to run
if [[ -n "$ARM_FILTER" ]]; then
    ARMS=("$ARM_FILTER")
else
    ARMS=("baseline" "lumen")
fi

# Track all metrics files for final report
METRICS_FILES=()

for TASK_FILE in "${TASK_FILES[@]}"; do
    TASK_ID=$(jq -r '.id' "$TASK_FILE")
    REPO_URL=$(jq -r '.repo' "$TASK_FILE")
    COMMIT=$(jq -r '.commit' "$TASK_FILE")
    LANGUAGE=$(jq -r '.language' "$TASK_FILE")
    CATEGORY=$(jq -r '.category' "$TASK_FILE")

    # Read setup commands as bash array
    SETUP_CMDS=()
    while IFS= read -r cmd; do
        SETUP_CMDS+=("$cmd")
    done < <(jq -r '.setup_commands[]? // empty' "$TASK_FILE")

    REPO_NAME=$(echo "$REPO_URL" | sed 's|.*/||;s|\.git$||')
    REPO_DIR="$BENCH_REPOS_DIR/$REPO_NAME"

    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo " Task: $TASK_ID ($CATEGORY)"
    echo " Repo: $REPO_URL @ $COMMIT"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # Clone/setup repo
    "$SCRIPT_DIR/setup_repo.sh" "$REPO_URL" "$COMMIT" "$REPO_DIR" "${SETUP_CMDS[@]}"

    for ARM in "${ARMS[@]}"; do
        echo ""
        echo "  ── Arm: $ARM ──"

        # Pre-index for lumen arm
        if [[ "$ARM" == "lumen" ]]; then
            echo "  [index] Pre-indexing repo..."
            LUMEN_BIN_ABS="$(cd "$(dirname "$LUMEN_BIN")" && pwd)/$(basename "$LUMEN_BIN")"
            "$LUMEN_BIN_ABS" index --path "$REPO_DIR" 2>/dev/null || {
                echo "  [index] WARNING: Lumen indexing failed, continuing anyway"
            }
        fi

        for TRIAL in $(seq 1 "$TRIALS"); do
            TRIAL_DIR="$RESULTS_DIR/$TASK_ID/$ARM/trial-$TRIAL"

            echo "  Trial $TRIAL/$TRIALS..."
            "$SCRIPT_DIR/run_trial.sh" "$TASK_FILE" "$ARM" "$TRIAL" "$REPO_DIR" "$TRIAL_DIR"

            METRICS_FILES+=("$TRIAL_DIR/metrics.json")
        done
    done

    # Run evaluation
    echo ""
    echo "  [eval] Evaluating results..."
    EVAL_TYPE=$(jq -r '.evaluation.type' "$TASK_FILE")
    EVAL_DIR="$RESULTS_DIR/$TASK_ID"

    case "$EVAL_TYPE" in
        test_pass)
            TEST_CMD=$(jq -r '.evaluation.test_cmd' "$TASK_FILE")
            for ARM in "${ARMS[@]}"; do
                for TRIAL in $(seq 1 "$TRIALS"); do
                    TRIAL_DIR="$EVAL_DIR/$ARM/trial-$TRIAL"
                    "$SCRIPT_DIR/../eval/eval_test_pass.sh" "$REPO_DIR" "$TEST_CMD" > "$TRIAL_DIR/eval_result.json" 2>&1 || true
                    # Reset repo for next evaluation
                    git -C "$REPO_DIR" checkout . 2>/dev/null || true
                    git -C "$REPO_DIR" clean -fd 2>/dev/null || true
                done
            done
            ;;
        compile_and_test)
            TEST_CMD=$(jq -r '.evaluation.test_cmd' "$TASK_FILE")
            for ARM in "${ARMS[@]}"; do
                for TRIAL in $(seq 1 "$TRIALS"); do
                    TRIAL_DIR="$EVAL_DIR/$ARM/trial-$TRIAL"
                    "$SCRIPT_DIR/../eval/eval_test_pass.sh" "$REPO_DIR" "$TEST_CMD" > "$TRIAL_DIR/eval_result.json" 2>&1 || true
                    # Check grep_absent patterns
                    ABSENT_PATTERNS=$(jq -r '.evaluation.grep_absent[]? // empty' "$TASK_FILE")
                    if [[ -n "$ABSENT_PATTERNS" ]]; then
                        GREP_PASS=true
                        while IFS= read -r pattern; do
                            if grep -rq "$pattern" "$REPO_DIR" --include="*.go" --include="*.py" --include="*.rs" 2>/dev/null; then
                                GREP_PASS=false
                                break
                            fi
                        done <<< "$ABSENT_PATTERNS"
                        jq --argjson grep_pass "$GREP_PASS" '. + {grep_absent_pass: $grep_pass}' "$TRIAL_DIR/eval_result.json" > "$TRIAL_DIR/eval_result_tmp.json"
                        mv "$TRIAL_DIR/eval_result_tmp.json" "$TRIAL_DIR/eval_result.json"
                    fi
                    git -C "$REPO_DIR" checkout . 2>/dev/null || true
                    git -C "$REPO_DIR" clean -fd 2>/dev/null || true
                done
            done
            ;;
        locations|llm_judge)
            echo "  [eval] $EVAL_TYPE evaluation requires manual/script evaluation (see eval/)"
            ;;
    esac

    echo ""
done

# Aggregate all metrics
echo "============================================="
echo " Aggregating results..."
echo "============================================="

# Concatenate all metrics into a single JSONL file
COMBINED="$RESULTS_DIR/all_metrics.jsonl"
for mf in "${METRICS_FILES[@]}"; do
    if [[ -f "$mf" ]]; then
        cat "$mf" >> "$COMBINED"
        echo "" >> "$COMBINED"
    fi
done

echo "Results written to: $RESULTS_DIR"
echo "Metrics: $COMBINED"
echo ""
echo "Generate report with:"
echo "  python $BENCH_DIR/eval/report.py --results-dir $RESULTS_DIR"
