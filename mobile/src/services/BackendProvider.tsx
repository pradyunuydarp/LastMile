import React, { PropsWithChildren, createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { gateway } from './gateway';
import {
  BackendSnapshot,
  BookRidePayload,
  BookRideResponse,
  DriverRoutePayload,
  DriverRouteResponse,
  PickupPoint,
  Rider,
  Trip,
} from '../types';

interface BackendContextValue {
  snapshot: BackendSnapshot | null;
  loading: boolean;
  error?: string;
  refresh: () => Promise<void>;
  matchDriver: (driverId: string, stationId: string) => Promise<Trip | null>;
  bookRide: (payload: BookRidePayload) => Promise<BookRideResponse | null>;
  pickupPoints: PickupPoint[];
  refreshPickupPoints: () => Promise<void>;
  saveDriverRoute: (payload: DriverRoutePayload) => Promise<DriverRouteResponse | null>;
  startDriverTrip: (driverId: string, simulate?: boolean) => Promise<DriverRouteResponse | null>;
  acceptDriverRequest: (driverId: string, riderId: string) => Promise<Trip | null>;
}

const BackendContext = createContext<BackendContextValue | undefined>(undefined);

export const BackendProvider = ({ children }: PropsWithChildren) => {
  const [snapshot, setSnapshot] = useState<BackendSnapshot | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [pickupPoints, setPickupPoints] = useState<PickupPoint[]>([]);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      console.log('[api] refresh snapshot: start');
      const data = await gateway.fetchSnapshot();
      setSnapshot(data);
      setError(undefined);
      console.log('[api] refresh snapshot: success', { at: new Date().toISOString(), trips: data.trips.length });
    } catch (err) {
      console.log('[api] refresh snapshot: error', err);
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const refreshPickupPoints = useCallback(async () => {
    try {
      const data = await gateway.fetchPickupPoints();
      setPickupPoints(data);
    } catch (err) {
      console.warn('[api] pickup fetch failed', err);
    }
  }, []);

  useEffect(() => {
    refreshPickupPoints();
  }, [refreshPickupPoints]);

  const matchDriver = useCallback(
    async (driverId: string, stationId: string) => {
      try {
        console.log('[api] matchDriver: start', { driverId, stationId });
        const trip = await gateway.triggerMatch(driverId, stationId);
        setSnapshot((current) => {
          if (!current) {
            return null;
          }

          const updatedTrips = [trip, ...current.trips];
          return {
            ...current,
            trips: updatedTrips,
            highlightTrip: trip,
            metrics: {
              ...current.metrics,
              pendingMatches: current.metrics.pendingMatches + 1,
            },
            lastUpdated: new Date().toISOString(),
          };
        });
        console.log('[api] matchDriver: success', { tripId: trip.id });
        return trip;
      } catch (err) {
        console.log('[api] matchDriver: error', err);
        setError((err as Error).message);
        return null;
      }
    },
    [],
  );

  const bookRide = useCallback(
    async (payload: BookRidePayload) => {
      try {
        console.log('[api] bookRide: start', payload);
        const response = await gateway.bookRide(payload);
        setSnapshot((current) => {
          if (!current) {
            return current;
          }
          const riders = [response.rider, ...current.riders.filter((rider) => rider.id !== response.rider.id)];
          const trips = response.trip ? [response.trip, ...current.trips] : current.trips;
          return {
            ...current,
            riders,
            trips,
            highlightTrip: response.trip ?? current.highlightTrip,
            lastUpdated: new Date().toISOString(),
          };
        });
        return response;
      } catch (err) {
        console.log('[api] bookRide: error', err);
        setError((err as Error).message);
        return null;
      }
    },
    [],
  );

  const saveDriverRoute = useCallback(
    async (payload: DriverRoutePayload) => {
      try {
        return await gateway.saveDriverRoute(payload);
      } catch (err) {
        setError((err as Error).message);
        return null;
      }
    },
    [],
  );

  const startDriverTrip = useCallback(
    async (driverId: string, simulate?: boolean) => {
      try {
        return await gateway.startDriverTrip(driverId, simulate);
      } catch (err) {
        setError((err as Error).message);
        return null;
      }
    },
    [],
  );

  const acceptDriverRequest = useCallback(
    async (driverId: string, riderId: string) => {
      try {
        const trip = await gateway.acceptDriverRequest(driverId, riderId);
        setSnapshot((current) => {
          if (!current) {
            return current;
          }
          const riders = current.riders.map((rider): Rider =>
            rider.id === trip.riderId ? { ...rider, status: 'matched' as Rider['status'] } : rider,
          );
          return {
            ...current,
            riders,
            trips: [trip, ...current.trips],
            highlightTrip: trip,
            lastUpdated: new Date().toISOString(),
          };
        });
        return trip;
      } catch (err) {
        setError((err as Error).message);
        return null;
      }
    },
    [],
  );

  const value = useMemo<BackendContextValue>(
    () => ({
      snapshot,
      loading,
      error,
      refresh,
      matchDriver,
      bookRide,
      pickupPoints,
      refreshPickupPoints,
      saveDriverRoute,
      startDriverTrip,
      acceptDriverRequest,
    }),
    [snapshot, loading, error, refresh, matchDriver, bookRide, pickupPoints, refreshPickupPoints, saveDriverRoute, startDriverTrip, acceptDriverRequest],
  );

  return <BackendContext.Provider value={value}>{children}</BackendContext.Provider>;
};

export const useBackend = () => {
  const context = useContext(BackendContext);
  if (!context) {
    throw new Error('useBackend must be used within BackendProvider');
  }
  return context;
};
