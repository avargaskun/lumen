# Benchmark Design: Measuring Lumen's Impact on Claude Performance

## Goals

1. **Prove value** — convincing, reproducible evidence that Lumen reduces
   tokens/cost/time while maintaining or improving answer quality
2. **Guide development** — identify which embedding models, chunker
   improvements, and retrieval strategies yield the most impact
3. **Use existing benchmarks** — adapt proven evaluation frameworks with clear
   expected outcomes rather than inventing from scratch

## What We Already Have

`bench-mcp.sh` runs knowledge questions across 3 scenarios (baseline / mcp-only
/ mcp-full), captures token/cost/time metrics via `claude --output-format
stream-json`, and uses an Opus 4.6 LLM judge for quality ranking. Results are
compelling (2-4x speedup, 48-90% cost savings), but limited to "explain how X
works" questions with no objective ground truth.

---

## Benchmark Categories

### Category 1: Code Understanding (extend existing)

**What it tests:** Can Claude answer architectural questions about a codebase?

**Current approach:** LLM judge ranks answers across scenarios. This works well
and should be extended, not replaced.

**Improvements:**

- Add difficulty tiers (easy/medium/hard/very-hard) per language — the original
  Go benchmark had this, extend to Python/TypeScript/Rust
- Add multi-hop questions requiring cross-file reasoning: _"Trace the request
  lifecycle from Router.ServeHTTP through middleware to the final handler"_
- Add coverage to all 6+ languages in `testdata/fixtures/`

**Ground truth mechanism:** LLM judge (Opus) — acceptable for this category
since there's no single "correct" answer to architecture questions. Strengthen
by:

- Adding structured rubrics to judge prompts (explicit criteria with weights)
- Running judge 3× and taking majority vote to reduce judge variance

### Category 2: Symbol Location (new — adapt from code navigation benchmarks)

**What it tests:** Can Claude find specific functions, types, or methods?

**Why this matters:** This is Lumen's core value prop — semantic search should
find symbols faster than grep/glob traversal.

**Existing benchmarks to adapt:**

| Benchmark                  | What it does                                                     | Adaptable? |
| -------------------------- | ---------------------------------------------------------------- | ---------- |
| **CodeSearchNet**          | Maps NL queries → code functions across 6 languages              | Yes — directly |
| **CoSQA** (Microsoft)      | 20K NL↔code pairs from Bing search logs                          | Yes — query format |
| **Code Search Challenge**  | NL → function retrieval                                          | Yes — scoring method |

**Design:**

```json
{
  "id": "go-router-serve",
  "language": "go",
  "query": "Find the function that handles HTTP request routing and dispatches to registered handlers",
  "expected": {
    "file": "mux.go",
    "symbol": "Router.ServeHTTP",
    "line_range": [145, 180]
  },
  "difficulty": "easy"
}
```

**Scoring:** Binary — did Claude's answer reference the correct file and symbol?
Also measure tokens/time to get there. No LLM judge needed.

