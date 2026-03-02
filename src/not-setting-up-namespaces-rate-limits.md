# Not Setting Up Namespace Rate Limits

> [!TIP]
> * Without per-namespace rate limits, one [namespace](terms/namespace.md)'s traffic can starve all others, which is especially dangerous in multi-tenant setups.
> * Configure namespace-level rate limits via [dynamic configuration](terms/dynamic-config.md) to ensure fair resource allocation across teams and workloads.
> * Namespace rate limits complement persistence rate limits by providing isolation between tenants rather than just protecting the database.

## What?

Temporal supports multiple namespaces on a single cluster, letting teams or applications share infrastructure while maintaining logical isolation. Without rate limits, however, this isolation is only logical, not physical. A namespace generating a massive burst of workflow starts, [signals](terms/signals.md), or [queries](terms/queries.md) can consume all available server and database capacity, starving every other namespace on the cluster.

By default, Temporal's namespace rate limits are either unset or set very high. Any namespace can use as much capacity as it wants, and the only backstop is the global persistence rate limit (if configured).

## Why?

In multi-tenant environments, resource isolation is critical. Without namespace rate limits:

- **Noisy neighbor problem.** One team running a load test or deploying a bug that starts workflows in a tight loop can bring down workflows for every other team on the cluster.
- **No predictable performance.** Teams cannot reason about their workflows' performance because it depends on what every other namespace is doing simultaneously.
- **Incident blast radius is unbounded.** A problem in one namespace becomes a cluster-wide incident affecting all namespaces.

Even in single-tenant setups with multiple namespaces (e.g., separate namespaces for different services), rate limits prevent one service from impacting others during traffic spikes or failure scenarios.

## How?

Configure namespace rate limits via Temporal's [dynamic configuration](terms/dynamic-config.md). Set them globally as defaults and override per namespace.

Key dynamic configuration values:

- **`frontend.namespaceRPS`** -- Maximum frontend requests per second for a given namespace, per frontend host. This limits the rate of API calls (start workflow, signal, query, etc.) that a namespace can make.
- **`frontend.globalNamespaceRPS`** -- Cluster-wide frontend request rate limit for a namespace, distributed across all frontend hosts. This is typically the more useful setting as it doesn't depend on the number of frontend hosts.
- **`frontend.maxNamespaceVisibilityRPSPerInstance`** -- Rate limit for [visibility](terms/visibility.md) (list/count workflows) operations per namespace.
- **`frontend.maxNamespaceVisibilityBurstRatioPerInstance`** -- Burst ratio for visibility operations allowing short spikes above the sustained rate.

```yaml
# Example dynamic configuration
# Default for all namespaces
frontend.globalNamespaceRPS:
  - value: 2000
    constraints: {}

# Override for a high-throughput namespace
frontend.globalNamespaceRPS:
  - value: 5000
    constraints:
      namespace: "high-throughput-namespace"

# Override for a low-priority namespace
frontend.globalNamespaceRPS:
  - value: 500
    constraints:
      namespace: "low-priority-namespace"
```

**Best practices:**

- **Set a reasonable default for all namespaces.** This ensures any new namespace is automatically rate-limited without manual intervention.
- **Grant higher limits to namespaces that need them.** Teams can request higher limits based on their expected traffic, and you can increase limits after verifying the cluster has capacity.
- **Monitor `service_errors_resource_exhausted` metrics by namespace.** This tells you which namespaces are hitting their limits and whether limits need to be adjusted.
- **Combine with persistence rate limits.** Namespace rate limits control how many requests reach the server, while [persistence rate limits](not-setting-up-persistence-rate-limits.md) protect the database from the total combined load. Both layers are needed for robust protection.
