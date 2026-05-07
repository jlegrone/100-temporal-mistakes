"""Reference Python implementation of the Activity Policy Interceptor spec.

See ../../../specs/activitypolicyinterceptor/README.md for the language-agnostic
specification. Requirement numbers cited in inline comments refer to that
document.
"""

from __future__ import annotations

import asyncio
import dataclasses
import enum
import logging
from datetime import timedelta
from typing import Any, Callable, Optional

from temporalio import activity, workflow
from temporalio.exceptions import ApplicationError
from temporalio.worker import (
    ActivityInboundInterceptor,
    ExecuteActivityInput,
    Interceptor,
    StartActivityInput,
    StartLocalActivityInput,
    WorkflowInboundInterceptor,
    WorkflowInterceptorClassInput,
    WorkflowOutboundInterceptor,
)

# ----------------------------------------------------------------------
# Public constants
# ----------------------------------------------------------------------

# Policy identifiers (must match conformance_tests.json).
POLICY_SCHEDULE_TO_CLOSE_REQUIRED = "schedule_to_close_required"
POLICY_MAX_ATTEMPTS = "max_attempts_too_low"
POLICY_LOCAL_ACTIVITY_START_TO_CLOSE = "local_activity_start_to_close_too_long"
POLICY_TIMEOUTS_PERMIT_RETRIES = "timeouts_permit_retries"

POLICY_VIOLATION_ERROR_TYPE = "PolicyViolationError"

# Canonical policy ordering for multi-violation aggregation (spec req 21).
_CANONICAL_ORDER = (
    POLICY_SCHEDULE_TO_CLOSE_REQUIRED,
    POLICY_MAX_ATTEMPTS,
    POLICY_LOCAL_ACTIVITY_START_TO_CLOSE,
    POLICY_TIMEOUTS_PERMIT_RETRIES,
)

# Default applied by req 9 when neither heartbeat nor start-to-close is set.
DEFAULT_HEARTBEAT_TIMEOUT = timedelta(seconds=30)
# Fallback cadence for auto-heartbeat when no heartbeat timeout is configured (req 23).
DEFAULT_AUTO_HEARTBEAT_CADENCE = timedelta(seconds=30)
# Default value for Options.required_retries (req 1).
DEFAULT_REQUIRED_RETRIES = 2
# Temporal default retry-policy values used by req 16 when the corresponding
# RetryPolicy fields are unset.
DEFAULT_RETRY_INITIAL_INTERVAL = timedelta(seconds=1)
DEFAULT_RETRY_BACKOFF_COEFFICIENT = 2.0
DEFAULT_MAX_INTERVAL_MULTIPLIER = 100
# Local-activity start-to-close upper bound (req 19).
LOCAL_ACTIVITY_MAX_START_TO_CLOSE = timedelta(seconds=10)


class Severity(enum.Enum):
    """Severity levels per spec requirement 2."""

    IGNORE = "Ignore"
    WARN = "Warn"
    ERROR = "Error"


# Default severities per spec requirement 3.
_DEFAULT_SEVERITIES: dict[str, Severity] = {
    POLICY_SCHEDULE_TO_CLOSE_REQUIRED: Severity.ERROR,
    POLICY_MAX_ATTEMPTS: Severity.ERROR,
    POLICY_LOCAL_ACTIVITY_START_TO_CLOSE: Severity.ERROR,
    POLICY_TIMEOUTS_PERMIT_RETRIES: Severity.ERROR,
}


@dataclasses.dataclass
class Options:
    """Configuration object per spec requirement 1.

    Attributes:
        severities: Override the default severity for any policy by mapping
            its identifier to a `Severity`. Unspecified policies keep their
            default `Severity.ERROR`.
        auto_heartbeat: When True (default per req 23), the interceptor records
            heartbeats during activity execution at half the configured
            heartbeat-timeout cadence (or every 30s when no heartbeat timeout
            is configured).
        required_retries: Minimum number of retries the timeout configuration
            must permit per req 16. Defaults to 2. A value of 0 disables the
            timeouts_permit_retries policy entirely.
        logger: Optional structured logger handle. Defaults to the workflow's
            built-in logger inside workflow code.
    """

    severities: dict[str, Severity] = dataclasses.field(default_factory=dict)
    auto_heartbeat: bool = True
    required_retries: int = DEFAULT_REQUIRED_RETRIES
    logger: Optional[logging.Logger] = None

    def severity_for(self, policy: str) -> Severity:
        return self.severities.get(
            policy, _DEFAULT_SEVERITIES.get(policy, Severity.ERROR)
        )


