# Not Setting Up Persistence Rate Limits

> [!TIP]
> * Without persistence rate limits, a burst of workflow traffic or a misbehaving workflow can overwhelm your database, causing cascading failures across all [namespaces](terms/namespace.md).
> * Configure persistence rate limits via [dynamic configuration](terms/dynamic-config.md) to protect your [Temporal server backend](terms/temporal-server-backend.md).
> * Rate limits act as a safety valve: they cause individual requests to be throttled rather than letting the entire system degrade.

## What?

Temporal server relies heavily on its persistence layer (Cassandra, MySQL, or PostgreSQL) for storing [workflow histories](terms/event-history.md), managing [task queues](terms/task-queue.md), and maintaining cluster state. Every workflow start, activity completion, [heartbeat](terms/heartbeat.md), [signal](terms/signals.md), and [query](terms/queries.md) generates database operations.

By default, Temporal does not impose aggressive rate limits on persistence operations. The server issues as many database requests as the workload demands, trusting that the database can handle it. That trust is misplaced. A sudden spike in workflow starts, a workflow that signals thousands of other workflows in a tight loop, or even normal growth can push the database past its capacity.

## Why?

When the persistence layer is overloaded:

- **Latency increases across the board.** All workflows slow down, not just the ones causing the spike. A single misbehaving namespace can degrade the entire cluster.
- **Timeouts cascade.** Database operations start timing out, triggering retries that add more load, creating a vicious cycle.
- **History service shards can get stuck.** If the database is too slow to respond, history shards may stop making progress, causing workflows to appear frozen.
- **Recovery is slow.** Even after the spike subsides, the database takes time to recover as it processes the backlog. The system remains degraded throughout.

Rate limits turn a potential system-wide outage into localized throttling. When a rate limit is hit, affected requests receive a `ResourceExhausted` error and the caller (SDK or internal service) backs off and retries. The rest of the system continues normally.

## How?

Configure persistence rate limits via Temporal's [dynamic configuration](terms/dynamic-config.md). The key settings control the maximum persistence requests per second at different granularities.

Key dynamic configuration values to set:

- **`history.persistenceMaxQPS`** -- Maximum persistence requests per second for the history service per host. This is the most impactful setting as the history service generates the most database traffic.
- **`history.persistenceGlobalMaxQPS`** -- Global (cluster-wide) persistence rate limit for the history service, distributed across all hosts.
- **`matching.persistenceMaxQPS`** -- Maximum persistence requests per second for the matching service per host.
- **`matching.persistenceGlobalMaxQPS`** -- Global persistence rate limit for the matching service.
- **`frontend.persistenceMaxQPS`** -- Maximum persistence requests per second for the frontend service per host.

**Start with conservative limits and adjust based on monitoring.** Set limits based on what your database can sustainably handle, leaving headroom for spikes. Monitor `persistence_latency` and `persistence_errors` metrics to detect when limits are too tight (causing excessive throttling) or too loose (allowing database overload).

```yaml
# Example dynamic configuration
history.persistenceMaxQPS:
  - value: 3000
    constraints: {}
matching.persistenceMaxQPS:
  - value: 3000
    constraints: {}
frontend.persistenceMaxQPS:
  - value: 3000
    constraints: {}
```

**Combine with namespace rate limits.** Persistence rate limits protect the database globally, but they don't prevent one namespace from consuming all the available capacity. See [Not setting up namespace rate limits](not-setting-up-namespaces-rate-limits.md) for the complementary protection layer.
