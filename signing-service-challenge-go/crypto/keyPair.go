package crypto

type KeyPair interface {
	Signer
	Marshaler
	Unmarshaler
}
