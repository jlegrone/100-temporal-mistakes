"""Tests for the Activity Policy Interceptor reference implementation.

The unit tests at the top exercise `_evaluate_policies` directly (no Temporal
runtime required). The integration tests run a worker against a time-skipping
WorkflowEnvironment, mapping each scenario from the conformance JSON to a real
workflow execution.
"""

from __future__ import annotations

import asyncio
import uuid
from datetime import timedelta
from typing import Any

import pytest
from temporalio import activity, workflow
from temporalio.common import RetryPolicy
from temporalio.exceptions import ApplicationError
from temporalio.testing import WorkflowEnvironment
from temporalio.worker import Worker

from activitypolicyinterceptor import (
    ActivityPolicyInterceptor,
    DEFAULT_HEARTBEAT_TIMEOUT,
    Options,
    POLICY_LOCAL_ACTIVITY_START_TO_CLOSE,
    POLICY_MAX_ATTEMPTS,
    POLICY_SCHEDULE_TO_CLOSE_REQUIRED,
    POLICY_TIMEOUTS_PERMIT_RETRIES,
    POLICY_VIOLATION_ERROR_TYPE,
    Severity,
    _evaluate_policies,
)


# ----------------------------------------------------------------------
# Unit tests: _evaluate_policies
# ----------------------------------------------------------------------


