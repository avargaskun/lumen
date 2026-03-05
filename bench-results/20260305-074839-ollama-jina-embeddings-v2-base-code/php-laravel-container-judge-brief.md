## Content Quality

**haiku/baseline** — Most comprehensive and well-organized. Covers all five topics with accurate code snippets, specific line references (e.g., `Container.php:278-308`, `Container.php:943-1008`), includes the internal data structures, and ends with a clean summary table and flow diagram. Minor issue: some code snippets appear slightly paraphrased rather than verbatim, but line references are consistent throughout.

**haiku/solo** — Nearly as complete as baseline, with the same specific line references and accurate code. Slightly better structured with a clear "Key Classes & Interfaces" table up front and a concise resolution flow summary. Coverage is equivalent but marginally more focused, avoiding some redundant detail.

**haiku/together** — The longest and most verbose answer. Adds a section on `Container::call()` and method-level DI that the others omit, which is a nice bonus. However, the extra length doesn't proportionally add value — much of the content duplicates what's in the other answers. The flow diagram at the end is helpful but the overall answer is bloated.

## Efficiency

haiku/solo is the clear efficiency winner at $0.051 and 35.2s, using roughly half the cache reads and 85% of the output tokens compared to baseline, while delivering equivalent content quality. haiku/together costs 2x more than solo and takes 55% longer, with the extra tokens mostly going to verbose prose rather than substantively new information.

## Verdict

**Winner: haiku/solo**
