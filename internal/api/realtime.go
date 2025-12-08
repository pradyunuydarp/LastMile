package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	socketio "github.com/googollee/go-socket.io"
)

type RealtimeHub struct {
	server  *socketio.Server
	logger  *slog.Logger
	gateway *Gateway

	mu               sync.Mutex
	drivers          map[string]*driverSession
	riders           map[string]*riderSession
	pending          map[string]*pendingQueue
	rooms            map[string]*tripRoom
	pendingApprovals map[string]*pendingApproval
}

type driverSession struct {
	id        string
	name      string
	conn      socketio.Conn
	connected time.Time
}

type riderSession struct {
	id        string
	name      string
	conn      socketio.Conn
	connected time.Time
}

type pendingQueue struct {
	rider    Rider
	station  *Station
	pickup   *PickupPoint
	attempts []driverAttempt
	index    int
	waiting  string
	timer    *time.Timer
}

type pendingApproval struct {
	trip    Trip
	pickup  *PickupPoint
	station *Station
	rider   *Rider
	timer   *time.Timer
}

type tripRoom struct {
	id         string
	driverID   string
	driverName string
	riderID    string
	riderName  string
	pickup     *PickupPoint
	station    *Station
	status     string
	updatedAt  time.Time
	lastLat    float64
	lastLon    float64
}

type sessionInit struct {
	Role   string `json:"role"`
	UserID string `json:"userId"`
	Name   string `json:"name"`
}

type driverResponsePayload struct {
	RiderID string `json:"riderId"`
	Accept  bool   `json:"accept"`
	Reason  string `json:"reason,omitempty"`
}

type tripStatusPayload struct {
	TripID      string          `json:"tripId"`
	Status      string          `json:"status"`
	DriverID    string          `json:"driverId"`
	RiderID     string          `json:"riderId"`
	Pickup      *PickupPoint    `json:"pickup,omitempty"`
	Station     *Station        `json:"station,omitempty"`
	LastLat     float64         `json:"latitude,omitempty"`
	LastLon     float64         `json:"longitude,omitempty"`
	RecordedAt  time.Time       `json:"recordedAt"`
	Description string          `json:"description,omitempty"`
	Trip        *Trip           `json:"trip,omitempty"`
	Rider       *Rider          `json:"rider,omitempty"`
	Attempts    []driverAttempt `json:"attempts,omitempty"`
}

type riderApprovalPayload struct {
	TripID string `json:"tripId"`
	Accept bool   `json:"accept"`
	Reason string `json:"reason,omitempty"`
}

func NewRealtimeHub(logger *slog.Logger) *RealtimeHub {
	l := logger
	if l == nil {
		l = slog.Default()
	}

	server := socketio.NewServer(nil)
	hub := &RealtimeHub{
		server:           server,
		logger:           l.With("component", "realtime"),
		drivers:          make(map[string]*driverSession),
		riders:           make(map[string]*riderSession),
		pending:          make(map[string]*pendingQueue),
		rooms:            make(map[string]*tripRoom),
		pendingApprovals: make(map[string]*pendingApproval),
	}

	hub.registerHandlers()
	go server.Serve()

	return hub
}

func (h *RealtimeHub) Close() {
	if h == nil || h.server == nil {
		return
	}
	h.server.Close()
}

func (h *RealtimeHub) Handler() http.Handler {
	return h.server
}

func (h *RealtimeHub) BindGateway(gw *Gateway) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.gateway = gw
}

