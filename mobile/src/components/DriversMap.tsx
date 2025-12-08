import React, { useEffect, useMemo, useState } from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { Driver } from '../types';
import { GlassCard } from './GlassCard';
import { palette } from '../theme';
import { gateway } from '../services/gateway';
import { MapView, Marker, MAP_PROVIDER_DEFAULT } from './SafeMap';

type LatLng = {
  latitude: number;
  longitude: number;
};

interface DriversMapProps {
  drivers: Driver[];
  userLocation?: LatLng | null;
}

const DEFAULT_REGION: LatLng = {
  latitude: 12.8456, // Electronic City, Bengaluru default center
  longitude: 77.6600,
};

export const DriversMap: React.FC<DriversMapProps> = ({ drivers, userLocation }) => {
  const [driverLocations, setDriverLocations] = useState<Record<string, LatLng>>({});

  useEffect(() => {
    if (!drivers.length) {
      return;
    }

    const unsubscribers = drivers.map((driver, index) => {
      // Seed with a fallback position so markers render immediately
      setDriverLocations((prev) => {
        if (prev[driver.id]) {
          return prev;
        }
        return {
          ...prev,
          [driver.id]: generateFallbackCoordinate(index),
        };
      });

      return gateway.subscribeToLocationUpdates(driver.id, (update) => {
        if (typeof update.latitude !== 'number' || typeof update.longitude !== 'number') {
          return;
        }
        setDriverLocations((prev) => ({
          ...prev,
          [driver.id]: {
            latitude: update.latitude,
            longitude: update.longitude,
          },
        }));
      });
    });

    return () => {
      unsubscribers.forEach((unsubscribe) => {
        try {
          unsubscribe();
        } catch {
          // ignore cleanup errors from stale sockets
        }
      });
    };
  }, [drivers]);

  const markers = useMemo(() => {
    return drivers.map((driver, index) => ({
      id: driver.id,
      name: driver.name,
      etaMinutes: driver.etaMinutes,
      coordinate: driverLocations[driver.id] ?? generateFallbackCoordinate(index),
    }));
  }, [drivers, driverLocations]);

  const region = useMemo(() => {
    const focus = userLocation ?? markers[0]?.coordinate ?? DEFAULT_REGION;
    return {
      latitude: focus.latitude,
      longitude: focus.longitude,
      latitudeDelta: 0.04,
      longitudeDelta: 0.04,
    };
  }, [markers, userLocation]);

  if (!drivers.length) {
    return null;
  }

  return (
    <GlassCard style={styles.card}>
      <View style={styles.headerRow}>
        <Text style={styles.title}>Driver map</Text>
        <Text style={styles.subtitle}>{drivers.length} active</Text>
      </View>
      <View style={styles.mapShell}>
        <MapView provider={MAP_PROVIDER_DEFAULT} style={StyleSheet.absoluteFillObject} region={region}>
          {markers.map((marker) => (
            <Marker
              key={marker.id}
              coordinate={marker.coordinate}
              title={marker.name}
              description={`ETA ${marker.etaMinutes}m`}
            />
          ))}
          {userLocation ? (
            <Marker
              coordinate={userLocation}
              title="You"
              pinColor="#38bdf8"
            />
          ) : null}
        </MapView>
      </View>
    </GlassCard>
  );
};

const generateFallbackCoordinate = (index: number): LatLng => {
  const angle = (index % 6) * (Math.PI / 3);
  const radius = 0.01 + (index % 3) * 0.003;
  return {
    latitude: DEFAULT_REGION.latitude + Math.sin(angle) * radius,
    longitude: DEFAULT_REGION.longitude + Math.cos(angle) * radius,
  };
};

const styles = StyleSheet.create({
  card: {
    marginBottom: 24,
    paddingBottom: 16,
  },
  headerRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  title: {
    fontSize: 18,
    fontWeight: '600',
    color: palette.white,
  },
  subtitle: {
    color: palette.slate,
  },
  mapShell: {
    height: 220,
    borderRadius: 20,
    overflow: 'hidden',
  },
});

export default DriversMap;
