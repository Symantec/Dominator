package x509util

import (
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"strings"

	"github.com/Cloud-Foundations/Dominator/lib/constants"
)

func getList(cert *x509.Certificate, oid string) (map[string]struct{}, error) {
	list := make(map[string]struct{})
	for _, extension := range cert.Extensions {
		if extension.Id.String() != oid {
			continue
		}
		var lines []string
		rest, err := asn1.Unmarshal(extension.Value, &lines)
		if err != nil {
			return nil, err
		}
		if len(rest) > 0 {
			return nil, fmt.Errorf("%d extra bytes in extension", len(rest))
		}
		for _, line := range lines {
			list[line] = struct{}{}
		}
		return list, nil
	}
	return list, nil
}

func getPermittedMethods(cert *x509.Certificate) (map[string]struct{}, error) {
	methodList, err := getList(cert, constants.PermittedMethodListOID)
	if err != nil {
		return nil, fmt.Errorf("error getting method list: %s", err)
	}
	for method := range methodList {
		if strings.Count(method, ".") != 1 {
			return nil, fmt.Errorf("bad line: \"%s\"", method)
		}
	}
	return methodList, nil
}
