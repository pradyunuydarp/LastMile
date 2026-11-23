import { useState, useEffect, useRef, useMemo } from 'react';
import type { FormEvent, ReactNode } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { MapPin, Navigation, Users, LogOut, Power, Search, CirclePlay, CircleAlert, CircleCheck, Check, Radio, LocateFixed, CarFront, Inbox, Route, Clock } from 'lucide-react';
import { MapContainer, Marker, Popup } from 'react-leaflet';
import L from 'leaflet';
import { acceptDriverRequest, bookRide, fetchDriverRequests, fetchPickupPoints, fetchSnapshot, saveDriverRoute, startDriverTrip } from '../lib/backend';
import type { BookRideResponse, Driver, DriverRequestsResponse, PickupPoint, TripStatusPayload } from '../lib/types';
import { DESTINATIONS } from '../lib/destinations';
import { DriversMap } from './DriversMap';
import { DriverCard } from './DriverCard';
import { Room } from './Room';
import { useRealtime } from '../contexts/RealtimeContext';
import { BaseTiles, MapUpdater, RouteLine } from './MapTiles';
import { dotIcon, mapIcons, stopIcon } from './mapIcons';
import { SegmentedControl } from './SegmentedControl';

// Tidy circular marker instead of Leaflet's default blue pin.
L.Marker.prototype.options.icon = mapIcons.dot;

const stationDotIcon = dotIcon('sys-indigo', 16);

/* ---------------------------------------------------------------------------
   Small presentational building blocks (Apple-style inset grouped UI).
   --------------------------------------------------------------------------- */

function CardHeader({ title, subtitle, accessory }: { title: string; subtitle?: string; accessory?: ReactNode }) {
    return (
        <div className="mb-5 flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
            <div className="min-w-0">
                <h2 className="card-title">{title}</h2>
                {subtitle ? <p className="card-subtitle">{subtitle}</p> : null}
            </div>
            {accessory}
        </div>
    );
}

function EmptyState({ icon, title, children }: { icon: ReactNode; title: string; children?: ReactNode }) {
    return (
        <div className="flex flex-col items-center justify-center rounded-[14px] bg-grouped-3 px-6 py-10 text-center">
            <span className="mb-3 text-label-3 [&>svg]:h-8 [&>svg]:w-8" aria-hidden="true">{icon}</span>
            <p className="text-headline text-label">{title}</p>
            {children ? <p className="mt-1 max-w-sm text-subhead text-label-2">{children}</p> : null}
        </div>
    );
}

function Stat({ icon, tint, label, children }: { icon: ReactNode; tint: string; label: string; children: ReactNode }) {
    return (
        <div className="stat min-w-0">
            <div className="flex items-center gap-2.5">
                <span className={`icon-tile ${tint} [&>svg]:h-4 [&>svg]:w-4`} aria-hidden="true">{icon}</span>
                <span className="text-footnote font-medium text-label-2">{label}</span>
            </div>
            <div className="mt-3 truncate text-title-3 text-label">{children}</div>
        </div>
    );
}

function Avatar({ name, tint = 'bg-sys-blue', size = 'h-10 w-10' }: { name?: string; tint?: string; size?: string }) {
    const letters = (name ?? '?')
        .replace(/\(.*?\)/g, '')
        .split(/[^\p{L}\p{N}]+/u)
        .filter(Boolean)
        .slice(0, 2)
        .map((part) => part.charAt(0).toUpperCase())
        .join('') || '?';
    return (
        <span aria-hidden="true" className={`flex shrink-0 items-center justify-center rounded-full text-footnote font-semibold text-white ${tint} ${size}`}>
            {letters}
        </span>
    );
}

/** Maps a backend status string to a pill colour. */
const statusTone = (status?: string) => {
    const value = (status ?? '').toLowerCase();
    if (/(complete|matched|accepted|approved)/.test(value)) return 'pill-green';
    if (/(cancel|declin|reject|fail|expired)/.test(value)) return 'pill-red';
    if (/(progress|active|riding|en_route)/.test(value)) return 'pill-blue';
    if (value) return 'pill-orange';
    return 'pill-gray';
};

const humanize = (value?: string) => (value ?? '').replace(/_/g, ' ');

/** Success copy gets a green notice; everything else stays neutral. */
const noticeTone = (message: string) =>
    /^(route saved|trip started|simulated trip started|trip confirmed)/i.test(message) ? 'notice-green' : 'notice-gray';

