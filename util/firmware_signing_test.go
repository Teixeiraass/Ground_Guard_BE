package util

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeKey(t *testing.T, curve elliptic.Curve) (string, *ecdsa.PrivateKey) {
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "key.pem")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}), 0o600))
	return path, key
}

func TestFirmwareSignatureVerifiesWithPublicKey(t *testing.T) {
	path, original := writeKey(t, elliptic.P256())
	key, err := LoadFirmwareSigningKey(path)
	require.NoError(t, err)

	digest := sha256.Sum256([]byte("firmware image"))
	sigB64, err := SignFirmwareDigest(key, digest[:])
	require.NoError(t, err)
	sig, err := base64.StdEncoding.DecodeString(sigB64)
	require.NoError(t, err)

	require.True(t, ecdsa.VerifyASN1(&original.PublicKey, digest[:], sig))

	tampered := sha256.Sum256([]byte("evil image"))
	require.False(t, ecdsa.VerifyASN1(&original.PublicKey, tampered[:], sig))
}

func TestLoadFirmwareSigningKeyRejectsWrongCurveAndGarbage(t *testing.T) {
	path, _ := writeKey(t, elliptic.P384())
	_, err := LoadFirmwareSigningKey(path)
	require.Error(t, err)

	garbage := filepath.Join(t.TempDir(), "garbage.pem")
	require.NoError(t, os.WriteFile(garbage, []byte("not a key"), 0o600))
	_, err = LoadFirmwareSigningKey(garbage)
	require.Error(t, err)

	_, err = LoadFirmwareSigningKey(filepath.Join(t.TempDir(), "missing.pem"))
	require.Error(t, err)
}
