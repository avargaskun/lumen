## Content Quality

**Rank: 1st — haiku / solo, 2nd — haiku / baseline, 3rd — haiku / together**

**haiku / solo** provides the most technically precise and well-structured answer. It correctly explains the Rack interface, Rack::Builder assembly, ActionDispatch::MiddlewareStack, and the controller-level ActionController::MiddlewareStack with action filtering. The code excerpts from `metal.rb` and `engine.rb` are accurate with specific line references (metal.rb:288-337, metal.rb:18-63, engine.rb:515-523). The action-filtering strategy explanation (INCLUDE/EXCLUDE/NULL lambdas) is thorough and correctly sourced. The call signature hierarchy at the end is a clean summary. It also correctly notes the thread-safe caching via `@app_build_lock`. The one weakness is it references some classes without file locations (e.g., `ActionDispatch::Request`), but this is minor.

**haiku / baseline** is nearly as strong, covering the same core concepts with accurate code references. It adds a useful comparison of three composition patterns (Metal Controllers, Sinatra, Rails Engines) and includes a visual request flow diagram. The file/line references (metal.rb:18-63, metal.rb:315-327, engine.rb:514-522) are precise. However, it spends significant space on Sinatra's middleware assembly, which is tangential to the Rails-focused question. The default middleware stack listing (application.rb:41-43, 66) appears to be inferred/assembled rather than directly read from code, which slightly reduces precision. The key call signatures table at the end is helpful.

**haiku / together** is the weakest of the three, though still competent. It covers the right topics but with less code depth. Several code snippets appear to be reconstructed from general Rails knowledge rather than directly extracted from fixture files (e.g., the `config.middleware.insert_before`, `config.middleware.swap` API, and the default middleware list). The file references are sparser and some line numbers (e.g., application.rb:768-770, engine.rb:766-768) don't appear to be verified against actual fixture content. The "Summary: Request Lifecycle" numbered list is useful but generic. It correctly identifies the key classes but doesn't go as deep into the action-filtering middleware internals.

## Efficiency Analysis

| Metric | baseline | solo | together |
|--------|----------|------|----------|
| Duration | 77.2s | 29.9s | 33.4s |
| Input Tokens | 1,591 | 42 | 58 |
| Cache Read | 114,571 | 60,198 | 254,743 |
| Output Tokens | 3,559 | 2,990 | 3,206 |
| Cost | $0.213 | $0.047 | $0.081 |

**haiku / solo** is the clear winner on efficiency — it produced the best answer at the lowest cost ($0.047), fastest time (29.9s), and fewest tokens consumed. It read roughly half the cached context of baseline and a quarter of together, yet delivered the most precise output.

**haiku / together** consumed by far the most cache-read tokens (254K) at nearly double the cost of solo, yet produced a weaker answer. The large context read didn't translate into better quality — suggesting it pulled in too much irrelevant material.

**haiku / baseline** was the slowest (77.2s) and most expensive ($0.213, 4.5x the cost of solo) while producing a middle-ranked answer. The extra time and tokens went partly toward the Sinatra tangent.

**Recommendation**: **haiku / solo** offers the best quality-to-cost tradeoff by a wide margin — best content quality at 22% the cost of baseline and 58% the cost of together. It demonstrates that focused, efficient context retrieval outperforms both broader searches (together) and heavier baseline approaches.
