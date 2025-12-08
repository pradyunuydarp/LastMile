import React, { useCallback, useMemo, useState } from 'react';
import { FlatList, ScrollView, StyleSheet, Text, TextInput, TouchableOpacity, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useBackend } from '../services/BackendProvider';
import { useAuth } from '../context/AuthContext';
import { palette } from '../theme';
import { RiderCard } from '../components/RiderCard';
import { GlassCard } from '../components/GlassCard';
import { LiquidButton } from '../components/LiquidButton';
import { BookRideResponse } from '../types';
import { MapView, Marker, MAP_PROVIDER_DEFAULT } from '../components/SafeMap';
import { useRealtime } from '../context/RealtimeContext';

const DEFAULT_REGION = {
  latitude: 12.8456,
  longitude: 77.66,
  latitudeDelta: 0.05,
  longitudeDelta: 0.05,
};

const RidersScreen = () => {
  const { snapshot, bookRide, pickupPoints } = useBackend();
  const { riderStatus: liveStatus, approvalRequest, respondToApproval } = useRealtime();
  const { user, role } = useAuth();
  const insets = useSafeAreaInsets();
  const [pickupQuery, setPickupQuery] = useState('');
  const [selectedPickupId, setSelectedPickupId] = useState<string | null>(null);
  const [bookingError, setBookingError] = useState<string | null>(null);
  const [bookingResult, setBookingResult] = useState<BookRideResponse | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const riders = snapshot?.riders ?? [];
  const drivers = snapshot?.drivers ?? [];
  const isDriver = role === 'driver';
  const driverProfile = useMemo(
    () => (isDriver ? drivers.find((driver) => driver.id === user?.id) ?? null : null),
    [drivers, isDriver, user?.id],
  );
  const driverPickupIds = useMemo(
    () => driverProfile?.route.pickupPoints?.map((point) => point.id) ?? [],
    [driverProfile?.route.pickupPoints],
  );
  const visibleRiders = useMemo(() => {
    if (!isDriver) {
      return riders;
    }
    if (!driverPickupIds.length) {
      return [];
    }
    return riders.filter((rider) => rider.pickupPointId && driverPickupIds.includes(rider.pickupPointId));
  }, [driverPickupIds, isDriver, riders]);

  const selectedPickup = useMemo(
    () => pickupPoints.find((point) => point.id === selectedPickupId) ?? null,
    [pickupPoints, selectedPickupId],
  );

  const filteredPickups = useMemo(() => {
    if (!pickupQuery.trim()) {
      return pickupPoints.slice(0, 8);
    }
    const query = pickupQuery.toLowerCase();
    return pickupPoints
      .filter((point) => point.name.toLowerCase().includes(query) || point.stationName.toLowerCase().includes(query))
      .slice(0, 8);
  }, [pickupPoints, pickupQuery]);

  const mapRegion = useMemo(() => {
    if (selectedPickup) {
      return {
        latitude: selectedPickup.latitude,
        longitude: selectedPickup.longitude,
        latitudeDelta: 0.02,
        longitudeDelta: 0.02,
      };
    }
    return DEFAULT_REGION;
  }, [selectedPickup]);

  const driverMapRegion = useMemo(() => {
    if (!isDriver) {
      return mapRegion;
    }
    if (driverProfile?.latitude && driverProfile.longitude) {
      return {
        latitude: driverProfile.latitude,
        longitude: driverProfile.longitude,
        latitudeDelta: 0.03,
        longitudeDelta: 0.03,
      };
    }
    const riderWithPickup = visibleRiders.find((rider) => rider.pickup?.latitude && rider.pickup.longitude);
    if (riderWithPickup?.pickup) {
      return {
        latitude: riderWithPickup.pickup.latitude,
        longitude: riderWithPickup.pickup.longitude,
        latitudeDelta: 0.03,
        longitudeDelta: 0.03,
      };
    }
    return DEFAULT_REGION;
  }, [driverProfile, isDriver, mapRegion, visibleRiders]);

  const handleBook = useCallback(async () => {
    if (!selectedPickup) {
      setBookingError('Choose a pickup point to continue.');
      return;
    }
    setBookingError(null);
    setSubmitting(true);
    const metadata = user?.user_metadata as Record<string, unknown> | undefined;
    const profileName = typeof metadata?.full_name === 'string' ? (metadata?.full_name as string) : undefined;
    const riderName =
      (typeof user?.name === 'string' ? user?.name : undefined) ||
      (typeof user?.email === 'string' ? user?.email : undefined) ||
      profileName ||
      undefined;
    const result = await bookRide({
      command: 'book',
      address: selectedPickup.name,
      destination: selectedPickup.stationName,
      pickupPointId: selectedPickup.id,
      riderId: user?.id,
      name: riderName,
    });
    setSubmitting(false);
    if (!result) {
      setBookingError('Unable to book right now, please try again.');
      return;
    }
    setBookingResult(result);
    setPickupQuery('');
    setSelectedPickupId(null);
  }, [bookRide, selectedPickup, user?.id, user?.name]);

  const header = useMemo(() => {
    if (isDriver) {
      return (
        <View>
          <Text style={styles.heading}>Riders along your route</Text>
          <GlassCard>
            <Text style={styles.formTitle}>Live rider queue</Text>
            <Text style={styles.formCopy}>
              {driverPickupIds.length
                ? 'Showing riders who selected pickup points from your Trips tab.'
                : 'Save your pickup plan in the Trips tab to start seeing matched riders.'}
            </Text>
            <View style={styles.mapShell}>
              <MapView provider={MAP_PROVIDER_DEFAULT} style={StyleSheet.absoluteFillObject} region={driverMapRegion}>
                {driverProfile?.latitude && driverProfile.longitude ? (
                  <Marker
                    coordinate={{ latitude: driverProfile.latitude, longitude: driverProfile.longitude }}
                    title="You"
                    pinColor="#38bdf8"
                  />
                ) : null}
                {visibleRiders.map((rider) =>
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
          </GlassCard>
        </View>
      );
    }

    return (
      <View>
        <Text style={styles.heading}>Riders awaiting pickup</Text>
          <GlassCard>
            <Text style={styles.formTitle}>Book a last-mile drop</Text>
          <Text style={styles.formCopy}>Pick a pickup cluster near your station. We will alert nearby drivers.</Text>
          <TextInput
            style={styles.input}
            placeholder="Search pickup point"
            placeholderTextColor="#94a3b8"
            value={pickupQuery}
            onChangeText={setPickupQuery}
          />
          <View style={styles.pickupGrid}>
            {filteredPickups.map((point) => (
              <TouchableOpacity
                key={point.id}
                style={[styles.pickupChip, selectedPickupId === point.id && styles.pickupChipSelected]}
                onPress={() => setSelectedPickupId(point.id)}
              >
                <Text style={styles.pickupName}>{point.name}</Text>
                <Text style={styles.pickupStation}>{point.stationName}</Text>
              </TouchableOpacity>
            ))}
          </View>
          {selectedPickup ? (
            <Text style={styles.selection}>Drop-off station · {selectedPickup.stationName}</Text>
          ) : (
            <Text style={styles.selectionMuted}>Tap a pickup suggestion above to continue.</Text>
          )}
          <View style={styles.mapShell}>
            <MapView provider={MAP_PROVIDER_DEFAULT} style={StyleSheet.absoluteFillObject} region={mapRegion}>
              {selectedPickup ? (
                <Marker
                  coordinate={{ latitude: selectedPickup.latitude, longitude: selectedPickup.longitude }}
                  title={selectedPickup.name}
                  description={`Nearest station ${selectedPickup.stationName}`}
                />
              ) : null}
              {drivers
                .filter((driver) => driver.latitude && driver.longitude)
                .map((driver) => (
                  <Marker
                    key={driver.id}
                    coordinate={{ latitude: driver.latitude!, longitude: driver.longitude! }}
                    pinColor="#22d3ee"
                    title={driver.name}
                  />
                ))}
            </MapView>
          </View>
          {bookingError ? <Text style={styles.errorText}>{bookingError}</Text> : null}
          <LiquidButton label={submitting ? 'Booking…' : 'Request pickup'} onPress={handleBook} loading={submitting} />
        </GlassCard>
        {bookingResult ? (
          <GlassCard>
            <Text style={styles.resultTitle}>{bookingResult.message}</Text>
            <Text style={styles.resultMeta}>Destination · {bookingResult.requestedDestination}</Text>
            {bookingResult.attempts.map((attempt) => (
              <View key={attempt.driverId} style={styles.attemptRow}>
                <Text style={[styles.attemptLabel, attempt.accepted && styles.attemptAccepted]}>
                  {attempt.driverName}
                </Text>
                <Text style={styles.attemptStatus}>
                  {attempt.accepted ? 'Accepted' : attempt.reason ?? 'Pending'}
                </Text>
              </View>
            ))}
          </GlassCard>
        ) : null}
        {approvalRequest ? (
          <GlassCard>
            <Text style={styles.resultTitle}>{approvalRequest.driverName ?? 'Driver'} accepted your ride</Text>
            <Text style={styles.resultMeta}>Pickup · {approvalRequest.pickup?.name ?? '—'}</Text>
            <Text style={styles.resultMeta}>Station · {approvalRequest.station?.name ?? '—'}</Text>
            <View style={styles.approvalRow}>
              <TouchableOpacity style={styles.approveButton} onPress={() => respondToApproval(approvalRequest.tripId, true)}>
                <Text style={styles.approveText}>Approve</Text>
              </TouchableOpacity>
              <TouchableOpacity
                style={[styles.approveButton, styles.declineButton]}
                onPress={() => respondToApproval(approvalRequest.tripId, false)}
              >
                <Text style={styles.declineText}>Decline</Text>
              </TouchableOpacity>
            </View>
          </GlassCard>
        ) : null}
        {liveStatus ? (
          <GlassCard>
            <Text style={styles.resultTitle}>Trip status · {liveStatus.status}</Text>
            <Text style={styles.resultMeta}>Driver · {liveStatus.driverId ?? 'Assigning'}</Text>
            <View style={styles.mapShell}>
              <MapView
                provider={MAP_PROVIDER_DEFAULT}
                style={StyleSheet.absoluteFillObject}
                region={{
                  latitude: liveStatus.latitude ?? selectedPickup?.latitude ?? 12.8456,
                  longitude: liveStatus.longitude ?? selectedPickup?.longitude ?? 77.66,
                  latitudeDelta: 0.02,
                  longitudeDelta: 0.02,
                }}
              >
                {liveStatus.latitude && liveStatus.longitude ? (
                  <Marker coordinate={{ latitude: liveStatus.latitude, longitude: liveStatus.longitude }} title="Driver" />
                ) : null}
                {liveStatus.pickup?.latitude && liveStatus.pickup.longitude ? (
                  <Marker coordinate={{ latitude: liveStatus.pickup.latitude, longitude: liveStatus.pickup.longitude }} title={liveStatus.pickup.name} />
                ) : null}
                {liveStatus.station?.latitude && liveStatus.station.longitude ? (
                  <Marker coordinate={{ latitude: liveStatus.station.latitude, longitude: liveStatus.station.longitude }} title={liveStatus.station.name} />
                ) : null}
              </MapView>
            </View>
          </GlassCard>
        ) : null}
      </View>
    );
  }, [
    approvalRequest,
    bookingError,
    bookingResult,
    driverMapRegion,
    driverPickupIds.length,
    driverProfile?.latitude,
    driverProfile?.longitude,
    drivers,
    filteredPickups,
    handleBook,
    isDriver,
    mapRegion,
    pickupQuery,
    liveStatus,
    respondToApproval,
    selectedPickup,
    selectedPickupId,
    submitting,
    visibleRiders,
  ]);

  if (!isDriver) {
    return (
      <ScrollView
        style={styles.container}
        contentContainerStyle={[styles.listContent, { paddingTop: insets.top + 8 }]}
      >
        {header}
      </ScrollView>
    );
  }

  return (
    <View style={styles.container}>
      <FlatList
        data={visibleRiders}
        keyExtractor={(rider) => rider.id}
        renderItem={({ item }) => <RiderCard rider={item} />}
        ListHeaderComponent={header}
        contentContainerStyle={[styles.listContent, { paddingTop: insets.top + 8 }]}
        ListEmptyComponent={
          <Text style={styles.emptyText}>
            No riders have queued up along your pickup clusters yet.
          </Text>
        }
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
  formTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: palette.white,
    marginBottom: 4,
  },
  formCopy: {
    color: palette.slate,
    marginBottom: 12,
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
  selection: {
    color: palette.white,
    marginBottom: 12,
  },
  selectionMuted: {
    color: palette.slate,
    marginBottom: 12,
  },
  mapShell: {
    height: 220,
    borderRadius: 16,
    overflow: 'hidden',
    marginBottom: 12,
  },
  errorText: {
    color: palette.rose,
    marginBottom: 12,
  },
  resultTitle: {
    color: palette.white,
    fontSize: 16,
    fontWeight: '600',
    marginBottom: 6,
  },
  resultMeta: {
    color: palette.slate,
    fontSize: 12,
    marginBottom: 12,
  },
  attemptRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginTop: 4,
  },
  attemptLabel: {
    color: palette.slate,
  },
  attemptAccepted: {
    color: palette.mint,
  },
  attemptStatus: {
    color: palette.white,
    fontSize: 12,
  },
  approvalRow: {
    flexDirection: 'row',
    columnGap: 12,
    marginTop: 12,
  },
  approveButton: {
    flex: 1,
    backgroundColor: palette.teal,
    borderRadius: 12,
    paddingVertical: 12,
    alignItems: 'center',
  },
  approveText: {
    color: palette.navy,
    fontWeight: '700',
  },
  declineButton: {
    backgroundColor: 'transparent',
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.25)',
  },
  declineText: {
    color: palette.rose,
    fontWeight: '700',
  },
  listContent: {
    padding: 20,
    paddingBottom: 80,
  },
  emptyText: {
    color: palette.slate,
    textAlign: 'center',
    marginTop: 40,
  },
});

export default RidersScreen;
