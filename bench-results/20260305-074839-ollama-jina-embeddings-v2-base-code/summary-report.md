# Benchmark Summary

Generated: 2026-03-05 06:56 UTC  |  Results: `20260305-074839-ollama-jina-embeddings-v2-base-code`

| Scenario | Description |
|----------|-------------|
| **baseline** | All default Claude tools, no MCP |
| **solo** | `semantic_search` MCP tool only |
| **together** | All default tools + MCP |

## Overall: Aggregated by Scenario

Totals across all 8 questions × 1 models.

| Model | Scenario | Total Time | Total Input Tok | Total Output Tok | Total Cost (USD) |
|-------|----------|------------|-----------------|------------------|------------------|
| **haiku** | baseline | 356.4s | 1949 | 31073 | $0.8111 |
| **haiku** | solo | 303.7s | 472 | 33164 | $0.5370 |
| **haiku** | together | 327.4s | 6947 | 34334 | $0.6796 |

---

## go-registry-concurrency [go / hard]

> How does TSDB compaction work end-to-end? Explain the Compactor interface, LeveledCompactor, and how the DB triggers compaction. Show relevant types, interfaces, and key method signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **haiku** | baseline | 62.0s | 50 | 247666 | 4007 | $0.1614 |  |
| **haiku** | solo | 25.3s | 34 | 46349 | 2336 | $0.0493 |  |
| **haiku** | together | 41.7s | 550 | 190151 | 3405 | $0.0897 | 🏆 Winner |

### Quality Ranking (Opus 4.6)

## Content Quality

**haiku/baseline**: Most comprehensive — covers all phases (Plan, Compact, Write, PopulateBlock), includes BlockPopulator interface, configuration Options struct, and two clear flow diagrams. File/line references are specific (e.g., compact.go:52-77, db.go:1175-1243). Minor issue: BlockMeta fields shown as strings instead of proper types (ULID should be `ulid.ULID`).

**haiku/together**: Nearly as complete — covers the same core interfaces with accurate signatures, includes OOO compaction and stale series compaction (unique among the three), and adds a useful design points table. Line references are present and specific. The "Based on the code in the test fixtures" preamble is odd/misleading but content is solid.

**haiku/solo**: Correct and well-structured but slightly less detailed — omits BlockPopulator interface, OOO compaction, and configuration options. The summary table of methods is a nice touch for quick reference. Line references are present throughout.

## Efficiency

Solo is dramatically cheaper ($0.049 vs $0.161 baseline, $0.090 together) and fastest (25s vs 62s, 42s), using far fewer tokens. Together offers a strong middle ground — nearly baseline-level completeness at ~55% of the cost. Baseline's 4x cost premium over solo buys marginal additional detail.

## Verdict

**Winner: haiku/together**

---

## py-django-queryset [python / hard]

> How does the Django QuerySet evaluation and filtering pipeline work? Explain QuerySet chaining, lazy evaluation, the Query class, how lookups and filters are compiled into SQL, and how the Manager ties it all together. Show key classes and method signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **haiku** | baseline | 44.7s | 66 | 293563 | 4468 | $0.0927 |  |
| **haiku** | solo | 32.6s | 34 | 38061 | 3645 | $0.0478 | 🏆 Winner |
| **haiku** | together | 56.3s | 84 | 329016 | 5904 | $0.1013 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

1. **haiku/together** — Most comprehensive: includes WhereNode tree visualizations, detailed `Manager.from_queryset()` with `_get_queryset_methods`, and a clear end-to-end flow diagram with file references; covers all topics thoroughly with good structural clarity.
2. **haiku/solo** — Strong coverage with specific file:line references (e.g., `django-query.py:306–321`, `django-query.py:2168–2172`), clean summary table with locations, and accurate code; slightly less detailed on WhereNode and Manager internals than together.
3. **haiku/baseline** — Solid and correct with a nice step-by-step chaining walkthrough, but lacks file:line references to the actual fixtures, and some class signatures (e.g., `SQLQuery`) feel more reconstructed than grounded in the source.

## Efficiency

