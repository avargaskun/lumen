#!/usr/bin/env python3
"""
report.py — Aggregate benchmark results and generate a comparison report.

Usage:
    python report.py --results-dir results/20250305-120000/

Reads all_metrics.jsonl and eval_result.json files to produce a markdown
report comparing baseline vs lumen performance.
"""

import argparse
import json
import os
import sys
from collections import defaultdict
from pathlib import Path


def load_metrics(results_dir: str) -> list[dict]:
    """Load all metrics from the JSONL file."""
    metrics_file = os.path.join(results_dir, "all_metrics.jsonl")
    metrics = []
    if os.path.exists(metrics_file):
        with open(metrics_file) as f:
            for line in f:
                line = line.strip()
                if line:
                    metrics.append(json.loads(line))
    return metrics


def load_eval_results(results_dir: str) -> dict:
    """Load evaluation results for each task/arm/trial."""
    evals = {}
    for root, _dirs, files in os.walk(results_dir):
        if "eval_result.json" in files:
            with open(os.path.join(root, "eval_result.json")) as f:
                data = json.load(f)
            # Parse path to get task_id/arm/trial
            parts = Path(root).relative_to(results_dir).parts
            if len(parts) >= 3:
                task_id, arm, trial = parts[0], parts[1], parts[2]
                key = (task_id, arm, trial)
                evals[key] = data
    return evals


