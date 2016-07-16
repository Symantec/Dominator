package srpc

import (
	"crypto/tls"
	"fmt"
	"os"
	"path"
	"strings"
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
	for _, name := range names {
		if !strings.HasSuffix(name, ".key") {
			continue
		}
		cert, err := tls.LoadX509KeyPair(
			path.Join(directory, name[:len(name)-3]+"cert"),
			path.Join(directory, name))
		if err != nil {
			return nil, fmt.Errorf("unable to load keypair: %s", err)
		}
		certs = append(certs, cert)
	}
	return certs, nil
}
