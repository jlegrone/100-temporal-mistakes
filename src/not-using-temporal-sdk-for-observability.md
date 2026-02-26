# Not Using the Temporal SDK for Observability

> [!TIP]
> * Standard logging, metrics, and tracing libraries run on every [replay](terms/replay.md), producing duplicate and misleading output.
> * The Temporal SDK provides replay-aware logging and metrics that are automatically suppressed during replay.
> * Use `workflow.GetLogger()`, the SDK's metrics handler, and OpenTelemetry integrations instead of direct library calls in workflow code.

## What?

Developers often reach for familiar observability tools inside workflow code -- `log.Info()`, a Prometheus counter, a Datadog span. This works fine during initial execution, but workflow code also runs during [replay](terms/replay.md). Every log statement, every metric increment, and every trace span fires again each time the workflow is replayed.

The result: duplicated logs that make debugging harder, inflated metrics that misrepresent actual activity, and noisy traces that obscure real issues.

## Why?

When a [worker](terms/worker.md) restarts or a workflow is evicted from cache, the SDK replays the [workflow history](terms/event-history.md) to reconstruct its state. During replay, your workflow code re-executes from the beginning up to the point where new work needs to happen. Any observability calls embedded in that code path fire again.

Consider a workflow that processes 100 items and logs each one. After a single worker restart, you now have 200 log entries for 100 items. After two restarts, 300. With millions of workflows, this noise becomes a real operational problem:

- **Logs**: Searching for a specific event becomes difficult when every message has duplicates. Log volume (and cost) grows with replay frequency rather than actual business events.
- **Metrics**: Counters are incremented on replay, making dashboards and alerts unreliable. A spike in your "orders processed" metric might just mean workers restarted, not that you actually processed more orders.
- **Traces**: Replay-generated spans pollute your tracing backend and make it harder to follow the real execution flow.

## How?

Every Temporal SDK provides replay-aware alternatives. Use them instead of direct library calls inside workflow code.

### Logging

Use the SDK's workflow logger, which automatically skips log emission during replay:

```go
// Go
logger := workflow.GetLogger(ctx)
logger.Info("Processing order", "orderID", orderID)
```

```typescript
// TypeScript
import { log } from '@temporalio/workflow';
log.info('Processing order', { orderID });
```

```python
# Python
from temporalio import workflow
workflow.logger.info("Processing order", extra={"orderID": order_id})
```

### Metrics

Use the SDK's metrics handler. In Go, configure a metrics handler on the client options and use `workflow.GetMetricsHandler(ctx)` inside workflows. The handler is replay-aware.

### Tracing

Use the OpenTelemetry [interceptors](terms/interceptor.md) provided by the SDK. These create spans that are correctly linked to the workflow execution and are suppressed during replay.

### Important caveat

This applies to **workflow code only**. Activity code runs outside the replay mechanism and can safely use any logging, metrics, or tracing library directly. The distinction is important -- don't over-correct by wrapping activity observability in SDK-specific calls when it's unnecessary.
