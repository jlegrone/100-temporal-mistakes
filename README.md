# 100 Temporal Mistakes

A collection of common mistakes made when using [Temporal](https://temporal.io) and how to avoid them. This is a work in progress!

## Workflow Limits
- [Overflowing workflow history length](src/overflowing-workflow-history-length.md)
- [Overflowing workflow history bytes](src/overflowing-workflow-history-bytes.md)
- [Overflowing maximum individual payload size](src/overflowing-maximum-individual-payload-size.md)
- [Shard contention due to concurrent updates](src/workflow-lock-contention-due-to-concurrent-updates.md)
	- Concurrent activities
	- Concurrent heartbeats
    - Concurrent child workflows
	- Frequent signals, updates, queries
	- [Naive batch processing implementations](src/naive-batch-processing-implementation.md)
- [Exceeding 10s task timeout](src/exceeding-10s-task-timeout.md)

## Workflow Replay
- [Not using workflow versioning/patching](src/not_using_workflow_versioning/)
- [Incorrect workflow patching (wrong version numbers, removing branch before it's safe)](src/incorrect-workflow-patching.md)
- [Thinking that replay means re-running activities](src/thinking-replay-means-rerunning-activities.md)
- [Performing network calls in workflow code](src/performing_network_calls_in_workflow_code/)
- [Using system time instead of workflow time](src/using_system_time_instead_of_workflow_time/)
- [Reading environment variables in workflow code](src/reading_environment_variables_in_workflow_code/)
- [Modifying shared state in workflow code](src/modifying_shared_state_in_workflow_code/)
- [Performing expensive computation in workflow code](src/performing_expensive_computation_in_workflow_code/)
- [Passing too much information from activities to workflow code](src/passing_too_much_information_from_activities/)
- [Lossy payload serialization/deserialization](src/lossy-payload-serialization.md)
- [Breaking changes to payloads](src/breaking-changes-to-payloads.md)
- [Not using the return value in a side effect](src/not_using_return_value_in_side_effect/)
- [Not using the Temporal SDK for logging, metrics, tracing](src/not_using_temporal_sdk_for_observability/)
- [Not using workflow replay for debugging](src/not_using_workflow_replay_for_debugging/)
- [Not using static analysis / sandboxed SDK](src/not-using-static-analysis-sandboxed-sdk.md)
- [Modifying workflow history in interceptors or shared libraries](src/modifying-workflow-history-in-interceptors.md)

## Timeouts and Retries
- [Assuming workflow timeouts allow graceful cleanup (treated like termination, not cancellation)](src/assuming_workflow_timeouts_allow_graceful_cleanup/)
- [Preventing activity retries](src/preventing-activity-retries.md)
	- Not setting activity heartbeat timeout
	- Not setting start to close activity timeout
	- Setting start to close and schedule to close timeout to same value
- [Using workflow retries](src/using-workflow-retries.md)
- [Setting too-short workflow or activity timeouts (keep long enough to survive downstream failures)](src/setting-too-short-timeouts.md)
- [Not setting a workflow timeout (they'll run for 10 years!)](src/not-setting-a-workflow-timeout.md)

## Cancellation
- [Assuming activity cancellation means workflow cancellation](src/assuming_activity_cancelation_means_workflow_cancelation/)
- [Not using ParentClosePolicy when graceful cleanup on cancellation is needed in child workflows](src/not_using_parent_close_policy/)
- [Deadlocking when workflow canceled (handle cancel signal)](src/deadlocking_when_workflow_canceled/)
- [Not using a disconnected context to perform cleanup or other deferred child workflows/activities after workflow canceled](src/not_using_disconnected_context_for_cleanup/)
- [Not sending heartbeats from activities you want to handle cancellation](src/not_sending_heartbeats_for_cancellation/)

## Software Design
- [Not making activities idempotent (at least once execution semantic, even with max attempts == 1)](src/not-making-activities-idempotent.md)
- [Not leveraging workflow input/response payloads (not all workflow engines support these!)](src/not_leveraging_workflow_input_response_payloads/)
- [Using more than one input/response payload (only supported in Go, Java SDKs?)](src/using_more_than_one_input_response_payload/)
- [Doing too many things in one workflow (scale out rather than scale up) (shard contention)](src/doing-too-many-things-in-one-workflow.md)
- [Over-using activities (do more in one activity without checkpointing)](src/over-using-activities.md)
- [Unnecessary usage of workflows (maybe you don't need the durability for a CRUD API)?](src/unnecessary-usage-of-workflows.md)
- [Not using activity heartbeat details](src/not_using_activity_heartbeat_details/)
- [Not using ContinueAsNew (limit your maximum code age!)](src/not_using_continue_as_new/)
- [Not draining signals before completing workflow](src/not_draining_signals_before_completing_workflow/)
- [Depending on `ListWorkflow` API outside of debugging/operational use cases](src/depending-on-list-workflow-api.md)
- [Doing work outside of workflow (eg. before starting, or before sending a signal) -- move into the workflow for more durability](src/doing_work_outside_of_the_workflow/)
- [Wrapping a queue with a workflow (unless you have a good reason!)](src/wrapping_a_queue_with_a_workflow/)
- [Unnecessary child workflows](src/unnecessary-child-workflows.md)
- [Custom task orchestration frameworks/abstractions (you should probably use an existing declarative workflow engine if Temporal's programming model doesn't suit your needs)](src/custom-task-orchestration-frameworks.md)
- [Returning both a payload and an error value](src/returning_both_payload_and_error/)
- [Using local activities](src/using_local_activities/)
	- Don't exceed 10s task timeout
- [Polling workflow results](src/polling_workflow_results/)
	- Just use child workflows, or the SDK client from outside a worker!
- [Starting workflows from activities](src/starting_workflows_from_activities/)
- [Assuming signals/updates will be received in a specific order](src/assuming_signal_update_order/)
- [Not properly scoping semantic workflow IDs](src/not-properly-scoping-semantic-workflow-ids.md)
- [Not waiting for child workflows to start before exiting when using disconnected context](src/not_waiting_for_child_workflows_to_start/)
- [Writing polling loops in workflow code](src/writing_polling_loops_in_workflow_code/)
    - [Use an activity, or (rarely) a child workflow + ContinueAsNew](https://community.temporal.io/t/long-polling-inside-workflows-or-activities/11348/2)
- [Querying closed workflows](src/querying-closed-workflows.md)
- [Storing sensitive data in workflow history](src/storing_sensitive_data_in_workflow_history/)
- [Fallible local activities](src/fallible_local_activities/)
    - https://youtu.be/b2AnXkqCwgw?feature=shared&t=429

## Testing

## Operations
- [Terminating rather than canceling](src/terminating-rather-than-canceling.md)
- [Not monitoring STSL](src/not-monitoring-stsl.md)
- [Not monitoring sync match rate](src/not-monitoring-sync-match-rate.md)
- [Not enabling autotuning (still preview feature though)](src/not_enabling_autotuning/)
- [Not knowing about `workflow reset`](src/not-knowing-about-workflow-reset.md)
- [Not knowing about batch operations API](src/not-knowing-about-batch-operations-api.md)
- [Underutilizing namespaces](src/underutilizing-namespaces.md)
- [Not draining activity tasks before graceful worker shutdown](src/not_draining_activity_tasks_before_shutdown/)
- [Downloading workflow history from UI with "DecodePayloads" option enabled](src/downloading-history-with-decode-payloads.md)
- [Not validating replay safety before worker deployments](src/not_validating_replay_safety_before_deployments/)
- [Not setting up TLS](src/not_setting_up_tls/)
- [Not enabling ringpop TLS](src/not-enabling-ringpop-tls.md)
- [Not setting up persistence rate limits](src/not-setting-up-persistence-rate-limits.md)
- [Not setting up namespaces rate limits](src/not-setting-up-namespaces-rate-limits.md)

## Other
- [Starting workflows on the wrong task queue](src/starting-workflows-on-wrong-task-queue.md)
- [Not understanding why you're using Temporal](src/not-understanding-why-youre-using-temporal.md)
- [Not investing in a Temporal platform team (?)](src/not-investing-in-temporal-platform-team.md)
- [Using the Temporal UI use cases other than debugging (eg. as the main interface for end users)](src/using-temporal-ui-for-non-debugging.md)

---

<sub>[100 Temporal Mistakes and How to Avoid Them](https://github.com/jlegrone/100-temporal-mistakes) by [Jacob LeGrone](https://jacoblegrone.com) is licensed under [CC BY-NC 4.0](https://creativecommons.org/licenses/by-nc/4.0/?ref=chooser-v1)</sub>
