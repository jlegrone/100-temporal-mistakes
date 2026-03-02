# Not Using Workflow Replay for Debugging

> [!TIP]
> * The Temporal SDK can replay a single workflow's [history](terms/event-history.md) locally, without connecting to a server, letting you step through the exact execution in a debugger.
> * You can download a workflow's history from the Temporal CLI or UI and replay it on your machine.
> * This is often the fastest way to reproduce and diagnose workflow bugs, yet many teams don't know the feature exists.

## What?

Temporal's [replay](terms/replay.md) mechanism is not just an internal runtime detail -- it's a debugging tool. Every Temporal SDK provides an API to replay a workflow execution from a history file (or history object). You point it at a downloaded history, and the SDK re-executes your workflow code step by step, exactly as it originally ran.

This means you can:

- Set breakpoints in your IDE and step through the workflow logic.
- Inspect the state at any point during execution.
- Reproduce production bugs locally without needing the original environment, database state, or timing conditions.

## Why?

Debugging workflow code through logs alone is painful. Workflows can run for hours, days, or weeks, interacting with dozens of activities and [child workflows](terms/child-workflow.md). Reproducing the exact sequence of events that led to a bug in a test environment is often impractical.

Replay-based debugging sidesteps all of that. The workflow history *is* the reproduction case -- it contains every activity result, [signal](terms/signals.md), timer, and decision point the workflow encountered. Replaying it locally gives you a deterministic reproduction every time.

Without this technique, teams often resort to:

- Adding more logging and redeploying, hoping to catch the issue next time.
- Trying to manually reconstruct the sequence of events.
- Guessing at the root cause based on incomplete information.

All of which are slower and less reliable than just replaying the history.

## How?

### Step 1: Download the workflow history

Using the Temporal CLI:

```bash
temporal workflow show \
  --workflow-id your-workflow-id \
  --run-id your-run-id \
  --output json > history.json
```

You can also download the history from the Temporal Web UI as a JSON file.

### Step 2: Replay it locally

In Go:

```go
func TestReplayWorkflow(t *testing.T) {
    replayer := worker.NewWorkflowReplayer()
    replayer.RegisterWorkflow(YourWorkflow)
    err := replayer.ReplayWorkflowHistoryFromJSONFile(nil, "history.json")
    require.NoError(t, err)
}
```

In TypeScript:

```typescript
import { Worker } from '@temporalio/worker';

const history = await JSON.parse(fs.readFileSync('history.json', 'utf8'));
await Worker.runReplayHistory(
  { workflowsPath: require.resolve('./workflows') },
  history
);
```

In Python:

```python
from temporalio.worker import Replayer

async def test_replay():
    replayer = Replayer(workflows=[YourWorkflow])
    await replayer.replay_workflow(history)
```

### Step 3: Debug

Set breakpoints in your workflow code and run the replay test under your debugger. The workflow will execute exactly as it did in production, stopping at your breakpoints so you can inspect state.

### Bonus: Replay tests as regression tests

Once you've captured a history that exposed a bug, keep it as a test fixture. Replay tests make excellent regression tests -- they verify that your current code can correctly process histories produced by previous versions, catching [non-determinism](terms/non-determinism.md) errors before deployment.
