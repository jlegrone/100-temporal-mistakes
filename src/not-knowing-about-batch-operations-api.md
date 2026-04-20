# Not Knowing About the Batch Operations API

> [!TIP]
> Temporal's batch operations API can terminate, [cancel](terms/cancelation.md), [signal](terms/signals.md), or reset many workflows matching a [visibility](terms/visibility.md) query in a single request. Use it instead of writing scripts that iterate over workflows one by one.

The batch operations API lets you execute an operation across a large number of workflows at once. You provide a visibility query (the same query language used in the Temporal UI's workflow list) and an operation type, and Temporal applies that operation to every matching workflow. Without this API, teams end up writing custom scripts that list workflows, iterate over them, and apply operations one at a time -- scripts that are slow, error-prone, and often don't handle rate limiting or partial failures well.

Batch operations solve real operational problems at scale. During incident response, when thousands of workflows are stuck due to a bug, doing individual operations is painfully slow. Batch operations run server-side with proper rate limiting built in, track progress as a single logical action, and provide an audit record of what was done and why.

```bash
# Cancel all workflows of a specific type
temporal batch terminate \
  --query 'WorkflowType = "OrderProcessing" AND ExecutionStatus = "Running"' \
  --reason "Canceling due to upstream service outage"

# Signal all running workflows
temporal batch signal \
  --query 'ExecutionStatus = "Running"' \
  --signal-name "force-refresh" \
  --reason "Pushing config update to all running workflows"
```

The power of batch operations depends on the quality of your visibility query. Use [search attributes](terms/search-attributes.md) to tag workflows with metadata (team, environment, version) so you can target exactly the right set. Always test your query first to verify it matches only the expected workflows -- a too-broad query on a destructive operation like terminate causes significant damage.

See also: [Not Knowing About Workflow Reset](not-knowing-about-workflow-reset.md).
