# iOS Tasks Snapshot (Nov 23, 2025)

## Completed
- Switched Expo entrypoint back to `expo-router/entry` and removed the temporary `AppEntry.tsx` shim; Expo Router now matches the standard setup.
- Added pnpm/monorepo-friendly Metro settings (`watchFolders`, `nodeModulesPaths`) so symlinked deps like `expo-router` resolve correctly, enabled Metro symlink + package-exports support, added a local `mobile/node_modules/.pnpm/expo-router@6.0.15_.../node_modules/expo-router` symlink pointing to the root store so Metro can find `expo-router/entry`, and wired a `postinstall` helper (`scripts/fix-expo-router.js`) to recreate it after each install.
- Stopped the nested NavigationContainer crash by removing the extra container from `mobile/src/App.tsx`; theming now lives in `app/_layout.tsx` via `ThemeProvider`, so only expo-router owns the navigation container.
- UI polish: safe-area top padding on all tab screens (avoid notch/dynamic island overlap) and console logs for refresh/match actions to debug in Metro.
- Set `EXPO_USE_METRO_WORKSPACE_ROOT=1` in `start`/`ios` scripts and cleared Metro cache by default to avoid stale resolver state.
- Cleaned up Babel config (removed deprecated `expo-router/babel` plugin warning).
- Wired Expo project to EAS: set `extra.eas.projectId` to `b7b5eda9-9eec-4f72-8540-33f031402806` after `eas init`.

## Pending
- Run `pnpm install` (root and mobile) to refresh node_modules after the entrypoint/Metro changes, then restart with `pnpm start` and `pnpm ios`.
- If Metro still cannot resolve, clear caches manually: `pnpm exec expo start --clear` and delete `mobile/.expo`.
- Pods remain to be installed with network access (`cd mobile/ios && pod install --repo-update`), then rebuild.
- Verify the app no longer shows a black screen once the bundler resolves `expo-router/entry`.