Solo is the clear efficiency winner at $0.048 and 32.6s — roughly half the cost and fastest runtime — while together costs $0.10 for only marginally better content. Baseline sits in between at $0.09 but delivers the least grounded answer, making it the worst value overall.

**Winner: haiku/solo**

---

## ts-disposable-events [typescript / hard]

> How do Disposable and IDisposable work together with the EventEmitter system? Explain the lifecycle management pattern, how listeners are registered and cleaned up, and how events are typed and fired. Show key interfaces and class relationships.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **haiku** | baseline | 40.9s | 82 | 408794 | 3904 | $0.0985 |  |
| **haiku** | solo | 45.2s | 98 | 281246 | 4522 | $0.1012 |  |
| **haiku** | together | 42.3s | 67 | 331457 | 4281 | $0.1163 | 🏆 Winner |

### Quality Ranking (Opus 4.6)

## Content Quality

**haiku/together** is the most complete answer: it covers all key interfaces with accurate line references, provides a clear class hierarchy, includes the full registration/firing/cleanup lifecycle with code, documents EmitterOptions hooks, shows a realistic usage example, and has a well-organized summary table of design features.

**haiku/solo** is nearly as thorough, with good coverage of the same topics, accurate code excerpts, and a useful integration example (EventMultiplexer), though its organization is slightly less cohesive and the memory leak prevention section feels tacked on.

**haiku/baseline** covers the same ground with correct information and decent line references, but its structure is more fragmented with smaller sections and the "Complete Example: Debounced Event" feels less illustrative of the core pattern than the Document/DisposableStore example in the other answers.

## Efficiency

All three runs are nearly identical in duration (~40-45s) and cost ($0.098-$0.116), with baseline being marginally cheapest and together being most expensive. The cost difference is small (~$0.02) and together's slightly higher cost buys the most polished, well-structured answer.

## Verdict

**Winner: haiku/together**

---

## java-petclinic-domain [java / hard]

> How is the PetClinic domain model structured? Explain the entity hierarchy (Owner, Pet, Visit, Vet), how JPA/Hibernate maps the relationships, and how the repository layer exposes data access. Show key classes, annotations, and method signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **haiku** | baseline | 13.6s | 26 | 78504 | 1569 | $0.0239 |  |
| **haiku** | solo | 27.8s | 42 | 54495 | 3773 | $0.0381 | 🏆 Winner |
| **haiku** | together | 25.6s | 34 | 119560 | 3615 | $0.0484 |  |

### Quality Ranking (Opus 4.6)



## Content Quality

**haiku/together** and **haiku/solo** are very close — both present the full entity hierarchy (Owner, Pet, Visit, Vet, PetType, Specialty), all JPA relationship annotations with cascade/fetch strategies, and the repository layer with method signatures. Together has a slightly better-organized relationship summary table but includes an error in the hierarchy diagram (placing Pet under Visit). **haiku/solo** is accurate throughout with clean structure and correct hierarchy, plus good design pattern observations. **haiku/baseline** is noticeably weaker — it only covers the three fixture files actually present in the repo (BaseEntity, NamedEntity, Person, PetTypeRepository) and explicitly notes the rest is missing, making it incomplete for the question asked.

Rank: haiku/solo > haiku/together > haiku/baseline

## Efficiency

Baseline is cheapest ($0.024) and fastest (13.6s) but delivers an incomplete answer. Solo ($0.038, 27.8s) provides a comprehensive answer at moderate cost. Together ($0.048, 25.6s) is the most expensive with similar quality to solo but 27% higher cost. Solo offers the best quality-to-cost tradeoff.

## Verdict

**Winner: haiku/solo**

---

## js-express-lifecycle [javascript / hard]

> How does Express handle the full request/response lifecycle? Explain middleware chaining, how the Router works, how error-handling middleware differs from regular middleware, and how app.use and route mounting compose. Show key function signatures and flow.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **haiku** | baseline | 35.0s | 26 | 79058 | 4185 | $0.0679 | 🏆 Winner |
| **haiku** | solo | 55.0s | 114 | 413657 | 6212 | $0.1232 |  |
| **haiku** | together | 32.8s | 26 | 82998 | 3916 | $0.0654 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

