# Overflowing Workflow History Bytes

> [!TIP]
> Temporal enforces a maximum history size in bytes (50MB by default), separate from the [event count limit](overflowing-workflow-history-length.md). Large activity results, signal payloads, and workflow inputs are the most common contributors to hitting this limit. Workflows that hit this limit are automatically terminated.

Temporal enforces a hard limit on the total byte size of a workflow's history (50MB by default, configurable via [dynamic configuration](terms/dynamic-config.md)). When exceeded, the server [terminates](terms/terminate.md) the workflow with no chance for cleanup. A workflow can hit this limit well before 50k events if individual events carry large [payloads](terms/payload.md) -- tens of activity completions with megabyte-sized results are enough.

<!-- TODO: Add techniques to monitor history size. Link to history size custom search attribute, and check if there's already an out of the box metric coming from the worker SDK or temporal cloud. If not, create an example interceptor that records a metric before the workflow completes. -->
<!-- TODO: Is workflow_history_size_bytes a real metric and is it available from both cloud and self-hosted temporal server? -->
To stay within limits, monitor workflow history size (TODO: add explanation for how to do this). For long-running workflows that accumulate results, use [ContinueAsNew](terms/continue-as-new.md) to start a fresh history, resetting both the event count and history size. Monitor the `workflow_history_size_bytes` metric and set alerts well below the configured limit. If larger payloads are genuinely required for use in workflow code, an [external storage codec](https://docs.temporal.io/external-storage) can offload them to external storage at the serialization layer.

See also: [Overflowing workflow history length](overflowing-workflow-history-length.md), [Passing too much information from activities](passing_too_much_information_from_activities/), [Overflowing maximum individual payload size](overflowing-maximum-individual-payload-size.md).
