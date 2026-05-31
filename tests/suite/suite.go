package suite

import (
	"context"
	"net"
	ssov1 "sso/gen/go/sso"
	"sso/internal/config"
	"strconv"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Suite struct {
	*testing.T
	Cfg        *config.Config
	AuthClient ssov1.AuthClient
}

func New(t *testing.T) (context.Context, *Suite) {
	t.Helper()
	t.Parallel()

	cfg := config.MustLoad()

	ctx, cancelCtx := context.WithTimeout(context.Background(), cfg.GRPC.Timeout)

	grpcAddress := net.JoinHostPort("localhost", strconv.Itoa(cfg.GRPC.Port))

	cc, err := grpc.NewClient(
		grpcAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc server connection failed: %v", err)
	}

	t.Cleanup(func() {
		t.Helper()
		cc.Close()
		cancelCtx()
	})

	authClient := ssov1.NewAuthClient(cc)

	return ctx, &Suite{
		T:          t,
		Cfg:        cfg,
		AuthClient: authClient,
	}
}
