# Not Knowing About the Batch Operations API

> [!TIP]
> * Temporal provides a batch operations API that can perform operations (terminate, [cancel](terms/cancellation.md), [signal](terms/signals.md), reset) on many workflows matching a [visibility](terms/visibility.md) query.
> * Use batch operations instead of writing scripts that iterate over workflows one by one -- it's more efficient and handles rate limiting for you.
> * Available via `tctl` and the SDK APIs, this is an essential tool for operational incident response.

## What?

The batch operations API lets you execute an operation across a large number of workflows in a single request. You provide a visibility query (the same query language used in the Temporal UI's workflow list) and an operation type, and Temporal applies that operation to every workflow matching the query.

Supported operations include:
- **Cancel**: cooperatively cancel matching workflows.
- **Terminate**: forcibly [terminate](terms/terminate.md) matching workflows.
- **Signal**: send a signal to matching workflows.
- **Reset**: [reset](not-knowing-about-workflow-reset.md) matching workflows to a specific point.

Without knowing about this API, teams end up writing custom scripts that list workflows, iterate over them, and apply operations one at a time. These scripts are slow, error-prone, and often don't handle rate limiting or partial failures well.

## Why?

Batch operations solve real operational problems:

- **Incident response at scale**: A bug causes thousands of workflows to get stuck. You need to cancel or reset all of them. Doing this one by one is painfully slow. Batch operations handle the entire set efficiently.
- **Rate limiting**: The Temporal server enforces rate limits on API calls. A naive script blasting individual requests will quickly hit these limits and either fail or need complex retry logic. Batch operations are handled server-side with proper rate limiting built in.
- **Atomicity of intent**: A batch operation is a single logical action. You can track its progress, and it won't leave you in a state where half the workflows were processed and the other half weren't because your script crashed.
- **Auditability**: Batch operations are tracked by the server and visible in the system, giving you a record of what was done and why.

## How?

1. **Via the CLI**:
   ```bash
   # Cancel all workflows of a specific type that are running
   tctl batch terminate \
     --query 'WorkflowType = "OrderProcessing" AND ExecutionStatus = "Running"' \
     --reason "Canceling due to upstream service outage"

   # Signal all running workflows in a namespace
   tctl batch signal \
     --query 'ExecutionStatus = "Running"' \
     --signal-name "force-refresh" \
     --reason "Pushing config update to all running workflows"

   # Reset workflows affected by a bad deployment
   tctl batch reset \
     --query 'WorkflowType = "PaymentFlow" AND ExecutionStatus = "Running"' \
     --reset-type BadBinary \
     --reason "Resetting workflows affected by v2.3.0 bug"
   ```

2. **Via the SDK**: All Temporal SDKs expose batch operation methods on the workflow client. Use these when you need to integrate batch operations into your operational tooling or automation.

3. **Craft precise visibility queries**. The power of batch operations depends on the quality of your visibility query. Use [search attributes](terms/search-attributes.md) to tag workflows with metadata (team, environment, feature flag, version) so you can target exactly the right set of workflows.

4. **Test your query first**. Before running a batch operation, run the visibility query alone to verify it matches the expected set of workflows. A too-broad query applied to a destructive operation like terminate can cause significant damage.
