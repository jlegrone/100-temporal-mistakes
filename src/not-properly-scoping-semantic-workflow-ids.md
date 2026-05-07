# Not Properly Scoping Semantic Workflow IDs

<!-- TODO: Reword the TIP. The point is to avoid collisions in workflow ids chosen by different workers/workflow types & workflow executions. -->
> [!TIP]
> [Workflow IDs](terms/workflow-id.md) serve as both unique identifiers and [idempotency](terms/idempotency.md) keys. Scope them to the business entity they represent (e.g., `order-{orderId}`) to get natural deduplication.

IDs that are too broad (e.g., `"process-order"` for all orders) cause unintended conflicts -- only one order can be processed at a time. IDs that are too narrow or random (UUIDs) throw away Temporal's built-in deduplication, so retried starts create duplicate workflows. Poorly scoped IDs also make it harder to find workflows in the UI.

Design IDs around the business entity: `user-{userId}`, `order-{orderId}`, `tenant-{tenantId}/order-{orderId}`. The key principle: if two callers independently start the same logical operation, they should derive the same workflow ID. When you need multiple executions over time, include the distinguishing dimension: `daily-report-{userId}-{date}`.

<!-- TODO: Add before and after v1/v2 client code example where the v2 version correctly prefixes the customer ID with workflow type to ensure uniqueness across multiple workflow IDs of different types that might relate to the same customer. -->
