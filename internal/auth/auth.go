package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/gidyon/pandemic-api/internal/services"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthenticateRequest authenticates incoming request
func AuthenticateRequest(ctx context.Context) error {
	_, err := ParseFromCtx(ctx)
	if err != nil {
		return err
	}
	return nil
}

// AuthenticateUser authenticates credentials in their jwt with phone number
func AuthenticateUser(ctx context.Context, phone string) error {
	// Validation
	switch {
	case phone == "":
		return services.MissingFieldError("phone")
	}

	claims, err := ParseFromCtx(ctx)
	if err != nil {
		return err
	}
	if claims.PhoneNumber != phone {
		return status.Error(codes.Unauthenticated, "phone do not match")
	}
	return nil
}

// AuthenticateGroup authenticates whether token belongs to member of a particular group
func AuthenticateGroup(ctx context.Context, group string) (*Payload, error) {
	claims, err := ParseFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	err = matchGroup(claims.Payload.Group, group)
	if err != nil {
		return nil, err
	}

	return claims.Payload, nil
}

// AuthenticateGroupAndID authenticates member of a particular group and ad having given ID
func AuthenticateGroupAndID(ctx context.Context, group, tokenID string) error {
	payload, err := AuthenticateGroup(ctx, group)
	if err != nil {
		return err
	}
	if payload.ID != tokenID {
		return errors.New("Token ID do not match")
	}
	return nil
}

// AuthenticateGroupFromToken authenticates member of a particular group from token
func AuthenticateGroupFromToken(token, group string) (*Payload, error) {
	claims, err := ParseToken(token)
	if err != nil {
		return nil, err
	}
	err = matchGroup(claims.Payload.Group, group)
	if err != nil {
		return nil, err
	}
	return claims.Payload, nil
}

// AuthenticateGroupAndIDFromToken authenticates group and tokenID of a particular group
func AuthenticateGroupAndIDFromToken(token string, group string, tokenID string) error {
	claims, err := AuthenticateGroupFromToken(token, group)
	if err != nil {
		return err
	}
	if claims.ID != tokenID {
		return errors.New("Token ID do not match")
	}
	return nil
}

func addTokenMD(ctx context.Context, token string) context.Context {
	return metadata.NewIncomingContext(
		ctx, metadata.Pairs("authorization", fmt.Sprintf("Bearer %s", token)),
	)
}

func matchGroup(claimGroupID, group string) error {
	if claimGroupID != group {
		return status.Errorf(
			codes.PermissionDenied,
			"permission denied: group %s does not belong to %s group",
			claimGroupID,
			group,
		)
	}
	return nil
}
