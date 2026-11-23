
import { useEffect, useLayoutEffect, useRef, useState, useMemo } from 'react';
import type { CSSProperties } from 'react';
import { MapContainer, Marker, Popup } from 'react-leaflet';
import { Navigation, Phone, MessageSquare, Shield, Star, MapPin, Users, Minimize2, Zap, CircleCheck } from 'lucide-react';
import { useRealtime } from '../contexts/RealtimeContext';
import { useAuth } from '../contexts/AuthContext';
import type { TripStatusPayload } from '../lib/types';
import { DESTINATIONS } from '../lib/destinations';
import { BaseTiles, MapUpdater } from './MapTiles';
import { mapIcons } from './mapIcons';

interface RoomProps {
    tripId: string;
    initialStatus: TripStatusPayload;
    onClose: () => void;
    seats?: number;
}

export function Room({ tripId, initialStatus, onClose, seats }: RoomProps) {
    const { user } = useAuth();
    const { socket } = useRealtime();
    const [status, setStatus] = useState<TripStatusPayload>(initialStatus);
    const [driverLocation, setDriverLocation] = useState<[number, number] | null>(
        initialStatus.latitude && initialStatus.longitude
            ? [initialStatus.latitude, initialStatus.longitude]
            : null
    );

    const isDriver = user?.id === status.driverId;
    const partnerName = isDriver ? 'Rider' : 'Driver';

    // Find destination coordinates (if available)
    const destinationPoint = useMemo(() => {
        const destName = status.trip?.destination;
        if (!destName) return null;
        return DESTINATIONS.find(d => d.name === destName);
    }, [status.trip?.destination]);

    useEffect(() => {
        if (!socket) return;

        const onLocation = (payload: TripStatusPayload) => {
            if (payload.tripId === tripId && payload.latitude && payload.longitude) {
                setDriverLocation([payload.latitude, payload.longitude]);
            }
        };

        const onStatus = (payload: TripStatusPayload) => {
            if (payload.tripId === tripId) {
                // Merge: status events don't repeat the trip/pickup details, so
                // replacing the state would lose the drop-off destination.
                setStatus((prev) => ({
                    ...prev,
                    ...payload,
                    trip: payload.trip ?? prev.trip,
                    pickup: payload.pickup ?? prev.pickup,
                    station: payload.station ?? prev.station,
                }));
                if (payload.status === 'completed' || payload.status === 'cancelled') {
                    // Optional: Auto-close or show summary
                }
            }
        };

        socket.on('trip:location', onLocation);
        socket.on('trip:status', onStatus);

        return () => {
            socket.off('trip:location', onLocation);
            socket.off('trip:status', onStatus);
        };
    }, [socket, tripId]);

    const handleAction = async (action: 'pickup' | 'dropoff' | 'simulate') => {
        console.log('handleAction called', action, tripId);
        try {
            await fetch(`/api/trips/${action}?tripId=${tripId}`, {
                method: 'POST',
            });
        } catch (err) {
            console.error('Action failed:', err);
        }
    };

    // Keep Leaflet's attribution visible above the bottom sheet on phones.
    const sheetRef = useRef<HTMLDivElement | null>(null);
    const [sheetHeight, setSheetHeight] = useState(0);
    useLayoutEffect(() => {
        const node = sheetRef.current;
        if (!node || typeof ResizeObserver === 'undefined') return;
        const observer = new ResizeObserver(() => setSheetHeight(node.offsetHeight));
        observer.observe(node);
        return () => observer.disconnect();
    }, []);

    const completed = status.status === 'completed';
    const awaitingPickup = status.status === 'awaiting_pickup' || status.status === 'pending';
    const statusLabel = completed
        ? 'Completed'
        : awaitingPickup
            ? 'Heading to pickup'
            : status.status === 'in_progress'
                ? 'On trip'
                : (status.status ?? '').replace(/_/g, ' ');
    const statusPill = completed ? 'pill-green' : awaitingPickup ? 'pill-orange' : 'pill-blue';

    return (
        <div
            className="room-map fixed inset-0 z-[2000] bg-grouped animate-fade"
            style={{ '--sheet-h': `${sheetHeight}px` } as CSSProperties}
        >
            {/* Map Area */}
            <div className="absolute inset-0">
                <MapContainer
                    center={driverLocation || [12.9716, 77.5946]}
                    zoom={15}
                    className="absolute inset-0"
                    zoomControl={false}
                >
                    <BaseTiles />

                    {/* Pickup Point (Station) */}
                    {status.pickup && status.pickup.latitude && status.pickup.longitude && (
                        <Marker position={[status.pickup.latitude, status.pickup.longitude]} icon={mapIcons.station}>
                            <Popup>Pickup: {status.pickup.name}</Popup>
                        </Marker>
                    )}

                    {/* Dropoff Point (Destination) */}
                    {destinationPoint ? (
                        <Marker position={[destinationPoint.latitude, destinationPoint.longitude]} icon={mapIcons.destination}>
                            <Popup>Dropoff: {destinationPoint.name}</Popup>
                        </Marker>
                    ) : null}

                    {/* Driver Marker */}
                    {driverLocation && (
                        <Marker position={driverLocation} icon={mapIcons.car} zIndexOffset={1000}>
                            <Popup>Driver</Popup>
                        </Marker>
                    )}

                    {driverLocation && <MapUpdater center={driverLocation} />}
                </MapContainer>
            </div>

            {/* Navigation bar (material over the map) */}
            <header className="material hairline-b absolute inset-x-0 top-0 z-[1000] pt-[env(safe-area-inset-top)]">
                <div className="flex items-center justify-between gap-3 px-4 py-3 sm:px-5">
                    <div className="min-w-0">
                        <h2 className="truncate text-title-3 text-label">
                            {status.status === 'completed' ? 'Trip Completed' : 'In Ride'}
                        </h2>
                        <p className="truncate text-footnote text-label-2">
                            {completed ? 'You’ve arrived' : status.status === 'awaiting_pickup' ? 'Heading to pickup' : 'On the way to destination'}
                        </p>
                    </div>
                    <div className="flex shrink-0 items-center gap-2">
                        {isDriver && (
                            <button
                                onClick={() => handleAction('simulate')}
                                className="icon-btn text-orange-text"
                                title="Fast Forward Simulation"
                                aria-label="Fast forward simulation"
                            >
                                <Zap className="h-[18px] w-[18px]" aria-hidden="true" />
                            </button>
                        )}
                        <button className="icon-btn hidden sm:inline-flex" title="Call" aria-label="Call">
                            <Phone className="h-[18px] w-[18px]" aria-hidden="true" />
                        </button>
                        <button className="icon-btn hidden sm:inline-flex" title="Message" aria-label="Message">
                            <MessageSquare className="h-[18px] w-[18px]" aria-hidden="true" />
                        </button>
                        <button className="icon-btn hidden sm:inline-flex" title="Safety" aria-label="Safety">
                            <Shield className="h-[18px] w-[18px]" aria-hidden="true" />
                        </button>
                        <button onClick={onClose} className="icon-btn" title="Minimize" aria-label="Minimize">
                            <Minimize2 className="h-[18px] w-[18px]" aria-hidden="true" />
                        </button>
                    </div>
                </div>
            </header>

            {/* Trip card */}
            <div className="pointer-events-none absolute inset-x-4 top-[calc(env(safe-area-inset-top)+76px)] z-[1000] sm:inset-x-5 md:right-auto md:w-[360px]">
                <div className="material-thick pointer-events-auto animate-pop rounded-[20px] p-4 shadow-float">
                    <div className="flex items-center gap-3">
                        <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-fill text-label-2" aria-hidden="true">
                            <Users className="h-5 w-5" />
                        </div>
                        <div className="min-w-0 flex-1">
                            <h3 className="text-headline text-label">{partnerName}</h3>
                            <div className="flex items-center gap-1 text-footnote text-label-2">
                                <Star className="h-3 w-3 text-sys-orange" fill="currentColor" aria-hidden="true" />
                                <span>4.9</span>
                            </div>
                        </div>
                        <span className={`pill ${statusPill}`}>{statusLabel}</span>
                    </div>

                    <div className="mt-4 rounded-[14px] bg-fill">
                        <div className="relative flex items-start gap-3 px-3.5 py-3">
                            <span className="mt-1 flex h-3 w-3 shrink-0 items-center justify-center rounded-full bg-sys-green ring-[3px] ring-sys-green/25" aria-hidden="true" />
                            <div className="min-w-0">
                                <p className="text-caption-2 font-semibold uppercase tracking-[0.05em] text-label-2">Pickup</p>
                                <p className="truncate text-subhead font-medium text-label">{status.pickup?.name}</p>
                            </div>
                            {/* connector between the two stops */}
                            <span className="absolute -bottom-2 left-[19.5px] h-4 border-l-2 border-dotted border-label-3" aria-hidden="true" />
                        </div>
                        <div className="hairline-t ml-[38px]" />
                        <div className="flex items-start gap-3 px-3.5 py-3">
                            <span className="mt-1 flex h-3 w-3 shrink-0 items-center justify-center rounded-[3px] bg-sys-red ring-[3px] ring-sys-red/20" aria-hidden="true" />
                            <div className="min-w-0">
                                <p className="text-caption-2 font-semibold uppercase tracking-[0.05em] text-label-2">Dropoff</p>
                                <p className="truncate text-subhead font-medium text-label">{status.trip?.destination ?? status.station?.name}</p>
                            </div>
                        </div>
                    </div>
                    {typeof seats === 'number' && (
                        <div className="mt-3 flex items-center justify-between px-1">
                            <p className="text-footnote text-label-2">Seats Available</p>
                            <p className="tabular text-headline text-label">{seats}</p>
                        </div>
                    )}
                </div>
            </div>

            {/* Bottom Actions */}
            <div
                ref={sheetRef}
                className="material-thick absolute inset-x-0 bottom-0 z-[1000] rounded-t-[22px] p-4 pb-[max(1rem,env(safe-area-inset-bottom))] shadow-float md:bottom-5 md:left-5 md:right-auto md:w-[360px] md:rounded-[20px]"
            >
                {isDriver ? (
                    <div className="flex gap-3">
                        {status.status === 'awaiting_pickup' || status.status === 'pending' ? (
                            <button
                                onClick={() => handleAction('pickup')}
                                className="btn btn-filled btn-lg flex-1"
                            >
                                <Navigation className="h-5 w-5" aria-hidden="true" />
                                Arrived at Pickup
                            </button>
                        ) : status.status === 'in_progress' ? (
                            <button
                                onClick={() => handleAction('dropoff')}
                                className="btn btn-filled btn-lg flex-1"
                            >
                                <MapPin className="h-5 w-5" aria-hidden="true" />
                                Complete Dropoff
                            </button>
                        ) : (
                            <div className="flex min-h-[50px] w-full items-center justify-center gap-2 rounded-full bg-sys-green/15 text-headline text-green-text">
                                <CircleCheck className="h-5 w-5" aria-hidden="true" />
                                Trip Completed
                            </div>
                        )}
                    </div>
                ) : (
                    <div className="flex min-h-[50px] items-center justify-center gap-2.5 rounded-full bg-fill px-4 text-center">
                        {completed ? (
                            <CircleCheck className="h-5 w-5 text-green-text" aria-hidden="true" />
                        ) : (
                            <span className="h-2 w-2 shrink-0 animate-breathe rounded-full bg-sys-blue" aria-hidden="true" />
                        )}
                        <p className="text-callout font-medium text-label">
                            {status.status === 'awaiting_pickup' || status.status === 'pending' ? 'Your driver is on the way' : status.status === 'in_progress' ? 'Enjoy your LastMile Trip' : 'Trip Completed'}
                        </p>
                    </div>
                )}
            </div>
        </div>
    );
}
