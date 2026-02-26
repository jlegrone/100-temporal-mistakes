# Not Making Activities Idempotent

> [!TIP]
> * Temporal has at-least-once execution semantics for activities -- even with max attempts set to 1, infrastructure failures can cause an activity to execute more than once.
> * Activities must be designed to produce the same result when run multiple times with the same input.
> * Use [idempotency](terms/idempotency.md) keys, check-then-act patterns, or database constraints to enforce idempotency.

## What?

A common misconception is that setting `MaximumAttempts` to 1 on an activity [retry policy](terms/retry-policy.md) guarantees that the activity will only run once. This is not the case. Temporal provides at-least-once execution semantics for activities, not exactly-once.

Consider this scenario: a [worker](terms/worker.md) picks up an activity task, executes the activity successfully (e.g. charges a credit card), but crashes before it can report the result back to the Temporal server. The server, unaware that the activity completed, may schedule the activity for execution again on another worker. The result: the credit card is charged twice.

This can also happen during network partitions, worker deployments, or any situation where the acknowledgment of a completed activity is lost.

## Why?

The consequences of non-idempotent activities range from annoying to catastrophic depending on the domain:
- **Financial operations**: duplicate charges, duplicate refunds, or duplicate transfers.
- **Messaging**: duplicate emails, SMS, or push notifications sent to users.
- **State mutations**: records created or updated twice leading to inconsistent state.
- **External API calls**: side effects triggered multiple times in third-party systems.

Because the duplicate execution can happen at any time due to infrastructure issues, it is not something you can reliably reproduce in development or testing environments. It will happen in production, and when it does, you need your activities to handle it gracefully.

## How?

There are several strategies to make activities idempotent:

**Idempotency keys**: Generate a unique key for each logical operation (e.g. derived from the [workflow ID](terms/workflow-id.md) and activity input) and pass it to the downstream system. Payment providers, for instance, typically accept an idempotency key and will return the result of the original request if the same key is sent again.

**Check-then-act**: Before performing the operation, check whether it has already been done. For example, before inserting a record, check if a record with the same unique identifier already exists. Be aware that this approach is subject to race conditions unless combined with proper locking or database constraints.

**Database constraints**: Use unique constraints or conditional writes (e.g. `INSERT ... ON CONFLICT DO NOTHING`, `PutItem` with a condition expression in DynamoDB) to ensure that duplicate operations are rejected at the storage level.

**Natural idempotency**: Some operations are inherently idempotent. Setting a value to a specific state (as opposed to incrementing it), or upserting a record with a fixed ID, will produce the same result regardless of how many times they run.

The best approach depends on your use case, but the key takeaway is: always assume your activity can run more than once, and design accordingly.