func (h *RealtimeHub) registerHandlers() {
	h.server.OnConnect("/", func(conn socketio.Conn) error {
		conn.SetContext(map[string]any{})
		h.logger.Info("socket connected", "sid", conn.ID())
		return nil
	})

	h.server.OnEvent("/", "session:init", func(conn socketio.Conn, payload sessionInit) {
		role := normalizeRole(payload.Role)
		if role == "" || payload.UserID == "" {
			conn.Emit("session:error", map[string]string{"message": "role and userId required"})
			return
		}
		if role == "driver" {
			h.registerDriver(conn, payload)
		} else {
			h.registerRider(conn, payload)
		}
	})

	h.server.OnEvent("/", "driver:rider-response", func(conn socketio.Conn, payload driverResponsePayload) {
		driverID := h.driverIDForConn(conn)
		if driverID == "" {
			return
		}
		h.handleDriverResponse(driverID, payload)
	})

	h.server.OnEvent("/", "rider:approval-response", func(conn socketio.Conn, payload riderApprovalPayload) {
		riderID := h.riderIDForConn(conn)
		if riderID == "" || payload.TripID == "" {
			return
		}
		h.handleRiderApproval(payload.TripID, riderID, payload.Accept, payload.Reason)
	})

	h.server.OnEvent("/", "trip:complete", func(conn socketio.Conn, tripID string) {
		driverID := h.driverIDForConn(conn)
		if driverID == "" {
			return
		}
		h.completeTripRoom(tripID, "completed (manual)")
	})

	h.server.OnDisconnect("/", func(conn socketio.Conn, reason string) {
		h.removeConn(conn)
		h.logger.Info("socket disconnected", "sid", conn.ID(), "reason", reason)
	})

	h.server.OnError("/", func(conn socketio.Conn, err error) {
		h.logger.Warn("socket error", "sid", conn.ID(), "err", err)
	})
}

func (h *RealtimeHub) registerDriver(conn socketio.Conn, payload sessionInit) {
	session := &driverSession{
		id:        payload.UserID,
		name:      payload.Name,
		conn:      conn,
		connected: time.Now(),
	}

	h.mu.Lock()
	h.drivers[payload.UserID] = session
	h.mu.Unlock()

	conn.Emit("session:ack", map[string]string{
		"role":   "driver",
		"userId": payload.UserID,
	})

	go h.RefreshDriverQueue(payload.UserID)
}

func (h *RealtimeHub) registerRider(conn socketio.Conn, payload sessionInit) {
	session := &riderSession{
		id:        payload.UserID,
		name:      payload.Name,
		conn:      conn,
		connected: time.Now(),
	}

	h.mu.Lock()
	h.riders[payload.UserID] = session
	h.mu.Unlock()

	conn.Emit("session:ack", map[string]string{
		"role":   "rider",
		"userId": payload.UserID,
	})

	go h.deliverPendingApprovals(payload.UserID)
}

func (h *RealtimeHub) removeConn(conn socketio.Conn) {
	h.mu.Lock()
	for id, session := range h.drivers {
		if session.conn == conn {
			delete(h.drivers, id)
			h.mu.Unlock()
			return
		}
	}
	var cancelTrips []string
	for id, session := range h.riders {
		if session.conn == conn {
			delete(h.riders, id)
			for tripID, ctx := range h.pendingApprovals {
				if ctx.rider != nil && ctx.rider.ID == id {
					if ctx.timer != nil {
						ctx.timer.Stop()
					}
					delete(h.pendingApprovals, tripID)
					cancelTrips = append(cancelTrips, tripID)
				}
			}
			break
		}
	}
	h.mu.Unlock()
	for _, tripID := range cancelTrips {
		if err := h.gateway.CancelTrip(tripID, "rider_disconnected"); err != nil {
			h.logger.Warn("cancel trip on disconnect failed", "tripId", tripID, "err", err)
		}
	}
}

func (h *RealtimeHub) driverIDForConn(conn socketio.Conn) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, session := range h.drivers {
		if session.conn == conn {
			return id
		}
	}
	return ""
}

func (h *RealtimeHub) riderIDForConn(conn socketio.Conn) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, session := range h.riders {
		if session.conn == conn {
			return id
		}
	}
	return ""
}

func (h *RealtimeHub) handleDriverResponse(driverID string, payload driverResponsePayload) {
	h.mu.Lock()
	queue, ok := h.pending[payload.RiderID]
	if !ok || queue.waiting != driverID {
		h.mu.Unlock()
		return
	}
	if queue.timer != nil {
		queue.timer.Stop()
	}
	if payload.Accept {
		delete(h.pending, payload.RiderID)
		h.mu.Unlock()
		h.startTripRoom(driverID, queue)
		return
	}

	queue.index++
	queue.waiting = ""
	h.pending[payload.RiderID] = queue
	h.mu.Unlock()
	go h.dispatchNext(queue)
}