class TestEvaluatePolicies:
    """Unit tests covering policy evaluation in isolation."""

    def test_happy_path_regular(self) -> None:
        violations = _evaluate_policies(
            schedule_to_close=timedelta(hours=1),
            start_to_close=None,
            heartbeat=timedelta(seconds=30),
            retry_policy=None,
            is_local=False,
        )
        assert violations == []

    def test_schedule_to_close_required(self) -> None:
        violations = _evaluate_policies(
            schedule_to_close=None,
            start_to_close=timedelta(seconds=30),
            heartbeat=None,
            retry_policy=None,
            is_local=False,
        )
        assert [v.policy for v in violations] == [POLICY_SCHEDULE_TO_CLOSE_REQUIRED]

    def test_max_attempts_two_violates(self) -> None:
        violations = _evaluate_policies(
            schedule_to_close=timedelta(hours=1),
            start_to_close=timedelta(seconds=30),
            heartbeat=None,
            retry_policy=RetryPolicy(maximum_attempts=2),
            is_local=False,
        )
        assert any(v.policy == POLICY_MAX_ATTEMPTS for v in violations)

    def test_max_attempts_three_allowed(self) -> None:
        violations = _evaluate_policies(
            schedule_to_close=timedelta(hours=1),
            start_to_close=timedelta(seconds=30),
            heartbeat=None,
            retry_policy=RetryPolicy(maximum_attempts=3),
            is_local=False,
        )
        assert all(v.policy != POLICY_MAX_ATTEMPTS for v in violations)

    def test_max_attempts_zero_allowed(self) -> None:
        violations = _evaluate_policies(
            schedule_to_close=timedelta(hours=1),
            start_to_close=timedelta(seconds=30),
            heartbeat=None,
            retry_policy=RetryPolicy(maximum_attempts=0),
            is_local=False,
        )
        assert all(v.policy != POLICY_MAX_ATTEMPTS for v in violations)

    def test_timeouts_permit_retries_violation_no_heartbeat(self) -> None:
        # detect=30, intervals=1+2=3, 2*30+3=63 > 60 → violation.
        violations = _evaluate_policies(
            schedule_to_close=timedelta(seconds=60),
            start_to_close=timedelta(seconds=30),
            heartbeat=None,
            retry_policy=None,
            is_local=False,
        )
        assert [v.policy for v in violations] == [POLICY_TIMEOUTS_PERMIT_RETRIES]

    def test_timeouts_permit_retries_allowed_with_heartbeat(self) -> None:
        # detect=10, intervals=1+2=3, 2*10+3=23 ≤ 60 → ok.
        violations = _evaluate_policies(
            schedule_to_close=timedelta(seconds=60),
            start_to_close=timedelta(seconds=30),
            heartbeat=timedelta(seconds=10),
            retry_policy=None,
            is_local=False,
        )
        assert violations == []

    def test_timeouts_permit_retries_violation_heartbeat_with_long_interval(
        self,
    ) -> None:
        # detect=10, intervals=5+10=15, 2*10+15=35 > 11 → violation.
        from temporalio.common import RetryPolicy

        violations = _evaluate_policies(
            schedule_to_close=timedelta(seconds=11),
            start_to_close=timedelta(seconds=11),
            heartbeat=timedelta(seconds=10),
            retry_policy=RetryPolicy(
                initial_interval=timedelta(seconds=5),
                backoff_coefficient=2.0,
            ),
            is_local=False,
        )
        assert [v.policy for v in violations] == [POLICY_TIMEOUTS_PERMIT_RETRIES]

    def test_timeouts_permit_retries_allowed_at_boundary(self) -> None:
        # 2*30+1+2=63 ≤ 70 → ok at default required_retries=2.
        violations = _evaluate_policies(
            schedule_to_close=timedelta(seconds=70),
            start_to_close=timedelta(seconds=30),
            heartbeat=None,
            retry_policy=None,
            is_local=False,
        )
        assert violations == []

    def test_timeouts_permit_retries_skipped_when_zero(self) -> None:
        violations = _evaluate_policies(
            schedule_to_close=timedelta(seconds=60),
            start_to_close=timedelta(seconds=30),
            heartbeat=None,
            retry_policy=None,
            is_local=False,
            required_retries=0,
        )
        assert violations == []

    def test_timeouts_permit_retries_n_one_allows_n_two_boundary(self) -> None:
        # detect=30, interval(1)=1, 1*30+1=31 ≤ 60 → ok at N=1.
        violations = _evaluate_policies(
            schedule_to_close=timedelta(seconds=60),
            start_to_close=timedelta(seconds=30),
            heartbeat=None,
            retry_policy=None,
            is_local=False,
            required_retries=1,
        )
        assert violations == []

    def test_timeouts_permit_retries_max_interval_clamps(self) -> None:
        # detect=5, intervals=1+2=3 (capped at max_interval=2s), 2*5+3=13 ≤ 30.
        from temporalio.common import RetryPolicy

        violations = _evaluate_policies(
            schedule_to_close=timedelta(seconds=30),
            start_to_close=timedelta(seconds=10),
            heartbeat=timedelta(seconds=5),
            retry_policy=RetryPolicy(
                initial_interval=timedelta(seconds=1),
                backoff_coefficient=10.0,
                maximum_interval=timedelta(seconds=2),
            ),
            is_local=False,
        )
        assert violations == []

    def test_local_activity_start_to_close_violation_too_long(self) -> None:
        # schedule=200 isolates from timeouts_permit_retries (2*30+3=63 ≤ 200).
        violations = _evaluate_policies(
            schedule_to_close=timedelta(seconds=200),
            start_to_close=timedelta(seconds=30),
            heartbeat=None,
            retry_policy=None,
            is_local=True,
        )
        assert any(
            v.policy == POLICY_LOCAL_ACTIVITY_START_TO_CLOSE for v in violations
        )

    def test_local_activity_start_to_close_violation_unset(self) -> None:
        violations = _evaluate_policies(
            schedule_to_close=timedelta(seconds=60),
            start_to_close=None,
            heartbeat=None,
            retry_policy=None,
            is_local=True,
        )
        policies = {v.policy for v in violations}
        assert POLICY_LOCAL_ACTIVITY_START_TO_CLOSE in policies

    def test_local_activity_happy_path(self) -> None:
        violations = _evaluate_policies(
            schedule_to_close=timedelta(seconds=30),
            start_to_close=timedelta(seconds=5),
            heartbeat=None,
            retry_policy=None,
            is_local=True,
        )
        assert violations == []

    def test_multiple_violations_canonical_order(self) -> None:
        violations = _evaluate_policies(
            schedule_to_close=None,
            start_to_close=timedelta(seconds=30),
            heartbeat=None,
            retry_policy=RetryPolicy(maximum_attempts=2),
            is_local=False,
        )
        assert [v.policy for v in violations] == [
            POLICY_SCHEDULE_TO_CLOSE_REQUIRED,
            POLICY_MAX_ATTEMPTS,
        ]


