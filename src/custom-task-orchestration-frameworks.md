# Building Custom Task Orchestration Frameworks on Top of Temporal

> [!TIP]
> Custom orchestration layers on top of Temporal (DSL engines, YAML-driven executors, generic task graph frameworks) end up reimplementing what the SDK already provides, but with worse developer experience than other off the shelf declarative workflow engines.

Teams sometimes build an abstraction layer on top of Temporal: a custom DSL, a YAML/JSON-driven workflow engine, or a generic "task graph executor" that translates configuration into Temporal workflow and activity calls. The result is a meta-orchestrator that reimplements step sequencing, parallel execution, retry logic, and error handling -- all things the SDK already provides -- but with less type safety, worse debugging, and ongoing maintenance burden. The framework usually starts simple ("just define your steps in YAML!") and grows until it's a worse version of the Temporal SDK.

Temporal's programming model is intentionally code-first. Instead of hiding this behind a configuration layer, invest in good workflow patterns, shared libraries, and domain-specific helpers. If you genuinely need a declarative approach, start from Temporal's [DSL workflow samples](https://github.com/temporalio/samples-go/tree/main/dsl) rather than building from scratch. If your requirements truly call for a config-driven workflow engine, evaluate existing solutions (Argo Workflows, Apache Airflow, Prefect) that are purpose-built for that model.
