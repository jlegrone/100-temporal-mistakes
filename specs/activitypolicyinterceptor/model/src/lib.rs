//! Stateright model of Temporal activity timeouts + retry policy.
//!
//! Property under test: for every activity-options/retry-policy combination
//! that the activitypolicyinterceptor (specs/activitypolicyinterceptor) accepts,
//! the deterministic timeline where each worker crashes 1ms after starting an
//! attempt must allow at least three attempts (one initial + two retries) to
//! start within the schedule_to_close budget.
//!
//! Time is encoded as integer milliseconds. The backoff coefficient is encoded
//! as `coeff × 10` to keep the entire state space integer.

use stateright::{Model, Property};

/// One activity-scheduling configuration. The model quantifies over a bounded
/// grid of `Config` values via [`Search::configs`].
#[derive(Clone, Copy, Debug, Hash, PartialEq, Eq, PartialOrd, Ord)]
pub struct Config {
    pub schedule_to_close_ms: u32,
    pub start_to_close_ms: u32,
    /// 0 means "unset" (the no-heartbeat regime where requirement 16 applies).
    pub heartbeat_ms: u32,
    pub initial_interval_ms: u32,
    /// 0 means "unset"; Temporal default is 100 × initial_interval.
    pub max_interval_ms: u32,
    /// `backoff_coefficient × 10`, integer to keep state hashable.
    pub backoff_coeff_x10: u32,
    /// 0 means "unlimited" per Temporal semantics.
    pub max_attempts: u32,
}

impl Config {
    /// Mirrors the validation predicates the interceptor enforces at Error
    /// severity: requirement 8 (schedule_to_close required), requirements
    /// 12–13 (max_attempts ∈ {0} ∪ ≥3), and requirement 16 (no-heartbeat
    /// 2.1× ratio). Other policies (auto-heartbeat, local-activity rules)
    /// don't gate this property.
    pub fn interceptor_accepts(&self) -> bool {
        if self.schedule_to_close_ms == 0 {
            return false;
        }
        if self.start_to_close_ms == 0 {
            return false;
        }
        if self.start_to_close_ms > self.schedule_to_close_ms {
            return false;
        }
        if self.max_attempts == 1 || self.max_attempts == 2 {
            return false;
        }
        if self.heartbeat_ms == 0 {
            // ceil(2.1 × start_to_close)
            let min_required = ((21u64 * self.start_to_close_ms as u64) + 9) / 10;
            if (self.schedule_to_close_ms as u64) < min_required {
                return false;
            }
        } else if self.heartbeat_ms > self.start_to_close_ms {
            // Heartbeat timeout must be ≤ start-to-close to be meaningful;
            // Temporal rejects this configuration server-side.
            return false;
        }
        // Retry-policy sanity: max_interval, when set, must be ≥ initial.
        if self.max_interval_ms != 0 && self.max_interval_ms < self.initial_interval_ms {
            return false;
        }
        if self.backoff_coeff_x10 < 10 {
            // Temporal rejects backoff_coefficient < 1.0.
            return false;
        }
        true
    }

    /// Time elapsed from attempt start until Temporal detects the worker
    /// failure. With heartbeat set, detection fires within one heartbeat
    /// interval; without, only the start_to_close timeout fires.
    pub fn detection_delay_ms(&self) -> u32 {
        if self.heartbeat_ms > 0 {
            self.heartbeat_ms
        } else {
            self.start_to_close_ms
        }
    }

    /// Closed-form check of the property the model verifies dynamically:
    /// is `schedule_to_close` large enough for a *third* attempt to start,
    /// given each attempt crashes 1ms after start and uses this retry policy?
    ///
    /// Time to attempt-3 start =
    ///   detect(1) + interval(1) + detect(2) + interval(2)
    /// where `detect = detection_delay_ms()` and `interval(n)` is the retry
    /// interval after the n-th attempt. Detection at exactly the deadline
    /// counts as in-budget; backoff that *ends* at the deadline lets the
    /// next attempt start exactly there (start at deadline is allowed).
    pub fn permits_two_retries_under_crash_at_1ms(&self) -> bool {
        if !self.interceptor_accepts() {
            return false;
        }
        if self.max_attempts != 0 && self.max_attempts < 3 {
            return false;
        }
        let detect = self.detection_delay_ms() as u64;
        let interval_1 = self.retry_interval_ms(1) as u64;
        let interval_2 = self.retry_interval_ms(2) as u64;
        let third_start = 2 * detect + interval_1 + interval_2;
        third_start <= self.schedule_to_close_ms as u64
    }