def median(values: list) -> float:
    """Calculate median of a list of numbers."""
    if not values:
        return 0.0
    s = sorted(values)
    n = len(s)
    if n % 2 == 0:
        return (s[n // 2 - 1] + s[n // 2]) / 2
    return s[n // 2]


def percentile(values: list, p: float) -> float:
    """Calculate percentile of a list of numbers."""
    if not values:
        return 0.0
    s = sorted(values)
    k = (len(s) - 1) * p
    f = int(k)
    c = f + 1
    if c >= len(s):
        return s[f]
    return s[f] + (k - f) * (s[c] - s[f])


def format_number(n: float, decimals: int = 0) -> str:
    if decimals == 0:
        return f"{int(n):,}"
    return f"{n:,.{decimals}f}"


def generate_report(results_dir: str) -> str:
    """Generate a markdown report from benchmark results."""
    metrics = load_metrics(results_dir)
    evals = load_eval_results(results_dir)

    if not metrics:
        return "# Benchmark Report\n\nNo metrics found."

    # Group metrics by task and arm
    by_task_arm: dict[tuple[str, str], list[dict]] = defaultdict(list)
    for m in metrics:
        key = (m["task_id"], m["arm"])
        by_task_arm[key].append(m)

    # Get unique tasks and arms
    tasks = sorted(set(m["task_id"] for m in metrics))
    arms = sorted(set(m["arm"] for m in metrics))

    lines = [
        "# Lumen Benchmark Report",
        "",
        f"**Results directory**: `{results_dir}`",
        f"**Tasks**: {len(tasks)}",
        f"**Arms**: {', '.join(arms)}",
        f"**Trials per arm**: {max(len(v) for v in by_task_arm.values()) if by_task_arm else 0}",
        "",
        "---",
        "",
        "## Summary: Token Usage (median across trials)",
        "",
        "| Task | Baseline Tokens | Lumen Tokens | Delta | Delta % |",
        "| ---- | --------------: | -----------: | ----: | ------: |",
    ]

    total_baseline_tokens = []
    total_lumen_tokens = []

    for task in tasks:
        baseline = by_task_arm.get((task, "baseline"), [])
        lumen = by_task_arm.get((task, "lumen"), [])

        b_tokens = median([m["total_tokens"] for m in baseline]) if baseline else 0
        l_tokens = median([m["total_tokens"] for m in lumen]) if lumen else 0
        delta = l_tokens - b_tokens
        pct = (delta / b_tokens * 100) if b_tokens > 0 else 0

        if baseline:
            total_baseline_tokens.extend([m["total_tokens"] for m in baseline])
        if lumen:
            total_lumen_tokens.extend([m["total_tokens"] for m in lumen])

        sign = "+" if delta > 0 else ""
        lines.append(
            f"| {task} | {format_number(b_tokens)} | {format_number(l_tokens)} "
            f"| {sign}{format_number(delta)} | {sign}{pct:.1f}% |"
        )

    # Overall
    if total_baseline_tokens and total_lumen_tokens:
        ob = median(total_baseline_tokens)
        ol = median(total_lumen_tokens)
        od = ol - ob
        op = (od / ob * 100) if ob > 0 else 0
        sign = "+" if od > 0 else ""
        lines.append(
            f"| **OVERALL** | **{format_number(ob)}** | **{format_number(ol)}** "
            f"| **{sign}{format_number(od)}** | **{sign}{op:.1f}%** |"
        )

    lines.extend([
        "",
        "## Tool Call Comparison (median)",
        "",
        "| Task | Arm | Total Calls | Exploration | Productive | Exploration Ratio |",
        "| ---- | --- | ----------: | ----------: | ---------: | ----------------: |",
    ])

    for task in tasks:
        for arm in arms:
            trials = by_task_arm.get((task, arm), [])
            if not trials:
                continue
            total_calls = median([m["total_tool_calls"] for m in trials])
            expl = median([m["exploration_calls"] for m in trials])
            prod = median([m["productive_calls"] for m in trials])
            ratio = median([m["exploration_ratio"] for m in trials])
            lines.append(
                f"| {task} | {arm} | {format_number(total_calls)} "
                f"| {format_number(expl)} | {format_number(prod)} "
                f"| {ratio:.2f} |"
            )

    lines.extend([
        "",
        "## Wall-Clock Time (median, seconds)",
        "",
        "| Task | Baseline | Lumen | Delta |",
        "| ---- | -------: | ----: | ----: |",
    ])

    for task in tasks:
        baseline = by_task_arm.get((task, "baseline"), [])
        lumen = by_task_arm.get((task, "lumen"), [])

        b_time = median([m["wall_clock_ms"] / 1000 for m in baseline]) if baseline else 0
        l_time = median([m["wall_clock_ms"] / 1000 for m in lumen]) if lumen else 0
        delta = l_time - b_time
        sign = "+" if delta > 0 else ""
        lines.append(
            f"| {task} | {b_time:.1f}s | {l_time:.1f}s | {sign}{delta:.1f}s |"
        )

    lines.extend([
        "",
        "## Cost Comparison (median USD)",
        "",
        "| Task | Baseline | Lumen | Savings |",
        "| ---- | -------: | ----: | ------: |",
    ])

    for task in tasks:
        baseline = by_task_arm.get((task, "baseline"), [])
        lumen = by_task_arm.get((task, "lumen"), [])

        b_cost = median([m.get("cost_usd", 0) for m in baseline]) if baseline else 0
        l_cost = median([m.get("cost_usd", 0) for m in lumen]) if lumen else 0
        savings = b_cost - l_cost
        lines.append(
            f"| {task} | ${b_cost:.4f} | ${l_cost:.4f} | ${savings:.4f} |"
        )

    lines.extend([
        "",
        "## Detailed Tool Call Breakdown",
        "",
    ])

    # Collect all tool names across all metrics
    all_tools = set()
    for m in metrics:
        all_tools.update(m.get("tool_calls", {}).keys())
    tool_names = sorted(all_tools)

    if tool_names:
        header = "| Task | Arm | " + " | ".join(tool_names) + " |"
        sep = "| ---- | --- | " + " | ".join(["---:"] * len(tool_names)) + " |"
        lines.append(header)
        lines.append(sep)

        for task in tasks:
            for arm in arms:
                trials = by_task_arm.get((task, arm), [])
                if not trials:
                    continue
                counts = []
                for tool in tool_names:
                    vals = [m.get("tool_calls", {}).get(tool, 0) for m in trials]
                    counts.append(format_number(median(vals)))
                lines.append(f"| {task} | {arm} | " + " | ".join(counts) + " |")

    lines.extend([
        "",
        "## Statistical Notes",
        "",
        "- All values are **medians** across trials to handle LLM non-determinism",
        "- Negative delta = Lumen used fewer tokens/less time (improvement)",
        "- Positive delta = Lumen used more tokens/more time (regression)",
        "- For statistical significance with N=5 trials, use Wilcoxon signed-rank test",
        "- Exploration ratio = (Read + Grep + Glob + Search calls) / total tool calls",
        "",
        "## Success Criteria",
        "",
        "- [ ] Token reduction >= 15% median on medium+ repos",
        "- [ ] First-correct-file improves by >= 2 tool calls",
        "- [ ] No regression in task success rate",
        "- [ ] Minimal effect on tiny repos (null hypothesis control)",
    ])

    return "\n".join(lines)


def main():
    parser = argparse.ArgumentParser(description="Generate benchmark report")
    parser.add_argument("--results-dir", required=True, help="Path to results directory")
    parser.add_argument("--output", help="Output file (default: stdout)")
    args = parser.parse_args()

    report = generate_report(args.results_dir)

    if args.output:
        with open(args.output, "w") as f:
            f.write(report)
        print(f"Report written to {args.output}", file=sys.stderr)
    else:
        print(report)


if __name__ == "__main__":
    main()
