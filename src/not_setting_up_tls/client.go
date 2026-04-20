package not_setting_up_tls

import (
	"crypto/tls"
	"crypto/x509"

	"go.temporal.io/sdk/client"
)

// @@@SNIPSTART not-setting-up-tls-good

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

// @@@SNIPEND
