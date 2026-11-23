// Command loadgen hammers the Matching service with realistic requests so the
// Horizontal Pod Autoscaler has something to react to.
//
// Every few seconds it prints which Matching replicas answered, e.g.
//
//	served_by  matching-6d9f-abcde=34%  matching-6d9f-fghij=33%  matching-6d9f-klmno=33%
//
// which is the clearest possible proof that scaling 1 -> 5 actually spreads
// the load (and that the headless Service + round_robin fix works).
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "lastmile/gen/go/matching"
	"lastmile/internal/catalog"
	"lastmile/internal/pkg/chassis"
)

func main() {
	addr := chassis.Getenv("MATCHING_ADDR", "dns:///matching.lastmile.svc.cluster.local:50053")
	workers := envInt("CONCURRENCY", 10)
	driversPerRequest := envInt("DRIVERS_PER_REQUEST", 200)
	pause := time.Duration(envInt("PAUSE_MS", 0)) * time.Millisecond

	conn, err := chassis.Dial(addr)
	if err != nil {
		log.Fatalf("dial %s: %v", addr, err)
	}
	defer conn.Close()
	client := pb.NewMatchingServiceClient(conn)

	log.Printf("load: %d workers -> %s (%d drivers per request)", workers, addr, driversPerRequest)

	var mu sync.Mutex
	servedBy := map[string]int{}
	errors := 0

	for w := 0; w < workers; w++ {
		go func(id int) {
			rng := rand.New(rand.NewSource(int64(id)))
			for {
				req := syntheticRequest(rng, id, driversPerRequest)
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				resp, err := client.RankDrivers(ctx, req)
				cancel()
				mu.Lock()
				if err != nil {
					errors++
				} else {
					servedBy[resp.ServedBy]++
				}
				mu.Unlock()
				if err != nil {
					time.Sleep(100 * time.Millisecond)
				}
				if pause > 0 {
					time.Sleep(pause)
				}
			}
		}(w)
	}

	for range time.Tick(5 * time.Second) {
		mu.Lock()
		report(servedBy, errors)
		servedBy, errors = map[string]int{}, 0
		mu.Unlock()
	}
}

func report(servedBy map[string]int, errors int) {
	total := 0
	names := make([]string, 0, len(servedBy))
	for name, n := range servedBy {
		names = append(names, name)
		total += n
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, fmt.Sprintf("%s=%d%%", name, servedBy[name]*100/max(total, 1)))
	}
	log.Printf("%5d req/s  replicas=%d  errors=%d  served_by  %s",
		total/5, len(names), errors, strings.Join(parts, "  "))
}

// syntheticRequest builds a rider at a random station and a crowd of drivers
// scattered around Bengaluru with random routes, so the rules do real work.
func syntheticRequest(rng *rand.Rand, worker, n int) *pb.RankDriversRequest {
	stations := catalog.Stations()
	st := stations[rng.Intn(len(stations))]
	area := st.Areas[rng.Intn(len(st.Areas))]

	drivers := make([]*pb.DriverState, n)
	for i := range drivers {
		route := []string{"pickup-" + stations[rng.Intn(len(stations))].ID, "pickup-" + stations[rng.Intn(len(stations))].ID}
		dest := "unknown"
		if rng.Intn(2) == 0 {
			dest = area.Name
		}
		drivers[i] = &pb.DriverState{
			DriverId:        fmt.Sprintf("load-driver-%d-%d", worker, i),
			PickupPointIds:  route,
			NextPickupIndex: int32(rng.Intn(2)),
			SeatsAvailable:  int32(rng.Intn(4)),
			Destination:     dest,
			Position:        &pb.GeoPoint{Latitude: 12.80 + rng.Float64()*0.15, Longitude: 77.58 + rng.Float64()*0.15},
			TripActive:      rng.Intn(5) != 0,
		}
	}
	return &pb.RankDriversRequest{
		Rider: &pb.RiderRequest{
			RiderId:       fmt.Sprintf("load-rider-%d", worker),
			PickupPointId: "pickup-" + st.ID,
			StationId:     st.ID,
			Destination:   area.Name,
			Pickup:        &pb.GeoPoint{Latitude: st.Latitude, Longitude: st.Longitude},
			ArrivalUnix:   time.Now().Add(5 * time.Minute).Unix(),
		},
		Drivers: drivers,
	}
}

func envInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v >= 0 {
		return v
	}
	return def
}
