package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
)

// GenerateRSAKeyPair generates a new RSAKeyPair.
func GenerateRSAKeyPair() (*RSAKeyPair, error) {
	// Security has been ignored for the sake of simplicity.
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	return &RSAKeyPair{
		Public:  &key.PublicKey,
		Private: key,
	}, nil
}

// GenerateECCKeyPair generates a new ECCKeyPair.
func GenerateECCKeyPair() (*ECCKeyPair, error) {
	// Security has been ignored for the sake of simplicity.
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, err
	}

	return &ECCKeyPair{
		Public:  &key.PublicKey,
		Private: key,
	}, nil
}