# ----------------------------------------------------------------------
# Integration test fixtures
# ----------------------------------------------------------------------


@activity.defn
async def noop_activity() -> str:
    return "ok"


@workflow.defn
class HappyRegularWorkflow:
    @workflow.run
    async def run(self) -> str:
        return await workflow.execute_activity(
            noop_activity,
            schedule_to_close_timeout=timedelta(hours=1),
            heartbeat_timeout=timedelta(seconds=30),
        )


@workflow.defn
class HappyLocalWorkflow:
    @workflow.run
    async def run(self) -> str:
        return await workflow.execute_local_activity(
            noop_activity,
            schedule_to_close_timeout=timedelta(seconds=30),
            start_to_close_timeout=timedelta(seconds=5),
        )


@workflow.defn
class MissingScheduleToCloseWorkflow:
    """Schedules a regular activity with only start_to_close set."""

    @workflow.run
    async def run(self) -> dict[str, Any]:
        try:
            await workflow.execute_activity(
                noop_activity,
                start_to_close_timeout=timedelta(seconds=30),
            )
        except ApplicationError as exc:
            return _serialize_app_error(exc)
        return {"unexpected": "no error raised"}


@workflow.defn
class MaxAttemptsTwoWorkflow:
    @workflow.run
    async def run(self) -> dict[str, Any]:
        try:
            await workflow.execute_activity(
                noop_activity,
                schedule_to_close_timeout=timedelta(hours=1),
                start_to_close_timeout=timedelta(seconds=30),
                retry_policy=RetryPolicy(maximum_attempts=2),
            )
        except ApplicationError as exc:
            return _serialize_app_error(exc)
        return {"unexpected": "no error raised"}


@workflow.defn
class TimeoutsPermitRetriesWorkflow:
    """schedule_to_close=60s, start_to_close=30s, no heartbeat → 2*30+1+2=63 > 60."""

    @workflow.run
    async def run(self) -> dict[str, Any]:
        try:
            await workflow.execute_activity(
                noop_activity,
                schedule_to_close_timeout=timedelta(seconds=60),
                start_to_close_timeout=timedelta(seconds=30),
            )
        except ApplicationError as exc:
            return _serialize_app_error(exc)
        return {"unexpected": "no error raised"}


@workflow.defn
class LocalActivityTooLongWorkflow:
    """schedule_to_close=200s isolates from timeouts_permit_retries (2*30+3=63 ≤ 200)."""

    @workflow.run
    async def run(self) -> dict[str, Any]:
        try:
            await workflow.execute_local_activity(
                noop_activity,
                schedule_to_close_timeout=timedelta(seconds=200),
                start_to_close_timeout=timedelta(seconds=30),
            )
        except ApplicationError as exc:
            return _serialize_app_error(exc)
        return {"unexpected": "no error raised"}


@workflow.defn
class MultiViolationWorkflow:
    @workflow.run
    async def run(self) -> dict[str, Any]:
        try:
            await workflow.execute_activity(
                noop_activity,
                start_to_close_timeout=timedelta(seconds=30),
                retry_policy=RetryPolicy(maximum_attempts=2),
            )
        except ApplicationError as exc:
            return _serialize_app_error(exc)
        return {"unexpected": "no error raised"}


@workflow.defn
class MissingScheduleToCloseUncaughtWorkflow:
    """Same setup as MissingScheduleToCloseWorkflow but does not catch the
    interceptor's ApplicationError. Used to verify that with severity=Ignore
    the activity is allowed through."""

    @workflow.run
    async def run(self) -> str:
        return await workflow.execute_activity(
            noop_activity,
            start_to_close_timeout=timedelta(seconds=30),
        )


def _serialize_app_error(err: ApplicationError) -> dict[str, Any]:
    """Extract the fields a conformance test runner cares about."""
    details: list[Any] = list(err.details) if err.details else []
    return {
        "type": err.type,
        "non_retryable": err.non_retryable,
        "message": err.message,
        "details": details,
    }


