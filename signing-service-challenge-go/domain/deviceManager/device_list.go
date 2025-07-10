package deviceManager

import (
	"context"
	"log/slog"

	"github.com/fiskaly/coding-challenges/signing-service-challenge/domain"
)

func (h *Handler) ListDevices(ctx context.Context) ([]*domain.Device, error) {
	deviceRepository := h.storage.Devices()

	devices, err := deviceRepository.List(ctx, domain.DeviceFilter{})
	if err != nil {
		slog.Error("failed fetching devices", "error", err)
		return nil, err
	}

	return devices, nil
}
