"""Runtime conformance test runner for the Activity Policy Interceptor spec.

Loads `specs/activitypolicyinterceptor/conformance_tests.json` at runtime —
the same canonical source of truth that the Go test runner embeds via
`//go:embed` — and executes every `schedule_activity` scenario as a real
workflow against a time-skipping `WorkflowEnvironment`.

`execute_activity` scenarios are skipped (they require longer-lived activity
runtime fixtures than this runner sets up).
"""

from __future__ import annotations

import json
import os
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
    Options,
    POLICY_VIOLATION_ERROR_TYPE,
    Severity,
)

# ----------------------------------------------------------------------
# Locate and parse the canonical conformance JSON.
#
# Path resolution happens inside a function (not at module top level) because
# the Temporal workflow sandbox restricts pathlib.Path.resolve() during the
# import-and-validate step, and the workflow definitions in this module would
# trip that restriction.
# ----------------------------------------------------------------------


def _load_conformance() -> dict[str, Any]:
    # Resolve <repo>/specs/activitypolicyinterceptor/conformance_tests.json
    # from this file's location: <repo>/examples/py/activitypolicyinterceptor/.
    here = os.path.dirname(os.path.abspath(__file__))
    spec_path = os.path.normpath(
        os.path.join(here, "..", "..", "..", "specs", "activitypolicyinterceptor", "conformance_tests.json")
    )
    if not os.path.exists(spec_path):
        raise RuntimeError(
            f"conformance JSON not found at {spec_path}; this test file must "
            "live under <repo>/examples/py/activitypolicyinterceptor/"
        )
    with open(spec_path, "r", encoding="utf-8") as f:
        return json.load(f)


def _seconds(value: Any) -> timedelta | None:
    if value is None:
        return None
    return timedelta(seconds=float(value))


def _build_options(cfg: dict[str, Any]) -> Options:
    opts = Options()
    sev = cfg.get("Severities")
    if sev:
        opts.severities = {policy: Severity(name) for policy, name in sev.items()}
    if "AutoHeartbeat" in cfg and cfg["AutoHeartbeat"] is not None:
        opts.auto_heartbeat = bool(cfg["AutoHeartbeat"])
    if "RequiredRetries" in cfg and cfg["RequiredRetries"] is not None:
        opts.required_retries = int(cfg["RequiredRetries"])
    return opts


def _build_retry_policy(rp: dict[str, Any] | None) -> RetryPolicy | None:
    if rp is None:
        return None
    kwargs: dict[str, Any] = {}
    if rp.get("initial_interval_seconds") is not None:
        kwargs["initial_interval"] = _seconds(rp["initial_interval_seconds"])
    if rp.get("backoff_coefficient") is not None:
        kwargs["backoff_coefficient"] = float(rp["backoff_coefficient"])
    if rp.get("maximum_interval_seconds") is not None:
        kwargs["maximum_interval"] = _seconds(rp["maximum_interval_seconds"])
    if rp.get("maximum_attempts") is not None:
        kwargs["maximum_attempts"] = int(rp["maximum_attempts"])
    return RetryPolicy(**kwargs)


# ----------------------------------------------------------------------
# Conformance workflow + activity fixtures.
#
# A single workflow + activity pair drives every scenario. Activity options
# are passed through the workflow argument so we don't need to redefine a
# workflow per case (which would explode test compilation time).
# ----------------------------------------------------------------------


@activity.defn(name="noopActivity")
async def conformance_activity() -> dict[str, Any]:
    info = activity.info()
    return {
        "heartbeat_seconds": (
            info.heartbeat_timeout.total_seconds()
            if info.heartbeat_timeout
            else None
        ),
    }


@workflow.defn
class _ConformanceRegularWorkflow:
    @workflow.run
    async def run(self, args: dict[str, Any]) -> dict[str, Any]:
        kwargs: dict[str, Any] = {}
        if args.get("schedule_to_close_seconds") is not None:
            kwargs["schedule_to_close_timeout"] = timedelta(
                seconds=args["schedule_to_close_seconds"]
            )
        if args.get("start_to_close_seconds") is not None:
            kwargs["start_to_close_timeout"] = timedelta(
                seconds=args["start_to_close_seconds"]
            )
        if args.get("heartbeat_seconds") is not None:
            kwargs["heartbeat_timeout"] = timedelta(
                seconds=args["heartbeat_seconds"]
            )
        if args.get("retry_policy") is not None:
            kwargs["retry_policy"] = _build_retry_policy(args["retry_policy"])
        try:
            result = await workflow.execute_activity(
                conformance_activity, **kwargs
            )
            return {"ok": result}
        except ApplicationError as exc:
            return _serialize(exc)


