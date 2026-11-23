// Package geo holds the small amount of geometry LastMile needs.
package geo

import "math"

const earthRadiusMeters = 6371000.0

// HaversineMeters returns the great-circle distance between two points.
func HaversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// Known reports whether a coordinate looks set (0,0 is our "unknown").
func Known(lat, lon float64) bool {
	return lat != 0 || lon != 0
}

// Lerp linearly interpolates between two points; t runs from 0 to 1.
// Good enough for the few-kilometre hops in our simulated trips.
func Lerp(lat1, lon1, lat2, lon2, t float64) (float64, float64) {
	return lat1 + (lat2-lat1)*t, lon1 + (lon2-lon1)*t
}
