package jwt

import (
	"context"
	"crypto/rsa"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const (
	UserID string = "userID"
)

func NewToken(
	userID string,
	exp time.Duration,
	key *rsa.PrivateKey,
) (string, error) {

	token, err := jwt.NewBuilder().
		Issuer("gophermart").
		Audience([]string{string(userID)}).
		IssuedAt(time.Now()).
		Expiration(time.Now().Add(exp)).
		Claim(UserID, userID).
		Build()
	if err != nil {
		return "", err
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, key))
	if err != nil {
		return "", err
	}
	return string(signed), nil
}

// jwt.Token из контекста
func TokenFromContext(ctx context.Context) jwt.Token {
	if token, ok := ctx.Value(jwtauth.TokenCtxKey).(jwt.Token); ok {
		return token
	}
	return nil
}

// Атрибут из private claims jwt.Token
func ClaimJWTFromContext[T any](ctx context.Context, key string) (T, bool) {
	var result T
	token := TokenFromContext(ctx)
	if token == nil {
		return result, false
	}

	claims := token.PrivateClaims()
	if value, ok := claims[key]; ok {
		if result, ok := value.(T); ok {
			return result, true
		}
	}

	return result, false
}
