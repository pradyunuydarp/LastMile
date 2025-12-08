import React from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { Trip, Driver, Rider, Station } from '../types';
import { palette } from '../theme';
import { GlassCard } from './GlassCard';

interface TripCardProps {
  trip: Trip;
  driver?: Driver;
  rider?: Rider;
  station?: Station;
}

export const TripCard: React.FC<TripCardProps> = ({ trip, driver, rider, station }) => (
  <GlassCard>
    <Text style={styles.status}>{trip.status.replace('_', ' ')}</Text>
    <Text style={styles.title}>
      {driver?.name ?? trip.driverId} ⇄ {rider?.name ?? trip.riderId}
    </Text>
    {station ? <Text style={styles.subtitle}>Station · {station.name}</Text> : null}
    {trip.destination ? <Text style={styles.subtitle}>Dest · {trip.destination}</Text> : null}
    <View style={styles.row}>
      <Text style={styles.meta}>ETA · {trip.etaMinutes ?? driver?.etaMinutes ?? '--'} min</Text>
      <Text style={styles.meta}>Seats left · {driver?.seatsAvailable ?? '—'}</Text>
    </View>
  </GlassCard>
);

const styles = StyleSheet.create({
  status: {
    color: palette.rose,
    fontSize: 12,
    textTransform: 'capitalize',
  },
  title: {
    color: palette.white,
    fontSize: 20,
    fontWeight: '700',
    marginTop: 6,
  },
  subtitle: {
    color: palette.slate,
    marginTop: 4,
  },
  row: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginTop: 14,
  },
  meta: {
    color: palette.white,
    fontWeight: '500',
  },
});
