package clients

import (
	"context"

	permv1 "github.com/PoinkCorporation/Permissions-service/gen/go/permissions"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PermissionsClient struct {
	client permv1.PermissionsClient
}

func NewPermissionsClient(address string) (*PermissionsClient, error) {
	cc, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &PermissionsClient{
		client: permv1.NewPermissionsClient(cc),
	}, nil
}
func (c *PermissionsClient) AssignRole(ctx context.Context, userID int64, role string) error {
	_, err := c.client.AssignRole(ctx, &permv1.AssignRoleRequest{
		UserId: userID,
		Role:   role,
	})
	return err
}
