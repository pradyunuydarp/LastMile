import { useEffect } from 'react';
import { Polyline, TileLayer, useMap } from 'react-leaflet';
import type { LatLngExpression } from 'leaflet';
import { useColorScheme } from './useColorScheme';

const LIGHT_TILES = 'https://{s}.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}{r}.png';
const DARK_TILES = 'https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png';
const ATTRIBUTION =
    '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>';

/** CARTO basemap that follows the system light/dark appearance. */
export function BaseTiles() {
    const scheme = useColorScheme();
    return (
        <TileLayer
            key={scheme}
            url={scheme === 'dark' ? DARK_TILES : LIGHT_TILES}
            attribution={ATTRIBUTION}
            subdomains="abcd"
            maxZoom={20}
        />
    );
}

/** Re-centres the map whenever `center` changes. */
export function MapUpdater({ center }: { center: [number, number] }) {
    const map = useMap();
    useEffect(() => {
        map.setView(center, map.getZoom());
    }, [center, map]);
    return null;
}

/** Apple-Maps-style route line: a light casing under a blue stroke. */
export function RouteLine({ positions }: { positions: LatLngExpression[] }) {
    return (
        <>
            <Polyline positions={positions} pathOptions={{ className: 'lm-route-casing', weight: 9, opacity: 1, lineCap: 'round', lineJoin: 'round' }} />
            <Polyline positions={positions} pathOptions={{ className: 'lm-route', weight: 5, opacity: 1, lineCap: 'round', lineJoin: 'round' }} />
        </>
    );
}
