# Not Enabling Ringpop TLS

> [!TIP]
> * Ringpop is the protocol Temporal server nodes use for cluster membership and internal routing. It operates on a separate port from the gRPC frontend.
> * Even if you have configured TLS for client-to-server and inter-service gRPC communication, Ringpop traffic may still be unencrypted.
> * Enable Ringpop TLS separately to fully secure all inter-node communication within the Temporal cluster.

## What?

Temporal's server nodes form a cluster using Ringpop, a protocol based on SWIM (Scalable Weakly-consistent Infection-style Process Group Membership). Ringpop handles membership discovery and consistent hashing, which the server uses to route requests to the correct node (e.g., routing a [workflow task](terms/workflow-task.md) to the history service shard that owns it).

Ringpop communicates over a dedicated port (typically the service's gRPC port + 1) using TCP. This traffic is separate from gRPC inter-service communication and has its own TLS configuration. Teams commonly set up TLS for the gRPC frontend and inter-service connections but overlook Ringpop, leaving cluster membership traffic unencrypted.

## Why?

An attacker with network access to the Ringpop port could:

- **Observe cluster topology.** Ringpop messages reveal the addresses and roles of all nodes in the cluster.
- **Inject a rogue node.** Without authentication, a malicious process could join the Ringpop membership ring, disrupting request routing or intercepting internal traffic.
- **Disrupt cluster stability.** By sending crafted membership messages, an attacker could cause nodes to be incorrectly marked as failed, leading to unnecessary resharding and service disruption.

Even in a private network, defense in depth dictates that all internal cluster communication should be encrypted and authenticated. If your security posture requires TLS for gRPC traffic, the same reasoning applies to Ringpop.

## How?

Configure Ringpop TLS in the Temporal server's YAML configuration file, separately from the gRPC TLS settings. Each service (frontend, history, matching, worker) has its own Ringpop configuration section.

```yaml
global:
  membership:
    tls:
      enabled: true
      certFile: /path/to/ringpop-cert.pem
      keyFile: /path/to/ringpop-key.pem
      caFile: /path/to/ca-cert.pem
      # Require and verify client certificates from other nodes
      requireClientAuth: true
```

**Key considerations:**

- **Use dedicated certificates.** While you can reuse the same certificates as your gRPC TLS configuration, using separate certificates for Ringpop gives you finer-grained control and limits the blast radius if a certificate is compromised.
- **All nodes must be configured consistently.** Ringpop TLS is all-or-nothing within a cluster. If some nodes have TLS enabled and others don't, they won't be able to communicate and the cluster will split.
- **Plan for certificate rotation.** As with any TLS deployment, certificates expire. Ensure you have a process for rotating Ringpop certificates without cluster downtime.

See also: [Not setting up TLS](not-setting-up-tls.md) for securing client-to-server and inter-service gRPC connections.
