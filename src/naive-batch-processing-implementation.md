# Naïve Batch Processing Implementation

> [!TIP]
> * It is possible to process relatively large amount of data via Temporal by implementing batch processing workflows
> * Doing it properly requires paying attentions to Temporal limits like history size and workflow lock contention as well as minimizing data transfer and storage.

## What?

Temporal can be used to process large data sets. Usually batch processing workflows operate on list of arbitrary data, split the said list in batches and process each batch in sequence or in parallel.

## Why?

While implementing batch operations, it is quite easy to make the process suboptimal adding unnecessary latency. One of the main drivers for extra latency is [workflow lock contention](<workflow-lock-contention-due-to-concurrent-updates.md>).

It is also important, when implementing batch processing workflows to pay attention [history limits](<overflowing-workflow-history-size.md>).

## How?

One way to limit [workflow lock contention](<workflow-lock-contention-due-to-concurrent-updates.md>) is to design your batch processing workflow to handle each batch in a child workflow. The pattern can be repeated for batches of batches, batches of batches of batches, ... essentially creating a tree of workflows. Concurrency can be controlled at each level of the tree.

You should also pay attention to the data itself and remember that all activity and workflows inputs and outputs are serialized and persisted in Temporal backend. This means that each time data is passed around, it transits over the network and use disk space. If possible, compute batches using IDs referencing each batch of data and let the lower level workflows that do the actual work look up the data to process. You'll minimize the bandwidth needed to process everything.

## Alternatives

Temporal is however not a pure data plane service but more a control plane / orchestration one. You can use it to orchestrate the job performing the processing but you should avoid making the entire data transit over Temporal workflow inputs and outputs.

This means that while batch processing can be implemented on top of Temporal, high throughput, low latency data processing pipelines can only be achieved with dedicated systems and systems with less overhead like Kafka, ... at the expense or more complex workflow control and error handling code.
