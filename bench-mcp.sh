#!/usr/bin/env bash
# bench-mcp.sh — benchmark baseline vs agent-index MCP across questions and models
set -eufo pipefail

REPO="$(cd "$(dirname "$0")" && pwd)"
FIXTURES="$REPO/testdata/fixtures/go"
BINARY="$REPO/agent-index"

# ── Questions ─────────────────────────────────────────────────────────────────
QUESTIONS=(
  # EASY: single-file, direct lookup
  "What label matcher types are available and how is a Matcher created? Show the type definitions and constructor."
  # MEDIUM: spans 2–3 files, requires understanding algorithm
  "How does histogram bucket counting work? Show me the relevant function signatures."
  # HARD: large codebase, multiple files, complex interactions
  "How does TSDB compaction work end-to-end? Explain the Compactor interface, LeveledCompactor, and how the DB triggers compaction. Show relevant types, interfaces, and key method signatures."
)
Q_SLUGS=(
  "label-matcher"
  "histogram"
  "tsdb-compaction"
)
Q_DIFFICULTY=(
  "easy"
  "medium"
  "hard"
)

# ── Models ────────────────────────────────────────────────────────────────────
MODELS=("haiku" "opus")
FILTER_MODELS=()
FILTER_QUESTIONS=()

# ── CLI flags ─────────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --model)    FILTER_MODELS+=("$2");    shift 2 ;;
    --question) FILTER_QUESTIONS+=("$2"); shift 2 ;;
    *) echo "Unknown arg: $1"; exit 1 ;;
  esac
done