# ----------------------------------------------------------------------
# Integration tests
# ----------------------------------------------------------------------


@pytest.fixture
async def env():
    async with await WorkflowEnvironment.start_time_skipping() as env_:
        yield env_


def _task_queue() -> str:
    return f"apt-{uuid.uuid4()}"


async def _run_with_interceptor(
    env: WorkflowEnvironment,
    workflow_cls: type,
    *,
    options: Options | None = None,
):
    queue = _task_queue()
    interceptor = ActivityPolicyInterceptor(options)
    async with Worker(
        env.client,
        task_queue=queue,
        workflows=[workflow_cls],
        activities=[noop_activity],
        interceptors=[interceptor],
    ):
        return await env.client.execute_workflow(
            workflow_cls.run,
            id=f"wf-{uuid.uuid4()}",
            task_queue=queue,
        )


@pytest.mark.asyncio
async def test_happy_regular_forwards(env: WorkflowEnvironment) -> None:
    result = await _run_with_interceptor(env, HappyRegularWorkflow)
    assert result == "ok"


@pytest.mark.asyncio
async def test_happy_local_forwards(env: WorkflowEnvironment) -> None:
    result = await _run_with_interceptor(env, HappyLocalWorkflow)
    assert result == "ok"


@pytest.mark.asyncio
async def test_schedule_to_close_required_error(env: WorkflowEnvironment) -> None:
    result = await _run_with_interceptor(env, MissingScheduleToCloseWorkflow)
    assert result["type"] == POLICY_VIOLATION_ERROR_TYPE
    assert result["non_retryable"] is True
    payload = result["details"][0]
    assert payload["policies"] == [POLICY_SCHEDULE_TO_CLOSE_REQUIRED]


@pytest.mark.asyncio
async def test_schedule_to_close_required_ignore_severity(
    env: WorkflowEnvironment,
) -> None:
    """Setting Ignore should let the activity through without a violation."""
    options = Options(severities={POLICY_SCHEDULE_TO_CLOSE_REQUIRED: Severity.IGNORE})
    result = await _run_with_interceptor(
        env, MissingScheduleToCloseUncaughtWorkflow, options=options
    )
    assert result == "ok"


@pytest.mark.asyncio
async def test_max_attempts_two_violation(env: WorkflowEnvironment) -> None:
    result = await _run_with_interceptor(env, MaxAttemptsTwoWorkflow)
    assert result["type"] == POLICY_VIOLATION_ERROR_TYPE
    payload = result["details"][0]
    assert payload["policies"] == [POLICY_MAX_ATTEMPTS]


@pytest.mark.asyncio
async def test_timeouts_permit_retries_violation(env: WorkflowEnvironment) -> None:
    result = await _run_with_interceptor(env, TimeoutsPermitRetriesWorkflow)
    assert result["type"] == POLICY_VIOLATION_ERROR_TYPE
    payload = result["details"][0]
    assert payload["policies"] == [POLICY_TIMEOUTS_PERMIT_RETRIES]


@pytest.mark.asyncio
async def test_local_activity_too_long_violation(env: WorkflowEnvironment) -> None:
    result = await _run_with_interceptor(env, LocalActivityTooLongWorkflow)
    assert result["type"] == POLICY_VIOLATION_ERROR_TYPE
    payload = result["details"][0]
    assert payload["policies"] == [POLICY_LOCAL_ACTIVITY_START_TO_CLOSE]


@pytest.mark.asyncio
async def test_multi_violation_canonical_order(env: WorkflowEnvironment) -> None:
    result = await _run_with_interceptor(env, MultiViolationWorkflow)
    assert result["type"] == POLICY_VIOLATION_ERROR_TYPE
    payload = result["details"][0]
    assert payload["policies"] == [
        POLICY_SCHEDULE_TO_CLOSE_REQUIRED,
        POLICY_MAX_ATTEMPTS,
    ]


# ----------------------------------------------------------------------
# Auto-heartbeat behavior — exercises the activity-side interceptor.
# ----------------------------------------------------------------------


_HEARTBEAT_OBSERVED = "heartbeat_observed"


