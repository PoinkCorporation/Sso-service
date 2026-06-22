package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/PoinkCorporation/SSO-service/internal/cache"
	"github.com/PoinkCorporation/SSO-service/internal/domain/models"
	"github.com/PoinkCorporation/SSO-service/internal/lib/jwt"
	"github.com/PoinkCorporation/SSO-service/internal/lib/logger/sl"
	"github.com/PoinkCorporation/SSO-service/internal/storage"

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
	sessionValidator SessionValidator
	tokenTTL         time.Duration
	cache            *cache.Cache
}

func New(
	log *slog.Logger,
	storage Storage,
	roleAssigner RoleAssigner,
	profileCreator ProfileCreator,
	sessionValidator SessionValidator,
	tokenTTL time.Duration,
	cacheInstance *cache.Cache,
) *Auth {
	return &Auth{
		log:              log,
		storage:          storage,
		roleAssigner:     roleAssigner,
		profileCreator:   profileCreator,
		sessionValidator: sessionValidator,
		tokenTTL:         tokenTTL,
		cache:            cacheInstance,
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

func appCacheKey(appID int) string {
	return fmt.Sprintf("sso:app:%d", appID)
}

func (a *Auth) getApp(ctx context.Context, appID int) (models.App, error) {
	key := appCacheKey(appID)
	var cached models.App
	if err := a.cache.Get(ctx, key, &cached); err == nil {
		return cached, nil
	}

	app, err := a.storage.App(ctx, appID)
	if err != nil {
		return models.App{}, err
	}

	_ = a.cache.Set(ctx, key, app)

	return app, nil
}

func sessionValidCacheKey(jti string) string {
	return fmt.Sprintf("sso:session_valid:%s", jti)
}

func (a *Auth) validateSessionCached(ctx context.Context, jti string) (bool, error) {
	key := sessionValidCacheKey(jti)
	var cached bool
	if err := a.cache.Get(ctx, key, &cached); err == nil {
		return cached, nil
	}

	valid, err := a.sessionValidator.ValidateSession(ctx, jti)
	if err != nil {
		return false, err
	}

	_ = a.cache.Set(ctx, key, valid)

	return valid, nil
}

// Login checks if user with given credentials exists in the system and returns access token
// along with session information (session_id, refresh_token).
func (a *Auth) Login(
	ctx context.Context,
	email string,
	password string,
	appID int,
) (string, error) {
	const op = "Auth.Login"

	log := a.log.With(
		slog.String("op", op),
		slog.String("username", email),
	)

	user, err := a.storage.User(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			a.log.Debug("user not found", sl.Err(err))
			return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}
		a.log.Error("failed to get user", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		a.log.Debug("invalid credentials", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	app, err := 	a.getApp(ctx, appID)
	if err != nil {
		if errors.Is(err, storage.ErrAppNotFound) {
			return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	log.Debug("user logged in successfully")

	jti := uuid.New().String()
	token, err := jwt.NewToken(user, app, a.tokenTTL, jti)
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

		app, err := a.getApp(context.Background(), int(appIDFloat))
		if err != nil {
			return nil, fmt.Errorf("app lookup: %w", err)
		}

		return []byte(app.Secret), nil
	})
	if err != nil {
		log.Debug("invalid token", sl.Err(err))
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

	valid, err := a.validateSessionCached(ctx, jti)
	if err != nil {
		log.Warn("failed to validate session", sl.Err(err))
		return 0, "", fmt.Errorf("%s: %w", op, ErrInvalidToken)
	}
	if !valid {
		log.Debug("session is not active", slog.String("jti", jti))
		return 0, "", fmt.Errorf("%s: %w", op, ErrInvalidToken)
	}

	return int64(uidFloat), email, nil
}

// IssueToken generates a new JWT for an existing user (no password check).
func (a *Auth) IssueToken(ctx context.Context, userID int64, email string, appID int) (string, string, error) {
	const op = "Auth.IssueToken"

	log := a.log.With(
		slog.String("op", op),
		slog.Int64("user_id", userID),
	)

	app, err := 	a.getApp(ctx, appID)
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

// ValidateToken checks JWT signature and returns user ID, email, and app ID WITHOUT checking session existence.
func (a *Auth) ValidateToken(ctx context.Context, tokenString string) (int64, string, int, error) {
	const op = "Auth.ValidateToken"

	log := a.log.With(slog.String("op", op))

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

		app, err := a.getApp(context.Background(), int(appIDFloat))
		if err != nil {
			return nil, fmt.Errorf("app lookup: %w", err)
		}

		return []byte(app.Secret), nil
	})
	if err != nil {
		log.Debug("invalid token", sl.Err(err))
		return 0, "", 0, fmt.Errorf("%s: %w", op, ErrInvalidToken)
	}

	claims, ok := token.Claims.(jwtlib.MapClaims)
	if !ok {
		return 0, "", 0, fmt.Errorf("%s: invalid claims", op)
	}

	uidFloat, ok := claims["uid"].(float64)
	if !ok {
		return 0, "", 0, fmt.Errorf("%s: invalid uid claim", op)
	}

	email, ok := claims["email"].(string)
	if !ok {
		return 0, "", 0, fmt.Errorf("%s: invalid email claim", op)
	}

	appIDFloat, ok := claims["app_id"].(float64)
	if !ok {
		return 0, "", 0, fmt.Errorf("%s: invalid app_id claim", op)
	}

	log.Info("token validated successfully", slog.Int64("uid", int64(uidFloat)))

	return int64(uidFloat), email, int(appIDFloat), nil
}
