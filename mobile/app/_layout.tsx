import React from 'react';
import { Stack } from 'expo-router';
import { ThemeProvider, DefaultTheme, Theme } from '@react-navigation/native';
import { palette } from '../src/theme';

const navTheme: Theme = {
  ...DefaultTheme,
  colors: {
    ...DefaultTheme.colors,
    background: palette.navy,
    card: palette.navy,
    text: palette.white,
    border: 'rgba(255,255,255,0.08)',
    primary: palette.mint,
  },
};

/**
 * Root layout for expo-router. We delegate actual UI to `app/index.tsx`,
 * so this stack only suppresses the default headers.
 */
export default function RootLayout() {
  return (
    <ThemeProvider value={navTheme}>
      <Stack screenOptions={{ headerShown: false }} />
    </ThemeProvider>
  );
}
