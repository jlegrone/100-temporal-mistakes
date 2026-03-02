# Not Properly Scoping Semantic Workflow IDs

> [!TIP]
> * Temporal uses [workflow IDs](terms/workflow-id.md) for deduplication -- you can't start two workflows with the same ID simultaneously by default.
> * IDs that are too broad cause unexpected conflicts; IDs that are too narrow or random lose the deduplication benefit.
> * Scope workflow IDs to the business entity they represent (e.g., `process-order-{orderId}`).

## What?

Temporal workflow IDs serve a dual purpose: they uniquely identify a workflow execution and provide an [idempotency](terms/idempotency.md) mechanism. By default, starting a workflow with an ID that already has a running execution fails with a `WorkflowExecutionAlreadyStarted` error.

The mistake is choosing workflow IDs that don't align with your business semantics -- either too broad, too narrow, or entirely random.

## Why?

**IDs that are too broad** cause unintended conflicts. If you use `"process-order"` as your workflow ID for all order processing, only one order can be processed at a time. The second order submission fails because the first workflow is still running.

**IDs that are too narrow or random** (e.g., UUIDs) throw away Temporal's built-in deduplication. If a client retries starting a workflow due to a transient error and uses a different random ID each time, you get duplicate workflows processing the same business operation. You then need to build your own deduplication mechanism on top.

**Poorly scoped IDs** also make it harder to find and manage workflows in the Temporal UI or via the API. A workflow ID like `process-order-12345` tells you exactly what it's doing and for which entity. A UUID tells you nothing.

## How?

Design your workflow IDs around the business entity or operation they represent:

- **One workflow per entity:** `user-{userId}`, `order-{orderId}`, `subscription-{subscriptionId}`
- **One workflow per operation on an entity:** `charge-order-{orderId}`, `ship-order-{orderId}`
- **Namespaced IDs for multi-tenant systems:** `tenant-{tenantId}/order-{orderId}`

The key principle: if two callers independently decide to start the same logical operation, they should derive the same workflow ID. This gives you natural deduplication without any extra effort.

When you need to allow multiple executions for the same entity over time (e.g., a daily report for user X), include the distinguishing dimension in the ID: `daily-report-{userId}-{date}`.

If you genuinely need multiple concurrent workflows for the same entity, consider whether a single workflow with [child workflows](terms/child-workflow.md) or activities would be a better fit. If not, you can use the `WorkflowIDReusePolicy` and `WorkflowIDConflictPolicy` options to control the behavior, but think carefully about whether your ID scheme is the real problem first.
