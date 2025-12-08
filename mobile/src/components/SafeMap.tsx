import React from 'react';
import { StyleProp, StyleSheet, Text, View, ViewStyle } from 'react-native';
import type { MapViewProps, MapMarkerProps, MapPolylineProps } from 'react-native-maps';

type MapsModule = typeof import('react-native-maps');

let Maps: MapsModule | null = null;
try {
  Maps = require('react-native-maps');
} catch (err) {
  console.warn(
    '[maps] react-native-maps native module unavailable; falling back to placeholder map.',
    err,
  );
}

export const MAP_PROVIDER_DEFAULT = Maps?.PROVIDER_DEFAULT;

type FallbackProps = {
  style?: StyleProp<ViewStyle>;
};

const FallbackMap: React.FC<FallbackProps> = ({ style }) => (
  <View style={[styles.fallback, style]}>
    <Text style={styles.fallbackText}>Map preview unavailable in this build.</Text>
  </View>
);

export const MapView: React.FC<MapViewProps> = ({ style, ...rest }) => {
  if (!Maps) {
    return <FallbackMap style={style} />;
  }
  const Component = Maps.default;
  return <Component style={style} {...rest} />;
};

export const Marker: React.FC<MapMarkerProps> = (props) => {
  if (!Maps) {
    return null;
  }
  const Component = Maps.Marker;
  return <Component {...props} />;
};

export const Polyline: React.FC<MapPolylineProps> = (props) => {
  if (!Maps) {
    return null;
  }
  const Component = Maps.Polyline;
  return <Component {...props} />;
};

const styles = StyleSheet.create({
  fallback: {
    width: '100%',
    height: '100%',
    backgroundColor: '#0f172a',
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: 16,
  },
  fallbackText: {
    color: '#94a3b8',
    fontSize: 12,
  },
});

export default MapView;
