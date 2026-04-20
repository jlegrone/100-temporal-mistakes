# Review Tracker

## Approved

(none yet)

## Revisit

- [ ] **#1 Not Understanding Why You're Using Temporal** ([`not-understanding-why-youre-using-temporal.md`](src/not-understanding-why-youre-using-temporal.md))
  - [ ] Add cross-reference to "Unnecessary Usage of Workflows"
  - [ ] Link to Temporal's own documentation on when to use (or not use) Temporal

- [ ] **#2 Unnecessary Usage of Workflows** ([`unnecessary-usage-of-workflows.md`](src/unnecessary-usage-of-workflows.md))
  - [ ] Add cross-reference to "Not Understanding Why You're Using Temporal"
  - [ ] Clarify replay phrasing on line 12 -- replay happens on recovery, not every execution
  - [x] ~~Nuance the TIP (line 5): duration isn't the deciding factor~~ — fixed, now focuses on durability/retry needs
  - [x] ~~Fix wrong advice on line 28: "completes in milliseconds, direct execution is fine"~~ — replaced with retry-focused question

- [ ] **#3 Building Custom Task Orchestration Frameworks on Top of Temporal** ([`custom-task-orchestration-frameworks.md`](src/custom-task-orchestration-frameworks.md))
  - [ ] Verify DSL samples link on line 42 is still valid (`temporalio/samples-go/tree/main/dsl`)

- [ ] **#4 Using the Temporal UI for Non-Debugging Purposes** ([`using-temporal-ui-for-non-debugging.md`](src/using-temporal-ui-for-non-debugging.md))
  - [ ] Change `## Solution` (line 26) to `## How?` for consistency with all other entries
  - [ ] Consider noting that Temporal Cloud has more granular RBAC, while open-source has only namespace-level access control (line 16)
