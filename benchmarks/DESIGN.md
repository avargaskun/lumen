# Lumen Benchmark Suite — Design Document

## Objective

Measure whether Lumen's semantic search MCP tool meaningfully improves Claude
Code's performance on real software engineering tasks, compared to Claude Code's
built-in tools (Grep, Glob, Read).

## What We Measure

| Metric              | Unit            | Why it matters                                    |
| ------------------- | --------------- | ------------------------------------------------- |
| **Total tokens**    | input + output  | Direct cost proxy — fewer tokens = cheaper         |
| **Tool calls**      | count by type   | Measures exploration overhead                     |
| **Exploration ratio** | grep+glob+read / total calls | How much "wandering" vs productive work |
| **Task success**    | binary pass/fail| Does the task complete correctly?                  |
| **Wall-clock time** | seconds         | End-to-end latency                                |
| **First-correct-file** | tool call #  | How quickly the agent finds the right code         |

## Methodology: A/B Controlled Comparison

Every task runs **twice** under identical conditions:

- **Control** (`baseline`): Claude Code with default tools only (no Lumen)
- **Treatment** (`lumen`): Claude Code with Lumen plugin active + pre-built index

The harness:
1. Clones the target repo at a pinned commit
2. Pre-indexes with Lumen (treatment arm only — index build time excluded)
3. Runs the identical prompt through `claude -p` with `--output-format json`
4. Captures the full conversation JSON (tool calls, tokens, timing)
5. Evaluates the result against the expected outcome

### Statistical Rigor

- Each task runs **N=5** times per arm (LLM output is non-deterministic)
- Report **median** and **p25/p75** for all metrics
- Use paired Wilcoxon signed-rank test for significance (non-parametric,
  handles small N)
- Track **temperature=1** (default) for realistic variance

## Task Categories

Tasks are modeled after SWE-bench: real repos, real problems, verifiable
outcomes. Five categories test different aspects of code search:

### 1. Bug Fix (SWE-bench style)

Real GitHub issues with known patches. The agent must locate the bug and produce
a correct fix. Tests whether semantic search helps the agent find the relevant
code faster.

**Evaluation**: `git diff` against expected patch OR existing test suite passes.

### 2. Cross-File Navigation

"Find all implementations of interface X" or "trace the call chain from A to B".
These tasks require understanding code structure across multiple files — exactly
where semantic search should shine vs grep.

**Evaluation**: The agent must list specific file:line locations. Checked against
a known answer set (precision + recall).

### 3. Feature Addition

