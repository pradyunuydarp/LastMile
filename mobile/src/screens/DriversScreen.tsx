import React, { useEffect, useMemo, useState } from 'react';
import { FlatList, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useBackend } from '../services/BackendProvider';
import { palette } from '../theme';
import { DriverCard } from '../components/DriverCard';
import DriversMap from '../components/DriversMap';
import * as Location from 'expo-location';

const DriversScreen = () => {
  const { snapshot } = useBackend();
  const insets = useSafeAreaInsets();
  const [userLocation, setUserLocation] = useState<Location.LocationObject | null>(null);
  const [locationError, setLocationError] = useState<string | null>(null);

  const drivers = snapshot?.drivers ?? [];

  useEffect(() => {
    let mounted = true;
    (async () => {
      try {
        const { status } = await Location.requestForegroundPermissionsAsync();
        if (status !== 'granted') {
          if (mounted) {
            setLocationError('Location permission denied');
          }
          return;
        }
        const position = await Location.getCurrentPositionAsync({});
        if (mounted) {
          setUserLocation(position);
        }
      } catch (err) {
        if (mounted) {
          setLocationError('Unable to fetch your location');
        }
      }
    })();

    return () => {
      mounted = false;
    };
  }, []);

  const header = useMemo(() => (
    <View>
      <Text style={styles.heading}>Drivers en route</Text>
      <DriversMap
        drivers={drivers}
        userLocation={userLocation ? { latitude: userLocation.coords.latitude, longitude: userLocation.coords.longitude } : null}
      />
      {locationError ? <Text style={styles.errorText}>{locationError}</Text> : null}
    </View>
  ), [drivers, userLocation, locationError]);

  return (
    <View style={styles.container}>
      <FlatList
        data={drivers}
        keyExtractor={(driver) => driver.id}
        renderItem={({ item }) => <DriverCard driver={item} />}
        ListHeaderComponent={header}
        contentContainerStyle={[styles.listContent, { paddingTop: insets.top + 8 }]}
      />
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: palette.navy,
  },
  heading: {
    fontSize: 28,
    fontWeight: '700',
    color: palette.white,
    marginBottom: 16,
  },
  errorText: {
    color: palette.rose,
    marginTop: 8,
  },
  listContent: {
    padding: 20,
    paddingBottom: 80,
  },
});

export default DriversScreen;
