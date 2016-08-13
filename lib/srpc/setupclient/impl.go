package setupclient

import (
	"crypto/tls"
	"github.com/Symantec/Dominator/lib/srpc"
)

func setupTls(ignoreMissingCerts bool) error {
	if *certDirectory == "" {
		return nil
	}
	// Load certificates.
	certs, err := srpc.LoadCertificates(*certDirectory)
	if err != nil {
		return err
	}
	if certs == nil {
		if ignoreMissingCerts {
			return nil
		}
		return srpc.ErrorMissingCertificate
	}
	// Setup client.
	clientConfig := new(tls.Config)
	clientConfig.InsecureSkipVerify = true
	clientConfig.MinVersion = tls.VersionTLS12
	clientConfig.Certificates = certs
	srpc.RegisterClientTlsConfig(clientConfig)
	return nil
}
