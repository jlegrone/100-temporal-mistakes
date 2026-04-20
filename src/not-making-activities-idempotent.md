# Not Making Activities Idempotent

> [!TIP]
> Temporal provides at-least-once execution semantics for activities -- even with max attempts set to 1, infrastructure failures can cause an activity to execute more than once. Design activities to produce the same result when run multiple times with the same input.

Setting `MaximumAttempts` to 1 on a [retry policy](terms/retry-policy.md) does not guarantee an activity runs only once. A [worker](terms/worker.md) can complete an activity (e.g. charge a credit card) then crash before reporting the result. The server, unaware of success, schedules the activity again on another worker. This also happens during network partitions and deployments.

The consequences depend on the domain: duplicate charges, duplicate notifications, records created twice, or side effects triggered multiple times in third-party systems. Because duplicate execution can happen at any time due to infrastructure issues, you cannot reliably reproduce it in development.

Several strategies help:
- **[Idempotency](terms/idempotency.md) keys**: Derive a unique key from the [workflow ID](terms/workflow-id.md) and activity input; pass it to downstream systems (payment providers typically accept these).
- **Database constraints**: Use `INSERT ... ON CONFLICT DO NOTHING` or conditional writes to reject duplicates at the storage level.
- **Natural idempotency**: Setting a value to a specific state (vs. incrementing) or upserting with a fixed ID is inherently idempotent.

Always assume your activity can run more than once, and design accordingly.
