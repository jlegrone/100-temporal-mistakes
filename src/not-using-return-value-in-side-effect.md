# Not Using the Return Value of SideEffect

> [!TIP]
> * `SideEffect` records its result in history on first execution and returns the recorded value on [replay](terms/replay.md) -- the function is *not* re-executed.
> * If you ignore the return value and rely on the function's side effects (writing to a variable, calling an external service), that logic will re-execute on every replay, defeating the purpose.
> * Always use the value returned by `SideEffect`, not any side effects of the function you pass to it.

## What?

`SideEffect` is a Temporal SDK primitive designed for capturing small, non-deterministic values inside workflow code -- things like generating a UUID, reading the current time, or picking a random number. It works by:

1. **First execution**: Running the provided function, recording the result in history.
2. **Replay**: Returning the recorded result *without* running the function again.

The mistake happens when developers use `SideEffect` to "wrap" some non-deterministic logic but then ignore the return value and instead rely on what the function itself did. For example:

```go
// WRONG: ignoring the return value
var myUUID string
workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
    myUUID = uuid.New().String() // setting an external variable as a side effect
    return nil
})
// myUUID is set during first execution, but during replay the function
// doesn't run -- myUUID stays empty.
```

## Why?

During the initial workflow execution, this appears to work perfectly. The function runs, the variable gets set, and the workflow proceeds. The bug only surfaces during [replay](terms/replay.md):

- The `SideEffect` function is **not** re-executed on replay. The SDK returns the recorded value (which in the broken example above is `nil`).
- The variable that was set as a side effect of the function retains its zero value.
- The workflow now behaves differently than it did during the original execution, potentially causing non-determinism errors or silent logic bugs.

This is particularly insidious because it passes all testing that doesn't involve replay.

## How?

Always use the return value from `SideEffect` as the source of truth:

```go
// CORRECT: using the returned value
var myUUID string
encodedValue := workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
    return uuid.New().String()
})
err := encodedValue.Get(&myUUID)
if err != nil {
    // handle error
}
// myUUID is correctly set during both first execution and replay.
```

The same principle applies across all SDKs. In TypeScript:

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

If the logic you need to run has genuine side effects (calling an external API, writing to a database), it doesn't belong in `SideEffect` at all -- use an activity instead. `SideEffect` is strictly for capturing small, non-deterministic *values*.
