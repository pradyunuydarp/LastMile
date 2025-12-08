export type RootStackParamList = {
  Dashboard: undefined;
  Drivers: undefined;
  Riders: undefined;
  DriverRequests: undefined;
  Trips: undefined;
  Profile: undefined;
  SignIn: undefined;
  SignUp: undefined;
  ForgotPassword: undefined;
};

export type TripStatus = 'pending' | 'accepted' | 'in_progress' | 'completed' | 'cancelled';

export type Station = {
  id: string;
  name: string;
  nearbyAreas: string[];
  loadFactor?: number; // percentage 0-1 representing how busy the station nearby area is
  latitude?: number;
  longitude?: number;
};

export type PickupPoint = {
  id: string;
  name: string;
  stationId: string;
  stationName: string;
  latitude: number;
  longitude: number;
};

export type Route = {
  id: string;
  targetStationIds: string[];
  destination: string;
  pickupPoints?: PickupPoint[];
};

export type Driver = {
  id: string;
  name: string;
  carDetails: string;
  seatsAvailable: number;
  etaMinutes: number;
  status: 'en_route' | 'arriving' | 'boarding' | 'active';
  route: Route;
  latitude?: number;
  longitude?: number;
};

export type Rider = {
  id: string;
  name: string;
  destination: string;
  arrivalTime: string; // ISO timestamp
  stationId: string;
  status: 'waiting' | 'matched' | 'picked_up';
  pickupPointId?: string;
  pickup?: PickupPoint;
};

export type Trip = {
  id: string;
  driverId: string;
  riderId: string;
  stationId?: string;
  destination?: string;
  pickupPointId?: string;
  pickup?: PickupPoint;
  etaMinutes?: number;
  status: TripStatus;
  createdAt?: string;
  completedAt?: string;
  roomId?: string;
};

export type BackendMetrics = {
  pendingMatches: number;
  ridersWaiting: number;
  seatsOpen: number;
  avgWaitMinutes: number;
  version: string;
};

export type BackendSnapshot = {
  drivers: Driver[];
  riders: Rider[];
  trips: Trip[];
  stations: Station[];
  metrics: BackendMetrics;
  highlightTrip?: Trip;
  lastUpdated: string;
};

export type DriverSummary = {
  id: string;
  name: string;
  seatsAvailable: number;
  nextStop: string;
};

export type DriverRequest = {
  id: string;
  name: string;
  destination: string;
  arrivalTime: string;
  station: Station;
  pickup?: PickupPoint;
  status: string;
  distanceMeters: number;
};

export type DriverRequestsResponse = {
  driver: DriverSummary;
  requests: DriverRequest[];
  generatedAt: string;
};

export type DriverAttempt = {
  driverId: string;
  driverName: string;
  distanceMeters: number;
  accepted: boolean;
  reason?: string;
};

export type BookRidePayload = {
  command: string;
  riderId?: string;
  name?: string;
  address?: string;
  destination?: string;
  stationId?: string;
  pickupPointId?: string;
};

export type BookRideResponse = {
  status: string;
  message: string;
  rider: Rider;
  station: Station;
  pickup?: PickupPoint;
  requestedDestination: string;
  attempts: DriverAttempt[];
  trip?: Trip;
};

export type DriverRoutePayload = {
  driverId: string;
  name: string;
  carDetails: string;
  pickupPointIds: string[];
  seats: number;
};

export type DriverRouteResponse = {
  driverId: string;
  pickupPoints: PickupPoint[];
  seatsTotal: number;
  seatsAvailable: number;
  targetStations: string[];
  destination: string;
};
