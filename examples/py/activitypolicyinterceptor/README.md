# Activity Policy Interceptor (Python)

Reference Python implementation of the [Activity Policy Interceptor specification](../../../specs/activitypolicyinterceptor/README.md). Implements all four validation policies, the default-heartbeat behavior, and auto-heartbeat. Surfaces violations as [`ApplicationFailure`](https://docs.temporal.io/references/failures#application-failure) with `type="PolicyViolationError"` and the spec's structured `details[0]` payload.

## Install

```bash
uv venv
source .venv/bin/activate
uv pip install -e ".[test]"
```

## Use

```python
from activitypolicyinterceptor import ActivityPolicyInterceptor, Options, Severity

worker = Worker(
    client,
    task_queue="my-queue",
    workflows=[...],
    activities=[...],
    interceptors=[ActivityPolicyInterceptor()],  # all policies at Error severity
)
```

To downgrade or disable individual policies:

```python
ActivityPolicyInterceptor(
    Options(
        severities={
            "max_attempts_too_low": Severity.WARN,
            "timeouts_permit_retries": Severity.IGNORE,
        },
        auto_heartbeat=False,
    )
)
```

## Test

```bash
uv run pytest -v
```

The unit tests cover policy evaluation in isolation; the integration tests use the Temporal Python SDK's `WorkflowEnvironment.start_time_skipping()` to run real workflows against a worker that has the interceptor installed.

## Layout

- `activitypolicyinterceptor.py` — single-module implementation.
- `test_activitypolicyinterceptor.py` — unit + integration tests.
- `pyproject.toml` — package metadata (`temporalio`, `pytest`, `pytest-asyncio`).
