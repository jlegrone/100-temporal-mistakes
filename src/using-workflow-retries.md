# Using Workflow Retries

> [!TIP]
> * Retrying an entire workflow throws away all accumulated state and starts from scratch -- the opposite of what makes Temporal useful.
> * Activities already have their own [retry policies](terms/retry-policy.md); handle failures internally through activity retries, compensation, and conditional logic.
> * Workflow-level retries should be reserved for simple, short-lived workflows where there's no meaningful state to preserve.

## What?

Temporal allows you to specify a `RetryPolicy` when starting a workflow, causing the entire workflow to be retried from the beginning if it fails. This sounds reasonable at first -- after all, retries work great for activities. But workflows and activities are fundamentally different.

When an activity is retried, you're re-executing a single, typically short operation. When a workflow is retried, you're throwing away everything the workflow has done so far -- every completed activity, every side effect, every decision -- and starting over from scratch. If your workflow processed 99 out of 100 items and then failed, a workflow retry will start again from item 1.

## Why?

The entire value proposition of Temporal is that workflows are durable. They survive process crashes, server failures, and network outages. The workflow history captures every step, and [replay](terms/replay.md) restores the workflow to exactly where it left off. Workflow retries throw all of that away.

This creates several problems:

**Wasted work.** Everything the workflow accomplished before failing is discarded. Activities that called external services, sent emails, or modified databases ran for nothing (or worse, will run again on retry, potentially causing duplicates).

**Non-idempotent side effects.** If your workflow triggered side effects -- payments, notifications, resource creation -- those side effects already happened. Retrying the workflow from scratch may duplicate them unless every single activity is [idempotent](terms/idempotency.md).

**Compounding failure.** If the workflow failed due to a bug in workflow logic rather than a transient issue, retrying it will just fail the same way. You burn through retry attempts while accomplishing nothing.

The right approach is to handle failures within the workflow itself. Use activity retry policies for transient failures. Use try/catch blocks and compensation logic for business-level failures. Use conditional branching to skip already-completed steps. This is what Temporal was designed for.

## How?

**Remove workflow-level retry policies** from most of your workflows. The default behavior (no retries) is almost always what you want.

**Push retry logic down to activities.** Each activity should have its own `RetryPolicy` tuned to the operation it performs. A call to an external API might retry with exponential backoff. A database write might retry a few times with short intervals.

**Implement compensation for partial failures.** If step 3 of your workflow fails and you need to undo steps 1 and 2, write explicit compensation logic in your workflow. This is the Saga pattern, and it's a natural fit for Temporal workflows.

**Reserve workflow retries for truly simple cases.** A workflow that calls a single activity, does no branching, and has no side effects beyond that activity? A workflow retry is fine there. But if your workflow has grown beyond that, remove the retry policy and handle failures internally.
