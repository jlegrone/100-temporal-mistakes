# Not Using Static Analysis or the Sandboxed SDK

> [!TIP]
> * Several Temporal SDKs provide built-in mechanisms to detect [non-deterministic](terms/non-determinism.md) code -- sandboxes, linters, and isolates -- but they only help if you enable and pay attention to them.
> * Catching non-determinism at development time or during CI is far cheaper than discovering it in production during [replay](terms/replay.md).
> * Choose the tools appropriate for your SDK: TypeScript has a V8 isolate sandbox, Python and .NET have runtime sandboxing, Go has linting rules.

## What?

Writing deterministic workflow code is one of Temporal's fundamental requirements. Workflow code must produce the same sequence of commands when replayed as it did during the original execution. Violating this rule (by using random numbers, current time, network calls, or non-deterministic data structures directly) causes non-determinism errors that break [replay](terms/replay.md).

The good news is that most Temporal SDKs ship with tools to catch these violations early. The bad news is that many teams either don't know these tools exist or don't enable them.

## Why?

Without early detection, non-determinism errors only surface when a workflow is actually replayed -- typically after a [worker](terms/worker.md) restart, a deployment, or a rebalance. By that point:

- The workflow is stuck and can't make progress.
- The error message may be cryptic (e.g., "non-deterministic workflow detected" with a history event mismatch).
- Diagnosing which line of code caused the issue requires careful comparison of the expected and actual command sequences.
- All in-flight workflows running the affected code are potentially impacted.

Catching these issues during development or in CI avoids all of this.

## How?

### TypeScript

The TypeScript SDK runs workflow code inside a V8 isolate sandbox by default. This sandbox restricts access to non-deterministic APIs -- `Date.now()`, `Math.random()`, `setTimeout`, and network APIs are replaced with deterministic alternatives. If your workflow code tries to use a forbidden API, you get an immediate error.

The sandbox is enabled by default. Make sure you haven't disabled it (via `unsafeAllowNonDeterminism` or similar configuration).

### Python

The Python SDK includes a runtime sandbox that detects imports of known non-deterministic modules (like `random`, `datetime`, `os`) and operations within workflow code. It raises warnings or errors when it detects violations.

Review the sandbox restrictions in the SDK documentation and ensure they're not suppressed in your worker configuration.

### .NET

The .NET SDK provides runtime detection of non-deterministic calls within workflow code. Similar to Python, it monitors for usage of restricted APIs during execution.

### Go

Go doesn't have a runtime sandbox, but the community provides linting rules that catch common non-deterministic patterns at build time. Use static analysis tools like `go vet` or custom linters configured for Temporal to flag issues such as:

- Direct use of `time.Now()` instead of `workflow.Now()`.
- Use of `rand` instead of `workflow.SideEffect`.
- Goroutine creation with `go` instead of `workflow.Go`.

### General recommendations

1. **Enable sandbox/linting in CI**: Make determinism checks part of your continuous integration pipeline so violations are caught before code is merged.
2. **Don't suppress warnings**: When the sandbox or linter flags something, investigate it rather than disabling the check. The warnings exist for a reason.
3. **Educate the team**: The most common source of non-determinism is developers who are new to Temporal and don't yet understand the replay model. Pair tooling with documentation.
