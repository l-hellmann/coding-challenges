package deviceManager

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/fiskaly/coding-challenges/signing-service-challenge/api/apiError"
	"github.com/fiskaly/coding-challenges/signing-service-challenge/crypto"
	"github.com/fiskaly/coding-challenges/signing-service-challenge/domain"
	"github.com/fiskaly/coding-challenges/signing-service-challenge/null"
	"github.com/google/uuid"
)

type NewDevice struct {
	Id               null.Null[string]
	Label            null.Null[string]
	SigningAlgorithm domain.SigningAlgorithm
}

func (h *Handler) CreateDevice(ctx context.Context, in NewDevice) (*domain.Device, error) {
	deviceRepository := h.storage.Devices()

	newDevice := &domain.Device{}
	newDevice.SigningAlgorithm = in.SigningAlgorithm

	if value, filled := in.Id.Value(); filled {
		uuidFromString, err := uuid.Parse(value)
		if err != nil {
			slog.Error("invalid uuid", "error", err)
			return nil, err
		}
		count, err := deviceRepository.Count(ctx, domain.DeviceFilter{
			IDs:   []uuid.UUID{uuidFromString},
			Limit: 1,
		})
		if err != nil {
			slog.Error("failed fetching device count", "error", err)
			return nil, err
		}
		if count > 0 {
			return nil, apiError.New(http.StatusConflict, "device with this uuid already exists")
		}
		newDevice.Id = uuidFromString
	} else {
		randomUuid, err := uuid.NewRandom()
		if err != nil {
			slog.Error("uuid generation failed", "error", err)
			return nil, err
		}
		newDevice.Id = randomUuid
	}

	var (
		keyPair crypto.KeyPair
		err     error
	)
	switch newDevice.SigningAlgorithm {
	case domain.SigningAlgorithmRsa:
		keyPair, err = crypto.GenerateRSAKeyPair()
		if err != nil {
			slog.Error("rsa key pair generation", "error", err)
			return nil, err
		}
	case domain.SigningAlgorithmEcc:
		keyPair, err = crypto.GenerateECCKeyPair()
		if err != nil {
			slog.Error("ecc key pair generation", "error", err)
			return nil, err
		}
	default:
		slog.Error("invalid signing algorithm")
		return nil, errors.New("invalid signing algorithm")
	}

	publicKeyBytes, privateKeyBytes, err := crypto.EncodeKeyPair(keyPair)
	if err != nil {
		slog.Error("encode key pair", "error", err)
		return nil, err
	}

	newDevice.Label = in.Label.SqlNull()
	newDevice.PrivateKey = string(privateKeyBytes)
	newDevice.PublicKey = string(publicKeyBytes)

	if err := deviceRepository.Create(ctx, newDevice); err != nil {
		slog.Error("creating device failed", "error", err)
		return nil, err
	}
	return newDevice, nil
}
