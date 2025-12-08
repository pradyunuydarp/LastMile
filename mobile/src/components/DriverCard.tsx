import React from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { Driver } from '../types';
import { palette } from '../theme';
import { GlassCard } from './GlassCard';
import { LocationMap } from './LocationMap';

interface DriverCardProps {
  driver: Driver;
}

export const DriverCard: React.FC<DriverCardProps> = ({ driver }) => {
  return (
    <GlassCard>
      <View style={styles.header}>
        <Text style={styles.name}>{driver.name}</Text>
        <Text style={styles.status}>{driver.status.replace('_', ' ')}</Text>
      </View>
      <Text style={styles.car}>{driver.carDetails}</Text>
      <View style={styles.metaRow}>
        <Text style={styles.meta}>ETA · {driver.etaMinutes} min</Text>
        <Text style={styles.meta}>Seats · {driver.seatsAvailable}</Text>
      </View>
      <Text style={styles.destination}>Stations → {driver.route.destination}</Text>
      {driver.route.pickupPoints?.length ? (
        <Text style={styles.pickups}>
          Stops: {driver.route.pickupPoints.map((point) => point.name).join(', ')}
        </Text>
      ) : null}
      <LocationMap
        driverId={driver.id}
        initialLat={driver.latitude || 12.8456}
        initialLong={driver.longitude || 77.66}
      />
    </GlassCard>
  );
};

const styles = StyleSheet.create({
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  name: {
    color: palette.white,
    fontSize: 18,
    fontWeight: '600',
  },
  status: {
    color: palette.mint,
    fontSize: 12,
    textTransform: 'capitalize',
  },
  car: {
    color: palette.slate,
    marginTop: 4,
  },
  metaRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginTop: 8,
  },
  meta: {
    color: palette.white,
    fontSize: 14,
  },
  destination: {
    color: palette.gold,
    marginTop: 12,
    fontWeight: '600',
  },
  pickups: {
    color: palette.slate,
    marginTop: 4,
    fontSize: 12,
  },
});
