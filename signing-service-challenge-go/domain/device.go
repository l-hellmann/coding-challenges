// Package domain defines the core business entities and interfaces for the signing service.
// This package contains the domain models and contracts that define the business logic.
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

// SigningAlgorithm represents the cryptographic algorithm used for signing
type SigningAlgorithm string

// Validate checks if the signing algorithm is supported
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

// Supported signing algorithms
const (
	SigningAlgorithmEcc = SigningAlgorithm("ECC") // Elliptic Curve Cryptography
	SigningAlgorithmRsa = SigningAlgorithm("RSA") // RSA algorithm
)

// Device represents a cryptographic signing device with its associated keys and metadata
type Device struct {
	Id               uuid.UUID        // Unique identifier for the device
	Label            sql.Null[string] // Optional human-readable label
	SigningAlgorithm SigningAlgorithm // Cryptographic algorithm used for signing
	PrivateKey       string           // Private key in PEM format
	PublicKey        string           // Public key in PEM format
	SignatureCounter int              // Number of signatures created with this device
	LastSignature    sql.Null[string] // Most recent signature created
	CreatedAt        time.Time        // Device creation timestamp
	UpdatedAt        time.Time        // Last modification timestamp
}

// Copy creates a deep copy of the device to prevent unintended mutations
func (d *Device) Copy() *Device {
	newDevice := new(Device)

	// shallow copy of everything
	*newDevice = *d

	// manual clone of uuid to ensure complete independence
	newDevice.Id = uuid.UUID(slices.Clone(d.Id[:]))
	return newDevice
}

// DeviceFilter defines filtering criteria for device queries
type DeviceFilter struct {
	IDs    []uuid.UUID // Filter by specific device IDs
	Limit  int         // Maximum number of results to return
	Offset int         // Number of results to skip for pagination
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
