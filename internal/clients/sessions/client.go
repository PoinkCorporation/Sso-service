package clients

import (
	"context"

	sessionsv1 "github.com/PoinkCorporation/Sessions-service/gen/go/sessions"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type SessionsClient struct {
	client sessionsv1.SessionsClient
	conn   *grpc.ClientConn
}

func NewSessionsClient(address string) (*SessionsClient, error) {
	cc, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, err
	}
	return &SessionsClient{
		client: sessionsv1.NewSessionsClient(cc),
		conn:   cc,
	}, nil
}

func (c *SessionsClient) ValidateSession(ctx context.Context, jti string) (bool, error) {
	resp, err := c.client.ValidateSession(ctx, &sessionsv1.ValidateSessionRequest{
		AccessTokenJti: jti,
	})
	if err != nil {
		return false, err
	}
	return resp.Valid, nil
}

func (c *SessionsClient) Close() error {
	return c.conn.Close()
}
