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
	"github.com/google/uuid"
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

type ProfileCreator interface {
	CreateDefaultProfile(ctx context.Context, userID int64, username string) error
}

type UserProvider interface {
	User(ctx context.Context, email string) (models.User, error)
}

type AppProvider interface {
	App(ctx context.Context, appID int) (models.App, error)
}

type SessionCreator interface {
	CreateSession(
		ctx context.Context,
		userID int64,
		email string,
		appID int,
		jti string,
		expiresAt time.Time,
		deviceName, deviceIP, userAgent string,
	) (string, string, error)
}

type SessionValidator interface {
	ValidateSession(ctx context.Context, jti string) (bool, error)
}

type Storage interface {
	UserSaver
	UserProvider
	AppProvider
}

type Auth struct {
	log              *slog.Logger
	storage          Storage
	roleAssigner     RoleAssigner
	profileCreator   ProfileCreator
	sessionCreator   SessionCreator
	sessionValidator SessionValidator
	tokenTTL         time.Duration
}

func New(
	log *slog.Logger,
	storage Storage,
	roleAssigner RoleAssigner,
	profileCreator ProfileCreator,
	sessionCreator SessionCreator,
	sessionValidator SessionValidator,
	tokenTTL time.Duration,
) *Auth {
	return &Auth{
		log:              log,
		storage:          storage,
		roleAssigner:     roleAssigner,
		profileCreator:   profileCreator,
		sessionCreator:   sessionCreator,
		sessionValidator: sessionValidator,
		tokenTTL:         tokenTTL,
	}
}

// RegisterNewUser registers new user in the system and returns user ID.
// If user with given username already exists, returns error.
func (a *Auth) RegisterNewUser(ctx context.Context, email string, pass string) (int64, error) {
	const op = "Auth.RegisterNewUser"

	log := a.log.With(
		slog.String("op", op),
		slog.String("email", email),
	)

	log.Info("registering user")

	passHash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", sl.Err(err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := a.storage.SaveUser(ctx, email, passHash)
	if err != nil {
		log.Error("failed to save user", sl.Err(err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	if err := a.roleAssigner.AssignRole(ctx, id, "user"); err != nil {
		log.Warn("failed to assign default role", sl.Err(err))
	}

	if err := a.profileCreator.CreateDefaultProfile(ctx, id, email); err != nil {
		log.Warn("failed to create default profile", sl.Err(err))
	}

	return id, nil
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

// Login checks if user with given credentials exists in the system and returns access token
// along with session information (session_id, refresh_token).
func (a *Auth) Login(
	ctx context.Context,
	email string,
	password string,
	appID int,
	deviceName, deviceIP, userAgent string,
) (string, string, string, error) {
	const op = "Auth.Login"

	log := a.log.With(
		slog.String("op", op),
		slog.String("username", email),
	)

	log.Info("attempting to login user")

	user, err := a.storage.User(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			a.log.Warn("user not found", sl.Err(err))
			return "", "", "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}
		a.log.Error("failed to get user", sl.Err(err))
		return "", "", "", fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		a.log.Info("invalid credentials", sl.Err(err))
		return "", "", "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	app, err := a.storage.App(ctx, appID)
	if err != nil {
		if errors.Is(err, storage.ErrAppNotFound) {
			return "", "", "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}
		return "", "", "", fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user logged in successfully")

	jti := uuid.New().String()
	token, err := jwt.NewToken(user, app, a.tokenTTL, jti)
	if err != nil {
		a.log.Error("failed to generate token", sl.Err(err))
		return "", "", "", fmt.Errorf("%s: %w", op, err)
	}

	expiresAt := time.Now().Add(a.tokenTTL)

	sessionID, refreshToken, err := a.sessionCreator.CreateSession(
		ctx, user.ID, user.Email, appID, jti, expiresAt,
		deviceName, deviceIP, userAgent,
	)
	if err != nil {
		a.log.Error("failed to create session", sl.Err(err))
		return "", "", "", fmt.Errorf("%s: %w", op, err)
	}

	return token, sessionID, refreshToken, nil
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

	jti, ok := claims["jti"].(string)
	if !ok {
		return 0, "", fmt.Errorf("%s: invalid jti claim", op)
	}

	valid, err := a.sessionValidator.ValidateSession(ctx, jti)
	if err != nil {
		log.Warn("failed to validate session", sl.Err(err))
		return 0, "", fmt.Errorf("%s: %w", op, ErrInvalidToken)
	}
	if !valid {
		log.Warn("session is not active", slog.String("jti", jti))
		return 0, "", fmt.Errorf("%s: %w", op, ErrInvalidToken)
	}

	log.Info("token validated successfully", slog.Int64("uid", int64(uidFloat)))

	return int64(uidFloat), email, nil
}

// IssueToken generates a new JWT for an existing user (no password check).
func (a *Auth) IssueToken(ctx context.Context, userID int64, email string, appID int) (string, string, error) {
	const op = "Auth.IssueToken"

	log := a.log.With(
		slog.String("op", op),
		slog.Int64("user_id", userID),
	)

	app, err := a.storage.App(ctx, appID)
	if err != nil {
		log.Error("failed to get app", sl.Err(err))
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	jti := uuid.New().String()
	user := models.User{ID: userID, Email: email}

	token, err := jwt.NewToken(user, app, a.tokenTTL, jti)
	if err != nil {
		log.Error("failed to generate token", sl.Err(err))
		return "", "", fmt.Errorf("%s: %w", op, err)
	}

	return token, jti, nil
}
