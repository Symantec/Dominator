package repowatch

import (
	"bytes"
	"encoding/base64"
	"encoding/pem"
	"testing"
)

const testKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
Key: Value

MIICXAIBAAKBgQDHcTjuu3Ue8t2LsqWzOgNuRksfRfABNGf1Dz7/hlVT89El1pxw
zRltkYm3r/xA1eYcfIWKqM6yzXYMunxqAqRt0qgUFUXjmbV0Fgdm9rFSQcZtgPW/
YphIotfEJdEQSmNVC8Oba1fPKqpQ7a0Z5ZoQNnjVZXF3AU+WmHsJ/lZF0QIDAQAB
AoGAMpcVyfjjDKaua/E09vGQTTindZdX+fZBKHhlkouQuWrvcpmttS9Rc+mm9WE+
q3OWm0M63KFVTSWw/CmRxZJGAuJTbUZrRgcCymUOYAz4eiO8+4I0cNZqlJtr3fXR
hfrb4+7B4f9M05OfLrBt09D6cxvBtDv3CRj0hvcpm6wsXcECQQD7X0QXy1FO4SRY
U3ekyC4SagriIHZtJV1DghMd6ryAit38bUPM+QI76KA8e6PDrzh/tqrB9Ocd3sCE
yCeQG78HAkEAyx01Gun+q0datTDGvLaFkVCR8KK6YLMOwp9220+Z0bOGaybG+BMd
P8bULayXaePheVVDETQLk6WQqT/GvuvGZwJAbqAsbXRTIi2/OwfwvZpDfGMiywWS
WNJ6yvzxwNbPgpqauz6y+gAUVZ0496VKGxKAAOS5HYbUN7cSbt1PXAJ5/QJBALVS
y7fNMSaiup2Kf8C0iKTjcoWKICx3bTPdu/OpKj6Er/k0Uuff2Hq4+24S59EGOKFi
tk7DUZprcatGXhzOyv0CQDoBVbA41SbmBgV0pXcuaJHuctzunA9clrvczS5JGwiI
EemNmQrsBDBHPzEpYhmDf9KdeKeoLSLOWnWdmKGf7Pc=
-----END RSA PRIVATE KEY-----
`

func TestPackedKey(t *testing.T) {
	block, _ := pem.Decode([]byte(testKeyPEM))
	keyMap := map[string]string{
		"KeyType":    "RSA",
		"PrivateKey": base64.StdEncoding.EncodeToString(block.Bytes),
	}
	for key, value := range block.Headers {
		keyMap[key] = value
	}
	buffer := &bytes.Buffer{}
	if err := writeKeyAsPEM(buffer, keyMap); err != nil {
		t.Fatal(err)
	}
	if pemData := buffer.String(); pemData != testKeyPEM {
		t.Fatalf("extracted PEM: %s != test PEM: %s", pemData, testKeyPEM)
	}
}

func TestKeyWithSpaces(t *testing.T) {
	block, _ := pem.Decode([]byte(testKeyPEM))
	packedKey := base64.StdEncoding.EncodeToString(block.Bytes)
	var keyWithSpaces []byte
	for index, ch := range packedKey {
		keyWithSpaces = append(keyWithSpaces, byte(ch))
		if index%10 == 9 {
			keyWithSpaces = append(keyWithSpaces, byte(' '))
		}
	}
	keyMap := map[string]string{
		"KeyType":    "RSA",
		"PrivateKey": string(keyWithSpaces),
	}
	t.Logf("base64: %s", keyMap["PrivateKey"])
	for key, value := range block.Headers {
		keyMap[key] = value
	}
	buffer := &bytes.Buffer{}
	if err := writeKeyAsPEM(buffer, keyMap); err != nil {
		t.Fatal(err)
	}
	if pemData := buffer.String(); pemData != testKeyPEM {
		t.Fatalf("extracted PEM: %s != test PEM: %s", pemData, testKeyPEM)
	}
}
