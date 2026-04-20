# Storing Sensitive Data in Workflow History

> [!TIP]
> [Workflow history](terms/event-history.md) is persisted in the [Temporal server backend](terms/temporal-server-backend.md) and visible through the UI and API. Sensitive data (PII, credentials, financial data) in history creates security and compliance risks.

Everything that flows through a Temporal workflow is persisted: workflow inputs and outputs, activity inputs and outputs, [signal](terms/signals.md) and [update](terms/updates.md) payloads, [query](terms/queries.md) results, [search attributes](terms/search-attributes.md), and memo fields. This data is accessible via the Temporal UI, CLI, and API to anyone with [namespace](terms/namespace.md) permissions -- typically operations teams, developers, and on-call engineers, which is far broader access than your production databases. Passing sensitive information (passwords, API keys, PII, credit card numbers, health records) directly as workflow or activity parameters creates compliance risks under regulations like GDPR, HIPAA, and PCI-DSS.

The most robust approach is to keep sensitive data out of Temporal entirely: store it in a system with proper access controls (a secrets manager, an encrypted database) and pass only references through workflows:

```go
// Good: pass a reference, not the data
type ProcessPaymentInput struct {
    PaymentTokenID string // Reference to payment details stored in a vault
    OrderID        string
    Amount         int64
}

// Bad: pass the sensitive data directly
type ProcessPaymentInput struct {
    CreditCardNumber string
    CVV              string
    OrderID          string
    Amount           int64
}
```

For defense in depth, also configure a custom [data converter](terms/data-converter.md) with encryption so that [payloads](terms/payload.md) are stored as encrypted blobs on the server. The UI shows encrypted data unless configured with a codec server that can decrypt it. Combining both approaches -- minimizing sensitive data flowing through Temporal and encrypting what remains -- reduces the blast radius if either layer is ever compromised.
