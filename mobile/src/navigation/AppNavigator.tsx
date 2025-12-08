import React from 'react';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { createStackNavigator } from '@react-navigation/stack';
import DashboardScreen from '../screens/DashboardScreen';
import DriversScreen from '../screens/DriversScreen';
import RidersScreen from '../screens/RidersScreen';
import DriverRequestsScreen from '../screens/DriverRequestsScreen';
import DriverTripsScreen from '../screens/DriverTripsScreen';
import { ProfileScreen } from '../screens/ProfileScreen';
import { SignInScreen } from '../screens/SignInScreen';
import { SignUpScreen } from '../screens/SignUpScreen';
import { ForgotPasswordScreen } from '../screens/ForgotPasswordScreen';
import { Ionicons } from '@expo/vector-icons';
import { palette } from '../theme';
import { useAuth } from '../context/AuthContext';
import { RootStackParamList } from '../types';

const Tab = createBottomTabNavigator<RootStackParamList>();
const Stack = createStackNavigator<RootStackParamList>();

const TAB_ICONS: Record<string, keyof typeof Ionicons.glyphMap> = {
  Dashboard: 'compass-outline',
  Drivers: 'car-outline',
  Riders: 'people-outline',
  Profile: 'person-outline',
  DriverRequests: 'people-circle-outline',
  Trips: 'map-outline',
};

const tabScreenOptions = ({ route }: { route: { name: string } }) => ({
  headerShown: false,
  tabBarActiveTintColor: palette.mint,
  tabBarInactiveTintColor: '#6b7280',
  tabBarStyle: {
    backgroundColor: 'rgba(15,23,42,0.95)',
    borderTopColor: 'rgba(255,255,255,0.08)',
  },
  tabBarIcon: ({ color, size }: { color: string; size: number }) => {
    const iconName = TAB_ICONS[route.name] ?? 'ellipse-outline';
    return <Ionicons name={iconName} size={size} color={color} />;
  },
});

const RiderTabs = () => (
  <Tab.Navigator screenOptions={tabScreenOptions}>
    <Tab.Screen name="Dashboard" component={DashboardScreen} />
    <Tab.Screen name="Drivers" component={DriversScreen} />
    <Tab.Screen
      name="Riders"
      component={RidersScreen}
      options={{ title: 'Book Ride', tabBarLabel: 'Book' }}
    />
    <Tab.Screen name="Profile" component={ProfileScreen} />
  </Tab.Navigator>
);

const DriverTabs = () => (
  <Tab.Navigator screenOptions={tabScreenOptions}>
    <Tab.Screen name="Dashboard" component={DashboardScreen} />
    <Tab.Screen
      name="DriverRequests"
      component={DriverRequestsScreen}
      options={{ title: 'Riders', tabBarLabel: 'Riders' }}
    />
    <Tab.Screen name="Trips" component={DriverTripsScreen} />
    <Tab.Screen name="Profile" component={ProfileScreen} />
  </Tab.Navigator>
);

const AuthStack = () => (
  <Stack.Navigator screenOptions={{ headerShown: false }}>
    <Stack.Screen name="SignIn" component={SignInScreen} />
    <Stack.Screen name="SignUp" component={SignUpScreen} />
    <Stack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
  </Stack.Navigator>
);

const AppNavigator = () => {
  const { session, role } = useAuth();
  const isDriver = role === 'driver';

  if (!session) {
    return <AuthStack />;
  }

  return isDriver ? <DriverTabs /> : <RiderTabs />;
};

export default AppNavigator;
