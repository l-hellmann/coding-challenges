package api

import (
	"bytes"
	stdcrypto "crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/fiskaly/coding-challenges/signing-service-challenge/domain"
	"github.com/fiskaly/coding-challenges/signing-service-challenge/lock"
	"github.com/fiskaly/coding-challenges/signing-service-challenge/null"
	"github.com/fiskaly/coding-challenges/signing-service-challenge/persistence"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type TypedResponse[T any] struct {
	Data T `json:"data"`
}

func makeRequest(
	assert *require.Assertions,
	inputDto any,
	method string,
	urlPath string,
	handle http.Handler,
	outputDto any,
) *httptest.ResponseRecorder {
	buf := bytes.NewBuffer(nil)
	if inputDto != nil {
		buf = bytes.NewBuffer(nil)
		err := json.NewEncoder(buf).Encode(inputDto)
		assert.NoError(err)
	}

	url, err := url.JoinPath("http://localhost/", urlPath)
	assert.NoError(err)

	var req *http.Request
	if inputDto != nil {
		req = httptest.NewRequest(method, url, buf)
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	res := httptest.NewRecorder()
	handle.ServeHTTP(res, req)

	if outputDto != nil && res.Body.Len() > 0 {
		err := json.NewDecoder(res.Body).Decode(&outputDto)
		assert.NoError(err)
	}
	return res
}

func createDevice(
	assert *require.Assertions,
	api http.Handler,
	alg domain.SigningAlgorithm,
) PostDeviceOutputDto {
	var deviceDto TypedResponse[PostDeviceOutputDto]
	postResponse := makeRequest(
		assert,
		PostDeviceInputDto{
			SigningAlgorithm: alg,
		},
		http.MethodPost,
		"/api/v0/device",
		api,
		&deviceDto,
	)
	assert.Equal(http.StatusCreated, postResponse.Code)

	return deviceDto.Data
}

func validateSignature(
	assert *require.Assertions,
	signDto PutDeviceSignOutputDto,
	device PostDeviceOutputDto,
) {
	parts := strings.SplitN(signDto.SignedData, "_", 3)
	assert.Len(parts, 3)

	sum := sha256.Sum256([]byte(parts[2]))
	signature, err := base64.StdEncoding.DecodeString(signDto.Signature)
	assert.NoError(err)

	assert.GreaterOrEqual(len(device.PublicKeys), 1)
	block, rest := pem.Decode([]byte(device.PublicKeys[0]))
	assert.Len(rest, 0)

	switch device.SigningAlgorithm {
	case domain.SigningAlgorithmRsa:
		publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
		assert.NoError(err)

		err = rsa.VerifyPKCS1v15(publicKey, stdcrypto.SHA256, sum[:], signature)
		assert.NoError(err)
	case domain.SigningAlgorithmEcc:
		publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		assert.NoError(err)
		assert.IsType(publicKey, &ecdsa.PublicKey{})
		valid := ecdsa.VerifyASN1(publicKey.(*ecdsa.PublicKey), sum[:], signature)
		assert.True(valid)
	default:
		assert.Fail("unknown signing algorithm")
	}

}

func TestPostDevice(t *testing.T) {
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]()
	api := NewServer(storage, locker).mux()

	var out TypedResponse[PostDeviceOutputDto]
	response := makeRequest(
		assert,
		PostDeviceInputDto{
			SigningAlgorithm: domain.SigningAlgorithmRsa,
			Label:            null.New("fooSigner"),
		},
		http.MethodPost,
		"/api/v0/device",
		api,
		&out,
	)
	assert.Equal(http.StatusCreated, response.Code)

	id, err := uuid.Parse(out.Data.Id)
	assert.NoError(err)
	assert.NotEqual(uuid.Nil, id)
	assert.Equal(out.Data.SigningAlgorithm, domain.SigningAlgorithmRsa)
	assert.True(out.Data.Label.Filled())
	assert.Equal(out.Data.Label.Some(), "fooSigner")
	assert.Greater(len(out.Data.PublicKeys), 0)
}

func TestPostDeviceBadRequest(t *testing.T) {
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]()
	api := NewServer(storage, locker).mux()
	{
		var out ErrorResponse
		response := makeRequest(
			assert,
			nil,
			http.MethodPost,
			"/api/v0/device",
			api,
			&out,
		)
		assert.Equal(http.StatusBadRequest, response.Code)
	}
	{
		var out ErrorResponse
		response := makeRequest(
			assert,
			PostDeviceInputDto{},
			http.MethodPost,
			"/api/v0/device",
			api,
			&out,
		)
		assert.Equal(http.StatusBadRequest, response.Code)
	}
	{
		var out ErrorResponse
		response := makeRequest(
			assert,
			PostDeviceInputDto{
				Id:               null.New("foo"),
				SigningAlgorithm: domain.SigningAlgorithmRsa,
			},
			http.MethodPost,
			"/api/v0/device",
			api,
			&out,
		)
		assert.Equal(http.StatusBadRequest, response.Code)
	}
	{
		var out ErrorResponse
		response := makeRequest(
			assert,
			PostDeviceInputDto{
				SigningAlgorithm: "usa",
			},
			http.MethodPost,
			"/api/v0/device",
			api,
			&out,
		)
		assert.Equal(http.StatusBadRequest, response.Code)
	}

}