func (h *RealtimeHub) dispatchNext(queue *pendingQueue) {
	h.mu.Lock()
	if queue.index >= len(queue.attempts) {
		delete(h.pending, queue.rider.ID)
		h.mu.Unlock()
		h.notifyRiderStatus(queue.rider.ID, tripStatusPayload{
			RiderID:    queue.rider.ID,
			Status:     "no_drivers",
			RecordedAt: time.Now(),
			Attempts:   queue.attempts,
		})
		return
	}

	attempt := queue.attempts[queue.index]
	queue.waiting = attempt.DriverID
	if queue.timer != nil {
		queue.timer.Stop()
	}
	timer := time.AfterFunc(20*time.Second, func() {
		h.handleDriverTimeout(queue.rider.ID, attempt.DriverID)
	})
	queue.timer = timer
	h.pending[queue.rider.ID] = queue
	h.mu.Unlock()

	payload := map[string]any{
		"rider": map[string]any{
			"id":          queue.rider.ID,
			"name":        queue.rider.Name,
			"destination": queue.rider.Destination,
			"pickupId":    queue.rider.PickupPointID,
			"pickupName":  pickupName(queue.pickup),
			"status":      queue.rider.Status,
		},
		"pickup":  queue.pickup,
		"station": queue.station,
		"attempt": queue.index + 1,
		"total":   len(queue.attempts),
	}
	h.emitToDriver(attempt.DriverID, "driver:rider-offer", payload)
}

func (h *RealtimeHub) handleDriverTimeout(riderID, driverID string) {
	h.mu.Lock()
	queue, ok := h.pending[riderID]
	if !ok || queue.waiting != driverID {
		h.mu.Unlock()
		return
	}
	queue.index++
	queue.waiting = ""
	queue.timer = nil
	h.pending[riderID] = queue
	h.mu.Unlock()
	go h.dispatchNext(queue)
}

func (h *RealtimeHub) EnqueueRiderRequest(rider Rider, station *Station, pickup *PickupPoint, attempts []driverAttempt) {
	if len(attempts) == 0 {
		h.notifyRiderStatus(rider.ID, tripStatusPayload{
			Status:     "no_drivers",
			RiderID:    rider.ID,
			RecordedAt: time.Now(),
		})
		return
	}

	queue := &pendingQueue{
		rider:    rider,
		station:  station,
		pickup:   pickup,
		attempts: attempts,
		index:    0,
	}

	h.mu.Lock()
	h.pending[rider.ID] = queue
	h.mu.Unlock()

	go h.dispatchNext(queue)
}

func (h *RealtimeHub) startTripRoom(driverID string, queue *pendingQueue) {
	if h.gateway == nil {
		return
	}

	trip, err := h.gateway.acceptTripFromRealtime(driverID, queue.rider.ID)
	if err != nil {
		h.emitToDriver(driverID, "driver:rider-error", map[string]string{
			"message": err.Error(),
		})
		go h.dispatchNext(queue)
		return
	}

	room := &tripRoom{
		id:         trip.ID,
		driverID:   driverID,
		driverName: queue.attempts[queue.index].DriverName,
		riderID:    trip.RiderID,
		riderName:  queue.rider.Name,
		pickup:     queue.pickup,
		station:    queue.station,
		status:     "awaiting_pickup",
		updatedAt:  time.Now(),
	}
	h.mu.Lock()
	h.rooms[room.id] = room
	h.mu.Unlock()

	h.joinRoom(driverID, room.id)
	h.joinRiderRoom(trip.RiderID, room.id)

	payload := tripStatusPayload{
		TripID:     trip.ID,
		Status:     room.status,
		DriverID:   driverID,
		RiderID:    trip.RiderID,
		Pickup:     queue.pickup,
		Station:    queue.station,
		RecordedAt: time.Now(),
		Trip:       &trip,
		Rider:      &queue.rider,
	}
	h.emitToDriver(driverID, "trip:room-created", payload)
	h.notifyRiderStatus(trip.RiderID, payload)
}

