## Content Quality

1. **haiku/together** — Most comprehensive: includes WhereNode tree visualizations, detailed `Manager.from_queryset()` with `_get_queryset_methods`, and a clear end-to-end flow diagram with file references; covers all topics thoroughly with good structural clarity.
2. **haiku/solo** — Strong coverage with specific file:line references (e.g., `django-query.py:306–321`, `django-query.py:2168–2172`), clean summary table with locations, and accurate code; slightly less detailed on WhereNode and Manager internals than together.
3. **haiku/baseline** — Solid and correct with a nice step-by-step chaining walkthrough, but lacks file:line references to the actual fixtures, and some class signatures (e.g., `SQLQuery`) feel more reconstructed than grounded in the source.

## Efficiency

Solo is the clear efficiency winner at $0.048 and 32.6s — roughly half the cost and fastest runtime — while together costs $0.10 for only marginally better content. Baseline sits in between at $0.09 but delivers the least grounded answer, making it the worst value overall.

**Winner: haiku/solo**