func TestSignBasic(t *testing.T) {
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]()
	api := NewServer(storage, locker).mux()

	device := createDevice(
		assert,
		api,
		domain.SigningAlgorithmRsa,
	)

	var firstSignDto TypedResponse[PutDeviceSignOutputDto]
	{
		signResponse := makeRequest(
			assert,
			PutDeviceSignInputDto{
				Data: "lorem ipsum dolor",
			},
			http.MethodPut,
			fmt.Sprintf("/api/v0/device/%s/sign", device.Id),
			api,
			&firstSignDto,
		)
		assert.Equal(http.StatusOK, signResponse.Code)

		parts := strings.SplitN(firstSignDto.Data.SignedData, "_", 3)
		assert.Len(parts, 3)
		assert.Equal("1", parts[0])

		deviceIdBytes, err := base64.StdEncoding.DecodeString(parts[1])
		assert.NoError(err)

		deviceId, err := uuid.FromBytes(deviceIdBytes)
		assert.NoError(err)
		assert.Equal(deviceId.String(), device.Id)

		validateSignature(
			assert,
			firstSignDto.Data,
			device,
		)
	}
	{
		var signDto TypedResponse[PutDeviceSignOutputDto]
		signResponse := makeRequest(
			assert,
			PutDeviceSignInputDto{
				Data: "lorem ipsum dolor sit am",
			},
			http.MethodPut,
			fmt.Sprintf("/api/v0/device/%s/sign", device.Id),
			api,
			&signDto,
		)
		assert.Equal(http.StatusOK, signResponse.Code)
		parts := strings.SplitN(signDto.Data.SignedData, "_", 3)
		assert.Len(parts, 3)
		assert.Equal("2", parts[0])
		assert.Equal(firstSignDto.Data.Signature, parts[1])

		validateSignature(
			assert,
			signDto.Data,
			device,
		)
	}
}

func TestSignEmpty(t *testing.T) {
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]()
	api := NewServer(storage, locker).mux()

	device := createDevice(
		assert,
		api,
		domain.SigningAlgorithmEcc,
	)

	signResponse := makeRequest(
		assert,
		PutDeviceSignInputDto{
			Data: "",
		},
		http.MethodPut,
		fmt.Sprintf("/api/v0/device/%s/sign", device.Id),
		api,
		nil,
	)
	assert.Equal(http.StatusNoContent, signResponse.Code)
}

func TestSignConcurrent(t *testing.T) {
	const runs = 25
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]()
	api := NewServer(storage, locker).mux()

	device := createDevice(
		assert,
		api,
		domain.SigningAlgorithmEcc,
	)

	generateData := func() string {
		buf := make([]byte, 512)
		n, err := io.ReadFull(rand.Reader, buf)
		assert.NoError(err)
		assert.Equal(512, n)
		return string(buf)
	}

	wg := sync.WaitGroup{}
	wg.Add(runs)
	for i := 0; i < runs; i++ {
		go func() {
			defer wg.Done()
			var signDto TypedResponse[PutDeviceSignOutputDto]
			signResponse := makeRequest(
				assert,
				PutDeviceSignInputDto{
					Data: generateData(),
				},
				http.MethodPut,
				fmt.Sprintf("/api/v0/device/%s/sign", device.Id),
				api,
				&signDto,
			)
			assert.Equal(http.StatusOK, signResponse.Code)

			validateSignature(
				assert,
				signDto.Data,
				device,
			)
		}()
	}
	wg.Wait()

	deviceId, err := uuid.Parse(device.Id)
	assert.NoError(err)
	d, err := storage.Devices().GetByID(t.Context(), deviceId)
	assert.NoError(err)
	assert.Equal(25, d.SignatureCounter)
}

