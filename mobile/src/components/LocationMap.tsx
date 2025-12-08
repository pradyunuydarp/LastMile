import React, { useEffect, useState } from 'react';
import { StyleSheet, View } from 'react-native';
import { gateway } from '../services/gateway';
import { MapView, Marker, MAP_PROVIDER_DEFAULT } from './SafeMap';

interface LocationMapProps {
    driverId: string;
    initialLat: number;
    initialLong: number;
    userLocation?: { latitude: number; longitude: number } | null;
}

export const LocationMap: React.FC<LocationMapProps> = ({ driverId, initialLat, initialLong, userLocation }) => {
    const [location, setLocation] = useState({ latitude: initialLat, longitude: initialLong });

    useEffect(() => {
        const unsubscribe = gateway.subscribeToLocationUpdates(driverId, (update) => {
            if (update.latitude && update.longitude) {
                setLocation({
                    latitude: update.latitude,
                    longitude: update.longitude,
                });
            }
        });

        return () => {
            unsubscribe();
        };
    }, [driverId]);

    return (
        <View style={styles.container}>
            <MapView
                provider={MAP_PROVIDER_DEFAULT}
                style={styles.map}
                region={{
                    latitude: location.latitude,
                    longitude: location.longitude,
                    latitudeDelta: 0.01,
                    longitudeDelta: 0.01,
                }}
            >
                <Marker
                    coordinate={location}
                    title="Driver"
                    description={driverId}
                />
                {userLocation && (
                    <Marker
                        coordinate={userLocation}
                        title="You"
                        pinColor="blue"
                    />
                )}
            </MapView>
        </View>
    );
};

const styles = StyleSheet.create({
    container: {
        height: 300,
        width: '100%',
        borderRadius: 12,
        overflow: 'hidden',
        marginVertical: 10,
    },
    map: {
        ...StyleSheet.absoluteFillObject,
    },
});
