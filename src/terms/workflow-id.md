# Workflow ID

The workflow ID is a user-defined identifier for a workflow execution. It must be unique within a namespace at any given time (by default, you cannot start two workflows with the same ID concurrently). This uniqueness constraint enables natural deduplication: starting a workflow with a deterministic ID like `process-order-12345` guarantees at-most-one active execution for that order.

Workflow IDs should be scoped to the business entity they represent -- broad enough to provide deduplication but narrow enough to avoid unnecessary conflicts. The ID reuse policy controls what happens when starting a workflow with an ID that was previously used by a completed or failed workflow.

## Related

- [Not Properly Scoping Semantic Workflow IDs](../not-properly-scoping-semantic-workflow-ids.md)
- [Depending on List Workflow API](../depending-on-list-workflow-api.md)
- [Starting Workflows from Activities](../starting_workflows_from_activities/)
- [Namespace](namespace.md)
- [Idempotency](idempotency.md)
