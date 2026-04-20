# Not Understanding Why You're Using Temporal

> [!TIP]
> Temporal trades simplicity for specific guarantees -- durability, reliability, and visibility. Without understanding these trade-offs, teams are prone to misuse Temporal where simpler tools work just as well, or under-use it by not leveraging its programming model.

Teams adopt Temporal because they've heard it solves distributed systems problems, or because another team uses it successfully. But they skip understanding what specific problems Temporal solves and whether those match their use case. This leads to two failure modes: misuse (using Temporal where simpler tools suffice) and under-use (adopting Temporal but writing code as if it were a regular framework, missing the [replay](terms/replay.md) semantics and durability guarantees that justify the complexity).

Temporal provides durability (workflow state survives crashes), reliability (activities retry automatically), and [visibility](terms/visibility.md) (every execution has a full inspectable history). If your use case doesn't need these, Temporal adds overhead without payoff. If it does but your team doesn't understand the programming model, you'll fight against it -- writing [non-deterministic](terms/non-determinism.md) workflow code, ignoring [versioning](terms/versioning.md) constraints, or treating activities like regular function calls.

Before adopting, answer concretely: what processes need to survive crashes? Where are you writing manual retry logic or state machines? During adoption, invest in education -- the replay model and determinism constraints cannot be learned by trial and error. After adoption, periodically review whether new workflows actually need Temporal's guarantees or are written as workflows out of habit.

See also: [Unnecessary Usage of Workflows](unnecessary-usage-of-workflows.md).
