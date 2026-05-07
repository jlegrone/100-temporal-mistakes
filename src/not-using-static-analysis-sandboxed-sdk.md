# Not Using Static Analysis or the Sandboxed SDK

> [!TIP]
> Most Temporal SDKs ship with tools to catch [non-deterministic](terms/non-determinism.md) code at development time -- sandboxes, linters, and isolates. Catching violations early is far cheaper than discovering them in production during [replay](terms/replay.md).

Without early detection, non-determinism errors only surface when a workflow actually replays -- typically after a [worker](terms/worker.md) restart or deployment. By that point workflows will not be able to make progress and you are left with a difficult operational problem.

<!-- TODO: Fact check this whole paragraph. Also make sure that semantics are correct, and link to relevant SDK docs. -->
The TypeScript SDK runs workflow code inside a V8 isolate sandbox by default, replacing `Date.now()`, `Math.random()`, `setTimeout`, and network APIs with deterministic alternatives. The Python and .NET SDKs include runtime sandboxes that detect imports of known non-deterministic modules. Go doesn't have a runtime sandbox, but static analysis tools can flag common patterns like direct use of `time.Now()`, `rand`, or goroutine creation with `go` instead of `workflow.Go`.

<!-- TODO(jlegrone): Create or link to a Go linter for Temporal workflow determinism checks -->

Enable sandbox/linting in CI so violations are caught before code is merged. When the sandbox or linter flags something, investigate rather than disabling the check.
