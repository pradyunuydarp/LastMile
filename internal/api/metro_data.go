package api

import (
	"time"

	"lastmile/internal/catalog"
)

// defaultStations converts the shared catalog into the gateway's view model.
// The gateway refreshes this from the Station service at startup (see
// LoadStations); the catalog is the fallback if that service is down.
func defaultStations() []Station {
	src := catalog.Stations()
	out := make([]Station, 0, len(src))
	for _, s := range src {
		out = append(out, Station{
			ID:          s.ID,
			Name:        s.Name,
			NearbyAreas: s.AreaNames(),
			LoadFactor:  s.LoadFactor,
			Latitude:    s.Latitude,
			Longitude:   s.Longitude,
		})
	}
	return out
}

// pickupPointsFor turns stations into pickup points. In LastMile riders are
// always picked up AT a metro station, so there is one pickup per station.
func pickupPointsFor(stations []Station) []PickupPoint {
	points := make([]PickupPoint, 0, len(stations))
	for _, s := range stations {
		points = append(points, PickupPoint{
			ID:          "pickup-" + s.ID,
			Name:        s.Name + " Metro Station",
			StationID:   s.ID,
			StationName: s.Name,
			Latitude:    s.Latitude,
			Longitude:   s.Longitude,
		})
	}
	return points
}

func defaultPickupPoints() []PickupPoint {
	return pickupPointsFor(defaultStations())
}

// seedRiders are "bot" riders already waiting at stations when the gateway
// starts, so a driver can demo the full flow without a second person. Bots
// auto-confirm when a driver accepts them.
func seedRiders(now time.Time) []Rider {
	mk := func(id, name, stationID, destination string, in time.Duration) Rider {
		pickupID := "pickup-" + stationID
		var pickup *PickupPoint
		for _, p := range defaultPickupPoints() {
			if p.ID == pickupID {
				pp := p
				pickup = &pp
			}
		}
		return Rider{
			ID: id, Name: name, Destination: destination, ArrivalTime: now.Add(in),
			StationID: stationID, Status: "waiting", PickupPointID: pickupID, Pickup: pickup, Bot: true,
		}
	}
	return []Rider{
		mk("bot-rider-meera", "Meera (bot)", "station-ecity", "Wipro Gate", 5*time.Minute),
		mk("bot-rider-rahul", "Rahul (bot)", "station-konappana", "Siemens Campus", 9*time.Minute),
		mk("bot-rider-anita", "Anita (bot)", "station-huskur", "D Mart Huskur", 12*time.Minute),
	}
}

// dropoffFor returns where a rider should be dropped: their destination area
// if we know its coordinates, otherwise the station itself.
func dropoffFor(destination string, fallback *Station) (lat, lon float64, ok bool) {
	if a, found := catalog.AreaByName(destination); found {
		return a.Latitude, a.Longitude, true
	}
	if fallback != nil {
		return fallback.Latitude, fallback.Longitude, true
	}
	return 0, 0, false
}
