# Not Setting Up Namespace Rate Limits
<!-- TODO: Delete this mistake from the repo. -->

> [!TIP]
> Without per-namespace rate limits, one [namespace](terms/namespace.md)'s traffic can starve all others. Configure namespace-level rate limits via [dynamic configuration](terms/dynamic-config.md) to ensure fair resource allocation, especially in multi-tenant setups.

Temporal supports multiple namespaces on a single cluster, letting teams or applications share infrastructure while maintaining logical isolation. Without rate limits, however, this isolation is only logical, not physical. A namespace generating a massive burst of workflow starts, [signals](terms/signals.md), or [queries](terms/queries.md) can consume all available server and database capacity, starving every other namespace on the cluster. Even in single-tenant setups with multiple namespaces, rate limits prevent one service from impacting others during traffic spikes or failure scenarios.

Configure namespace rate limits via Temporal's [dynamic configuration](terms/dynamic-config.md). The key settings are `frontend.globalNamespaceRPS` (cluster-wide frontend request rate limit per namespace, distributed across all frontend hosts) and `frontend.maxNamespaceVisibilityRPSPerInstance` (rate limit for [visibility](terms/visibility.md) operations):

```yaml
# Default for all namespaces
frontend.globalNamespaceRPS:
  - value: 2000
    constraints: {}

# Override for a high-throughput namespace
frontend.globalNamespaceRPS:
  - value: 5000
    constraints:
      namespace: "high-throughput-namespace"
```

Set a reasonable default for all namespaces so new namespaces are automatically rate-limited. Grant higher limits to namespaces that need them after verifying cluster capacity. Monitor `service_errors_resource_exhausted` metrics by namespace to see which namespaces are hitting their limits.

See also: [Not Setting Up Persistence Rate Limits](not-setting-up-persistence-rate-limits.md) for the complementary protection layer, [Underutilizing Namespaces](underutilizing-namespaces.md).
