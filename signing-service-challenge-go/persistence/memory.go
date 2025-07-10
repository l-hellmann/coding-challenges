package persistence

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/fiskaly/coding-challenges/signing-service-challenge/domain"
	"github.com/google/uuid"
)

// TODO: in-memory persistence ...

type MemoryStorage struct {
	devices *deviceRepository
	mu      sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		devices: &deviceRepository{
			data: make(map[uuid.UUID]*domain.Device),
		},
	}
}

func (m *MemoryStorage) Devices() domain.DeviceRepository {
	return m.devices
}

func (m *MemoryStorage) WithTransaction(ctx context.Context, fn func(ctx context.Context, s Storage) error) error {
	// For in-memory storage, we can implement simple locking
	// In a real database implementation; this would start a DB transaction
	m.mu.Lock()
	defer m.mu.Unlock()

	return fn(ctx, m)
}

func (m *MemoryStorage) Health(_ context.Context) error {
	// Always healthy for in-memory storage
	return nil
}

func (m *MemoryStorage) Close() error {
	return nil
}

type deviceRepository struct {
	data map[uuid.UUID]*domain.Device
	mu   sync.RWMutex
}

func (r *deviceRepository) Create(_ context.Context, device *domain.Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if device == nil {
		return ErrInvalidInput
	}

	if _, exists := r.data[device.Id]; exists {
		return ErrAlreadyExists
	}

	now := time.Now()
	if device.CreatedAt.IsZero() {
		device.CreatedAt = now
	}
	device.UpdatedAt = now

	r.data[device.Id] = device.Copy()

	return nil
}

func (r *deviceRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	device, exists := r.data[id]
	if !exists {
		return nil, ErrNotFound
	}

	return device.Copy(), nil
}

func (r *deviceRepository) List(_ context.Context, filter domain.DeviceFilter) ([]*domain.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var devices []*domain.Device

	for _, device := range r.data {
		if r.matchesFilter(device, filter) {
			devices = append(devices, device.Copy())
		}
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].CreatedAt.Before(devices[j].CreatedAt)
	})

	start := filter.Offset
	if start > len(devices) {
		return []*domain.Device{}, nil
	}

	end := len(devices)
	if filter.Limit > 0 && start+filter.Limit < end {
		end = start + filter.Limit
	}

	return devices[start:end], nil
}

func (r *deviceRepository) Update(_ context.Context, device *domain.Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.data[device.Id]
	if !exists {
		return ErrNotFound
	}

	device.UpdatedAt = time.Now()
	device.CreatedAt = existing.CreatedAt

	r.data[device.Id] = device.Copy()

	return nil
}

func (r *deviceRepository) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.data[id]
	if !exists {
		return ErrNotFound
	}

	delete(r.data, id)

	return nil
}

func (r *deviceRepository) Count(_ context.Context, filter domain.DeviceFilter) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := int64(0)
	for _, device := range r.data {
		if r.matchesFilter(device, filter) {
			count++
		}
	}

	return count, nil
}

func (r *deviceRepository) matchesFilter(device *domain.Device, filter domain.DeviceFilter) bool {
	// Check ID filter
	if len(filter.IDs) > 0 {
		found := false
		for _, id := range filter.IDs {
			if device.Id == id {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
