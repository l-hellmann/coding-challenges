package api

import (
	"errors"
	"net/http"

	"github.com/fiskaly/coding-challenges/signing-service-challenge/domain"
	"github.com/fiskaly/coding-challenges/signing-service-challenge/domain/deviceManager"
	"github.com/fiskaly/coding-challenges/signing-service-challenge/null"
	"github.com/google/uuid"
)

type PostDeviceInputDto struct {
	// it would probably be better to generate uuid ourselves
	Id               null.Null[string]       `json:"id,omitzero"`
	SigningAlgorithm domain.SigningAlgorithm `json:"signing_algorithm"`
	Label            null.Null[string]       `json:"label,omitzero"`
}

func (d PostDeviceInputDto) Validate() error {
	var validationErr error
	if id, filled := d.Id.Value(); filled {
		validationErr = errors.Join(
			validationErr,
			uuid.Validate(id),
		)
	}
	validationErr = errors.Join(
		validationErr,
		d.SigningAlgorithm.Validate(),
	)
	return validationErr
}

func (d *DeviceHandler) Post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dto, success := ParseBody[PostDeviceInputDto](ctx, w, r.Body)
	if !success {
		return
	}

	newDevice, err := d.devices.CreateDevice(ctx, deviceManager.NewDevice{
		Id:               dto.Id,
		Label:            dto.Label,
		SigningAlgorithm: dto.SigningAlgorithm,
	})
	if err != nil {
		WriteError(w, err)
		return
	}

	out := PostDeviceOutputDto{
		Id:               newDevice.Id.String(),
		SigningAlgorithm: newDevice.SigningAlgorithm,
		PublicKeys:       newDevice.PublicKeys,
	}
	if newDevice.Label.Valid {
		out.Label = null.New(newDevice.Label.V)
	}

	WriteAPIResponse(w, http.StatusCreated, out)
}

type PostDeviceOutputDto struct {
	Id               string                  `json:"id"`
	SigningAlgorithm domain.SigningAlgorithm `json:"signing_algorithm"`
	Label            null.Null[string]       `json:"label,omitzero"`
	PublicKeys       []string                `json:"public_keys"`
}
