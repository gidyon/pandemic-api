package auth

import (
	"context"
	"fmt"
	"os"

	"github.com/Pallinder/go-randomdata"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	signingKey                      = []byte(os.Getenv("JWT_TOKEN"))
	signingMethod jwt.SigningMethod = jwt.SigningMethodHS256
)

// Payload contains jwt payload
type Payload struct {
	ID           string
	FullName     string
	PhoneNumber  string
	EmailAddress string
	Group        string
	Label        string
}

// Claims contains JWT claims information
type Claims struct {
	*Payload
	jwt.StandardClaims
}

// GenToken json web token
func GenToken(
	ctx context.Context, payload *Payload, group string, expires int64,
) (tokenStr string, err error) {
	// Handling any panic is good trust me!
	defer func() {
		if err2 := recover(); err2 != nil {
			err = fmt.Errorf("%v", err2)
		}
	}()

	if payload == nil {
		payload = &Payload{
			ID:           uuid.New().String(),
			FullName:     randomdata.SillyName(),
			EmailAddress: randomdata.Email(),
			PhoneNumber:  randomdata.PhoneNumber(),
			Group:        group,
		}
	}

	token := jwt.NewWithClaims(signingMethod, Claims{
		Payload: payload,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expires,
			Issuer:    "mcfp",
		},
	})

	// Generate the token
	return token.SignedString(signingKey)
}

// ParseToken parses a jwt token and return claims or error if token is invalid
func ParseToken(tokenString string) (claims *Claims, err error) {
	// Handling any panic is good trust me!
	defer func() {
		if err2 := recover(); err2 != nil {
			err = fmt.Errorf("%v", err2)
		}
	}()

	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			return signingKey, nil
		},
	)
	if err != nil {
		return nil, status.Errorf(
			codes.Unauthenticated, "failed to parse token with claims: %v", err,
		)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, status.Error(codes.Unauthenticated, "the token is not valid")
	}
	return claims, nil
}

// ParseFromCtx jwt token from context
func ParseFromCtx(ctx context.Context) (*Claims, error) {
	token, err := grpc_auth.AuthFromMD(ctx, "Bearer")
	if err != nil {
		return nil, status.Errorf(
			codes.Internal, "failed to get Bearer from authorization header: %v", err,
		)
	}

	return ParseToken(token)
}
