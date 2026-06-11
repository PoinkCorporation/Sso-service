package clients

import (
	"context"
	"time"

	sessionsv1 "github.com/PoinkCorporation/Sessions-service/gen/go/sessions"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type SessionsClient struct {
	client sessionsv1.SessionsClient
	conn   *grpc.ClientConn
}

func NewSessionsClient(address string) (*SessionsClient, error) {
	cc, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &SessionsClient{
		client: sessionsv1.NewSessionsClient(cc),
		conn:   cc,
	}, nil
}

func (c *SessionsClient) CreateSession(
	ctx context.Context,
	userID int64,
	email string,
	appID int,
	jti string,
	expiresAt time.Time,
	deviceName, deviceIP, userAgent string,
) (string, string, error) {
	resp, err := c.client.CreateSession(ctx, &sessionsv1.CreateSessionRequest{
		UserId:         userID,
		Email:          email,
		AppId:          int32(appID),
		AccessTokenJti: jti,
		ExpiresAt:      expiresAt.Unix(),
		DeviceName:     deviceName,
		DeviceIp:       deviceIP,
		UserAgent:      userAgent,
	})
	if err != nil {
		return "", "", err
	}
	return resp.SessionId, resp.RefreshToken, nil
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
