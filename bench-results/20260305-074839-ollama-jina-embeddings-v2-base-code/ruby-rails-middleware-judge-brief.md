## Content Quality

**haiku/baseline** — Most comprehensive answer with accurate detail on ActionController::MiddlewareStack, action filtering strategies (INCLUDE/EXCLUDE/NULL), Engine middleware assembly, and Sinatra comparison. Includes specific file:line references (metal.rb:18-63, engine.rb:514-522) and a clear request flow diagram. Minor issue: includes Sinatra coverage that's tangential to the Rails question.

**haiku/solo** — Strong technical depth with accurate code excerpts and clear call signature hierarchy. Good file references (metal.rb:288-337, engine.rb:515-523) and thorough explanation of action-aware middleware filtering. Well-structured with a clean class summary table. Slightly more focused than baseline.

**haiku/together** — Solid coverage with good structure and accurate content. Includes env_config details and middleware manipulation methods (insert_before, swap, delete) that others miss. However, the "Default Middleware Stack" section is vague ("referenced but implementation details in separate file") and the code excerpts are slightly less detailed than the other two.

## Efficiency

Solo is dramatically cheaper ($0.047) and faster (30s) than baseline ($0.213, 77s) while delivering comparable quality — a 4.5x cost reduction with minimal quality loss. Together falls in between on cost ($0.081) with high cache read tokens (255K) but doesn't produce a noticeably better answer than solo.

## Verdict

**Winner: haiku/solo**
