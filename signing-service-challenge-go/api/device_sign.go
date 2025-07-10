package api

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type PutDeviceSignInputDto struct {
	Data string `json:"data"`
}

func (d PutDeviceSignInputDto) Validate() error {
	return nil
}

func (d *DeviceHandler) Sign(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dto, success := ParseBody[PutDeviceSignInputDto](ctx, w, r.Body)
	if !success {
		return
	}

	if len(dto.Data) == 0 {
		WriteAPIResponse(w, http.StatusNoContent, nil)
		return
	}

	deviceId, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error("invalid uuid", "error", err)
		WriteErrorResponse(w, http.StatusBadRequest, "invalid uuid", err.Error())
		return
	}

	// Acquire a unique lock for the device so we can safely increment the sign counter.
	// But if it needs to be done concurrently and without care for the order of requests,
	// signing could be done without a lock, incrementing the sign counter with a channel.
	lock, err := d.locker.Acquire(ctx, deviceId)
	if err != nil {
		slog.Error("unable to acquire lock", "error", err)
		WriteInternalError(w)
		return
	}
	defer lock.Unlock()

	signedData, err := d.devices.SignData(ctx, deviceId, dto.Data)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteAPIResponse(w, http.StatusOK,
		PutDeviceSignOutputDto{
			Signature: signedData.Signature,

			// Change the order of parts, placing the last signature before the data to be signed, for better parsing.
			// The data could contain underscores, but we know only 2 are part of formatting,
			// therefore, others must be part of the data.
			SignedData: fmt.Sprintf("%d_%s_%s", signedData.SignatureCounter, signedData.LastSignature, signedData.Data),
		},
	)
}

type PutDeviceSignOutputDto struct {
	Signature  string `json:"signature"`
	SignedData string `json:"signed_data"`
}
