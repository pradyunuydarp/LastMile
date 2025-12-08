import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { ScrollView, StyleSheet, Text, TextInput, TouchableOpacity, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useBackend } from '../services/BackendProvider';
import { useAuth } from '../context/AuthContext';
import { GlassCard } from '../components/GlassCard';
import { LiquidButton } from '../components/LiquidButton';
import { palette } from '../theme';
import { PickupPoint } from '../types';
import { MapView, Marker, MAP_PROVIDER_DEFAULT, Polyline } from '../components/SafeMap';
import { TripRoom, useRealtime } from '../context/RealtimeContext';

const DEFAULT_REGION = {
  latitude: 12.8456,
  longitude: 77.66,
  latitudeDelta: 0.05,
  longitudeDelta: 0.05,
};

const DriverTripsScreen = () => {
  const { pickupPoints, saveDriverRoute, startDriverTrip, snapshot } = useBackend();
  const { user } = useAuth();
  const insets = useSafeAreaInsets();
  const driver = useMemo(() => snapshot?.drivers.find((d) => d.id === user?.id), [snapshot, user?.id]);
  const { rooms, completeTrip, ready: realtimeReady } = useRealtime();

  const [routeQuery, setRouteQuery] = useState('');
  const [selectedPickupIds, setSelectedPickupIds] = useState<string[]>(driver?.route.pickupPoints?.map((p) => p.id) ?? []);
  const [seatCount, setSeatCount] = useState(() =>
    driver?.route.pickupPoints?.length ? `${driver.seatsAvailable || driver.route.pickupPoints.length}` : '3',
  );
  const [routeMessage, setRouteMessage] = useState<string | null>(null);
  const [tripMessage, setTripMessage] = useState<string | null>(null);
  const [savingRoute, setSavingRoute] = useState(false);
  const [startingTrip, setStartingTrip] = useState(false);

  useEffect(() => {
    if (driver?.route.pickupPoints?.length) {
      setSelectedPickupIds(driver.route.pickupPoints.map((point) => point.id));
      if (driver.seatsAvailable) {
        setSeatCount(`${driver.seatsAvailable}`);
      }
    }
  }, [driver?.route.pickupPoints, driver?.seatsAvailable]);

  const filteredPickups = useMemo(() => {
    if (!routeQuery.trim()) {
      return pickupPoints.slice(0, 12);
    }
    const query = routeQuery.toLowerCase();
    return pickupPoints
      .filter((point) => point.name.toLowerCase().includes(query) || point.stationName.toLowerCase().includes(query))
      .slice(0, 12);
  }, [pickupPoints, routeQuery]);

  const selectedPickups = useMemo<PickupPoint[]>(
    () => selectedPickupIds.map((id) => pickupPoints.find((p) => p.id === id)).filter(Boolean) as PickupPoint[],
    [pickupPoints, selectedPickupIds],
  );

  const mapRegion = useMemo(() => {
    if (driver?.latitude && driver?.longitude) {
      return {
        latitude: driver.latitude,
        longitude: driver.longitude,
        latitudeDelta: 0.03,
        longitudeDelta: 0.03,
      };
    }
    if (selectedPickups.length) {
      return {
        latitude: selectedPickups[0].latitude,
        longitude: selectedPickups[0].longitude,
        latitudeDelta: 0.03,
        longitudeDelta: 0.03,
      };
    }
    return DEFAULT_REGION;
  }, [driver, selectedPickups]);

  const togglePickup = (id: string) => {
    setSelectedPickupIds((current) => (current.includes(id) ? current.filter((pid) => pid !== id) : [...current, id]));
  };

  const handleSaveRoute = useCallback(async () => {
    if (!user?.id) {
      setRouteMessage('Sign in as a driver to configure routes.');
      return;
    }
    if (!selectedPickupIds.length) {
      setRouteMessage('Select at least one pickup cluster.');
      return;
    }
    const seats = Number.parseInt(seatCount, 10) || 1;
    setSavingRoute(true);
    const response = await saveDriverRoute({
      driverId: user.id,
      name: driver?.name ?? user.name ?? 'Metro Captain',
      carDetails: driver?.carDetails ?? 'Cab',
      pickupPointIds: selectedPickupIds,
      seats,
    });
    setSavingRoute(false);
    if (response) {
      setRouteMessage(`Route saved · ${response.pickupPoints.length} stops, ${response.seatsTotal} seats`);
    } else {
      setRouteMessage('Failed to save route, try again.');
    }
  }, [driver, saveDriverRoute, seatCount, selectedPickupIds, user]);

  const handleStartTrip = useCallback(
    async (simulate?: boolean) => {
      if (!user?.id) {
        setTripMessage('Sign in as a driver to start trips.');
        return;
      }
      setStartingTrip(true);
      const result = await startDriverTrip(user.id, simulate);
      setStartingTrip(false);
      if (result) {
        setTripMessage(simulate ? 'Simulated trip started. Watch the map!' : 'Trip started. Share live location.');
      } else {
        setTripMessage('Unable to start trip, save a route first.');
      }
    },
    [startDriverTrip, user?.id],
  );

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={[styles.content, { paddingTop: insets.top + 8 }]}
    >
      <Text style={styles.heading}>Trip planning</Text>
      <GlassCard>
        <Text style={styles.sectionTitle}>Plan your metro drops</Text>
        <Text style={styles.sectionCopy}>Pick the pickup clusters you will cover before heading to the station.</Text>
        <TextInput
          style={styles.input}
          placeholder="Search pickup"
          placeholderTextColor="#94a3b8"
          value={routeQuery}
          onChangeText={setRouteQuery}
        />
        <View style={styles.pickupGrid}>
          {filteredPickups.map((point) => (
            <TouchableOpacity
              key={point.id}
              style={[styles.pickupChip, selectedPickupIds.includes(point.id) && styles.pickupChipSelected]}
              onPress={() => togglePickup(point.id)}
            >
              <Text style={styles.pickupName}>{point.name}</Text>
              <Text style={styles.pickupStation}>{point.stationName}</Text>
            </TouchableOpacity>
          ))}
        </View>
        {selectedPickups.length ? (
          <Text style={styles.selectionText}>
            Stops in order: {selectedPickups.map((p, index) => `${index + 1}. ${p.name}`).join('  ·  ')}
          </Text>
        ) : (
          <Text style={styles.selectionSubtle}>Tap pickup chips to curate your route.</Text>
        )}
        <TextInput
          style={styles.input}
          placeholder="Seats available"
          placeholderTextColor="#94a3b8"
          keyboardType="number-pad"
          value={seatCount}
          onChangeText={setSeatCount}
        />
        {routeMessage ? <Text style={styles.helperText}>{routeMessage}</Text> : null}
        <LiquidButton label={savingRoute ? 'Saving route…' : 'Save route'} onPress={handleSaveRoute} loading={savingRoute} />
        <View style={styles.tripButtons}>
          <TouchableOpacity style={styles.tripButton} onPress={() => handleStartTrip(false)} disabled={startingTrip}>
            <Text style={styles.tripButtonText}>{startingTrip ? 'Starting…' : 'Start Trip'}</Text>
          </TouchableOpacity>
          <TouchableOpacity
            style={[styles.tripButton, styles.tripButtonGhost]}
            onPress={() => handleStartTrip(true)}
            disabled={startingTrip}
          >
            <Text style={styles.tripButtonGhostText}>Simulated Trip</Text>
          </TouchableOpacity>
        </View>
        {tripMessage ? <Text style={styles.helperText}>{tripMessage}</Text> : null}
        <View style={styles.routeMap}>
        <MapView provider={MAP_PROVIDER_DEFAULT} style={StyleSheet.absoluteFillObject} region={mapRegion}>
            {driver?.latitude && driver.longitude ? (
              <Marker coordinate={{ latitude: driver.latitude, longitude: driver.longitude }} title="You" pinColor="#38bdf8" />
            ) : null}
            {selectedPickups.map((point) => (
              <Marker
                key={point.id}
                coordinate={{ latitude: point.latitude, longitude: point.longitude }}
                title={point.name}
                description={point.stationName}
              />
            ))}
            {selectedPickups.length >= 2 ? (
              <Polyline
                coordinates={selectedPickups.map((point) => ({ latitude: point.latitude, longitude: point.longitude }))}
                strokeColor="#fbbf24"
                strokeWidth={4}
              />
            ) : null}
          </MapView>
        </View>
      </GlassCard>

      <GlassCard>
        <Text style={styles.sectionTitle}>Trip readiness</Text>
        <Text style={styles.sectionCopy}>
          Destination · {driver?.route.destination ?? '—'}
        </Text>
        <Text style={styles.sectionCopy}>Seats available · {driver?.seatsAvailable ?? seatCount}</Text>
      </GlassCard>

      <GlassCard>
        <View style={styles.roomHeader}>
          <Text style={styles.sectionTitle}>Active trip rooms</Text>
          <Text style={[styles.roomStatus, { color: realtimeReady ? palette.teal : palette.slate }]}>
            {realtimeReady ? 'Connected' : 'Connecting…'}
          </Text>
        </View>
        {rooms.length ? (
          rooms.map((room: TripRoom) => (
            <View key={room.tripId} style={styles.roomCard}>
              <View style={styles.roomRow}>
                <View>
                  <Text style={styles.roomTitle}>{room.trip?.riderId ?? room.riderId}</Text>
                  <Text style={styles.roomMeta}>Pickup · {room.pickup?.name ?? '—'}</Text>
                  <Text style={styles.roomMeta}>Station · {room.station?.name ?? '—'}</Text>
                </View>
                <Text style={[styles.roomBadge, room.status === 'completed' && styles.roomBadgeComplete]}>
                  {room.status}
                </Text>
              </View>
              <View style={styles.roomMap}>
                <MapView
                  provider={MAP_PROVIDER_DEFAULT}
                  style={StyleSheet.absoluteFillObject}
                  region={{
                    latitude: room.latitude ?? room.pickup?.latitude ?? DEFAULT_REGION.latitude,
                    longitude: room.longitude ?? room.pickup?.longitude ?? DEFAULT_REGION.longitude,
                    latitudeDelta: 0.02,
                    longitudeDelta: 0.02,
                  }}
                >
                  {room.latitude && room.longitude ? (
                    <Marker coordinate={{ latitude: room.latitude, longitude: room.longitude }} title="Driver" pinColor="#0ea5e9" />
                  ) : null}
                  {room.pickup?.latitude && room.pickup.longitude ? (
                    <Marker coordinate={{ latitude: room.pickup.latitude, longitude: room.pickup.longitude }} title="Pickup" pinColor="#fbbf24" />
                  ) : null}
                  {room.station?.latitude && room.station.longitude ? (
                    <Marker coordinate={{ latitude: room.station.latitude, longitude: room.station.longitude }} title="Station" pinColor="#34d399" />
                  ) : null}
                </MapView>
              </View>
              <TouchableOpacity style={styles.roomComplete} onPress={() => completeTrip(room.tripId)}>
                <Text style={styles.roomCompleteText}>Mark Complete</Text>
              </TouchableOpacity>
            </View>
          ))
        ) : (
          <Text style={styles.sectionCopy}>No rides assigned yet.</Text>
        )}
      </GlassCard>
    </ScrollView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: palette.navy,
  },
  content: {
    padding: 20,
    paddingBottom: 60,
  },
  heading: {
    fontSize: 28,
    fontWeight: '700',
    color: palette.white,
    marginBottom: 16,
  },
  sectionTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: palette.white,
    marginBottom: 4,
  },
  sectionCopy: {
    color: palette.slate,
    marginBottom: 10,
  },
  input: {
    backgroundColor: 'rgba(15,23,42,0.7)',
    borderColor: 'rgba(255,255,255,0.08)',
    borderWidth: 1,
    borderRadius: 12,
    paddingHorizontal: 14,
    paddingVertical: 12,
    color: palette.white,
    marginBottom: 12,
  },
  pickupGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
    marginBottom: 12,
  },
  pickupChip: {
    paddingHorizontal: 12,
    paddingVertical: 10,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.08)',
    backgroundColor: 'rgba(15,23,42,0.5)',
    width: '48%',
  },
  pickupChipSelected: {
    borderColor: palette.mint,
    backgroundColor: 'rgba(16,185,129,0.15)',
  },
  pickupName: {
    color: palette.white,
    fontWeight: '600',
  },
  pickupStation: {
    color: palette.slate,
    fontSize: 12,
  },
  selectionText: {
    color: palette.white,
    marginBottom: 12,
  },
  selectionSubtle: {
    color: palette.slate,
    marginBottom: 12,
  },
  helperText: {
    color: palette.slate,
    marginTop: 8,
    marginBottom: 8,
  },
  routeMap: {
    height: 220,
    borderRadius: 16,
    overflow: 'hidden',
    marginTop: 12,
  },
  tripButtons: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginTop: 12,
    gap: 12,
  },
  tripButton: {
    flex: 1,
    backgroundColor: '#10b981',
    paddingVertical: 14,
    borderRadius: 12,
    alignItems: 'center',
  },
  tripButtonText: {
    color: palette.white,
    fontWeight: '600',
  },
  tripButtonGhost: {
    backgroundColor: 'transparent',
    borderWidth: 1,
    borderColor: '#10b981',
  },
  tripButtonGhostText: {
    color: '#10b981',
    fontWeight: '600',
  },
  roomHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  roomStatus: {
    fontSize: 12,
    textTransform: 'uppercase',
  },
  roomCard: {
    marginTop: 12,
    borderRadius: 16,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.08)',
    padding: 12,
    backgroundColor: 'rgba(15,23,42,0.5)',
  },
  roomRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  roomTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: palette.white,
  },
  roomMeta: {
    color: palette.slate,
    fontSize: 12,
  },
  roomBadge: {
    color: palette.gold,
    textTransform: 'uppercase',
    fontSize: 12,
    fontWeight: '600',
  },
  roomBadgeComplete: {
    color: palette.teal,
  },
  roomMap: {
    height: 160,
    borderRadius: 12,
    overflow: 'hidden',
    marginTop: 12,
  },
  roomComplete: {
    marginTop: 12,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: palette.teal,
    paddingVertical: 10,
    alignItems: 'center',
  },
  roomCompleteText: {
    color: palette.teal,
    fontWeight: '600',
  },
});

export default DriverTripsScreen;
