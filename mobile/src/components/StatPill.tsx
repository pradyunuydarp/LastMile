import React from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { palette } from '../theme';

interface StatPillProps {
  label: string;
  value: string;
}

export const StatPill: React.FC<StatPillProps> = ({ label, value }) => (
  <View style={styles.pill}>
    <Text style={styles.value}>{value}</Text>
    <Text style={styles.label}>{label}</Text>
  </View>
);

const styles = StyleSheet.create({
  pill: {
    paddingVertical: 8,
    paddingHorizontal: 14,
    borderRadius: 999,
    backgroundColor: 'rgba(255,255,255,0.12)',
    marginRight: 12,
  },
  value: {
    color: palette.white,
    fontWeight: '600',
  },
  label: {
    color: palette.slate,
    fontSize: 12,
  },
});