    /// Retry interval scheduled *after* the Nth attempt fails (1-indexed).
    /// Mirrors Temporal: `interval = min(max_interval, initial × backoff^(N-1))`,
    /// where `max_interval` defaults to 100 × initial when unset.
    pub fn retry_interval_ms(&self, attempt_just_failed: u32) -> u32 {
        let max_interval: u64 = if self.max_interval_ms == 0 {
            (self.initial_interval_ms as u64).saturating_mul(100)
        } else {
            self.max_interval_ms as u64
        };
        let mut interval = self.initial_interval_ms as u64;
        let coeff = self.backoff_coeff_x10 as u64;
        for _ in 1..attempt_just_failed {
            interval = interval.saturating_mul(coeff) / 10;
            if interval >= max_interval {
                interval = max_interval;
                break;
            }
        }
        std::cmp::min(interval, max_interval) as u32
    }
}

#[derive(Clone, Copy, Debug, Hash, PartialEq, Eq, PartialOrd, Ord)]
pub enum Phase {
    /// First attempt has not started yet.
    Idle,
    /// Worker is running an attempt that began at `since_ms`. The model fires
    /// the crash deterministically 1ms later.
    Running { since_ms: u32 },
    /// Crash already happened; server has not yet detected failure.
    AwaitingDetection { since_ms: u32 },
    /// Failure detected; next attempt is scheduled to start at `until_ms`.
    Backoff { until_ms: u32 },
    /// Terminal state — either property satisfied (3 attempts started in time)
    /// or deadline exhausted.
    Done,
}

#[derive(Clone, Copy, Debug, Hash, PartialEq, Eq, PartialOrd, Ord)]
pub struct Sim {
    pub config: Config,
    pub now_ms: u32,
    pub attempts_started: u32,
    pub phase: Phase,
}

impl Sim {
    pub fn init(config: Config) -> Self {
        Sim {
            config,
            now_ms: 0,
            attempts_started: 0,
            phase: Phase::Idle,
        }
    }

    pub fn deadline(&self) -> u32 {
        self.config.schedule_to_close_ms
    }

    /// Once a third attempt has started, the property is satisfied — no
    /// further exploration is useful.
    pub fn satisfied(&self) -> bool {
        self.attempts_started >= 3
    }
}

/// Discrete event in the timeline. Each one advances `now_ms` to the event
/// time, so traces are easy to read in counterexamples.
#[derive(Clone, Copy, Debug, Hash, PartialEq, Eq)]
pub enum Step {
    Start,
    Crash,
    Detect,
    ScheduleRetry,
    Deadline,
}

const CRASH_DELAY_MS: u32 = 1;

/// Which acceptance predicate gates the search's initial states.
#[derive(Clone, Copy, Debug)]
pub enum Filter {
    /// Configs that the activitypolicyinterceptor accepts today (spec §16).
    /// The property fails on this filter — the search produces counterexamples.
    InterceptorCurrent,
    /// Configs that satisfy the closed-form "≥2 retries fit" rule. Demonstrates
    /// the corrected predicate the spec would need to hold.
    StrictPermitsTwoRetries,
}

impl Filter {
    pub fn accepts(self, c: &Config) -> bool {
        match self {
            Filter::InterceptorCurrent => c.interceptor_accepts(),
            Filter::StrictPermitsTwoRetries => c.permits_two_retries_under_crash_at_1ms(),
        }
    }
}

/// Stateright model. Initial states enumerate the bounded config grid; each
/// timeline evolves deterministically.
#[derive(Clone, Debug)]
pub struct Search {
    pub configs: Vec<Config>,
    pub filter: Filter,
}

