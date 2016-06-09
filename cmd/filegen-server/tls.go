package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	"io/ioutil"
)

func setupTls(caFile, certFile, keyFile string) error {
	// Load certificates and key.
	caData, err := ioutil.ReadFile(caFile)
	if err != nil {
		return fmt.Errorf("unable to load CA file: \"%s\": %s", caFile, err)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caData) {
		return fmt.Errorf("unable to parse CA file")
	}
	// Setup server.
	serverConfig := new(tls.Config)
	serverConfig.ClientAuth = tls.RequireAndVerifyClientCert
	serverConfig.MinVersion = tls.VersionTLS12
	serverConfig.ClientCAs = caCertPool
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("unable to load keypair: %s", err)
	}
	serverConfig.Certificates = append(serverConfig.Certificates, cert)
	srpc.RegisterServerTlsConfig(serverConfig, true)
	return nil
}
