import { useEffect } from 'react';
import * as Notifications from 'expo-notifications';
import * as Device from 'expo-device';
import { Platform } from 'react-native';
import Constants from 'expo-constants';
import { useAuth } from '../context/AuthContext';
import { gateway } from '../services/gateway';

const notificationBehavior: Notifications.NotificationBehavior = {
  shouldShowAlert: true,
  shouldShowBanner: true,
  shouldShowList: true,
  shouldPlaySound: true,
  shouldSetBadge: false,
};

Notifications.setNotificationHandler({
  handleNotification: async () => ({ ...notificationBehavior }),
});

export function usePushNotifications() {
  const { user } = useAuth();
  const userId = user?.id;

  useEffect(() => {
    if (!userId) {
      return;
    }

    const stableUserId = userId;

    let cancelled = false;

    async function register() {
      if (!Device.isDevice) {
        await Notifications.requestPermissionsAsync({
          ios: { allowAlert: true, allowBadge: true, allowSound: true },
        });
        return;
      }
      const settings = await Notifications.getPermissionsAsync();
      let granted = settings.granted || settings.ios?.status === Notifications.IosAuthorizationStatus.PROVISIONAL;
      if (!granted) {
        const request = await Notifications.requestPermissionsAsync({
          ios: {
            allowAlert: true,
            allowBadge: true,
            allowSound: true,
          },
        });
        granted = request.granted || request.ios?.status === Notifications.IosAuthorizationStatus.PROVISIONAL;
      }
      if (!granted || cancelled) {
        return;
      }
      if (Platform.OS === 'android') {
        await Notifications.setNotificationChannelAsync('default', {
          name: 'default',
          importance: Notifications.AndroidImportance.MAX,
        });
      }
      const projectId = Constants?.expoConfig?.extra?.eas?.projectId ?? Constants?.easConfig?.projectId;
      const tokenResponse = await Notifications.getExpoPushTokenAsync(projectId ? { projectId } : undefined);
      if (!tokenResponse.data || cancelled) {
        return;
      }
      try {
        await gateway.registerPushToken(stableUserId, tokenResponse.data);
      } catch (err) {
        console.warn('registerPushToken failed', err);
      }
    }

    register();

    return () => {
      cancelled = true;
    };
  }, [userId]);
}
