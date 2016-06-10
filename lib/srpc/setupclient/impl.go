package setupclient

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	"os"
	"path"
)

var (
	certFile = flag.String("certFile",
		path.Join(os.Getenv("HOME"), ".ssl/cert.pem"),
		"Name of file containing the user SSL certificate")
	keyFile = flag.String("keyFile",
		path.Join(os.Getenv("HOME"), ".ssl/key.pem"),
		"Name of file containing the user SSL key")
)

func setupTls(ignoreMissingCerts bool) error {
	// Load certificates and key.
	cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
	if ignoreMissingCerts && os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("unable to load keypair: %s", err)
	}
	// Setup client.
	clientConfig := new(tls.Config)
	clientConfig.InsecureSkipVerify = true
	clientConfig.MinVersion = tls.VersionTLS12
	clientConfig.Certificates = append(clientConfig.Certificates, cert)
	srpc.RegisterClientTlsConfig(clientConfig)
	return nil
}
