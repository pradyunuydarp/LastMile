import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { io, Socket } from 'socket.io-client';
import { useAuth } from './AuthContext';
import type { DriverRequestsResponse, PickupPoint, Station, Trip } from '../types';

type DriverOffer = {
  riderId: string;
  riderName: string;
  pickup?: PickupPoint;
  station?: Station;
  destination?: string;
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

export type TripRoom = {
  tripId: string;
  driverId: string;
  riderId: string;
  status: string;
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
  queue: DriverRequestsResponse | null;
  rooms: TripRoom[];
  riderStatus: RiderStatus | null;
  approvalRequest: ApprovalRequest | null;
  respondToOffer: (riderId: string, accept: boolean) => void;
  completeTrip: (tripId: string) => void;
  respondToApproval: (tripId: string, accept: boolean) => void;
};

const RealtimeContext = createContext<RealtimeContextValue | undefined>(undefined);

const baseUrl = (process.env.EXPO_PUBLIC_API_URL ?? 'http://localhost:8082').replace(/\/$/, '');
const isNgrok = baseUrl.includes('ngrok');

export const RealtimeProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { user, role } = useAuth();
  const [socket, setSocket] = useState<Socket | null>(null);
  const [ready, setReady] = useState(false);
  const [queue, setQueue] = useState<DriverRequestsResponse | null>(null);
  const [offers, setOffers] = useState<DriverOffer[]>([]);
  const [rooms, setRooms] = useState<TripRoom[]>([]);
  const [riderStatus, setRiderStatus] = useState<RiderStatus | null>(null);
  const [approvalRequest, setApprovalRequest] = useState<ApprovalRequest | null>(null);

  useEffect(() => {
    if (!user?.id || !role) {
      setQueue(null);
      setOffers([]);
      setRooms([]);
      setReady(false);
      setApprovalRequest(null);
      if (socket) {
        socket.disconnect();
        setSocket(null);
      }
      return;
    }

    const client = io(baseUrl, {
      path: '/socket.io/',
      transports: ['websocket', 'polling'],
      upgrade: true,
      query: isNgrok ? { 'ngrok-skip-browser-warning': 'true' } : undefined,
    });
    setSocket(client);

    client.on('connect', () => {
      client.emit('session:init', {
        role,
        userId: user.id,
        name: user.user_metadata?.full_name ?? user.name ?? user.email ?? 'LastMile User',
      });
    });
    client.on('session:ack', () => setReady(true));
    client.on('disconnect', () => {
      setReady(false);
      setOffers([]);
      setRooms([]);
      setQueue(null);
      setApprovalRequest(null);
    });
    client.on('driver:rider-queue', (payload: DriverRequestsResponse) => {
      setQueue(payload);
    });
    client.on('driver:rider-offer', (payload: any) => {
      const rider = payload?.rider ?? {};
      const offer: DriverOffer = {
        riderId: rider.id,
        riderName: rider.name ?? 'Rider',
        pickup: payload?.pickup,
        station: payload?.station,
        destination: rider.destination,
        attempt: payload?.attempt ?? 1,
        total: payload?.total ?? 1,
      };
      setOffers((current) => {
        const filtered = current.filter((item) => item.riderId !== offer.riderId);
        return [...filtered, offer];
      });
    });
    client.on('trip:room-created', (payload: any) => {
      const tripId = payload?.tripId ?? payload?.trip?.id;
      if (!tripId) {
        return;
      }
      setOffers((current) => current.filter((offer) => offer.riderId !== payload?.riderId));
      setRooms((current) => {
        const filtered = current.filter((room) => room.tripId !== tripId);
        return [
          ...filtered,
          {
            tripId,
            driverId: payload?.driverId,
            riderId: payload?.riderId,
            status: payload?.status ?? 'awaiting_pickup',
            pickup: payload?.pickup,
            station: payload?.station,
            latitude: payload?.latitude,
            longitude: payload?.longitude,
            updatedAt: payload?.recordedAt,
            trip: payload?.trip,
          },
        ];
      });
    });
    const updateRoom = (payload: any) => {
      const tripId = payload?.tripId;
      if (!tripId) {
        return;
      }
      setRooms((current) => {
        const next = current.filter((room) => room.tripId !== tripId);
        if (payload?.status === 'completed') {
          return next;
        }
        return [
          ...next,
          {
            tripId,
            driverId: payload?.driverId,
            riderId: payload?.riderId,
            status: payload?.status ?? 'in_progress',
            pickup: payload?.pickup,
            station: payload?.station,
            latitude: payload?.latitude,
            longitude: payload?.longitude,
            updatedAt: payload?.recordedAt,
            trip: payload?.trip ?? next.find((room) => room.tripId === tripId)?.trip,
          },
        ];
      });
    };
    client.on('trip:location', updateRoom);
    client.on('trip:status', updateRoom);
    client.on('rider:status', (payload: RiderStatus) => setRiderStatus(payload));
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
      socket?.emit('driver:rider-response', { riderId, accept });
      if (!accept) {
        setOffers((current) => current.filter((offer) => offer.riderId !== riderId));
      }
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
    if (riderStatus.tripId === approvalRequest.tripId && riderStatus.status !== 'awaiting_rider') {
      setApprovalRequest(null);
    }
  }, [approvalRequest, riderStatus]);

  const value = useMemo(
    () => ({
      ready,
      offers,
      queue,
      rooms,
      riderStatus,
      approvalRequest,
      respondToOffer,
      completeTrip,
      respondToApproval,
    }),
    [approvalRequest, completeTrip, offers, queue, ready, respondToApproval, respondToOffer, riderStatus, rooms],
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
