const fs = require('fs');
const path = require('path');

// This script ensures pnpm's virtual store path for expo-router exists under
// mobile/node_modules/.pnpm so Metro can resolve `expo-router/entry`.

function ensureSymlink() {
  let entryPath;
  try {
    entryPath = require.resolve('expo-router/entry');
  } catch (err) {
    console.warn('[fix-expo-router] expo-router not installed yet; skipping');
    return;
  }

  const match = entryPath.match(/node_modules\/\.pnpm\/([^/]+)\/node_modules\/expo-router\/entry\.js$/);
  if (!match) {
    console.warn('[fix-expo-router] Unexpected expo-router path:', entryPath);
    return;
  }

  const storeFolder = match[1];
  const storeRoot = path.join(path.dirname(path.dirname(entryPath)), '..', '..'); // .../node_modules/.pnpm
  const targetExpoRouter = path.join(storeRoot, storeFolder, 'node_modules', 'expo-router');

  const linkDir = path.join(__dirname, '..', 'node_modules', '.pnpm', storeFolder, 'node_modules');
  const linkPath = path.join(linkDir, 'expo-router');

  fs.mkdirSync(linkDir, { recursive: true });

  fs.rmSync(linkPath, { force: true, recursive: true });

  fs.symlinkSync(path.relative(linkDir, targetExpoRouter), linkPath, 'dir');
  console.log('[fix-expo-router] linked', linkPath, '->', targetExpoRouter);
}

ensureSymlink();
