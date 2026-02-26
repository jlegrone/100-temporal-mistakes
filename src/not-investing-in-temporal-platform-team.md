# Not Investing in a Temporal Platform Team

> [!TIP]
> * As Temporal adoption grows across an organization, a dedicated platform team prevents fragmentation and operational drift.
> * The platform team owns shared infrastructure, establishes best practices, provides SDKs and templates, and handles server upgrades.
> * Without one, every team reinvents the wheel and operational quality varies wildly.

## What?

When Temporal goes from "one team's tool" to "multiple teams building on it," the operational and knowledge burden grows non-linearly. Each team needs to understand server operations, monitoring, SDK best practices, [versioning](terms/versioning.md), and failure modes. Without a dedicated platform team, each product team figures this out independently -- often making the same mistakes along the way.

A Temporal platform team is a small group that owns the shared Temporal infrastructure and serves as the center of expertise for the organization. They don't write your workflows for you, but they make sure the foundation is solid and the path to production is well-paved.

## Why?

Temporal is infrastructure. Like databases, message queues, or container orchestration, it needs dedicated ownership once it reaches a certain scale. Here's what goes wrong without it:

**Inconsistent operational quality.** One team sets up proper alerting on [Schedule-To-Start latency](terms/schedule-to-start-latency.md) and [workflow lock contention](<workflow-lock-contention-due-to-concurrent-updates.md>), another team doesn't monitor anything. When the second team's workflows start silently [failing due to wrong task queues](<starting-workflows-on-wrong-task-queue.md>), nobody notices until customers complain.

**Duplicated effort.** Every team builds their own wrapper libraries, their own deployment patterns, their own monitoring dashboards. Five teams each spend a week solving the same problem is five weeks of wasted engineering time.

**Risky upgrades.** Temporal server upgrades, schema migrations, and [dynamic configuration](<terms/dynamic-config.md>) changes affect all teams. Without a team that owns this process, upgrades either don't happen (leading to version drift) or happen without coordination (leading to outages).

**Knowledge silos.** Temporal expertise stays locked within individual teams. When someone leaves, the knowledge leaves with them. New teams starting with Temporal have no one to learn from and repeat all the early mistakes.

## How?

**Start small.** A Temporal platform team doesn't need to be large. Even one or two engineers who own the server infrastructure and establish patterns can make a significant difference. Scale the team as adoption grows.

**Define clear responsibilities.** The platform team should own:
- Temporal server operations (deployment, upgrades, scaling, monitoring)
- Shared client libraries and templates that encode best practices
- Documentation and onboarding guides for product teams
- Consultation and review of new workflow designs
- [Namespace](terms/namespace.md) management and multi-tenancy configuration

**Provide golden paths.** Create starter templates that include proper monitoring, error handling, and [versioning](terms/versioning.md) patterns out of the box. If you make the right thing easy, teams will do the right thing.

**Don't gatekeep.** The platform team should accelerate product teams, not block them. Provide self-service tooling for common operations like creating namespaces, deploying [workers](terms/worker.md), and viewing metrics. Reserve reviews for architecture decisions, not day-to-day workflow development.

**Establish feedback loops.** Regularly collect pain points from product teams. The patterns they struggle with are the ones the platform team should solve once and share with everyone.
