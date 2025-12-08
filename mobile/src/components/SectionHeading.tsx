import React from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { palette } from '../theme';

interface SectionHeadingProps {
  label: string;
  subtitle?: string;
  actionSlot?: React.ReactNode;
}

export const SectionHeading: React.FC<SectionHeadingProps> = ({ label, subtitle, actionSlot }) => (
  <View style={styles.container}>
    <View>
      <Text style={styles.label}>{label}</Text>
      {subtitle ? <Text style={styles.subtitle}>{subtitle}</Text> : null}
    </View>
    {actionSlot}
  </View>
);

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  label: {
    fontSize: 18,
    color: palette.white,
    fontWeight: '600',
  },
  subtitle: {
    color: palette.slate,
    fontSize: 13,
    marginTop: 2,
  },
});
