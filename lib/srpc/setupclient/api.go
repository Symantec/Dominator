/*
	Package setupclient assists in setting up TLS credentials for a client.

	Package setupclient provides convenience functions for setting up a client
	(tool) with TLS credentials.
*/
package setupclient

// SetupTls loads a client certificate from files and registers it with the
// lib/srpc package. The following command-line flags are registered
// with the standard flag package:
//   -certFile: Name of file containing the SSL certificate
//   -keyFile:  Name of file containing the SSL key
func SetupTls(ignoreMissingCerts bool) error {
	return setupTls(ignoreMissingCerts)
}
