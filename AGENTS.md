# AGENTS.md

This repository is a learning project for implementing Web server internals in
Go. Agents should optimize for understanding, incremental progress, and clear
explanations rather than feature volume.

## Project Intent

The project explores how Web servers and Web frameworks work underneath common
application-level abstractions.

The learner is already comfortable with Go and backend development, so avoid
spending too much time on basic Go syntax or ordinary CRUD API architecture.
Prefer deeper discussion of HTTP, TCP, parsing, concurrency, routing,
middleware, streaming, and robustness.

## Working Style

- Proceed step by step.
- Before a major implementation step, clarify the specific learning objective.
- After a meaningful implementation step, summarize what was learned and what
  remains unclear.
- Keep changes small and inspectable.
- Prefer standard library implementations first when the goal is to understand
  the mechanism.
- Use external libraries only when they help compare designs or when the learner
  explicitly wants to study that library.
- Keep `TODO.md` updated as a living roadmap, not a fixed plan.
- If the learning direction changes, update the roadmap instead of forcing the
  original plan.

## Implementation Guidance

- Prefer starting from `net.Listener` and raw TCP behavior before using
  `net/http` abstractions.
- When implementing protocol behavior, include tests or small reproducible
  examples where practical.
- Make parsing logic explicit enough to study. State machines and incremental
  parsing are preferred over opaque shortcuts when the topic is HTTP parsing.
- Be careful with goroutine lifecycle, cancellation, deadlines, and resource
  cleanup.
- When introducing abstractions such as handlers, middleware, or routers, explain
  what problem the abstraction solves and which behavior it hides.
- When comparing with `net/http` or common frameworks, focus on the underlying
  behavior rather than surface API convenience.

## Documentation Guidance

- `README.md` should describe the project purpose and scope.
- `TODO.md` should track the current learning roadmap, progress, and open
  questions.
- Add notes to `TODO.md` when a completed step changes the next learning
  direction.
- Do not treat the roadmap as immutable.
