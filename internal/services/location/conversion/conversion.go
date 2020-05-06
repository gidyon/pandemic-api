package conversion

import (
	"fmt"
	"github.com/gidyon/pandemic-api/pkg/api/location"
	"math"
	"strconv"
	"time"
)

// NewDMS creates a new DMS
func NewDMS(loc *location.Location, radius float64) *DMS {
	if radius == 0 {
		radius = 5.0
	}

	// Parse longitude
	long, err := strconv.ParseFloat(fmt.Sprintf("%3.7f", loc.Longitude), 32)
	if err != nil {
		long = float64(loc.Longitude)
	}
	loc.Longitude = float32(long)

	// Parse latitude
	lat, err := strconv.ParseFloat(fmt.Sprintf("%3.7f", loc.Latitude), 32)
	if err != nil {
		lat = float64(loc.Latitude)
	}
	loc.Latitude = float32(lat)

	// println(int(loc.Longitude * 1000000))

	if loc.Longitude < 0 {
		loc.Longitude = loc.Longitude - loc.Longitude - loc.Longitude
	}
	if loc.Latitude < 0 {
		loc.Latitude = loc.Latitude - loc.Latitude - loc.Latitude
	}

	latMinute := 60 * (float64(loc.Latitude) - math.Trunc(float64(loc.Latitude)))
	longMinute := 60 * (float64(loc.Longitude) - math.Trunc(float64(loc.Longitude)))

	dms := &DMS{
		location: loc,

		latDegree:  int(loc.Latitude),
		longDegree: int(loc.Longitude),

		latMinute:  int(latMinute),
		longMinute: int(longMinute),

		latSecond:  float64((latMinute - math.Trunc(latMinute)) * 60),
		longSecond: float64((longMinute - math.Trunc(longMinute)) * 60),

		radius:       radius,
		latLessZero:  loc.Latitude < 0,
		longLessZero: loc.Longitude < 0,
	}

	return dms
}

// DMS is location representation
type DMS struct {
	location        *location.Location
	latDegree       int
	longDegree      int
	latMinute       int
	longMinute      int
	latSecond       float64
	longSecond      float64
	radius          float64
	fractionalParts int
	latLessZero     bool
	longLessZero    bool
}

// Location returns the location
func (dms *DMS) Location() *location.Location {
	return dms.location
}

// LatitudeDegrees returns dms latitude degrees
func (dms *DMS) LatitudeDegrees() int {
	return dms.latDegree
}

// LongitudeDegrees returns dms longitude degrees
func (dms *DMS) LongitudeDegrees() int {
	return dms.longDegree
}

// LatitudeMinutes returns dms latitude minutes
func (dms *DMS) LatitudeMinutes() int {
	return dms.latMinute
}

// LongitudeMinutes returns dms longitude minutes
func (dms *DMS) LongitudeMinutes() int {
	return dms.longMinute
}

// LatitudeSeconds returns dms latitude seconds
func (dms *DMS) LatitudeSeconds() float64 {
	return dms.latSecond
}

// LongitudeSeconds returns dms longitude seconds
func (dms *DMS) LongitudeSeconds() float64 {
	return dms.longSecond
}

// NewDD creates a new DD
func NewDD(dms *DMS) *DD {
	dd := &DD{
		longitude: float64(dms.longDegree+(dms.longMinute/60)) + dms.longSecond/3600.00,
		latitude:  float64(dms.latDegree+(dms.latMinute/60)) + dms.latSecond/3600.00,
	}

	if dms.latLessZero {
		dd.latitude = dd.latitude - dd.latitude - dd.latitude
	}
	if dms.longLessZero {
		dd.longitude = dd.longitude - dd.longitude - dd.longitude
	}

	return dd
}

// DD is degrees decimal representation of a geopgraphic cordinate
type DD struct {
	longitude float64
	latitude  float64
}

func (dd *DD) String() string {
	return fmt.Sprintf("lat:%3.6f long:%3.6f", dd.latitude, dd.longitude)
}

func convertToDD(dms *DMS) *DD {
	return NewDD(dms)
}

func (dms *DMS) getBox(sec float64) float64 {
	doubleRange := float64(dms.radius) * 0.0036
	box := doubleRange
	for i := doubleRange; i <= (sec - math.Trunc(sec)); i += doubleRange {
		box += doubleRange
	}
	secBox := 0.0
	if sec > 0 {
		secBox = math.Trunc(sec) + box
	} else {
		secBox = math.Trunc(sec) - box
	}
	return secBox
}

// GeoFenceID returns the geo fence id of a location
func GeoFenceID(loc *location.Location, radius float64) string {
	dms := NewDMS(loc, radius)
	dms.latSecond = dms.getBox(float64(dms.latSecond))
	dms.longSecond = dms.getBox(float64(dms.longSecond))
	id := NewDD(dms).String()
	return id
}

// GetTimestampRange gets lower and upper timestamp range
func GetTimestampRange(loc *location.Location, dur time.Duration) (int64, int64) {
	min := loc.Timestamp / dur.Milliseconds()
	max := min + dur.Milliseconds()
	return min, max
}

// GetTimeID gets time id of a location
func GetTimeID(loc *location.Location, dur time.Duration) string {
	rangeTime := loc.Timestamp / dur.Milliseconds()
	max := rangeTime*dur.Milliseconds() + rangeTime
	return fmt.Sprint(max)
}
