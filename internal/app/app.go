package app

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/PoinkCorporation/SSO-service/internal/cache"
	grpcapp "github.com/PoinkCorporation/SSO-service/internal/app/grpc"
	permclients "github.com/PoinkCorporation/SSO-service/internal/clients/permissions"
	profileclients "github.com/PoinkCorporation/SSO-service/internal/clients/profile"
	sessionsclients "github.com/PoinkCorporation/SSO-service/internal/clients/sessions"
	"github.com/PoinkCorporation/SSO-service/internal/services/auth"
	"github.com/PoinkCorporation/SSO-service/internal/storage/postgresql"
)

type App struct {
	GRPCServer *grpcapp.App
	storage    *postgresql.PGRepo
	cache      *cache.Cache
}

func New(
	log *slog.Logger,
	grpcPort int,
	storagePath string,
	permissionsAddress string,
	profileAddress string,
	sessionsAddress string,
	tokenTTL time.Duration,
	redisAddr string,
	redisPassword string,
	redisDB int,
	cacheTTL time.Duration,
) (*App, error) {
	storage, err := postgresql.New(storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to init storage: %w", err)
	}

	permClient, err := permclients.NewPermissionsClient(permissionsAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to init permissions client: %w", err)
	}

	profileClient, err := profileclients.NewProfileClient(profileAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to init profile client: %w", err)
	}

	sessionsClient, err := sessionsclients.NewSessionsClient(sessionsAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to init sessions client: %w", err)
	}

	redisCache := cache.New(redisAddr, redisPassword, redisDB, cacheTTL, log)

	authService := auth.New(log, storage, permClient, profileClient, sessionsClient, tokenTTL, redisCache)

	grpcApp := grpcapp.New(log, authService, grpcPort)

	return &App{
		GRPCServer: grpcApp,
		storage:    storage,
		cache:      redisCache,
	}, nil
}

func (a *App) Stop() {
	a.GRPCServer.Stop()
	a.storage.Close()
	a.cache.Close()
}