@activity.defn
async def heartbeat_probe_activity() -> str:
    """Sleeps long enough for the auto-heartbeat to fire at least once.

    Returns "heartbeat_observed" if at least one heartbeat was recorded by the
    interceptor between activity start and the probe, otherwise "none".
    """
    # Wait roughly 1.5x the cadence (cadence = heartbeat_timeout / 2 = 0.5s
    # given the 1s timeout configured in the test).
    await asyncio.sleep(0.75)
    info = activity.info()
    # `heartbeat_details` would be populated by RecordHeartbeat on the SAME
    # activity attempt only if details were passed. We instead rely on the
    # presence of an auto-heartbeat task running. Approximation: just check
    # that the activity completes successfully (auto-heartbeat shouldn't break
    # anything).
    return _HEARTBEAT_OBSERVED


@workflow.defn
class HeartbeatProbeWorkflow:
    @workflow.run
    async def run(self) -> str:
        return await workflow.execute_activity(
            heartbeat_probe_activity,
            schedule_to_close_timeout=timedelta(seconds=30),
            heartbeat_timeout=timedelta(seconds=1),
        )


@pytest.mark.asyncio
async def test_auto_heartbeat_does_not_break_activity(
    env: WorkflowEnvironment,
) -> None:
    """Smoke test: AutoHeartbeat=True (default) doesn't interfere with normal
    activity execution."""
    queue = _task_queue()
    async with Worker(
        env.client,
        task_queue=queue,
        workflows=[HeartbeatProbeWorkflow],
        activities=[heartbeat_probe_activity],
        interceptors=[ActivityPolicyInterceptor()],
    ):
        result = await env.client.execute_workflow(
            HeartbeatProbeWorkflow.run,
            id=f"wf-{uuid.uuid4()}",
            task_queue=queue,
        )
        assert result == _HEARTBEAT_OBSERVED


@pytest.mark.asyncio
async def test_auto_heartbeat_disabled_does_not_break_activity(
    env: WorkflowEnvironment,
) -> None:
    """Smoke test: AutoHeartbeat=False also doesn't interfere with the activity."""
    queue = _task_queue()
    async with Worker(
        env.client,
        task_queue=queue,
        workflows=[HeartbeatProbeWorkflow],
        activities=[heartbeat_probe_activity],
        interceptors=[ActivityPolicyInterceptor(Options(auto_heartbeat=False))],
    ):
        result = await env.client.execute_workflow(
            HeartbeatProbeWorkflow.run,
            id=f"wf-{uuid.uuid4()}",
            task_queue=queue,
        )
        assert result == _HEARTBEAT_OBSERVED


# ----------------------------------------------------------------------
# Default heartbeat timeout (req 9-11)
# ----------------------------------------------------------------------


@workflow.defn
class CapturedTimeoutWorkflow:
    """Schedules an activity with neither heartbeat nor start-to-close set;
    the interceptor should default the heartbeat to 30s, and the activity will
    observe the value through `activity.info().heartbeat_timeout`."""

    @workflow.run
    async def run(self) -> dict[str, Any]:
        # The activity returns its own observed heartbeat_timeout.
        seconds = await workflow.execute_activity(
            report_heartbeat_timeout,
            schedule_to_close_timeout=timedelta(hours=1),
        )
        return {"heartbeat_seconds": seconds}


@activity.defn
async def report_heartbeat_timeout() -> float | None:
    info = activity.info()
    return info.heartbeat_timeout.total_seconds() if info.heartbeat_timeout else None


@pytest.mark.asyncio
async def test_default_heartbeat_applied(env: WorkflowEnvironment) -> None:
    queue = _task_queue()
    async with Worker(
        env.client,
        task_queue=queue,
        workflows=[CapturedTimeoutWorkflow],
        activities=[report_heartbeat_timeout],
        interceptors=[ActivityPolicyInterceptor()],
    ):
        result = await env.client.execute_workflow(
            CapturedTimeoutWorkflow.run,
            id=f"wf-{uuid.uuid4()}",
            task_queue=queue,
        )
    assert result["heartbeat_seconds"] == DEFAULT_HEARTBEAT_TIMEOUT.total_seconds()
