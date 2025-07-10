package domain

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
)

// TODO: signature device domain model ...

type SigningAlgorithm string

func (s SigningAlgorithm) Validate() error {
	isValid := slices.Contains([]SigningAlgorithm{
		SigningAlgorithmEcc,
		SigningAlgorithmRsa,
	}, s)
	if !isValid {
		return errors.New("signing algorithm invalid value")
	}
	return nil
}

const (
	SigningAlgorithmEcc = SigningAlgorithm("ECC")
	SigningAlgorithmRsa = SigningAlgorithm("RSA")
)

type Device struct {
	Id               uuid.UUID // could be string
	Label            sql.Null[string]
	SigningAlgorithm SigningAlgorithm
	PrivateKey       string
	PublicKeys       []string
	SignatureCounter int
	LastSignature    sql.Null[string]
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (d *Device) Copy() *Device {
	newDevice := new(Device)

	// shallow copy of everything
	*newDevice = *d

	// manual clone of uuid
	newDevice.Id = uuid.UUID(slices.Clone(d.Id[:]))
	newDevice.PublicKeys = slices.Clone(d.PublicKeys)
	return newDevice
}

type DeviceFilter struct {
	IDs    []uuid.UUID
	Limit  int
	Offset int
}

// DeviceRepository defines the contract for device storage operations
type DeviceRepository interface {
	Create(ctx context.Context, device *Device) error
	GetByID(ctx context.Context, id uuid.UUID) (*Device, error)
	List(ctx context.Context, filter DeviceFilter) ([]*Device, error)
	Update(ctx context.Context, device *Device) error
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context, filter DeviceFilter) (int64, error)
}
