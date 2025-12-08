import React, { useMemo, useState } from 'react';
import { RefreshControl, ScrollView, StyleSheet, Text, View, TouchableOpacity } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { RootStackParamList } from '../types';
import { useBackend } from '../services/BackendProvider';
import { useAuth } from '../context/AuthContext';
import { palette } from '../theme';
import { GlassCard } from '../components/GlassCard';
import { SectionHeading } from '../components/SectionHeading';
import { LiquidButton } from '../components/LiquidButton';
import { StatPill } from '../components/StatPill';
import { TripCard } from '../components/TripCard';
import { StationChip } from '../components/StationChip';
import { LocationMap } from '../components/LocationMap';

import * as Location from 'expo-location';
import { MapView, Marker, MAP_PROVIDER_DEFAULT } from '../components/SafeMap';

const DEFAULT_REGION = {
  latitude: 12.8456,
  longitude: 77.66,
  latitudeDelta: 0.04,
  longitudeDelta: 0.04,
};

const buildRegion = (latitude: number, longitude: number, delta = 0.035) => ({
  latitude,
  longitude,
  latitudeDelta: delta,
  longitudeDelta: delta,
});

const formatArrival = (iso: string) =>
  new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });

const DashboardScreen = () => {
  const { snapshot, loading, refresh, matchDriver } = useBackend();
  const { user, role } = useAuth();
  const [matchLoading, setMatchLoading] = useState(false);
  const [userLocation, setUserLocation] = useState<Location.LocationObject | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const insets = useSafeAreaInsets();
  const navigation = useNavigation<StackNavigationProp<RootStackParamList>>();

  React.useEffect(() => {
    (async () => {
      let { status } = await Location.requestForegroundPermissionsAsync();
      if (status !== 'granted') {
        setErrorMsg('Permission to access location was denied');
        return;
      }

      let location = await Location.getCurrentPositionAsync({});
      setUserLocation(location);
    })();
  }, []);

  const isDriver = role === 'driver';
  const drivers = snapshot?.drivers ?? [];
  const riders = snapshot?.riders ?? [];
  const highlight = snapshot?.highlightTrip;
  const userLatLng = useMemo(() => {
    if (!userLocation) {
      return null;
    }
    return {
      latitude: userLocation.coords.latitude,
      longitude: userLocation.coords.longitude,
    };
  }, [userLocation]);

  const driverForHighlight = useMemo(
    () => drivers.find((driver) => driver.id === highlight?.driverId),
    [drivers, highlight?.driverId],
  );

  const riderForHighlight = useMemo(
    () => riders.find((rider) => rider.id === highlight?.riderId),
    [riders, highlight?.riderId],
  );

  const stationForHighlight = useMemo(() => {
    if (!driverForHighlight) return undefined;
    const stationId = driverForHighlight.route.targetStationIds[0];
    return snapshot?.stations.find((station) => station.id === stationId);
  }, [snapshot, driverForHighlight]);

  const driverProfile = useMemo(() => {
    if (!isDriver) {
      return null;
    }
    return drivers.find((driver) => driver.id === user?.id) ?? null;
  }, [drivers, isDriver, user?.id]);

  const driverPickupIds = useMemo(
    () => driverProfile?.route.pickupPoints?.map((point) => point.id) ?? [],
    [driverProfile?.route.pickupPoints],
  );

  const riderNetwork = useMemo(() => {
    if (!isDriver) {
      return [];
    }
    if (!driverPickupIds.length) {
      return riders;
    }
    return riders.filter((rider) => rider.pickupPointId && driverPickupIds.includes(rider.pickupPointId));
  }, [driverPickupIds, isDriver, riders]);

  const highlightedDrivers = useMemo(() => drivers.slice(0, 3), [drivers]);
  const highlightedRiders = useMemo(() => riderNetwork.slice(0, 3), [riderNetwork]);

  const driverMapRegion = useMemo(() => {
    if (!isDriver && userLatLng) {
      return buildRegion(userLatLng.latitude, userLatLng.longitude, 0.05);
    }
    const withLocation = drivers.find((driver) => driver.latitude && driver.longitude);
    if (withLocation?.latitude && withLocation.longitude) {
      return buildRegion(withLocation.latitude, withLocation.longitude, 0.05);
    }
    if (userLatLng) {
      return buildRegion(userLatLng.latitude, userLatLng.longitude, 0.05);
    }
    return DEFAULT_REGION;
  }, [drivers, isDriver, userLatLng]);

  const riderMapRegion = useMemo(() => {
    if (!isDriver) {
      return DEFAULT_REGION;
    }
    if (driverProfile?.latitude && driverProfile.longitude) {
      return buildRegion(driverProfile.latitude, driverProfile.longitude, 0.03);
    }
    const pickupMatch = riderNetwork.find((rider) => rider.pickup?.latitude && rider.pickup.longitude);
    if (pickupMatch?.pickup) {
      return buildRegion(pickupMatch.pickup.latitude, pickupMatch.pickup.longitude, 0.03);
    }
    if (userLatLng) {
      return buildRegion(userLatLng.latitude, userLatLng.longitude, 0.04);
    }
    return DEFAULT_REGION;
  }, [driverProfile?.latitude, driverProfile?.longitude, isDriver, riderNetwork, userLatLng]);

  const networkCard = isDriver ? (
    <GlassCard>
      <SectionHeading
        label="Rider network"
        subtitle={
          driverPickupIds.length
            ? 'Showing riders queued along your pickup route.'
            : 'Save a pickup plan in Trips to start targeting riders.'
        }
      />
      <View style={styles.networkMapShell}>
        <MapView
          provider={MAP_PROVIDER_DEFAULT}
          style={StyleSheet.absoluteFillObject}
          region={riderMapRegion}
        >
          {driverProfile?.latitude && driverProfile.longitude ? (
            <Marker
              coordinate={{ latitude: driverProfile.latitude, longitude: driverProfile.longitude }}
              title="You"
              pinColor="#38bdf8"
            />
          ) : null}
          {riderNetwork.map((rider) =>
            rider.pickup?.latitude && rider.pickup.longitude ? (
              <Marker
                key={rider.id}
                coordinate={{ latitude: rider.pickup.latitude, longitude: rider.pickup.longitude }}
                title={rider.name}
                description={rider.pickup.name}
                pinColor="#fbbf24"
              />
            ) : null,
          )}
        </MapView>
      </View>
      {highlightedRiders.length ? (
        highlightedRiders.map((rider) => (
          <View key={rider.id} style={styles.networkRow}>
            <View>
              <Text style={styles.networkName}>{rider.name}</Text>
              <Text style={styles.networkMeta}>Pickup · {rider.pickup?.name ?? rider.destination}</Text>
            </View>
            <Text style={styles.networkBadge}>{formatArrival(rider.arrivalTime)}</Text>
          </View>
        ))
      ) : (
        <Text style={styles.networkEmpty}>No riders match your active pickup points yet.</Text>
      )}
    </GlassCard>
  ) : (
    <GlassCard>
      <SectionHeading label="Driver network" subtitle="Live drivers covering your metro stations." />
      <View style={styles.networkMapShell}>
        <MapView
          provider={MAP_PROVIDER_DEFAULT}
          style={StyleSheet.absoluteFillObject}
          region={driverMapRegion}
        >
          {userLatLng ? (
            <Marker coordinate={userLatLng} title="You" pinColor="#38bdf8" />
          ) : null}
          {drivers.map((driver) =>
            driver.latitude && driver.longitude ? (
              <Marker
                key={driver.id}
                coordinate={{ latitude: driver.latitude, longitude: driver.longitude }}
                title={driver.name}
                description={`${driver.seatsAvailable} seats`}
                pinColor="#22d3ee"
              />
            ) : null,
          )}
        </MapView>
      </View>
      {highlightedDrivers.length ? (
        highlightedDrivers.map((driver) => (
          <View key={driver.id} style={styles.networkRow}>
            <View>
              <Text style={styles.networkName}>{driver.name}</Text>
              <Text style={styles.networkMeta}>Destination · {driver.route.destination}</Text>
            </View>
            <Text style={styles.networkBadge}>{driver.seatsAvailable} seats</Text>
          </View>
        ))
      ) : (
        <Text style={styles.networkEmpty}>Drivers will appear here once they come online.</Text>
      )}
    </GlassCard>
  );

  const handleMatch = async () => {
    if (!driverForHighlight || !stationForHighlight) return;
    console.log('[ui] Match now pressed', {
      driverId: driverForHighlight.id,
      stationId: stationForHighlight.id,
    });
    setMatchLoading(true);
    await matchDriver(driverForHighlight.id, stationForHighlight.id);
    setMatchLoading(false);
  };

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={[styles.content, { paddingTop: insets.top + 8 }]}
      refreshControl={<RefreshControl tintColor={palette.white} refreshing={loading} onRefresh={refresh} />}>
      <View style={styles.headerRow}>
        <View>
          <Text style={styles.heroTitle}>Station control</Text>
          <Text style={styles.heroSubtitle}>Monitor riders, drivers, and trips in real-time.</Text>
          {userLocation && (
            <Text style={{ color: palette.mint, fontSize: 12 }}>
              📍 {userLocation.coords.latitude.toFixed(4)}, {userLocation.coords.longitude.toFixed(4)}
            </Text>
          )}
          {errorMsg && <Text style={{ color: palette.rose, fontSize: 12 }}>{errorMsg}</Text>}
        </View>
        <TouchableOpacity onPress={() => navigation.navigate('Profile')} style={styles.profileButton}>
          <Text style={styles.profileButtonText}>Profile</Text>
        </TouchableOpacity>
      </View>

      {networkCard}

      <GlassCard>
        <SectionHeading label="Vitals" subtitle={`Updated ${relativeTime(snapshot?.lastUpdated)}`} />
        <ScrollView horizontal showsHorizontalScrollIndicator={false} style={styles.pillRow}>
          <StatPill label="Pending matches" value={`${snapshot?.metrics.pendingMatches ?? 0}`} />
          <StatPill label="Riders waiting" value={`${snapshot?.metrics.ridersWaiting ?? 0}`} />
          <StatPill label="Seats open" value={`${snapshot?.metrics.seatsOpen ?? 0}`} />
          <StatPill label="Avg wait" value={`${snapshot?.metrics.avgWaitMinutes ?? 0}m`} />
        </ScrollView>
      </GlassCard>



      {highlight ? (
        <>
          <TripCard trip={highlight} driver={driverForHighlight} rider={riderForHighlight} station={stationForHighlight} />
          {driverForHighlight ? (
            <LocationMap
              driverId={driverForHighlight.id}
              initialLat={37.7749}
              initialLong={-122.4194}
              userLocation={userLocation ? { latitude: userLocation.coords.latitude, longitude: userLocation.coords.longitude } : null}
            />
          ) : null}
        </>
      ) : null}

      <GlassCard>
        <SectionHeading label="Nearby stations" subtitle="Load factor shows heat" />
        {snapshot?.stations.map((station) => (
          <StationChip key={station.id} station={station} />
        ))}
      </GlassCard>

      {driverForHighlight && stationForHighlight ? (
        <GlassCard>
          <SectionHeading label="Match control" subtitle="Trigger a proactive match" />
          <Text style={styles.ctaCopy}>
            {driverForHighlight.name} is {driverForHighlight.etaMinutes} minutes from {stationForHighlight.name} with
            {driverForHighlight.seatsAvailable} open seats.
          </Text>
          <LiquidButton label="Match now" onPress={handleMatch} loading={matchLoading} />
        </GlassCard>
      ) : null}
    </ScrollView>
  );
};

