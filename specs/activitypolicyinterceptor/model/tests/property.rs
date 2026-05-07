//! End-to-end property test: run the Stateright checker over the default grid
//! and assert that every interceptor-accepted config reaches three attempts.
//!
//! When this test fails, `cargo run --release` from the same crate prints the
//! full counterexample trace.

use activitypolicyinterceptor_model::{Filter, Search};
use stateright::{Checker, Model};

/// Strict predicate must satisfy the property over the full default grid.
/// This validates that the model's closed-form derivation matches its
/// dynamic timeline simulation.
#[test]
fn strict_grid_property_holds() {
    let search = Search::default_grid_strict();
    let accepted = search
        .configs
        .iter()
        .filter(|c| Filter::StrictPermitsTwoRetries.accepts(c))
        .count();
    eprintln!("strict grid: {} accepted configs", accepted);
    let checker = search
        .checker()
        .threads(num_cpus())
        .spawn_bfs()
        .join();
    let discoveries = checker.discoveries();
    if !discoveries.is_empty() {
        for (name, path) in discoveries {
            eprintln!("--- {name} ---");
            for (state, action) in &path.into_vec() {
                eprintln!(
                    "  t={:>6}ms attempts={} phase={:?} action={:?}",
                    state.now_ms, state.attempts_started, state.phase, action,
                );
            }
        }
        panic!("strict predicate should permit ≥2 retries but counterexamples were found");
    }
}

#[test]
fn smoke_grid_finds_known_boundary() {
    // The smoke grid contains the 21s/10s case from the spec analysis.
    // We don't assert pass/fail here — the test exists so CI surfaces a
    // discovery summary even when the default grid is too large.
    let checker = Search::smoke_grid()
        .checker()
        .threads(1)
        .spawn_bfs()
        .join();
    println!(
        "smoke: states={} discoveries={}",
        checker.unique_state_count(),
        checker.discoveries().len(),
    );
}

#[test]
fn default_grid_property() {
    let checker = Search::default_grid()
        .checker()
        .threads(num_cpus())
        .spawn_bfs()
        .join();

    let discoveries = checker.discoveries();
    if !discoveries.is_empty() {
        let count = discoveries.len();
        for (name, path) in discoveries {
            eprintln!("--- {name} ---");
            let trace = path.into_vec();
            for (state, action) in &trace {
                eprintln!(
                    "  t={:>6}ms attempts={} phase={:?} action={:?}",
                    state.now_ms, state.attempts_started, state.phase, action,
                );
            }
            let last = &trace.last().expect("non-empty path").0;
            eprintln!(
                "  config: schedule_to_close={}ms, start_to_close={}ms, heartbeat={}ms, \
                 initial={}ms, max_interval={}ms, backoff={:.1}, max_attempts={}",
                last.config.schedule_to_close_ms,
                last.config.start_to_close_ms,
                last.config.heartbeat_ms,
                last.config.initial_interval_ms,
                last.config.max_interval_ms,
                last.config.backoff_coeff_x10 as f64 / 10.0,
                last.config.max_attempts,
            );
        }
        panic!("property violated: {count} counterexample(s) found");
    }
}

fn num_cpus() -> usize {
    std::thread::available_parallelism()
        .map(|n| n.get())
        .unwrap_or(1)
}
