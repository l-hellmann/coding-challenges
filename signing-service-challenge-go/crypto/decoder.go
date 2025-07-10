package crypto

import (
	"bytes"
	"io"
)

type Unmarshaler interface {
	UnmarshalPrivateKey(privateKeyBytes []byte) error
}

type PrivateKeyDecoder struct {
	r io.Reader
}

func NewPrivateKeyDecoder(reader io.Reader) *PrivateKeyDecoder {
	return &PrivateKeyDecoder{r: reader}
}

func (d *PrivateKeyDecoder) Decode(keyPair Unmarshaler) error {
	bytes, err := io.ReadAll(d.r)
	if err != nil {
		return err
	}
	return keyPair.UnmarshalPrivateKey(bytes)
}

func DecodePrivateKey(p []byte, keyPair KeyPair) error {
	if err := NewPrivateKeyDecoder(bytes.NewReader(p)).Decode(keyPair); err != nil {
		return err
	}
	return nil
}