func (h *RealtimeHub) BroadcastLocation(driverID string, lat, lon float64) {
	h.mu.Lock()
	rooms := make([]*tripRoom, 0, len(h.rooms))
	for _, room := range h.rooms {
		if room.driverID == driverID && room.status != "completed" {
			room.lastLat = lat
			room.lastLon = lon
			room.updatedAt = time.Now()
			rooms = append(rooms, room)
		}
	}
	h.mu.Unlock()

	for _, room := range rooms {
		h.server.BroadcastToRoom("/", roomSocket(room.id), "trip:location", tripStatusPayload{
			TripID:     room.id,
			Status:     room.status,
			DriverID:   room.driverID,
			RiderID:    room.riderID,
			Pickup:     room.pickup,
			Station:    room.station,
			LastLat:    lat,
			LastLon:    lon,
			RecordedAt: time.Now(),
		})
	}
}

func (h *RealtimeHub) PickupArrived(driverID string, pickupID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, room := range h.rooms {
		if room.driverID != driverID || room.pickup == nil || room.pickup.ID != pickupID {
			continue
		}
		if room.status == "awaiting_pickup" {
			room.status = "in_progress"
			h.server.BroadcastToRoom("/", roomSocket(room.id), "trip:status", tripStatusPayload{
				TripID:     room.id,
				Status:     room.status,
				DriverID:   room.driverID,
				RiderID:    room.riderID,
				Pickup:     room.pickup,
				Station:    room.station,
				RecordedAt: time.Now(),
			})
		}
	}
}

func (h *RealtimeHub) completeTripRoom(tripID, reason string) {
	h.mu.Lock()
	room, ok := h.rooms[tripID]
	if ok {
		room.status = "completed"
	}
	h.mu.Unlock()

	if ok {
		h.server.BroadcastToRoom("/", roomSocket(tripID), "trip:status", tripStatusPayload{
			TripID:      tripID,
			Status:      "completed",
			DriverID:    room.driverID,
			RiderID:     room.riderID,
			RecordedAt:  time.Now(),
			Description: reason,
		})
	}
	if h.gateway != nil {
		if _, err := h.gateway.completeTrip(tripID); err != nil {
			h.logger.Warn("completeTripRoom failed", "tripId", tripID, "err", err)
		}
	}
}

func (h *RealtimeHub) notifyRiderStatus(riderID string, payload tripStatusPayload) {
	session := h.riderSession(riderID)
	if session == nil {
		return
	}
	session.conn.Emit("rider:status", payload)
}

func (h *RealtimeHub) emitToDriver(driverID, event string, payload any) {
	session := h.driverSession(driverID)
	if session == nil {
		return
	}
	session.conn.Emit(event, payload)
}

func (h *RealtimeHub) driverSession(driverID string) *driverSession {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.drivers[driverID]
}

func (h *RealtimeHub) riderSession(riderID string) *riderSession {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.riders[riderID]
}

func (h *RealtimeHub) deliverPendingApprovals(riderID string) {
	h.mu.Lock()
	session := h.riders[riderID]
	if session == nil {
		h.mu.Unlock()
		return
	}
	pending := make([]*pendingApproval, 0)
	for _, ctx := range h.pendingApprovals {
		if ctx.rider != nil && ctx.rider.ID == riderID {
			pending = append(pending, ctx)
		}
	}
	h.mu.Unlock()
	for _, ctx := range pending {
		session.conn.Emit("rider:approval-request", h.approvalPayload(ctx))
	}
}

func (h *RealtimeHub) joinRoom(driverID, tripID string) {
	h.mu.Lock()
	session := h.drivers[driverID]
	h.mu.Unlock()
	if session != nil {
		session.conn.Join(roomSocket(tripID))
	}
}

func (h *RealtimeHub) joinRiderRoom(riderID, tripID string) {
	h.mu.Lock()
	session := h.riders[riderID]
	h.mu.Unlock()
	if session != nil {
		session.conn.Join(roomSocket(tripID))
	}
}

func (h *RealtimeHub) RefreshDriverQueue(driverID string) {
	if h.gateway == nil {
		return
	}
	resp, err := h.gateway.driverRequests(driverID)
	if err != nil {
		h.logger.Warn("push driver queue failed", "driverId", driverID, "err", err)
		return
	}
	h.emitToDriver(driverID, "driver:rider-queue", resp)
}

func (h *RealtimeHub) CompleteTrip(tripID, reason string) {
	h.completeTripRoom(tripID, reason)
}

