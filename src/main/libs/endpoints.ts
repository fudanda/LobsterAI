import { app } from 'electron';

import type { SqliteStore } from '../sqliteStore';

let cachedTestMode: boolean | null = null;

const UPDATE_API_BASE_URL_ENV = 'LOBSTERAI_UPDATE_API_BASE_URL';

const UpdateApiEnvironment = {
  Test: 'test',
  Prod: 'prod',
} as const;

type UpdateApiEnvironment = typeof UpdateApiEnvironment[keyof typeof UpdateApiEnvironment];

const UpdateEndpointName = {
  Auto: 'update',
  Manual: 'update-manual',
} as const;

type UpdateEndpointName = typeof UpdateEndpointName[keyof typeof UpdateEndpointName];

type AppConfigStoreValue = {
  app?: {
    testMode?: boolean;
  };
};

/**
 * Read testMode from store and cache it.
 * Call once at startup and again whenever app_config changes.
 */
export function refreshEndpointsTestMode(store: SqliteStore): void {
  const appConfig = store.get<AppConfigStoreValue>('app_config');
  cachedTestMode = appConfig?.app?.testMode === true;
}

/**
 * Whether the app is in test mode.
 * Uses cached value after init; falls back to !app.isPackaged before init.
 */
const isTestMode = (): boolean => {
  return cachedTestMode ?? !app.isPackaged;
};

/**
 * Server API base URL — switches based on testMode.
 * Used for auth exchange/refresh, models, proxy, etc.
 */
export const getServerApiBaseUrl = (): string => {
  return isTestMode()
    ? 'https://lobsterai-server.inner.youdao.com'
    : 'https://lobsterai-server.youdao.com';
};

const getUpdateApiEnvironment = (): UpdateApiEnvironment => (
  isTestMode() ? UpdateApiEnvironment.Test : UpdateApiEnvironment.Prod
);

const getUpdateApiOverrideBaseUrl = (): string | null => {
  const value = process.env[UPDATE_API_BASE_URL_ENV]?.trim();
  if (!value) {
    return null;
  }
  const normalized = value.replace(/\/+$/, '');
  console.log(`[UpdateAPI] using override base URL from ${UPDATE_API_BASE_URL_ENV}: ${normalized}`);
  return normalized;
};

const getUpdateApiOverrideUrl = (endpointName: UpdateEndpointName): string | null => {
  const baseUrl = getUpdateApiOverrideBaseUrl();
  if (!baseUrl) {
    return null;
  }
  return `${baseUrl}/openapi/get/luna/hardware/lobsterai/${getUpdateApiEnvironment()}/${endpointName}`;
};

export const getUpdateCheckUrl = (): string => (
  getUpdateApiOverrideUrl(UpdateEndpointName.Auto)
  ?? (isTestMode()
    ? 'https://api-overmind.youdao.com/openapi/get/luna/hardware/lobsterai/test/update'
    : 'https://api-overmind.youdao.com/openapi/get/luna/hardware/lobsterai/prod/update')
);

export const getManualUpdateCheckUrl = (): string => (
  getUpdateApiOverrideUrl(UpdateEndpointName.Manual)
  ?? (isTestMode()
    ? 'https://api-overmind.youdao.com/openapi/get/luna/hardware/lobsterai/test/update-manual'
    : 'https://api-overmind.youdao.com/openapi/get/luna/hardware/lobsterai/prod/update-manual')
);

export const getFallbackDownloadUrl = (): string => (
  isTestMode()
    ? 'https://lobsterai.inner.youdao.com/#/download-list'
    : 'https://lobsterai.youdao.com/#/download-list'
);

export const getSkillStoreUrl = (): string => (
  isTestMode()
    ? 'https://api-overmind.youdao.com/openapi/get/luna/hardware/lobsterai/test/skill-store'
    : 'https://api-overmind.youdao.com/openapi/get/luna/hardware/lobsterai/prod/skill-store'
);
