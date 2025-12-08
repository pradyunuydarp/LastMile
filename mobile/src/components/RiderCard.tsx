import React from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { Rider } from '../types';
import { palette } from '../theme';
import { GlassCard } from './GlassCard';

interface RiderCardProps {
  rider: Rider;
}

const formatTime = (iso: string) =>
  new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });

export const RiderCard: React.FC<RiderCardProps> = ({ rider }) => (
  <GlassCard>
    <View style={styles.row}>
      <View>
        <Text style={styles.name}>{rider.name}</Text>
        <Text style={styles.destination}>Pickup · {rider.pickup?.name ?? rider.destination}</Text>
      </View>
      <Text style={styles.status}>{rider.status}</Text>
    </View>
    <View style={styles.metaRow}>
      <Text style={styles.meta}>Arrival · {formatTime(rider.arrivalTime)}</Text>
      <Text style={styles.meta}>Drop · {rider.pickup?.stationName ?? rider.stationId.replace('station-', '')}</Text>
    </View>
  </GlassCard>
);

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  name: {
    color: palette.white,
    fontSize: 18,
    fontWeight: '600',
  },
  destination: {
    color: palette.slate,
    marginTop: 4,
  },
  status: {
    color: palette.rose,
    textTransform: 'capitalize',
  },
  metaRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginTop: 12,
  },
  meta: {
    color: palette.white,
  },
});
