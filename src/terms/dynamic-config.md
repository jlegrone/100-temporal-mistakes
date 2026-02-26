# Dynamic Config

Dynamic configuration allows Temporal server operators to change server behavior at runtime without restarting the server. It controls limits (history size, payload size, rate limits), feature flags, and other operational parameters. Values can be scoped globally, per namespace, or per task queue. Dynamic config is typically managed through a YAML file that the server watches for changes.

## Related

- [Overflowing workflow history size](../overflowing-workflow-history-size.md)
- [Overflowing workflow history bytes](../overflowing-workflow-history-bytes.md)
- [Not setting up persistence rate limits](../not-setting-up-persistence-rate-limits.md)
- [Not setting up namespaces rate limits](../not-setting-up-namespaces-rate-limits.md)
- [Namespace](namespace.md)
- [Temporal Server Backend](temporal-server-backend.md)
