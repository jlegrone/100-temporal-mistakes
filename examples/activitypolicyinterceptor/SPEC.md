# Specification: Activity Policy Interceptor

## Overview

The activity policy interceptor enforces and/or implements the policies from the "Grand Unified Theory of Activities" slide in `SLIDES.md`. It is implemented as a Temporal [worker interceptor](https://docs.temporal.io/encyclopedia/interceptors) and applies to both the workflow-side [activity](https://docs.temporal.io/activities) scheduling path and the activity-side execution path.

The four theory policies covered by this spec and their interceptor treatment:

| # | Policy | Interceptor role |
|---|---|---|
| 1 | Set a [schedule-to-close timeout](https://docs.temporal.io/encyclopedia/detecting-activity-failures) | Enforce — reject violators |
| 2 | Set a heartbeat or start-to-close timeout | Enforce — reject violators |
| 3 | Cleanup activities must heartbeat | Implement — auto-heartbeat helper |
| 4 | Prefer unlimited maximum attempts in the [retry policy](https://docs.temporal.io/encyclopedia/retry-policies) | Enforce — reject violators |

Throughout this spec, "the interceptor" refers to the activity policy interceptor as installed on a worker. "The host SDK" refers to whichever Temporal SDK the implementer is targeting. Spec readers are assumed to be familiar with Temporal concepts; concept names follow the linked upstream documentation and may be spelled differently in each SDK.

---

## Configuration

1. THE INTERCEPTOR SHALL be constructable from a configuration object containing at minimum these named options, each with the listed default:

   | Option | Type | Default |
   |---|---|---|
   | `RequireScheduleToClose` | boolean | `true` |
   | `RequireHeartbeatOrStartToClose` | boolean | `true` |
   | `ProhibitMaxAttempts` | boolean | `true` |
   | `AutoHeartbeat` | boolean | `false` |
   | `ViolationMode` | enum: `Fail` or `Warn` | `Fail` |
   | `Logger` | a structured logger handle, or null for the host SDK's default | `null` |

2. WHEN the interceptor is constructed with no options THE INTERCEPTOR SHALL apply the defaults from requirement 1.

3. THE INTERCEPTOR SHALL define `ViolationMode` as a closed enumeration with exactly two values: `Fail` and `Warn`. No other values are permitted.

4. THE INTERCEPTOR SHALL use the host SDK's recommended forward-compatibility mechanism for interceptor implementations (e.g. embedding the SDK's base interceptor types in Go, extending base interceptor classes in TypeScript and Java, subclassing in Python) so that future SDK additions to the interceptor surface do not break the implementation.

---

## Workflow-Side Validation: Schedule-to-Close

5. WHEN a workflow schedules an activity with no schedule-to-close timeout (zero or unset) AND `RequireScheduleToClose == true` AND `ViolationMode == Fail` THE INTERCEPTOR SHALL prevent the activity from being scheduled and SHALL surface a `PolicyViolationError` with `Policy == "schedule_to_close_required"` to the workflow as the result of the scheduling call.

6. WHEN a workflow schedules an activity with no schedule-to-close timeout AND `RequireScheduleToClose == true` AND `ViolationMode == Warn` THE INTERCEPTOR SHALL emit a warning log entry with structured fields `policy="schedule_to_close_required"`, `activity_type=<type>`, `workflow_id=<id>`, `run_id=<run_id>`, AND THE INTERCEPTOR SHALL forward the scheduling call to the next interceptor unchanged.

7. WHEN a workflow schedules an activity with a schedule-to-close timeout greater than zero THE INTERCEPTOR SHALL forward the scheduling call to the next interceptor unchanged regardless of `RequireScheduleToClose`.

8. WHERE `RequireScheduleToClose == false` THE INTERCEPTOR SHALL NOT raise a violation (error or warning) for a missing schedule-to-close timeout.

---

## Workflow-Side Validation: Heartbeat or Start-to-Close

9. WHEN a workflow schedules an activity with neither a heartbeat timeout nor a start-to-close timeout set AND `RequireHeartbeatOrStartToClose == true` AND `ViolationMode == Fail` THE INTERCEPTOR SHALL prevent the activity from being scheduled and SHALL surface a `PolicyViolationError` with `Policy == "heartbeat_or_start_to_close_required"` to the workflow as the result of the scheduling call.

10. WHEN a workflow schedules an activity with neither a heartbeat timeout nor a start-to-close timeout set AND `RequireHeartbeatOrStartToClose == true` AND `ViolationMode == Warn` THE INTERCEPTOR SHALL emit a warning log entry with structured fields `policy="heartbeat_or_start_to_close_required"`, `activity_type=<type>`, `workflow_id=<id>`, `run_id=<run_id>`.

11. WHEN at least one of (heartbeat timeout > 0) OR (start-to-close timeout > 0) holds THE INTERCEPTOR SHALL forward the scheduling call to the next interceptor unchanged.

---

## Workflow-Side Validation: Maximum Attempts

12. WHEN a workflow schedules an activity whose retry policy specifies a maximum attempts value greater than zero AND `ProhibitMaxAttempts == true` AND `ViolationMode == Fail` THE INTERCEPTOR SHALL prevent the activity from being scheduled and SHALL surface a `PolicyViolationError` with `Policy == "max_attempts_prohibited"` to the workflow as the result of the scheduling call.

13. WHEN a workflow schedules an activity whose retry policy specifies a maximum attempts value greater than zero AND `ProhibitMaxAttempts == true` AND `ViolationMode == Warn` THE INTERCEPTOR SHALL emit a warning log entry with structured fields `policy="max_attempts_prohibited"`, `activity_type=<type>`, `workflow_id=<id>`, `run_id=<run_id>`, `max_attempts=<n>`.

14. WHEN a workflow schedules an activity whose retry policy is null/absent OR whose maximum attempts value is zero THE INTERCEPTOR SHALL forward the scheduling call to the next interceptor unchanged.

---

## Workflow-Side Validation: Local Activities

15. WHERE the host SDK exposes a separate [local activity](https://docs.temporal.io/local-activity) scheduling path THE INTERCEPTOR SHALL apply the same validation rules as for regular activities, with the following exceptions: requirement 9 (heartbeat or start-to-close) is replaced by a requirement that start-to-close is set (since local activities do not support heartbeats), AND a 10-second upper bound on start-to-close per requirements 17–19.

16. THE INTERCEPTOR SHALL NOT enforce a heartbeat-timeout requirement on local activities.

17. WHEN a workflow schedules a local activity with a start-to-close timeout greater than or equal to 10 seconds AND `RequireHeartbeatOrStartToClose == true` AND `ViolationMode == Fail` THE INTERCEPTOR SHALL prevent the activity from being scheduled and SHALL surface a `PolicyViolationError` with `Policy == "local_activity_start_to_close_under_10s_required"` and `Details` indicating that local activities are intended to be short-lived (under 10 seconds).

18. WHEN a workflow schedules a local activity with a start-to-close timeout greater than or equal to 10 seconds AND `RequireHeartbeatOrStartToClose == true` AND `ViolationMode == Warn` THE INTERCEPTOR SHALL emit a warning log entry with structured fields `policy="local_activity_start_to_close_under_10s_required"`, `activity_type=<type>`, `workflow_id=<id>`, `run_id=<run_id>`, `start_to_close_seconds=<value>`.

19. WHEN a workflow schedules a local activity with a start-to-close timeout greater than zero AND less than 10 seconds THE INTERCEPTOR SHALL forward the scheduling call to the next interceptor unchanged for purposes of this rule.

---

## Workflow-Side Validation: Multiple Violations

20. WHEN a single activity-scheduling call violates more than one policy AND `ViolationMode == Fail` THE INTERCEPTOR SHALL surface a single `PolicyViolationError` whose `Policy` field contains all violated policy identifiers joined with `,`, in the order: `schedule_to_close_required`, `heartbeat_or_start_to_close_required`, `local_activity_start_to_close_under_10s_required`, `max_attempts_prohibited` (e.g., `"schedule_to_close_required,max_attempts_prohibited"`).

21. WHEN a single activity-scheduling call violates more than one policy AND `ViolationMode == Warn` THE INTERCEPTOR SHALL emit one warning log entry per violated policy.

---

## Activity-Side: Auto-Heartbeat

22. WHERE `AutoHeartbeat == true` WHEN an activity begins execution THE INTERCEPTOR SHALL ensure heartbeats are recorded at half the activity's configured heartbeat timeout cadence, falling back to a 30-second cadence when no heartbeat timeout is configured.

23. WHERE `AutoHeartbeat == true` WHEN an activity returns from execution by any means (return, exception/panic, or context/cancelation) THE INTERCEPTOR SHALL stop emitting auto-heartbeats before returning control to the worker.

24. WHERE `AutoHeartbeat == false` THE INTERCEPTOR SHALL NOT emit any heartbeats on the activity's behalf.

25. THE INTERCEPTOR SHALL NOT interfere with manual heartbeats issued by the activity itself; the auto-heartbeat mechanism is additive, not exclusive.

---

## Error Reporting

When the upstream requirements refer to a `PolicyViolationError`, the concrete artifact surfaced to the workflow is a Temporal [`ApplicationFailure`](https://docs.temporal.io/references/failures#application-failure) with the shape defined below. Identifying a violation across SDKs is done by matching `type` and reading the structured payload from `details`, not by host-language type assertions on a custom class.

26. THE INTERCEPTOR SHALL surface every policy violation as an `ApplicationFailure` with `type == "PolicyViolationError"` and `non_retryable == true`. THE INTERCEPTOR SHALL leave `next_retry_delay` and `cause` unset.

27. THE INTERCEPTOR SHALL set the `ApplicationFailure`'s `message` field to the literal form `"activity policy violation: <policies>: <explanation>"`, where `<policies>` is the comma-joined list of violated policy identifiers in canonical order (per requirement 20) and `<explanation>` is a short human-readable description.

28. THE INTERCEPTOR SHALL include exactly one structured payload as the first element of the `ApplicationFailure`'s `details` array (`details[0]`) with these fields, encoded in `snake_case` regardless of host-language naming conventions:

    | Field | Type | Description |
    |---|---|---|
    | `policies` | array of strings | Violated policy identifiers in canonical order (per requirement 20). |
    | `activity_type` | string | The activity type name passed to the scheduling call. |
    | `workflow_id` | string | The current workflow's ID. |
    | `run_id` | string | The current run ID. |
    | `explanation` | string | Human-readable explanation suitable for log lines. |

29. THE INTERCEPTOR SHALL NOT add additional entries to the `details` array beyond the structured payload defined in requirement 28.

---

## Conformance Tests

Conformance test cases live in [`conformance_tests.json`](conformance_tests.json) alongside this spec. Each test specifies an interceptor configuration, a scenario (scheduling or executing an activity), and the expected outcome (violation, forwarded, or execution side effects). Implementations in any host SDK should translate these tests into runnable cases using the SDK's test harness.
