# Not Using the Return Value of SideEffect

<!-- TODO: Include the sideeffect workflow helper function here to demonstrate how to use the type system to make this mistake more preventable in go (though it's still possible to modify variables inside the side effect callback). -->
<!-- TODO: Consider renaming this mistake to something along the lines of modifying variables that exist in the parent scope of the side effect function. -->
<!-- TODO: Double check if there is already support for generic side effect functions in the Go SDK now, or at least an open issue. -->
<!-- TODO: Could we make a linter that requires the value returned by side effect function to be read? -->

> [!TIP]
> `SideEffect` records its result in [history](terms/event-history.md) on first execution and returns the recorded value on [replay](terms/replay.md) -- if you ignore the return value and rely on the function's side effects, that logic won't re-execute on replay.

`SideEffect` captures small non-deterministic values (UUIDs, random numbers) inside workflow code. It runs the provided function once, records the result, and returns the recorded value on replay without running the function again. The mistake happens when developers rely on what the function *did* rather than what it *returned*:

<!--SNIPSTART not-using-side-effect-return-bad-->
[not_using_return_value_in_side_effect/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_using_return_value_in_side_effect/workflow.go)
```go

// WRONG: ignoring the return value
func MyWorkflowV1(ctx workflow.Context) error {
	var myUUID string
	workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
		myUUID = uuid.New().String() // Sets variable as a side effect
		return nil
	})
	// During replay, the function doesn't run -- myUUID stays empty!
	_ = myUUID
	return nil
}

```
<!--SNIPEND-->

<!--SNIPSTART not-using-side-effect-return-good-->
[not_using_return_value_in_side_effect/workflow.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_using_return_value_in_side_effect/workflow.go)
```go

// CORRECT: using the returned value
func MyWorkflowV2(ctx workflow.Context) error {
	var myUUID string
	encodedValue := workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
		return uuid.New().String()
	})
	if err := encodedValue.Get(&myUUID); err != nil {
		return err
	}
	// myUUID is correctly set during both first execution and replay
	_ = myUUID
	return nil
}

```
<!--SNIPEND-->

```typescript
// CORRECT
const myUUID = await workflow.sideEffect(() => crypto.randomUUID());

// WRONG -- relying on closure mutation
let myUUID: string;
await workflow.sideEffect(() => {
  myUUID = crypto.randomUUID();
  return undefined;
});
```

If your logic has genuine side effects (calling an API, writing to a database), it doesn't belong in `SideEffect` at all -- use an activity instead. `SideEffect` is strictly for capturing small, non-deterministic *values*.
