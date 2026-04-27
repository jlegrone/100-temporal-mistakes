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

##### Draft Outline

Total: ~32 min talk + Q&A. Four mental-model sections, each with 2–3 mistakes that fall out of the same misunderstanding, ending with the design pattern that prevents them. Closing distills the patterns into a small toolkit. No live demos — code on slides.

**Open (~3 min)**
* Why this talk: collected mistakes from years of Datadog Temporal use; speaking as a practitioner, not a vendor.
* Frame: Temporal *is* complicated. The complications exist for good reasons (durability, replay, at-least-once). The way through is a handful of mental models, not a checklist.
* Roadmap: 4 mental models → 4 patterns at the end.

**Part 1 — Replay is not what you think (~7 min)**
* Mental model: workflow code re-executes; activities don't.
* Mistake: Thinking replay means re-running activities. *(sets up the model)*
* Mistake: Performing network calls / using system time in workflow code. *(non-determinism error shown via slide screenshot)*
* Mistake: Not using the return value in a side effect. *(the trap that looks correct)*
* Pattern: keep workflow code deterministic; activities are the boundary. SDK sandboxes / static analysis catch the rest.

**Part 2 — Activities are at-least-once, always (~7 min)**
* Mental model: an activity can run more than once, even with `MaxAttempts=1`.
* Mistake: Not making activities idempotent.
* Mistake: Preventing activity retries — the three timeout misconfigurations. *(`#presentation-include` from the repo)*
* Mistake: Setting too-short timeouts (base on outage tolerance, not happy path).
* Pattern: idempotent by construction; heartbeat + start-to-close + schedule-to-close on every activity.

**Part 3 — Cancellation is cooperative, not a kill (~7 min)**
* Mental model: cancellation is a request the workflow has to handle. Termination and execution-timeout are `kill -9`.
* Mistake: Deadlocking when workflow canceled. *(before/after code on slide — selector fix)*
* Mistake: Not using a disconnected context for cleanup.
* Mistake: Assuming workflow timeouts allow graceful cleanup. *(the surprise that lands well)*
* Pattern: workflows orchestrate cleanup via selector + disconnected context + soft (internal) timeout.

**Part 4 — Scale out, not up (~5 min)**
* Mental model: Temporal scales across many workflows, not within one.
* Mistake: Doing too many things in one workflow / wrapping a queue with a workflow. *(combine into one beat)*
* Mistake: Not using ContinueAsNew (history limits + code age).
* Pattern: fan-out via dedicated child workflows + ContinueAsNew for long-lived state.
* Brief Datadog-scale anecdote here if appropriate (with permission).

**Close — the toolkit (~3 min)**
* Distill the four patterns into a one-slide checklist:
  1. Determinism: keep non-deterministic work in activities; lean on the SDK sandbox/static analysis.
  2. Idempotency + complete activity timeout configuration on every activity.
  3. Cancellation-aware workflows with selector + disconnected context.
  4. Fan out across many small workflows + ContinueAsNew.
* Bonus tools to know about (one slide, no deep dive): replay tests in CI, batch operations API, workflow reset.
* Pointer to the repo for the ~50 mistakes the talk didn't cover.
* Q&A.

**Things explicitly left in Potential (not in talk)**
* Versioning / patching lifecycle (too deep for 30-40 min; mention briefly under "replay tests in CI").
* Most observability mistakes (STSL, sync match rate, autotuning) — operational, niche; cut.
* Most payload mistakes (lossy serialization, multiple inputs, sensitive data) — important but not foundational; cut.
* All "not knowing about X" operational tools beyond the 3 in the bonus slide.
* Custom orchestration frameworks, polling loops, signal-order, semantic IDs, child-workflow start race — defer to repo.

##### Final Outline

**Introduction**

* TBD

**Part One: Activities (~11 min)**

* Mental model: an activity gives the workflow access to the outside world. The Temporal server owns its lifecycle — it can run more than once, it can be canceled, and its timeouts decide how long Temporal keeps trying.
* Mistake — [Not making activities idempotent](src/not-making-activities-idempotent.md).
  * Pattern: idempotency key from workflow ID + input; DB constraints; naturally idempotent ops.
* Mistake — [Preventing activity retries](src/preventing-activity-retries.md) (3 timeout misconfigurations: no heartbeat timeout; no start-to-close; start-to-close == schedule-to-close).
  * Pattern: set heartbeat + start-to-close + schedule-to-close on every activity.
* Mistake — [Setting too-short timeouts](src/setting-too-short-timeouts.md).
  * Pattern: base schedule-to-close on outage tolerance, not happy-path latency.
