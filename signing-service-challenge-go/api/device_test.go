// Package api contains integration tests for the device API endpoints
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

// TypedResponse wraps API responses with a data field
type TypedResponse[T any] struct {
	Data T `json:"data"`
}

// makeRequest is a helper function to create and execute HTTP requests for testing
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

// createDevice is a helper function to create a new device for testing
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

// validateSignature verifies that a signature is valid for the given device and data
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

// TestPostDevice verifies that creating a new device works correctly
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

// TestPostDeviceBadRequest verifies that invalid device creation requests are rejected
// This test covers multiple invalid scenarios to ensure proper input validation
func TestPostDeviceBadRequest(t *testing.T) {
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]()
	api := NewServer(storage, locker).mux()

	// Test case 1: Completely empty request body (nil input)
	// Should return 400 Bad Request because no data was provided
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

	// Test case 2: Empty DTO with no required fields
	// Should return 400 Bad Request because SigningAlgorithm is required
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

	// Test case 3: An invalid ID format provided
	// Should return 400 Bad Request because ID should be auto-generated, not provided
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

	// Test case 4: Invalid signing algorithm
	// Should return 400 Bad Request because "foo" is not a valid signing algorithm
	{
		var out ErrorResponse
		response := makeRequest(
			assert,
			PostDeviceInputDto{
				SigningAlgorithm: "foo",
			},
			http.MethodPost,
			"/api/v0/device",
			api,
			&out,
		)
		assert.Equal(http.StatusBadRequest, response.Code)
	}

}

// TestSignBasic verifies that basic signing functionality works correctly
// This test covers the core signing workflow and signature chaining mechanism
func TestSignBasic(t *testing.T) {
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]()
	api := NewServer(storage, locker).mux()

	// Create a test device with RSA signing algorithm
	device := createDevice(
		assert,
		api,
		domain.SigningAlgorithmRsa,
	)

	// First signing operation - tests initial signature creation
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

		// Parse the signed data format: "counter_deviceId_originalData"
		parts := strings.SplitN(firstSignDto.Data.SignedData, "_", 3)
		assert.Len(parts, 3)
		assert.Equal("1", parts[0]) // First signature should have counter = 1

		// Verify the device ID is correctly embedded in the signed data
		deviceIdBytes, err := base64.StdEncoding.DecodeString(parts[1])
		assert.NoError(err)

		deviceId, err := uuid.FromBytes(deviceIdBytes)
		assert.NoError(err)
		assert.Equal(deviceId.String(), device.Id)

		// Cryptographically validate that the signature is correct
		validateSignature(
			assert,
			firstSignDto.Data,
			device,
		)
	}

	// Second signing operation - tests signature chaining
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

		// Parse the signed data for the second signature
		parts := strings.SplitN(signDto.Data.SignedData, "_", 3)
		assert.Len(parts, 3)
		assert.Equal("2", parts[0]) // Second signature should have counter = 2

		// Verify signature chaining: second signature should include first signature
		assert.Equal(firstSignDto.Data.Signature, parts[1])

		// Cryptographically validate the second signature
		validateSignature(
			assert,
			signDto.Data,
			device,
		)
	}
}

// TestSignEmpty verifies that signing empty data returns no content
// This test ensures the API handles edge cases properly when no data is provided to sign
func TestSignEmpty(t *testing.T) {
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]()
	api := NewServer(storage, locker).mux()

	// Create a test device with ECC signing algorithm
	device := createDevice(
		assert,
		api,
		domain.SigningAlgorithmEcc,
	)

	// Attempt to sign empty data
	signResponse := makeRequest(
		assert,
		PutDeviceSignInputDto{
			Data: "", // Empty string input
		},
		http.MethodPut,
		fmt.Sprintf("/api/v0/device/%s/sign", device.Id),
		api,
		nil, // No output expected for 204 response
	)

	// Should return 204 No Content since there's nothing to sign
	assert.Equal(http.StatusNoContent, signResponse.Code)
}

// TestSignConcurrent verifies that concurrent signing operations work correctly and maintain signature counter
// This test is crucial for validating the locking mechanism and ensuring thread safety
func TestSignConcurrent(t *testing.T) {
	const runs = 25 // Number of concurrent signing operations
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]() // Memory-based locking for concurrency control
	api := NewServer(storage, locker).mux()

	// Create a test device with ECC signing algorithm
	device := createDevice(
		assert,
		api,
		domain.SigningAlgorithmEcc,
	)

	// Helper function to generate random data for each signing operation
	// This ensures each goroutine signs different data
	generateData := func() string {
		buf := make([]byte, 512)
		n, err := io.ReadFull(rand.Reader, buf)
		assert.NoError(err)
		assert.Equal(512, n)
		return string(buf)
	}

	// Launch multiple goroutines to perform concurrent signing operations
	wg := sync.WaitGroup{}
	wg.Add(runs)
	for i := 0; i < runs; i++ {
		go func() {
			defer wg.Done()
			var signDto TypedResponse[PutDeviceSignOutputDto]

			// Each goroutine makes a signing request with unique random data
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

			// Validate that each signature is cryptographically correct
			validateSignature(
				assert,
				signDto.Data,
				device,
			)
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify that the signature counter was incremented correctly
	// This is the key test for concurrency safety - the counter should be exactly 25
	var out TypedResponse[GetDeviceOutputDto]
	deviceResponse := makeRequest(
		assert,
		nil,
		http.MethodGet,
		fmt.Sprintf("/api/v0/device/%s", device.Id),
		api,
		&out,
	)
	assert.Equal(http.StatusOK, deviceResponse.Code)

	assert.Equal(25, out.Data.SignatureCounter) // Must be exactly 25, not less due to race conditions
}

// TestSignBadRequest verifies that invalid signing requests are rejected
// This test ensures proper input validation for the signing endpoint
func TestSignBadRequest(t *testing.T) {
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]()
	api := NewServer(storage, locker).mux()

	// Attempt to sign with an invalid UUID format
	var out ErrorResponse
	signResponse := makeRequest(
		assert,
		PutDeviceSignInputDto{
			Data: "foo",
		},
		http.MethodPut,
		"/api/v0/device/invalid-uuid/sign", // Invalid UUID format
		api,
		&out,
	)

	// Should return 400 Bad Request due to invalid UUID format
	assert.Equal(http.StatusBadRequest, signResponse.Code)
}