const relativeTime = (iso?: string) => {
  if (!iso) return '—';
  const diff = Date.now() - new Date(iso).getTime();
  const minutes = Math.round(diff / 60000);
  if (minutes <= 1) return 'a moment ago';
  return `${minutes}m ago`;
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: palette.navy,
  },
  content: {
    padding: 20,
    paddingBottom: 80,
  },
  heroTitle: {
    fontSize: 32,
    fontWeight: '700',
    color: palette.white,
  },
  heroSubtitle: {
    color: palette.slate,
    marginTop: 4,
    marginBottom: 20,
  },
  pillRow: {
    marginHorizontal: -4,
  },
  ctaCopy: {
    color: palette.white,
    marginBottom: 16,
    lineHeight: 22,
  },
  headerRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    marginBottom: 20,
  },
  profileButton: {
    backgroundColor: 'rgba(255,255,255,0.1)',
    paddingHorizontal: 12,
    paddingVertical: 8,
    borderRadius: 20,
  },
  profileButtonText: {
    color: palette.white,
    fontWeight: '600',
  },
  networkMapShell: {
    height: 220,
    borderRadius: 20,
    overflow: 'hidden',
    marginBottom: 16,
  },
  networkRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: 8,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: 'rgba(255,255,255,0.08)',
  },
  networkName: {
    color: palette.white,
    fontSize: 16,
    fontWeight: '600',
  },
  networkMeta: {
    color: palette.slate,
    fontSize: 12,
    marginTop: 2,
  },
  networkBadge: {
    color: palette.gold,
    fontWeight: '600',
  },
  networkEmpty: {
    color: palette.slate,
    textAlign: 'center',
    paddingVertical: 12,
  },
});

export default DashboardScreen;
