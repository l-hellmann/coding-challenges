package deviceManager

import (
	"context"
	"errors"
	"log/slog"

	"github.com/fiskaly/coding-challenges/signing-service-challenge/persistence"
	"github.com/google/uuid"
)

func (h *Handler) DeleteDevice(ctx context.Context, deviceId uuid.UUID) error {
	deviceRepository := h.storage.Devices()

	if err := deviceRepository.Delete(ctx, deviceId); err != nil {
		if errors.Is(err, persistence.ErrNotFound) {
			return nil
		}
		slog.Error("deleting device failed", "error", err)
		return err
	}

	return nil
}
