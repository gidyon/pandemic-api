package main

import (
	"context"
	"errors"
	"github.com/Pallinder/go-randomdata"
	"github.com/gidyon/pandemic-api/internal/auth"
	"github.com/gidyon/pandemic-api/internal/services"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getToken(sqlDB *gorm.DB, phone, deviceID string) (string, error) {
	// Get user
	// Get from database
	userDB := &services.UserModel{}
	err := sqlDB.First(userDB, "phone_number=?", phone).Error
	switch {
	case err == nil:
	case errors.Is(err, gorm.ErrRecordNotFound):
		err = status.Errorf(codes.NotFound, "user not with phone number %s found: %v", phone, err)
	default:
		err = status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if err != nil {
		return "", err
	}

	// Compare id
	if userDB.DeviceToken != deviceID {
		return "", status.Errorf(codes.InvalidArgument, "device id %s do not match with user", deviceID)
	}

	// Generate token
	return auth.GenToken(context.Background(), &auth.Payload{
		ID:           uuid.New().String(),
		FullName:     randomdata.SillyName(),
		EmailAddress: randomdata.Email(),
		PhoneNumber:  phone,
		Group:        "USER",
	}, "", 0)
}
