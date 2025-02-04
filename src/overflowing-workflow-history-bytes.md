# Overflowing Maximum Workflow History Bytes

> [!TIP]
> - Workflows histories record many events during the execution of a workflow including all the inputs and outputs of activities and child workflows, signals payloads, ...
> - Workflow history size (in terms of bytes utilization) are capped at 50MB by default. When that limit is crossed, workflows are terminated without giving a chance to cleanup after themselves.

## What?
Alongside the [number of events a workflow history can hold](overflowing-workflow-history-size.md), a second limit can easily be reached, the individual history bytes size.

Every input and output of workflows, child workflows and activities are being serialized (using a [data converter](<terms/data-converter.md>) and stored in the [temporal server backend](<terms/temporal-server-backend.md>) inside a workflow history. Signals payloads are also recorded in the workflow history. Other history events (timers, workflow tasks, ...) also need a couple of bytes of storage ultimately.

While [each individual payload for an activity or workflow input / output are capped](overflowing-maximum-individual-payload-size.md), this doesn't prevent total history bytes size to grow big when you sum the size of all events an history contains.

By default, maximum history size is set to 50MB. If an history grow larger, the server will immediately [terminate](terminate.md) the workflow without giving it a chance to perform any cleanup operation.

Temporal servers will also log warning when it sees an history crossing the warning size limit which is by default set to 10MB. The limit can be tweaked via the `limit.historySize.warning` server [dynamic-config](dynamic-config.md).
## Why?
Temporal data model relies heavily on [replay](terms/replay.md) which is not a free operation as it involves downloading workflow histories over the network and re-running workflow code many times during the life of a workflow, especially when in memory caches are not hot. So large histories take more time to replay than small ones. 10ms additional replay delay may compound quickly when you run millions of workflows and you have to replay all of them at once.
## Solutions
To workaround the issue, it is recommended to:
1. Trim down your inputs and outputs to the minimum first
2. Store the payloads in an external system (a database, blob storage, disk if you use [sessions](terms/sessions.md) , …) and pass references (IDs, links, …) to as inputs, outputs or signal payload.
3. Fan out work on multiple smaller workflows as each will have their own limit
4. Look at the [large payload codec](<terms/large-payload-codec.md>) (which offloads activities and workflows inputs / outputs to blob storage automatically).
5. Tweak the `limit.historySize.error` server [dynamic-config](dynamic-config.md).
