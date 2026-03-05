#!/usr/bin/env bash
# eval_locations.sh — Evaluate cross-file navigation tasks.
#
# Checks whether the agent's response mentions the expected file/symbol locations.
#
# Usage: eval_locations.sh <conversation.json> <task.json>
# Outputs JSON with precision, recall, and per-location match status.

set -euo pipefail

CONV_FILE="$1"
TASK_FILE="$2"

# Extract the agent's final text response
AGENT_RESPONSE=$(jq -r '
    [.. | objects | select(.type == "text") | .text // empty] | last // ""
' "$CONV_FILE")

# Extract expected locations from task
EXPECTED=$(jq -c '.evaluation.expected_locations // []' "$TASK_FILE")

# Check each expected location against the response
jq -n --arg response "$AGENT_RESPONSE" --argjson expected "$EXPECTED" '
def check_location(loc):
    ($response | ascii_downcase) as $resp |
    (loc.file | ascii_downcase) as $file |
    (loc.symbol // "" | ascii_downcase) as $symbol |
    {
        file: loc.file,
        symbol: (loc.symbol // ""),
        file_found: ($resp | contains($file)),
        symbol_found: (if $symbol == "" then true else ($resp | contains($symbol)) end),
        matched: (($resp | contains($file)) and (if $symbol == "" then true else ($resp | contains($symbol)) end))
    };

($expected | map(check_location(.))) as $results |
($results | map(select(.matched)) | length) as $matched |
($results | length) as $total |
{
    locations: $results,
    matched: $matched,
    total: $total,
    recall: (if $total > 0 then ($matched / $total) else 0 end),
    pass: (($matched / ([1, $total] | max)) >= 0.75)
}
'
