package crypto

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

// RSAKeyPair is a DTO that holds RSA private and public keys.
type RSAKeyPair struct {
	Public  *rsa.PublicKey
	Private *rsa.PrivateKey
}

func (r *RSAKeyPair) MarshalKeyPair() ([]byte, []byte, error) {
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(r.Private)
	publicKeyBytes := x509.MarshalPKCS1PublicKey(r.Public)

	encodedPrivate := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA_PRIVATE_KEY",
		Bytes: privateKeyBytes,
	})

	encodePublic := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA_PUBLIC_KEY",
		Bytes: publicKeyBytes,
	})

	return encodePublic, encodedPrivate, nil
}

func (r *RSAKeyPair) UnmarshalPrivateKey(privateKeyBytes []byte) error {
	block, _ := pem.Decode(privateKeyBytes)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return err
	}

	r.Private = privateKey
	r.Public = &privateKey.PublicKey
	return nil
}
