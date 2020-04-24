package services

import (
	"time"

	"github.com/jinzhu/gorm"

	"github.com/gidyon/pandemic-api/pkg/api/location"
)

// LocationModel is a geographic location
type LocationModel struct {
	UserID        string
	Latitude      float32
	Longitude     float32
	PlaceMark     string
	GeoFenceID    string
	TimeID        string
	Timestamp     int64
	Accuracy      float32
	Speed         float32
	SpeedAccuracy float32
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
const UsersTable = "users"

// UserModel contains user data
type UserModel struct {
	PhoneNumber string
	FullName    string
	County      string
	Status      int8
	DeviceToken string
	UpdatedAt   time.Time
	CreatedAt   time.Time
}

// TableName returns the name f the table
func (*UserModel) TableName() string {
	return UsersTable
}

// ContactMessagesTable is table for contact messages
const ContactMessagesTable = "contact_messages"

// ContactMessageData is database model for contacts point
type ContactMessageData struct {
	UserPhone     string
	PatientPhone  string
	Message       string
	ContactsCount int32
	Sent          bool
	gorm.Model
}

// TableName returns the name of the table
func (*ContactMessageData) TableName() string {
	return ContactMessagesTable
}

// GeneralMessagesTable is table containing general messages
const GeneralMessagesTable = "general_messages"

// GeneralMessageData is database model for general messages
type GeneralMessageData struct {
	MessageID string
	UserPhone string
	Title     string
	Data      []byte
	Sent      bool
	gorm.Model
}

// TableName returns the name of the table
func (*GeneralMessageData) TableName() string {
	return GeneralMessagesTable
}
