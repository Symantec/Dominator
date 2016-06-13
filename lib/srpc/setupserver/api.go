/*
	Package setupserver assists in setting up TLS credentials for a server.

	Package setupserver provides convenience functions for setting up a server
	with TLS credentials.
*/
package setupserver

// SetupTls loads client and server certificates from files and registers them
// with the lib/srpc package. The following command-line flags are registered
// with the standard flag package:
//   -caFile:   Name of file containing the root of trust
//   -certFile: Name of file containing the SSL certificate
//   -keyFile:  Name of file containing the SSL key
func SetupTls() error {
	return setupTls(true)
}

// SetupTlsClientOnly loads a client certificate from files and registers it
// with the lib/srpc package.
func SetupTlsClientOnly() error {
	return setupTls(false)
}
