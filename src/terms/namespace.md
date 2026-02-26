# Namespace

A namespace is a unit of isolation within a Temporal cluster. Each namespace has its own set of workflows, visibility data, rate limits, and access controls. Workflows in different namespaces cannot interact directly (no cross-namespace signals or child workflows).

Namespaces enable multi-tenancy, environment separation (dev/staging/prod), and blast radius containment. Per-namespace rate limits prevent one team's workload from starving others, and separate visibility stores allow independent querying. Temporal Cloud provides namespaces as the primary unit of provisioning.

## Related

- [Underutilizing Namespaces](../underutilizing-namespaces.md)
- [Not Setting Up Namespaces Rate Limits](../not-setting-up-namespaces-rate-limits.md)
- [Not Properly Scoping Semantic Workflow IDs](../not-properly-scoping-semantic-workflow-ids.md)
- [Dynamic Config](dynamic-config.md)
- [Visibility](visibility.md)
- [Temporal Server Backend](temporal-server-backend.md)