**haiku/baseline** is the strongest answer: it provides accurate code with specific file/line references (e.g., `express-express.js:36-39`, `express-application.js:190-244`), covers all requested topics thoroughly, includes a clear flow diagram, and adds practical details like the `mounted_app` wrapper restoring prototypes and the `app.get` special case for settings.

**haiku/together** is a close second: similarly thorough with file/line references, and uniquely covers lazy router instantiation and settings inheritance via `Object.setPrototypeOf` on the `mount` event — details the others miss — but is slightly more verbose and some code snippets feel padded.

**haiku/solo** ranks third: while comprehensive in structure, it lacks real file/line references (uses generic comments like "simplified" and "conceptual"), includes more pseudocode than actual source, and the router internals section is speculative rather than grounded in the actual codebase.

## Efficiency

Baseline and together are comparable in cost (~$0.065-0.068) and runtime (~33-35s), while solo cost nearly double ($0.12) and took 55s due to heavy cache reads (413K tokens) from exploration. The baseline offers the best quality-to-cost ratio — top-quality answer at the lowest cost and fastest time.

## Verdict

**Winner: haiku/baseline**

---

## ruby-rails-middleware [ruby / hard]

> How does the Rails middleware stack work? Explain how Rack middleware is assembled, how ActionDispatch integrates, how requests flow through the stack, and how custom middleware is added. Show key classes, modules, and call signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **haiku** | baseline | 77.2s | 1591 | 114571 | 3559 | $0.2128 |  |
| **haiku** | solo | 29.9s | 42 | 60198 | 2990 | $0.0470 | 🏆 Winner |
| **haiku** | together | 33.4s | 58 | 254743 | 3206 | $0.0809 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

**haiku/baseline** — Most comprehensive answer with accurate detail on ActionController::MiddlewareStack, action filtering strategies (INCLUDE/EXCLUDE/NULL), Engine middleware assembly, and Sinatra comparison. Includes specific file:line references (metal.rb:18-63, engine.rb:514-522) and a clear request flow diagram. Minor issue: includes Sinatra coverage that's tangential to the Rails question.

**haiku/solo** — Strong technical depth with accurate code excerpts and clear call signature hierarchy. Good file references (metal.rb:288-337, engine.rb:515-523) and thorough explanation of action-aware middleware filtering. Well-structured with a clean class summary table. Slightly more focused than baseline.

**haiku/together** — Solid coverage with good structure and accurate content. Includes env_config details and middleware manipulation methods (insert_before, swap, delete) that others miss. However, the "Default Middleware Stack" section is vague ("referenced but implementation details in separate file") and the code excerpts are slightly less detailed than the other two.

## Efficiency

Solo is dramatically cheaper ($0.047) and faster (30s) than baseline ($0.213, 77s) while delivering comparable quality — a 4.5x cost reduction with minimal quality loss. Together falls in between on cost ($0.081) with high cache read tokens (255K) but doesn't produce a noticeably better answer than solo.

## Verdict

**Winner: haiku/solo**

---

## rust-ripgrep-pipeline [rust / hard]

> How does ripgrep's search pipeline work end-to-end? Explain the searcher/matcher/sink architecture, how file walking is parallelized, how the Grep and Searcher types interact, and how results flow to the output layer. Show key traits, structs, and method signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **haiku** | baseline | 42.0s | 42 | 171533 | 4831 | $0.0747 | 🏆 Winner |
| **haiku** | solo | 52.4s | 50 | 125894 | 5822 | $0.0798 |  |
| **haiku** | together | 40.2s | 6028 | 165060 | 3897 | $0.0762 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

**haiku/baseline**: Most polished and comprehensive. Covers all layers with accurate signatures, includes a clear ASCII flow diagram, and references specific file locations (e.g., `rg-main.rs:77–101`, `rg-search.rs:39–90`). The method signatures summary at the end is a useful touch. Minor issue: some signatures are slightly simplified/approximated (e.g., the Sink trait shown twice with different signatures).

