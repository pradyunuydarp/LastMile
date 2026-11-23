// Command location runs the LastMile Location service.
package main

import (
	"context"
	"os"
	"time"

	locationpb "lastmile/gen/go/location"
	stationpb "lastmile/gen/go/station"
	"lastmile/internal/location"
	"lastmile/internal/pkg/chassis"
)

func main() {
	svc := chassis.New("location")
	server := location.NewServer(svc.Logger)
	locationpb.RegisterLocationServiceServer(svc.GRPC, server)

	// Proximity detection needs station coordinates. We start with the
	// built-in catalog and refresh from the Station service once it answers,
	// so Location can boot even if Station is down.
	if addr := os.Getenv("STATION_ADDR"); addr != "" {
		go loadStations(svc, server, addr)
	}

	if err := svc.Run(chassis.Getenv("LOCATION_GRPC_ADDR", ":50054")); err != nil {
		svc.Logger.Error("service stopped", "err", err)
		os.Exit(1)
	}
}

func loadStations(svc *chassis.Service, server *location.Server, addr string) {
	conn, err := chassis.Dial(addr)
	if err != nil {
		svc.Logger.Warn("station client setup failed; using built-in catalog", "err", err)
		return
	}
	defer conn.Close()
	client := stationpb.NewStationServiceClient(conn)

	for attempt := 1; attempt <= 30; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		resp, err := client.ListStations(ctx, &stationpb.ListStationsRequest{})
		cancel()
		if err == nil && len(resp.Stations) > 0 {
			anchors := make([]location.Anchor, 0, len(resp.Stations))
			for _, s := range resp.Stations {
				anchors = append(anchors, location.Anchor{ID: s.Id, Latitude: s.Latitude, Longitude: s.Longitude})
			}
			server.SetAnchors(anchors)
			svc.Logger.Info("loaded stations from Station service", "count", len(anchors))
			return
		}
		time.Sleep(2 * time.Second)
	}
	svc.Logger.Warn("Station service never answered; staying on built-in catalog")
}
