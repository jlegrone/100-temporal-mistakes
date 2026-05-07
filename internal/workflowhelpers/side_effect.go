package workflowhelpers

import (
	"go.temporal.io/sdk/workflow"
)

// SideEffect executes the provided function once and records its result into
// the workflow history. On replay, the recorded result is returned without
// executing the function again.
//
// Common use cases: generating a random number, UUID, or reading a
// non-deterministic value. The only way to fail SideEffect is to panic,
// which causes a workflow task failure.
//
// Caution: do not use SideEffect to modify closures. Always retrieve the
// result from the return value.
func SideEffect[T any](ctx workflow.Context, f func(workflow.Context) T) (T, error) {
	var result T
	err := workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
		return f(ctx)
	}).Get(&result)
	return result, err
}

// MutableSideEffect executes the provided function once, then looks up the
// history for a value with the given id. If there is no existing value, it
// records the function result. If there is an existing value and it equals
// the new result, it returns the existing value without recording. Otherwise,
// it records the new value.
//
// During replay, the function is not executed; the recorded value is returned.
//
// One good use case is accessing dynamically changing config without breaking
// determinism. For example, LookupEnv uses MutableSideEffect to access
// environment variables.
func MutableSideEffect[T comparable](ctx workflow.Context, id string, f func(workflow.Context) T) (T, error) {
	var result T
	err := workflow.MutableSideEffect(ctx, id,
		func(ctx workflow.Context) interface{} {
			return f(ctx)
		}, func(a, b interface{}) bool {
			return a == b
		},
	).Get(&result)
	return result, err
}