func (h *RealtimeHub) RequestRiderApproval(trip Trip, pickup *PickupPoint, station *Station, rider *Rider) {
	if rider == nil {
		return
	}
	h.mu.Lock()
	session := h.riders[rider.ID]
	ctx := &pendingApproval{trip: trip, pickup: pickup, station: station, rider: rider}
	ctx.timer = time.AfterFunc(25*time.Second, func() {
		h.handleApprovalTimeout(trip.ID)
	})
	h.pendingApprovals[trip.ID] = ctx
	h.mu.Unlock()

	if session != nil {
		session.conn.Emit("rider:approval-request", h.approvalPayload(ctx))
	}
}

func (h *RealtimeHub) handleRiderApproval(tripID, riderID string, accept bool, reason string) {
	ctx := h.popApproval(tripID)
	if ctx == nil {
		return
	}
	if accept {
		if err := h.gateway.FinalizeTrip(tripID); err != nil {
			h.logger.Warn("finalize trip via rider approval failed", "tripId", tripID, "err", err)
		}
	} else {
		if reason == "" {
			reason = "rider_declined"
		}
		if err := h.gateway.CancelTrip(tripID, reason); err != nil {
			h.logger.Warn("cancel trip via rider response failed", "tripId", tripID, "err", err)
		}
	}
}

func (h *RealtimeHub) handleApprovalTimeout(tripID string) {
	ctx := h.popApproval(tripID)
	if ctx == nil {
		return
	}
	if err := h.gateway.CancelTrip(tripID, "rider_timeout"); err != nil {
		h.logger.Warn("cancel trip timeout failed", "tripId", tripID, "err", err)
	}
}

func (h *RealtimeHub) popApproval(tripID string) *pendingApproval {
	h.mu.Lock()
	ctx, ok := h.pendingApprovals[tripID]
	if ok {
		delete(h.pendingApprovals, tripID)
	}
	h.mu.Unlock()
	if ok && ctx.timer != nil {
		ctx.timer.Stop()
	}
	return ctx
}

func (h *RealtimeHub) NotifyDriverTripCancelled(driverID, tripID, reason string) {
	h.emitToDriver(driverID, "driver:trip-cancelled", map[string]string{
		"tripId": tripID,
		"reason": reason,
	})
}

func (h *RealtimeHub) ClearApproval(tripID string) {
	h.popApproval(tripID)
}

func (h *RealtimeHub) CreateRoomForTrip(trip Trip, pickup *PickupPoint, station *Station, rider *Rider) {
	if h == nil {
		return
	}
	room := &tripRoom{
		id:         trip.ID,
		driverID:   trip.DriverID,
		driverName: "",
		riderID:    trip.RiderID,
		riderName:  riderName(rider),
		pickup:     pickup,
		station:    station,
		status:     trip.Status,
		updatedAt:  time.Now(),
	}

	payload := tripStatusPayload{
		TripID:     trip.ID,
		Status:     trip.Status,
		DriverID:   trip.DriverID,
		RiderID:    trip.RiderID,
		Pickup:     pickup,
		Station:    station,
		RecordedAt: time.Now(),
		Trip:       &trip,
		Rider:      rider,
	}

	h.mu.Lock()
	h.rooms[trip.ID] = room
	h.mu.Unlock()

	h.joinRoom(trip.DriverID, trip.ID)
	if rider != nil {
		h.joinRiderRoom(rider.ID, trip.ID)
		h.notifyRiderStatus(rider.ID, payload)
	}
	h.emitToDriver(trip.DriverID, "trip:room-created", payload)
}

func riderName(rider *Rider) string {
	if rider == nil {
		return ""
	}
	return rider.Name
}

func roomSocket(tripID string) string {
	return fmt.Sprintf("trip:%s", tripID)
}

func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "driver":
		return "driver"
	case "rider":
		return "rider"
	default:
		return ""
	}
}

func (h *RealtimeHub) driverName(driverID string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if session, ok := h.drivers[driverID]; ok {
		return session.name
	}
	return ""
}

func (h *RealtimeHub) approvalPayload(ctx *pendingApproval) map[string]any {
	if ctx == nil {
		return nil
	}
	return map[string]any{
		"tripId":     ctx.trip.ID,
		"driverId":   ctx.trip.DriverID,
		"driverName": h.driverName(ctx.trip.DriverID),
		"pickup":     ctx.pickup,
		"station":    ctx.station,
	}
}