# ----------------------------------------------------------------------
# Internal helpers
# ----------------------------------------------------------------------


@dataclasses.dataclass
class _Violation:
    policy: str
    explanation: str
    extra_fields: dict[str, Any] = dataclasses.field(default_factory=dict)


def _is_unset_or_zero(td: Optional[timedelta]) -> bool:
    return td is None or td.total_seconds() == 0


def _evaluate_policies(
    *,
    schedule_to_close: Optional[timedelta],
    start_to_close: Optional[timedelta],
    heartbeat: Optional[timedelta],
    retry_policy: Any,
    is_local: bool,
    required_retries: int = DEFAULT_REQUIRED_RETRIES,
) -> list[_Violation]:
    """Evaluate the four validation policies in canonical order.

    Returns the list of violated policies (empty when none).
    """
    violations: list[_Violation] = []

    # Policy 1: schedule_to_close_required (regular and local).
    if _is_unset_or_zero(schedule_to_close):
        violations.append(
            _Violation(
                policy=POLICY_SCHEDULE_TO_CLOSE_REQUIRED,
                explanation="schedule_to_close_timeout must be set to a positive value",
            )
        )

    # Policy 4: max_attempts_too_low (regular and local).
    if retry_policy is not None:
        max_attempts = getattr(retry_policy, "maximum_attempts", None)
        if max_attempts is not None and 0 < max_attempts < 3:
            violations.append(
                _Violation(
                    policy=POLICY_MAX_ATTEMPTS,
                    explanation="retry_policy.maximum_attempts must be 0 (unlimited) or >= 3",
                    extra_fields={"max_attempts": max_attempts},
                )
            )

    # Policy 19: local_activity_start_to_close_too_long.
    if is_local:
        if _is_unset_or_zero(start_to_close) or (
            start_to_close is not None
            and start_to_close >= LOCAL_ACTIVITY_MAX_START_TO_CLOSE
        ):
            violations.append(
                _Violation(
                    policy=POLICY_LOCAL_ACTIVITY_START_TO_CLOSE,
                    explanation="local activity start_to_close_timeout must be set and < 10 seconds",
                    extra_fields={
                        "start_to_close_seconds": (
                            start_to_close.total_seconds() if start_to_close else None
                        )
                    },
                )
            )

    # Policy 16: timeouts_permit_retries — closed-form check that the
    # (required_retries+1)-th attempt can start within schedule_to_close
    # under the worst-case crash-during-attempt failure model.
    if (
        required_retries > 0
        and schedule_to_close is not None
        and schedule_to_close.total_seconds() > 0
    ):
        detect = _detection_delay(heartbeat, start_to_close)
        if detect is not None and detect.total_seconds() > 0:
            min_required = _min_schedule_to_close(detect, retry_policy, required_retries)
            if schedule_to_close < min_required:
                violations.append(
                    _Violation(
                        policy=POLICY_TIMEOUTS_PERMIT_RETRIES,
                        explanation=(
                            f"schedule_to_close_timeout must be >= {min_required} "
                            f"to permit {required_retries} retries "
                            f"(detection_delay={detect})"
                        ),
                        extra_fields={
                            "required_retries": required_retries,
                            "detection_delay_seconds": detect.total_seconds(),
                            "min_schedule_to_close_seconds": min_required.total_seconds(),
                        },
                    )
                )

    return _canonicalize(violations)


def _detection_delay(
    heartbeat: Optional[timedelta], start_to_close: Optional[timedelta]
) -> Optional[timedelta]:
    """Implements the detection_delay definition from spec req 16: heartbeat
    when set, otherwise start_to_close. Returns None if neither is set."""
    if heartbeat is not None and heartbeat.total_seconds() > 0:
        return heartbeat
    if start_to_close is not None and start_to_close.total_seconds() > 0:
        return start_to_close
    return None


