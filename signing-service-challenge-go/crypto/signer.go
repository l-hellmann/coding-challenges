// Package crypto provides cryptographic signing functionality for the signing service.
// This package implements RSA and ECDSA signing algorithms.
package crypto

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
)

// Signer defines a contract for different types of signing implementations.
type Signer interface {
	Sign(dataToBeSigned []byte) ([]byte, error)
}

// TODO: implement RSA and ECDSA signing ...

// Sign implements the Signer interface for ECC (Elliptic Curve) key pairs
// It creates a SHA256 hash of the data and signs it using ECDSA with ASN.1 encoding
func (e *ECCKeyPair) Sign(data []byte) ([]byte, error) {
	sum := sha256.Sum256(data)
	return ecdsa.SignASN1(rand.Reader, e.Private, sum[:])
}

// Sign implements the Signer interface for RSA key pairs
// It creates a SHA256 hash of the data and signs it using RSA PKCS1v15 padding
func (r *RSAKeyPair) Sign(data []byte) ([]byte, error) {
	sum := sha256.Sum256(data)
	return rsa.SignPKCS1v15(nil, r.Private, crypto.SHA256, sum[:])
}
