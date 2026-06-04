package app

import (
	"fmt"
	"log/slog"
	"time"

	grpcapp "github.com/PoinkCorporation/SSO-service/internal/app/grpc"
	clients "github.com/PoinkCorporation/SSO-service/internal/clients/permissions"
	"github.com/PoinkCorporation/SSO-service/internal/services/auth"
	"github.com/PoinkCorporation/SSO-service/internal/storage/postgresql"
)

type App struct {
	GRPCServer *grpcapp.App
	storage    *postgresql.PGRepo
}

func New(
	log *slog.Logger,
	grpcPort int,
	storagePath string,
	permissionsAddress string,
	tokenTTL time.Duration,
) (*App, error) {
	storage, err := postgresql.New(storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to init storage: %w", err)
	}

	permClient, err := clients.NewPermissionsClient(permissionsAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to init permissions client: %w", err)
	}

	authService := auth.New(log, storage, permClient, tokenTTL)

	grpcApp := grpcapp.New(log, authService, grpcPort)

	return &App{
		GRPCServer: grpcApp,
		storage:    storage,
	}, nil
}

func (a *App) Stop() {
	a.GRPCServer.Stop()
	a.storage.Close()
}
