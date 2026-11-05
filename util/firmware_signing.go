package util

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// LoadFirmwareSigningKey reads an ECDSA P-256 private key in PEM (SEC1
// "EC PRIVATE KEY" from `openssl ecparam -genkey`, or PKCS#8).
func LoadFirmwareSigningKey(path string) (*ecdsa.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("firmware signing key: no PEM block found")
	}

	var key *ecdsa.PrivateKey
	switch block.Type {
	case "EC PRIVATE KEY":
		key, err = x509.ParseECPrivateKey(block.Bytes)
	case "PRIVATE KEY":
		var parsed any
		parsed, err = x509.ParsePKCS8PrivateKey(block.Bytes)
		if err == nil {
			var ok bool
			if key, ok = parsed.(*ecdsa.PrivateKey); !ok {
				err = errors.New("firmware signing key: not an ECDSA key")
			}
		}
	default:
		err = fmt.Errorf("firmware signing key: unsupported PEM type %q", block.Type)
	}
	if err != nil {
		return nil, err
	}
	if key.Curve != elliptic.P256() {
		return nil, errors.New("firmware signing key: must be P-256 (prime256v1)")
	}
	return key, nil
}

// SignFirmwareDigest signs the raw SHA-256 digest of a firmware image and
// returns the base64 DER signature the controller verifies with mbedtls.
func SignFirmwareDigest(key *ecdsa.PrivateKey, digest []byte) (string, error) {
	sig, err := ecdsa.SignASN1(rand.Reader, key, digest)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}
