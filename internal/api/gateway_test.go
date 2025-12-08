package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSnapshotHandler(t *testing.T) {
	gw := NewGateway(nil, nil, nil, nil)
	gw.drivers = []Driver{
		{
			ID:             "driver-ramesh",
			Name:           "Ramesh",
			CarDetails:     "Hyundai",
			SeatsAvailable: 2,
			Route: Route{
				ID:               "route-1",
				TargetStationIDs: []string{"station-ecity"},
				Destination:      "Wipro Gate",
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/aggregates/snapshot", nil)
	rr := httptest.NewRecorder()

	gw.SnapshotHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var snapshot BackendSnapshot
	if err := json.Unmarshal(rr.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("failed to decode snapshot: %v", err)
	}

	if len(snapshot.Drivers) == 0 || len(snapshot.Stations) == 0 {
		t.Fatalf("snapshot missing core data: %+v", snapshot)
	}
}

func TestMatchHandlerCreatesTrip(t *testing.T) {
	gw := NewGateway(nil, nil, nil, nil)
	gw.drivers = []Driver{
		{
			ID:             "driver-ramesh",
			Name:           "Ramesh",
			CarDetails:     "Hyundai",
			SeatsAvailable: 2,
			Route: Route{
				ID:               "route-1",
				TargetStationIDs: []string{"station-ecity"},
				Destination:      "Wipro Gate",
			},
		},
	}

	body, _ := json.Marshal(matchRequest{
		DriverID:  "driver-ramesh",
		StationID: "station-ecity",
	})

	req := httptest.NewRequest(http.MethodPost, "/matching/match", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	gw.MatchHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var trip Trip
	if err := json.Unmarshal(rr.Body.Bytes(), &trip); err != nil {
		t.Fatalf("failed to decode trip: %v", err)
	}

	if trip.DriverID != "driver-ramesh" || trip.StationID != "station-ecity" {
		t.Fatalf("unexpected trip payload: %+v", trip)
	}

	snapshot := gw.snapshot()
	if snapshot.HighlightTrip == nil || snapshot.HighlightTrip.ID != trip.ID {
		t.Fatalf("snapshot highlight not updated: %+v", snapshot.HighlightTrip)
	}
}

func TestDriverRequestsHandler(t *testing.T) {
	gw := NewGateway(nil, nil, nil, nil)
	pickup := gw.pickupPoints[0]
	pp := pickup
	gw.drivers = []Driver{
		{
			ID:             "driver-1",
			Name:           "Lakshmi",
			CarDetails:     "Suzuki XL6",
			SeatsAvailable: 3,
			Route: Route{
				ID:               "route-99",
				TargetStationIDs: []string{"station-ecity"},
				Destination:      "Wipro Gate",
			},
			Latitude:  12.8456,
			Longitude: 77.66,
		},
	}
	gw.riders = []Rider{
		{
			ID:            "rider-1",
			Name:          "Priya",
			Destination:   "Wipro Gate",
			ArrivalTime:   time.Now().Add(5 * time.Minute),
			StationID:     "station-ecity",
			Status:        "waiting",
			PickupPointID: pp.ID,
			Pickup:        &pp,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/drivers/requests?driverId=driver-1", nil)
	rr := httptest.NewRecorder()
	gw.DriverRequestsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var payload driverRequestsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Driver.ID != "driver-1" {
		t.Fatalf("unexpected driver summary: %+v", payload.Driver)
	}
	if len(payload.Requests) != 1 || payload.Requests[0].ID != "rider-1" {
		t.Fatalf("unexpected rider requests: %+v", payload.Requests)
	}
}

func TestBookRideHandler(t *testing.T) {
	gw := NewGateway(nil, nil, nil, nil)
	pickup := gw.pickupPoints[0]
	gw.drivers = []Driver{
		{
			ID:             "driver-2",
			Name:           "Ravi",
			CarDetails:     "Toyota Innova",
			SeatsAvailable: 1,
			Route: Route{
				ID:               "route-2",
				TargetStationIDs: []string{"station-ecity"},
				Destination:      "Wipro Gate",
			},
			Latitude:  12.8456,
			Longitude: 77.6603,
		},
	}
	gw.riders = nil

	body, _ := json.Marshal(bookRideRequest{
		Command:       "book",
		Name:          "Ananya",
		PickupPointID: pickup.ID,
	})

	req := httptest.NewRequest(http.MethodPost, "/rides/book", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	gw.BookRideHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var payload bookRideResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Status != "awaiting_rider" {
		t.Fatalf("expected awaiting_rider status, got %s (attempts=%+v)", payload.Status, payload.Attempts)
	}
	if payload.Trip == nil {
		t.Fatalf("expected trip in response")
	}
	if len(payload.Attempts) == 0 || !payload.Attempts[0].Accepted {
		t.Fatalf("expected first attempt accepted, got %+v", payload.Attempts)
	}
}

func TestDriverAcceptHandler(t *testing.T) {
	gw := NewGateway(nil, nil, nil, nil)
	pickup := gw.pickupPoints[0]
	rider := Rider{
		ID:            "rider-accept",
		Name:          "Sahana",
		Destination:   pickup.Name,
		ArrivalTime:   time.Now().Add(4 * time.Minute),
		StationID:     pickup.StationID,
		Status:        "waiting",
		PickupPointID: pickup.ID,
		Pickup:        &pickup,
	}
	gw.riders = append([]Rider{rider}, gw.riders...)
	gw.drivers = []Driver{
		{
			ID:             "driver-accept",
			Name:           "Meera",
			CarDetails:     "Sedan",
			SeatsAvailable: 2,
			Route: Route{
				ID:               "route-accept",
				TargetStationIDs: []string{pickup.StationID},
				Destination:      pickup.Name,
			},
		},
	}

	body, _ := json.Marshal(driverAcceptRequest{DriverID: "driver-accept", RiderID: "rider-accept"})
	req := httptest.NewRequest(http.MethodPost, "/drivers/requests/accept", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	gw.DriverAcceptHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var trip Trip
	if err := json.Unmarshal(rr.Body.Bytes(), &trip); err != nil {
		t.Fatalf("failed to decode trip: %v", err)
	}
	if trip.RiderID != "rider-accept" || trip.DriverID != "driver-accept" {
		t.Fatalf("unexpected trip: %+v", trip)
	}
	if trip.PickupPointID != pickup.ID {
		t.Fatalf("expected pickup %s, got %s", pickup.ID, trip.PickupPointID)
	}
}
