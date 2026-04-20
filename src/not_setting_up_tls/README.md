# Not Setting Up TLS

> [!TIP]
> By default, all Temporal SDK-to-server communication is unencrypted. In any production environment, TLS must be configured for [worker](../terms/worker.md)-to-server and client-to-server connections.

Temporal SDKs communicate with the Temporal server over gRPC. Out of the box, these connections are unencrypted -- all data flowing between your workers and the server (workflow inputs, activity outputs, [heartbeat](../terms/heartbeat.md) [payloads](../terms/payload.md), etc.) travels in plaintext. Even if you use a custom [data converter](../terms/data-converter.md) to encrypt payloads at the application level, metadata such as [workflow IDs](../terms/workflow-id.md), [task queue](../terms/task-queue.md) names, and [namespace](../terms/namespace.md) names still travel in the clear without TLS. Beyond confidentiality, TLS provides integrity (protection against tampering) and, with mutual TLS, authentication -- without mTLS, any process that can reach the Temporal server's gRPC port can start workflows, send [signals](../terms/signals.md), or [terminate](../terms/terminate.md) running workflows.

Configure TLS on all SDK clients. In Go:

<!--SNIPSTART not-setting-up-tls-good-->
[not_setting_up_tls/client.go](https://github.com/jlegrone/100-temporal-mistakes/blob/main/not_setting_up_tls/client.go)
```go

// Good: configure TLS on all SDK clients.
var (
	certPool   *x509.CertPool
	clientCert tls.Certificate
)

var ClientOptions = client.Options{
	HostPort: "temporal.example.com:7233",
	ConnectionOptions: client.ConnectionOptions{
		TLS: &tls.Config{
			RootCAs:      certPool,
			Certificates: []tls.Certificate{clientCert},
		},
	},
}

```
<!--SNIPEND-->

Use mutual TLS (mTLS) when possible -- the server verifies the client's certificate, ensuring only authorized workers and clients can connect. If you self-host Temporal, also configure TLS for inter-service communication between the frontend, history, matching, and worker services.

See also: [Not Enabling Ringpop TLS](../not-enabling-ringpop-tls.md) for the often-overlooked cluster membership protocol.
