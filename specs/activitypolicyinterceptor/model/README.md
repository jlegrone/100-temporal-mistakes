# Formal model: timeouts permit ≥2 retries

Stateright model that verifies whether configurations the
[activitypolicyinterceptor](../README.md) accepts allow at least two retries
(i.e., a third attempt to start) when each worker that picks up an attempt
crashes 1ms after starting it.

## TL;DR

The interceptor's current rules — spec requirement 16's `2.1 × start_to_close`
ratio (no-heartbeat case) and the unconditional pass for heartbeat-enabled
configs — **do not** guarantee ≥2 retries. The model produces concrete
counterexamples in both regimes. A closed-form predicate that *does* satisfy
the property is encoded as `Config::permits_two_retries_under_crash_at_1ms`
and exhaustively verified against the dynamic simulation over the same grid.

| Test | Result | What it shows |
|---|---|---|
| `strict_grid_property_holds` | ✅ passes | The closed-form predicate is sound (matches the dynamic simulation across 24,156 accepted configs). |
| `default_grid_property` | ❌ fails | The spec-as-written admits configs that only allow 1 attempt before the deadline. |
| `smoke_grid_finds_known_boundary` | n/a | Reproduces the boundary case used in the design notes. |

## Running the model

From the model crate directory:

```sh
cargo test --release        # property tests + unit tests
cargo run --release          # default grid; prints first counterexample trace
cargo run --release smoke    # tiny grid (~ instant); reproduces the canonical case
```

Default-grid run completes in a few seconds on a laptop (≈160k unique states
across 32k interceptor-accepted configs).

## What the model proves and what it doesn't

**Bounded verification.** The default grid samples the parameter space at
discrete steps (1s for the timeouts; specific values for retry-policy fields).
Bounds and granularity are listed in `Search::default_grid` in `src/lib.rs`.
Failure within these bounds is a real counterexample; success within these
bounds is *not* a full proof — pathological values outside the grid could
still violate the property. The property is monotone enough that the chosen
bounds give high confidence, but expanding them is straightforward.

**Crash model.** The model fires a single, deterministic crash exactly 1ms
after each attempt starts. Crashes happening later (mid-attempt) make the
property *easier* to satisfy because they consume less of the attempt's
deadline budget — so the worst case modeled is the right one.

**Timeline arithmetic.** Failure detection is modeled as taking exactly
`heartbeat_timeout` (when set) or `start_to_close_timeout` (otherwise). This
matches the upper bound the Temporal server uses; in practice detection can
fire sooner without changing the inequality.

## Counterexamples found

### Without heartbeat — spec requirement 16's 2.1× ratio is too tight

```
config: schedule_to_close=21s, start_to_close=10s, heartbeat=unset,
        initial_interval=1s, max_interval=unset, backoff=2.0, max_attempts=0

t=     0ms  Start    attempt 1 begins
t=     1ms  Crash    worker dies
t= 10000ms  Detect   start_to_close timeout fires
            Backoff  retry scheduled at t=11000ms (initial_interval=1s)
t= 11000ms  Start    attempt 2 begins
t= 11001ms  Crash
t= 21000ms  Detect   start_to_close timeout fires (deadline=21000)
            Backoff  retry would start at t=23000ms (interval=2s after backoff)
t= 21001ms  Done     attempt 3 never starts
```

Two attempts execute (1 retry), not three (2 retries).

### With heartbeat — no rule applies, configs with `initial_interval >
schedule_to_close - heartbeat_timeout` permit zero retries

```
config: schedule_to_close=11s, start_to_close=11s, heartbeat=10s,
        initial_interval=5s, max_interval=60s, backoff=2.0, max_attempts=5

t=     0ms  Start    attempt 1 begins
t=     1ms  Crash
t= 10000ms  Detect   heartbeat timeout fires
            Backoff  retry scheduled at t=15000ms (initial_interval=5s)
t= 11001ms  Done     deadline=11000 reached; attempt 2 never starts
```

One attempt executes — the heartbeat-set escape hatch in spec req 16 silently
permits configs that fail even the *one-retry* property the spec is supposed
to guarantee.

## The corrected predicate

A closed-form check that, by exhaustive comparison against the dynamic
simulation over the default grid, exactly captures the property:

```
2 × detect + interval(1) + interval(2) ≤ schedule_to_close
```

where

- `detect = heartbeat_timeout` if set, otherwise `start_to_close_timeout`
- `interval(n) = min(max_interval_or_default, initial_interval × backoff^(n-1))`

The recommended spec change for requirement 16 is to evaluate this inequality
directly (replacing both the no-heartbeat 2.1× rule *and* the unconditional
heartbeat-set bypass). Filing that change is out of scope for this model.

## File map

- `src/lib.rs` — `Config`, `interceptor_accepts`, `permits_two_retries_under_crash_at_1ms`, `Sim`, `Search`, Stateright `Model` impl, unit tests.
- `src/main.rs` — CLI driver. `cargo run --release [smoke]`.
- `tests/property.rs` — three integration tests: smoke, strict-grid (must hold), default-grid (currently fails — documents the gap).

## Tool choice

Stateright is overkill for this property — the timeline is deterministic, so
each config evolves through ~5 states with no branching. A pure simulator
plus proptest would have worked. The reason to keep Stateright is twofold:

1. The state machine encoding is more readable than a raw simulator and gives
   trace output for free.
2. The strict-vs-current grids share one model, so there is exactly one
   timeline implementation to audit.

If the property is later extended to cover concurrent activity scheduling or
worker autoscaling, Stateright's nondeterminism handling will start to pay
off. If instead the team wants stronger coverage of the parameter space than
a discrete grid allows, swap in TLA+ or Alloy.
