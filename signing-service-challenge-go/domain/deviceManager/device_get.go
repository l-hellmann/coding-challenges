package deviceManager

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/fiskaly/coding-challenges/signing-service-challenge/api/apiError"
	"github.com/fiskaly/coding-challenges/signing-service-challenge/domain"
	"github.com/google/uuid"
)

func (h *Handler) GetDevice(ctx context.Context, deviceId uuid.UUID) (*domain.Device, error) {
	deviceRepository := h.storage.Devices()

	device, err := deviceRepository.GetByID(ctx, deviceId)
	if err != nil {
		slog.Error("failed fetching device", "error", err)
		return nil, apiError.New(http.StatusNotFound, "device not found")
	}

	return device, nil
}
