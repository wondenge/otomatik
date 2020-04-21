package otomatik

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"testing"
)

func TestEncodeDecodeRSAPrivateKey(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 128) // make tests faster; small key size OK for testing
	if err != nil {
		t.Fatal(err)
	}

	// test save
	savedBytes, err := encodePrivateKey(privateKey)
	if err != nil {
		t.Fatal("error saving private key:", err)
	}

	// test load
	loadedKey, err := decodePrivateKey(savedBytes)
	if err != nil {
		t.Error("error loading private key:", err)
	}

	// verify loaded key is correct
	if !privateKeysSame(privateKey, loadedKey) {
		t.Error("Expected key bytes to be the same, but they weren't")
	}
}

func TestSaveAndLoadECCPrivateKey(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// test save
	savedBytes, err := encodePrivateKey(privateKey)
	if err != nil {
		t.Fatal("error saving private key:", err)
	}

	// test load
	loadedKey, err := decodePrivateKey(savedBytes)
	if err != nil {
		t.Error("error loading private key:", err)
	}

	// verify loaded key is correct
	if !privateKeysSame(privateKey, loadedKey) {
		t.Error("Expected key bytes to be the same, but they weren't")
	}
}

// privateKeysSame compares the bytes of a and b and returns true if they are the same.
func privateKeysSame(a, b crypto.PrivateKey) bool {
	return bytes.Equal(privateKeyBytes(a), privateKeyBytes(b))
}

// privateKeyBytes returns the bytes of DER-encoded key.
func privateKeyBytes(key crypto.PrivateKey) []byte {
	var keyBytes []byte
	switch key := key.(type) {
	case *rsa.PrivateKey:
		keyBytes = x509.MarshalPKCS1PrivateKey(key)
	case *ecdsa.PrivateKey:
		keyBytes, _ = x509.MarshalECPrivateKey(key)
	}
	return keyBytes
}