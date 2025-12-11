import { useState, useEffect, useRef, useMemo } from 'react';
import type { FormEvent } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { MapPin, Navigation, Users, LogOut, Power, Search, PlayCircle } from 'lucide-react';
import { MapContainer, TileLayer, Marker, Popup, Polyline, useMap } from 'react-leaflet';
import 'leaflet/dist/leaflet.css';
import L from 'leaflet';
import { acceptDriverRequest, bookRide, fetchDriverRequests, fetchPickupPoints, fetchSnapshot, saveDriverRoute, startDriverTrip } from '../lib/backend';
import type { BookRideResponse, Driver, DriverRequestsResponse, PickupPoint, TripStatusPayload } from '../lib/types';
import { DESTINATIONS } from '../lib/destinations';
import { DriversMap } from './DriversMap';
import { DriverCard } from './DriverCard';
import { Room } from './Room';
import { useRealtime } from '../contexts/RealtimeContext';

// Fix Leaflet default icon issue
import icon from 'leaflet/dist/images/marker-icon.png';
import iconShadow from 'leaflet/dist/images/marker-shadow.png';

let DefaultIcon = L.icon({
    iconUrl: icon,
    shadowUrl: iconShadow,
    iconSize: [25, 41],
    iconAnchor: [12, 41],
});

L.Marker.prototype.options.icon = DefaultIcon;

// Component to center map on location update
function MapUpdater({ center }: { center: [number, number] }) {
    const map = useMap();
    useEffect(() => {
        map.setView(center, map.getZoom());
    }, [center, map]);
    return null;
}

const formatDistance = (meters?: number) => {
    if (!meters || !Number.isFinite(meters) || meters === Number.POSITIVE_INFINITY) {
        return '—';
    }
    if (meters >= 1000) {
        return `${(meters / 1000).toFixed(1)} km`;
    }
    return `${Math.round(meters)} m`;
};

