package main

import (
	"crypto/tls"
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	"os"
)

func setupTls() {
	if *certFile == "" || *keyFile == "" {
		return
	}
	clientConfig := new(tls.Config)
	clientConfig.InsecureSkipVerify = true
	clientConfig.MinVersion = tls.VersionTLS12
	cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to load keypair\t%s\n",
			err)
		os.Exit(1)
	}
	clientConfig.Certificates = append(clientConfig.Certificates, cert)
	srpc.RegisterClientTlsConfig(clientConfig)
}
