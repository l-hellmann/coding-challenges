package api

import (
	"log/slog"
	"net/http"

	"github.com/fiskaly/coding-challenges/signing-service-challenge/domain/deviceManager"
	"github.com/fiskaly/coding-challenges/signing-service-challenge/lock"
	"github.com/fiskaly/coding-challenges/signing-service-challenge/persistence"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Server manages HTTP requests and dispatches them to the appropriate services.
type Server struct {
	device *DeviceHandler
}

// NewServer is a factory to instantiate a new Server.
func NewServer(
	storage persistence.Storage,
	locker lock.Locker[uuid.UUID],
) *Server {
	deviceService := deviceManager.New(storage)

	return &Server{
		// TODO: add services / further dependencies here ...
		device: NewDeviceHandler(
			deviceService,
			locker,
		),
	}
}

// mux creates and configures the HTTP request multiplexer with all routes and middleware
func (s *Server) mux() http.Handler {
	mux := chi.NewMux()

	// Add logging middleware to track all incoming requests
	mux.Use(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			slog.Info("incoming request", "method", request.Method, "url", request.URL)
			handler.ServeHTTP(writer, request)
		})
	})

	// Health check endpoint
	mux.Handle("/api/v0/health", http.HandlerFunc(s.Health))

	// TODO: register further HandlerFuncs here ...

	// Device management endpoints
	mux.Post("/api/v0/device", s.device.Post)          // Create a new device
	mux.Get("/api/v0/device", s.device.List)           // List all devices
	mux.Get("/api/v0/device/{id}", s.device.Get)       // Get a specific device
	mux.Delete("/api/v0/device/{id}", s.device.Delete) // Delete a device
	mux.Put("/api/v0/device/{id}/sign", s.device.Sign) // Sign data with a device
	return mux
}

// Run registers all HandlerFuncs for the existing HTTP routes and starts the Server.
func (s *Server) Run(listenAddress string) error {
	slog.Info("server listening", "port", listenAddress)

	return http.ListenAndServe(listenAddress, s.mux())
}
