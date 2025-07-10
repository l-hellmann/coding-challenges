package main

import (
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/fiskaly/coding-challenges/signing-service-challenge/api"
	"github.com/fiskaly/coding-challenges/signing-service-challenge/lock"
	"github.com/fiskaly/coding-challenges/signing-service-challenge/persistence"
	"github.com/google/uuid"
)

var config = struct {
	ListenAddress string
}{}

func main() {
	flag.StringVar(&config.ListenAddress, "listen-address", ":8080", "api listen address")
	flag.Parse()

	loggerOptions := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logger := slog.Handler(slog.NewTextHandler(os.Stdout, loggerOptions))
	slog.SetDefault(slog.New(logger))

	storage := persistence.NewMemoryStorage()
	defer storage.Close()

	server := api.NewServer(
		storage,
		lock.NewMemoryLocker[uuid.UUID](),
	)

	if err := server.Run(config.ListenAddress); err != nil {
		log.Fatal("Could not start server on ", config.ListenAddress)
	}
}
