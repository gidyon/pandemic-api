package main

import (
	"fmt"
	"github.com/gidyon/pandemic-api/internal/services/location/conversion"
	"github.com/gidyon/pandemic-api/pkg/api/location"
)

func main() {
	locs := []*location.Location{
		{
			Longitude: 37.085331,
			Latitude:  1.04698,
		},
		{
			Longitude: 37.085332,
			Latitude:  1.04698,
		},
		{
			Longitude: 37.085333,
			Latitude:  1.04698,
		},
		{
			Longitude: 37.085334,
			Latitude:  1.04698,
		},
		{
			Longitude: 37.085335,
			Latitude:  1.04698,
		},
	}

	for _, loc := range locs {
		print(loc)
		fmt.Println()
	}

}

func print(loc *location.Location) {
	// dms := conversion.NewDMS(loc, 1.5)
	// fmt.Printf(
	// 	"latMinute: %d\nlongMinute: %d\nlatSecond: %f\nlongSecond: %f\n",
	// 	dms.LatitudeMinutes(),
	// 	dms.LongitudeMinutes(),
	// 	dms.LatitudeSeconds(),
	// 	dms.LongitudeSeconds(),
	// )
	fmt.Printf("long: %f lat: %f\n", loc.Longitude, loc.Latitude)
	// fmt.Printf("dd is %s\n", conversion.NewDD(dms).String())
	fmt.Printf("dd is %s\n", conversion.GeoFenceID(loc, 1.5))
}
