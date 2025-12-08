import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { io, Socket } from 'socket.io-client';
import type { DriverRequestsResponse, PickupPoint, Station, Trip } from '../lib/types';
import { useAuth } from './AuthContext';

type DriverOffer = {
    riderId: string;
    riderName: string;
    destination?: string;
    pickupId?: string;
    pickupName?: string;
    pickup?: PickupPoint;
    station?: Station;
    attempt: number;
    total: number;
};

type ApprovalRequest = {
    tripId: string;
    driverId: string;
    driverName?: string;
    pickup?: PickupPoint;
    station?: Station;
};

export type TripRoomState = {
    tripId: string;
    status: string;
    driverId: string;
    riderId: string;
    pickup?: PickupPoint;
    station?: Station;
    latitude?: number;
    longitude?: number;
    updatedAt?: string;
    trip?: Trip;
};

export type RiderStatus = {
    tripId?: string;
    status: string;
    riderId: string;
    driverId?: string;
    pickup?: PickupPoint;
    station?: Station;
    latitude?: number;
    longitude?: number;
    recordedAt: string;
    description?: string;
};

type RealtimeContextValue = {
    ready: boolean;
    offers: DriverOffer[];
    queueSummary: DriverRequestsResponse | null;
    rooms: TripRoomState[];
    riderStatus: RiderStatus | null;
    approvalRequest: ApprovalRequest | null;
    respondToOffer: (riderId: string, accept: boolean) => void;
    completeTrip: (tripId: string) => void;
    respondToApproval: (tripId: string, accept: boolean) => void;
};

const RealtimeContext = createContext<RealtimeContextValue | undefined>(undefined);

const baseGatewayUrl = (import.meta.env.VITE_GATEWAY_URL as string | undefined)?.replace(/\/$/, '') ?? window.location.origin;
const isNgrok = baseGatewayUrl.includes('ngrok');

