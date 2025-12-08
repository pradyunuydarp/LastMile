import { MapContainer, Marker, Popup, TileLayer } from 'react-leaflet';
import type { Driver } from '../lib/types';

interface DriversMapProps {
  drivers: Driver[];
  userLocation?: { lat: number; lng: number } | null;
}

const DEFAULT_CENTER: [number, number] = [12.8456, 77.66];

export function DriversMap({ drivers, userLocation }: DriversMapProps) {
  const firstWithCoords = drivers.find((driver) => typeof driver.latitude === 'number' && typeof driver.longitude === 'number');
  const center: [number, number] = userLocation
    ? [userLocation.lat, userLocation.lng]
    : firstWithCoords
    ? [firstWithCoords.latitude as number, firstWithCoords.longitude as number]
    : DEFAULT_CENTER;

  return (
    <MapContainer center={center} zoom={13} className="h-[340px] w-full rounded-2xl overflow-hidden">
      <TileLayer
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
      />
      {drivers.map((driver) => {
        const latitude = driver.latitude ?? firstWithCoords?.latitude ?? DEFAULT_CENTER[0];
        const longitude = driver.longitude ?? firstWithCoords?.longitude ?? DEFAULT_CENTER[1];
        return (
          <Marker key={driver.id} position={[latitude, longitude]}>
            <Popup>
              <div className="text-sm text-slate-800">
                <p className="font-semibold">{driver.name}</p>
                <p>ETA: {driver.etaMinutes} min</p>
                <p>Seats: {driver.seatsAvailable}</p>
              </div>
            </Popup>
          </Marker>
        );
      })}
      {userLocation ? (
        <Marker position={[userLocation.lat, userLocation.lng]}>
          <Popup>You are here</Popup>
        </Marker>
      ) : null}
    </MapContainer>
  );
}
