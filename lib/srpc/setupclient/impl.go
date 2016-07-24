package setupclient

import (
	"crypto/tls"
	"flag"
	"github.com/Symantec/Dominator/lib/srpc"
	"os"
	"path"
)

var (
	certDirectory = flag.String("certDirectory",
		path.Join(os.Getenv("HOME"), ".ssl"),
		"Name of directory containing user SSL certificates")
)

func setupTls(ignoreMissingCerts bool) error {
	if *certDirectory == "" {
		return nil
	}
	// Load certificates.
	certs, err := srpc.LoadCertificates(*certDirectory)
	if ignoreMissingCerts && os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	// Setup client.
	clientConfig := new(tls.Config)
	clientConfig.InsecureSkipVerify = true
	clientConfig.MinVersion = tls.VersionTLS12
	clientConfig.Certificates = certs
	srpc.RegisterClientTlsConfig(clientConfig)
	return nil
}
