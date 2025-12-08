import React, { useState } from 'react';
import { View, Text, TextInput, TouchableOpacity, StyleSheet, Alert, ActivityIndicator } from 'react-native';
import { BlurView } from 'expo-blur';
import { LinearGradient } from 'expo-linear-gradient';
import { gateway } from '../services/gateway';

import { useAuth } from '../context/AuthContext';

interface SignInScreenProps {
    navigation: any;
}

export const SignInScreen: React.FC<SignInScreenProps> = ({ navigation }) => {
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [intendedRole, setIntendedRole] = useState<'driver' | 'rider'>('rider');
    const [loading, setLoading] = useState(false);
    const { signIn } = useAuth();

    const handleSignIn = async () => {
        if (!email || !password) {
            Alert.alert('Error', 'Please enter both email and password');
            return;
        }

        setLoading(true);
        try {
            const response = await gateway.signIn({ email, password });
            const patchedUser = {
                ...response.user,
                role: intendedRole,
                user_metadata: {
                    ...(response.user?.user_metadata || {}),
                    role: intendedRole,
                },
            };
            signIn(response.access_token, patchedUser);
        } catch (error: any) {
            Alert.alert('Login Failed', error.message || 'Something went wrong');
        } finally {
            setLoading(false);
        }
    };

    return (
        <View style={styles.container}>
            <LinearGradient
                colors={['#1a1a1a', '#000000']}
                style={styles.background}
            />

            <View style={styles.content}>
                <Text style={styles.title}>LastMile</Text>
                <Text style={styles.subtitle}>Sign in to continue</Text>

                <BlurView intensity={20} tint="dark" style={styles.card}>
                    <TextInput
                        style={styles.input}
                        placeholder="Email"
                        placeholderTextColor="#666"
                        value={email}
                        onChangeText={setEmail}
                        autoCapitalize="none"
                        keyboardType="email-address"
                    />

                    <TextInput
                        style={styles.input}
                        placeholder="Password"
                        placeholderTextColor="#666"
                        value={password}
                        onChangeText={setPassword}
                        secureTextEntry
                    />

                    <View style={styles.roleToggleRow}>
                        <Text style={styles.roleLabel}>Log in as</Text>
                        <View style={styles.roleToggle}>
                            <TouchableOpacity
                                style={[
                                    styles.roleButton,
                                    intendedRole === 'rider' && styles.roleButtonActive,
                                ]}
                                onPress={() => setIntendedRole('rider')}
                            >
                                <Text
                                    style={[
                                        styles.roleButtonText,
                                        intendedRole === 'rider' && styles.roleButtonTextActive,
                                    ]}
                                >
                                    Rider
                                </Text>
                            </TouchableOpacity>
                            <TouchableOpacity
                                style={[
                                    styles.roleButton,
                                    intendedRole === 'driver' && styles.roleButtonActive,
                                ]}
                                onPress={() => setIntendedRole('driver')}
                            >
                                <Text
                                    style={[
                                        styles.roleButtonText,
                                        intendedRole === 'driver' && styles.roleButtonTextActive,
                                    ]}
                                >
                                    Driver
                                </Text>
                            </TouchableOpacity>
                        </View>
                    </View>

                    <TouchableOpacity
                        style={styles.button}
                        onPress={handleSignIn}
                        disabled={loading}
                    >
                        {loading ? (
                            <ActivityIndicator color="#fff" />
                        ) : (
                            <Text style={styles.buttonText}>Sign In</Text>
                        )}
                    </TouchableOpacity>

                    <TouchableOpacity onPress={() => navigation.navigate('ForgotPassword')}>
                        <Text style={styles.linkText}>Forgot Password?</Text>
                    </TouchableOpacity>
                </BlurView>

                <View style={styles.footer}>
                    <Text style={styles.footerText}>Don't have an account? </Text>
                    <TouchableOpacity onPress={() => navigation.navigate('SignUp')}>
                        <Text style={styles.linkTextBold}>Sign Up</Text>
                    </TouchableOpacity>
                </View>
            </View>
        </View>
    );
};

const styles = StyleSheet.create({
    container: {
        flex: 1,
        justifyContent: 'center',
    },
    background: {
        position: 'absolute',
        left: 0,
        right: 0,
        top: 0,
        bottom: 0,
    },
    content: {
        padding: 24,
    },
    title: {
        fontSize: 40,
        fontWeight: 'bold',
        color: '#fff',
        textAlign: 'center',
        marginBottom: 8,
    },
    subtitle: {
        fontSize: 16,
        color: '#999',
        textAlign: 'center',
        marginBottom: 40,
    },
    card: {
        borderRadius: 24,
        padding: 24,
        overflow: 'hidden',
        backgroundColor: 'rgba(255,255,255,0.05)',
        borderWidth: 1,
        borderColor: 'rgba(255,255,255,0.1)',
    },
    input: {
        backgroundColor: 'rgba(0,0,0,0.3)',
        borderRadius: 12,
        padding: 16,
        color: '#fff',
        fontSize: 16,
        marginBottom: 16,
        borderWidth: 1,
        borderColor: 'rgba(255,255,255,0.1)',
    },
    roleToggleRow: {
        marginBottom: 16,
    },
    roleLabel: {
        color: '#ccc',
        fontSize: 14,
        marginBottom: 8,
    },
    roleToggle: {
        flexDirection: 'row',
        backgroundColor: 'rgba(255,255,255,0.06)',
        borderRadius: 12,
        padding: 4,
        borderWidth: 1,
        borderColor: 'rgba(255,255,255,0.08)',
    },
    roleButton: {
        flex: 1,
        paddingVertical: 10,
        borderRadius: 8,
        alignItems: 'center',
    },
    roleButtonActive: {
        backgroundColor: '#fff',
    },
    roleButtonText: {
        color: '#94a3b8',
        fontWeight: '600',
    },
    roleButtonTextActive: {
        color: '#0f172a',
    },
    button: {
        backgroundColor: '#fff',
        borderRadius: 12,
        padding: 16,
        alignItems: 'center',
        marginTop: 8,
        marginBottom: 16,
    },
    buttonText: {
        color: '#000',
        fontSize: 16,
        fontWeight: 'bold',
    },
    linkText: {
        color: '#999',
        textAlign: 'center',
        fontSize: 14,
    },
    footer: {
        flexDirection: 'row',
        justifyContent: 'center',
        marginTop: 24,
    },
    footerText: {
        color: '#666',
        fontSize: 14,
    },
    linkTextBold: {
        color: '#fff',
        fontSize: 14,
        fontWeight: 'bold',
    },
});
