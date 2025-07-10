package persistence

import (
	"context"
	"errors"

	"github.com/fiskaly/coding-challenges/signing-service-challenge/domain"
)

var (
	ErrNotFound      = errors.New("record not found")
	ErrAlreadyExists = errors.New("record already exists")
	ErrInvalidInput  = errors.New("invalid input")
)

// Storage handles transactions and provides repository access
type Storage interface {
	Devices() domain.DeviceRepository

	WithTransaction(ctx context.Context, fn func(ctx context.Context, s Storage) error) error

	Health(ctx context.Context) error
	Close() error
}
