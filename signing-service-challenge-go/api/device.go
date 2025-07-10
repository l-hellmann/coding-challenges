package api

import (
	"github.com/fiskaly/coding-challenges/signing-service-challenge/domain/deviceManager"
	"github.com/fiskaly/coding-challenges/signing-service-challenge/lock"
	"github.com/google/uuid"
)

type DeviceHandler struct {
	devices *deviceManager.Handler
	locker  lock.Locker[uuid.UUID]
}

func NewDeviceHandler(
	devices *deviceManager.Handler,
	locker lock.Locker[uuid.UUID],
) *DeviceHandler {
	return &DeviceHandler{
		devices: devices,
		locker:  locker,
	}
}
