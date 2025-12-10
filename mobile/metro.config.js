const path = require('path');
const { getDefaultConfig } = require('expo/metro-config');

const projectRoot = __dirname;
const workspaceRoot = path.resolve(projectRoot, '..');

const config = getDefaultConfig(projectRoot);

// Ensure Metro follows pnpm symlinks that point to the workspace root.
config.watchFolders = [workspaceRoot];
config.resolver.nodeModulesPaths = [
  path.join(projectRoot, 'node_modules'),
  path.join(workspaceRoot, 'node_modules'),
];

// Always resolve shared deps such as react/react-native from the workspace root
// so Metro never loads two copies (which triggers "Invalid hook call").
config.resolver.extraNodeModules = {
  ...config.resolver.extraNodeModules,
  react: path.join(workspaceRoot, 'node_modules', 'react'),
  'react-native': path.join(workspaceRoot, 'node_modules', 'react-native'),
};
config.resolver.disableHierarchicalLookup = true;

// pnpm installs dependencies as symlinks inside a virtual store. These flags
// let Metro follow the symlinks and respect package "exports" fields (needed
// by expo-router).
config.resolver.unstable_enableSymlinks = true;
config.resolver.unstable_enablePackageExports = true;

module.exports = config;
