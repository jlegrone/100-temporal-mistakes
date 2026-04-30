use activitypolicyinterceptor_model::{Config, Phase, Search, Sim, Step};
use stateright::{Checker, Model};

fn main() {
    let grid_kind = std::env::args().nth(1).unwrap_or_else(|| "default".to_string());
    let search = match grid_kind.as_str() {
        "smoke" => Search::smoke_grid(),
        "default" | _ => Search::default_grid(),
    };

    eprintln!(
        "Running BFS over {} configs ({} accepted by interceptor)…",
        search.configs.len(),
        search
            .configs
            .iter()
            .filter(|c| c.interceptor_accepts())
            .count(),
    );

    let checker = search
        .checker()
        .threads(num_cpus())
        .spawn_bfs()
        .join();

    eprintln!(
        "Explored {} unique states (max depth {}).",
        checker.unique_state_count(),
        checker.max_depth(),
    );

    let discoveries = checker.discoveries();
    if discoveries.is_empty() {
        println!("PROPERTY HOLDS within the configured grid.");
        return;
    }

    println!("PROPERTY FAILED — counterexamples found:");
    for (name, path) in discoveries {
        println!("\n--- discovery: {name} ---");
        let trace = path.into_vec();
        let last_state = &trace.last().expect("non-empty path").0;
        println!("config: {}", format_config(&last_state.config));
        println!("attempts_started at terminal: {}", last_state.attempts_started);
        println!(
            "now_ms at terminal: {} (deadline {})",
            last_state.now_ms,
            last_state.config.schedule_to_close_ms,
        );
        println!("trace:");
        for (state, action) in &trace {
            println!(
                "  t={:>6}ms  attempts={}  phase={:?}  -- {}",
                state.now_ms,
                state.attempts_started,
                state.phase,
                match action {
                    Some(a) => format!("{:?}", a),
                    None => "(terminal)".to_string(),
                },
            );
        }
    }
    std::process::exit(1);
}

fn format_config(c: &Config) -> String {
    format!(
        "schedule_to_close={}ms, start_to_close={}ms, heartbeat={}ms, \
         initial_interval={}ms, max_interval={}ms, backoff={:.1}, max_attempts={}",
        c.schedule_to_close_ms,
        c.start_to_close_ms,
        c.heartbeat_ms,
        c.initial_interval_ms,
        c.max_interval_ms,
        c.backoff_coeff_x10 as f64 / 10.0,
        c.max_attempts,
    )
}

fn num_cpus() -> usize {
    std::thread::available_parallelism()
        .map(|n| n.get())
        .unwrap_or(1)
}

// Re-export to keep the `Phase`/`Sim`/`Step` symbols in the binary's
// path table — useful when reading panic/Debug output of the trace.
#[allow(dead_code)]
fn _force_use(_: Phase, _: Sim, _: Step) {}
