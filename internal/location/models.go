package location

import (
	"github.com/gidyon/fightcovid19/pkg/api/location"
	"time"
)

type locationModel struct {
	ID        int64
	UserID    string
	Latitude  float32
	Longitude float32
	PlaceMark string
	CreatedAt time.Time
}

func getLocationDB(locationPB *location.Location) *locationModel {
	return &locationModel{
		Latitude:  locationPB.Latitude,
		Longitude: locationPB.Longitude,
		PlaceMark: locationPB.PlaceMark,
	}
}

func getLocationPB(locationDB *locationModel) *location.Location {
	return &location.Location{
		Timestamp: locationDB.CreatedAt.Unix(),
		Longitude: locationDB.Longitude,
		Latitude:  locationDB.Latitude,
		PlaceMark: locationDB.PlaceMark,
	}
}

const usersTable = "users"

type userModel struct {
	UserID      string
	PhoneNumber string
	Status      string
	UpdatedAt   time.Time
	CreatedAt   time.Time
}

func (*userModel) TableName() string {
	return usersTable
}
