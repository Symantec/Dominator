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
	clientConfig := new(tls.Config)
	clientConfig.InsecureSkipVerify = true
	clientConfig.MinVersion = tls.VersionTLS12
	clientConfig.RootCAs = caCertPool
	cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to load keypair\t%s\n",
			err)
		os.Exit(1)
	}
	clientConfig.Certificates = append(clientConfig.Certificates, cert)
	srpc.RegisterClientTlsConfig(clientConfig)
}
