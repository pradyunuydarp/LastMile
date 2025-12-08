import React from 'react';
import { ActivityIndicator, Pressable, StyleSheet, Text } from 'react-native';
import { LinearGradient } from 'expo-linear-gradient';
import { palette } from '../theme';

interface LiquidButtonProps {
  label: string;
  onPress: () => void;
  loading?: boolean;
}

export const LiquidButton: React.FC<LiquidButtonProps> = ({ label, onPress, loading }) => (
  <Pressable onPress={onPress} disabled={loading} style={({ pressed }) => [{ opacity: pressed ? 0.8 : 1 }] }>
    <LinearGradient colors={['#6ef3d6', '#4facfe']} start={{ x: 0, y: 0 }} end={{ x: 1, y: 1 }} style={styles.gradient}>
      {loading ? <ActivityIndicator color={palette.navy} /> : <Text style={styles.label}>{label}</Text>}
    </LinearGradient>
  </Pressable>
);

const styles = StyleSheet.create({
  gradient: {
    borderRadius: 999,
    paddingVertical: 12,
    paddingHorizontal: 24,
    alignItems: 'center',
  },
  label: {
    fontWeight: '600',
    letterSpacing: 0.5,
    color: palette.navy,
  },
});
