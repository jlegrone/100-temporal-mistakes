# Not Setting Up Persistence Rate Limits
<!-- TODO: Delete this mistake from the repo. -->

> [!TIP]
> Without persistence rate limits, a burst of workflow traffic can overwhelm your database, causing cascading failures across all [namespaces](terms/namespace.md). Configure persistence rate limits via [dynamic configuration](terms/dynamic-config.md) to protect your [Temporal server backend](terms/temporal-server-backend.md).

Temporal server relies heavily on its persistence layer (Cassandra, MySQL, or PostgreSQL) for storing [workflow histories](terms/event-history.md), managing [task queues](terms/task-queue.md), and maintaining cluster state. Every workflow start, activity completion, [heartbeat](terms/heartbeat.md), [signal](terms/signals.md), and [query](terms/queries.md) generates database operations. By default, Temporal does not impose aggressive rate limits on these operations. When the persistence layer is overloaded, latency increases across the board, timeouts cascade as retries add more load, history service shards can get stuck, and recovery is slow even after the spike subsides. Rate limits turn a potential system-wide outage into localized throttling -- affected requests receive a `ResourceExhausted` error and the caller backs off, while the rest of the system continues normally.

Configure persistence rate limits via [dynamic configuration](terms/dynamic-config.md). The most impactful setting is `history.persistenceMaxQPS`, since the history service generates the most database traffic:

```yaml
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

Start with conservative limits based on what your database can sustainably handle, leaving headroom for spikes. Monitor `persistence_latency` and `persistence_errors` metrics to detect when limits are too tight (excessive throttling) or too loose (allowing database overload).

See also: [Not Setting Up Namespace Rate Limits](not-setting-up-namespaces-rate-limits.md) for per-namespace isolation that prevents one namespace from consuming all available capacity.
