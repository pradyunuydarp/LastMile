// Package catalog is the single source of truth for LastMile's reference data:
// the metro stations we serve and the "nearby areas" (drop-off destinations)
// around each one.
//
// The Station service seeds itself from this catalog, and the gateway and
// Location service use it as a fallback when the Station service is
// unreachable. Before this package existed the same coordinates were copied
// into three places (and the Location service was still using San Francisco
// coordinates), so proximity detection could never fire for Bengaluru.
package catalog

import "strings"

// Area is a named place near a metro station where riders can be dropped off.
type Area struct {
	ID        string
	Name      string
	Latitude  float64
	Longitude float64
}

// Station is a metro station and the areas it serves.
type Station struct {
	ID         string
	Name       string
	Latitude   float64
	Longitude  float64
	LoadFactor float64 // rough "how busy" hint used by the UI
	Areas      []Area
}

// AreaNames returns the names of the station's nearby areas.
func (s Station) AreaNames() []string {
	names := make([]string, 0, len(s.Areas))
	for _, a := range s.Areas {
		names = append(names, a.Name)
	}
	return names
}

// ApproachRadiusMeters is how close a driver must be to a station before we
// treat them as "approaching". At ~20 km/h city speed, 1.5 km is roughly four
// to five minutes out, which is what the spec means by matching "just minutes
// before the driver reaches the metro station".
const ApproachRadiusMeters = 1500.0

// Stations returns a fresh copy of the station catalog (Electronic City and
// Outer Ring Road clusters in Bengaluru). Callers may mutate the result.
func Stations() []Station {
	out := make([]Station, len(stations))
	for i, s := range stations {
		s.Areas = append([]Area(nil), s.Areas...)
		out[i] = s
	}
	return out
}

// StationByID looks up a station by ID.
func StationByID(id string) (Station, bool) {
	for _, s := range stations {
		if s.ID == id {
			s.Areas = append([]Area(nil), s.Areas...)
			return s, true
		}
	}
	return Station{}, false
}

// AreaByName finds a nearby area by (case-insensitive) name across all stations.
func AreaByName(name string) (Area, bool) {
	want := strings.ToLower(strings.TrimSpace(name))
	if want == "" {
		return Area{}, false
	}
	for _, s := range stations {
		for _, a := range s.Areas {
			if strings.ToLower(a.Name) == want {
				return a, true
			}
		}
	}
	return Area{}, false
}

var stations = []Station{
	{
		ID: "station-ecity", Name: "Electronic City", Latitude: 12.8456, Longitude: 77.6600, LoadFactor: 0.85,
		Areas: []Area{
			{"area-wipro-gate", "Wipro Gate", 12.8467, 77.6624},
			{"area-infosys-gate", "Infosys Gate", 12.8459, 77.6666},
			{"area-velankani", "Velankani Tech Park", 12.8449, 77.6615},
			{"area-neeladri", "Neeladri Road", 12.8442, 77.6574},
			{"area-doddathogur", "Doddathogur Cross", 12.8365, 77.6642},
			{"area-singasandra", "Singasandra", 12.8840, 77.6540},
		},
	},
	{
		ID: "station-konappana", Name: "Konappana Agrahara", Latitude: 12.8519, Longitude: 77.6546, LoadFactor: 0.65,
		Areas: []Area{
			{"area-konappana", "Konappana Bus Stop", 12.8513, 77.6541},
			{"area-siemens", "Siemens Campus", 12.8553, 77.6515},
			{"area-pes-it", "PES IT Junction", 12.8581, 77.6493},
			{"area-hosa-road", "Hosa Road Junction", 12.8721, 77.6647},
		},
	},
	{
		ID: "station-huskur", Name: "Huskur Road", Latitude: 12.8209, Longitude: 77.6954, LoadFactor: 0.45,
		Areas: []Area{
			{"area-huskur", "Huskur Junction", 12.8188, 77.6924},
			{"area-dmart", "D Mart Huskur", 12.8170, 77.6972},
			{"area-ecity-phase2", "Electronic City Phase 2", 12.8149, 77.6968},
		},
	},
	{
		ID: "station-bommasandra", Name: "Bommasandra", Latitude: 12.8006, Longitude: 77.7003, LoadFactor: 0.52,
		Areas: []Area{
			{"area-bommasandra", "Bommasandra Industrial", 12.8019, 77.7018},
			{"area-narayana", "Narayana Health City", 12.8008, 77.6846},
			{"area-chandapura", "Chandapura Circle", 12.8011, 77.7039},
			{"area-attibele", "Attibele Checkpost", 12.7842, 77.7721},
		},
	},
	{
		ID: "station-silkboard", Name: "Central Silk Board", Latitude: 12.9165, Longitude: 77.6238, LoadFactor: 0.90,
		Areas: []Area{
			{"area-silkboard", "Silk Board Flyover", 12.9160, 77.6239},
			{"area-madiwala", "Madiwala Police Station", 12.9188, 77.6176},
		},
	},
	{
		ID: "station-hsr", Name: "HSR Layout", Latitude: 12.9121, Longitude: 77.6387, LoadFactor: 0.70,
		Areas: []Area{
			{"area-hsr-27th", "HSR 27th Main", 12.9082, 77.6475},
			{"area-hsr-bda", "HSR BDA Complex", 12.9129, 77.6382},
			{"area-agara", "Agara Lake", 12.9215, 77.6510},
			{"area-kudlu-gate", "Kudlu Gate", 12.8936, 77.6513},
			{"area-haralur", "Haralur Road", 12.9004, 77.6492},
		},
	},
	{
		ID: "station-btm", Name: "BTM Layout", Latitude: 12.9122, Longitude: 77.6092, LoadFactor: 0.60,
		Areas: []Area{
			{"area-btm2", "BTM 2nd Stage", 12.9169, 77.6105},
			{"area-jayadeva", "Jayadeva Hospital", 12.9189, 77.5956},
		},
	},
	{
		ID: "station-koramangala", Name: "Koramangala", Latitude: 12.9345, Longitude: 77.6266, LoadFactor: 0.58,
		Areas: []Area{
			{"area-forum", "Forum Mall", 12.9349, 77.6113},
			{"area-sonyworld", "Sony World Junction", 12.9353, 77.6393},
			{"area-ejipura", "Ejipura Signal", 12.9304, 77.6260},
		},
	},
	{
		ID: "station-bellandur", Name: "Bellandur", Latitude: 12.9381, Longitude: 77.6951, LoadFactor: 0.55,
		Areas: []Area{
			{"area-bellandur-gate", "Bellandur Gate", 12.9378, 77.6790},
			{"area-iblur", "Iblur Junction", 12.9248, 77.6773},
		},
	},
}
