# Not Using the Temporal SDK for Observability

<!-- TODO: include metrics & logs in the example code. Add a v1 (bad) and v2 version, and include an Example unit test for the go worker that performs an assertion on the log output of the worker. Also note which log fields and metric tags are missing if you don't use the SDK logger and metrics interfaces. Also note that you have the ability to customize the logger and metrics adapters in the client or worker options. -->

> [!TIP]
> Standard logging, metrics, and tracing libraries run on every [replay](../terms/replay.md), producing duplicate and misleading output. Use the SDK's replay-aware alternatives in workflow code instead.

Developers often reach for familiar observability tools inside workflow code -- `log.Info()`, a Prometheus counter, a Datadog span. These work fine during initial execution, but workflow code also runs during [replay](../terms/replay.md). When a [worker](../terms/worker.md) restarts or a workflow is evicted from cache, the SDK replays the [workflow history](../terms/event-history.md) to reconstruct state, and every log statement, metric increment, and trace span fires again. The result is duplicated logs that make debugging harder, inflated metrics that misrepresent actual activity, and noisy traces that obscure real issues.

Every Temporal SDK provides replay-aware alternatives. Use the SDK's workflow logger, which automatically skips emission during replay:

<!--SNIPSTART not-using-temporal-sdk-for-observability-good-->
[not_using_temporal_sdk_for_observability/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_using_temporal_sdk_for_observability/workflow.go)
```go

// Good: use the SDK's replay-aware workflow logger.
func MyWorkflow(ctx workflow.Context, orderID string) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Processing order", "orderID", orderID)
	return nil
}

```
<!--SNIPEND-->

```typescript
import { log } from '@temporalio/workflow';
log.info('Processing order', { orderID });
```

```python
from temporalio import workflow
workflow.logger.info("Processing order", extra={"orderID": order_id})
```

For metrics, configure a metrics handler on the client options and use `workflow.GetMetricsHandler(ctx)` inside workflows -- the handler is replay-aware. For tracing, use the OpenTelemetry [interceptors](../terms/interceptor.md) provided by the SDK, which create spans correctly linked to the workflow execution and suppressed during replay. Note that this applies to workflow code only -- activity code runs outside the replay mechanism and can safely use any logging, metrics, or tracing library directly.
