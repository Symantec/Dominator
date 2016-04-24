package x509util

import "crypto/x509"

// GetPermittedMethods decodes the list of permitted methods in the certificate.
func GetPermittedMethods(cert *x509.Certificate) (map[string]struct{}, error) {
	return getPermittedMethods(cert)
}
