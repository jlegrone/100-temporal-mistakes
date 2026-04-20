# Idempotency

Idempotency means that performing an operation multiple times produces the same result as performing it once. In Temporal, activity idempotency is critical because activities have at-least-once execution semantics -- even with `MaximumAttempts` set to 1, infrastructure failures (worker crashes, network partitions) can cause an activity to execute more than once.

Common strategies for achieving idempotency include: using idempotency keys (unique tokens that prevent duplicate processing), check-then-act patterns (verify the operation hasn't already been done), database constraints (unique indexes that reject duplicates), and naturally idempotent operations (reads, upserts).

## Related

- [Not Making Activities Idempotent](../not-making-activities-idempotent.md)
- [Doing Work Outside of Workflow](../doing_work_outside_of_the_workflow/)
- [Not Knowing About Workflow Reset](../not-knowing-about-workflow-reset.md)
- [Starting Workflows from Activities](../starting_workflows_from_activities/)
- [Activity Task](activity-task.md)
- [Retry Policy](retry-policy.md)
