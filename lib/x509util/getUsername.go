package x509util

import (
	"crypto/x509"
)

func getUsername(cert *x509.Certificate) (string, error) {
	return cert.Subject.CommonName, nil
}
