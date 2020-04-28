package services

import (
	"github.com/jinzhu/gorm"

	"github.com/gidyon/pandemic-api/pkg/api/location"
)

// LocationModel is a geographic location
type LocationModel struct {
	UserID        string  `gorm:"type:varchar(50);not null"`
	Latitude      float32 `gorm:"type:float(10);not null"`
	Longitude     float32 `gorm:"type:float(10);not null"`
	PlaceMark     string  `gorm:"type:varchar(50);not null"`
	GeoFenceID    string  `gorm:"type:varchar(50);not null"`
	TimeID        string  `gorm:"type:varchar(50);not null"`
	Timestamp     int64   `gorm:"type:int(15);not null"`
	Accuracy      float32 `gorm:"type:float(10);not null"`
	Speed         float32 `gorm:"type:float(10);not null"`
	SpeedAccuracy float32 `gorm:"type:float(10);not null"`
	gorm.Model
}

// GetLocationDB creates location model from given location proto
func GetLocationDB(locationPB *location.Location) *LocationModel {
	return &LocationModel{
		Latitude:      locationPB.Latitude,
		Longitude:     locationPB.Longitude,
		PlaceMark:     locationPB.Placemark,
		GeoFenceID:    locationPB.GeoFenceId,
		TimeID:        locationPB.TimeId,
		Timestamp:     locationPB.Timestamp,
		Accuracy:      locationPB.Accuracy,
		Speed:         locationPB.Speed,
		SpeedAccuracy: locationPB.SpeedAccuracy,
	}
}

// GetLocationPB creates proto location from location model
func GetLocationPB(locationDB *LocationModel) *location.Location {
	return &location.Location{
		Longitude:     locationDB.Longitude,
		Latitude:      locationDB.Latitude,
		Placemark:     locationDB.PlaceMark,
		GeoFenceId:    locationDB.GeoFenceID,
		TimeId:        locationDB.TimeID,
		Timestamp:     locationDB.CreatedAt.Unix(),
		Accuracy:      locationDB.Accuracy,
		Speed:         locationDB.Speed,
		SpeedAccuracy: locationDB.SpeedAccuracy,
	}
}

// UsersTable is users table
const UsersTable = "app_users"

// UserModel contains user data
type UserModel struct {
	PhoneNumber string `gorm:"type:varchar(15);not null"`
	FullName    string `gorm:"type:varchar(50);not null"`
	County      string `gorm:"type:varchar(50);not null"`
	Status      int8   `gorm:"type:tinyint(1);default:0"`
	DeviceToken string `gorm:"type:varchar(256);not null"`
	gorm.Model
}

// TableName returns the name f the table
func (*UserModel) TableName() string {
	return UsersTable
}

// MessagesTable is messages table
const MessagesTable = "messages"

// Message model
type Message struct {
	UserPhone string `gorm:"type:varchar(15);not null"`
	Title     string `gorm:"type:varchar(30);not null"`
	Message   string `gorm:"type:varchar(256);not null"`
	DateTime  string `gorm:"type:varchar(30);not null"`
	Data      []byte `gorm:"type:json"`
	Sent      bool   `gorm:"type:tinyint(1);default:0"`
	Type      int8   `gorm:"type:tinyint(1);default:0"`
	gorm.Model
}

// TableName returns the name of the table
func (*Message) TableName() string {
	return MessagesTable
}