export const RealtimeProvider = ({ children }: { children: React.ReactNode }) => {
    const { user, role } = useAuth();
    const [socket, setSocket] = useState<Socket | null>(null);
    const [ready, setReady] = useState(false);
    const [queueSummary, setQueueSummary] = useState<DriverRequestsResponse | null>(null);
    const offersRef = useRef<Record<string, DriverOffer>>({});
    const roomsRef = useRef<Record<string, TripRoomState>>({});
    const [offersVersion, setOffersVersion] = useState(0);
    const [roomsVersion, setRoomsVersion] = useState(0);
    const [riderStatus, setRiderStatus] = useState<RiderStatus | null>(null);
    const [approvalRequest, setApprovalRequest] = useState<ApprovalRequest | null>(null);

    useEffect(() => {
        if (!user?.id || !role) {
            setQueueSummary(null);
            setReady(false);
            if (socket) {
                socket.disconnect();
                setSocket(null);
            }
            return;
        }

        const client = io(baseGatewayUrl, {
            path: '/socket.io/',
            transports: ['websocket'],
            query: isNgrok ? { 'ngrok-skip-browser-warning': 'true' } : undefined,
        });
        setSocket(client);

        const name =
            (user.user_metadata?.full_name as string | undefined) ??
            (user.user_metadata?.name as string | undefined) ??
            user.email ??
            'LastMile User';

        client.on('connect', () => {
            client.emit('session:init', { role, userId: user.id, name });
        });
        client.on('session:ack', () => setReady(true));
        client.on('disconnect', () => {
            setReady(false);
            setQueueSummary(null);
            offersRef.current = {};
            roomsRef.current = {};
            setOffersVersion((n) => n + 1);
            setRoomsVersion((n) => n + 1);
        });
        client.on('driver:rider-queue', (payload: DriverRequestsResponse) => {
            setQueueSummary(payload);
        });
        client.on('driver:rider-offer', (payload: any) => {
            const rider = payload?.rider ?? {};
            const offer: DriverOffer = {
                riderId: rider.id,
                riderName: rider.name ?? 'Rider',
                destination: rider.destination,
                pickupId: rider.pickupId,
                pickupName: rider.pickupName,
                pickup: payload.pickup,
                station: payload.station,
                attempt: payload.attempt,
                total: payload.total,
            };
            offersRef.current = { ...offersRef.current, [offer.riderId]: offer };
            setOffersVersion((n) => n + 1);
        });
        client.on('trip:room-created', (payload: any) => {
            const tripId = payload?.tripId ?? payload?.trip?.id;
            if (!tripId) {
                return;
            }
            const riderId = payload?.riderId ?? payload?.trip?.riderId;
            if (riderId) {
                const nextOffers = { ...offersRef.current };
                delete nextOffers[riderId];
                offersRef.current = nextOffers;
                setOffersVersion((n) => n + 1);
            }
            roomsRef.current = {
                ...roomsRef.current,
                [tripId]: {
                    tripId,
                    status: payload?.status ?? 'awaiting_pickup',
                    driverId: payload?.driverId,
                    riderId,
                    pickup: payload?.pickup,
                    station: payload?.station,
                    latitude: payload?.latitude,
                    longitude: payload?.longitude,
                    updatedAt: payload?.recordedAt,
                    trip: payload?.trip,
                },
            };
            setRoomsVersion((n) => n + 1);
        });
        client.on('trip:location', (payload: any) => {
            const tripId = payload?.tripId;
            if (!tripId) {
                return;
            }
            const existing = roomsRef.current[tripId];
            roomsRef.current = {
                ...roomsRef.current,
                [tripId]: {
                    ...(existing ?? {
                        tripId,
                        driverId: payload?.driverId,
                        riderId: payload?.riderId,
                    }),
                    status: payload?.status ?? existing?.status ?? 'in_progress',
                    pickup: payload?.pickup ?? existing?.pickup,
                    station: payload?.station ?? existing?.station,
                    latitude: payload?.latitude,
                    longitude: payload?.longitude,
                    updatedAt: payload?.recordedAt,
                },
            };
            setRoomsVersion((n) => n + 1);
        });
        client.on('trip:status', (payload: any) => {
            const tripId = payload?.tripId;
            if (!tripId) {
                return;
            }
            if (payload?.status === 'completed') {
                const nextRooms = { ...roomsRef.current };
                delete nextRooms[tripId];
                roomsRef.current = nextRooms;
            } else {
                roomsRef.current = {
                    ...roomsRef.current,
                    [tripId]: {
                        ...(roomsRef.current[tripId] ?? {
                            tripId,
                            driverId: payload?.driverId,
                            riderId: payload?.riderId,
                        }),
                        status: payload?.status,
                        pickup: payload?.pickup ?? roomsRef.current[tripId]?.pickup,
                        station: payload?.station ?? roomsRef.current[tripId]?.station,
                        updatedAt: payload?.recordedAt,
                    },
                };
            }
            setRoomsVersion((n) => n + 1);
            setRoomsVersion((n) => n + 1);
        });
        client.on('rider:status', (payload: RiderStatus) => {
            setRiderStatus(payload);
        });
        client.on('driver:rider-error', (payload: any) => {
            console.warn('[realtime] driver error', payload);
        });
        client.on('rider:approval-request', (payload: any) => {
            if (!payload?.tripId) {
                return;
            }
            setApprovalRequest({
                tripId: payload.tripId,
                driverId: payload.driverId,
                driverName: payload.driverName,
                pickup: payload.pickup,
                station: payload.station,
            });
        });

        return () => {
            client.disconnect();
            setSocket(null);
        };
    }, [user?.id, role]);

    const respondToOffer = useCallback(
        (riderId: string, accept: boolean) => {
            if (!socket) {
                return;
            }
            socket.emit('driver:rider-response', { riderId, accept });
            const nextOffers = { ...offersRef.current };
            delete nextOffers[riderId];
            offersRef.current = nextOffers;
            setOffersVersion((n) => n + 1);
        },
        [socket],
    );

    const completeTrip = useCallback(
        (tripId: string) => {
            socket?.emit('trip:complete', tripId);
        },
        [socket],
    );

    const respondToApproval = useCallback(
        (tripId: string, accept: boolean) => {
            if (!tripId) {
                return;
            }
            socket?.emit('rider:approval-response', { tripId, accept });
            setApprovalRequest(null);
        },
        [socket],
    );

    useEffect(() => {
        if (!approvalRequest || !riderStatus?.tripId) {
            return;
        }
        if (approvalRequest.tripId === riderStatus.tripId && riderStatus.status !== 'awaiting_rider') {
            setApprovalRequest(null);
        }
    }, [approvalRequest, riderStatus]);

    const offers = useMemo(() => Object.values(offersRef.current), [offersVersion, ready]);
    const rooms = useMemo(() => Object.values(roomsRef.current), [roomsVersion, ready]);

    const value = useMemo<RealtimeContextValue>(
        () => ({
            ready,
            offers,
            queueSummary,
            rooms,
            riderStatus,
            approvalRequest,
            respondToOffer,
            completeTrip,
            respondToApproval,
        }),
        [ready, offers, queueSummary, rooms, riderStatus, approvalRequest, respondToOffer, completeTrip, respondToApproval],
    );

    return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
};

export const useRealtime = () => {
    const context = useContext(RealtimeContext);
    if (!context) {
        throw new Error('useRealtime must be used within RealtimeProvider');
    }
    return context;
};
