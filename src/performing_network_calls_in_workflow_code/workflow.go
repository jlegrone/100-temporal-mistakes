package performing_network_calls_in_workflow_code

import (
	"context"
	"encoding/json"
	"net/http"

	"go.temporal.io/sdk/workflow"
)

// Config represents a configuration fetched from an external API.
type Config struct {
	Region string `json:"region"`
}

// @@@SNIPSTART performing-network-calls-bad

// BAD: network call in workflow code
func MyWorkflowV1(ctx workflow.Context) error {
	resp, err := http.Get("https://api.example.com/config")
	_ = resp
	_ = err
	// ...
	return nil
}

// @@@SNIPEND

// @@@SNIPSTART performing-network-calls-good

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

// @@@SNIPEND
