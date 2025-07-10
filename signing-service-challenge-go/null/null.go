package null

import (
	"database/sql"
	"encoding/json"
	"errors"
)

var (
	_ json.Unmarshaler = (*Null[string])(nil)
	_ json.Marshaler   = (*Null[string])(nil)
)

type Null[T any] struct {
	value  T
	filled bool
}

func New[T any](value T) Null[T] {
	return Null[T]{
		value:  value,
		filled: true,
	}
}

func Empty[T any]() Null[T] {
	return Null[T]{}
}

func (n Null[T]) Some() T {
	return n.value
}

func (n Null[T]) Value() (T, bool) {
	return n.value, n.filled
}

func (n Null[T]) Filled() bool {
	return n.filled
}

func (n Null[T]) Expect(errMsg string) (T, error) {
	if !n.filled {
		return n.value, errors.New(errMsg)
	}
	return n.value, nil
}

func (n Null[T]) SqlNull() sql.Null[T] {
	return sql.Null[T]{
		V:     n.value,
		Valid: n.filled,
	}
}

func (n Null[T]) IsZero() bool {
	return !n.filled
}

func (n Null[T]) MarshalJSON() ([]byte, error) {
	if !n.filled {
		return []byte("null"), nil
	}
	return json.Marshal(n.value)
}

func (n *Null[T]) UnmarshalJSON(bytes []byte) error {
	if string(bytes) == "null" {
		return nil
	}
	var v T
	if err := json.Unmarshal(bytes, &v); err != nil {
		return err
	}
	n.value = v
	n.filled = true
	return nil
}
