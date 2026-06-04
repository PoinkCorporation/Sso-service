package jwt

import (
	"github.com/PoinkCorporation/SSO-service/internal/domain/models"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// NewToken creates new JWT token for given user and app.
func NewToken(user models.User, app models.App, duration time.Duration) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	// Add all required info to the token
	claims := token.Claims.(jwt.MapClaims)
	claims["uid"] = user.ID
	claims["email"] = user.Email
	claims["exp"] = time.Now().Add(duration).Unix()
	claims["app_id"] = app.ID

	// Sign the token using the app's secret key
	tokenString, err := token.SignedString([]byte(app.Secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
