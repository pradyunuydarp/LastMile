package api

// defaultStations returns the canonical list of metro stations and their nearby areas
// for Electronic City + Outer Ring Road cluster in Bengaluru. The data is intentionally
// duplicated at runtime instead of sharing references to keep the gateway state mutable
// without affecting the catalog.
func defaultStations() []Station {
	return []Station{
		{
			ID:          "station-ecity",
			Name:        "Electronic City",
			NearbyAreas: []string{"Wipro Gate", "Infosys Gate", "Velankani Tech Park", "Neeladri Road", "Doddathogur Cross", "Singasandra"},
			LoadFactor:  0.85,
			Latitude:    12.8456,
			Longitude:   77.66,
		},
		{
			ID:          "station-konappana",
			Name:        "Konappana Agrahara",
			NearbyAreas: []string{"Konappana Bus Stop", "Siemens Campus", "PES IT Junction", "Hosa Road Junction"},
			LoadFactor:  0.65,
			Latitude:    12.8519,
			Longitude:   77.6546,
		},
		{
			ID:          "station-huskur",
			Name:        "Huskur Road",
			NearbyAreas: []string{"Huskur Junction", "D Mart Huskur", "Electronic City Phase 2"},
			LoadFactor:  0.45,
			Latitude:    12.8209,
			Longitude:   77.6954,
		},
		{
			ID:          "station-bommasandra",
			Name:        "Bommasandra",
			NearbyAreas: []string{"Bommasandra Industrial", "Narayana Health City", "Chandapura Circle", "Attibele Checkpost"},
			LoadFactor:  0.52,
			Latitude:    12.8006,
			Longitude:   77.7003,
		},
		{
			ID:          "station-silkboard",
			Name:        "Central Silk Board",
			NearbyAreas: []string{"Silk Board Flyover", "Madiwala Police Station", "Singasandra"},
			LoadFactor:  0.9,
			Latitude:    12.9165,
			Longitude:   77.6238,
		},
		{
			ID:          "station-hsr",
			Name:        "HSR Layout",
			NearbyAreas: []string{"HSR 27th Main", "HSR BDA Complex", "Agara Lake", "Kudlu Gate", "Haralur Road"},
			LoadFactor:  0.7,
			Latitude:    12.9121,
			Longitude:   77.6387,
		},
		{
			ID:          "station-btm",
			Name:        "BTM Layout",
			NearbyAreas: []string{"BTM 2nd Stage", "Jayadeva Hospital", "Madiwala"},
			LoadFactor:  0.6,
			Latitude:    12.9122,
			Longitude:   77.6092,
		},
		{
			ID:          "station-koramangala",
			Name:        "Koramangala",
			NearbyAreas: []string{"Forum Mall", "Sony World", "Ejipura Signal"},
			LoadFactor:  0.58,
			Latitude:    12.9345,
			Longitude:   77.6266,
		},
		{
			ID:          "station-bellandur",
			Name:        "Bellandur",
			NearbyAreas: []string{"Bellandur Gate", "Iblur Junction", "Kasavanahalli"},
			LoadFactor:  0.55,
			Latitude:    12.9381,
			Longitude:   77.6951,
		},
	}
}

// defaultPickupPoints returns the metro stations themselves as the only valid pickup points.
func defaultPickupPoints() []PickupPoint {
	stations := defaultStations()
	points := make([]PickupPoint, 0, len(stations))
	for _, s := range stations {
		points = append(points, PickupPoint{
			ID:          "pickup-" + s.ID,
			Name:        s.Name + " Metro Station", // Explicit naming
			StationID:   s.ID,
			StationName: s.Name,
			Latitude:    s.Latitude,
			Longitude:   s.Longitude,
		})
	}
	return points
}