[[ ${#FILTER_MODELS[@]} -gt 0 ]] && MODELS=("${FILTER_MODELS[@]}")

# Build filtered question index
Q_INDICES=()
for i in "${!Q_SLUGS[@]}"; do
  if [[ ${#FILTER_QUESTIONS[@]} -eq 0 ]]; then
    Q_INDICES+=("$i")
  else
    for fq in "${FILTER_QUESTIONS[@]}"; do
      if [[ "${Q_SLUGS[$i]}" == "$fq" ]]; then
        Q_INDICES+=("$i")
        break
      fi
    done
  fi
done

# ── Build ──────────────────────────────────────────────────────────────────────
echo "Building agent-index..."
CGO_ENABLED=1 go build -o agent-index .

# ── Index ─────────────────────────────────────────────────────────────────────
echo "Indexing fixtures..."
./agent-index index "$FIXTURES" 2>&1 | tail -1

# ── MCP configs ───────────────────────────────────────────────────────────────
MCP_ENABLED=$(mktemp /tmp/bench-mcp-enabled-XXXXXX).json
MCP_EMPTY=$(mktemp /tmp/bench-mcp-empty-XXXXXX).json
trap 'rm -f "$MCP_ENABLED" "$MCP_EMPTY"' EXIT

cat > "$MCP_ENABLED" <<EOF
{"mcpServers":{"agent-index":{"command":"$BINARY","args":["stdio"]}}}
EOF
echo '{"mcpServers":{}}' > "$MCP_EMPTY"

# ── Results dir ───────────────────────────────────────────────────────────────
RESULTS_DIR="$REPO/bench-results/$(date +%Y%m%d-%H%M%S)"
mkdir -p "$RESULTS_DIR"

# ── Run one scenario ──────────────────────────────────────────────────────────
run() {
  local mcp_cfg="$1" model="$2" q_idx="$3" scenario="$4" allowed_tools="$5"
  local slug="${Q_SLUGS[$q_idx]}-${model}-${scenario}"
  local prompt="The Go project is at $FIXTURES. Answer this question about the code: ${QUESTIONS[$q_idx]}"
  local raw="$RESULTS_DIR/$slug-raw.jsonl"
  local answer_file="$RESULTS_DIR/$slug-answer.txt"

  printf "  %-28s %-12s %-10s ... " "${Q_SLUGS[$q_idx]}" "$model" "$scenario"

  local allowed_tools_arg=()
  [[ -n "$allowed_tools" ]] && allowed_tools_arg=(--allowedTools "$allowed_tools")

  claude \
    --output-format stream-json \
    --verbose \
    --model "$model" \
    --strict-mcp-config \
    --mcp-config "$mcp_cfg" \
    ${allowed_tools_arg[@]:+"${allowed_tools_arg[@]}"} \
    -p "$prompt" \
  > "$raw" 2>&1 || true

  local result_line
  result_line=$(grep -m1 '"type":"result"' "$raw" || true)
  if [[ -n "$result_line" ]]; then
    local cost duration_ms input_tokens cache_read cache_created output_tokens
    cost=$(echo "$result_line"          | jq -r '.total_cost_usd')
    duration_ms=$(echo "$result_line"   | jq -r '.duration_ms')
    input_tokens=$(echo "$result_line"  | jq -r '.usage.input_tokens // 0')
    cache_read=$(echo "$result_line"    | jq -r '.usage.cache_read_input_tokens // 0')
    cache_created=$(echo "$result_line" | jq -r '.usage.cache_creation_input_tokens // 0')
    output_tokens=$(echo "$result_line" | jq -r '.usage.output_tokens // 0')

    echo "$result_line" | jq -r '.result' > "$answer_file"
    echo "{\"cost_usd\":$cost,\"duration_ms\":$duration_ms,\"input_tokens\":$input_tokens,\"cache_read\":$cache_read,\"cache_created\":$cache_created,\"output_tokens\":$output_tokens}" \
      > "$RESULTS_DIR/$slug-metrics.json"

    local cost_fmt dur_s
    cost_fmt=$(printf "%.4f" "$cost")
    dur_s=$(echo "scale=1; $duration_ms/1000" | bc)
    printf "done  [%6ss  \$%s  in=%s+%scr  out=%s]\n" \
      "$dur_s" "$cost_fmt" "$input_tokens" "$cache_read" "$output_tokens"
  else
    echo "FAILED (no result event)"
  fi
}

# ── Run LLM judge for one question ────────────────────────────────────────────
run_judge() {
  local q_idx="$1"
  local slug="${Q_SLUGS[$q_idx]}"
  local question="${QUESTIONS[$q_idx]}"
  local judge_file="$RESULTS_DIR/$slug-judge.md"
  local judge_brief_file="$RESULTS_DIR/$slug-judge-brief.md"

  # Collect answers and metrics
  local all_answers=""
  local metrics_table="| Run | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) |"$'\n'
  metrics_table+="|----|----------|-----------|------------|------------|------------|"$'\n'
  local have_answers=false

  for model in "${MODELS[@]}"; do
    for scenario in baseline mcp-only mcp-full; do
      local af="$RESULTS_DIR/${slug}-${model}-${scenario}-answer.txt"
      local mf="$RESULTS_DIR/${slug}-${model}-${scenario}-metrics.json"
      if [[ -f "$af" ]]; then
        have_answers=true
        all_answers+="
Answer [${model} / ${scenario}]:
$(cat "$af")
"
      fi
      if [[ -f "$mf" ]]; then
        local in_tok cr_tok out_tok cost_usd dur_ms cost_fmt dur_s
        in_tok=$(jq -r '.input_tokens'   "$mf")
        cr_tok=$(jq -r '.cache_read'     "$mf")
        out_tok=$(jq -r '.output_tokens' "$mf")
        cost_usd=$(jq -r '.cost_usd'     "$mf")
        dur_ms=$(jq -r '.duration_ms'    "$mf")
        cost_fmt=$(printf "%.5f" "$cost_usd")
        dur_s=$(echo "scale=1; $dur_ms/1000" | bc)
        metrics_table+="| ${model} / ${scenario} | ${dur_s}s | $in_tok | $cr_tok | $out_tok | \$$cost_fmt |"$'\n'
      fi
    done
  done

  if ! $have_answers; then
    return
  fi

  printf "  Judging %-28s ... " "$slug"

  # Brief verdict for summary (content quality + efficiency)
  claude -p --model claude-opus-4-6 \
    "You are a judge evaluating AI answers to a Go codebase question. Be concise.

Question: $question

$all_answers

Metrics:
$metrics_table

Evaluate in two sections:

## Content Quality
Rank the answers [model/scenario] from best to worst. One sentence per answer covering correctness, completeness, and use of specific file/line references.

## Efficiency
One or two sentences comparing runtime, token usage, and cost across scenarios. Note which scenario offers the best quality-to-cost tradeoff." \
    > "$judge_brief_file" 2>&1 || echo "_Judge unavailable_" > "$judge_brief_file"

  # Detailed analysis for detail report
  claude -p --model claude-opus-4-6 \
    "You are a judge evaluating AI answers to a question about a Go codebase.

Question: $question

$all_answers

Metrics:
$metrics_table

Evaluate in two sections:

## Content Quality
Rank the answers from best to worst. For each, write a paragraph covering: (1) correctness, (2) completeness, (3) precision of file/line references, (4) whether it used the right tools/approach to find information.

## Efficiency Analysis
Compare runtime, token usage, and cost across all scenarios. Identify which scenarios were most efficient, note any surprising differences, and recommend the best quality-to-cost tradeoff." \
    > "$judge_file" 2>&1 || echo "_Judge unavailable_" > "$judge_file"

  echo "done"
}

# ── Emit aggregate stats table across all questions ───────────────────────────
emit_overall_stats() {
  echo "## Overall: Aggregated by Scenario"
  echo ""
  echo "Totals across all ${#Q_INDICES[@]} questions × ${#MODELS[@]} models."
  echo ""
  echo "| Model | Scenario | Total Time | Total Input Tok | Total Output Tok | Total Cost (USD) |"
  echo "|-------|----------|------------|-----------------|------------------|------------------|"

  for model in "${MODELS[@]}"; do
    for scenario in baseline mcp-only mcp-full; do
      local total_dur_ms=0 total_in=0 total_out=0 total_cost_scaled=0 count=0
      for q_idx in "${Q_INDICES[@]}"; do
        local mf="$RESULTS_DIR/${Q_SLUGS[$q_idx]}-${model}-${scenario}-metrics.json"
        if [[ -f "$mf" ]]; then
          total_dur_ms=$(( total_dur_ms + $(jq -r '.duration_ms'    "$mf") ))
          total_in=$(( total_in         + $(jq -r '.input_tokens'   "$mf") ))
          total_out=$(( total_out       + $(jq -r '.output_tokens'  "$mf") ))
          # cost: multiply by 100000 to keep integer arithmetic, divide at end
          local cost_scaled
          cost_scaled=$(jq -r '(.cost_usd * 100000) | round' "$mf")
          total_cost_scaled=$(( total_cost_scaled + cost_scaled ))
          (( count++ )) || true
        fi
      done
      if [[ $count -gt 0 ]]; then
        local dur_s cost_fmt
        dur_s=$(echo "scale=1; $total_dur_ms/1000" | bc)
        cost_fmt=$(printf "%.4f" "$(echo "scale=5; $total_cost_scaled/100000" | bc)")
        echo "| **$model** | $scenario | ${dur_s}s | $total_in | $total_out | \$$cost_fmt |"
      else
        echo "| **$model** | $scenario | — | — | — | — |"
      fi
    done
  done
  echo ""
}

# ── Generate reports ───────────────────────────────────────────────────────────
generate_reports() {
  local summary_file="$RESULTS_DIR/summary-report.md"
  local detail_file="$RESULTS_DIR/detail-report.md"

  # ── Summary report ────────────────────────────────────────────────────────
  {
    echo "# Benchmark Summary"
    echo ""
    echo "Generated: $(date -u '+%Y-%m-%d %H:%M UTC')  |  Results: \`$(basename "$RESULTS_DIR")\`"
    echo ""
    echo "| Scenario | Description |"
    echo "|----------|-------------|"
    echo "| **baseline** | All default Claude tools, no MCP |"
    echo "| **mcp-only** | \`semantic_search\` MCP tool only |"
    echo "| **mcp-full** | All default tools + MCP |"
    echo ""
    emit_overall_stats
    echo "---"
    echo ""

    for q_idx in "${Q_INDICES[@]}"; do
      local slug="${Q_SLUGS[$q_idx]}"
      local difficulty="${Q_DIFFICULTY[$q_idx]}"
      local question="${QUESTIONS[$q_idx]}"

      echo "## $slug [$difficulty]"
      echo ""
      echo "> $question"
      echo ""
      echo "### Time & Tokens"
      echo ""
      echo "| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) |"
      echo "|-------|----------|----------|-----------|------------|------------|------------|"

      for model in "${MODELS[@]}"; do
        for scenario in baseline mcp-only mcp-full; do
          local metrics_file="$RESULTS_DIR/${slug}-${model}-${scenario}-metrics.json"
          if [[ -f "$metrics_file" ]]; then
            local in_tok cr_tok out_tok cost_usd dur_ms cost_fmt dur_s
            in_tok=$(jq -r '.input_tokens'   "$metrics_file")
            cr_tok=$(jq -r '.cache_read'     "$metrics_file")
            out_tok=$(jq -r '.output_tokens' "$metrics_file")
            cost_usd=$(jq -r '.cost_usd'     "$metrics_file")
            dur_ms=$(jq -r '.duration_ms'    "$metrics_file")
            cost_fmt=$(printf "%.4f" "$cost_usd")
            dur_s=$(echo "scale=1; $dur_ms/1000" | bc)
            echo "| **$model** | $scenario | ${dur_s}s | $in_tok | $cr_tok | $out_tok | \$$cost_fmt |"
          else
            echo "| **$model** | $scenario | — | — | — | — | — |"
          fi
        done
      done

      echo ""

      local judge_brief_file="$RESULTS_DIR/$slug-judge-brief.md"
      if [[ -f "$judge_brief_file" ]]; then
        echo "### Quality Ranking (Opus 4.6)"
        echo ""
        cat "$judge_brief_file"
        echo ""
      fi

      echo "---"
      echo ""
    done

    echo "_Full answers and detailed analysis: \`detail-report.md\`_"
  } > "$summary_file"

  # ── Detail report ─────────────────────────────────────────────────────────
  {
    echo "# Benchmark Detail Report"
    echo ""
    echo "Generated: $(date -u '+%Y-%m-%d %H:%M UTC')  |  Results: \`$(basename "$RESULTS_DIR")\`"
    echo ""

    for q_idx in "${Q_INDICES[@]}"; do
      local slug="${Q_SLUGS[$q_idx]}"
      local difficulty="${Q_DIFFICULTY[$q_idx]}"
      local question="${QUESTIONS[$q_idx]}"

      echo "---"
      echo ""
      echo "## $slug [$difficulty]"
      echo ""
      echo "**Question:** $question"
      echo ""

      echo "### Metrics"
      echo ""
      echo "| Model | Scenario | Duration | Input Tok | Cache Read | Cache Created | Output Tok | Cost (USD) |"
      echo "|-------|----------|----------|-----------|------------|---------------|------------|------------|"

      for model in "${MODELS[@]}"; do
        for scenario in baseline mcp-only mcp-full; do
          local metrics_file="$RESULTS_DIR/${slug}-${model}-${scenario}-metrics.json"
          if [[ -f "$metrics_file" ]]; then
            local in_tok cr_tok cc_tok out_tok cost_usd dur_ms cost_fmt dur_s
            in_tok=$(jq -r '.input_tokens'    "$metrics_file")
            cr_tok=$(jq -r '.cache_read'      "$metrics_file")
            cc_tok=$(jq -r '.cache_created'   "$metrics_file")
            out_tok=$(jq -r '.output_tokens'  "$metrics_file")
            cost_usd=$(jq -r '.cost_usd'      "$metrics_file")
            dur_ms=$(jq -r '.duration_ms'     "$metrics_file")
            cost_fmt=$(printf "%.5f" "$cost_usd")
            dur_s=$(echo "scale=1; $dur_ms/1000" | bc)
            echo "| **$model** | $scenario | ${dur_s}s | $in_tok | $cr_tok | $cc_tok | $out_tok | \$$cost_fmt |"
          else
            echo "| **$model** | $scenario | — | — | — | — | — | — |"
          fi
        done
      done

      echo ""

      for model in "${MODELS[@]}"; do
        for scenario in baseline mcp-only mcp-full; do
          local af="$RESULTS_DIR/${slug}-${model}-${scenario}-answer.txt"
          if [[ -f "$af" ]]; then
            echo "### Answer: \`$model\` / \`$scenario\`"
            echo ""
            cat "$af"
            echo ""
          fi
        done
      done

      local judge_file="$RESULTS_DIR/$slug-judge.md"
      if [[ -f "$judge_file" ]]; then
        echo "### Full Judge Analysis (Opus 4.6)"
        echo ""
        cat "$judge_file"
        echo ""
      fi
    done
  } > "$detail_file"

  echo ""
  echo "Reports written:"
  echo "  Summary : $summary_file"
  echo "  Detail  : $detail_file"
}

# ── Main loop ─────────────────────────────────────────────────────────────────
echo ""
echo "Running benchmarks (${#MODELS[@]} models × ${#Q_INDICES[@]} questions × 3 scenarios)..."
echo ""

for model in "${MODELS[@]}"; do
  echo "── Model: $model ──────────────────────────────────────────"
  for q_idx in "${Q_INDICES[@]}"; do
    run "$MCP_EMPTY"   "$model" "$q_idx" "baseline" ""
    run "$MCP_ENABLED" "$model" "$q_idx" "mcp-only"  "mcp__agent-index__semantic_search"
    run "$MCP_ENABLED" "$model" "$q_idx" "mcp-full"  ""
  done
  echo ""
done

echo "── Generating LLM judge reports ──────────────────────────────"
for q_idx in "${Q_INDICES[@]}"; do
  run_judge "$q_idx"
done

generate_reports