func TestSignBadRequest(t *testing.T) {
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]()
	api := NewServer(storage, locker).mux()

	var out ErrorResponse
	signResponse := makeRequest(
		assert,
		PutDeviceSignInputDto{
			Data: "foo",
		},
		http.MethodPut,
		"/api/v0/device/invalid-uuid/sign",
		api,
		&out,
	)
	assert.Equal(http.StatusBadRequest, signResponse.Code)
}

func TestSignNotFound(t *testing.T) {
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]()
	api := NewServer(storage, locker).mux()

	var out ErrorResponse
	signResponse := makeRequest(
		assert,
		PutDeviceSignInputDto{
			Data: "bar",
		},
		http.MethodPut,
		"/api/v0/device/993d8948-cb1b-4ce8-98f8-f8b866578faf/sign",
		api,
		&out,
	)
	assert.Equal(http.StatusNotFound, signResponse.Code)
}

func TestGetDevice(t *testing.T) {
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]()
	api := NewServer(storage, locker).mux()

	device := createDevice(
		assert,
		api,
		domain.SigningAlgorithmRsa,
	)

	var out TypedResponse[GetDeviceOutputDto]
	getResponse := makeRequest(
		assert,
		nil,
		http.MethodGet,
		fmt.Sprintf("/api/v0/device/%s", device.Id),
		api,
		&out,
	)
	assert.Equal(http.StatusOK, getResponse.Code)

	assert.Len(out.Data.PublicKeys, 1)
	assert.Equal(device.PublicKeys, out.Data.PublicKeys)
	assert.Equal(device.Label, out.Data.Label)
	assert.Equal(device.SigningAlgorithm, out.Data.SigningAlgorithm)
}

func TestListDevice(t *testing.T) {
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]()
	api := NewServer(storage, locker).mux()

	device1 := createDevice(
		assert,
		api,
		domain.SigningAlgorithmRsa,
	)
	device2 := createDevice(
		assert,
		api,
		domain.SigningAlgorithmEcc,
	)
	device3 := createDevice(
		assert,
		api,
		domain.SigningAlgorithmRsa,
	)

	var out TypedResponse[ListDeviceOutputDto]
	listResponse := makeRequest(
		assert,
		nil,
		http.MethodGet,
		"/api/v0/device",
		api,
		&out,
	)
	assert.Equal(http.StatusOK, listResponse.Code)

	items := out.Data.Items
	assert.Len(items, 3)
	assert.Equal(device1.Id, items[0].Id)
	assert.Equal(device2.Id, items[1].Id)
	assert.Equal(device3.Id, items[2].Id)
}

func TestDeleteDevice(t *testing.T) {
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]()
	api := NewServer(storage, locker).mux()

	{
		deleteResponse := makeRequest(
			assert,
			nil,
			http.MethodDelete,
			"/api/v0/device/993d8948-cb1b-4ce8-98f8-f8b866578faf",
			api,
			nil,
		)
		assert.Equal(http.StatusOK, deleteResponse.Code)
	}

	device := createDevice(
		assert,
		api,
		domain.SigningAlgorithmRsa,
	)

	{
		deleteResponse := makeRequest(
			assert,
			nil,
			http.MethodDelete,
			fmt.Sprintf("/api/v0/device/%s", device.Id),
			api,
			nil,
		)
		assert.Equal(http.StatusOK, deleteResponse.Code)
	}

	var out TypedResponse[GetDeviceOutputDto]
	getResponse := makeRequest(
		assert,
		nil,
		http.MethodGet,
		fmt.Sprintf("/api/v0/device/%s", device.Id),
		api,
		&out,
	)
	assert.Equal(http.StatusNotFound, getResponse.Code)
}
