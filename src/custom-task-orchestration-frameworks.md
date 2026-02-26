# Building Custom Task Orchestration Frameworks on Top of Temporal

> [!TIP]
> * Avoid building generic DSL engines or workflow-of-workflow abstractions that replicate what Temporal already provides.
> * Custom orchestration layers add complexity, bugs, and maintenance burden while obscuring Temporal's built-in capabilities.
> * If Temporal's programming model doesn't suit your needs, evaluate existing declarative workflow solutions before building your own.

## What?

Teams adopting Temporal sometimes build an abstraction layer on top of it: a custom DSL, a YAML/JSON-driven workflow engine, or a generic "task graph executor" that takes a configuration describing steps and dependencies, then translates it into Temporal workflow and activity calls.

These frameworks typically aim to:
- Let non-developers define workflows through configuration
- Provide a "simpler" interface than Temporal's SDK
- Standardize workflow patterns across an organization
- Enable dynamic workflow creation from user input

The result is often a meta-orchestrator -- a workflow that reads a task graph from a config and executes it step by step, essentially reimplementing workflow orchestration on top of an orchestration engine.

## Why?

The fundamental problem is that these custom frameworks end up reimplementing (poorly) what Temporal already provides:

1. **Duplicated functionality**: Step sequencing, parallel execution, retry logic, error handling, conditional branching -- Temporal's SDK already handles all of these. A custom framework rewrites them in a less tested, less optimized way.
2. **Loss of type safety**: Configuration-driven approaches replace compile-time checked workflow code with runtime-interpreted strings and maps. Bugs that the compiler would catch now surface in production.
3. **Debugging nightmares**: When something goes wrong, you're debugging your framework's interpretation of a config, not a straightforward workflow. Stack traces point to your generic executor, not to the business logic that failed.
4. **Maintenance burden**: The framework becomes a critical piece of infrastructure that needs ongoing investment. Every Temporal SDK update, every new feature, every edge case in workflow execution needs to be accounted for in your abstraction layer.
5. **Impedance mismatch**: Temporal's programming model is intentionally code-first. Trying to hide the code behind a configuration layer often means you can't leverage Temporal's most powerful features (complex branching, dynamic activity selection, signals, queries) without making your DSL increasingly complex.

The irony is that the framework usually starts simple ("just define your steps in YAML!") and grows in complexity until it's essentially a worse version of the Temporal SDK.

## How?

Before building a custom orchestration layer, consider these alternatives:

### Embrace the code-first model

Temporal's power comes from expressing workflows as code. Instead of hiding this behind a DSL, invest in good workflow patterns, shared libraries, and code generation where appropriate. Teach your team to write workflows -- the learning curve pays off quickly.

### Use Temporal's existing DSL samples

Temporal provides [DSL workflow samples](https://github.com/temporalio/samples-go/tree/main/dsl) that demonstrate how to build configuration-driven workflows. If you genuinely need a declarative approach, start from these rather than building from scratch.

### Build thin, domain-specific abstractions

Instead of a generic orchestration framework, build small, focused helper functions for your specific domain:

```go
// Good: a domain-specific helper, not a generic framework
func RunETLPipeline(ctx workflow.Context, config ETLConfig) error {
    if err := Extract(ctx, config.Source); err != nil {
        return err
    }
    if err := Transform(ctx, config.Rules); err != nil {
        return err
    }
    return Load(ctx, config.Destination)
}
```

This keeps the code readable, type-safe, and directly debuggable while still providing a reusable pattern.

### Evaluate existing tools

If your requirements genuinely call for a declarative, config-driven workflow engine, look at existing solutions (Argo Workflows, Apache Airflow, Prefect) that are purpose-built for that model. Using the right tool for the job is better than bending Temporal into something it's not designed to be.
