# Storing Sensitive Data in Workflow History

<!-- TODO: Note Temporal payload encryption and/or external storage as potential workarounds for this problem. Find an official temporal doc link for both approaches (https://github.com/temporalio/features/issues/765 is the issue for external payload storage, but there's probably a better doc link). -->

> [!TIP]
> [Workflow history](../terms/event-history.md) is persisted in the [Temporal server backend](../terms/temporal-server-backend.md) and visible through the UI and API. Sensitive data (PII, credentials, financial data) in history creates security and compliance risks.

Everything that flows through a Temporal workflow is persisted: workflow inputs and outputs, activity inputs and outputs, [signal](../terms/signals.md) and [update](../terms/updates.md) payloads, [query](../terms/queries.md) results, [search attributes](../terms/search-attributes.md), and memo fields. This data is accessible via the Temporal UI, CLI, and API to anyone with [namespace](../terms/namespace.md) permissions -- typically operations teams, developers, and on-call engineers, which is far broader access than your production databases. Passing sensitive information (passwords, API keys, PII, credit card numbers, health records) directly as workflow or activity parameters creates compliance risks under regulations like GDPR, HIPAA, and PCI-DSS.

The most robust approach is to keep sensitive data out of Temporal entirely: store it in a system with proper access controls (a secrets manager, an encrypted database) and pass only references through workflows:

<!--SNIPSTART storing-sensitive-data-good-->
[storing_sensitive_data_in_workflow_history/client.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/storing_sensitive_data_in_workflow_history/client.go)
```go

// Good: pass a reference, not the data
type ProcessPaymentInputGood struct {
	PaymentTokenID string // Reference to payment details stored in a vault
	OrderID        string
	Amount         int64
}

```
<!--SNIPEND-->

<!--SNIPSTART storing-sensitive-data-bad-->
[storing_sensitive_data_in_workflow_history/client.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/storing_sensitive_data_in_workflow_history/client.go)
```go

// Bad: pass the sensitive data directly
type ProcessPaymentInputBad struct {
	CreditCardNumber string
	CVV              string
	OrderID          string
	Amount           int64
}

```
<!--SNIPEND-->

For defense in depth, also configure a custom [data converter](../terms/data-converter.md) with encryption so that [payloads](../terms/payload.md) are stored as encrypted blobs on the server. The UI shows encrypted data unless configured with a codec server that can decrypt it. Combining both approaches -- minimizing sensitive data flowing through Temporal and encrypting what remains -- reduces the blast radius if either layer is ever compromised.
