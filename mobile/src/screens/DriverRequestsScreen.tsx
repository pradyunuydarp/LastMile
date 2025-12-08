import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { RefreshControl, ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useAuth } from '../context/AuthContext';
import { useBackend } from '../services/BackendProvider';
import { DriverRequest, DriverRequestsResponse } from '../types';
import { gateway } from '../services/gateway';
import { GlassCard } from '../components/GlassCard';
import DriverRequestCard from '../components/DriverRequestCard';
import { palette } from '../theme';
import { MapView, Marker, MAP_PROVIDER_DEFAULT } from '../components/SafeMap';
import { useRealtime } from '../context/RealtimeContext';

const DEFAULT_REGION = {
  latitude: 12.8456,
  longitude: 77.66,
  latitudeDelta: 0.04,
  longitudeDelta: 0.04,
};

const DriverRequestsScreen: React.FC = () => {
  const { user } = useAuth();
  const { snapshot, refresh, acceptDriverRequest } = useBackend();
  const { offers, queue, respondToOffer, ready: realtimeReady } = useRealtime();
  const insets = useSafeAreaInsets();
  const [data, setData] = useState<DriverRequestsResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [acceptingId, setAcceptingId] = useState<string | null>(null);
  const [actionMessage, setActionMessage] = useState<string | null>(null);

  const driver = useMemo(() => snapshot?.drivers.find((d) => d.id === user?.id), [snapshot, user]);

  const load = useCallback(async () => {
    if (!user?.id) {
      return;
    }
    setLoading(true);
    try {
      const response = await gateway.fetchDriverRequests(user.id);
      setData(response);
      setError(null);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [user?.id]);

  useEffect(() => {
    if (realtimeReady) {
      return;
    }
    load();
    const interval = setInterval(load, 15000);
    return () => clearInterval(interval);
  }, [load, realtimeReady]);

  useEffect(() => {
    if (queue) {
      setData(queue);
      setError(null);
    }
  }, [queue]);

  const mapRegion = useMemo(() => {
    if (driver?.latitude && driver?.longitude) {
      return {
        latitude: driver.latitude,
        longitude: driver.longitude,
        latitudeDelta: 0.03,
        longitudeDelta: 0.03,
      };
    }
    if (data?.requests.length) {
      const first = data.requests.find((req) => req.station.latitude && req.station.longitude);
      if (first?.station.latitude && first.station.longitude) {
        return {
          latitude: first.station.latitude,
          longitude: first.station.longitude,
          latitudeDelta: 0.03,
          longitudeDelta: 0.03,
        };
      }
    }
    return DEFAULT_REGION;
  }, [data, driver]);

  const handleAccept = useCallback(
    async (request: DriverRequest) => {
      if (!user?.id) {
        setActionMessage('Sign in as a driver to accept riders.');
        return;
      }
      setAcceptingId(request.id);
      setActionMessage(null);
      const trip = await acceptDriverRequest(user.id, request.id);
      if (!trip) {
        setActionMessage('Unable to confirm this rider right now.');
        setAcceptingId(null);
        return;
      }
      setActionMessage(`Trip confirmed · ${request.name} will be notified.`);
      await load();
      await refresh();
      setAcceptingId(null);
    },
    [acceptDriverRequest, load, refresh, user?.id],
  );

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={[styles.content, { paddingTop: insets.top + 8 }]}
      refreshControl={<RefreshControl refreshing={loading} tintColor={palette.white} onRefresh={() => { load(); refresh(); }} />}
    >
      <Text style={styles.heading}>Customer requests</Text>
      <GlassCard>
        <Text style={styles.summaryTitle}>{data ? data.driver.name : driver?.name ?? 'Driver console'}</Text>
        <Text style={styles.summaryCopy}>
          {data ? `${data.driver.seatsAvailable} seats open · Next stop ${data.driver.nextStop || '—'}` : 'No live requests yet.'}
        </Text>
        <Text style={styles.summaryMeta}>Updated {data ? new Date(data.generatedAt).toLocaleTimeString() : '—'}</Text>
      </GlassCard>

      {actionMessage ? <Text style={styles.helperText}>{actionMessage}</Text> : null}

      <View style={styles.mapShell}>
        {data?.requests.length ? (
          <MapView provider={MAP_PROVIDER_DEFAULT} style={StyleSheet.absoluteFillObject} region={mapRegion}>
            {driver?.latitude && driver?.longitude ? (
              <Marker coordinate={{ latitude: driver.latitude, longitude: driver.longitude }} title="You" pinColor="#38bdf8" />
            ) : null}
            {data.requests.map((request) =>
              request.pickup?.latitude && request.pickup.longitude ? (
                <Marker
                  key={request.id}
                  coordinate={{
                    latitude: request.pickup.latitude,
                    longitude: request.pickup.longitude,
                  }}
                  title={request.name}
                  description={request.pickup.name}
                  pinColor={request.status === 'matched' ? '#34d399' : '#fbbf24'}
                />
              ) : null,
            )}
          </MapView>
        ) : (
          <Text style={styles.mapEmpty}>Requests will appear once riders check in near your stations.</Text>
        )}
      </View>

      {error ? <Text style={styles.errorText}>{error}</Text> : null}

      {data?.requests.map((request) => (
        <DriverRequestCard
          key={request.id}
          request={request}
          onAccept={handleAccept}
          accepting={acceptingId === request.id}
        />
      ))}

      {!data?.requests.length ? <Text style={styles.emptyState}>No live rider requests near your stops right now.</Text> : null}

      <View style={styles.offerSection}>
        <View style={styles.offerHeader}>
          <Text style={styles.offerTitle}>Live socket offers</Text>
          <Text style={[styles.offerStatus, { color: realtimeReady ? palette.emerald : palette.slate }]}>
            {realtimeReady ? 'Connected' : 'Connecting…'}
          </Text>
        </View>
        {offers.length ? (
          offers.map((offer) => (
            <GlassCard key={offer.riderId} style={styles.offerCard}>
              <Text style={styles.offerName}>{offer.riderName}</Text>
              <Text style={styles.offerMeta}>Pickup · {offer.pickup?.name ?? offer.station?.name ?? 'Pending'}</Text>
              <Text style={styles.offerMeta}>Destination · {offer.destination ?? '—'}</Text>
              <Text style={styles.offerMeta}>Attempt {offer.attempt} of {offer.total}</Text>
              <View style={styles.offerActions}>
                <TouchableOpacity style={styles.offerAccept} onPress={() => respondToOffer(offer.riderId, true)}>
                  <Text style={styles.offerAcceptText}>Accept</Text>
                </TouchableOpacity>
                <TouchableOpacity style={styles.offerDecline} onPress={() => respondToOffer(offer.riderId, false)}>
                  <Text style={styles.offerDeclineText}>Pass</Text>
                </TouchableOpacity>
              </View>
            </GlassCard>
          ))
        ) : (
          <Text style={styles.offerMeta}>No realtime riders yet.</Text>
        )}
      </View>
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
  summaryTitle: {
    fontSize: 20,
    fontWeight: '600',
    color: palette.white,
  },
  summaryCopy: {
    color: palette.slate,
    marginTop: 4,
  },
  summaryMeta: {
    color: palette.slate,
    marginTop: 8,
    fontSize: 12,
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
  helperText: {
    color: palette.slate,
    marginBottom: 8,
  },
  mapShell: {
    height: 250,
    borderRadius: 20,
    overflow: 'hidden',
    marginBottom: 24,
  },
  mapEmpty: {
    color: palette.slate,
    textAlign: 'center',
    marginTop: 16,
  },
  errorText: {
    color: palette.rose,
    marginBottom: 12,
  },
  emptyState: {
    color: palette.slate,
    textAlign: 'center',
    marginTop: 12,
  },
  offerSection: {
    marginTop: 24,
  },
  offerHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  offerTitle: {
    fontSize: 18,
    fontWeight: '700',
    color: palette.white,
  },
  offerStatus: {
    fontSize: 12,
    textTransform: 'uppercase',
  },
  offerCard: {
    marginBottom: 12,
  },
  offerName: {
    fontSize: 16,
    fontWeight: '600',
    color: palette.white,
  },
  offerMeta: {
    color: palette.slate,
    fontSize: 12,
  },
  offerActions: {
    flexDirection: 'row',
    columnGap: 12,
    marginTop: 12,
  },
  offerAccept: {
    flex: 1,
    backgroundColor: palette.teal,
    borderRadius: 12,
    paddingVertical: 10,
    alignItems: 'center',
  },
  offerAcceptText: {
    color: palette.navy,
    fontWeight: '700',
  },
  offerDecline: {
    flex: 1,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: palette.slate,
    paddingVertical: 10,
    alignItems: 'center',
  },
  offerDeclineText: {
    color: palette.slate,
    fontWeight: '700',
  },
});

export default DriverRequestsScreen;
