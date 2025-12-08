import { BackendSnapshot, Driver, PickupPoint, Rider, Station, Trip } from '../types';

const pickupPoints: PickupPoint[] = [
  {
    id: 'pickup-fidi-gate',
    name: 'Embarcadero Plaza',
    stationId: 'station-central',
    stationName: 'Central Metro',
    latitude: 37.7953,
    longitude: -122.395,
  },
  {
    id: 'pickup-mission-dolores',
    name: 'Mission Dolores Park',
    stationId: 'station-mission',
    stationName: 'Mission Square',
    latitude: 37.7596,
    longitude: -122.4269,
  },
  {
    id: 'pickup-sunset-park',
    name: 'Sunset Reservoir',
    stationId: 'station-sunset',
    stationName: 'Sunset Terminal',
    latitude: 37.7436,
    longitude: -122.4862,
  },
];

const stations: Station[] = [
  {
    id: 'station-central',
    name: 'Central Metro',
    nearbyAreas: ['Financial District', 'Embarcadero', 'SoMa'],
    loadFactor: 0.72,
  },
  {
    id: 'station-mission',
    name: 'Mission Square',
    nearbyAreas: ['Mission District', 'Bernal Heights'],
    loadFactor: 0.58,
  },
  {
    id: 'station-sunset',
    name: 'Sunset Terminal',
    nearbyAreas: ['Outer Sunset', 'Golden Gate Heights'],
    loadFactor: 0.41,
  },
];

const drivers: Driver[] = [
  {
    id: 'driver-ava',
    name: 'Ava Chen',
    carDetails: 'Polestar 2 • EV',
    seatsAvailable: 2,
    etaMinutes: 4,
    status: 'en_route',
    route: {
      id: 'route-ava',
      destination: 'Financial District',
      targetStationIds: ['station-central'],
      pickupPoints: [pickupPoints[0]],
    },
  },
  {
    id: 'driver-noah',
    name: 'Noah Patel',
    carDetails: 'Model Y • AWD',
    seatsAvailable: 3,
    etaMinutes: 7,
    status: 'arriving',
    route: {
      id: 'route-noah',
      destination: 'Mission District',
      targetStationIds: ['station-mission'],
      pickupPoints: [pickupPoints[1]],
    },
  },
  {
    id: 'driver-lina',
    name: 'Lina Reyes',
    carDetails: 'ID.4 • Comfort',
    seatsAvailable: 1,
    etaMinutes: 2,
    status: 'boarding',
    route: {
      id: 'route-lina',
      destination: 'Golden Gate Heights',
      targetStationIds: ['station-sunset'],
      pickupPoints: [pickupPoints[2]],
    },
  },
];

const riders: Rider[] = [
  {
    id: 'rider-julian',
    name: 'Julian Wright',
    destination: 'Financial District',
    arrivalTime: new Date(Date.now() + 5 * 60 * 1000).toISOString(),
    stationId: 'station-central',
    status: 'waiting',
    pickupPointId: pickupPoints[0].id,
    pickup: pickupPoints[0],
  },
  {
    id: 'rider-mei',
    name: 'Mei Huang',
    destination: 'Mission District',
    arrivalTime: new Date(Date.now() + 9 * 60 * 1000).toISOString(),
    stationId: 'station-mission',
    status: 'waiting',
    pickupPointId: pickupPoints[1].id,
    pickup: pickupPoints[1],
  },
  {
    id: 'rider-rohan',
    name: 'Rohan Iyer',
    destination: 'Golden Gate Heights',
    arrivalTime: new Date(Date.now() + 12 * 60 * 1000).toISOString(),
    stationId: 'station-sunset',
    status: 'matched',
    pickupPointId: pickupPoints[2].id,
    pickup: pickupPoints[2],
  },
];

const trips: Trip[] = [
  {
    id: 'trip-1',
    driverId: 'driver-ava',
    riderId: 'rider-julian',
    stationId: 'station-central',
    destination: 'Financial District',
    etaMinutes: 5,
    status: 'pending',
    createdAt: new Date(Date.now() - 2 * 60 * 1000).toISOString(),
  },
  {
    id: 'trip-2',
    driverId: 'driver-lina',
    riderId: 'rider-rohan',
    stationId: 'station-sunset',
    destination: 'Golden Gate Heights',
    etaMinutes: 2,
    status: 'in_progress',
    createdAt: new Date(Date.now() - 10 * 60 * 1000).toISOString(),
  },
];

export const mockSnapshot: BackendSnapshot = {
  drivers,
  riders,
  trips,
  stations,
  metrics: {
    pendingMatches: 2,
    ridersWaiting: riders.filter((r) => r.status === 'waiting').length,
    seatsOpen: drivers.reduce((acc, d) => acc + d.seatsAvailable, 0),
    avgWaitMinutes: 6,
    version: 'mock',
  },
  highlightTrip: trips[0],
  lastUpdated: new Date().toISOString(),
};

export const mockSnapshotWithTrip = (trip: Trip): BackendSnapshot => ({
  ...mockSnapshot,
  trips: [trip, ...mockSnapshot.trips],
  highlightTrip: trip,
  metrics: {
    ...mockSnapshot.metrics,
    pendingMatches: mockSnapshot.metrics.pendingMatches + 1,
  },
  lastUpdated: new Date().toISOString(),
});

export const createMockTrip = (driverId: string, stationId: string): Trip => {
  const driver = drivers.find((d) => d.id === driverId) ?? drivers[0];
  const candidateRider =
    riders.find((r) => r.stationId === stationId && r.status !== 'picked_up') ??
    riders[0];

  return {
    id: `trip-${Date.now()}`,
    driverId: driver.id,
    riderId: candidateRider.id,
    stationId,
    destination: candidateRider.destination,
    pickupPointId: candidateRider.pickupPointId,
    pickup: candidateRider.pickup,
    etaMinutes: driver.etaMinutes,
    status: 'pending',
    createdAt: new Date().toISOString(),
  };
};
