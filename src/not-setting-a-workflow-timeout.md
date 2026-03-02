# Not Setting a Workflow Timeout

> [!TIP]
> * Without an [execution timeout](terms/workflow-execution-timeout.md), a workflow runs until it completes, fails, or is manually [terminated](terms/terminate.md) -- the default is effectively 10 years.
> * A bug that prevents a workflow from completing results in stuck workflows consuming resources indefinitely with no automatic safety net.
> * Always set a reasonable execution timeout as a backstop, even if you expect the workflow to complete well before it.

## What?

When starting a workflow, you can set a `WorkflowExecutionTimeout` that limits how long the entire workflow execution (including all retries and [continue-as-new](terms/continue-as-new.md) chains) can run. If you don't set one, the default is effectively 10 years. The workflow runs until it completes on its own, fails terminally, or someone manually terminates it.

Many teams skip this configuration because their workflows "should" complete in a known timeframe. The workflow processes an order, runs some ETL, or handles a user request -- it'll finish in minutes or hours. Why bother with a timeout?

## Why?

The problem isn't your workflow under normal conditions. The problem is your workflow under abnormal conditions -- and those are precisely the conditions you need to plan for.

Here are some ways a workflow can get stuck without a timeout:

**Bugs in workflow logic.** A conditional branch that never triggers, a loop that never exits, a [signal](terms/signals.md) wait that never receives its signal. Your workflow sits there forever, occupying space in the server's persistence layer and holding external resources.

**Abandoned workflows.** A workflow started for a user action that was later cancelled out of band. Nobody sends the signal or makes the API call to terminate it. Without a timeout, it lingers indefinitely.

**Resource accumulation.** One stuck workflow is manageable. A hundred are annoying. Ten thousand -- which happens quickly if the bug is in a high-volume workflow -- start impacting server performance. Each stuck workflow has [history](terms/event-history.md) that must be retained, and in aggregate this puts pressure on your [temporal server backend](terms/temporal-server-backend.md).

The execution timeout acts as a safety net. It doesn't replace proper workflow design, but it ensures that no workflow runs forever even if everything else fails. When the timeout fires, the workflow is [terminated](terms/terminate.md) and shows up clearly in your monitoring as timed out.

## How?

**Set an execution timeout on every workflow.** Pick a duration that is generous enough to handle the worst case (including downstream outages and retries) but bounded enough to catch genuinely stuck workflows. If your workflow normally completes in 10 minutes and you're okay waiting through a 1-hour outage, set the timeout to 2 hours. See also [setting too-short timeouts](setting-too-short-timeouts.md) -- the goal is a reasonable upper bound, not a tight deadline.

**Use workflow run timeout for continue-as-new chains.** If your workflow uses [continue-as-new](terms/continue-as-new.md), the execution timeout applies to the entire chain. The [run timeout](terms/workflow-run-timeout.md) applies to each individual run. Set both: run timeout to catch a single stuck run, execution timeout to catch an endlessly looping chain.

**Monitor for timeout terminations.** A workflow hitting its execution timeout is a sign that something went wrong. Set up alerts for workflows that end with a `Timeout` status so you can investigate and fix the root cause rather than silently relying on the safety net.
