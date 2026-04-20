# Performing Network Calls in Workflow Code

> [!TIP]
> Network calls (HTTP requests, database queries, gRPC calls) in workflow code are re-executed on every [replay](terms/replay.md), producing potentially different results each time and breaking determinism. Move all network I/O into activities.

Workflow code must be deterministic because it re-executes during replay to reconstruct state. A network call is inherently [non-deterministic](terms/non-determinism.md): it might return different data, fail differently, or time out depending on when it runs. Unlike activity calls whose results come from [history](terms/event-history.md), network calls made directly in workflow code execute again on every replay. The different result causes the workflow to take a different code path, produce different commands, or fail outright with a non-determinism error.

<!--SNIPSTART performing-network-calls-bad-->
[performing_network_calls_in_workflow_code/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/performing_network_calls_in_workflow_code/workflow.go)
```go

// BAD: network call in workflow code
func MyWorkflowV1(ctx workflow.Context) error {
	resp, err := http.Get("https://api.example.com/config")
	_ = resp
	_ = err
	// ...
	return nil
}

```
<!--SNIPEND-->

<!--SNIPSTART performing-network-calls-good-->
[performing_network_calls_in_workflow_code/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/performing_network_calls_in_workflow_code/workflow.go)
```go

// GOOD: network call in an activity
func FetchConfigActivity(ctx context.Context) (Config, error) {
	resp, err := http.Get("https://api.example.com/config")
	if err != nil {
		return Config{}, err
	}
	defer resp.Body.Close()
	var config Config
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return Config{}, err
	}
	return config, err
}

func MyWorkflowV2(ctx workflow.Context) error {
	var config Config
	if err := workflow.ExecuteActivity(ctx, FetchConfigActivity).Get(ctx, &config); err != nil {
		return err
	}
	// config is the same value on replay
	_ = config
	return nil
}

```
<!--SNIPEND-->

If you need a small piece of non-deterministic data (like a UUID) without the overhead of a full activity, use `workflow.SideEffect` -- but for anything involving network I/O, prefer activities because they come with retries, timeouts, and [heartbeating](terms/heartbeat.md).