impl Search {
    /// Default coarse grid — enough to surface short-`start_to_close`
    /// counterexamples while keeping the run under a minute on a laptop.
    pub fn default_grid() -> Self {
        let schedule_to_close = step_range(1_000, 30_000, 1_000);
        let start_to_close = step_range(1_000, 15_000, 1_000);
        let heartbeats = vec![0u32, 1_000, 5_000, 10_000];
        let initial_intervals = vec![100u32, 1_000, 5_000];
        let max_intervals = vec![0u32, 60_000];
        let backoffs = vec![10u32, 20];
        let max_attempts_choices = vec![0u32, 3, 5];

        let mut configs = Vec::new();
        for &stc in &schedule_to_close {
            for &sttc in &start_to_close {
                for &hb in &heartbeats {
                    for &ii in &initial_intervals {
                        for &mi in &max_intervals {
                            for &bc in &backoffs {
                                for &ma in &max_attempts_choices {
                                    configs.push(Config {
                                        schedule_to_close_ms: stc,
                                        start_to_close_ms: sttc,
                                        heartbeat_ms: hb,
                                        initial_interval_ms: ii,
                                        max_interval_ms: mi,
                                        backoff_coeff_x10: bc,
                                        max_attempts: ma,
                                    });
                                }
                            }
                        }
                    }
                }
            }
        }
        Search {
            configs,
            filter: Filter::InterceptorCurrent,
        }
    }

    /// Same grid as [`default_grid`](Search::default_grid), but gated by the
    /// strict closed-form predicate. The property must hold here.
    pub fn default_grid_strict() -> Self {
        Self {
            filter: Filter::StrictPermitsTwoRetries,
            ..Self::default_grid()
        }
    }

    /// Tiny grid for fast smoke-test runs.
    pub fn smoke_grid() -> Self {
        Search {
            filter: Filter::InterceptorCurrent,
            configs: vec![
                // Boundary case from the back-of-envelope analysis: at
                // start=10s, the 2.1× rule gives 21s but two retries need ~23s.
                Config {
                    schedule_to_close_ms: 21_000,
                    start_to_close_ms: 10_000,
                    heartbeat_ms: 0,
                    initial_interval_ms: 1_000,
                    max_interval_ms: 0,
                    backoff_coeff_x10: 20,
                    max_attempts: 0,
                },
                // Comfortable case that should pass: 30× start_to_close.
                Config {
                    schedule_to_close_ms: 60_000,
                    start_to_close_ms: 2_000,
                    heartbeat_ms: 0,
                    initial_interval_ms: 1_000,
                    max_interval_ms: 0,
                    backoff_coeff_x10: 20,
                    max_attempts: 0,
                },
            ],
        }
    }
}

fn step_range(start: u32, end_inclusive: u32, step: u32) -> Vec<u32> {
    let mut v = Vec::new();
    let mut x = start;
    while x <= end_inclusive {
        v.push(x);
        x += step;
    }
    v
}

impl Model for Search {
    type State = Sim;
    type Action = Step;

    fn init_states(&self) -> Vec<Self::State> {
        let filter = self.filter;
        self.configs
            .iter()
            .filter(|c| filter.accepts(c))
            .copied()
            .map(Sim::init)
            .collect()
    }

    fn actions(&self, state: &Self::State, actions: &mut Vec<Self::Action>) {
        if matches!(state.phase, Phase::Done) {
            return;
        }
        if state.satisfied() {
            return;
        }
        if state.now_ms > state.deadline() {
            return;
        }
        match state.phase {
            Phase::Idle => actions.push(Step::Start),
            Phase::Backoff { until_ms } => {
                let start_at = std::cmp::max(state.now_ms, until_ms);
                if start_at > state.deadline() {
                    actions.push(Step::Deadline);
                } else {
                    actions.push(Step::Start);
                }
            }
            Phase::Running { .. } => actions.push(Step::Crash),
            Phase::AwaitingDetection { since_ms } => {
                let detect_at = since_ms + state.config.detection_delay_ms();
                if detect_at > state.deadline() {
                    actions.push(Step::Deadline);
                } else {
                    actions.push(Step::Detect);
                }
            }
            Phase::Done => {}
        }
        // The `max_attempts` ceiling: if the next attempt would exceed it,
        // there's no point continuing — Temporal would not retry.
        if let Phase::Backoff { .. } = state.phase {
            if let Some(only) = actions.last() {
                if matches!(only, Step::Start)
                    && state.config.max_attempts != 0
                    && state.attempts_started >= state.config.max_attempts
                {
                    actions.clear();
                    actions.push(Step::Deadline);
                }
            }
        }
    }

