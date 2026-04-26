# Review Tracker

All 68 mistakes have been condensed to the target format (TLDR + 1-3 paragraphs, no section headers). One duplicate was merged (#6 into #12). Net reduction: ~2,264 lines.

## Approved

(pending user review of rendered markdown)

## Revisit

- [ ] **#3 Building Custom Task Orchestration Frameworks** ([`custom-task-orchestration-frameworks.md`](src/custom-task-orchestration-frameworks.md))
  - [ ] Verify DSL samples link is still valid (`temporalio/samples-go/tree/main/dsl`)

- [ ] **#5 Thinking Replay Means Re-Running Activities** ([`thinking-replay-means-rerunning-activities.md`](src/thinking-replay-means-rerunning-activities.md))
  - [ ] Verify "speculative execution" is still accurate terminology

- [x] **Assuming Activity Cancellation Means Workflow Cancellation** ([`assuming-activity-cancelation-means-workflow-cancelation`](src/assuming_activity_cancelation_means_workflow_cancelation/))
  - [x] ~~Replace code example~~ — now shows an activity that incorrectly updates DB status on cancellation instead of letting the workflow orchestrate cleanup

- [ ] **#12 Not Using Static Analysis or the Sandboxed SDK** ([`not-using-static-analysis-sandboxed-sdk.md`](src/not-using-static-analysis-sandboxed-sdk.md))
  - [ ] Link to specific Go linter for Temporal determinism checks if one exists
  - [x] Added TODO comment for Go linter tool

## Completed

All 68 entries condensed:
- Category 1: Adoption and Fundamentals (3, after deleting self-hosting/platform entry)
- Category 2: Workflow Determinism and Replay (8)
- Category 3: Activity Design and Lifecycle (7)
- Category 4: Timeouts and Retries (6)
- Category 5: Workflow Design Patterns (11)
- Category 6: Signals, Updates, and Workflow Interaction (5)
- Category 7: Cancellation and Shutdown (7, after merging duplicate)
- Category 8: Payloads and Serialization (7)
- Category 9: Workflow History Limits (3)
- Category 10: Versioning and Deployments (3)
- Category 11: Observability and Debugging (6, after deleting Temporal UI entry)
- Category 12: Infrastructure and Security (2, after deleting 4 self-hosting entries)
- Category 13: Organization and Scale (0, after deleting platform team entry)

Deleted: `not-sending-heartbeats-from-activities-you-want-to-handle-cancellation.md` (duplicate of `not-sending-heartbeats-for-cancellation.md`)

Deleted (self-hosting / out-of-scope per review):
- `not_setting_up_tls/` (directory with `client.go`, `README.md`)
- `not-enabling-ringpop-tls.md`
- `not-setting-up-persistence-rate-limits.md`
- `not-setting-up-namespaces-rate-limits.md`
- `underutilizing-namespaces.md`
- `not-understanding-why-youre-using-temporal.md`
- `not-investing-in-temporal-platform-team.md`
- `using-temporal-ui-for-non-debugging.md`
