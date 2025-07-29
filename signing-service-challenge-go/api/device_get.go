package api

import (
	"log/slog"
	"net/http"

	"github.com/fiskaly/coding-challenges/signing-service-challenge/domain"
	"github.com/fiskaly/coding-challenges/signing-service-challenge/null"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (d *DeviceHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	deviceId, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		slog.Error("invalid uuid", "error", err)
		WriteErrorResponse(w, http.StatusBadRequest, "invalid uuid", err.Error())
		return
	}

	device, err := d.devices.GetDevice(ctx, deviceId)
	if err != nil {
		WriteError(w, err)
		return
	}

	out := GetDeviceOutputDto{
		Id:               device.Id.String(),
		SigningAlgorithm: device.SigningAlgorithm,
		PublicKey:        device.PublicKey,
		SignatureCounter: device.SignatureCounter,
	}
	if device.Label.Valid {
		out.Label = null.New(device.Label.V)
	}

	WriteAPIResponse(w, http.StatusOK, out)
}

type GetDeviceOutputDto struct {
	Id               string                  `json:"id"`
	SigningAlgorithm domain.SigningAlgorithm `json:"signing_algorithm"`
	Label            null.Null[string]       `json:"label,omitzero"`
	PublicKey        string                  `json:"public_key"`
	SignatureCounter int                     `json:"signature_counter"`
}
