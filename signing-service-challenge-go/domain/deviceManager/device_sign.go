package deviceManager

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"

	"github.com/fiskaly/coding-challenges/signing-service-challenge/api/apiError"
	"github.com/fiskaly/coding-challenges/signing-service-challenge/crypto"
	"github.com/fiskaly/coding-challenges/signing-service-challenge/domain"
	"github.com/google/uuid"
)

type SignedData struct {
	Signature        string
	SignatureCounter int
	Data             string
	LastSignature    string
}

func (h *Handler) SignData(ctx context.Context, deviceId uuid.UUID, data string) (*SignedData, error) {
	deviceRepository := h.storage.Devices()

	device, err := deviceRepository.GetByID(ctx, deviceId)
	if err != nil {
		slog.Error("failed fetching device", "error", err)
		return nil, apiError.New(http.StatusNotFound, "device not found")
	}

	var keyPair crypto.KeyPair
	switch device.SigningAlgorithm {
	case domain.SigningAlgorithmRsa:
		keyPair = new(crypto.RSAKeyPair)
	case domain.SigningAlgorithmEcc:
		keyPair = new(crypto.ECCKeyPair)
	default:
		slog.Error("unknown signing algorithm")
		return nil, errors.New("unknown signing algorithm")
	}

	if err := crypto.DecodePrivateKey([]byte(device.PrivateKey), keyPair); err != nil {
		slog.Error("decode private key", "error", err)
		return nil, err
	}

	signature, err := keyPair.Sign([]byte(data))
	if err != nil {
		slog.Error("signing failed", "error", err)
		return nil, err
	}
	base64Signature := base64.StdEncoding.EncodeToString(signature)

	device.SignatureCounter++

	lastSignature := base64.StdEncoding.EncodeToString(deviceId[:])
	if device.LastSignature.Valid {
		lastSignature = device.LastSignature.V
	}

	device.LastSignature = sql.Null[string]{
		V:     base64Signature,
		Valid: true,
	}

	if err := deviceRepository.Update(ctx, device); err != nil {
		slog.Error("failed updating device", "error", err)
		return nil, err
	}

	return &SignedData{
		Signature:        base64Signature,
		SignatureCounter: device.SignatureCounter,
		Data:             data,
		LastSignature:    lastSignature,
	}, nil
}