export function Dashboard() {
    const { user, signOut, role: authRole } = useAuth();
    const { ready: realtimeReady, socket, offers, queueSummary, rooms, riderStatus, approvalRequest, respondToOffer, completeTrip, respondToApproval } = useRealtime();
    const [activeRoom, setActiveRoom] = useState<{ id: string; status: TripStatusPayload } | null>(null);

    useEffect(() => {
        if (!socket) return;
        const onRoomCreated = (payload: TripStatusPayload) => {
            setActiveRoom({ id: payload.tripId, status: payload });
        };
        socket.on('trip:room-created', onRoomCreated);

        // Also check if we should already be in a room from riderStatus
        if (riderStatus?.tripId && (riderStatus.status === 'awaiting_pickup' || riderStatus.status === 'in_progress')) {
            if (!activeRoom || activeRoom.id !== riderStatus.tripId) {
                // Cast or ensure compatibility. Assuming RiderStatus is compatible enough or we need to map it.
                // Ideally we should use a proper converter, but for now we trust the shape overlap for key fields.
                // We need to ensure 'status' field matches TripStatusPayload type if strict.
                setActiveRoom({ id: riderStatus.tripId, status: riderStatus as any });
            }
        }

        return () => {
            socket.off('trip:room-created', onRoomCreated);
        };
    }, [socket, riderStatus]);

    const role = user?.user_metadata?.role || authRole || 'rider';
    const isDriver = typeof role === 'string' ? role.toLowerCase() === 'driver' : role === 2;
    const driverTabs = [
        { key: 'dashboard', label: 'Dashboard' },
        { key: 'riders', label: 'Riders' },
        { key: 'trips', label: 'Trips' },
        { key: 'profile', label: 'Profile' },
    ] as const;
    const riderTabs = [
        { key: 'dashboard', label: 'Dashboard' },
        { key: 'drivers', label: 'Drivers' },
        { key: 'book', label: 'Book' },
        { key: 'profile', label: 'Profile' },
    ] as const;
    const [activeTab, setActiveTab] = useState<string>('dashboard');

    useEffect(() => {
        setActiveTab('dashboard');
    }, [isDriver]);

    const tabs = isDriver ? driverTabs : riderTabs;

    const [isOnline, setIsOnline] = useState(false);
    const [status, setStatus] = useState('offline');
    const [location, setLocation] = useState<{ lat: number; lng: number } | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [drivers, setDrivers] = useState<Driver[]>([]);
    const [snapshotLoading, setSnapshotLoading] = useState(true);
    const [snapshotError, setSnapshotError] = useState<string | null>(null);
    const [driverRequests, setDriverRequests] = useState<DriverRequestsResponse | null>(null);
    const [driverRequestError, setDriverRequestError] = useState<string | null>(null);
    const [pickupPoints, setPickupPoints] = useState<PickupPoint[]>([]);
    const [pickupQuery, setPickupQuery] = useState('');
    const [selectedPickupId, setSelectedPickupId] = useState<string | null>(null);
    const [routeQuery, setRouteQuery] = useState('');
    const [bookingState, setBookingState] = useState<BookRideResponse | null>(null);
    const [bookingError, setBookingError] = useState<string | null>(null);
    const [bookingLoading, setBookingLoading] = useState(false);
    const [routePickupIds, setRoutePickupIds] = useState<string[]>([]);
    const [routeSeatCount, setRouteSeatCount] = useState('3');
    const [routeStatus, setRouteStatus] = useState<string | null>(null);
    const [tripStatus, setTripStatus] = useState<string | null>(null);
    const [acceptingRiderId, setAcceptingRiderId] = useState<string | null>(null);
    const [acceptFeedback, setAcceptFeedback] = useState<string | null>(null);
    const [savingRoute, setSavingRoute] = useState(false);
    const [startingTrip, setStartingTrip] = useState(false);
    const [simulating, setSimulating] = useState(false);
    const [selectedDestination, setSelectedDestination] = useState<string | null>(null);
    const watchIdRef = useRef<number | null>(null);
    const selectedPickup = useMemo<PickupPoint | null>(() => pickupPoints.find((point) => point.id === selectedPickupId) ?? null, [pickupPoints, selectedPickupId]);
    const filteredPickupOptions = useMemo<PickupPoint[]>(() => {
        if (!pickupQuery.trim()) {
            return pickupPoints.slice(0, 8);
        }
        const query = pickupQuery.toLowerCase();
        return pickupPoints
            .filter((point) => point.name.toLowerCase().includes(query) || point.stationName.toLowerCase().includes(query))
            .slice(0, 8);
    }, [pickupPoints, pickupQuery]);
    const filteredRouteOptions = useMemo<PickupPoint[]>(() => {
        if (!routeQuery.trim()) {
            return pickupPoints.slice(0, 12);
        }
        const query = routeQuery.toLowerCase();
        return pickupPoints
            .filter((point) => point.name.toLowerCase().includes(query) || point.stationName.toLowerCase().includes(query))
            .slice(0, 12);
    }, [pickupPoints, routeQuery]);
    const selectedRoutePoints = useMemo<PickupPoint[]>(
        () => routePickupIds.map((id) => pickupPoints.find((point) => point.id === id)).filter(Boolean) as PickupPoint[],
        [pickupPoints, routePickupIds],
    );

    // Mock route data for now
    const selfDriver = isDriver ? drivers.find((driver) => driver.id === user?.id) : null;
    const plannedDestination = selfDriver?.route.destination ?? (selectedRoutePoints[selectedRoutePoints.length - 1]?.stationName ?? '—');
    const plannedSeats = selfDriver?.seatsAvailable ?? (Number.parseInt(routeSeatCount || '0', 10) || 0);

    const toggleOnline = async () => {
        if (isOnline) {
            // Go Offline
            setIsOnline(false);
            setStatus('offline');
            if (watchIdRef.current !== null) {
                navigator.geolocation.clearWatch(watchIdRef.current);
                watchIdRef.current = null;
            }
        } else {
            // Go Online
            setIsOnline(true);
            setStatus('online');
            startLocationTracking();
        }
    };

    const startLocationTracking = () => {
        if (!navigator.geolocation) {
            setError('Geolocation is not supported by your browser');
            return;
        }

        watchIdRef.current = navigator.geolocation.watchPosition(
            async (position) => {
                const { latitude, longitude } = position.coords;
                setLocation({ lat: latitude, lng: longitude });

                // Send update to backend if driver
                if (isDriver) {
                    try {
                        await fetch('/api/location/update', {
                            method: 'POST',
                            headers: {
                                'Content-Type': 'application/json',
                            },
                            body: JSON.stringify({
                                driverId: user?.id,
                                latitude,
                                longitude,
                            }),
                        });
                    } catch (err) {
                        console.error('Failed to send location update:', err);
                    }
                }
            },
            (err) => {
                setError(err.message);
            },
            {
                enableHighAccuracy: true,
                timeout: 5000,
                maximumAge: 0,
            }
        );
    };

    useEffect(() => {
        // Start tracking for riders immediately to show them on map
        // For drivers, only track if online and NOT simulating
        if (!isDriver) {
            startLocationTracking();
        } else if (isOnline && !simulating) {
            startLocationTracking();
        } else if (simulating && watchIdRef.current) {
            // Stop real GPS if simulating
            navigator.geolocation.clearWatch(watchIdRef.current);
            watchIdRef.current = null;
        }

        return () => {
            if (watchIdRef.current !== null) {
                navigator.geolocation.clearWatch(watchIdRef.current);
            }
        };
    }, [isDriver, isOnline, simulating]);

    // Simulation Loop
    useEffect(() => {
        if (!simulating || !activeRoom?.status?.station) return;
        const station = activeRoom.status.station;
        if (!station.latitude || !station.longitude) return;

        const interval = setInterval(async () => {
            setLocation((prev) => {
                if (!prev) return { lat: station.latitude! - 0.01, lng: station.longitude! - 0.01 }; // Start offset
                const latDiff = station.latitude! - prev.lat;
                const lngDiff = station.longitude! - prev.lng;

                // Move 10% of distance or min step
                const step = 0.1;
                // Simple interpolation
                const newLat = prev.lat + latDiff * step;
                const newLng = prev.lng + lngDiff * step;

                // Send update
                // Send update to backend if driver
                if (isDriver && user?.id) {
                    fetch('/api/location/update', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                            driverId: user.id,
                            latitude: newLat,
                            longitude: newLng,
                        }),
                    }).catch(console.warn);
                }
                return { lat: newLat, lng: newLng };
            });
        }, 5000);

        return () => clearInterval(interval);
    }, [simulating, activeRoom, isDriver, user?.id]);

    useEffect(() => {
        let cancelled = false;
        const loadSnapshot = async () => {
            try {
                const data = await fetchSnapshot();
                if (!cancelled) {
                    setDrivers(data.drivers);
                    setSnapshotError(null);
                }
            } catch (err) {
                if (!cancelled) {
                    setSnapshotError((err as Error).message || 'Failed to load snapshot');
                }
            } finally {
                if (!cancelled) {
                    setSnapshotLoading(false);
                }
            }
        };

        loadSnapshot();
        const interval = setInterval(loadSnapshot, 15000);
        return () => {
            cancelled = true;
            clearInterval(interval);
        };
    }, []);

    useEffect(() => {
        let cancelled = false;
        const loadPickups = async () => {
            try {
                const data = await fetchPickupPoints();
                if (!cancelled) {
                    setPickupPoints(data);
                }
            } catch (err) {
                console.warn('pickup fetch failed', err);
            }
        };
        loadPickups();
        return () => {
            cancelled = true;
        };
    }, []);

    useEffect(() => {
        if (!isDriver || !user?.id) {
            setDriverRequests(null);
            return;
        }
        if (realtimeReady) {
            return;
        }
        let cancelled = false;
        const loadRequests = async () => {
            try {
                const data = await fetchDriverRequests(user.id);
                if (!cancelled) {
                    setDriverRequests(data);
                    setDriverRequestError(null);
                }
            } catch (err) {
                if (!cancelled) {
                    setDriverRequestError((err as Error).message || 'Failed to load requests');
                }
            }
        };
        loadRequests();
        const interval = setInterval(loadRequests, 15000);
        return () => {
            cancelled = true;
            clearInterval(interval);
        };
    }, [isDriver, user?.id, realtimeReady]);

    useEffect(() => {
        if (isDriver && queueSummary) {
            setDriverRequests(queueSummary);
            setDriverRequestError(null);
        }
    }, [isDriver, queueSummary]);

    const toggleRoutePickup = (id: string) => {
        setRoutePickupIds((current) => (current.includes(id) ? current.filter((value) => value !== id) : [...current, id]));
    };

    const handleBookRide = async (event: FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        if (!selectedPickup) {
            setBookingError('Choose a pickup point to book a cab.');
            return;
        }
        setBookingError(null);
        setBookingLoading(true);
        const riderName =
            (user?.user_metadata?.full_name as string | undefined) ??
            (user?.user_metadata?.name as string | undefined) ??
            user?.email ??
            undefined;
        try {
            const response = await bookRide({
                command: 'book',
                address: selectedPickup.name, // Sending Station Name as Address (Pickup)
                destination: selectedDestination || 'Nearby Location', // Sending Selected Area
                pickupPointId: selectedPickup.id, // Station ID as PickupPointID
                riderId: user?.id ?? undefined,
                name: riderName,
            });
            setBookingState(response);
            setPickupQuery('');
            setSelectedPickupId(null);
        } catch (err) {
            setBookingError((err as Error).message || 'Unable to book right now');
        } finally {
            setBookingLoading(false);
        }
    };

    const handleSaveRoute = async () => {
        if (!user?.id) {
            setRouteStatus('Sign in as a driver to save your route.');
            return;
        }
        if (!routePickupIds.length) {
            setRouteStatus('Select at least one pickup cluster.');
            return;
        }
        setSavingRoute(true);
        try {
            const response = await saveDriverRoute({
                driverId: user.id,
                name: selfDriver?.name ?? user.email ?? 'Driver',
                carDetails: selfDriver?.carDetails ?? 'Metro Cab',
                pickupPointIds: routePickupIds,
                seats: Number.parseInt(routeSeatCount, 10) || 1,
                destination: selectedDestination || 'unknown', // Send 'unknown' to match any rider destination
            });
            setRouteStatus(`Route saved · ${response.pickupPoints.length} stops, ${response.seatsTotal} seats`);
        } catch (err) {
            setRouteStatus((err as Error).message || 'Unable to save route');
        } finally {
            setSavingRoute(false);
        }
    };

    const handleStartTripPlan = async (simulate?: boolean) => {
        if (!user?.id) {
            setTripStatus('Sign in to start a trip.');
            return;
        }
        setStartingTrip(true);
        try {
            await startDriverTrip(user.id, simulate);
            setTripStatus(simulate ? 'Simulated trip started. Watch your map animation.' : 'Trip started. Keep updating your location.');
        } catch (err) {
            setTripStatus((err as Error).message || 'Unable to start trip');
        } finally {
            setStartingTrip(false);
        }
    };

    const handleAcceptRider = async (riderId: string, simulateMode = false) => {
        if (!user?.id) {
            setAcceptFeedback('Sign in as a driver to accept riders.');
            return;
        }
        setAcceptFeedback(null);
        setAcceptingRiderId(riderId);
        if (simulateMode) {
            setSimulating(true);
            // Ensure we have a starting location for simulation if none exists
            if (!location) {
                setLocation({ lat: 12.83, lng: 77.65 }); // Default somewhere near EC
            }
        }
        try {
            const trip = await acceptDriverRequest({ driverId: user.id, riderId });
            const riderLabel = trip.pickup?.name ?? trip.destination ?? riderId;
            setAcceptFeedback(`Trip confirmed for ${riderLabel}. ${simulateMode ? 'Simulation started.' : 'Go pick them up!'}`);
            try {
                const [snapshotData, requestData] = await Promise.all([
                    fetchSnapshot(),
                    fetchDriverRequests(user.id),
                ]);
                setDrivers(snapshotData.drivers);
                setDriverRequests(requestData);
                setSnapshotError(null);
                setDriverRequestError(null);
            } catch (refreshErr) {
                console.warn('post-accept refresh failed', refreshErr);
            }
        } catch (err) {
            setAcceptFeedback((err as Error).message || 'Unable to accept rider');
            setSimulating(false); // Revert logic if fail
        } finally {
            setAcceptingRiderId(null);
        }
    };

    const driverRequestCenter: [number, number] = (() => {
        if (selfDriver?.latitude && selfDriver?.longitude) {
            return [selfDriver.latitude, selfDriver.longitude];
        }
        const first = driverRequests?.requests.find(
            (req) => typeof req.station.latitude === 'number' && typeof req.station.longitude === 'number',
        );
        if (first?.station.latitude && first.station.longitude) {
            return [first.station.latitude, first.station.longitude];
        }
        return [12.8456, 77.66];
    })();

    const driverNetworkCard = (
        <div className="bg-slate-800 rounded-xl p-6 border border-slate-700 space-y-4">
            <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-3">
                <div>
                    <h2 className="text-xl font-bold">Driver network</h2>
                    <p className="text-slate-400 text-sm">Live view of drivers reported by the gateway.</p>
                </div>
                <div className="text-xs uppercase tracking-wide text-slate-500">
                    {snapshotLoading ? 'Refreshing…' : 'Auto-refreshing every 15s'}
                </div>
            </div>
            {snapshotError ? (
                <div className="bg-red-500/10 border border-red-500/30 text-sm text-red-200 rounded-lg p-3">
                    {snapshotError}
                </div>
            ) : null}
            {drivers.length ? (
                <div className="grid lg:grid-cols-2 gap-6">
                    <DriversMap drivers={drivers} userLocation={location} />
                    <div className="space-y-4 max-h-[360px] overflow-y-auto pr-2">
                        {drivers.map((driver) => (
                            <DriverCard key={driver.id} driver={driver} />
                        ))}
                    </div>
                </div>
            ) : (
                <p className="text-slate-400 text-sm">No active drivers yet. Once the gateway reports drivers they’ll appear here.</p>
            )}
        </div>
    );

    const riderNetworkCard = (
        <div className="bg-slate-800 rounded-xl p-6 border border-slate-700 space-y-4">
            <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-3">
                <div>
                    <h2 className="text-xl font-bold">Rider network</h2>
                    <p className="text-slate-400 text-sm">View and accept riders queued along your metro drops.</p>
                </div>
                <div className="text-xs text-slate-500">
                    {driverRequests?.generatedAt
                        ? `Updated ${new Date(driverRequests.generatedAt).toLocaleTimeString()}`
                        : 'Polling every 15s'}
                </div>
            </div>
            {driverRequestError ? (
                <div className="bg-red-500/10 border border-red-500/30 text-sm text-red-200 rounded-lg p-3">
                    {driverRequestError}
                </div>
            ) : null}
            {acceptFeedback ? (
                <div className="bg-emerald-500/10 border border-emerald-500/30 text-sm text-emerald-200 rounded-lg p-3">
                    {acceptFeedback}
                </div>
            ) : null}
            {driverRequests && driverRequests.requests.length ? (
                <div className="grid lg:grid-cols-2 gap-6">
                    <div className="h-64 rounded-lg overflow-hidden border border-slate-700">
                        <MapContainer center={driverRequestCenter} zoom={14} style={{ height: '100%', width: '100%' }}>
                            <TileLayer
                                url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                                attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
                            />
                            {selfDriver?.latitude && selfDriver.longitude ? (
                                <Marker position={[selfDriver.latitude, selfDriver.longitude]}>
                                    <Popup>You</Popup>
                                </Marker>
                            ) : null}
                            {driverRequests.requests.map((req) => {
                                const pickup = req.pickup;
                                if (pickup?.latitude && pickup.longitude) {
                                    return (
                                        <Marker key={req.id} position={[pickup.latitude, pickup.longitude]}>
                                            <Popup>
                                                <div>
                                                    <p className="font-semibold">{req.name}</p>
                                                    <p className="text-xs text-slate-500">{pickup.name}</p>
                                                </div>
                                            </Popup>
                                        </Marker>
                                    );
                                }
                                if (req.station.latitude && req.station.longitude) {
                                    return (
                                        <Marker key={req.id} position={[req.station.latitude, req.station.longitude]}>
                                            <Popup>
                                                <div>
                                                    <p className="font-semibold">{req.name}</p>
                                                    <p className="text-xs text-slate-500">{req.station.name}</p>
                                                </div>
                                            </Popup>
                                        </Marker>
                                    );
                                }
                                return null;
                            })}
                            <MapUpdater center={driverRequestCenter} />
                        </MapContainer>
                    </div>
                    <div className="space-y-3 max-h-64 overflow-y-auto pr-2">
                        {driverRequests.requests.map((req) => (
                            <div key={req.id} className="bg-slate-900/40 border border-slate-700 rounded-lg p-4">
                                <div className="flex justify-between items-center mb-1">
                                    <div>
                                        <p className="font-semibold">{req.name}</p>
                                        <p className="text-slate-400 text-sm">Pickup · {req.pickup?.name ?? req.station.name}</p>
                                    </div>
                                    <span className={`text-xs uppercase ${req.status === 'matched' ? 'text-emerald-300' : 'text-amber-300'}`}>
                                        {req.status}
                                    </span>
                                </div>
                                <p className="text-slate-400 text-sm">Station · {req.station.name}</p>
                                <p className="text-slate-500 text-xs">Distance · {formatDistance(req.distanceMeters)}</p>
                                <div className="flex gap-2">
                                    <button
                                        onClick={() => handleAcceptRider(req.id, false)}
                                        disabled={acceptingRiderId === req.id}
                                        className={`mt-3 flex-1 rounded-lg py-2 text-sm font-semibold transition ${acceptingRiderId === req.id
                                            ? 'bg-slate-700 text-slate-400 cursor-not-allowed'
                                            : 'bg-emerald-500 text-white hover:bg-emerald-600'
                                            }`}
                                    >
                                        {acceptingRiderId === req.id ? 'Confirming…' : 'Accept'}
                                    </button>
                                    <button
                                        onClick={() => handleAcceptRider(req.id, true)}
                                        disabled={acceptingRiderId === req.id}
                                        className={`mt-3 flex-none px-4 rounded-lg py-2 text-sm font-semibold transition ${acceptingRiderId === req.id
                                            ? 'bg-slate-700 text-slate-400 cursor-not-allowed'
                                            : 'bg-blue-600 text-white hover:bg-blue-700'
                                            }`}
                                        title="Simulate Trip"
                                    >
                                        <PlayCircle size={18} />
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            ) : (
                <p className="text-slate-400 text-sm">Waiting for riders near your stations. Hang tight.</p>
            )}
            <div className="border-t border-slate-700 pt-4 mt-4">
                <div className="flex items-center justify-between mb-3">
                    <h3 className="text-lg font-semibold">Live socket offers</h3>
                    <span className={`text-xs uppercase ${realtimeReady ? 'text-emerald-300' : 'text-slate-500'}`}>
                        {realtimeReady ? 'connected' : 'connecting…'}
                    </span>
                </div>
                {offers.length ? (
                    <div className="grid md:grid-cols-2 gap-3">
                        {offers.map((offer) => (
                            <div key={offer.riderId} className="bg-slate-900/50 border border-slate-700 rounded-lg p-4 space-y-2">
                                <div className="flex justify-between items-center">
                                    <div>
                                        <p className="font-semibold">{offer.riderName}</p>
                                        <p className="text-xs text-slate-400">
                                            Pickup · {offer.pickupName ?? offer.station?.name ?? 'pending'}
                                        </p>
                                    </div>
                                    <span className="text-xs text-slate-500">
                                        {offer.attempt}/{offer.total}
                                    </span>
                                </div>
                                <p className="text-slate-400 text-sm">Destination · {offer.destination ?? '—'}</p>
                                {offer.station?.name ? (
                                    <p className="text-slate-500 text-xs">Station · {offer.station.name}</p>
                                ) : null}
                                <div className="flex gap-2">
                                    <button
                                        onClick={() => respondToOffer(offer.riderId, true)}
                                        className="flex-1 rounded-lg bg-emerald-500/90 hover:bg-emerald-600 text-sm font-semibold py-1.5"
                                    >
                                        Accept
                                    </button>
                                    <button
                                        onClick={() => respondToOffer(offer.riderId, false)}
                                        className="flex-1 rounded-lg bg-slate-900 border border-slate-700 text-sm font-semibold py-1.5 hover:border-slate-500"
                                    >
                                        Pass
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>
                ) : (
                    <p className="text-slate-500 text-sm">No socket offers right now. Riders will appear here in real time.</p>
                )}
            </div>
        </div>
    );

    const riderBookingCard = (
        <div className="bg-slate-800 rounded-xl p-6 border border-slate-700 space-y-4">
            <div>
                <h2 className="text-xl font-bold">Book a ride from Metro</h2>
                <p className="text-slate-400 text-sm">Select your Metro Station and destination.</p>
            </div>
            <form onSubmit={handleBookRide} className="space-y-4">
                <input
                    value={pickupQuery}
                    onChange={(e) => setPickupQuery(e.target.value)}
                    className="w-full bg-slate-900 border border-slate-700 rounded-lg py-2.5 px-3 text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="Search Metro Station"
                />
                <div className="grid grid-cols-2 gap-3">
                    {filteredPickupOptions.map((point: PickupPoint) => (
                        <button
                            key={point.id}
                            type="button"
                            onClick={() => {
                                setSelectedPickupId(point.id);
                                setSelectedDestination(null); // Reset destination when station changes
                            }}
                            className={`rounded-lg border px-3 py-2 text-left transition ${selectedPickupId === point.id
                                ? 'border-emerald-400 bg-emerald-500/10 text-white'
                                : 'border-slate-700 text-slate-300 hover:border-slate-500'
                                }`}
                        >
                            <div className="font-semibold">{point.name}</div>
                        </button>
                    ))}
                </div>
                {selectedPickup ? (
                    <div>
                        <label className="text-xs text-slate-400 uppercase font-semibold mb-2 block">Select Destination</label>
                        {/* Find the Station object matching selectedPickup.stationId to get nearbyAreas */}
                        <div className="grid grid-cols-2 lg:grid-cols-3 gap-2">
                            {DESTINATIONS.filter(d => d.stationId === selectedPickup.stationId).map((dest) => (
                                <button
                                    key={dest.id}
                                    type="button"
                                    onClick={() => setSelectedDestination(dest.name)}
                                    className={`text-xs px-2 py-1.5 rounded border transition ${selectedDestination === dest.name
                                        ? 'bg-blue-500/20 border-blue-500 text-white'
                                        : 'border-slate-700 text-slate-400 hover:border-slate-500'}`}
                                >
                                    {dest.name}
                                </button>
                            ))}
                        </div>
                    </div>
                ) : (
                    <div className="text-slate-500 text-sm">Tap a station above to see destinations.</div>
                )}
                <div className="h-60 rounded-xl overflow-hidden border border-slate-700">
                    <MapContainer
                        center={selectedPickup ? [selectedPickup.latitude, selectedPickup.longitude] : [location?.lat ?? 12.8456, location?.lng ?? 77.66]}
                        zoom={selectedPickup ? 15 : 13}
                        style={{ height: '100%', width: '100%' }}
                    >
                        <TileLayer url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png" attribution="&copy; OpenStreetMap" />
                        {selectedPickup ? (
                            <Marker position={[selectedPickup.latitude, selectedPickup.longitude]}>
                                <Popup>
                                    <div>
                                        <p className="font-semibold">{selectedPickup.name}</p>
                                        <p className="text-xs text-slate-500">Nearest station {selectedPickup.stationName}</p>
                                    </div>
                                </Popup>
                            </Marker>
                        ) : null}
                        {drivers
                            .filter((driver) => typeof driver.latitude === 'number' && typeof driver.longitude === 'number')
                            .map((driver) => (
                                <Marker key={driver.id} position={[driver.latitude!, driver.longitude!]}>
                                    <Popup>{driver.name}</Popup>
                                </Marker>
                            ))}
                    </MapContainer>
                </div>
                {bookingError ? <div className="text-sm text-red-300">{bookingError}</div> : null}
                <button
                    type="submit"
                    disabled={bookingLoading}
                    className="w-full bg-blue-600 hover:bg-blue-700 transition-colors rounded-lg py-2.5 font-semibold disabled:opacity-50 disabled:cursor-not-allowed"
                >
                    {bookingLoading ? 'Booking…' : 'Book Ride'}
                </button>
            </form>
            {bookingState && (
                <div className="bg-slate-900/40 border border-slate-700 rounded-lg p-4 space-y-2">
                    <div className="font-semibold">{bookingState.message}</div>
                    <div className="text-xs text-slate-400">Destination · {bookingState.requestedDestination}</div>
                    <div className="space-y-1">
                        {bookingState.attempts.map((attempt) => (
                            <div key={attempt.driverId} className="flex justify-between text-sm">
                                <span className={attempt.accepted ? 'text-emerald-300 font-semibold' : 'text-slate-300'}>
                                    {attempt.driverName}
                                </span>
                                <span className="text-slate-400">
                                    {attempt.accepted ? 'Accepted' : attempt.reason || 'Pending'}
                                </span>
                            </div>
                        ))}
                    </div>
                </div>
            )}
            {approvalRequest ? (
                <div className="bg-blue-500/10 border border-blue-500/40 rounded-lg p-4 space-y-3">
                    <div>
                        <p className="font-semibold text-white">{approvalRequest.driverName || 'Driver'} accepted your request</p>
                        <p className="text-sm text-slate-200">Pickup · {approvalRequest.pickup?.name ?? '—'}</p>
                        <p className="text-sm text-slate-200">Station · {approvalRequest.station?.name ?? '—'}</p>
                    </div>
                    <div className="grid grid-cols-2 gap-3">
                        <button
                            onClick={() => respondToApproval(approvalRequest.tripId, true)}
                            className="rounded-lg bg-emerald-500 hover:bg-emerald-600 text-sm font-semibold py-2"
                        >
                            Approve ride
                        </button>
                        <button
                            onClick={() => respondToApproval(approvalRequest.tripId, false)}
                            className="rounded-lg bg-slate-900 border border-slate-700 text-sm font-semibold py-2 hover:border-slate-500"
                        >
                            Decline
                        </button>
                    </div>
                </div>
            ) : null}
            {riderStatus ? (
                <div className="bg-slate-900/40 border border-blue-500/40 rounded-lg p-4 space-y-2">
                    <div className="flex items-center justify-between">
                        <span className="font-semibold text-white">Live status · {riderStatus.status}</span>
                        <span className="text-xs text-slate-400">{new Date(riderStatus.recordedAt).toLocaleTimeString()}</span>
                    </div>
                    <p className="text-slate-300 text-sm">Driver · {riderStatus.driverId ?? 'Assigning'}</p>
                    <p className="text-slate-300 text-sm">Pickup · {riderStatus.pickup?.name ?? '—'}</p>
                    {riderStatus.latitude && riderStatus.longitude ? (
                        <div className="h-48 rounded-lg overflow-hidden border border-slate-700">
                            <MapContainer center={[riderStatus.latitude, riderStatus.longitude]} zoom={15} style={{ height: '100%', width: '100%' }}>
                                <TileLayer url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png" attribution="&copy; OpenStreetMap" />
                                <Marker position={[riderStatus.latitude, riderStatus.longitude]}>
                                    <Popup>Driver</Popup>
                                </Marker>
                                {riderStatus.pickup?.latitude && riderStatus.pickup.longitude ? (
                                    <Marker position={[riderStatus.pickup.latitude, riderStatus.pickup.longitude]}>
                                        <Popup>{riderStatus.pickup.name}</Popup>
                                    </Marker>
                                ) : null}
                                {riderStatus.station?.latitude && riderStatus.station.longitude ? (
                                    <Marker position={[riderStatus.station.latitude, riderStatus.station.longitude]}>
                                        <Popup>{riderStatus.station.name}</Popup>
                                    </Marker>
                                ) : null}
                            </MapContainer>
                        </div>
                    ) : null}
                </div>
            ) : null}
        </div>
    );

    const driverRoutePlanner = (
        <div className="bg-slate-800 rounded-xl p-6 border border-slate-700 space-y-4">
            <div>
                <h2 className="text-xl font-bold">Plan your metro route</h2>
                <p className="text-slate-400 text-sm">Select the Stations you are covering.</p>
            </div>
            <div className="space-y-3">
                <input
                    value={routeQuery}
                    onChange={(e) => setRouteQuery(e.target.value)}
                    className="w-full bg-slate-900 border border-slate-700 rounded-lg py-2 px-3 text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                    placeholder="Search Station"
                />
                <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
                    {filteredRouteOptions.map((point: PickupPoint) => (
                        <button
                            key={point.id}
                            type="button"
                            onClick={() => toggleRoutePickup(point.id)}
                            className={`rounded-lg border px-3 py-2 text-left transition ${routePickupIds.includes(point.id)
                                ? 'border-emerald-400 bg-emerald-500/10 text-white'
                                : 'border-slate-700 text-slate-300 hover:border-slate-500'
                                }`}
                        >
                            <div className="font-semibold">{point.name}</div>
                            <div className="text-xs text-slate-400">{point.stationName}</div>
                        </button>
                    ))}
                </div>
                {selectedRoutePoints.length ? (
                    <div className="text-slate-300 text-sm">Stops · {selectedRoutePoints.map((point) => point.name).join(', ')}</div>
                ) : (
                    <div className="text-slate-500 text-sm">Tap the chips above to add them to your shuttle route.</div>
                )}
                <input
                    value={routeSeatCount}
                    onChange={(e) => setRouteSeatCount(e.target.value)}
                    className="w-full bg-slate-900 border border-slate-700 rounded-lg py-2 px-3 text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                    placeholder="Seats available"
                />
                {routeStatus ? <div className="text-sm text-slate-300">{routeStatus}</div> : null}
                <div className="flex flex-col md:flex-row gap-3">
                    <button
                        onClick={handleSaveRoute}
                        className="flex-1 bg-emerald-500 hover:bg-emerald-600 transition-colors rounded-lg py-2.5 font-semibold text-white"
                        disabled={savingRoute}
                    >
                        {savingRoute ? 'Saving…' : 'Save Route'}
                    </button>
                    <button
                        onClick={() => handleStartTripPlan(false)}
                        className="flex-1 bg-slate-900 border border-slate-700 hover:border-slate-500 transition-colors rounded-lg py-2.5 font-semibold"
                        disabled={startingTrip}
                    >
                        {startingTrip ? 'Starting…' : 'Start Trip'}
                    </button>
                    <button
                        onClick={() => handleStartTripPlan(true)}
                        className="flex-1 bg-slate-900 border border-emerald-500 text-emerald-400 hover:bg-emerald-500/10 transition-colors rounded-lg py-2.5 font-semibold"
                        disabled={startingTrip}
                    >
                        Simulated Trip
                    </button>
                </div>
                {tripStatus ? <div className="text-sm text-slate-300">{tripStatus}</div> : null}
                <div className="h-64 rounded-xl overflow-hidden border border-slate-700">
                    <MapContainer
                        center={selectedRoutePoints.length ? [selectedRoutePoints[0].latitude, selectedRoutePoints[0].longitude] : [selfDriver?.latitude ?? 12.8456, selfDriver?.longitude ?? 77.66]}
                        zoom={selectedRoutePoints.length ? 14 : 12}
                        style={{ height: '100%', width: '100%' }}
                    >
                        <TileLayer url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png" attribution="&copy; OpenStreetMap" />
                        {selfDriver?.latitude && selfDriver.longitude ? (
                            <Marker position={[selfDriver.latitude, selfDriver.longitude]}>
                                <Popup>You</Popup>
                            </Marker>
                        ) : null}
                        {selectedRoutePoints.map((point: PickupPoint) => (
                            <Marker key={point.id} position={[point.latitude, point.longitude]}>
                                <Popup>{point.name}</Popup>
                            </Marker>
                        ))}
                        {selectedRoutePoints.length >= 2 ? (
                            <Polyline positions={selectedRoutePoints.map((point) => [point.latitude, point.longitude])} color="#fbbf24" />
                        ) : null}
                    </MapContainer>
                </div>
            </div>
        </div>
    );

    const tripRoomsPanel = (
        <div className="bg-slate-800 rounded-xl p-6 border border-slate-700 space-y-4">
            <div className="flex items-center justify-between">
                <div>
                    <h2 className="text-xl font-bold">Active trip rooms</h2>
                    <p className="text-slate-400 text-sm">Drivers and riders linked in realtime.</p>
                </div>
                <span className="text-xs text-slate-500">{rooms.length ? `${rooms.length} active` : 'No active trips'}</span>
            </div>
            {rooms.length ? (
                <div className="grid md:grid-cols-2 gap-4">
                    {rooms.map((room) => {
                        const centerLat = room.latitude ?? room.pickup?.latitude ?? room.station?.latitude ?? 12.8456;
                        const centerLon = room.longitude ?? room.pickup?.longitude ?? room.station?.longitude ?? 77.66;
                        return (
                            <div key={room.tripId} className="bg-slate-900/40 border border-slate-700 rounded-lg p-4 space-y-3">
                                <div className="flex justify-between items-center">
                                    <div>
                                        <p className="font-semibold text-white">{room.trip?.riderId ?? room.riderId}</p>
                                        <p className="text-xs text-slate-400">Trip · {room.tripId.slice(0, 8)}</p>
                                    </div>
                                    <span className={`text-xs uppercase ${room.status === 'completed' ? 'text-emerald-300' : 'text-amber-300'}`}>
                                        {room.status}
                                    </span>
                                </div>
                                <p className="text-sm text-slate-300">Pickup · {room.pickup?.name ?? '—'}</p>
                                <p className="text-sm text-slate-300">Station · {room.station?.name ?? '—'}</p>
                                <div className="h-40 rounded-lg overflow-hidden border border-slate-700">
                                    <MapContainer center={[centerLat, centerLon]} zoom={14} style={{ height: '100%', width: '100%' }}>
                                        <TileLayer url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png" attribution="&copy; OpenStreetMap" />
                                        {room.latitude && room.longitude ? (
                                            <Marker position={[room.latitude, room.longitude]}>
                                                <Popup>Driver</Popup>
                                            </Marker>
                                        ) : null}
                                        {room.pickup?.latitude && room.pickup.longitude ? (
                                            <Marker position={[room.pickup.latitude, room.pickup.longitude]}>
                                                <Popup>{room.pickup.name}</Popup>
                                            </Marker>
                                        ) : null}
                                        {room.station?.latitude && room.station.longitude ? (
                                            <Marker position={[room.station.latitude, room.station.longitude]}>
                                                <Popup>{room.station.name}</Popup>
                                            </Marker>
                                        ) : null}
                                        <MapUpdater center={[centerLat, centerLon]} />
                                    </MapContainer>
                                </div>
                                <div className="flex gap-2">
                                    <button
                                        onClick={() => setActiveRoom({ id: room.tripId, status: { ...room, recordedAt: room.updatedAt || new Date().toISOString() } })}
                                        className="flex-1 rounded-lg bg-blue-600 text-white py-2 text-sm font-semibold hover:bg-blue-700 transition"
                                    >
                                        Resume
                                    </button>
                                    <button
                                        onClick={() => completeTrip(room.tripId)}
                                        className="w-full rounded-lg border border-slate-600 py-2 text-sm font-semibold hover:border-emerald-400 transition"
                                    >
                                        Mark Complete
                                    </button>
                                </div>
                            </div>
                        );
                    })}
                </div>
            ) : (
                <p className="text-slate-500 text-sm">No active socket rooms yet.</p>
            )}
        </div>
    );

    const driverStatusCard = (
        <div className="bg-slate-800 rounded-xl p-6 border border-slate-700">
            <div className="flex justify-between items-center mb-6">
                <div>
                    <h2 className="text-xl font-bold mb-1">Current Status</h2>
                    <div className="flex items-center gap-2">
                        <div className={`w-2.5 h-2.5 rounded-full ${isOnline ? 'bg-green-500' : 'bg-slate-500'}`} />
                        <span className="text-slate-400 capitalize">{status}</span>
                    </div>
                </div>
                <button
                    onClick={toggleOnline}
                    className={`flex items-center gap-2 px-6 py-3 rounded-xl font-semibold transition-all ${isOnline
                        ? 'bg-red-500/10 text-red-500 hover:bg-red-500/20'
                        : 'bg-green-600 text-white hover:bg-green-700 shadow-lg shadow-green-900/20'
                        }`}
                >
                    <Power className="w-5 h-5" />
                    {isOnline ? 'Go Offline' : 'Go Online'}
                </button>
            </div>

            {error && (
                <div className="bg-red-500/10 border border-red-500/50 rounded-lg p-4 mb-6 text-red-200 text-sm">
                    {error}
                </div>
            )}

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="bg-slate-900/50 p-4 rounded-lg border border-slate-700/50">
                    <div className="flex items-center gap-3 mb-2">
                        <MapPin className="w-5 h-5 text-blue-500" />
                        <span className="text-slate-400 text-sm">Destination</span>
                    </div>
                    <p className="font-semibold text-lg">{plannedDestination}</p>
                </div>

                <div className="bg-slate-900/50 p-4 rounded-lg border border-slate-700/50">
                    <div className="flex items-center gap-3 mb-2">
                        <Users className="w-5 h-5 text-purple-500" />
                        <span className="text-slate-400 text-sm">Available Seats</span>
                    </div>
                    <p className="font-semibold text-lg">{plannedSeats}</p>
                </div>

                <div className="bg-slate-900/50 p-4 rounded-lg border border-slate-700/50">
                    <div className="flex items-center gap-3 mb-2">
                        <Navigation className="w-5 h-5 text-emerald-500" />
                        <span className="text-slate-400 text-sm">Location</span>
                    </div>
                    <p className="font-mono text-sm text-slate-300">
                        {location
                            ? `${location.lat.toFixed(4)}, ${location.lng.toFixed(4)}`
                            : 'Waiting for signal...'}
                    </p>
                </div>
            </div>
        </div>
    );

    const profilePanel = (
        <div className="bg-slate-800 rounded-xl p-6 border border-slate-700 space-y-4">
            <div>
                <h2 className="text-xl font-bold">Profile</h2>
                <p className="text-slate-400 text-sm">Your account details and session actions.</p>
            </div>
            <div className="grid gap-4 md:grid-cols-2">
                <div className="bg-slate-900/60 border border-slate-700 rounded-lg p-4">
                    <p className="text-slate-400 text-xs uppercase">Email</p>
                    <p className="text-white font-semibold">{user?.email ?? '—'}</p>
                </div>
                <div className="bg-slate-900/60 border border-slate-700 rounded-lg p-4">
                    <p className="text-slate-400 text-xs uppercase">Role</p>
                    <p className="text-white font-semibold">{isDriver ? 'Driver' : 'Rider'}</p>
                </div>
            </div>
            <div className="flex flex-wrap gap-3">
                <button
                    onClick={toggleOnline}
                    className="px-4 py-2 rounded-lg border border-slate-600 text-slate-200 hover:border-slate-400 transition"
                >
                    {isOnline ? 'Go Offline' : 'Go Online'}
                </button>
                <button
                    onClick={() => signOut()}
                    className="px-4 py-2 rounded-lg bg-red-500/80 text-white font-semibold hover:bg-red-500 transition"
                >
                    Sign Out
                </button>
            </div>
        </div>
    );
    const renderDriverContent = () => {
        switch (activeTab) {
            case 'dashboard':
                return (
                    <>
                        {driverNetworkCard}
                        {driverStatusCard}
                    </>
                );
            case 'riders':
                return riderNetworkCard;
            case 'trips':
                return (
                    <>
                        {driverRoutePlanner}
                        <div className="mt-6">{tripRoomsPanel}</div>
                    </>
                );
            case 'profile':
                return profilePanel;
            default:
                return driverNetworkCard;
        }
    };

    const renderRiderContent = () => {
        switch (activeTab) {
            case 'drivers':
                return driverNetworkCard;
            case 'book':
                return riderBookingCard;
            case 'profile':
                return profilePanel;
            default:
                return (
                    <>
                        {driverNetworkCard}
                        {riderBookingCard}
                    </>
                );
        }
    };

    return (
        <div className="min-h-screen bg-slate-900 text-white">
            {/* Header */}
            <header className="bg-slate-800 border-b border-slate-700 p-4">
                <div className="max-w-7xl mx-auto flex justify-between items-center">
                    <div className="flex items-center gap-3">
                        <div className={`w-10 h-10 rounded-full flex items-center justify-center ${isDriver ? 'bg-blue-600' : 'bg-purple-600'}`}>
                            {isDriver ? <Navigation className="w-6 h-6 text-white" /> : <Search className="w-6 h-6 text-white" />}
                        </div>
                        <div>
                            <h1 className="font-bold text-lg">{isDriver ? 'Driver Console' : 'Rider App'}</h1>
                            <p className="text-xs text-slate-400">{user?.email}</p>
                            <p className="text-[11px] uppercase tracking-wide text-slate-500">Logged in as {isDriver ? 'Driver' : 'Rider'}</p>
                        </div>
                    </div>
                    <button
                        onClick={() => signOut()}
                        className="p-2 hover:bg-slate-700 rounded-lg transition-colors"
                    >
                        <LogOut className="w-5 h-5 text-slate-400" />
                    </button>
                </div>
            </header>

            <main className="max-w-7xl mx-auto p-4 space-y-6">
                <div className="flex flex-wrap gap-2">
                    {tabs.map((tab) => {
                        const active = activeTab === tab.key;
                        return (
                            <button
                                key={tab.key}
                                onClick={() => setActiveTab(tab.key)}
                                className={`px-4 py-2 rounded-lg text-sm font-semibold border transition ${active
                                    ? 'bg-blue-600 border-blue-400 text-white'
                                    : 'border-slate-700 text-slate-300 hover:border-slate-500'
                                    }`}
                            >
                                {tab.label}
                            </button>
                        );
                    })}
                </div>

                {isDriver ? renderDriverContent() : renderRiderContent()}

                {/* Map View */}
                <div className="bg-slate-800 rounded-xl p-1 border border-slate-700 h-[500px] overflow-hidden relative z-0">
                    {location ? (
                        <MapContainer center={[location.lat, location.lng]} zoom={15} style={{ height: '100%', width: '100%' }}>
                            <TileLayer
                                url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                                attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
                            />
                            <Marker position={[location.lat, location.lng]}>
                                <Popup>
                                    {isDriver ? 'You are here' : 'Your Location'}
                                </Popup>
                            </Marker>
                            <MapUpdater center={[location.lat, location.lng]} />
                        </MapContainer>
                    ) : (
                        <div className="h-full flex items-center justify-center text-slate-400">
                            {error ? error : 'Acquiring location...'}
                        </div>
                    )}
                </div>
            </main>
            {activeRoom && (
                <Room
                    tripId={activeRoom.id}
                    initialStatus={activeRoom.status}
                    onClose={() => setActiveRoom(null)}
                    seats={typeof plannedSeats === 'number' ? plannedSeats : undefined}
                />
            )}
        </div>
    );
}
