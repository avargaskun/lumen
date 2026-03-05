#!/usr/bin/env bash
# collect_metrics.sh — Extract metrics from a Claude conversation JSON.
#
# Usage: collect_metrics.sh <conversation.json> <task_id> <arm> <trial> <elapsed_ms>
#
# Outputs JSON metrics to stdout.

set -euo pipefail

CONV_FILE="$1"
TASK_ID="$2"
ARM="$3"
TRIAL="$4"
ELAPSED_MS="$5"

# Use jq to parse the conversation JSON and extract metrics.
# Claude's --output-format json produces a structure with cost_usd, duration_ms,
# duration_api_ms, num_turns, and a result field.
#
# For tool call counting, we parse the full conversation messages.

jq --arg task_id "$TASK_ID" \
   --arg arm "$ARM" \
   --argjson trial "$TRIAL" \
   --argjson elapsed_ms "$ELAPSED_MS" \
'
# Extract top-level stats
def safe_num: if . == null then 0 else . end;

# Count tool calls by type from the conversation
def count_tool_calls:
  [.. | objects | select(.type == "tool_use") | .name // "unknown"]
  | group_by(.)
  | map({(.[0]): length})
  | add // {};

# Identify exploration vs productive tool calls
def classify_tools:
  . as $calls |
  {
    exploration: ([$calls[] | select(. == "Read" or . == "Grep" or . == "Glob" or . == "WebSearch" or . == "Agent" or . == "semantic_search")] | length),
    productive: ([$calls[] | select(. == "Edit" or . == "Write" or . == "Bash" or . == "NotebookEdit")] | length)
  };

# Extract all tool call names as flat array
def all_tool_names:
  [.. | objects | select(.type == "tool_use") | .name // "unknown"];

(all_tool_names) as $tool_names |
(count_tool_calls) as $tool_counts |
($tool_names | classify_tools) as $classification |

{
  task_id: $task_id,
  arm: $arm,
  trial: $trial,
  total_input_tokens: (.usage.input_tokens // .input_tokens // 0 | safe_num),
  total_output_tokens: (.usage.output_tokens // .output_tokens // 0 | safe_num),
  total_tokens: ((.usage.input_tokens // .input_tokens // 0 | safe_num) + (.usage.output_tokens // .output_tokens // 0 | safe_num)),
  cost_usd: (.cost_usd // 0),
  num_turns: (.num_turns // 0),
  tool_calls: $tool_counts,
  total_tool_calls: ($tool_names | length),
  exploration_calls: $classification.exploration,
  productive_calls: $classification.productive,
  exploration_ratio: (if ($tool_names | length) > 0 then ($classification.exploration / ($tool_names | length)) else 0 end),
  wall_clock_ms: $elapsed_ms,
  duration_api_ms: (.duration_api_ms // 0),
  is_error: (has("error") and .error != null)
}
' "$CONV_FILE"
