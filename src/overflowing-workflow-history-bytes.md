# Overflowing Workflow History Bytes

> [!TIP]
> * Temporal enforces a maximum history size in bytes, separate from the [event count limit](<overflowing-workflow-history-size.md>). Exceeding it causes the workflow to be [terminated](terms/terminate.md).
> * Large activity results, signal payloads, and workflow inputs are the most common culprits for hitting this limit.
> * Solution: minimize payload sizes, offload large data to external storage, and use [ContinueAsNew](terms/continue-as-new.md) to keep histories bounded.

## What?

In addition to the [50k event count limit](<overflowing-workflow-history-size.md>), Temporal enforces a separate hard limit on the total byte size of a workflow's history. By default, this limit is 50MB (configurable via [dynamic configuration](<terms/dynamic-config.md>)). When a workflow's history exceeds this byte-size threshold, the server [terminates](terms/terminate.md) it -- just like with the event count limit, there is no chance for cleanup.

A workflow can be terminated well before reaching 50k events if its individual events carry large [payloads](terms/payload.md). A workflow with only a few hundred activity completions can hit the byte limit if each result contains megabytes of serialized data.

## Why?

Developers often focus solely on the event count limit and overlook the byte-size limit. A workflow that processes modest numbers of activities might seem safe from the 50k event cap, but if those activities return large results (images, documents, serialized datasets, etc.), the cumulative history size in bytes grows quickly.

During [replay](terms/replay.md), the entire history must be fetched from the [server backend](<terms/temporal-server-backend.md>) and deserialized by the [worker](terms/worker.md). Large histories in bytes mean:

- Higher network bandwidth consumption between workers and the Temporal server.
- Longer replay times, directly affecting [workflow task](terms/workflow-task.md) processing latency.
- Increased memory pressure on workers, which must hold the full history in memory during replay.

Unlike the event count limit which you can estimate by counting scheduled operations, the byte-size limit depends on the actual data flowing through your workflow, making it harder to predict at design time.

## Solution

1. **Minimize payload sizes.** Audit what your activities return and what your [signals](terms/signals.md) carry. Return only the data the workflow actually needs to make decisions. If an activity produces a large result that is only needed by a subsequent activity, store it externally and pass a reference (ID, URL, S3 key, etc.) instead.

2. **Use external storage for large data.** Databases, blob stores, or shared file systems are better suited for moving large data between activities than Temporal's history. Treat the workflow as an orchestration layer -- it should coordinate work, not be a data pipeline.

3. **Use [ContinueAsNew](terms/continue-as-new.md).** For long-running workflows that accumulate results over time, periodically invoke ContinueAsNew to start a fresh history. This resets both the event count and the byte-size counters.

4. **Monitor history sizes.** Keep an eye on the `workflow_history_size_bytes` metric and set alerts well below the configured limit. Catching the trend early gives you time to refactor before workflows start getting terminated.

5. **Consider the [large payload codec](<terms/large-payload-codec.md>).** If only a fraction of your payloads are oversized, this codec can transparently offload them to external storage at the serialization layer.
