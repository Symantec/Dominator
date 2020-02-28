package repowatch

import (
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
)

func awsGetKey(secretId string) error {
	if secretId == "" {
		return nil
	}
	metadataClient, err := getMetadataClient()
	if err != nil {
		return err
	}
	secrets, err := getAwsSecret(metadataClient, secretId)
	if err != nil {
		return err
	}
	return writeSshKey(secrets)
}

// keyMap is mutated.
func writeKeyAsPEM(writer io.Writer, keyMap map[string]string) error {
	keyType := keyMap["KeyType"]
	if keyType == "" {
		return errors.New("no KeyType in map")
	}
	delete(keyMap, "KeyType")
	privateKeyBase64 := keyMap["PrivateKey"]
	if privateKeyBase64 == "" {
		return errors.New("no PrivateKey in map")
	}
	delete(keyMap, "PrivateKey")
	privateKey, err := base64.StdEncoding.DecodeString(
		strings.Replace(privateKeyBase64, " ", "", -1))
	if err != nil {
		return err
	}
	block := &pem.Block{
		Type:    keyType + " PRIVATE KEY",
		Headers: keyMap,
		Bytes:   privateKey,
	}
	return pem.Encode(writer, block)
}

// keyMap is mutated.
func writeSshKey(keyMap map[string]string) error {
	dirname := filepath.Join(os.Getenv("HOME"), ".ssh")
	if err := os.MkdirAll(dirname, 0700); err != nil {
		return err
	}
	var filename string
	switch keyType := keyMap["KeyType"]; keyType {
	case "DSA":
		filename = "id_dsa"
	case "RSA":
		filename = "id_rsa"
	default:
		return fmt.Errorf("unsupported key type: %s", keyType)
	}
	writer, err := fsutil.CreateRenamingWriter(filepath.Join(dirname, filename),
		fsutil.PrivateFilePerms)
	if err != nil {
		return err
	}
	if err := writeKeyAsPEM(writer, keyMap); err != nil {
		writer.Abort()
		return err
	}
	return writer.Close()
}