// TestSignNotFound verifies that signing with non-existent device returns 404
// This test ensures proper error handling when attempting to use a device that doesn't exist
func TestSignNotFound(t *testing.T) {
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]()
	api := NewServer(storage, locker).mux()

	// Attempt to sign with a valid UUID format but non-existent device
	var out ErrorResponse
	signResponse := makeRequest(
		assert,
		PutDeviceSignInputDto{
			Data: "bar",
		},
		http.MethodPut,
		"/api/v0/device/993d8948-cb1b-4ce8-98f8-f8b866578faf/sign", // Valid UUID but device doesn't exist
		api,
		&out,
	)

	// Should return 404 Not Found because the device doesn't exist in storage
	assert.Equal(http.StatusNotFound, signResponse.Code)
}

// TestGetDevice verifies that retrieving a device returns correct information
// This test ensures the GET endpoint returns complete and accurate device data
func TestGetDevice(t *testing.T) {
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]()
	api := NewServer(storage, locker).mux()

	// Create a test device to retrieve
	device := createDevice(
		assert,
		api,
		domain.SigningAlgorithmRsa,
	)

	// Retrieve the device using its ID
	var out TypedResponse[GetDeviceOutputDto]
	getResponse := makeRequest(
		assert,
		nil, // No request body for GET
		http.MethodGet,
		fmt.Sprintf("/api/v0/device/%s", device.Id),
		api,
		&out,
	)
	assert.Equal(http.StatusOK, getResponse.Code)

	// Verify all returned data matches what was created
	assert.Len(out.Data.PublicKeys, 1)                               // Should have exactly one public key
	assert.Equal(device.PublicKeys, out.Data.PublicKeys)             // Public keys should match
	assert.Equal(device.Label, out.Data.Label)                       // Labels should match
	assert.Equal(device.SigningAlgorithm, out.Data.SigningAlgorithm) // Algorithms should match
}

// TestListDevice verifies that listing devices returns all created devices
// This test ensures the list endpoint returns all devices in the correct order
func TestListDevice(t *testing.T) {
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]()
	api := NewServer(storage, locker).mux()

	// Create multiple devices with different signing algorithms
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

	// Retrieve the list of all devices
	var out TypedResponse[ListDeviceOutputDto]
	listResponse := makeRequest(
		assert,
		nil, // No request body for GET
		http.MethodGet,
		"/api/v0/device",
		api,
		&out,
	)
	assert.Equal(http.StatusOK, listResponse.Code)

	// Verify all devices are returned in creation order
	items := out.Data.Items
	assert.Len(items, 3)                  // Should have exactly 3 devices
	assert.Equal(device1.Id, items[0].Id) // First device created
	assert.Equal(device2.Id, items[1].Id) // Second device created
	assert.Equal(device3.Id, items[2].Id) // Third device created
}

// TestDeleteDevice verifies that deleting devices works correctly
// This test covers both deleting non-existent devices and actual device deletion
func TestDeleteDevice(t *testing.T) {
	assert := require.New(t)

	storage := persistence.NewMemoryStorage()
	locker := lock.NewMemoryLocker[uuid.UUID]()
	api := NewServer(storage, locker).mux()

	// Test case 1: Delete a non-existent device
	// Should return 200 OK (idempotent operation)
	{
		deleteResponse := makeRequest(
			assert,
			nil, // No request body for DELETE
			http.MethodDelete,
			"/api/v0/device/993d8948-cb1b-4ce8-98f8-f8b866578faf", // Valid UUID but device doesn't exist
			api,
			nil,
		)
		assert.Equal(http.StatusOK, deleteResponse.Code) // Deletion is idempotent
	}

	// Test case 2: Create and then delete an actual device
	device := createDevice(
		assert,
		api,
		domain.SigningAlgorithmRsa,
	)

	// Delete the created device
	{
		deleteResponse := makeRequest(
			assert,
			nil, // No request body for DELETE
			http.MethodDelete,
			fmt.Sprintf("/api/v0/device/%s", device.Id),
			api,
			nil,
		)
		assert.Equal(http.StatusOK, deleteResponse.Code) // Should succeed
	}

	// Test case 3: Verify the device is actually deleted
	// Attempting to get the deleted device should return 404
	var out TypedResponse[GetDeviceOutputDto]
	getResponse := makeRequest(
		assert,
		nil,
		http.MethodGet,
		fmt.Sprintf("/api/v0/device/%s", device.Id),
		api,
		&out,
	)
	assert.Equal(http.StatusNotFound, getResponse.Code) // Device should no longer exist
}