@workflow.defn
class _ConformanceLocalWorkflow:
    @workflow.run
    async def run(self, args: dict[str, Any]) -> dict[str, Any]:
        kwargs: dict[str, Any] = {}
        if args.get("schedule_to_close_seconds") is not None:
            kwargs["schedule_to_close_timeout"] = timedelta(
                seconds=args["schedule_to_close_seconds"]
            )
        if args.get("start_to_close_seconds") is not None:
            kwargs["start_to_close_timeout"] = timedelta(
                seconds=args["start_to_close_seconds"]
            )
        if args.get("retry_policy") is not None:
            kwargs["retry_policy"] = _build_retry_policy(args["retry_policy"])
        try:
            result = await workflow.execute_local_activity(
                conformance_activity, **kwargs
            )
            return {"ok": result}
        except ApplicationError as exc:
            return _serialize(exc)


def _serialize(err: ApplicationError) -> dict[str, Any]:
    return {
        "error": {
            "type": err.type,
            "non_retryable": err.non_retryable,
            "message": err.message,
            "details": list(err.details) if err.details else [],
        }
    }


# ----------------------------------------------------------------------
# Test runner: one parametrized test per `schedule_activity` case.
# ----------------------------------------------------------------------


def _runnable_cases() -> list[dict[str, Any]]:
    return list(_load_conformance()["tests"])


@pytest.fixture(scope="module")
async def env():
    async with await WorkflowEnvironment.start_time_skipping() as env_:
        yield env_


@pytest.mark.asyncio
@pytest.mark.parametrize("case", _runnable_cases(), ids=lambda c: c["id"])
async def test_conformance_case(case: dict[str, Any], env: WorkflowEnvironment) -> None:
    if case["scenario"]["type"] == "execute_activity":
        pytest.skip(
            f"execute_activity scenarios require activity-runtime fixtures "
            f"(req {case['spec_requirement']})"
        )

    if case["scenario"]["type"] != "schedule_activity":
        pytest.fail(f"unsupported scenario.type {case['scenario']['type']!r}")

    opts = _build_options(case["interceptor_config"])
    is_local = case["scenario"]["activity_kind"] == "local"
    wf_cls = _ConformanceLocalWorkflow if is_local else _ConformanceRegularWorkflow

    queue = f"apt-conf-{uuid.uuid4()}"
    async with Worker(
        env.client,
        task_queue=queue,
        workflows=[wf_cls],
        activities=[conformance_activity],
        interceptors=[ActivityPolicyInterceptor(opts)],
    ):
        result = await env.client.execute_workflow(
            wf_cls.run,
            case["scenario"]["activity_options"],
            id=f"wf-{uuid.uuid4()}",
            task_queue=queue,
        )

    expected = case["expected"]
    outcome = expected["outcome"]

    if outcome == "violation":
        assert "error" in result, f"expected violation; got result={result}"
        err = result["error"]
        af = expected["application_failure"]
        assert err["type"] == af["type"]
        assert err["non_retryable"] == af["non_retryable"]
        payload = err["details"][0]
        assert payload["policies"] == af["details_payload_policies"]
    elif outcome == "forwarded":
        assert "ok" in result, f"expected forwarded; got result={result}"
    elif outcome == "forwarded_with_mutation":
        assert "ok" in result, f"expected forwarded_with_mutation; got result={result}"
        mutated = expected.get("mutated_options") or {}
        if mutated.get("heartbeat_seconds") is not None:
            assert (
                result["ok"]["heartbeat_seconds"]
                == mutated["heartbeat_seconds"]
            ), "applied heartbeat_timeout mismatch"
    elif outcome == "warn_logged_and_forwarded":
        # Warn-mode forwards the call; we don't capture the workflow logger
        # output here, so just verify the call was forwarded.
        assert "ok" in result, f"expected warn_logged_and_forwarded; got {result}"
    else:
        pytest.fail(f"unsupported expected.outcome {outcome!r}")
