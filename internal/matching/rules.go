package matching

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	pb "lastmile/gen/go/matching"
	"lastmile/internal/pkg/geo"
)

// The matching rules, in plain words. A driver and a rider match when:
//
//  1. the driver has started their trip,
//  2. the rider's pickup point is on the driver's route,
//  3. the driver has not already passed that pickup,
//  4. the driver has a free seat,
//  5. they are going to the same place, and
//  6. the driver can reach the pickup no later than MaxWaitMinutes after the
//     rider's metro arrives.
//
// Everything here is a pure function of its inputs: no I/O, no clocks except
// the one passed in, no shared state. That makes the rules trivial to test and
// the service trivial to scale.
const (
	// CitySpeedKmph is a rough average speed for Bengaluru traffic.
	CitySpeedKmph = 20.0
	// MaxWaitMinutes is how long a rider will wait at the station for a car.
	MaxWaitMinutes = 15.0
)

// Verdict is the outcome of checking one driver against one rider.
type Verdict struct {
	OK             bool
	Reason         string  // why not, when OK is false
	DistanceMeters float64 // driver -> pickup; +Inf when position is unknown
	ETAMinutes     float64 // driver -> pickup; +Inf when position is unknown
}

// Check applies the rules to one driver/rider pair.
func Check(d *pb.DriverState, r *pb.RiderRequest, now time.Time) Verdict {
	v := Verdict{DistanceMeters: math.Inf(1), ETAMinutes: math.Inf(1)}
	if d.GetPosition() != nil && r.GetPickup() != nil &&
		geo.Known(d.Position.Latitude, d.Position.Longitude) &&
		geo.Known(r.Pickup.Latitude, r.Pickup.Longitude) {
		v.DistanceMeters = geo.HaversineMeters(d.Position.Latitude, d.Position.Longitude, r.Pickup.Latitude, r.Pickup.Longitude)
		v.ETAMinutes = v.DistanceMeters / 1000 / CitySpeedKmph * 60
	}

	if !d.GetTripActive() {
		v.Reason = "driver has not started a trip"
		return v
	}
	idx := indexOf(d.GetPickupPointIds(), r.GetPickupPointId())
	if idx < 0 {
		v.Reason = "pickup is not on the driver's route"
		return v
	}
	if idx < int(d.GetNextPickupIndex()) {
		v.Reason = "driver already passed this pickup"
		return v
	}
	if d.GetSeatsAvailable() <= 0 {
		v.Reason = "no free seats"
		return v
	}
	if !SameDestination(d.GetDestination(), r.GetDestination()) {
		v.Reason = fmt.Sprintf("driver is going to %q, rider to %q", d.GetDestination(), r.GetDestination())
		return v
	}
	if r.GetArrivalUnix() > 0 && !math.IsInf(v.ETAMinutes, 1) {
		minutesUntilArrival := math.Max(0, time.Unix(r.ArrivalUnix, 0).Sub(now).Minutes())
		if v.ETAMinutes > minutesUntilArrival+MaxWaitMinutes {
			v.Reason = fmt.Sprintf("driver is %.0f min away; rider would wait more than %.0f min", v.ETAMinutes, MaxWaitMinutes)
			return v
		}
	}

	v.OK = true
	return v
}

// SameDestination implements "the rider's destination must match the
// driver's". A driver with no fixed destination ("" or "unknown") accepts any
// rider. Otherwise we compare case-insensitively and allow one name to
// contain the other ("Wipro Gate" matches "Wipro Gate · Electronic City").
func SameDestination(driverDest, riderDest string) bool {
	d := strings.ToLower(strings.TrimSpace(driverDest))
	r := strings.ToLower(strings.TrimSpace(riderDest))
	if d == "" || d == "unknown" || r == "" {
		return true
	}
	return d == r || strings.Contains(d, r) || strings.Contains(r, d)
}

// RankDrivers returns the eligible drivers for a rider (closest first) and
// the reasons the others were ruled out.
func RankDrivers(r *pb.RiderRequest, drivers []*pb.DriverState, now time.Time) ([]*pb.DriverCandidate, []*pb.Rejection) {
	var out []*pb.DriverCandidate
	var rejected []*pb.Rejection
	for _, d := range drivers {
		v := Check(d, r, now)
		if !v.OK {
			rejected = append(rejected, &pb.Rejection{Id: d.GetDriverId(), Reason: v.Reason})
			continue
		}
		out = append(out, &pb.DriverCandidate{
			DriverId:       d.GetDriverId(),
			Name:           d.GetName(),
			DistanceMeters: finite(v.DistanceMeters),
			EtaMinutes:     finite(v.ETAMinutes),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return sortKey(out[i].DistanceMeters) < sortKey(out[j].DistanceMeters)
	})
	return out, rejected
}

// FindRiders returns the riders a driver can take (earliest metro arrival
// first) and the reasons the others were ruled out.
func FindRiders(d *pb.DriverState, riders []*pb.RiderRequest, now time.Time) ([]*pb.RiderCandidate, []*pb.Rejection) {
	type hit struct {
		r *pb.RiderRequest
		v Verdict
	}
	var hits []hit
	var rejected []*pb.Rejection
	for _, r := range riders {
		v := Check(d, r, now)
		if !v.OK {
			rejected = append(rejected, &pb.Rejection{Id: r.GetRiderId(), Reason: v.Reason})
			continue
		}
		hits = append(hits, hit{r, v})
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].r.GetArrivalUnix() != hits[j].r.GetArrivalUnix() {
			return hits[i].r.GetArrivalUnix() < hits[j].r.GetArrivalUnix()
		}
		return sortKey(hits[i].v.DistanceMeters) < sortKey(hits[j].v.DistanceMeters)
	})
	out := make([]*pb.RiderCandidate, 0, len(hits))
	for _, h := range hits {
		out = append(out, &pb.RiderCandidate{RiderId: h.r.GetRiderId(), DistanceMeters: finite(h.v.DistanceMeters)})
	}
	return out, rejected
}

func indexOf(items []string, target string) int {
	for i, item := range items {
		if item == target {
			return i
		}
	}
	return -1
}

// finite maps +Inf (unknown distance) to -1 so it survives JSON/proto.
func finite(v float64) float64 {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return -1
	}
	return v
}

// sortKey puts unknown distances (-1) last.
func sortKey(v float64) float64 {
	if v < 0 {
		return math.MaxFloat64
	}
	return v
}
