# Underutilizing Namespaces

> [!TIP]
> * [Namespaces](terms/namespace.md) provide isolation boundaries in Temporal: separate rate limits, separate [visibility](terms/visibility.md), and separate access controls.
> * Many teams put everything in the "default" namespace, losing all the benefits of isolation.
> * Use separate namespaces for different environments, teams, or reliability tiers.

## What?

A Temporal namespace is a logical isolation unit. Each namespace has its own workflow visibility, its own rate limits, its own retention policies, and its own access controls. Workflows in different namespaces are completely invisible to each other.

Despite this, many teams run all of their workflows in a single "default" namespace. Everything -- dev experiments, staging tests, production workflows from different teams with wildly different reliability requirements -- ends up in one big bucket. This is the Temporal equivalent of running all your microservices in one Kubernetes namespace with no resource quotas.

## Why?

Cramming everything into one namespace creates several problems:

- **No rate limit isolation**: One team's runaway workflow can consume the namespace's rate limit budget, starving other teams' workflows. If team A triggers a batch operation that creates 10,000 workflows per second, team B's latency-sensitive payment workflows get throttled.
- **Noisy visibility**: The workflow list becomes a mess. Finding your team's workflows among thousands of others requires careful [search attribute](terms/search-attributes.md) discipline that most teams don't have.
- **No access control granularity**: Everyone with access to the namespace can see and operate on everyone else's workflows. An operator trying to cancel a test workflow could accidentally target production workflows if the search query is too broad.
- **Blast radius**: Configuration changes (retention period, rate limits, archival settings) apply to the entire namespace. Changing retention from 30 days to 7 days because one team doesn't need the history affects all teams.
- **Operational confusion**: During incidents, it's harder to isolate the impact and triage when all workflows are mixed together.

## How?

1. **Separate by environment**: At minimum, use different namespaces for development, staging, and production. This prevents test workflows from interfering with production and gives you different retention and rate limit settings per environment.

2. **Separate by team or domain**: Give each team or business domain its own production namespace. This provides rate limit isolation, cleaner visibility, and independent access controls. For example: `payments-prod`, `notifications-prod`, `analytics-prod`.

3. **Separate by reliability tier**: If some workflows are latency-critical while others are best-effort batch jobs, put them in different namespaces. This lets you set tighter rate limits and monitoring for critical workflows without the noise from batch operations.

4. **Use naming conventions**: Adopt a consistent naming scheme like `{team}-{environment}` or `{domain}-{tier}` so namespaces are self-documenting and easy to manage at scale.

5. **Automate namespace provisioning**: Don't make creating a new namespace a heavyweight process. Use infrastructure-as-code to manage namespace creation, configuration, and access controls so that teams can get a new namespace without filing a ticket and waiting a week.

6. **Be aware of the tradeoffs**: More namespaces mean more operational overhead (monitoring, alerting, access management). Find the right balance for your organization -- the goal is meaningful isolation, not a namespace for every workflow type.
