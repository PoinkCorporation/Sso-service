package authgrpc

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ssov1 "github.com/PoinkCorporation/SSO-service/gen/go/sso"
	"github.com/PoinkCorporation/SSO-service/internal/services/auth"
	"github.com/PoinkCorporation/SSO-service/internal/storage"
)

type serverAPI struct {
	ssov1.UnimplementedAuthServer
	auth Auth
}

type Auth interface {
	Login(
		ctx context.Context,
		email string,
		password string,
		appID int,
	) (token string, err error)
	RegisterNewUser(
		ctx context.Context,
		email string,
		password string,
	) (userID int64, err error)
	Validate(
		ctx context.Context,
		token string,
	) (userID int64, email string, err error)
	ValidateToken(
		ctx context.Context,
		token string,
	) (userID int64, email string, appID int, err error)
	IssueToken(
		ctx context.Context,
		userID int64,
		email string,
		appID int,
	) (token string, jti string, err error)
}

func Register(gRPCServer *grpc.Server, auth Auth) {
	ssov1.RegisterAuthServer(gRPCServer, &serverAPI{auth: auth})
}

func (s *serverAPI) Login(
	ctx context.Context,
	in *ssov1.LoginRequest,
) (*ssov1.LoginResponse, error) {
	if in.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	if in.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	if in.GetAppId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "app_id is required")
	}

	token, err := s.auth.Login(
		ctx,
		in.GetEmail(),
		in.GetPassword(),
		int(in.GetAppId()),
	)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, status.Error(codes.InvalidArgument, "invalid email or password")
		}

		return nil, status.Error(codes.Internal, "failed to login")
	}

	return &ssov1.LoginResponse{
		Token: token,
	}, nil
}

func (s *serverAPI) Register(
	ctx context.Context,
	in *ssov1.RegisterRequest,
) (*ssov1.RegisterResponse, error) {
	if in.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	if in.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	uid, err := s.auth.RegisterNewUser(ctx, in.GetEmail(), in.GetPassword())
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		}

		return nil, status.Error(codes.Internal, "failed to register user")
	}

	return &ssov1.RegisterResponse{UserId: uid}, nil
}

func (s *serverAPI) Validate(
	ctx context.Context,
	in *ssov1.ValidateRequest,
) (*ssov1.ValidateResponse, error) {
	if in.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	uid, email, err := s.auth.Validate(ctx, in.GetToken())
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}
		return nil, status.Error(codes.Internal, "failed to validate token")
	}

	return &ssov1.ValidateResponse{UserId: uid, Email: email}, nil
}

func (s *serverAPI) IssueToken(
	ctx context.Context,
	in *ssov1.IssueTokenRequest,
) (*ssov1.IssueTokenResponse, error) {
	if in.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	if in.GetAppId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "app_id is required")
	}

	token, jti, err := s.auth.IssueToken(ctx, in.GetUserId(), in.GetEmail(), int(in.GetAppId()))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to issue token")
	}

	return &ssov1.IssueTokenResponse{Token: token, Jti: jti}, nil
}

func (s *serverAPI) ValidateToken(
	ctx context.Context,
	in *ssov1.ValidateTokenRequest,
) (*ssov1.ValidateTokenResponse, error) {
	if in.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	uid, email, appID, err := s.auth.ValidateToken(ctx, in.GetToken())
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}
		return nil, status.Error(codes.Internal, "failed to validate token")
	}

	return &ssov1.ValidateTokenResponse{UserId: uid, Email: email, AppId: int32(appID)}, nil
}
