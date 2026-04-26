# Underutilizing Namespaces
<!-- TODO: Delete this mistake from the repo. -->

> [!TIP]
> [Namespaces](terms/namespace.md) provide isolation boundaries for rate limits, [visibility](terms/visibility.md), and access controls. Many teams put everything in the "default" namespace, losing all the benefits of isolation.

A Temporal namespace is a logical isolation unit with its own workflow visibility, rate limits, retention policies, and access controls. Workflows in different namespaces are completely invisible to each other. Despite this, many teams run all their workflows in a single "default" namespace -- dev experiments, staging tests, and production workflows from different teams with wildly different reliability requirements all in one bucket. This creates the noisy neighbor problem (one team's runaway workflow starves others), a cluttered workflow list that requires careful [search attribute](terms/search-attributes.md) discipline to navigate, no access control granularity, and unbounded blast radius from configuration changes.

At minimum, use different namespaces for development, staging, and production. Beyond that, give each team or business domain its own production namespace (e.g., `payments-prod`, `notifications-prod`, `analytics-prod`) for rate limit isolation, cleaner visibility, and independent access controls. If some workflows are latency-critical while others are best-effort batch jobs, separate them by reliability tier. Adopt a consistent naming convention like `{team}-{environment}` so namespaces are self-documenting, and automate namespace provisioning with infrastructure-as-code so teams can get a new namespace without filing a ticket.

Be aware of the tradeoff: more namespaces mean more operational overhead for monitoring, alerting, and access management. The goal is meaningful isolation, not a namespace for every workflow type.

See also: [Not Setting Up Namespace Rate Limits](not-setting-up-namespaces-rate-limits.md), [Not Setting Up Persistence Rate Limits](not-setting-up-persistence-rate-limits.md).
