package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	"io/ioutil"
	"os"
)

func setupTls() {
	if *caFile == "" || *certFile == "" || *keyFile == "" {
		return
	}
	// Load certificates and key.
	caData, err := ioutil.ReadFile(*caFile)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to load CA file\t%s\n",
			err)
		os.Exit(1)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caData) {
		fmt.Fprintln(os.Stderr, "Unable to parse CA file")
		os.Exit(1)
	}
	// Setup server.
	serverConfig := new(tls.Config)
	serverConfig.ClientAuth = tls.RequireAndVerifyClientCert
	serverConfig.MinVersion = tls.VersionTLS12
	serverConfig.ClientCAs = caCertPool
	cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to load keypair\t%s\n",
			err)
		os.Exit(1)
	}
	serverConfig.Certificates = append(serverConfig.Certificates, cert)
	srpc.RegisterServerTlsConfig(serverConfig, true)
}
