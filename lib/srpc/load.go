package srpc

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Symantec/Dominator/lib/format"
)

func loadCertificates(directory string) ([]tls.Certificate, error) {
	dir, err := os.Open(directory)
	if err != nil {
		return nil, err
	}
	names, err := dir.Readdirnames(0)
	defer dir.Close()
	if err != nil {
		return nil, err
	}
	certs := make([]tls.Certificate, 0, len(names)/2)
	now := time.Now()
	for _, keyName := range names {
		if !strings.HasSuffix(keyName, ".key") {
			continue
		}
		certName := keyName[:len(keyName)-3] + "cert"
		cert, err := tls.LoadX509KeyPair(
			path.Join(directory, certName),
			path.Join(directory, keyName))
		if err != nil {
			return nil, fmt.Errorf("unable to load keypair: %s", err)
		}
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return nil, err
		}
		if notYet := x509Cert.NotBefore.Sub(now); notYet > 0 {
			return nil, fmt.Errorf("%s will not be valid for %s",
				certName, format.Duration(notYet))
		}
		if expired := now.Sub(x509Cert.NotAfter); expired > 0 {
			return nil, fmt.Errorf("%s expired %s ago",
				certName, format.Duration(expired))
		}
		certs = append(certs, cert)
	}
	if len(certs) < 1 {
		return nil, nil
	}
	return certs, nil
}
