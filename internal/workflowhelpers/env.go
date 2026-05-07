package workflowhelpers

import (
	"fmt"
	"os"

	"go.temporal.io/sdk/workflow"
)

type lookupEnvResult struct {
	Value string `json:"value"`
	Found bool   `json:"found"`
}

// LookupEnv retrieves the value of the environment variable named by the key
// using a MutableSideEffect. If the variable is present in the environment,
// the value (which may be empty) is returned and the boolean is true.
// Otherwise the returned value will be empty and the boolean will be false.
//
// Always use this function rather than os.LookupEnv or os.Getenv when you
// need to access environment variables in a workflow.
func LookupEnv(ctx workflow.Context, key string) (string, bool) {
	// XXX(jlegrone): A version check is required here if we ever change the
	//                side effect ID or encoding format.
	sideEffectID := fmt.Sprintf("lookup_env:%s", key)

	result, err := MutableSideEffect(ctx, sideEffectID, func(workflow.Context) lookupEnvResult {
		//workflowcheck:ignore reason: calls non-deterministic function os.LookupEnv
		val, ok := os.LookupEnv(key)
		return lookupEnvResult{Value: val, Found: ok}
	})
	if err != nil {
		workflow.GetLogger(ctx).Error(
			"LookupEnv side effect failed",
			"error", err,
			"env_var", key,
			"side_effect_id", sideEffectID,
		)
		return "", false
	}

	return result.Value, result.Found
}
