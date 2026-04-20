# Versioning

Versioning in Temporal refers to the mechanisms for safely evolving workflow code while running workflows are in flight. Because workflow code is re-executed during replay, changes to workflow logic can break existing executions if not properly versioned.

The primary SDK-level mechanism is patching (called `GetVersion` in Go, `patched()` in TypeScript). This allows a workflow to branch between old and new code paths based on whether the workflow was started before or after the change.

Worker Versioning (also called Build ID-based versioning) is a server-side feature that routes workflow and activity tasks to workers running compatible code versions, providing an alternative to inline patching for managing deployments.

## Related

- [Not using workflow versioning](../not_using_workflow_versioning/README.md)
- [Incorrect workflow patching](../incorrect-workflow-patching.md)
- [Not validating replay safety before deployments](../not_validating_replay_safety_before_deployments/README.md)
- [Breaking changes to payloads](../breaking-changes-to-payloads.md)
- [Replay](replay.md)
- [Non-determinism](non-determinism.md)
- [Workflow Task](workflow-task.md)
- [Temporal Worker Kubernetes Controller](temporal-worker-kubernetes-controller.md)
