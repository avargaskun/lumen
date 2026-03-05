#!/usr/bin/env python3
"""
eval_llm_judge.py — LLM-as-judge evaluator for codebase Q&A tasks.

Uses a separate model (GPT-4o or Claude Haiku) to evaluate whether
the agent's response contains required facts from the rubric.

Usage:
    python eval_llm_judge.py <conversation.json> <task.json> [--model claude-haiku-4-5-20251001]

Outputs JSON with per-fact scores and overall pass/fail.
"""

import argparse
import json
import os
import sys

try:
    import anthropic
except ImportError:
    print(json.dumps({
        "error": "anthropic package not installed. Run: pip install anthropic",
        "pass": False
    }))
    sys.exit(1)


def extract_agent_response(conv_path: str) -> str:
    """Extract the final text response from a Claude conversation JSON."""
    with open(conv_path) as f:
        conv = json.load(f)

    # Walk the conversation looking for assistant text blocks
    texts = []
    if isinstance(conv, dict):
        # Try to find result text
        result = conv.get("result", "")
        if result:
            return result

    # Fallback: extract all text content recursively
    def walk(obj):
        if isinstance(obj, dict):
            if obj.get("type") == "text" and "text" in obj:
                texts.append(obj["text"])
            for v in obj.values():
                walk(v)
        elif isinstance(obj, list):
            for item in obj:
                walk(item)

    walk(conv)
    return "\n".join(texts[-3:]) if texts else ""


def judge_response(response: str, rubric: list, model: str) -> dict:
    """Use an LLM to judge whether the response contains required facts."""
    client = anthropic.Anthropic()

    rubric_text = "\n".join(
        f"  {i+1}. [weight={item.get('weight', 1.0)}] {item['fact']}"
        for i, item in enumerate(rubric)
    )

    prompt = f"""You are an expert evaluator. Given an AI assistant's response about a codebase,
determine whether each required fact from the rubric is present in the response.

RUBRIC (required facts):
{rubric_text}

RESPONSE TO EVALUATE:
{response}

For each fact in the rubric, output a JSON object with:
- "fact_index": the 1-based index
- "present": true/false
- "evidence": a brief quote from the response that supports your judgment (or "not found")

Output ONLY a JSON array of these objects, no other text."""

    message = client.messages.create(
        model=model,
        max_tokens=2000,
        messages=[{"role": "user", "content": prompt}],
    )

    # Parse the judge's response
    judge_text = message.content[0].text.strip()
    # Try to extract JSON from the response
    if judge_text.startswith("["):
        judgments = json.loads(judge_text)
    else:
        # Try to find JSON array in the response
        start = judge_text.find("[")
        end = judge_text.rfind("]") + 1
        if start >= 0 and end > start:
            judgments = json.loads(judge_text[start:end])
        else:
            return {"error": "Could not parse judge response", "raw": judge_text, "pass": False}

    # Calculate weighted score
    total_weight = sum(item.get("weight", 1.0) for item in rubric)
    earned_weight = 0.0

    results = []
    for j in judgments:
        idx = j.get("fact_index", 0) - 1
        if 0 <= idx < len(rubric):
            weight = rubric[idx].get("weight", 1.0)
            present = j.get("present", False)
            if present:
                earned_weight += weight
            results.append({
                "fact": rubric[idx]["fact"],
                "weight": weight,
                "present": present,
                "evidence": j.get("evidence", ""),
            })

    score = earned_weight / total_weight if total_weight > 0 else 0.0

    return {
        "facts": results,
        "score": round(score, 3),
        "earned_weight": earned_weight,
        "total_weight": total_weight,
        "pass": score >= 0.6,  # 60% threshold
    }


def main():
    parser = argparse.ArgumentParser(description="LLM-as-judge evaluator")
    parser.add_argument("conversation", help="Path to conversation.json")
    parser.add_argument("task", help="Path to task.json")
    parser.add_argument("--model", default="claude-haiku-4-5-20251001",
                        help="Judge model (default: claude-haiku-4-5-20251001)")
    args = parser.parse_args()

    with open(args.task) as f:
        task = json.load(f)

    rubric = task.get("evaluation", {}).get("rubric", [])
    if not rubric:
        print(json.dumps({"error": "No rubric defined in task", "pass": False}))
        sys.exit(1)

    response = extract_agent_response(args.conversation)
    if not response:
        print(json.dumps({"error": "No response found in conversation", "pass": False}))
        sys.exit(1)

    result = judge_response(response, rubric, args.model)
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
