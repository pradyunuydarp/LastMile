import 'react-native-gesture-handler';
import React from 'react';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import AppNavigator from './navigation/AppNavigator';
import { BackendProvider } from './services/BackendProvider';
import { AuthProvider } from './context/AuthContext';
import { StatusBar } from 'expo-status-bar';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { RealtimeProvider } from './context/RealtimeContext';
import { usePushNotifications } from './hooks/usePushNotifications';

const App = () => (
  <GestureHandlerRootView style={{ flex: 1 }}>
    <SafeAreaProvider>
      <AuthProvider>
        <BackendProvider>
          <RealtimeProvider>
            <AppShell />
          </RealtimeProvider>
        </BackendProvider>
      </AuthProvider>
    </SafeAreaProvider>
  </GestureHandlerRootView>
);

const AppShell = () => {
  usePushNotifications();
  return (
    <>
      <StatusBar style="light" />
      <AppNavigator />
    </>
  );
};

export default App;
