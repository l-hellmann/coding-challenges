package crypto

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
)

// ECCKeyPair is a DTO that holds ECC private and public keys.
type ECCKeyPair struct {
	Public  *ecdsa.PublicKey
	Private *ecdsa.PrivateKey
}

func (e *ECCKeyPair) MarshalKeyPair() ([]byte, []byte, error) {
	privateKeyBytes, err := x509.MarshalECPrivateKey(e.Private)
	if err != nil {
		return nil, nil, err
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(e.Public)
	if err != nil {
		return nil, nil, err
	}

	encodedPrivate := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE_KEY",
		Bytes: privateKeyBytes,
	})

	encodedPublic := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC_KEY",
		Bytes: publicKeyBytes,
	})

	return encodedPublic, encodedPrivate, nil
}

func (e *ECCKeyPair) UnmarshalPrivateKey(privateKeyBytes []byte) error {
	block, _ := pem.Decode(privateKeyBytes)
	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return err
	}

	e.Private = privateKey
	e.Public = &privateKey.PublicKey
	return nil
}
