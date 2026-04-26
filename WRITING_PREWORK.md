<draft here, do prework below first>
---

### Prework

*Cobbled together from the exercises in [Better Business Writing](https://www.udemy.com/course/betterbusinesswriting/) from Mark Morris.*

#### Background

Source material is the [100 Temporal Mistakes](https://github.com/jlegrone/100-temporal-mistakes) repo (currently 68 entries across 12 categories), modeled on Teivah's [100 Go Mistakes](https://github.com/teivah/100-go-mistakes). The talk is the conference-stage version of that material — a curated subset, organized around mental models rather than the table of contents.

#### Purposes

* Intended audience: Engineers who have started using Temporal — anywhere from "first week" to "few months in." Assumes basic familiarity with workflows and activities; doesn't re-teach first principles.
* Intended reviewers: Colleagues.
* Forum: Conference stage; recorded for YouTube.
* What do you want this document to communicate?
  * Temporal has real complications. Most of them exist for good reason though!
  * The audience can navigate those complications by recognizing the patterns ahead of time, so they leave with a mental toolkit, not a list of warnings.
* What outcomes do you want from publishing this?
  * Viewers go back to work and can confidently find and fix issues in their own Temporal workloads.
  * The repo becomes the post-talk reference for everything the talk didn't have time to cover.

#### Objective

SMART-ish:

* Specific: Deliver a 30–40 minute conference talk that exposes intermediate Temporal users to the most common mistakes and the design patterns that prevent them.
* Measurable: The audience asks questions and there is significant small group discussion afterward.
* Assignable: Just me; colleagues review.
* Realistic: Yes — the source material exists and only needs curation, narrative, and slides.
* Time-bound: 1 week

#### Reader

Target reader: An engineer who has been using Temporal for somewhere between a few weeks to a few years. Comfortable with the basics (workflows, activities, signals exist) but maybe hasn't yet been bitten by most of the mistakes in this talk.

Who else might read: More experienced Temporal users looking for a refresher or validation; engineers evaluating Temporal who want to understand the failure modes before adopting; people who land on the YouTube recording later via search.

What do we know about the target reader?
They are shipping real workloads under deadline. They may not have a dedicated Temporal platform team. They have read some Temporal docs but learn best from worked examples and direct experience. They came to a conference talk specifically because they want practical, applicable knowledge — not a vendor pitch and not a tutorial.

Why are we writing to them? Is writing the best way to communicate?
A talk is the right medium because the *patterns* matter more than the per-mistake detail — a talk forces curation and narrative. The repo is there for anyone who wants the full reference afterward.

What do they want?
To stop being surprised by Temporal in production. To recognize bad patterns in their own code. To leave with a small number of habits they can apply in their own work.

What will readers' feelings be after reading this? How does that affect what we write?
Goal: empowered, not overwhelmed. They should walk away thinking *"Temporal is complicated for good reasons, and I now know what to look out for"* — not *"Temporal is a minefield."*

Pitfalls to avoid:
- Listing too many mistakes flatly — reads as a minefield.
- Sounding judgmental (the "mistakes" framing of the title already leans this way; avoid judgement everywhere else).
- Giving the impression that using Temporal correctly is hopeless or requires constant vigilance.

#### Voice

Speaking as myself, but also representing collective experience of engineers using Temporal at Datadog. An experienced and knowledgeable end user of Temporal sharing what I've learned the hard way.

How to come across:
- Pragmatic, hands-on, grounded in real experience.
- Confident but not preachy. Each "mistake" should land as *"here's a thing that surprised me / a team I worked with"* — not *"here's what you're doing wrong."*
- The talk closes with a small set of recommendations and design patterns that prevent most of the mistakes shown — leaving the audience with a manageable toolkit, not a checklist of fears.

#### Potential → Outline

(working section: "idea storm" notes start in Potential, migrate towards Outline; be ruthless\!)

##### Potential

###### 1. Mistakes

**Workflow Limits**
* Overflowing workflow history length
* Overflowing workflow history bytes
* Overflowing maximum individual payload size
* Workflow lock contention due to concurrent updates (concurrent activities, concurrent heartbeats, concurrent child workflows, frequent signals/updates/queries)
* Naive batch processing implementation
* Exceeding 10s task timeout

**Workflow Replay & Determinism**
* Not using workflow versioning / patching
* Incorrect workflow patching (removing version branch before safe; using wrong version numbers)
* Thinking replay means re-running activities
* Performing network calls in workflow code
* Using system time instead of workflow time
* Reading environment variables in workflow code
* Modifying shared state in workflow code
* Performing expensive computation in workflow code
* Passing too much information from activities to workflow code
* Lossy payload serialization / deserialization
* Breaking changes to payloads
* Not using the return value in a side effect
* Not using the Temporal SDK for logging, metrics, tracing
* Not using workflow replay for debugging
* Not using static analysis / sandboxed SDK
* Modifying workflow history in interceptors or shared libraries

**Timeouts & Retries**
* Assuming workflow timeouts allow graceful cleanup
* Preventing activity retries (no heartbeat timeout; no start-to-close; start-to-close == schedule-to-close)
* Using workflow retries
* Setting too-short timeouts
* Not setting a workflow timeout
* Retrying after a downstream error that should not be retried *(README-only, no entry yet)*
* Retrying too quickly/frequently on resource-overload errors *(README-only, no entry yet)*

**Cancellation & Shutdown**
* Assuming activity cancellation means workflow cancellation
* Not using ParentClosePolicy
* Deadlocking when workflow canceled
* Not using a disconnected context for cleanup
* Not sending heartbeats from activities you want to cancel
* Not draining activity tasks before graceful worker shutdown

**Software Design**
* Not making activities idempotent
* Not leveraging workflow input/response payloads
* Using more than one input/response payload
* Using a primitive type rather than an object for workflow/activity/signal/update payloads *(README-only, no entry yet)*
* Doing too many things in one workflow
* Over-using activities
* Unnecessary usage of workflows
* Not using activity heartbeat details
* Not using ContinueAsNew
* Not draining signals before completing workflow
* Depending on `ListWorkflow` API outside debugging/operational use cases
* Doing work outside of the workflow
* Wrapping a queue with a workflow
* Unnecessary child workflows
* Custom task orchestration frameworks / abstractions
* Returning both a payload and an error
* Using local activities
* Polling workflow results
* Starting workflows from activities
* Assuming signals/updates will be received in a specific order
* Not properly scoping semantic workflow IDs
* Not waiting for child workflows to start
* Writing polling loops in workflow code
* Querying closed workflows
* Storing sensitive data in workflow history
* Fallible local activities

**Operations**
* Terminating rather than canceling
* Not monitoring schedule-to-start latency (STSL)
* Not monitoring sync match rate
* Not enabling autotuning
* Not knowing about `workflow reset`
* Not knowing about the batch operations API
* Downloading workflow history from UI with "DecodePayloads" enabled
* Not validating replay safety before worker deployments

**Other**
* Starting workflows on the wrong task queue

###### 2. Recommended Design Patterns

**Workflow structure**
* Decompose by concern; one workflow = one entity/process with a clear lifecycle.
* Fan out to individual workflows per message/batch instead of funneling through a single long-running workflow.
* Split batch work into size-limited child workflows; nest "batches of batches" for concurrency control at each level.
* Use child workflows when you genuinely need independent lifecycle (`ParentClosePolicy`) or logical isolation; otherwise inline activities.
* Set `ParentClosePolicy` to `REQUEST_CANCEL` when child needs cleanup; `ABANDON` for genuinely independent lifecycles.
* Use deterministic, semantic workflow IDs (`order-{orderId}`, `tenant-{tenantId}/order-{orderId}`).
* Wait for child workflows to start (`GetChildWorkflowExecution()`) when using a disconnected context.

**Workflow lifetime**
* ContinueAsNew triggered by event count, elapsed time (e.g., 24 h), or explicit signal.
* Carry over only essential state when ContinueAsNew-ing; design input struct as a checkpoint from the start.
* Drain pending signals before completing or ContinueAsNew-ing.
* Always set a workflow execution timeout; set a run timeout for ContinueAsNew chains.
* Implement a soft (internal) timeout via a `workflow.NewTimerWithOptions` deadline so cleanup can run before the hard execution timeout terminates the workflow.

**Activities**
* Always make activities idempotent (idempotency keys derived from workflow ID + input; database upsert / `ON CONFLICT`; naturally idempotent operations).
* Set heartbeat timeout AND start-to-close timeout on every activity; schedule-to-close ≥ start-to-close × max_attempts plus backoff margin.
* Heartbeat from (almost) all activities; heartbeat at least every few seconds for activities running >10 s.
* Use heartbeat details to checkpoint progress so retries resume rather than restart.
* Group related operations into a single activity; use activities at boundaries with external systems.
* Return only what the workflow needs from an activity; pass references (IDs, URLs) for large data.
* Store large payloads externally (S3, DB, blob storage) and pass references through the workflow.
* Local activities only for fast, reliable operations expected to succeed; otherwise regular activity.
* Decide retryability at the activity (`NonRetryableApplicationError`).
* Encode partial results into the result struct; never return both a payload and an error.

**Cancellation**
* Workflows orchestrate cleanup/compensation on cancel; activities just stop work and return.
* Use `workflow.NewDisconnectedContext()` for cleanup activities or child workflows after cancellation.
* Use a selector to wait for the expected event OR cancellation OR timer.
* Test workflow cancellation based on timing of the cancel signal.
* Set `WorkerStopTimeout` aligned with infrastructure grace period (`terminationGracePeriodSeconds`).

**Determinism**
* Move all network I/O, system time access, env-var reads, and randomness into activities or side effects.
* Use `workflow.Now()` and `workflow.Sleep()` / `workflow.NewTimer` instead of language-native equivalents.
* Use `SideEffect` only for capturing small non-deterministic *values*; always use the returned value.
* Use the SDK's replay-aware logger, metrics handler, and OpenTelemetry interceptors.
* Restructure logic that needs to change frequently into activity code (no versioning concerns).

**Versioning & rollout**
* Use `workflow.GetVersion` / `patched()` for any change to workflow code.
* Patching lifecycle: introduce → wait for completion → deprecate (remove old branch, keep marker) → wait → remove patch.
* Worker Versioning + pinned workflows as an alternative.
* Replay tests in CI/CD with periodically captured representative histories; block deploys on failure.

**Signals & updates**
* Design state machines that tolerate any delivery order.
* Use Update validators (e.g., monotonically increasing ULIDs) to reject out-of-order messages at the API boundary.
* Drain signal channels before completion or ContinueAsNew; apply, don't discard.
* Push state changes via signals instead of polling loops in workflow code.

**Payloads**
* Wrap inputs in a single struct/object; return a single result object.
* Treat payload types like a database schema or public API: add optional fields with defaults; don't rename or remove.
* Use schema-friendly serialization (Protocol Buffers) where possible.
* Run round-trip serialization tests; replay tests catch incompatible payload changes.
* Pass references (IDs, vault tokens) instead of sensitive data; layer on a custom payload codec for encryption / external storage.

**Operations**
* Cancel before terminate; reserve termination for last-resort cases.
* Use the batch operations API (`temporal batch terminate / signal / reset`) with a visibility query.
* Use `workflow reset` to recover from bad code paths after deploying a fix.
* Custom search attributes for queryable state instead of querying closed workflows.
* Memo / typed return values for state you'd otherwise read via a query on a closed workflow.
* Always download workflow history with "DecodePayloads" disabled for replay/reset.

**Querying & visibility**
* Derive workflow IDs deterministically from domain data; interact by ID rather than via `ListWorkflow`.
* Use search attributes / external databases for application reads.

**Observability**
* SDK logger, metrics handler, and tracing interceptors instead of direct library calls in workflow code.
* Monitor STSL (`temporal_activity_schedule_to_start_latency`, `temporal_workflow_task_schedule_to_start_latency`).
* Monitor sync match rate (`temporal_matching_sync_match`) — leading indicator of capacity issues.
* Monitor `workflow_history_size_bytes`, `persistence_latency`, `persistence_errors`, `service_errors_resource_exhausted`.

###### 3. Automation

**Static analysis & sandboxes**
* TypeScript: V8 isolate sandbox (default).
* Python / .NET: runtime sandbox detecting non-deterministic imports.
* Go: `workflowcheck` analyzer (sdk-go/contrib/tools/workflowcheck) — flags `time.Now()`, `rand`, raw `go` keyword.
* TODO(jlegrone): Go linter for Temporal workflow determinism checks (open in repo).
* TODO: linter requiring `SideEffect` return value to be read.
* TODO: linter / runtime check requiring `ParentClosePolicy` to be set explicitly on every child workflow.

**CI / replay testing**
* Replay tests in CI/CD with `WorkflowReplayer` against captured histories.
* Automated/scheduled collection of representative histories from production for use as test fixtures.
* Run with very small sticky cache size in dev to surface replay issues earlier.
* Property-based tests for signal/update order invariance (e.g., `rapid` library in Go).
* Cancellation timing tests with `TestWorkflowEnvironment` + `RegisterDelayedCallback`.
* Round-trip serialization tests for payload types.

**Interceptors (worker-side enforcement)**
* Auto-heartbeating interceptor for long activities (paired with workflow-side interceptor that sets a heartbeat timeout).
* Interceptor enforcing schedule-to-close + heartbeat timeouts on every activity (hard error in dev, warn in prod); also prevent `MaxAttempts=1`.
* Interceptor that sets a default `ParentClosePolicy` (e.g., `REQUEST_CANCEL`) or rejects child workflows missing one.
* Interceptor monitoring unhandled signals via `GetUnhandledSignalNames` before workflow completion.
* Interceptor adding a query result to memo before workflow completion (closed-workflow read alternative).
* Interceptor + custom codec emitting metrics for workflow history size and payload size.
* Idempotency-test interceptor (TODO).
* Interceptor flagging local activities with complex retry policies or long timeouts.

**Data converter / codec**
* Custom data converter using Protocol Buffers for safer serialization.
* Encryption codec for sensitive payloads.
* External-storage codec / Large Payload Codec for offloading oversized data to blob storage.

**CLI / operational tools**
* `temporal batch terminate / signal / reset` driven by visibility queries.
* `temporal workflow reset` (including `--reset-type` variants and bad-binary reset).
* Worker Versioning (server feature) + pinned workflows.
* Resource-based worker tuning (`worker.NewResourceBasedTuner`).
* `WorkerStopTimeout` for graceful drain on shutdown.
* Eager workflow start to reduce start-up latency in special cases.

**Tooling gaps mentioned in repo TODOs**
* Timeout and retry policy lint/simulator.
* CLI tool to check whether any workflows are still active on a given patch version (so the old branch is safe to remove).

##### Outline

* \<move things from "potential", but *leave some behind\!*\>
