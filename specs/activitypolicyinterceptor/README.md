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
| 5 | Timeout configuration must permit at least `RequiredRetries` retries (default 2) | Validate — surface violation |

Throughout this spec, "the interceptor" refers to the activity policy interceptor as installed on a worker. "The host SDK" refers to whichever Temporal SDK the implementer is targeting. Spec readers are assumed to be familiar with Temporal concepts; concept names follow the linked upstream documentation and may be spelled differently in each SDK.

---

## Configuration

1. THE INTERCEPTOR SHOULD be constructable from a configuration object containing at minimum these named options:

   | Option | Type | Default |
   |---|---|---|
   | `Severities` | map of policy identifier → `Severity` | see requirement 3 |
   | `AutoHeartbeat` | boolean | `true` |
   | `RequiredRetries` | non-negative integer; minimum number of retries the timeout configuration must permit (per requirement 16) | `2` |
   | `Logger` | structured logger handle, or null for the host SDK's default | `null` |

2. THE INTERCEPTOR SHALL define `Severity` as a closed enumeration with exactly three values: `Ignore`, `Warn`, `Error`. No other values are permitted.

3. THE INTERCEPTOR SHALL apply these defaults for any entry of the `Severities` map that the caller does not specify:

   | Policy identifier | Default severity |
   |---|---|
   | `schedule_to_close_required` | `Error` |
   | `max_attempts_too_low` | `Error` |
   | `local_activity_start_to_close_too_long` | `Error` |
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

12. THE INTERCEPTOR SHALL evaluate the `max_attempts_too_low` policy on every workflow-initiated activity scheduling call (regular and local). The policy is violated when the retry policy's `maximum_attempts` is set to 1 or 2.

13. THE INTERCEPTOR SHALL NOT raise a violation when the retry policy is unset, when `maximum_attempts` is unset, when `maximum_attempts == 0` (unlimited per Temporal semantics), or when `maximum_attempts >= 3`.

14. WHEN this policy is violated and logged at `Warn` severity THE INTERCEPTOR SHALL include the additional log field `max_attempts=<value>`.

---

## Workflow-Side Validation: Timeouts Permit Retries

15. THE INTERCEPTOR SHALL evaluate the `timeouts_permit_retries` policy on every workflow-initiated activity scheduling call. The policy guarantees that the timeout configuration leaves room for at least `N = RequiredRetries` retries to start within the schedule-to-close budget under the worst-case failure model in which every attempt fails immediately after starting and is detected only via the activity's configured timeouts.

16. Define the following derived values from the activity options. When the retry policy or any of its fields are unset, the values default to Temporal's documented defaults: `initial_interval = 1s`, `backoff_coefficient = 2.0`, `maximum_interval` unset.

    - `detection_delay = heartbeat_timeout` when `heartbeat_timeout` is set, otherwise `start_to_close_timeout`. This represents the worst-case time Temporal needs to detect a failed attempt: heartbeat-driven detection fires within one heartbeat interval, while start-to-close-only detection takes the full attempt timeout.
    - `retry_interval(n) = min(M, initial_interval × backoff_coefficient^(n-1))` for `n ≥ 1`, where `M = maximum_interval` when set and `100 × initial_interval` otherwise (Temporal's default upper bound).

    THE INTERCEPTOR SHALL treat the activity options as a violation of the `timeouts_permit_retries` policy when `schedule_to_close_timeout` is set, at least one of `heartbeat_timeout` or `start_to_close_timeout` is set (so `detection_delay` is determinable), and the following inequality holds:

    ```
    N × detection_delay + Σ retry_interval(n) for n in 1..=N  >  schedule_to_close_timeout
    ```

    Equivalently, the (N+1)-th attempt cannot start within the schedule-to-close budget. THE INTERCEPTOR SHALL NOT raise this violation when `N == 0`.

17. WHEN this policy is violated and logged at `Warn` severity THE INTERCEPTOR SHALL include the additional log fields `required_retries=<N>`, `detection_delay_seconds=<value>`, and `min_schedule_to_close_seconds=<value>` (the lower bound implied by the inequality in requirement 16, expressed in seconds).

---

## Workflow-Side Validation: Local Activities

18. WHERE the host SDK exposes a separate [local activity](https://docs.temporal.io/local-activity) scheduling path THE INTERCEPTOR SHALL evaluate the same workflow-side validation policies on local activity scheduling calls as on regular activities, EXCEPT that requirements 9–11 (default heartbeat) do not apply, since local activities do not support heartbeats.

19. THE INTERCEPTOR SHALL evaluate the `local_activity_start_to_close_too_long` policy on every workflow-initiated local activity scheduling call. The policy is violated when the local activity's `start_to_close_timeout` is unset, zero, or greater than or equal to 10 seconds.

20. WHEN this policy is violated and logged at `Warn` severity THE INTERCEPTOR SHALL include the additional log field `start_to_close_seconds=<value>`.

---

## Workflow-Side Validation: Multiple Violations

21. WHEN a single activity-scheduling call violates more than one policy AND at least one violated policy has severity `Error` THE INTERCEPTOR SHALL surface a single `PolicyViolationError` whose `policies` array contains all violated policy identifiers whose severity is `Warn` or `Error` (i.e., excluding `Ignore`), in this canonical order: `schedule_to_close_required`, `max_attempts_too_low`, `local_activity_start_to_close_too_long`, `timeouts_permit_retries`.

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
