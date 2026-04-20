# Not Using the Return Value of SideEffect

> [!TIP]
> `SideEffect` records its result in [history](terms/event-history.md) on first execution and returns the recorded value on [replay](terms/replay.md) -- if you ignore the return value and rely on the function's side effects, that logic won't re-execute on replay.

`SideEffect` captures small non-deterministic values (UUIDs, random numbers) inside workflow code. It runs the provided function once, records the result, and returns the recorded value on replay without running the function again. The mistake happens when developers rely on what the function *did* rather than what it *returned*:

```go
// WRONG: ignoring the return value
var myUUID string
workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
    myUUID = uuid.New().String() // Sets variable as a side effect
    return nil
})
// During replay, the function doesn't run -- myUUID stays empty!

// CORRECT: using the returned value
var myUUID string
encodedValue := workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
    return uuid.New().String()
})
encodedValue.Get(&myUUID)
// myUUID is correctly set during both first execution and replay
```

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
