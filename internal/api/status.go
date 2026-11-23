package api

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"time"

	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	stationpb "lastmile/gen/go/station"
	"lastmile/internal/catalog"
)

// StatusChecker asks every downstream service for its gRPC health status.
// It powers GET /system/status, which is handy to keep on screen while you
// kill pods during the resilience demo.
type StatusChecker struct {
	started  time.Time
	instance string
	conns    map[string]*grpc.ClientConn
}

// NewStatusChecker builds a checker over named client connections.
func NewStatusChecker(instance string, conns map[string]*grpc.ClientConn) *StatusChecker {
	return &StatusChecker{started: time.Now(), instance: instance, conns: conns}
}

type serviceStatus struct {
	Name      string `json:"name"`
	Status    string `json:"status"` // SERVING, NOT_SERVING, DOWN
	LatencyMs int64  `json:"latencyMs"`
	Error     string `json:"error,omitempty"`
}

func (c *StatusChecker) check() []serviceStatus {
	out := make([]serviceStatus, 0, len(c.conns))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for name, conn := range c.conns {
		wg.Add(1)
		go func(name string, conn *grpc.ClientConn) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
			defer cancel()
			start := time.Now()
			resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
			st := serviceStatus{Name: name, LatencyMs: time.Since(start).Milliseconds()}
			if err != nil {
				st.Status, st.Error = "DOWN", err.Error()
			} else {
				st.Status = resp.GetStatus().String()
			}
			mu.Lock()
			out = append(out, st)
			mu.Unlock()
		}(name, conn)
	}
	wg.Wait()
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// AttachStatusChecker enables /system/status.
func (g *Gateway) AttachStatusChecker(c *StatusChecker) {
	g.mu.Lock()
	g.statusChecker = c
	g.mu.Unlock()
}

// HealthzHandler is the gateway's liveness probe: "is this process alive?".
// It deliberately does NOT check downstream services — if it did, one dead
// service would make Kubernetes restart a perfectly healthy gateway too.
func (g *Gateway) HealthzHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// SystemStatusHandler reports the health of every service plus which
// Matching replicas have been answering.
func (g *Gateway) SystemStatusHandler(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	checker := g.statusChecker
	servedBy := make(map[string]int, len(g.matchedByCount))
	for k, v := range g.matchedByCount {
		servedBy[k] = v
	}
	last := g.lastMatchedBy
	driverUp := g.driverServiceUp
	g.mu.Unlock()

	resp := map[string]any{
		"matching": map[string]any{"lastServedBy": last, "servedBy": servedBy},
		// False while the Driver service is down and the snapshot is cached.
		"driverDataFresh": driverUp,
		"checkedAt":       time.Now().UTC(),
	}
	if checker != nil {
		resp["gateway"] = map[string]any{
			"instance":      checker.instance,
			"uptimeSeconds": int(time.Since(checker.started).Seconds()),
		}
		resp["services"] = checker.check()
	}
	writeJSON(w, http.StatusOK, resp)
}

// LoadStations refreshes the station list from the Station service, retrying
// in the background. Until it succeeds the gateway serves the built-in
// catalog, so a Station outage at boot never blocks the gateway.
func (g *Gateway) LoadStations(client stationpb.StationServiceClient) {
	for attempt := 1; attempt <= 30; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		resp, err := client.ListStations(ctx, &stationpb.ListStationsRequest{})
		cancel()
		if err == nil && len(resp.Stations) > 0 {
			stations := make([]Station, 0, len(resp.Stations))
			for _, s := range resp.Stations {
				load := 0.5
				if c, ok := catalog.StationByID(s.Id); ok {
					load = c.LoadFactor
				}
				stations = append(stations, Station{
					ID: s.Id, Name: s.Name, NearbyAreas: append([]string(nil), s.NearbyAreas...),
					LoadFactor: load, Latitude: s.Latitude, Longitude: s.Longitude,
				})
			}
			g.mu.Lock()
			g.stations = stations
			g.pickupPoints = pickupPointsFor(stations)
			g.mu.Unlock()
			g.logger.Info("loaded stations from Station service", "count", len(stations))
			return
		}
		time.Sleep(2 * time.Second)
	}
	g.logger.Warn("Station service never answered; serving built-in catalog")
}
