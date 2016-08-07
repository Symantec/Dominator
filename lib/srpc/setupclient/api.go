/*
	Package setupclient assists in setting up TLS credentials for a client.

	Package setupclient provides convenience functions for setting up a client
	(tool) with TLS credentials.
*/
package setupclient

import (
	"flag"
	"os"
	"path"
)

var (
	certDirectory = flag.String("certDirectory",
		path.Join(os.Getenv("HOME"), ".ssl"),
		"Name of directory containing user SSL certificates")
)

// GetCertDirectory returns the directory containing the client certificates.
func GetCertDirectory() string {
	return *certDirectory
}

// SetupTls loads zero or more client certificates from files and registers them
// with the lib/srpc package. The following command-line flags are registered
// with the standard flag package:
//   -certDirectory: Name of directory containing user SSL certificates
func SetupTls(ignoreMissingCerts bool) error {
	return setupTls(ignoreMissingCerts)
}