function KeyValue({ label, children }: { label: string; children: ReactNode }) {
    return (
        <div className="list-row min-h-[44px] justify-between py-2.5">
            <span className="shrink-0 text-subhead text-label-2">{label}</span>
            <span className="min-w-0 truncate text-right text-subhead font-medium text-label">{children}</span>
        </div>
    );
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
    const { ready: realtimeReady, socket, approachAlert, offers, queueSummary, rooms, riderStatus, approvalRequest, respondToOffer, completeTrip, respondToApproval } = useRealtime();
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
                // eslint-disable-next-line react-hooks/set-state-in-effect -- reopens the room implied by rider:status; compares against this render's activeRoom on purpose
                setActiveRoom({ id: riderStatus.tripId, status: riderStatus as TripStatusPayload });
            }
        }

        return () => {
            socket.off('trip:room-created', onRoomCreated);
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps -- adding activeRoom would reopen the room right after the user minimises it
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

    // Driver and rider have different tab sets, so go back to the dashboard
    // tab when the role flips (adjusting state during render, not in an effect).
    const [tabsForDriver, setTabsForDriver] = useState(isDriver);
    if (tabsForDriver !== isDriver) {
        setTabsForDriver(isDriver);
        setActiveTab('dashboard');
    }

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
        // eslint-disable-next-line react-hooks/exhaustive-deps -- startLocationTracking is recreated every render; adding it would restart the GPS watch on each render
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
            // eslint-disable-next-line react-hooks/set-state-in-effect -- clears polled requests when the viewer stops being a driver, in step with the polling below
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
            // eslint-disable-next-line react-hooks/set-state-in-effect -- mirrors socket queue pushes into the same state the REST poll above writes
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
                name: selfDriver?.name ?? (user.user_metadata?.full_name as string | undefined) ?? user.name ?? user.email ?? 'Driver',
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

    const displayName =
        (user?.user_metadata?.full_name as string | undefined) ?? user?.name ?? user?.email ?? (isDriver ? 'Driver' : 'Rider');

    const driverNetworkCard = (
        <section className="card">
            <CardHeader
                title="Driver network"
                subtitle="Live view of drivers reported by the gateway."
                accessory={
                    <span className="pill pill-gray">
                        {snapshotLoading ? 'Refreshing…' : 'Auto-refreshing every 15s'}
                    </span>
                }
            />
            {snapshotError ? (
                <div className="notice notice-red mb-4" role="alert">
                    <CircleAlert className="mt-0.5 h-[18px] w-[18px] shrink-0" aria-hidden="true" />
                    <span>{snapshotError}</span>
                </div>
            ) : null}
            {drivers.length ? (
                <div className="grid gap-5 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,1fr)]">
                    <DriversMap drivers={drivers} userLocation={location} />
                    <div className="list max-h-[380px] self-start overflow-y-auto overscroll-contain" style={{ ['--row-inset' as string]: '68px' }}>
                        {drivers.map((driver) => (
                            <DriverCard key={driver.id} driver={driver} />
                        ))}
                    </div>
                </div>
            ) : (
                <EmptyState icon={<CarFront />} title="No active drivers yet">
                    Once the gateway reports drivers they’ll appear here.
                </EmptyState>
            )}
        </section>
    );

    const riderNetworkCard = (
        <>
            {approachAlert && (
                <div
                    key={approachAlert.receivedAt}
                    role="status"
                    className="card flex animate-pop items-start gap-4 !p-4 sm:!p-5"
                >
                    <span className="icon-tile h-10 w-10 rounded-[11px] bg-sys-orange [&>svg]:h-5 [&>svg]:w-5" aria-hidden="true">
                        <Navigation />
                    </span>
                    <div className="min-w-0 flex-1">
                        <p className="text-headline text-label">Approaching {approachAlert.stationName}</p>
                        <p className="tabular text-subhead text-label-2">
                            {(approachAlert.distanceMeters / 1000).toFixed(1)} km · ~{Math.max(1, Math.round(approachAlert.etaMinutes))} min out
                        </p>
                        <p className="mt-1 text-footnote text-label-2">
                            Location service detected the approach zone; Matching just refreshed your rider list.
                        </p>
                    </div>
                    <span className="pill pill-orange hidden sm:inline-flex">Approach zone</span>
                </div>
            )}

            <section className="card">
                <div className="mb-1 flex items-center justify-between gap-3">
                    <h3 className="card-title">Live socket offers</h3>
                    <span className={`pill ${realtimeReady ? 'pill-green' : 'pill-gray'}`}>
                        <span className={`pill-dot ${realtimeReady ? '' : 'animate-breathe'}`} aria-hidden="true" />
                        {realtimeReady ? 'connected' : 'connecting…'}
                    </span>
                </div>
                <p className="card-subtitle mb-5">Matching pushes ride offers to you over the realtime socket.</p>
                {offers.length ? (
                    <div className="grid gap-3 md:grid-cols-2">
                        {offers.map((offer) => (
                            <div key={offer.riderId} className="animate-pop rounded-[14px] bg-grouped-3 p-4">
                                <div className="flex items-start justify-between gap-3">
                                    <div className="flex min-w-0 items-center gap-3">
                                        <Avatar name={offer.riderName} tint="bg-sys-orange" />
                                        <div className="min-w-0">
                                            <p className="truncate text-callout font-semibold text-label">{offer.riderName}</p>
                                            <p className="truncate text-footnote text-label-2">
                                                Pickup · {offer.pickupName ?? offer.station?.name ?? 'pending'}
                                            </p>
                                        </div>
                                    </div>
                                    <span className="pill pill-blue tabular" title="Offer attempt">
                                        {offer.attempt}/{offer.total}
                                    </span>
                                </div>
                                <dl className="mt-3 space-y-1 text-footnote">
                                    <div className="flex justify-between gap-3">
                                        <dt className="text-label-2">Destination</dt>
                                        <dd className="truncate text-right font-medium text-label">{offer.destination ?? '—'}</dd>
                                    </div>
                                    {offer.station?.name ? (
                                        <div className="flex justify-between gap-3">
                                            <dt className="text-label-2">Station</dt>
                                            <dd className="truncate text-right font-medium text-label">{offer.station.name}</dd>
                                        </div>
                                    ) : null}
                                </dl>
                                <div className="mt-4 grid grid-cols-2 gap-2">
                                    <button
                                        onClick={() => respondToOffer(offer.riderId, false)}
                                        className="btn btn-gray btn-sm"
                                    >
                                        Pass
                                    </button>
                                    <button
                                        onClick={() => respondToOffer(offer.riderId, true)}
                                        className="btn btn-filled btn-sm"
                                    >
                                        Accept
                                    </button>
                                </div>
                            </div>
                        ))}
                    </div>
                ) : (
                    <EmptyState icon={<Radio />} title="No socket offers right now">
                        Riders will appear here in real time.
                    </EmptyState>
                )}
            </section>

            <section className="card">
                <CardHeader
                    title="Rider network"
                    subtitle="View and accept riders queued along your metro drops."
                    accessory={
                        <span className="pill pill-gray tabular">
                            {driverRequests?.generatedAt
                                ? `Updated ${new Date(driverRequests.generatedAt).toLocaleTimeString()}`
                                : 'Polling every 15s'}
                        </span>
                    }
                />
                {driverRequestError ? (
                    <div className="notice notice-red mb-4" role="alert">
                        <CircleAlert className="mt-0.5 h-[18px] w-[18px] shrink-0" aria-hidden="true" />
                        <span>{driverRequestError}</span>
                    </div>
                ) : null}
                {acceptFeedback ? (
                    <div className={`notice mb-4 animate-pop ${noticeTone(acceptFeedback)}`} role="status">
                        <CircleCheck className="mt-0.5 h-[18px] w-[18px] shrink-0" aria-hidden="true" />
                        <span>{acceptFeedback}</span>
                    </div>
                ) : null}
                {driverRequests && driverRequests.requests.length ? (
                    <div className="grid gap-5 lg:grid-cols-2">
                        <div className="map-frame h-72 lg:h-auto lg:min-h-[320px]">
                            <MapContainer center={driverRequestCenter} zoom={14} className="absolute inset-0">
                                <BaseTiles />
                                {selfDriver?.latitude && selfDriver.longitude ? (
                                    <Marker position={[selfDriver.latitude, selfDriver.longitude]} icon={mapIcons.driver}>
                                        <Popup>You</Popup>
                                    </Marker>
                                ) : null}
                                {driverRequests.requests.map((req) => {
                                    const pickup = req.pickup;
                                    if (pickup?.latitude && pickup.longitude) {
                                        return (
                                            <Marker key={req.id} position={[pickup.latitude, pickup.longitude]} icon={mapIcons.rider}>
                                                <Popup>
                                                    <div>
                                                        <p className="font-semibold">{req.name}</p>
                                                        <p className="text-label-2">{pickup.name}</p>
                                                    </div>
                                                </Popup>
                                            </Marker>
                                        );
                                    }
                                    if (req.station.latitude && req.station.longitude) {
                                        return (
                                            <Marker key={req.id} position={[req.station.latitude, req.station.longitude]} icon={mapIcons.rider}>
                                                <Popup>
                                                    <div>
                                                        <p className="font-semibold">{req.name}</p>
                                                        <p className="text-label-2">{req.station.name}</p>
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
                        <div className="list max-h-[360px] self-start overflow-y-auto overscroll-contain" style={{ ['--row-inset' as string]: '68px' }}>
                            {driverRequests.requests.map((req) => (
                                <div key={req.id} className="list-row items-start py-3.5">
                                    <Avatar name={req.name} tint="bg-sys-orange" />
                                    <div className="min-w-0 flex-1">
                                        <div className="flex items-start justify-between gap-2">
                                            <p className="truncate text-callout font-semibold text-label">{req.name}</p>
                                            <span className={`pill ${req.status === 'matched' ? 'pill-green' : 'pill-orange'}`}>
                                                {humanize(req.status)}
                                            </span>
                                        </div>
                                        <p className="truncate text-footnote text-label-2">Pickup · {req.pickup?.name ?? req.station.name}</p>
                                        <p className="truncate text-footnote text-label-2">
                                            Station · {req.station.name} · <span className="tabular">{formatDistance(req.distanceMeters)}</span>
                                        </p>
                                        <div className="mt-3 flex gap-2">
                                            <button
                                                onClick={() => handleAcceptRider(req.id, false)}
                                                disabled={acceptingRiderId === req.id}
                                                className="btn btn-filled btn-sm flex-1"
                                            >
                                                {acceptingRiderId === req.id ? 'Confirming…' : 'Accept'}
                                            </button>
                                            <button
                                                onClick={() => handleAcceptRider(req.id, true)}
                                                disabled={acceptingRiderId === req.id}
                                                className="btn btn-tinted btn-sm !px-3"
                                                title="Simulate Trip"
                                            >
                                                <CirclePlay size={18} />
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                ) : (
                    <EmptyState icon={<Inbox />} title="No riders queued yet">
                        Waiting for riders near your stations. Hang tight.
                    </EmptyState>
                )}
            </section>
        </>
    );

    const riderBookingCard = (
        <section className="card">
            <CardHeader title="Book a ride from Metro" subtitle="Select your metro station and destination." />
            <form onSubmit={handleBookRide}>
                <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,0.85fr)]">
                    <div className="min-w-0 space-y-6">
                        <div>
                            <p className="section-label mb-2">Station</p>
                            <div className="relative mb-3">
                                <Search className="field-icon" aria-hidden="true" />
                                <input
                                    value={pickupQuery}
                                    onChange={(e) => setPickupQuery(e.target.value)}
                                    className="field pl-10"
                                    placeholder="Search Metro Station"
                                    aria-label="Search metro station"
                                />
                            </div>
                            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
                                {filteredPickupOptions.map((point: PickupPoint) => (
                                    <button
                                        key={point.id}
                                        type="button"
                                        aria-pressed={selectedPickupId === point.id}
                                        onClick={() => {
                                            setSelectedPickupId(point.id);
                                            setSelectedDestination(null); // Reset destination when station changes
                                        }}
                                        className="chip"
                                    >
                                        <span className="chip-check" aria-hidden="true">
                                            <Check className="h-3.5 w-3.5" strokeWidth={3} />
                                        </span>
                                        <span className="min-w-0 flex-1 text-subhead font-semibold">{point.name}</span>
                                    </button>
                                ))}
                            </div>
                        </div>
                        <div>
                            <p className="section-label mb-2">Destination</p>
                            {selectedPickup ? (
                                <div className="flex flex-wrap gap-2 animate-fade">
                                    {/* Destinations are the nearby areas of the selected station. */}
                                    {DESTINATIONS.filter(d => d.stationId === selectedPickup.stationId).map((dest) => (
                                        <button
                                            key={dest.id}
                                            type="button"
                                            aria-pressed={selectedDestination === dest.name}
                                            onClick={() => setSelectedDestination(dest.name)}
                                            className="tag"
                                        >
                                            {dest.name}
                                        </button>
                                    ))}
                                </div>
                            ) : (
                                <p className="px-1 text-subhead text-label-2">Tap a station above to see destinations.</p>
                            )}
                        </div>
                        {bookingError ? (
                            <div className="notice notice-red" role="alert">
                                <CircleAlert className="mt-0.5 h-[18px] w-[18px] shrink-0" aria-hidden="true" />
                                <span>{bookingError}</span>
                            </div>
                        ) : null}
                        <button
                            type="submit"
                            disabled={bookingLoading}
                            className="btn btn-filled btn-lg w-full"
                        >
                            {bookingLoading ? 'Booking…' : 'Book Ride'}
                        </button>
                    </div>
                    <div className="map-frame h-64 lg:h-auto lg:min-h-[360px]">
                        <MapContainer
                            center={selectedPickup ? [selectedPickup.latitude, selectedPickup.longitude] : [location?.lat ?? 12.8456, location?.lng ?? 77.66]}
                            zoom={selectedPickup ? 15 : 13}
                            className="absolute inset-0"
                        >
                            <BaseTiles />
                            {selectedPickup ? (
                                <Marker position={[selectedPickup.latitude, selectedPickup.longitude]} icon={mapIcons.station}>
                                    <Popup>
                                        <div>
                                            <p className="font-semibold">{selectedPickup.name}</p>
                                            <p className="text-label-2">Nearest station {selectedPickup.stationName}</p>
                                        </div>
                                    </Popup>
                                </Marker>
                            ) : null}
                            {drivers
                                .filter((driver) => typeof driver.latitude === 'number' && typeof driver.longitude === 'number')
                                .map((driver) => (
                                    <Marker key={driver.id} position={[driver.latitude!, driver.longitude!]} icon={mapIcons.driver}>
                                        <Popup>{driver.name}</Popup>
                                    </Marker>
                                ))}
                        </MapContainer>
                    </div>
                </div>
            </form>
            {bookingState && (
                <div className="hairline-t mt-6 animate-pop pt-6">
                    <div className="flex items-start gap-3">
                        <span className="icon-tile h-9 w-9 rounded-full bg-sys-blue [&>svg]:h-[18px] [&>svg]:w-[18px]" aria-hidden="true">
                            <CarFront />
                        </span>
                        <div className="min-w-0 flex-1">
                            <p className="text-headline text-label">{bookingState.message}</p>
                            <p className="text-footnote text-label-2">Destination · {bookingState.requestedDestination}</p>
                        </div>
                    </div>
                    {bookingState.matchedBy && (
                        <div className="ml-12 mt-3 inline-flex items-center gap-2 rounded-full bg-sys-indigo/[var(--tint-a)] px-3 py-1 text-footnote text-indigo-text">
                            <span className="h-1.5 w-1.5 rounded-full bg-current" aria-hidden="true" />
                            Decided by <span className="font-mono font-medium">{bookingState.matchedBy}</span>
                        </div>
                    )}
                    {bookingState.attempts.length || bookingState.rejections?.length ? (
                        <div className="list mt-4" style={{ ['--row-inset' as string]: '36px' }}>
                            {bookingState.attempts.map((attempt) => (
                                <div key={attempt.driverId} className="list-row min-h-[44px] py-2.5">
                                    <span className={`h-2 w-2 shrink-0 rounded-full ${attempt.accepted ? 'bg-sys-green' : 'bg-sys-gray'}`} aria-hidden="true" />
                                    <span className={`min-w-0 flex-1 truncate text-subhead ${attempt.accepted ? 'font-semibold text-green-text' : 'text-label'}`}>
                                        {attempt.driverName}
                                    </span>
                                    <span className="text-right text-footnote text-label-2">
                                        {attempt.accepted ? 'Accepted' : attempt.reason || 'Pending'}
                                    </span>
                                </div>
                            ))}
                            {bookingState.rejections?.map((r) => (
                                <div key={`rej-${r.driverId}`} className="list-row min-h-[44px] items-start py-2.5">
                                    <span className="mt-[7px] h-2 w-2 shrink-0 rounded-full bg-sys-orange" aria-hidden="true" />
                                    <span className="min-w-0 flex-1 truncate text-subhead text-label-2 line-through">{r.driverName || r.driverId}</span>
                                    <span className="max-w-[60%] text-right text-footnote text-orange-text">{r.reason}</span>
                                </div>
                            ))}
                        </div>
                    ) : null}
                </div>
            )}
            {approvalRequest ? (
                <section className="mt-6 animate-pop rounded-[16px] bg-sys-blue/[var(--tint-a)] p-4 sm:p-5" aria-label="Ride approval">
                    <div className="flex items-start gap-3">
                        <span className="icon-tile h-10 w-10 rounded-full bg-sys-blue [&>svg]:h-5 [&>svg]:w-5" aria-hidden="true">
                            <CarFront />
                        </span>
                        <div className="min-w-0 flex-1">
                            <p className="text-headline text-label">{approvalRequest.driverName || 'Driver'} accepted your request</p>
                            <p className="text-subhead text-label-2">Pickup · {approvalRequest.pickup?.name ?? '—'}</p>
                            <p className="text-subhead text-label-2">Station · {approvalRequest.station?.name ?? '—'}</p>
                        </div>
                    </div>
                    <div className="mt-4 grid grid-cols-2 gap-2">
                        <button
                            onClick={() => respondToApproval(approvalRequest.tripId, false)}
                            className="btn btn-destructive !bg-grouped-2 hover:!bg-grouped-2/80"
                        >
                            Decline
                        </button>
                        <button
                            onClick={() => respondToApproval(approvalRequest.tripId, true)}
                            className="btn btn-filled"
                        >
                            Approve ride
                        </button>
                    </div>
                </section>
            ) : null}
            {riderStatus ? (
                <div className="hairline-t mt-6 animate-fade pt-6">
                    <div className="mb-3 flex items-center justify-between gap-3">
                        <p className="flex items-center gap-2 text-headline text-label">
                            Live status
                            <span className={`pill ${statusTone(riderStatus.status)}`}>{humanize(riderStatus.status)}</span>
                        </p>
                        <span className="flex items-center gap-1 text-footnote text-label-2 tabular">
                            <Clock className="h-3.5 w-3.5" aria-hidden="true" />
                            {new Date(riderStatus.recordedAt).toLocaleTimeString()}
                        </span>
                    </div>
                    <div className="list">
                        <KeyValue label="Driver">{riderStatus.driverId ?? 'Assigning'}</KeyValue>
                        <KeyValue label="Pickup">{riderStatus.pickup?.name ?? '—'}</KeyValue>
                    </div>
                    {riderStatus.latitude && riderStatus.longitude ? (
                        <div className="map-frame mt-3 h-48">
                            <MapContainer center={[riderStatus.latitude, riderStatus.longitude]} zoom={15} className="absolute inset-0">
                                <BaseTiles />
                                <Marker position={[riderStatus.latitude, riderStatus.longitude]} icon={mapIcons.driver}>
                                    <Popup>Driver</Popup>
                                </Marker>
                                {riderStatus.pickup?.latitude && riderStatus.pickup.longitude ? (
                                    <Marker position={[riderStatus.pickup.latitude, riderStatus.pickup.longitude]} icon={mapIcons.station}>
                                        <Popup>{riderStatus.pickup.name}</Popup>
                                    </Marker>
                                ) : null}
                                {riderStatus.station?.latitude && riderStatus.station.longitude ? (
                                    <Marker position={[riderStatus.station.latitude, riderStatus.station.longitude]} icon={stationDotIcon}>
                                        <Popup>{riderStatus.station.name}</Popup>
                                    </Marker>
                                ) : null}
                            </MapContainer>
                        </div>
                    ) : null}
                </div>
            ) : null}
        </section>
    );

    const driverRoutePlanner = (
        <section className="card">
            <CardHeader
                title="Plan your metro route"
                subtitle="Select the stations you are covering."
                accessory={
                    selectedRoutePoints.length ? (
                        <span className="pill pill-blue">{selectedRoutePoints.length} {selectedRoutePoints.length === 1 ? 'stop' : 'stops'}</span>
                    ) : null
                }
            />
            <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,0.85fr)]">
                <div className="min-w-0 space-y-4">
                    <div className="relative">
                        <Search className="field-icon" aria-hidden="true" />
                        <input
                            value={routeQuery}
                            onChange={(e) => setRouteQuery(e.target.value)}
                            className="field pl-10"
                            placeholder="Search Station"
                            aria-label="Search station"
                        />
                    </div>
                    <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
                        {filteredRouteOptions.map((point: PickupPoint) => {
                            const order = routePickupIds.indexOf(point.id) + 1;
                            return (
                                <button
                                    key={point.id}
                                    type="button"
                                    aria-pressed={order > 0}
                                    onClick={() => toggleRoutePickup(point.id)}
                                    className="chip"
                                >
                                    <span className="chip-check tabular text-caption-2 font-bold" aria-hidden="true">
                                        {order > 0 ? order : ''}
                                    </span>
                                    <span className="min-w-0 flex-1">
                                        <span className="block text-subhead font-semibold">{point.name}</span>
                                        <span className="block truncate text-footnote text-label-2">{point.stationName}</span>
                                    </span>
                                </button>
                            );
                        })}
                    </div>
                    <p className="px-1 text-footnote text-label-2">
                        {selectedRoutePoints.length
                            ? `Stops · ${selectedRoutePoints.map((point) => point.name).join(', ')}`
                            : 'Tap the stations above to add them to your shuttle route.'}
                    </p>
                    <label className="flex min-h-[48px] cursor-text items-center justify-between gap-4 rounded-[12px] bg-fill px-4 transition-shadow focus-within:shadow-[0_0_0_1px_rgb(var(--sys-blue)),0_0_0_4px_rgb(var(--sys-blue)/0.25)]">
                        <span className="flex items-center gap-2.5 text-callout text-label">
                            <Users className="h-[18px] w-[18px] text-label-2" aria-hidden="true" />
                            Seats available
                        </span>
                        <input
                            value={routeSeatCount}
                            onChange={(e) => setRouteSeatCount(e.target.value)}
                            inputMode="numeric"
                            className="w-20 bg-transparent text-right text-callout font-semibold text-label tabular outline-none"
                            placeholder="Seats"
                        />
                    </label>
                    {routeStatus ? (
                        <div className={`notice animate-pop ${noticeTone(routeStatus)}`} role="status">
                            <CircleCheck className="mt-0.5 h-[18px] w-[18px] shrink-0" aria-hidden="true" />
                            <span>{routeStatus}</span>
                        </div>
                    ) : null}
                    <div className="grid gap-2 sm:grid-cols-3">
                        <button
                            onClick={handleSaveRoute}
                            className="btn btn-filled"
                            disabled={savingRoute}
                        >
                            {savingRoute ? 'Saving…' : 'Save Route'}
                        </button>
                        <button
                            onClick={() => handleStartTripPlan(false)}
                            className="btn btn-tinted"
                            disabled={startingTrip}
                        >
                            {startingTrip ? 'Starting…' : 'Start Trip'}
                        </button>
                        <button
                            onClick={() => handleStartTripPlan(true)}
                            className="btn btn-tinted"
                            disabled={startingTrip}
                        >
                            <CirclePlay className="h-[18px] w-[18px]" aria-hidden="true" />
                            Simulated Trip
                        </button>
                    </div>
                    {tripStatus ? (
                        <div className={`notice animate-pop ${noticeTone(tripStatus)}`} role="status">
                            <Route className="mt-0.5 h-[18px] w-[18px] shrink-0" aria-hidden="true" />
                            <span>{tripStatus}</span>
                        </div>
                    ) : null}
                </div>
                <div className="map-frame h-72 lg:h-auto lg:min-h-[420px]">
                    <MapContainer
                        center={selectedRoutePoints.length ? [selectedRoutePoints[0].latitude, selectedRoutePoints[0].longitude] : [selfDriver?.latitude ?? 12.8456, selfDriver?.longitude ?? 77.66]}
                        zoom={selectedRoutePoints.length ? 14 : 12}
                        className="absolute inset-0"
                    >
                        <BaseTiles />
                        {selectedRoutePoints.length >= 2 ? (
                            <RouteLine positions={selectedRoutePoints.map((point) => [point.latitude, point.longitude] as [number, number])} />
                        ) : null}
                        {selfDriver?.latitude && selfDriver.longitude ? (
                            <Marker position={[selfDriver.latitude, selfDriver.longitude]} icon={mapIcons.driver}>
                                <Popup>You</Popup>
                            </Marker>
                        ) : null}
                        {selectedRoutePoints.map((point: PickupPoint, index) => (
                            <Marker key={point.id} position={[point.latitude, point.longitude]} icon={stopIcon(index + 1)}>
                                <Popup>{point.name}</Popup>
                            </Marker>
                        ))}
                    </MapContainer>
                </div>
            </div>
        </section>
    );

    const tripRoomsPanel = (
        <section className="card">
            <CardHeader
                title="Active trip rooms"
                subtitle="Drivers and riders linked in realtime."
                accessory={
                    <span className={`pill ${rooms.length ? 'pill-blue' : 'pill-gray'}`}>
                        {rooms.length ? `${rooms.length} active` : 'No active trips'}
                    </span>
                }
            />
            {rooms.length ? (
                <div className="grid gap-4 md:grid-cols-2">
                    {rooms.map((room) => {
                        const centerLat = room.latitude ?? room.pickup?.latitude ?? room.station?.latitude ?? 12.8456;
                        const centerLon = room.longitude ?? room.pickup?.longitude ?? room.station?.longitude ?? 77.66;
                        return (
                            <div key={room.tripId} className="animate-pop overflow-hidden rounded-[14px] bg-grouped-3">
                                <div className="flex items-start justify-between gap-3 p-4 pb-3">
                                    <div className="min-w-0">
                                        <p className="truncate text-callout font-semibold text-label">{room.trip?.riderId ?? room.riderId}</p>
                                        <p className="font-mono text-caption text-label-2">Trip · {room.tripId.slice(0, 8)}</p>
                                    </div>
                                    <span className={`pill ${statusTone(room.status)}`}>{humanize(room.status)}</span>
                                </div>
                                <div className="map-frame mx-4 h-40 rounded-[12px]">
                                    <MapContainer center={[centerLat, centerLon]} zoom={14} className="absolute inset-0">
                                        <BaseTiles />
                                        {room.pickup?.latitude && room.pickup.longitude ? (
                                            <Marker position={[room.pickup.latitude, room.pickup.longitude]} icon={mapIcons.station}>
                                                <Popup>{room.pickup.name}</Popup>
                                            </Marker>
                                        ) : null}
                                        {room.station?.latitude && room.station.longitude ? (
                                            <Marker position={[room.station.latitude, room.station.longitude]} icon={stationDotIcon}>
                                                <Popup>{room.station.name}</Popup>
                                            </Marker>
                                        ) : null}
                                        {room.latitude && room.longitude ? (
                                            <Marker position={[room.latitude, room.longitude]} icon={mapIcons.driver} zIndexOffset={1000}>
                                                <Popup>Driver</Popup>
                                            </Marker>
                                        ) : null}
                                        <MapUpdater center={[centerLat, centerLon]} />
                                    </MapContainer>
                                </div>
                                <div className="px-4 pt-1">
                                    <div className="flex items-center gap-2 py-1.5 text-footnote">
                                        <span className="h-2 w-2 shrink-0 rounded-full bg-sys-green" aria-hidden="true" />
                                        <span className="text-label-2">Pickup</span>
                                        <span className="min-w-0 truncate font-medium text-label">{room.pickup?.name ?? '—'}</span>
                                    </div>
                                    <div className="flex items-center gap-2 py-1.5 text-footnote">
                                        <span className="h-2 w-2 shrink-0 rounded-full bg-sys-indigo" aria-hidden="true" />
                                        <span className="text-label-2">Station</span>
                                        <span className="min-w-0 truncate font-medium text-label">{room.station?.name ?? '—'}</span>
                                    </div>
                                </div>
                                <div className="grid grid-cols-2 gap-2 p-4 pt-3">
                                    <button
                                        onClick={() => setActiveRoom({ id: room.tripId, status: { ...room, recordedAt: room.updatedAt || new Date().toISOString() } })}
                                        className="btn btn-filled btn-sm"
                                    >
                                        Resume
                                    </button>
                                    <button
                                        onClick={() => completeTrip(room.tripId)}
                                        className="btn btn-tinted btn-sm"
                                    >
                                        Mark Complete
                                    </button>
                                </div>
                            </div>
                        );
                    })}
                </div>
            ) : (
                <EmptyState icon={<Route />} title="No active trips">
                    No active socket rooms yet.
                </EmptyState>
            )}
        </section>
    );

    const driverStatusCard = (
        <section className="card">
            <div className="flex flex-wrap items-center justify-between gap-4">
                <div className="flex items-center gap-3.5">
                    <span
                        className={`flex h-11 w-11 items-center justify-center rounded-full transition-colors duration-300 ${isOnline ? 'bg-sys-green text-white' : 'bg-fill text-label-2'}`}
                        aria-hidden="true"
                    >
                        <Power className="h-5 w-5" />
                    </span>
                    <div>
                        <h2 className="card-title">Current Status</h2>
                        <p className="flex items-center gap-1.5 text-subhead text-label-2">
                            <span className={`h-2 w-2 rounded-full ${isOnline ? 'bg-sys-green' : 'bg-sys-gray'}`} aria-hidden="true" />
                            <span className="capitalize">{status}</span>
                        </p>
                    </div>
                </div>
                <button
                    onClick={toggleOnline}
                    className={`btn ${isOnline ? 'btn-destructive' : 'btn-filled'}`}
                >
                    <Power className="h-[18px] w-[18px]" aria-hidden="true" />
                    {isOnline ? 'Go Offline' : 'Go Online'}
                </button>
            </div>

            {error && (
                <div className="notice notice-red mt-5" role="alert">
                    <CircleAlert className="mt-0.5 h-[18px] w-[18px] shrink-0" aria-hidden="true" />
                    <span>{error}</span>
                </div>
            )}

            <div className="mt-5 grid grid-cols-1 gap-3 sm:grid-cols-3">
                <Stat icon={<MapPin />} tint="bg-sys-blue" label="Destination">
                    {plannedDestination}
                </Stat>
                <Stat icon={<Users />} tint="bg-sys-indigo" label="Available Seats">
                    <span className="tabular">{plannedSeats}</span>
                </Stat>
                <Stat icon={<Navigation />} tint="bg-sys-green" label="Location">
                    {location ? (
                        <span className="font-mono text-callout tabular">{`${location.lat.toFixed(4)}, ${location.lng.toFixed(4)}`}</span>
                    ) : (
                        <span className="text-callout font-normal text-label-2">Waiting for signal...</span>
                    )}
                </Stat>
            </div>
        </section>
    );

    const profilePanel = (
        <section className="card">
            <div className="mb-6 flex items-center gap-4">
                <Avatar name={displayName} tint={isDriver ? 'bg-sys-blue' : 'bg-sys-indigo'} size="h-16 w-16 !text-title-3" />
                <div className="min-w-0">
                    <h2 className="truncate text-title-2 text-label">{displayName}</h2>
                    <p className="text-subhead text-label-2">Your account details and session actions.</p>
                </div>
            </div>
            <div className="grid gap-6 md:grid-cols-2">
                <div>
                    <p className="section-label mb-2">Account</p>
                    <div className="list">
                        <KeyValue label="Email">{user?.email ?? '—'}</KeyValue>
                        <KeyValue label="Role">{isDriver ? 'Driver' : 'Rider'}</KeyValue>
                    </div>
                </div>
                <div>
                    <p className="section-label mb-2">Session</p>
                    <div className="list">
                        <button
                            onClick={toggleOnline}
                            className="list-row min-h-[44px] w-full py-2.5 text-left text-subhead font-medium text-blue-text transition-colors hover:bg-fill"
                        >
                            <Power className="h-[18px] w-[18px]" aria-hidden="true" />
                            {isOnline ? 'Go Offline' : 'Go Online'}
                        </button>
                        <button
                            onClick={() => signOut()}
                            className="list-row min-h-[44px] w-full py-2.5 text-left text-subhead font-medium text-red-text transition-colors hover:bg-fill"
                        >
                            <LogOut className="h-[18px] w-[18px]" aria-hidden="true" />
                            Sign Out
                        </button>
                    </div>
                </div>
            </div>
        </section>
    );

    const locationCard = (
        <section className="card">
            <CardHeader
                title="Your location"
                subtitle={
                    location
                        ? `${location.lat.toFixed(4)}, ${location.lng.toFixed(4)}`
                        : error
                            ? 'Location is unavailable in this browser.'
                            : 'Acquiring location…'
                }
                accessory={
                    location ? (
                        <span className="pill pill-green">
                            <span className="pill-dot" aria-hidden="true" />
                            {simulating ? 'Simulated' : 'Live'}
                        </span>
                    ) : null
                }
            />
            <div className={`map-frame transition-[height] duration-300 ${location ? 'h-[360px] sm:h-[420px]' : 'h-[200px]'}`}>
                {location ? (
                    <MapContainer center={[location.lat, location.lng]} zoom={15} className="absolute inset-0">
                        <BaseTiles />
                        <Marker position={[location.lat, location.lng]} icon={mapIcons.user}>
                            <Popup>
                                {isDriver ? 'You are here' : 'Your Location'}
                            </Popup>
                        </Marker>
                        <MapUpdater center={[location.lat, location.lng]} />
                    </MapContainer>
                ) : (
                    <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 px-6 text-center">
                        <LocateFixed className={`h-8 w-8 text-label-2 ${error ? '' : 'animate-breathe'}`} aria-hidden="true" />
                        <p className="max-w-xs text-subhead text-label-2">{error ? error : 'Acquiring location...'}</p>
                    </div>
                )}
            </div>
        </section>
    );

    const renderDriverContent = () => {
        switch (activeTab) {
            case 'dashboard':
                return (
                    <>
                        {driverStatusCard}
                        {driverNetworkCard}
                    </>
                );
            case 'riders':
                return riderNetworkCard;
            case 'trips':
                return (
                    <>
                        {driverRoutePlanner}
                        {tripRoomsPanel}
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
                        {riderBookingCard}
                        {driverNetworkCard}
                    </>
                );
        }
    };

    const pageSubtitles: Record<string, string> = isDriver
        ? {
            dashboard: 'Your status and the drivers around you.',
            riders: 'Offers and riders queued along your stations.',
            trips: 'Plan a route, start a trip and follow live rooms.',
            profile: 'Your account and session.',
        }
        : {
            dashboard: 'Book a ride and see drivers nearby.',
            drivers: 'Drivers reported live by the gateway.',
            book: 'Pick your station and where you’re headed.',
            profile: 'Your account and session.',
        };
    const activeTabLabel = tabs.find((tab) => tab.key === activeTab)?.label ?? 'Dashboard';

    return (
        <div className="min-h-screen bg-grouped text-label">
            {/* Translucent navigation bar */}
            <header className="material hairline-b sticky top-0 z-[1000]">
                <div className="mx-auto flex max-w-6xl flex-wrap items-center gap-x-4 gap-y-2.5 px-4 py-2.5 sm:px-6">
                    <div className="flex min-w-0 flex-1 items-center gap-3 md:flex-none md:basis-64">
                        <div className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px] text-white ${isDriver ? 'bg-accent' : 'bg-sys-indigo'}`} aria-hidden="true">
                            {isDriver ? <Navigation className="h-[18px] w-[18px]" /> : <Search className="h-[18px] w-[18px]" />}
                        </div>
                        <div className="min-w-0">
                            <p className="truncate text-headline text-label">{isDriver ? 'Driver Console' : 'Rider App'}</p>
                            <p className="truncate text-caption text-label-2">
                                <span className="sr-only">Logged in as {isDriver ? 'Driver' : 'Rider'} · </span>
                                {user?.email}
                            </p>
                        </div>
                    </div>
                    <nav aria-label="Sections" className="order-last w-full md:order-none md:flex md:w-auto md:flex-1 md:justify-center">
                        <SegmentedControl
                            ariaLabel="Sections"
                            options={tabs}
                            value={activeTab}
                            onChange={setActiveTab}
                            className="w-full md:w-[440px]"
                        />
                    </nav>
                    <div className="flex shrink-0 items-center justify-end gap-2 md:basis-64">
                        <span className={`pill hidden sm:inline-flex ${realtimeReady ? 'pill-green' : 'pill-gray'}`} title="Realtime connection">
                            <span className={`pill-dot ${realtimeReady ? '' : 'animate-breathe'}`} aria-hidden="true" />
                            {isDriver ? 'Driver' : 'Rider'}
                        </span>
                        <button
                            onClick={() => signOut()}
                            className="icon-btn"
                            aria-label="Sign out"
                            title="Sign out"
                        >
                            <LogOut className="h-[18px] w-[18px]" aria-hidden="true" />
                        </button>
                    </div>
                </div>
            </header>

            <main className="mx-auto max-w-6xl px-4 pb-[max(4rem,env(safe-area-inset-bottom))] sm:px-6">
                <div className="pb-5 pt-7 sm:pt-9">
                    <h1 className="text-large-title text-label">{activeTabLabel}</h1>
                    <p className="mt-1 text-callout text-label-2">{pageSubtitles[activeTab] ?? ''}</p>
                </div>

                <div key={activeTab} className="animate-in space-y-6">
                    {isDriver ? renderDriverContent() : renderRiderContent()}
                </div>
                <div className="mt-6">{locationCard}</div>
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