* Mistake — Activity can't be canceled because it doesn't [heartbeat](src/not_sending_heartbeats_for_cancellation/README.md). Cancellation is delivered through heartbeat responses; without heartbeats it sits on the server until the activity completes on its own.
  * Pattern: set a heartbeat timeout and heartbeat from any cancelable activity; carry progress in [heartbeat details](src/not_using_activity_heartbeat_details/README.md) so a retry resumes instead of restarting.
* Mistake — Calling external services without controlling retry behavior. *(README-only — no entries yet)*
  * Retrying non-retryable errors (auth, validation, "not found").
    * Pattern: decide retryability *from the activity* — return a non-retryable application error.
  * Hammering a struggling downstream during an outage.
    * Pattern: real backoff config; honor downstream backpressure; centralize via interceptor.
* Closing beat: the "well-formed activity" template — carry into Part Two.

**Part Two: Workflows (~10 min)**

* Mental model: workflow code re-executes on [replay](src/thinking-replay-means-rerunning-activities.md), scales across many small workflows rather than within one, and orchestrates cleanup when needed.

* **Avoiding workflow limits**
  * Mistake — [Doing too many things in one workflow](src/doing-too-many-things-in-one-workflow.md) / [wrapping a queue with a workflow](src/wrapping_a_queue_with_a_workflow/README.md). Lock contention, history bloat, replay cost.
    * Pattern: fan out to many small workflows, one per business entity or batch.
  * Mistake — [Not using ContinueAsNew](src/not_using_continue_as_new/README.md). Long-running workflows hit [history-length](src/overflowing-workflow-history-length.md) and [history-size](src/overflowing-workflow-history-bytes.md) limits and accumulate code-version baggage.
    * Pattern: ContinueAsNew on event count, elapsed time, or explicit signal; carry a compact checkpoint as input.
  * Mistake — [Not draining signals before completing the workflow](src/not_draining_signals_before_completing_workflow/README.md). Buffered signals are silently lost on completion or ContinueAsNew.
    * Pattern: drain pending signals before returning; apply them, don't discard.

* **Setting and using timeouts**
  * Mistake — [Not setting a workflow timeout](src/not-setting-a-workflow-timeout.md). Default is 10 years.
    * Pattern: set a generous execution timeout as a backstop; set a run timeout for ContinueAsNew chains.
  * Mistake — [Assuming workflow timeouts allow graceful cleanup](src/assuming_workflow_timeouts_allow_graceful_cleanup/README.md). Execution-timeout terminates; no defers, no handlers.
    * Pattern: implement business deadlines yourself with an internal soft-timeout timer; reserve execution-timeout as a backstop.

* **Handling cancellation and compensating actions**
  * Mistake — [Deadlocking when a workflow is canceled](src/deadlocking_when_workflow_canceled/README.md). Blocking on a signal/timer with no escape.
    * Pattern: selector that waits for the expected event OR `ctx.Done()`.
  * Mistake — [Not using a disconnected context for cleanup](src/not_using_disconnected_context_for_cleanup/README.md). Cleanup activities on a canceled context never dispatch.
    * Pattern: `workflow.NewDisconnectedContext()` for compensation; [wait for child workflows to start](src/not_waiting_for_child_workflows_to_start/README.md) before exiting.
  * Mistake — [Not using ParentClosePolicy](src/not_using_parent_close_policy/README.md). Default `TERMINATE` kills children with no cleanup.
    * Pattern: `REQUEST_CANCEL` when children need cleanup; `ABANDON` when their lifecycle is independent.

* **Dealing with determinism**
  * Mistake — Performing [network calls](src/performing_network_calls_in_workflow_code/README.md) or [using system time](src/using_system_time_instead_of_workflow_time/README.md) in workflow code (also: [env vars](src/reading_environment_variables_in_workflow_code/README.md), [shared state](src/modifying_shared_state_in_workflow_code/README.md)).
    * Pattern: keep all non-determinism in activities; `workflow.Now()` / `workflow.Sleep()` for time.
  * Side effects — [Not using the return value in a `SideEffect`](src/not_using_return_value_in_side_effect/README.md). The function doesn't run on replay; the recorded value does.
    * Pattern: always use the returned value; never rely on closure mutation.
  * Patching / version checks — [Not using workflow versioning](src/not_using_workflow_versioning/README.md) and [incorrect patching](src/incorrect-workflow-patching.md). Replay errors strike running workflows after deploy.
    * Pattern: patch lifecycle (introduce → wait → deprecate → wait → remove); restructure changing logic into activities.

* Closing beat: workflows are deterministic, cooperative, and small — durable orchestration on top of well-formed activities.

**Part Three: Tools & Techniques**

* TBD







# Part Two: Workflows

Need to be robust to:
- Downstream service errors
- Worker crashes & hangs
- Temporal server disruption

Should also:
- Not amplify bad requests
- Gracefully handle cancelation