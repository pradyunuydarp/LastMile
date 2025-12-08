import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, Alert } from 'react-native';
import { useAuth } from '../context/AuthContext';
import { gateway } from '../services/gateway';
import { NativeStackScreenProps } from '@react-navigation/native-stack';
import { RootStackParamList } from '../types';
import { palette as colors } from '../theme/colors';
import { spacing } from '../theme/spacing';
import { typography } from '../theme/typography';

type Props = NativeStackScreenProps<RootStackParamList, 'Profile'>;

export const ProfileScreen = ({ navigation }: Props) => {
    const [loading, setLoading] = useState(true);
    const [profile, setProfile] = useState<{
        username: string;
        full_name: string;
        role: string;
    } | null>(null);
    const { user, signOut } = useAuth();

    useEffect(() => {
        getProfile();
    }, [user]);

    const getProfile = async () => {
        try {
            setLoading(true);
            if (!user?.id) {
                // If no user in context, maybe we are not logged in properly or just signed out
                return;
            }

            // Fetch fresh profile data from backend
            const response = await gateway.getUser(user.id);
            if (response && response.user) {
                setProfile({
                    username: response.user.email, // Using email as username for now
                    full_name: response.user.name,
                    role: response.user.role === 2 ? 'Driver' : 'Rider', // Map enum to string
                });
            }
        } catch (error) {
            if (error instanceof Error) {
                console.warn('Failed to fetch profile:', error.message);
            }
        } finally {
            setLoading(false);
        }
    };

    const handleSignOut = async () => {
        await signOut();
        // Navigation to Auth stack is handled by AppNavigator based on session state
    };

    return (
        <View style={styles.container}>
            <View style={styles.header}>
                <Text style={styles.title}>Profile</Text>
            </View>

            {loading ? (
                <Text style={styles.loadingText}>Loading...</Text>
            ) : (
                <View style={styles.content}>
                    <View style={styles.infoRow}>
                        <Text style={styles.label}>Full Name</Text>
                        <Text style={styles.value}>{profile?.full_name || user?.name || 'N/A'}</Text>
                    </View>

                    <View style={styles.infoRow}>
                        <Text style={styles.label}>Email</Text>
                        <Text style={styles.value}>{user?.email || 'N/A'}</Text>
                    </View>

                    <View style={styles.infoRow}>
                        <Text style={styles.label}>Role</Text>
                        <Text style={styles.value}>{profile?.role || 'N/A'}</Text>
                    </View>

                    <TouchableOpacity
                        style={styles.signOutButton}
                        onPress={handleSignOut}
                    >
                        <Text style={styles.signOutText}>Sign Out</Text>
                    </TouchableOpacity>
                </View>
            )}
        </View>
    );
};

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: colors.navy,
        padding: spacing.md,
    },
    header: {
        marginTop: spacing.xl,
        marginBottom: spacing.xl,
    },
    title: {
        ...typography.h1,
        color: colors.white,
    },
    content: {
        flex: 1,
    },
    loadingText: {
        ...typography.body,
        color: colors.slate,
        textAlign: 'center',
        marginTop: spacing.xl,
    },
    infoRow: {
        marginBottom: spacing.lg,
        paddingBottom: spacing.sm,
        borderBottomWidth: 1,
        borderBottomColor: 'rgba(255,255,255,0.1)',
    },
    label: {
        ...typography.caption,
        color: colors.slate,
        marginBottom: spacing.xs,
    },
    value: {
        ...typography.h3,
        color: colors.white,
    },
    signOutButton: {
        backgroundColor: colors.rose,
        padding: spacing.md,
        borderRadius: 8,
        alignItems: 'center',
        marginTop: spacing.xl,
    },
    signOutText: {
        ...typography.button,
        color: colors.white,
    },
});