def _retry_interval(n: int, retry_policy: Any) -> timedelta:
    """Returns interval(n) per spec req 16. Unset retry-policy fields fall back
    to Temporal's documented defaults."""
    initial = DEFAULT_RETRY_INITIAL_INTERVAL
    backoff = DEFAULT_RETRY_BACKOFF_COEFFICIENT
    max_interval: Optional[timedelta] = None
    if retry_policy is not None:
        rp_initial = getattr(retry_policy, "initial_interval", None)
        if isinstance(rp_initial, timedelta) and rp_initial.total_seconds() > 0:
            initial = rp_initial
        rp_backoff = getattr(retry_policy, "backoff_coefficient", None)
        if isinstance(rp_backoff, (int, float)) and rp_backoff > 0:
            backoff = float(rp_backoff)
        rp_max = getattr(retry_policy, "maximum_interval", None)
        if isinstance(rp_max, timedelta) and rp_max.total_seconds() > 0:
            max_interval = rp_max
    if max_interval is None:
        max_interval = initial * DEFAULT_MAX_INTERVAL_MULTIPLIER
    scaled_seconds = initial.total_seconds() * (backoff ** (n - 1))
    if scaled_seconds > max_interval.total_seconds():
        return max_interval
    return timedelta(seconds=scaled_seconds)


def _min_schedule_to_close(
    detect: timedelta, retry_policy: Any, required_retries: int
) -> timedelta:
    """Computes N×detection_delay + Σ retry_interval(n) for n in 1..=N — the
    lower bound implied by spec req 16."""
    total = detect * required_retries
    for i in range(1, required_retries + 1):
        total += _retry_interval(i, retry_policy)
    return total


def _canonicalize(violations: list[_Violation]) -> list[_Violation]:
    """Sort violations by canonical policy order (spec req 21)."""
    order = {p: i for i, p in enumerate(_CANONICAL_ORDER)}
    return sorted(violations, key=lambda v: order[v.policy])


def _build_application_failure(
    violations: list[_Violation],
    *,
    activity_type: str,
    workflow_id: str,
    run_id: str,
) -> ApplicationError:
    """Build the ApplicationFailure shape per spec requirements 27-30."""
    policies = [v.policy for v in violations]
    explanation = "; ".join(v.explanation for v in violations)
    message = f"activity policy violation: {','.join(policies)}: {explanation}"
    details_payload = {
        "policies": policies,
        "activity_type": activity_type,
        "workflow_id": workflow_id,
        "run_id": run_id,
        "explanation": explanation,
    }
    # Spec req 27: type, non_retryable; leave next_retry_delay and cause unset.
    return ApplicationError(
        message,
        details_payload,
        type=POLICY_VIOLATION_ERROR_TYPE,
        non_retryable=True,
    )


# ----------------------------------------------------------------------
# Outbound interceptor (workflow side: validation + default heartbeat)
# ----------------------------------------------------------------------


