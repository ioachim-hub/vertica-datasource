import fs from 'fs';
import { glob } from 'glob';
import os from 'os';
import path from 'path';
import process from 'process';

import { SOURCE_DIR } from './constants.ts';

export function isWSL() {
  if (process.platform !== 'linux') {
    return false;
  }

  if (os.release().toLowerCase().includes('microsoft')) {
    return true;
  }

  try {
    return fs.readFileSync('/proc/version', 'utf8').toLowerCase().includes('microsoft');
  } catch {
    return false;
  }
}

function loadJson(filePath: string) {
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

export function getPackageJson() {
  return loadJson(path.resolve(process.cwd(), 'package.json'));
}

export function getPluginJson() {
  return loadJson(path.resolve(process.cwd(), SOURCE_DIR, 'plugin.json'));
}

export function getCPConfigVersion() {
  const configPath = path.resolve(process.cwd(), '.config', '.cprc.json');
  return fs.existsSync(configPath) ? loadJson(configPath).version : 'unknown';
}

export function hasReadme() {
  return fs.existsSync(path.resolve(process.cwd(), SOURCE_DIR, 'README.md'));
}

export async function getEntries() {
  const pluginJsonFiles = await glob('**/src/**/plugin.json', { absolute: true });
  const modules = await Promise.all(
    pluginJsonFiles.map((pluginJsonFile) => {
      return glob(`${path.dirname(pluginJsonFile)}/module.{ts,tsx,js,jsx}`, { absolute: true });
    })
  );

  return modules.reduce<Record<string, string>>((result, modulePaths) => {
    return modulePaths.reduce((innerResult, modulePath) => {
      const pluginPath = path.dirname(modulePath);
      const pluginName = path.relative(process.cwd(), pluginPath).replace(/src\/?/i, '');
      const entryName = pluginName === '' ? 'module' : `${pluginName}/module`;

      innerResult[entryName] = modulePath;
      return innerResult;
    }, result);
  }, {});
}
