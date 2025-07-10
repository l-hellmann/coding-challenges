package crypto

import (
	"bytes"
	"io"
)

type Marshaler interface {
	MarshalKeyPair() ([]byte, []byte, error)
}

type KeyPairEncoder struct {
	publicKeyWriter  io.Writer
	privateKeyWriter io.Writer
}

func NewKeyPairEncoder(
	publicKeyWriter io.Writer,
	privateKeyWriter io.Writer,
) *KeyPairEncoder {
	return &KeyPairEncoder{
		publicKeyWriter:  publicKeyWriter,
		privateKeyWriter: privateKeyWriter,
	}
}

func (e *KeyPairEncoder) Encode(keyPair Marshaler) error {
	publicKey, privateKey, err := keyPair.MarshalKeyPair()
	if err != nil {
		return err
	}
	if _, err := io.Copy(e.privateKeyWriter, bytes.NewReader(privateKey)); err != nil {
		return err
	}
	if _, err := io.Copy(e.publicKeyWriter, bytes.NewReader(publicKey)); err != nil {
		return err
	}
	return nil
}

func EncodeKeyPair(keyPair KeyPair) ([]byte, []byte, error) {
	pubKeyBuf, privKeyBuf := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	err := NewKeyPairEncoder(pubKeyBuf, privKeyBuf).Encode(keyPair)
	if err != nil {
		return nil, nil, err
	}
	return pubKeyBuf.Bytes(), privKeyBuf.Bytes(), err
}

type PublicKeyEncoder struct {
	w io.Writer
}

func NewPublicKeyEncoder(
	writer io.Writer,
) *PublicKeyEncoder {
	return &PublicKeyEncoder{
		w: writer,
	}
}

func (e *PublicKeyEncoder) Encode(keyPair Marshaler) error {
	publicKey, _, err := keyPair.MarshalKeyPair()
	if err != nil {
		return err
	}
	if _, err := io.Copy(e.w, bytes.NewReader(publicKey)); err != nil {
		return err
	}
	return nil
}

func EncodePublicKey(keyPair KeyPair) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := NewPublicKeyEncoder(buf).Encode(keyPair)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type PrivateKeyEncoder struct {
	w io.Writer
}

func NewPrivateKeyEncoder(
	writer io.Writer,
) *PrivateKeyEncoder {
	return &PrivateKeyEncoder{
		w: writer,
	}
}

func (e *PrivateKeyEncoder) Encode(keyPair Marshaler) error {
	_, privateKey, err := keyPair.MarshalKeyPair()
	if err != nil {
		return err
	}
	if _, err := io.Copy(e.w, bytes.NewReader(privateKey)); err != nil {
		return err
	}
	return nil
}

func EncodePrivateKey(keyPair KeyPair) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	err := NewPrivateKeyEncoder(buf).Encode(keyPair)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
