import React from 'react';
import { View, ViewProps, StyleSheet } from 'react-native';
import { BlurView } from 'expo-blur';
import { LinearGradient, type LinearGradientProps } from 'expo-linear-gradient';
import { glass } from '../theme';

interface GlassCardProps extends ViewProps {
  accent?: LinearGradientProps['colors'];
  children: React.ReactNode;
}

const DEFAULT_GRADIENT: LinearGradientProps['colors'] = [
  'rgba(255,255,255,0.08)',
  'rgba(15,23,42,0.3)',
] as const;

export const GlassCard: React.FC<GlassCardProps> = ({ accent, style, children, ...rest }) => {
  return (
    <LinearGradient colors={accent ?? DEFAULT_GRADIENT} style={styles.gradient}>
      <BlurView intensity={80} tint="light" style={styles.blur}>
        <View style={[styles.card, style]} {...rest}>
          {children}
        </View>
      </BlurView>
    </LinearGradient>
  );
};

const styles = StyleSheet.create({
  gradient: {
    borderRadius: 24,
    overflow: 'hidden',
    marginBottom: 16,
  },
  blur: {
    padding: 1,
  },
  card: {
    borderRadius: 24,
    padding: 18,
    borderColor: glass.border,
    borderWidth: StyleSheet.hairlineWidth,
    backgroundColor: 'rgba(255,255,255,0.08)',
    shadowColor: glass.shadow,
    shadowOffset: { width: 0, height: 16 },
    shadowOpacity: 0.35,
    shadowRadius: 32,
  },
});