Add a small feature that requires understanding existing patterns (e.g., "add a
new CLI subcommand following the pattern of existing ones"). Tests whether
semantic search helps the agent discover conventions faster.

**Evaluation**: Added code compiles + passes new test case provided in the task.

### 4. Codebase Q&A

Questions about architecture ("how does the auth middleware chain work?") where
the answer requires reading 3-10 files. Measures whether semantic search reduces
token burn from reading irrelevant files.

**Evaluation**: LLM-as-judge with a rubric (required facts that must appear in
the answer). Uses a separate model to avoid self-evaluation bias.

### 5. Refactoring

Rename a symbol, extract a function, or move code — tasks that require finding
all usage sites. Tests completeness of code discovery.

**Evaluation**: Code compiles + all references updated (verified by
grep for old name = 0 matches + tests pass).

## Task Definition Format

Each task is a JSON file in `benchmarks/tasks/`:

```json
{
  "id": "django-fix-queryset-filter-01",
  "category": "bug_fix",
  "repo": "https://github.com/django/django",
  "commit": "a1b2c3d4e5f6...",
  "language": "python",
  "prompt": "Fix the bug described in this issue: ...",
  "evaluation": {
    "type": "test_pass",
    "test_cmd": "python -m pytest tests/queryset/test_filter.py -x",
    "expected_exit_code": 0
  },
  "expected_files_touched": ["django/db/models/query.py"],
  "metadata": {
    "difficulty": "medium",
    "num_relevant_files": 3,
    "repo_size_files": 4500,
    "source": "swe-bench-lite"
  }
}
```

## Target Repositories

Tasks should span different repo sizes and languages to test Lumen's value
across contexts:

| Repo               | Language | Size   | Why                                          |
| ------------------ | -------- | ------ | -------------------------------------------- |
| django/django      | Python   | ~4500  | SWE-bench gold standard, complex codebase    |
| golang/go          | Go       | ~9000  | Large Go stdlib, Lumen's native AST support  |
| pallets/flask      | Python   | ~300   | Small repo — does Lumen help or add noise?   |
| stretchr/testify   | Go       | ~100   | Tiny repo — baseline where Lumen shouldn't matter |
| kubernetes/kubectl | Go       | ~1500  | Medium Go, deep package hierarchy            |
| astral-sh/ruff     | Rust     | ~3000  | Non-Go/Python — tests language-agnostic value|

## Harness Architecture

```
benchmarks/
├── DESIGN.md              # This document
├── tasks/                 # Task definitions (JSON)
│   ├── bug_fix/
│   ├── cross_file_nav/
│   ├── feature_add/
│   ├── codebase_qa/
│   └── refactor/
├── harness/
│   ├── runner.sh          # Main orchestrator
│   ├── setup_repo.sh      # Clone + checkout pinned commit
│   ├── run_trial.sh       # Single trial (one arm, one task)
│   ├── collect_metrics.sh # Extract metrics from conversation JSON
│   └── config.env         # Paths, model, repeat count
├── eval/
│   ├── eval_test_pass.sh  # Evaluator: run test suite
│   ├── eval_diff.sh       # Evaluator: compare git diff
│   ├── eval_locations.sh  # Evaluator: check file:line answers
│   ├── eval_llm_judge.py  # Evaluator: LLM-as-judge for Q&A
│   └── report.py          # Aggregate results → markdown table
└── fixtures/
    └── ...                # Pre-built indexes, expected outputs
```

### Runner Flow

```
for each task in tasks/:
  setup_repo(task.repo, task.commit)

  for arm in [baseline, lumen]:
    if arm == lumen:
      lumen index --path $REPO_DIR  # pre-index

    for trial in 1..N:
      run_trial(task, arm, trial)
        → claude -p "$PROMPT" --output-format json \
            [--mcp-config lumen.json]  # only for treatment arm
        → save conversation.json
        → collect_metrics(conversation.json) → metrics.jsonl

  evaluate(task, results/)

report(all_results/) → results.md
```

### Metric Extraction from Conversation JSON

Claude's `--output-format json` returns structured data including:
- `input_tokens`, `output_tokens` per turn
- Tool calls with name, arguments, and results
- Total wall-clock time

The collector parses this into:

```json
{
  "task_id": "...",
  "arm": "baseline|lumen",
  "trial": 1,
  "total_input_tokens": 12345,
  "total_output_tokens": 6789,
  "tool_calls": {
    "Read": 12,
    "Grep": 8,
    "Glob": 3,
    "Edit": 2,
    "Write": 1,
    "Bash": 0,
    "semantic_search": 0
  },
  "exploration_calls": 23,
  "productive_calls": 3,
  "exploration_ratio": 0.88,
  "wall_clock_seconds": 45.2,
  "first_correct_file_call": 5,
  "success": true
}
```

## Key Design Decisions

### Why not just use SWE-bench directly?

SWE-bench's evaluation harness is Python-heavy and tightly coupled to their
infrastructure. We need:
1. A/B comparison (not in SWE-bench)
2. Tool-call level metrics (not in SWE-bench)
3. Support for Go repos (SWE-bench is Python-only)
4. MCP plugin integration

But we **borrow SWE-bench's best ideas**: pinned commits, real issues, test-based
evaluation, and we include some SWE-bench-lite tasks directly.

### Why pre-index for the treatment arm?

Indexing time is a one-time cost, not a per-query cost. Including it would
penalize Lumen on wall-clock time for a cost users only pay once. The benchmark
measures search-time value, not setup cost.

### Why include tiny repos?

To test the null hypothesis: Lumen should NOT help on repos small enough that
grep finds everything instantly. If it does, we're measuring noise. If it
doesn't, that's a valid "no effect" baseline.

### Why LLM-as-judge for Q&A?

String matching can't evaluate natural language answers. A rubric-based judge
with required facts is the standard approach (used by LMSYS Arena, MT-Bench).
We use a different model family to avoid self-evaluation bias.

## Success Criteria

Lumen demonstrates value if, across the task suite:

1. **Token reduction ≥ 15%** median on medium+ repos (statistically significant)
2. **First-correct-file improves by ≥ 2 tool calls** median
3. **No regression** in task success rate (>= baseline pass rate)
4. **Minimal effect on tiny repos** (confirms we're measuring real signal)

## Running the Benchmarks

```bash
# Full suite (slow — runs N=5 trials × 2 arms × all tasks)
cd benchmarks && ./harness/runner.sh

# Single task, both arms
./harness/runner.sh --task bug_fix/django-queryset-01.json

# Generate report from existing results
python eval/report.py --results-dir results/
```
