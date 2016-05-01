package x509util

import (
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"strings"
)

func getPermittedMethods(cert *x509.Certificate) (map[string]struct{}, error) {
	methodList := make(map[string]struct{})
	for _, extension := range cert.Extensions {
		if extension.Id.String() != constants.PermittedMethodListOID {
			continue
		}
		var lines []string
		rest, err := asn1.Unmarshal(extension.Value, &lines)
		if err != nil {
			return nil, err
		}
		if len(rest) > 0 {
			return nil, fmt.Errorf("%d extra bytes in method extension",
				len(rest))
		}
		for _, sm := range lines {
			if strings.Count(sm, ".") == 1 {
				methodList[sm] = struct{}{}
			} else {
				return nil, fmt.Errorf("bad line: \"%s\"", sm)
			}
		}
		return methodList, nil
	}
	return methodList, nil
}
