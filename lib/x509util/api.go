/*
	Package x509util provides utility functions to process X509 certificates.
*/
package x509util

import "crypto/x509"

// GetPermittedMethods decodes the list of permitted methods in the certificate.
// The permitted methods are returned as keys in a map. An empty map indicates
// no methods are permitted. If there is a problem parsing the information an
// error is returned.
func GetPermittedMethods(cert *x509.Certificate) (map[string]struct{}, error) {
	return getPermittedMethods(cert)
}

// GetUsername decodes the username for whom the certificate was granted. It
// attests the identity of the user.
func GetUsername(cert *x509.Certificate) (string, error) {
	return getUsername(cert)
}
