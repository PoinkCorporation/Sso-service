package clients

import (
	"context"

	profilev1 "github.com/PoinkCorporation/Profile-service/gen/go/profile"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ProfileClient struct {
	client profilev1.ProfileClient
}

func NewProfileClient(address string) (*ProfileClient, error) {
	cc, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &ProfileClient{
		client: profilev1.NewProfileClient(cc),
	}, nil
}

func (c *ProfileClient) CreateDefaultProfile(ctx context.Context, userID int64, username string) error {
	_, err := c.client.UpdateProfile(ctx, &profilev1.UpdateProfileRequest{
		UserId: userID,
		Username: username,
	})
	return err
}
