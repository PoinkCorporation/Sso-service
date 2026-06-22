package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/PoinkCorporation/SSO-service/internal/app"
	"github.com/PoinkCorporation/SSO-service/internal/config"
	"github.com/PoinkCorporation/SSO-service/internal/otel"
	"github.com/PoinkCorporation/SSO-service/internal/lib/logger/sl"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	shutdown, err := otel.Init(context.Background(), "sso-service", "tempo:4317")
	if err != nil {
		log.Error("failed to init otel", "error", err)
	}
	defer shutdown(context.Background())

	application, err := app.New(log, cfg.GRPC.Port, cfg.StoragePath, cfg.PermissionsAddress, cfg.ProfileAddress, cfg.SessionsAddress, cfg.TokenTTL, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.CacheTTL)
	if err != nil {
		log.Error("failed to init app", sl.Err(err))
		os.Exit(1)
	}

	go func() {
		application.GRPCServer.MustRun()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("shutting down", slog.String("signal", sig.String()))

	application.Stop()
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}
