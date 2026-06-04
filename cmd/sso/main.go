package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/PoinkCorporation/SSO-service/internal/app"
	"github.com/PoinkCorporation/SSO-service/internal/config"
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

	application, err := app.New(log, cfg.GRPC.Port, cfg.StoragePath, cfg.PermissionsAddress, cfg.TokenTTL)
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
	}

	return log
}
