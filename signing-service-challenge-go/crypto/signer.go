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

func (e *ECCKeyPair) Sign(data []byte) ([]byte, error) {
	sum := sha256.Sum256(data)
	return ecdsa.SignASN1(rand.Reader, e.Private, sum[:])
}

func (r *RSAKeyPair) Sign(data []byte) ([]byte, error) {
	sum := sha256.Sum256(data)
	return rsa.SignPKCS1v15(nil, r.Private, crypto.SHA256, sum[:])
}
