import React from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { Station } from '../types';
import { palette } from '../theme';

interface StationChipProps {
  station: Station;
}

const percentage = (value?: number) => `${Math.round((value ?? 0) * 100)}%`;

export const StationChip: React.FC<StationChipProps> = ({ station }) => (
  <View style={styles.container}>
    <View>
      <Text style={styles.name}>{station.name}</Text>
      <Text style={styles.areas}>{station.nearbyAreas.join(' · ')}</Text>
    </View>
    <Text style={styles.load}>{percentage(station.loadFactor)}</Text>
  </View>
);

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: 8,
  },
  name: {
    color: palette.white,
    fontWeight: '600',
  },
  areas: {
    color: palette.slate,
    marginTop: 4,
  },
  load: {
    color: palette.mint,
    fontWeight: '600',
  },
});
