package app

import (
	"log/slog"
	"time"

	grpcapp "github.com/PoinkCorporation/SSO-service/internal/app/grpc"
	clients "github.com/PoinkCorporation/SSO-service/internal/clients/permissions"
	"github.com/PoinkCorporation/SSO-service/internal/services/auth"
	"github.com/PoinkCorporation/SSO-service/internal/storage/postgresql"
)

type App struct {
	GRPCServer *grpcapp.App
}

func New(
	log *slog.Logger,
	grpcPort int,
	storagePath string,
	permissionsAddress string,
	tokenTTL time.Duration,
) *App {
	storage, err := postgresql.New(storagePath)
	if err != nil {
		panic(err)
	}

	permClient, err := clients.NewPermissionsClient(permissionsAddress)
	if err != nil {
		panic(err)
	}

	authService := auth.New(log, storage, storage, storage, permClient, tokenTTL)

	grpcApp := grpcapp.New(log, authService, grpcPort)

	return &App{
		GRPCServer: grpcApp,
	}
}
