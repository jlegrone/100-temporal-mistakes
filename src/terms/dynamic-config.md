# Dynamic Config

Dynamic configuration allows Temporal server operators to change server behavior at runtime without restarting the server. It controls limits (history size, payload size, rate limits), feature flags, and other operational parameters. Values can be scoped globally, per namespace, or per task queue. Dynamic config is typically managed through a YAML file that the server watches for changes.

## Related

- [Overflowing workflow history length](../overflowing-workflow-history-length.md)
- [Overflowing workflow history bytes](../overflowing-workflow-history-bytes.md)
- [Namespace](namespace.md)
- [Temporal Server Backend](temporal-server-backend.md)