class _OutboundInterceptor(WorkflowOutboundInterceptor):
    def __init__(self, next_: WorkflowOutboundInterceptor, opts: Options):
        super().__init__(next_)
        self._opts = opts

    def start_activity(self, input_: StartActivityInput):
        self._validate_and_default(
            activity_type=input_.activity,
            input_=input_,
            is_local=False,
        )
        return self.next.start_activity(input_)

    def start_local_activity(self, input_: StartLocalActivityInput):
        self._validate_and_default(
            activity_type=input_.activity,
            input_=input_,
            is_local=True,
        )
        return self.next.start_local_activity(input_)

    def _validate_and_default(
        self,
        *,
        activity_type: str,
        input_: Any,
        is_local: bool,
    ) -> None:
        schedule_to_close: Optional[timedelta] = input_.schedule_to_close_timeout
        start_to_close: Optional[timedelta] = input_.start_to_close_timeout
        heartbeat: Optional[timedelta] = (
            getattr(input_, "heartbeat_timeout", None) if not is_local else None
        )
        retry_policy = input_.retry_policy

        # Apply default heartbeat per req 9-11. Mutate first so the
        # timeouts_permit_retries check sees the final state.
        if (
            not is_local
            and heartbeat is None
            and start_to_close is None
        ):
            input_.heartbeat_timeout = DEFAULT_HEARTBEAT_TIMEOUT
            heartbeat = DEFAULT_HEARTBEAT_TIMEOUT

        violations = _evaluate_policies(
            schedule_to_close=schedule_to_close,
            start_to_close=start_to_close,
            heartbeat=heartbeat,
            retry_policy=retry_policy,
            is_local=is_local,
            required_retries=self._opts.required_retries,
        )
        if not violations:
            return

        # Bucket each violation by configured severity.
        warn_or_error: list[_Violation] = []
        has_error = False
        for v in violations:
            sev = self._opts.severity_for(v.policy)
            if sev is Severity.IGNORE:
                continue  # req 4: no enforcement, no log
            warn_or_error.append(v)
            if sev is Severity.ERROR:
                has_error = True

        if not warn_or_error:
            return

        info = workflow.info()
        logger = self._opts.logger or workflow.logger

        # Emit one log entry per non-ignored violation (req 5; reqs 14, 20 add
        # policy-specific extras).
        for v in warn_or_error:
            sev = self._opts.severity_for(v.policy)
            mode = "fail" if sev is Severity.ERROR else "warn"
            logger.warning(
                f"activity policy violation: {v.policy}",
                extra={
                    "policy": v.policy,
                    "activity_type": activity_type,
                    "workflow_id": info.workflow_id,
                    "run_id": info.run_id,
                    "mode": mode,
                    **v.extra_fields,
                },
            )

        if has_error:
            # Req 21: surface a single ApplicationFailure naming all warn+error
            # policies (Ignore is excluded above) in canonical order.
            raise _build_application_failure(
                warn_or_error,
                activity_type=activity_type,
                workflow_id=info.workflow_id,
                run_id=info.run_id,
            )


# ----------------------------------------------------------------------
# Inbound interceptor (activity side: auto-heartbeat)
# ----------------------------------------------------------------------


class _ActivityInbound(ActivityInboundInterceptor):
    def __init__(self, next_: ActivityInboundInterceptor, opts: Options):
        super().__init__(next_)
        self._opts = opts

    async def execute_activity(self, input_: ExecuteActivityInput) -> Any:
        # Req 25: when AutoHeartbeat is False, do not emit any heartbeats.
        if not self._opts.auto_heartbeat:
            return await self.next.execute_activity(input_)

        # Req 23: cadence is half the configured heartbeat timeout, falling back
        # to 30s when no heartbeat timeout is configured.
        info = activity.info()
        timeout = info.heartbeat_timeout
        if timeout and timeout.total_seconds() > 0:
            cadence_seconds = timeout.total_seconds() / 2
        else:
            cadence_seconds = DEFAULT_AUTO_HEARTBEAT_CADENCE.total_seconds()

        task = asyncio.create_task(_heartbeat_loop(cadence_seconds))
        try:
            return await self.next.execute_activity(input_)
        finally:
            # Req 24: stop emitting heartbeats before returning, regardless of
            # how the activity exited.
            task.cancel()
            try:
                await task
            except asyncio.CancelledError:
                pass


async def _heartbeat_loop(cadence_seconds: float) -> None:
    while True:
        await asyncio.sleep(cadence_seconds)
        activity.heartbeat()


# ----------------------------------------------------------------------
# Public Interceptor entry point
# ----------------------------------------------------------------------


class ActivityPolicyInterceptor(Interceptor):
    """Worker interceptor implementing the Activity Policy Interceptor spec.

    Usage:

        from activitypolicyinterceptor import ActivityPolicyInterceptor, Options

        worker = Worker(
            client,
            task_queue="my-queue",
            workflows=[...],
            activities=[...],
            interceptors=[ActivityPolicyInterceptor()],  # or pass Options(...)
        )

    See specs/activitypolicyinterceptor/README.md for the spec this implements.
    """

    def __init__(self, options: Options | None = None):
        self._opts = options or Options()

    def intercept_activity(
        self, next_: ActivityInboundInterceptor
    ) -> ActivityInboundInterceptor:
        return _ActivityInbound(next_, self._opts)

    def workflow_interceptor_class(
        self, input_: WorkflowInterceptorClassInput
    ) -> type[WorkflowInboundInterceptor] | None:
        opts = self._opts

        class _Inbound(WorkflowInboundInterceptor):
            def init(self, outbound: WorkflowOutboundInterceptor) -> None:
                super().init(_OutboundInterceptor(outbound, opts))

        return _Inbound
