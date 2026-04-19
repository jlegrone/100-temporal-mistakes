# 100 Temporal Mistakes

A collection of common mistakes made when using [Temporal](https://temporal.io) and how to avoid them. This is a work in progress!

## Workflow Limits
- [Overflowing workflow history length](src/overflowing-workflow-history-size.md)
- [Overflowing workflow history bytes](src/overflowing-workflow-history-bytes.md)
- [Overflowing maximum individual payload size](src/overflowing-maximum-individual-payload-size.md)
- [Shard contention due to concurrent updates](src/workflow-lock-contention-due-to-concurrent-updates.md)
	- Concurrent activities
	- Concurrent heartbeats
    - Concurrent child workflows
	- Frequent signals, updates, queries
	- [Naive batch processing implementations](src/naive-batch-processing-implementation.md)
- Exceeding 10s task timeout

## Workflow Replay
- Not using workflow versioning/patching
- Incorrect workflow patching (wrong version numbers, removing branch before it's safe)
- Thinking that replay means re-running activities
- Performing network calls in workflow code
- Using system time instead of workflow time
- Reading environment variables in workflow code
- Modifying shared state in workflow code
- Performing expensive computation in workflow code
- Passing too much information from activities to workflow code
- Lossy payload serialization/deserialization
- Breaking changes to payloads
- Not using the return value in a side effect
- Not using the Temporal SDK for logging, metrics, tracing
- Not using workflow replay for debugging
- Not using static analysis / sandboxed SDK
- Modifying workflow history in interceptors or shared libraries

## Timeouts and Retries
- Assuming workflow timeouts allow graceful cleanup (treated like termination, not cancellation)
- Preventing activity retries
	- Not setting activity heartbeat timeout
	- Not setting start to close activity timeout
	- Setting start to close and schedule to close timeout to same value
- Using workflow retries
- Setting too-short workflow or activity timeouts (keep long enough to survive downstream failures)
- Not setting a workflow timeout (they'll run for 10 years!)

## Cancellation
- Assuming activity cancellation means workflow cancellation
- Not using ParentClosePolicy when graceful cleanup on cancellation is needed in child workflows
- Deadlocking when workflow cancelled (handle cancel signal)
- Not using a disconnected context to perform cleanup or other deferred child workflows/activities after workflow cancelled
- Not sending heartbeats from activities you want to handle cancellation.

## Software Design
- Not making activities idempotent (at least once execution semantic, even with max attempts == 1)
- Not leveraging workflow input/response payloads (not all workflow engines support these!)
- Using more than one input/response payload (only supported in Go, Java SDKs?)
- Doing too many things in one workflow (scale out rather than scale up) (shard contention)
- Over-using activities (do more in one activity without checkpointing)
- Unnecessary usage of workflows (maybe you don't need the durability for a CRUD API)?
- Not using activity heartbeat details
- Not using ContinueAsNew (limit your maximum code age!)
- Not draining signals before completing workflow
- Depending on `ListWorkflow` API outside of debugging/operational use cases
- Doing work outside of workflow (eg. before starting, or before sending a signal) -- move into the workflow for more durability
- Wrapping a queue with a workflow (unless you have a good reason!)
- Unnecessary child workflows
- Custom task orchestration frameworks/abstractions (you should probably use an existing declarative workflow engine if Temporal's programming model doesn't suit your needs)
- Returning both a payload and an error value
- Using local activities
	- Don't exceed 10s task timeout
- Polling workflow results
	- Just use child workflows, or the SDK client from outside a worker!
- Starting workflows from activities
- Assuming signals/updates will be received in a specific order
- Not properly scoping semantic workflow IDs
- Not waiting for child workflows to start before exiting when using disconnected context
- Writing polling loops in workflow code
    - [Use an activity, or (rarely) a child workflow + ContinueAsNew](https://community.temporal.io/t/long-polling-inside-workflows-or-activities/11348/2)
- Querying closed workflows
- Storing sensitive data in workflow history
- Fallible local activities
    - https://youtu.be/b2AnXkqCwgw?feature=shared&t=429

## Testing

## Operations
- Terminating rather than canceling
- Not monitoring STSL
- Not monitoring sync match rate
- Not enabling autotuning (still preview feature though)
- Not knowing about `workflow reset`
- Not knowing about batch operations API
- Underutilizing namespaces
- Not draining activity tasks before graceful worker shutdown
- Downloading workflow history from UI with "DecodePayloads" option enabled
- Not validating replay safety before worker deployments
- Not setting up TLS
- Not enabling ringpop TLS
- Not setting up persistence rate limits
- Not setting up namespaces rate limits

## Other
- Starting workflows on the wrong task queue
- Not understanding why you're using Temporal
- Not investing in a Temporal platform team (?)
- Using the Temporal UI use cases other than debugging (eg. as the main interface for end users)

---

<sub>[100 Temporal Mistakes and How to Avoid Them](https://github.com/jlegrone/100-temporal-mistakes) by [Jacob LeGrone](https://jacoblegrone.com) is licensed under [CC BY-NC 4.0](https://creativecommons.org/licenses/by-nc/4.0/?ref=chooser-v1)</sub>
