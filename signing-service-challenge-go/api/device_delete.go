package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (d *DeviceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	deviceId, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error("invalid uuid", "error", err)
		WriteErrorResponse(w, http.StatusBadRequest, "invalid uuid", err.Error())
		return
	}

	// lock here to ensure we don't delete the device while signing is in progress
	lock, err := d.locker.Acquire(ctx, deviceId)
	if err != nil {
		slog.Error("unable to acquire lock", "error", err)
		WriteInternalError(w)
		return
	}
	defer lock.Unlock()

	if err := d.devices.DeleteDevice(ctx, deviceId); err != nil {
		WriteError(w, err)
		return
	}

	WriteAPIResponse(w, http.StatusOK, nil)
}
