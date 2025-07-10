package deviceManager

import (
	"github.com/fiskaly/coding-challenges/signing-service-challenge/persistence"
)

type Handler struct {
	storage persistence.Storage
}

func New(
	storage persistence.Storage,
) *Handler {
	return &Handler{
		storage: storage,
	}
}
