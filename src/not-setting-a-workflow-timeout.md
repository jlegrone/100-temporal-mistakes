# Not Setting a Workflow Timeout

> [!TIP]
> Without an [execution timeout](terms/workflow-execution-timeout.md), workflows default to running for up to 10 years. A bug that prevents completion results in stuck workflows consuming resources indefinitely.

Many teams skip configuring `WorkflowExecutionTimeout` because their workflows "should" complete in a known timeframe. But a conditional branch that never triggers, a [signal](terms/signals.md) wait that never fires, or an abandoned workflow will sit forever, occupying persistence and holding external resources. One stuck workflow is manageable; ten thousand -- which happens quickly with a bug in a high-volume workflow -- start impacting server performance.

Set an execution timeout on every workflow. Pick a duration generous enough for the worst case (including downstream outages and retries) but bounded enough to catch genuinely stuck workflows. If your workflow normally completes in 10 minutes and you're willing to wait through a 1-hour outage, set the timeout to 2 hours. For [continue-as-new](terms/continue-as-new.md) chains, also set a [run timeout](terms/workflow-run-timeout.md) to catch a single stuck run. Monitor for timeout [terminations](terms/terminate.md) -- they're a sign something went wrong.

See also: [Setting Too-Short Timeouts](setting-too-short-timeouts.md).
