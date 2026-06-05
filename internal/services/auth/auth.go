package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"github.com/PoinkCorporation/SSO-service/internal/domain/models"
	"github.com/PoinkCorporation/SSO-service/internal/lib/jwt"
	"github.com/PoinkCorporation/SSO-service/internal/lib/logger/sl"
	"github.com/PoinkCorporation/SSO-service/internal/storage"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserSaver interface {
	SaveUser(
		ctx context.Context,
		email string,
		passHash []byte,
	) (uid int64, err error)
}

type RoleAssigner interface {
	AssignRole(ctx context.Context, userID int64, role string) error
}

type UserProvider interface {
	User(ctx context.Context, email string) (models.User, error)
}

type AppProvider interface {
	App(ctx context.Context, appID int) (models.App, error)
}

type Storage interface {
	UserSaver
	UserProvider
	AppProvider
}

type Auth struct {
	log          *slog.Logger
	storage      Storage
	roleAssigner RoleAssigner
	tokenTTL     time.Duration
}

func New(
	log *slog.Logger,
	storage Storage,
	roleAssigner RoleAssigner,
	tokenTTL time.Duration,
) *Auth {
	return &Auth{
		log:          log,
		storage:      storage,
		roleAssigner: roleAssigner,
		tokenTTL:     tokenTTL,
	}
}

// RegisterNewUser registers new user in the system and returns user ID.
// If user with given username already exists, returns error.
func (a *Auth) RegisterNewUser(ctx context.Context, email string, pass string) (int64, error) {
	const op = "Auth.RegisterNewUser"

	// Create a local logger with additional fields containing useful info about the current function call
	log := a.log.With(
		slog.String("op", op),
		slog.String("email", email),
	)

	log.Info("registering user")

	// Generate password hash and salt.
	passHash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", sl.Err(err))

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	// Save user to database
	id, err := a.storage.SaveUser(ctx, email, passHash)
	if err != nil {
		log.Error("failed to save user", sl.Err(err))

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	if err := a.roleAssigner.AssignRole(ctx, id, "user"); err != nil {
		log.Warn("failed to assign default role", sl.Err(err))
	}

	return id, nil
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

// Login checks if user with given credentials exists in the system and returns access token.
//
// If user exists, but password is incorrect, returns error.
// If user doesn't exist, returns error.
func (a *Auth) Login(
	ctx context.Context,
	email string,
	password string, // plain text password, be careful with logs!
	appID int, // ID of the app the user is logging into
) (string, error) {
	const op = "Auth.Login"

	log := a.log.With(
		slog.String("op", op),
		slog.String("username", email),
		// password should not be logged, or logged in masked form
	)

	log.Info("attempting to login user")

	// Get user from database
	user, err := a.storage.User(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			a.log.Warn("user not found", sl.Err(err))

			return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}

		a.log.Error("failed to get user", sl.Err(err))

		return "", fmt.Errorf("%s: %w", op, err)
	}

	// Verify password correctness
	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		a.log.Info("invalid credentials", sl.Err(err))

		return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	// Get app information
	app, err := a.storage.App(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user logged in successfully")

	// Create authorization token
	token, err := jwt.NewToken(user, app, a.tokenTTL)
	if err != nil {
		a.log.Error("failed to generate token", sl.Err(err))

		return "", fmt.Errorf("%s: %w", op, err)
	}

	return token, nil
}

// Validate checks if the given token is valid and returns user ID and email.
func (a *Auth) Validate(ctx context.Context, tokenString string) (int64, string, error) {
	const op = "Auth.Validate"

	log := a.log.With(slog.String("op", op))

	log.Info("validating token")

	token, err := jwtlib.Parse(tokenString, func(token *jwtlib.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		claims, ok := token.Claims.(jwtlib.MapClaims)
		if !ok {
			return nil, fmt.Errorf("invalid claims")
		}

		appIDFloat, ok := claims["app_id"].(float64)
		if !ok {
			return nil, fmt.Errorf("invalid app_id claim")
		}

		app, err := a.storage.App(context.Background(), int(appIDFloat))
		if err != nil {
			return nil, fmt.Errorf("app lookup: %w", err)
		}

		return []byte(app.Secret), nil
	})
	if err != nil {
		log.Warn("invalid token", sl.Err(err))
		return 0, "", fmt.Errorf("%s: %w", op, ErrInvalidToken)
	}

	claims, ok := token.Claims.(jwtlib.MapClaims)
	if !ok {
		return 0, "", fmt.Errorf("%s: invalid claims", op)
	}

	uidFloat, ok := claims["uid"].(float64)
	if !ok {
		return 0, "", fmt.Errorf("%s: invalid uid claim", op)
	}

	email, ok := claims["email"].(string)
	if !ok {
		return 0, "", fmt.Errorf("%s: invalid email claim", op)
	}

	log.Info("token validated successfully", slog.Int64("uid", int64(uidFloat)))

	return int64(uidFloat), email, nil
}
