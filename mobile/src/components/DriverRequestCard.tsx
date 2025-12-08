import React from 'react';
import { StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { DriverRequest } from '../types';
import { GlassCard } from './GlassCard';
import { palette } from '../theme';

interface Props {
	request: DriverRequest;
	onAccept?: (request: DriverRequest) => void;
	accepting?: boolean;
}

const formatTime = (iso: string) => new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });

const formatDistance = (meters: number) => {
  if (!Number.isFinite(meters) || meters === Number.POSITIVE_INFINITY) {
    return '—';
  }
  if (meters >= 1000) {
    return `${(meters / 1000).toFixed(1)} km`;
  }
  return `${Math.round(meters)} m`;
};

export const DriverRequestCard: React.FC<Props> = ({ request, onAccept, accepting }) => (
	<GlassCard>
		<View style={styles.row}>
      <View>
        <Text style={styles.name}>{request.name}</Text>
        <Text style={styles.destination}>Pickup · {request.pickup?.name ?? request.destination}</Text>
      </View>
      <Text style={[styles.status, request.status === 'matched' && styles.statusMatched]}>{request.status}</Text>
    </View>
    <View style={styles.metaRow}>
      <Text style={styles.meta}>Drop · {request.station.name}</Text>
      <Text style={styles.meta}>Request · {formatTime(request.arrivalTime)}</Text>
    </View>
		<View style={styles.metaRow}>
			<Text style={styles.metaMuted}>Distance · {formatDistance(request.distanceMeters)}</Text>
			<Text style={styles.metaMuted}>Station Cluster · {request.station.nearbyAreas.join(', ')}</Text>
		</View>
		{onAccept ? (
			<TouchableOpacity
				style={[styles.acceptButton, accepting && styles.acceptButtonDisabled]}
				onPress={() => onAccept(request)}
				disabled={accepting}
			>
				<Text style={styles.acceptButtonText}>{accepting ? 'Confirming…' : 'Accept Rider'}</Text>
			</TouchableOpacity>
		) : null}
	</GlassCard>
);

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  name: {
    color: palette.white,
    fontSize: 18,
    fontWeight: '600',
  },
  destination: {
    color: palette.gold,
    marginTop: 2,
  },
  status: {
    color: palette.rose,
    textTransform: 'capitalize',
    fontWeight: '600',
  },
  statusMatched: {
    color: palette.mint,
  },
  metaRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginTop: 8,
  },
  meta: {
    color: palette.white,
  },
	metaMuted: {
		color: palette.slate,
		fontSize: 12,
	},
	acceptButton: {
		marginTop: 12,
		backgroundColor: '#10b981',
		paddingVertical: 12,
		borderRadius: 12,
		alignItems: 'center',
	},
	acceptButtonDisabled: {
		opacity: 0.6,
	},
	acceptButtonText: {
		color: palette.white,
		fontWeight: '600',
	},
});

export default DriverRequestCard;
