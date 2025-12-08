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

// defaultPickupPoints exposes 30 curated pickup clusters around south Bengaluru metro stops.
func defaultPickupPoints() []PickupPoint {
	return []PickupPoint{
		{ID: "pickup-wipro-gate", Name: "Wipro Gate", StationID: "station-ecity", StationName: "Electronic City", Latitude: 12.8467, Longitude: 77.6624},
		{ID: "pickup-infosys-gate", Name: "Infosys Gate", StationID: "station-ecity", StationName: "Electronic City", Latitude: 12.8459, Longitude: 77.6666},
		{ID: "pickup-velankani", Name: "Velankani Tech Park", StationID: "station-ecity", StationName: "Electronic City", Latitude: 12.8449, Longitude: 77.6615},
		{ID: "pickup-neeladri", Name: "Neeladri Road", StationID: "station-ecity", StationName: "Electronic City", Latitude: 12.8442, Longitude: 77.6574},
		{ID: "pickup-doddathogur", Name: "Doddathogur Cross", StationID: "station-ecity", StationName: "Electronic City", Latitude: 12.8365, Longitude: 77.6642},
		{ID: "pickup-singasandra", Name: "Singasandra", StationID: "station-ecity", StationName: "Electronic City", Latitude: 12.884, Longitude: 77.654},
		{ID: "pickup-kudlu-gate", Name: "Kudlu Gate", StationID: "station-hsr", StationName: "HSR Layout", Latitude: 12.8936, Longitude: 77.6513},
		{ID: "pickup-hosa-road", Name: "Hosa Road Junction", StationID: "station-konappana", StationName: "Konappana Agrahara", Latitude: 12.8721, Longitude: 77.6647},
		{ID: "pickup-konappana", Name: "Konappana Bus Stop", StationID: "station-konappana", StationName: "Konappana Agrahara", Latitude: 12.8513, Longitude: 77.6541},
		{ID: "pickup-siemens", Name: "Siemens Campus", StationID: "station-konappana", StationName: "Konappana Agrahara", Latitude: 12.8553, Longitude: 77.6515},
		{ID: "pickup-pes-it", Name: "PES IT Junction", StationID: "station-konappana", StationName: "Konappana Agrahara", Latitude: 12.8581, Longitude: 77.6493},
		{ID: "pickup-huskur", Name: "Huskur Junction", StationID: "station-huskur", StationName: "Huskur Road", Latitude: 12.8188, Longitude: 77.6924},
		{ID: "pickup-dmart", Name: "D Mart Huskur", StationID: "station-huskur", StationName: "Huskur Road", Latitude: 12.817, Longitude: 77.6972},
		{ID: "pickup-ecity-phase2", Name: "Electronic City Phase 2", StationID: "station-huskur", StationName: "Huskur Road", Latitude: 12.8149, Longitude: 77.6968},
		{ID: "pickup-bommasandra", Name: "Bommasandra Industrial", StationID: "station-bommasandra", StationName: "Bommasandra", Latitude: 12.8019, Longitude: 77.7018},
		{ID: "pickup-narayana", Name: "Narayana Health City", StationID: "station-bommasandra", StationName: "Bommasandra", Latitude: 12.8008, Longitude: 77.6846},
		{ID: "pickup-chandapura", Name: "Chandapura Circle", StationID: "station-bommasandra", StationName: "Bommasandra", Latitude: 12.8011, Longitude: 77.7039},
		{ID: "pickup-attibele", Name: "Attibele Checkpost", StationID: "station-bommasandra", StationName: "Bommasandra", Latitude: 12.7842, Longitude: 77.7721},
		{ID: "pickup-silkboard", Name: "Silk Board Flyover", StationID: "station-silkboard", StationName: "Central Silk Board", Latitude: 12.916, Longitude: 77.6239},
		{ID: "pickup-madiwala", Name: "Madiwala Police Station", StationID: "station-silkboard", StationName: "Central Silk Board", Latitude: 12.9188, Longitude: 77.6176},
		{ID: "pickup-hsr-27th", Name: "HSR 27th Main", StationID: "station-hsr", StationName: "HSR Layout", Latitude: 12.9082, Longitude: 77.6475},
		{ID: "pickup-hsr-bda", Name: "HSR BDA Complex", StationID: "station-hsr", StationName: "HSR Layout", Latitude: 12.9129, Longitude: 77.6382},
		{ID: "pickup-agara", Name: "Agara Lake", StationID: "station-hsr", StationName: "HSR Layout", Latitude: 12.9215, Longitude: 77.651},
		{ID: "pickup-btm2", Name: "BTM 2nd Stage", StationID: "station-btm", StationName: "BTM Layout", Latitude: 12.9169, Longitude: 77.6105},
		{ID: "pickup-jayadeva", Name: "Jayadeva Hospital", StationID: "station-btm", StationName: "BTM Layout", Latitude: 12.9189, Longitude: 77.5956},
		{ID: "pickup-forum", Name: "Forum Mall", StationID: "station-koramangala", StationName: "Koramangala", Latitude: 12.9349, Longitude: 77.6113},
		{ID: "pickup-sonyworld", Name: "Sony World Junction", StationID: "station-koramangala", StationName: "Koramangala", Latitude: 12.9353, Longitude: 77.6393},
		{ID: "pickup-ejipura", Name: "Ejipura Signal", StationID: "station-koramangala", StationName: "Koramangala", Latitude: 12.9304, Longitude: 77.626},
		{ID: "pickup-bellandur-gate", Name: "Bellandur Gate", StationID: "station-bellandur", StationName: "Bellandur", Latitude: 12.9378, Longitude: 77.679},
		{ID: "pickup-iblur", Name: "Iblur Junction", StationID: "station-bellandur", StationName: "Bellandur", Latitude: 12.9248, Longitude: 77.6773},
		{ID: "pickup-haralur", Name: "Haralur Road", StationID: "station-hsr", StationName: "HSR Layout", Latitude: 12.9004, Longitude: 77.6492},
	}
}
