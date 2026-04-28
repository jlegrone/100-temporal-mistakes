# Specification: Activity Policy Interceptor

## Overview

The activity policy interceptor enforces and/or implements the policies from the "Grand Unified Theory of Activities" slide in `SLIDES.md`. It is implemented as a Temporal [worker interceptor](https://docs.temporal.io/encyclopedia/interceptors) and applies to both the workflow-side [activity](https://docs.temporal.io/activities) scheduling path and the activity-side execution path.

The five theory policies covered by this spec and their interceptor treatment:

| # | Policy | Interceptor role |
|---|---|---|
| 1 | Set a [schedule-to-close timeout](https://docs.temporal.io/encyclopedia/detecting-activity-failures) | Validate — surface violation |
| 2 | Set a heartbeat or start-to-close timeout | Implement — default `heartbeat_timeout` to 30s when neither is set |
| 3 | Cleanup activities must heartbeat | Implement — auto-heartbeat helper |
| 4 | Set [maximum attempts](https://docs.temporal.io/encyclopedia/retry-policies) to 0 (unlimited) or to a value ≥ 3 | Validate — surface violation |
| 5 | Timeout configuration must permit at least one retry | Validate — surface violation |

Throughout this spec, "the interceptor" refers to the activity policy interceptor as installed on a worker. "The host SDK" refers to whichever Temporal SDK the implementer is targeting. Spec readers are assumed to be familiar with Temporal concepts; concept names follow the linked upstream documentation and may be spelled differently in each SDK.

---

## Configuration

1. THE INTERCEPTOR SHOULD be constructable from a configuration object containing at minimum these named options:

   | Option | Type | Default |
   |---|---|---|
   | `Severities` | map of policy identifier → `Severity` | see requirement 3 |
   | `AutoHeartbeat` | boolean | `true` |
   | `Logger` | structured logger handle, or null for the host SDK's default | `null` |

2. THE INTERCEPTOR SHALL define `Severity` as a closed enumeration with exactly three values: `Ignore`, `Warn`, `Error`. No other values are permitted.

3. THE INTERCEPTOR SHALL apply these defaults for any entry of the `Severities` map that the caller does not specify:

   | Policy identifier | Default severity |
   |---|---|
   | `schedule_to_close_required` | `Error` |
   | `max_attempts_must_be_zero_or_at_least_3` | `Error` |
   | `local_activity_start_to_close_under_10s_required` | `Error` |
   | `timeouts_permit_retries` | `Error` |

4. WHEN a policy is evaluated AND its configured severity is `Ignore` THE INTERCEPTOR SHALL NOT enforce the policy and SHALL NOT emit a log entry for it.

5. WHEN a policy is violated AND its configured severity is `Warn` THE INTERCEPTOR SHALL emit a warning log entry containing the policy identifier, the contextual fields `activity_type`, `workflow_id`, and `run_id`, plus any policy-specific fields named in that policy's section, AND SHALL forward the scheduling call to the next interceptor unchanged.

6. WHEN a policy is violated AND its configured severity is `Error` THE INTERCEPTOR SHALL prevent the scheduling call from proceeding AND SHALL surface a `PolicyViolationError` shaped per the Error Reporting section.

7. THE INTERCEPTOR SHOULD use the host SDK's recommended forward-compatibility mechanism for interceptor implementations (e.g. embedding base interceptor types in Go, extending base classes in TypeScript and Java, subclassing in Python) so that future SDK additions to the interceptor surface do not break the implementation.

<!-- TODO: Define a mechanism for activities to opt out of auto-heartbeat (e.g., a per-activity decorator or marker, a per-scheduling-call ActivityOption, or a worker-level allow/deny list). -->

---

## Workflow-Side Validation: Schedule-to-Close

8. THE INTERCEPTOR SHALL evaluate the `schedule_to_close_required` policy on every workflow-initiated activity scheduling call (regular and local). The policy is violated when the activity's `schedule_to_close_timeout` is unset or zero.

---

## Workflow-Side Default: Heartbeat Timeout

9. WHEN a workflow schedules a regular (non-local) activity AND neither `heartbeat_timeout` nor `start_to_close_timeout` is set THE INTERCEPTOR SHALL set the activity's `heartbeat_timeout` to 30 seconds before forwarding the scheduling call to the next interceptor.

10. WHEN at least one of `heartbeat_timeout` or `start_to_close_timeout` is already set THE INTERCEPTOR SHALL NOT modify either field.

11. THE INTERCEPTOR SHALL NOT apply this default to local activities, which do not support heartbeats.

---

## Workflow-Side Validation: Maximum Attempts

12. THE INTERCEPTOR SHALL evaluate the `max_attempts_must_be_zero_or_at_least_3` policy on every workflow-initiated activity scheduling call (regular and local). The policy is violated when the retry policy's `maximum_attempts` is set to 1 or 2.

13. THE INTERCEPTOR SHALL NOT raise a violation when the retry policy is unset, when `maximum_attempts` is unset, when `maximum_attempts == 0` (unlimited per Temporal semantics), or when `maximum_attempts >= 3`.

14. WHEN this policy is violated and logged at `Warn` severity THE INTERCEPTOR SHALL include the additional log field `max_attempts=<value>`.

---

## Workflow-Side Validation: Timeouts Permit Retries

15. THE INTERCEPTOR SHALL evaluate the `timeouts_permit_retries` policy on every workflow-initiated activity scheduling call. The policy is violated when the activity's timeout configuration demonstrably leaves no room for at least one retry attempt.

16. THE INTERCEPTOR SHALL treat the following as a violation of the `timeouts_permit_retries` policy: `heartbeat_timeout` is unset AND `start_to_close_timeout` is set AND `schedule_to_close_timeout` is set AND `schedule_to_close_timeout < 2.1 * start_to_close_timeout`. WHEN `heartbeat_timeout` is set, this start-to-close-vs-schedule-to-close ratio rule SHALL NOT apply (heartbeat-driven retry detection covers worker failure during a long start-to-close window).

17. THE INTERCEPTOR SHOULD also detect other observable timeout configurations that prevent retries (e.g., a retry policy whose `maximum_interval` exceeds the remaining `schedule_to_close_timeout` budget after the first attempt) and surface them under the same `timeouts_permit_retries` policy identifier.

---

## Workflow-Side Validation: Local Activities

18. WHERE the host SDK exposes a separate [local activity](https://docs.temporal.io/local-activity) scheduling path THE INTERCEPTOR SHALL evaluate the same workflow-side validation policies on local activity scheduling calls as on regular activities, EXCEPT that requirements 9–11 (default heartbeat) do not apply, since local activities do not support heartbeats.

19. THE INTERCEPTOR SHALL evaluate the `local_activity_start_to_close_under_10s_required` policy on every workflow-initiated local activity scheduling call. The policy is violated when the local activity's `start_to_close_timeout` is unset, zero, or greater than or equal to 10 seconds.

20. WHEN this policy is violated and logged at `Warn` severity THE INTERCEPTOR SHALL include the additional log field `start_to_close_seconds=<value>`.

---

## Workflow-Side Validation: Multiple Violations

21. WHEN a single activity-scheduling call violates more than one policy AND at least one violated policy has severity `Error` THE INTERCEPTOR SHALL surface a single `PolicyViolationError` whose `policies` array contains all violated policy identifiers whose severity is `Warn` or `Error` (i.e., excluding `Ignore`), in this canonical order: `schedule_to_close_required`, `max_attempts_must_be_zero_or_at_least_3`, `local_activity_start_to_close_under_10s_required`, `timeouts_permit_retries`.

22. WHEN a single activity-scheduling call violates more than one policy AND no violated policy has severity `Error` THE INTERCEPTOR SHALL emit one warning log entry per violated policy whose severity is `Warn`, AND SHALL forward the scheduling call to the next interceptor unchanged.

---

## Activity-Side: Auto-Heartbeat

23. WHERE `AutoHeartbeat == true` WHEN an activity begins execution THE INTERCEPTOR SHALL ensure heartbeats are recorded at half the activity's configured heartbeat timeout cadence, falling back to a 30-second cadence when no heartbeat timeout is configured.

24. WHERE `AutoHeartbeat == true` WHEN an activity returns from execution by any means (return, exception/panic, or context/cancelation) THE INTERCEPTOR SHALL stop emitting auto-heartbeats before returning control to the worker.

25. WHERE `AutoHeartbeat == false` THE INTERCEPTOR SHALL NOT emit any heartbeats on the activity's behalf.

26. THE INTERCEPTOR SHALL NOT interfere with manual heartbeats issued by the activity itself; the auto-heartbeat mechanism is additive, not exclusive.

<!-- TODO: Define a mechanism for activities to opt out of auto-heartbeat. Possibilities include a per-activity decorator/option, a per-scheduling-call ActivityOption, or a worker-level allow/deny list. -->

---

## Error Reporting

When the upstream requirements refer to a `PolicyViolationError`, the concrete artifact surfaced to the workflow is a Temporal [`ApplicationFailure`](https://docs.temporal.io/references/failures#application-failure) with the shape defined below. Identifying a violation across SDKs is done by matching `type` and reading the structured payload from `details`, not by host-language type assertions on a custom class.

27. THE INTERCEPTOR SHALL surface every policy violation (when one or more policies has severity `Error`) as an `ApplicationFailure` with `type == "PolicyViolationError"` and `non_retryable == true`. THE INTERCEPTOR SHALL leave `next_retry_delay` and `cause` unset.

28. THE INTERCEPTOR SHALL set the `ApplicationFailure`'s `message` field to the literal form `"activity policy violation: <policies>: <explanation>"`, where `<policies>` is the comma-joined list of violated policy identifiers in canonical order (per requirement 21) and `<explanation>` is a short human-readable description.

29. THE INTERCEPTOR SHALL include exactly one structured payload as the first element of the `ApplicationFailure`'s `details` array (`details[0]`) with these fields, encoded in `snake_case` regardless of host-language naming conventions:

    | Field | Type | Description |
    |---|---|---|
    | `policies` | array of strings | Violated policy identifiers in canonical order (per requirement 21). |
    | `activity_type` | string | The activity type name passed to the scheduling call. |
    | `workflow_id` | string | The current workflow's ID. |
    | `run_id` | string | The current run ID. |
    | `explanation` | string | Human-readable explanation suitable for log lines. |

30. THE INTERCEPTOR SHALL NOT add additional entries to the `details` array beyond the structured payload defined in requirement 29.

---

## Conformance Tests

Conformance test cases live in [`conformance_tests.json`](conformance_tests.json) alongside this spec. Each test specifies an interceptor configuration, a scenario (scheduling or executing an activity), and the expected outcome (violation, forwarded, or execution side effects). Implementations in any host SDK should translate these tests into runnable cases using the SDK's test harness.
