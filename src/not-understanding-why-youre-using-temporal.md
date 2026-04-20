# Not Understanding Why You're Using Temporal

> [!TIP]
> Temporal trades simplicity for specific guarantees -- durability, reliability, and visibility. Without understanding these trade-offs, teams are prone to misuse Temporal where simpler tools work just as well, or under-use it by not leveraging its programming model.

## What?

Teams adopt Temporal because they've heard it solves distributed systems problems, or because another team in the organization uses it successfully. But they skip the step of understanding what specific problems Temporal is designed to solve and whether those problems match their use case.

This leads to two failure modes. The first is misuse: using Temporal for things that don't need it, like short-lived request-reply HTTP handlers or simple CRUD operations. You end up with all the operational complexity of Temporal for workflows that would be fine as a plain function call. The second is under-use: adopting Temporal but writing code as if it were a regular application framework, missing durability guarantees, [replay](terms/replay.md) semantics, and fault tolerance -- the very things that justify the complexity.

## Why?

Temporal is not a general-purpose application framework. It trades simplicity for specific guarantees:

- **Durability**: workflow state survives process crashes, restarts, and deployments. No need to manually persist checkpoints or build recovery logic.
- **Reliability**: activities retry automatically on failure. You get exactly-once execution semantics for workflow logic without writing retry loops.
- **[Visibility](terms/visibility.md)**: every workflow execution has a full history that you can inspect, debug, and audit after the fact.

If your use case doesn't need these properties, Temporal adds overhead without payoff. If your use case does need them but your team doesn't understand the programming model, you'll fight against it -- writing [non-deterministic](terms/non-determinism.md) workflow code, ignoring [versioning](terms/versioning.md) constraints, or treating activities like regular function calls.

Understanding the "why" also helps with buy-in. When your team understands that Temporal eliminates entire categories of failure handling code, they'll write better workflows and be more willing to adopt the constraints that come with the programming model.

## How?

**Before adopting Temporal**, answer these questions concretely:
1. What processes in our system need to survive crashes or restarts?
2. Where are we currently writing manual retry logic, state machines, or recovery code?
3. Do we have long-running processes that span minutes, hours, or days?
4. Do we need visibility into the state of in-flight operations?

If you can't point to specific problems in your codebase that match these, Temporal may not be the right tool.

**During adoption**, invest in education. Make sure the team understands the [replay](terms/replay.md) model, determinism constraints, and the difference between workflow code and activity code. You cannot learn these by trial and error without wasting significant time debugging non-determinism errors.

**After adoption**, periodically review whether new workflows actually need Temporal's guarantees or are written as Temporal workflows out of habit. Not every piece of business logic needs to be durable. Parts of your system that don't need Temporal shouldn't use it.