**haiku/together**: Strong architectural explanation with good emphasis on the "push model" design rationale. Correctly identifies the Sink sub-types (SinkMatch, SinkContext) and the ParallelVisitor trait. The data flow diagram is detailed. However, some Matcher signatures are slightly off (e.g., `&mut self` instead of `&self`), and file references are less precise than baseline.

**haiku/solo**: Thorough with extensive inline code from actual source. The parallel search code walkthrough is the most detailed of the three, showing atomic operations and buffer flushing clearly. However, the Sink trait is described as "implied, not directly shown" which is weaker, and the answer is the longest/most verbose without proportionally more insight.

## Efficiency

All three runs cost roughly the same ($0.075-$0.080) and completed in 40-52 seconds. Solo took longest (52.4s) and cost most ($0.080) while producing the most output tokens (5822). Together was fastest (40.2s) with fewest output tokens (3897) and moderate cost. Baseline sits in the middle on all dimensions. Together offers the best efficiency profile given comparable quality.

## Verdict

**Winner: haiku/baseline**

---

## php-laravel-container [php / hard]

> How does the Laravel service container resolve dependencies? Explain binding, contextual binding, automatic injection, how the container builds concrete classes, and how service providers register bindings. Show key classes, interfaces, and method signatures.

### Time & Tokens

| Model | Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost (USD) | Winner |
|-------|----------|----------|-----------|------------|------------|------------|--------|
| **haiku** | baseline | 40.6s | 66 | 258970 | 4550 | $0.0791 |  |
| **haiku** | solo | 35.2s | 58 | 90982 | 3864 | $0.0506 | 🏆 Winner |
| **haiku** | together | 54.7s | 100 | 416673 | 6110 | $0.1014 |  |

### Quality Ranking (Opus 4.6)

## Content Quality

**haiku/baseline** — Most comprehensive and well-organized. Covers all five topics with accurate code snippets, specific line references (e.g., `Container.php:278-308`, `Container.php:943-1008`), includes the internal data structures, and ends with a clean summary table and flow diagram. Minor issue: some code snippets appear slightly paraphrased rather than verbatim, but line references are consistent throughout.

**haiku/solo** — Nearly as complete as baseline, with the same specific line references and accurate code. Slightly better structured with a clear "Key Classes & Interfaces" table up front and a concise resolution flow summary. Coverage is equivalent but marginally more focused, avoiding some redundant detail.

**haiku/together** — The longest and most verbose answer. Adds a section on `Container::call()` and method-level DI that the others omit, which is a nice bonus. However, the extra length doesn't proportionally add value — much of the content duplicates what's in the other answers. The flow diagram at the end is helpful but the overall answer is bloated.

## Efficiency

haiku/solo is the clear efficiency winner at $0.051 and 35.2s, using roughly half the cache reads and 85% of the output tokens compared to baseline, while delivering equivalent content quality. haiku/together costs 2x more than solo and takes 55% longer, with the extra tokens mostly going to verbose prose rather than substantively new information.

## Verdict

**Winner: haiku/solo**

---

## Overall: Algorithm Comparison

| Question | Language | Difficulty | 🏆 Winner | Runner-up |
|----------|----------|------------|-----------|-----------|
| go-registry-concurrency | go | hard | haiku/together | haiku/solo |
| py-django-queryset | python | hard | haiku/solo | haiku/baseline |
| ts-disposable-events | typescript | hard | haiku/together | haiku/baseline |
| java-petclinic-domain | java | hard | haiku/solo | haiku/baseline |
| js-express-lifecycle | javascript | hard | haiku/baseline | haiku/together |
| ruby-rails-middleware | ruby | hard | haiku/solo | haiku/together |
| rust-ripgrep-pipeline | rust | hard | haiku/baseline | haiku/together |
| php-laravel-container | php | hard | haiku/solo | haiku/baseline |

**Scenario Win Counts** (across all 8 questions):

| Scenario | Wins |
|----------|------|
| baseline | 2 |
| solo | 4 |
| together | 2 |

**Overall winner: solo** — won 4 of 8 questions.

_Full answers and detailed analysis: `detail-report.md`_
