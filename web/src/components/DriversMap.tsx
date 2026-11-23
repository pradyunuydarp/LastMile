import { MapContainer, Marker, Popup } from 'react-leaflet';
import type { Driver } from '../lib/types';
import { BaseTiles } from './MapTiles';
import { mapIcons } from './mapIcons';

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
    <div className="map-frame h-[300px] w-full lg:h-full lg:min-h-[340px]">
      <MapContainer center={center} zoom={13} className="h-full w-full">
        <BaseTiles />
        {drivers.map((driver) => {
          const latitude = driver.latitude ?? firstWithCoords?.latitude ?? DEFAULT_CENTER[0];
          const longitude = driver.longitude ?? firstWithCoords?.longitude ?? DEFAULT_CENTER[1];
          return (
            <Marker key={driver.id} position={[latitude, longitude]} icon={mapIcons.driver}>
              <Popup>
                <div>
                  <p className="font-semibold">{driver.name}</p>
                  <p className="text-label-2">ETA {driver.etaMinutes} min · {driver.seatsAvailable} seats</p>
                </div>
              </Popup>
            </Marker>
          );
        })}
        {userLocation ? (
          <Marker position={[userLocation.lat, userLocation.lng]} icon={mapIcons.user}>
            <Popup>You are here</Popup>
          </Marker>
        ) : null}
      </MapContainer>
    </div>
  );
}
