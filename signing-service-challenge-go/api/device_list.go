package api

import (
	"net/http"

	"github.com/fiskaly/coding-challenges/signing-service-challenge/null"
)

func (d *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	devices, err := d.devices.ListDevices(ctx)
	if err != nil {
		WriteError(w, err)
		return
	}

	var out ListDeviceOutputDto
	for _, device := range devices {
		singleDevice := GetDeviceOutputDto{
			Id:               device.Id.String(),
			SigningAlgorithm: device.SigningAlgorithm,
			PublicKeys:       device.PublicKeys,
			SignatureCounter: device.SignatureCounter,
		}
		if device.Label.Valid {
			singleDevice.Label = null.New(device.Label.V)
		}

		out.Items = append(out.Items, singleDevice)
	}

	WriteAPIResponse(w, http.StatusOK, out)
}

type ListDeviceOutputDto struct {
	Items []GetDeviceOutputDto `json:"items"`
}