    fn next_state(&self, last: &Self::State, action: Self::Action) -> Option<Self::State> {
        let mut next = *last;
        match action {
            Step::Start => match last.phase {
                Phase::Idle => {
                    next.attempts_started = last.attempts_started + 1;
                    next.phase = Phase::Running { since_ms: last.now_ms };
                }
                Phase::Backoff { until_ms } => {
                    let start_at = std::cmp::max(last.now_ms, until_ms);
                    next.now_ms = start_at;
                    next.attempts_started = last.attempts_started + 1;
                    next.phase = Phase::Running { since_ms: start_at };
                }
                _ => return None,
            },
            Step::Crash => match last.phase {
                Phase::Running { since_ms } => {
                    next.now_ms = since_ms + CRASH_DELAY_MS;
                    next.phase = Phase::AwaitingDetection { since_ms };
                }
                _ => return None,
            },
            Step::Detect => match last.phase {
                Phase::AwaitingDetection { since_ms } => {
                    next.now_ms = since_ms + last.config.detection_delay_ms();
                    let interval = last.config.retry_interval_ms(last.attempts_started);
                    let backoff_until = next.now_ms.saturating_add(interval);
                    next.phase = Phase::Backoff { until_ms: backoff_until };
                }
                _ => return None,
            },
            Step::ScheduleRetry => return None, // folded into Detect
            Step::Deadline => {
                next.now_ms = last.deadline().saturating_add(1);
                next.phase = Phase::Done;
            }
        }
        Some(next)
    }

    fn properties(&self) -> Vec<Property<Self>> {
        vec![Property::<Self>::always(
            "every accepted config permits at least two retries",
            |model, sim| {
                if !model.filter.accepts(&sim.config) {
                    // Filtered out at init_states; kept defensive.
                    return true;
                }
                if sim.satisfied() {
                    return true;
                }
                let mut next = Vec::new();
                model.actions(sim, &mut next);
                if next.is_empty() {
                    return sim.attempts_started >= 3;
                }
                true
            },
        )]
    }
}

#[cfg(test)]
mod unit {
    use super::*;

    #[test]
    fn interceptor_accepts_basic() {
        let c = Config {
            schedule_to_close_ms: 21_000,
            start_to_close_ms: 10_000,
            heartbeat_ms: 0,
            initial_interval_ms: 1_000,
            max_interval_ms: 0,
            backoff_coeff_x10: 20,
            max_attempts: 0,
        };
        assert!(c.interceptor_accepts(), "21s ≥ ceil(2.1*10s)");
    }

    #[test]
    fn interceptor_rejects_below_ratio() {
        let c = Config {
            schedule_to_close_ms: 20_999,
            start_to_close_ms: 10_000,
            heartbeat_ms: 0,
            initial_interval_ms: 1_000,
            max_interval_ms: 0,
            backoff_coeff_x10: 20,
            max_attempts: 0,
        };
        assert!(!c.interceptor_accepts(), "20.999s < ceil(2.1*10s) = 21s");
    }

    #[test]
    fn interceptor_rejects_max_attempts_two() {
        let c = Config {
            schedule_to_close_ms: 60_000,
            start_to_close_ms: 1_000,
            heartbeat_ms: 0,
            initial_interval_ms: 1_000,
            max_interval_ms: 0,
            backoff_coeff_x10: 20,
            max_attempts: 2,
        };
        assert!(!c.interceptor_accepts());
    }

    #[test]
    fn retry_interval_doubles() {
        let c = Config {
            schedule_to_close_ms: 60_000,
            start_to_close_ms: 1_000,
            heartbeat_ms: 0,
            initial_interval_ms: 1_000,
            max_interval_ms: 0,
            backoff_coeff_x10: 20,
            max_attempts: 0,
        };
        assert_eq!(c.retry_interval_ms(1), 1_000); // after attempt 1
        assert_eq!(c.retry_interval_ms(2), 2_000); // after attempt 2
        assert_eq!(c.retry_interval_ms(3), 4_000); // after attempt 3
    }

    #[test]
    fn retry_interval_capped_by_max() {
        let c = Config {
            schedule_to_close_ms: 60_000,
            start_to_close_ms: 1_000,
            heartbeat_ms: 0,
            initial_interval_ms: 1_000,
            max_interval_ms: 3_000,
            backoff_coeff_x10: 20,
            max_attempts: 0,
        };
        assert_eq!(c.retry_interval_ms(5), 3_000);
    }
}
