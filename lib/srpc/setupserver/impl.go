package setupserver

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

var (
	caFile = flag.String("CAfile", "/etc/ssl/CA.pem",
		"Name of file containing the root of trust for identity and methods")
	certFile = flag.String("certFile",
		path.Join("/etc/ssl", getDirname(), "cert.pem"),
		"Name of file containing the SSL certificate")
	identityCaFile = flag.String("identityCAfile", "/etc/ssl/IdentityCA.pem",
		"Name of file containing the root of trust for identity only")
	keyFile = flag.String("keyFile",
		path.Join("/etc/ssl", getDirname(), "key.pem"),
		"Name of file containing the SSL key")
)

func getDirname() string {
	return path.Base(os.Args[0])
}

func setupTls(setupServer bool) error {
	// Load certificates and key.
	cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
	if err != nil {
		return fmt.Errorf("unable to load keypair: %s", err)
	}
	if setupServer {
		caData, err := ioutil.ReadFile(*caFile)
		if err != nil {
			return fmt.Errorf("unable to load CA file: \"%s\": %s",
				*caFile, err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caData) {
			return fmt.Errorf("unable to parse CA file")
		}
		serverConfig := new(tls.Config)
		serverConfig.ClientAuth = tls.RequireAndVerifyClientCert
		serverConfig.MinVersion = tls.VersionTLS12
		serverConfig.ClientCAs = caCertPool
		serverConfig.Certificates = append(serverConfig.Certificates, cert)
		if *identityCaFile != "" {
			identityCaData, err := ioutil.ReadFile(*identityCaFile)
			if err != nil {
				if !os.IsNotExist(err) {
					return fmt.Errorf("unable to load CA file: \"%s\": %s",
						*caFile, err)
				}
			} else {
				srpc.RegisterFullAuthCA(caCertPool)
				caCertPool := x509.NewCertPool()
				if !caCertPool.AppendCertsFromPEM(caData) {
					return fmt.Errorf("unable to parse CA file")
				}
				if !caCertPool.AppendCertsFromPEM(identityCaData) {
					return fmt.Errorf("unable to parse identity CA file")
				}
				serverConfig.ClientCAs = caCertPool
			}
		}
		srpc.RegisterServerTlsConfig(serverConfig, true)
	}
	// Setup client.
	clientConfig := new(tls.Config)
	clientConfig.InsecureSkipVerify = true
	clientConfig.MinVersion = tls.VersionTLS12
	clientConfig.Certificates = append(clientConfig.Certificates, cert)
	srpc.RegisterClientTlsConfig(clientConfig)
	return nil
}