**Question generation:** For each fixture set, extract all exported
functions/types using AST (Lumen's chunker already does this), then write NL
descriptions for a subset. This gives us hundreds of ground-truth pairs cheaply.

### Category 3: Cross-File Code Completion (adapt CrossCodeEval / RepoEval)

**What it tests:** Given a repo, can Claude complete a function body that
requires understanding code in other files?

**Existing benchmarks to adapt:**

| Benchmark                       | Source   | Languages             | Ground truth      |
| ------------------------------- | -------- | --------------------- | ----------------- |
| **CrossCodeEval** (Microsoft)   | ICLR '24 | Python, Java, TS, C#  | Exact match + EM  |
| **RepoEval** (Microsoft)        | NeurIPS  | Python                | Exact match       |
| **RepoBench** (Liu et al.)      | ICLR '24 | Python, Java          | Exact match + ES  |

**Why these are ideal:** They specifically test whether cross-file context
improves completions — exactly what Lumen provides. They have:

- Clear ground truth (the actual code that was there)
- Established metrics (Exact Match, Edit Similarity, CodeBLEU)
- Published baselines for comparison

**Design:**

1. Take a function from fixtures that references types/functions from other
   files
2. Blank out the function body
3. Ask Claude to complete it, once with baseline tools, once with semantic_search
4. Score against the original using Edit Similarity (ES) — more forgiving than
   exact match

**Adaptation approach:** Rather than using their full dataset, apply their
**methodology** to our fixtures:

- Extract functions with cross-file dependencies (imports, type references)
- Create completion prompts in their format
- Use their scoring metrics
- Compare: does `semantic_search` context produce higher ES than grep/glob?

### Category 4: Task Completion with Tests (adapt Aider Polyglot / SWE-bench Lite)

**What it tests:** Can Claude make a code change that passes tests?

**Existing benchmarks to adapt:**

| Benchmark               | Tasks | Languages | Ground truth           |
| ------------------------ | ----- | --------- | ---------------------- |
| **Aider Polyglot**       | 225   | 7 langs   | Test pass/fail         |
| **SWE-bench Lite**       | 300   | Python    | Test pass/fail         |
| **SWE-bench Verified**   | 500   | Python    | Human-verified patches |
| **Multi-SWE-bench**      | 856   | 19 langs  | Test pass/fail         |

**SWE-bench is the gold standard** for measuring coding agent performance, but:

- Full SWE-bench is expensive (~$100+ per full run)
- Setting up repo environments is complex
- It tests patch generation, not just search

**Recommended approach — SWE-bench Lite subset:**

1. Pick 20-30 tasks from SWE-bench Lite that are **search-bottlenecked** (the
   fix is simple but finding where to fix is hard)
2. Run each task with Claude + baseline tools vs Claude + Lumen
3. Measure: resolve rate, tokens used, time taken
4. Ground truth: the existing SWE-bench test harness

**Alternative — Aider-style against our fixtures:**

1. Create 3-5 small tasks per language against fixture code
2. Each task has a test file that currently fails
3. Claude makes edits, we run the test
4. Binary pass/fail + cost metrics

**The Aider approach is more practical** for regular CI runs. SWE-bench is
better for one-time publishable results.

### Category 5: Bug Finding (new — simple ground truth)

**What it tests:** Can Claude find a known bug in the codebase?

**No standard benchmark exists for this** with code search tools. But it's easy
to create:

1. Take a fixture file, introduce a realistic bug (nil deref, off-by-one,
   missing error check, race condition)
2. Store the bug location and type as ground truth
3. Ask Claude: "There's a bug in the error handling of package X. Find it."
4. Score: did Claude identify the correct file + line range + bug type?

**Why include this:** Every developer uses search to find bugs. If Lumen helps
Claude find bugs faster/cheaper, that's directly valuable.

---

## Metrics Framework

### Primary Metrics (capture for every benchmark run)

| Metric | Source | Purpose |
|--------|--------|---------|
| `input_tokens` | `claude --output-format stream-json` → result event | Cost driver |
| `output_tokens` | result event | Measures response precision |
| `duration_ms` | result event | Wall-clock time |
| `cost_usd` | result event | Dollar cost |
| `tool_calls` | count tool_use events in stream | Search efficiency |
| `correct` | ground truth check | Accuracy (categories 2-5) |
| `judge_rank` | LLM judge | Quality (category 1) |

### Derived Metrics

| Metric | Formula | What it shows |
|--------|---------|---------------|
| **Cost per correct answer** | `total_cost / correct_count` | ROI |
| **Token efficiency** | `correct_count / total_tokens` | Search precision |
| **Time to first relevant result** | `first_tool_result_ms` | Responsiveness |
| **First-call hit rate** | `first_search_returns_relevant / total` | Retrieval quality |
| **Search recall** | `relevant_files_found / relevant_files_total` | Completeness |

### Reporting

For each category × scenario × model:

```
| Scenario  | Accuracy | Median Cost | Median Time | Median Tokens | Cost/Correct |
|-----------|----------|-------------|-------------|---------------|--------------|
| baseline  | 60%      | $1.20       | 45s         | 12,400        | $2.00        |
| mcp-only  | 80%      | $0.35       | 18s         | 3,200         | $0.44        |
| mcp-full  | 85%      | $0.50       | 22s         | 4,800         | $0.59        |
```

---

## Statistical Validity

### Current problem

Each question runs once per scenario. LLM outputs are non-deterministic, so a
single run may not be representative.

### Approach

- **Minimum 3 runs per question×scenario** (configurable `--runs N`)
- Report: **median** (primary), min, max, stddev
- For accuracy: report **pass rate** across runs (e.g., "3/3", "2/3")
- Use `--effort medium` consistently to reduce variance (already done)
- Use deterministic fixture indexing (already done — index built once)
- **Paired comparison:** Always run baseline and mcp-only on the same question
  in the same session to control for API latency variance

### Sample size guidance

| Purpose | Runs per question | Total runs (30 questions × 3 scenarios) |
|---------|-------------------|-----------------------------------------|
| Quick sanity check | 1 | 90 |
| Development iteration | 3 | 270 |
| Publishable results | 5 | 450 |

---

## Existing Benchmarks: Concrete Adaptation Plan

### 1. CrossCodeEval → Category 3 (highest priority to adapt)

**Source:** https://github.com/AmazonScience/CrossCodeEval (Apache-2.0)

**What to take:**
- Their evaluation methodology (cross-file completion with retrieval)
- Their metrics: Exact Match (EM) and Edit Similarity (ES)
- Their prompt format for completion tasks

**What to change:**
- Use our fixture repos instead of their dataset (they use Python/Java/TS/C#)
- Replace their retrieval methods with Lumen's semantic_search vs baseline grep
- Run through Claude CLI instead of direct model API

**Effort:** Medium — need to create completion tasks from fixtures, implement
ES scoring

### 2. SWE-bench Lite → Category 4 (most credible for marketing)

**Source:** https://github.com/princeton-nlp/SWE-bench (MIT)

**What to take:**
- Their test harness and evaluation infrastructure
- A curated subset of 20-30 search-bottlenecked tasks
- Their standardized metrics (% resolved)

**What to change:**
- Add Lumen as an MCP server in the agent's tool configuration
- Compare: resolve rate with vs without Lumen
- Add token/cost tracking (they don't track this by default)

**Effort:** High — SWE-bench has complex Docker-based test environments.
Consider using existing SWE-bench harness (like SWE-agent or Aider's runner)
and just swapping the tool configuration.

**Filtering for search-bottlenecked tasks:** Select tasks where:
- The fix touches ≤ 3 files
- The repo has > 100 files (so search matters)
- Baseline agents spend > 50% of tokens on file exploration

### 3. CodeSearchNet → Category 2 (easiest to adapt)

**Source:** https://github.com/github/CodeSearchNet (MIT)

**What to take:**
- Their query↔function pairs for Go, Python, Java, JavaScript, PHP, Ruby
- Their evaluation metrics (MRR, NDCG)

**What to change:**
- Instead of measuring retrieval model quality, measure whether Claude+Lumen
  finds the right function faster/cheaper than Claude+grep
- Use their NL queries as-is, but evaluate Claude's answer rather than
  embedding similarity

**Effort:** Low — just need to select a subset of their queries that map to our
fixture files, or use their repos directly.

### 4. Aider Polyglot → Category 4 (practical alternative to SWE-bench)

**Source:** https://github.com/Aider-AI/aider (Apache-2.0)

**What to take:**
- Their exercism-based task format (write code to pass tests)
- Multi-language coverage (Python, JS, TS, Go, Rust, Java, C#, etc.)
- Their scoring approach (test pass/fail)

**What to change:**
- Run through Claude CLI instead of Aider
- Add Lumen MCP and compare with/without
- Track tokens/cost (Aider doesn't break this down)

**Effort:** Medium — need to adapt their runner to use Claude CLI

---

## Priority Order

Based on effort vs value:

| Priority | Category | Benchmark Source | Effort | Value |
|----------|----------|-----------------|--------|-------|
| **P0** | Symbol Location | CodeSearchNet subset | Low | High — objective, fast, cheap to run |
| **P0** | Code Understanding | Extend existing bench-mcp.sh | Low | High — already proven, just needs more questions |
| **P1** | Cross-File Completion | CrossCodeEval methodology | Medium | High — directly tests Lumen's value prop |
| **P1** | Bug Finding | Custom (no existing benchmark) | Medium | High — compelling real-world use case |
| **P2** | Task Completion | Aider Polyglot tasks | Medium | Medium — realistic but expensive to run |
| **P3** | Task Completion | SWE-bench Lite subset | High | Very high — gold standard credibility |

### Recommended starting point

1. **Extend bench-mcp.sh** with more questions, multi-run support, and
   languages — quick wins, proven approach
2. **Add symbol-location benchmark** using CodeSearchNet-style queries against
   fixtures — objective ground truth, cheap to run, fast iteration
3. **Add CrossCodeEval-style completion** — the strongest evidence that semantic
   search improves Claude's coding ability
4. **SWE-bench subset** — save for publishable results, too expensive for
   regular CI

---

## Implementation Sketch

### bench-mcp.sh extensions

```bash
# New flags
--runs N          # Run each question N times, report median/min/max
--category NAME   # Filter: knowledge | symbol | completion | bug
--language LANG   # Filter: go | python | typescript | rust | ...
```

### Question format (externalized from script)

```json
{
  "questions": [
    {
      "id": "go-tsdb-compaction",
      "category": "knowledge",
      "language": "go",
      "difficulty": "hard",
      "fixtures": "testdata/fixtures/go",
      "prompt": "How does TSDB compaction work end-to-end?...",
      "ground_truth": null,
      "judge_rubric": "Must cover: Compactor interface, LeveledCompactor, trigger paths"
    },
    {
      "id": "go-find-router-serve",
      "category": "symbol",
      "language": "go",
      "difficulty": "easy",
      "fixtures": "testdata/fixtures/go",
      "prompt": "Find the function that matches labels against matchers",
      "ground_truth": {
        "file": "labels.go",
        "symbol": "Matcher.Matches",
        "line_range": [42, 55]
      }
    }
  ]
}
```

### Scoring script (bench-score.sh)

For symbol-location and bug-finding, automated scoring:

```bash
# Extract file:line references from Claude's answer
# Check against ground_truth.file and ground_truth.line_range
# Output: correct/incorrect + cost metrics
```

### Directory structure

```
testdata/
  benchmarks/
    questions.json              # All questions, all categories
    completions/                # Blanked function bodies + originals
    bugs/                       # Bug-planted files + metadata
bench-mcp.sh                   # Extended runner
bench-score.sh                 # Ground-truth scorer
docs/BENCHMARK-DESIGN.md       # This document
docs/BENCHMARKS.md             # Results (existing)
```

---

## Open Questions

1. **Should we publish raw SWE-bench numbers?** Running even a subset is
   expensive ($200-500), but "Lumen improves SWE-bench resolve rate by X%" is
   extremely credible.

2. **How to handle non-determinism in ground-truth scoring?** Claude might find
   the right symbol but format the answer differently each time. Need robust
   answer parsing — probably regex for `file:line` patterns plus symbol name
   matching.

3. **Should benchmarks run in CI?** Symbol-location and knowledge questions are
   cheap enough for CI (~$5-10 per run). Completion and task benchmarks are
   better as periodic or pre-release checks.

4. **Embedding model comparison matrix:** The extended benchmarks already
   compare 4 embedding models. New benchmark categories should maintain this
   practice — each category run across jina, qwen3-8b, qwen3-4b, nomic to
   guide model selection.
