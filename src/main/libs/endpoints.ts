import { app } from 'electron';

import type { SqliteStore } from '../sqliteStore';

let cachedTestMode: boolean | null = null;

const UPDATE_API_BASE_URL_ENV = 'LOBSTERAI_UPDATE_API_BASE_URL';
const AUTH_API_BASE_URL_ENV = 'LOBSTERAI_AUTH_API_BASE_URL';
const API_BASE_URL_ENV = 'LOBSTERAI_API_BASE_URL';

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

const AuthEndpointName = {
  LoginURL: 'login-url',
} as const;

type AuthEndpointName = typeof AuthEndpointName[keyof typeof AuthEndpointName];

const PORTAL_BASE_TEST = 'https://c.youdao.com/dict/hardware/cowork/lobsterai-portal.html#';
const PORTAL_BASE_PROD = 'https://c.youdao.com/dict/hardware/octopus/lobsterai-portal.html#';

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

const getUpdateApiEnvironment = (): UpdateApiEnvironment => (
  isTestMode() ? UpdateApiEnvironment.Test : UpdateApiEnvironment.Prod
);

const getOverrideBaseUrl = (envName: string, prefix: string): string | null => {
  const value = process.env[envName]?.trim();
  if (!value) {
    return null;
  }
  const normalized = value.replace(/\/+$/, '');
  console.log(`[${prefix}] using override base URL from ${envName}: ${normalized}`);
  return normalized;
};

const getUpdateApiOverrideBaseUrl = (): string | null => (
  getOverrideBaseUrl(UPDATE_API_BASE_URL_ENV, 'UpdateAPI')
);

const getAuthApiOverrideBaseUrl = (): string | null => {
  const authBaseUrl = getOverrideBaseUrl(AUTH_API_BASE_URL_ENV, 'AuthAPI');
  if (authBaseUrl) {
    return authBaseUrl;
  }

  const sharedApiBaseUrl = getOverrideBaseUrl(API_BASE_URL_ENV, 'AuthAPI');
  if (sharedApiBaseUrl) {
    console.log(`[AuthAPI] ${AUTH_API_BASE_URL_ENV} is not set, fallback to ${API_BASE_URL_ENV}`);
    return sharedApiBaseUrl;
  }

  const updateBaseUrl = getUpdateApiOverrideBaseUrl();
  if (updateBaseUrl) {
    console.log(`[AuthAPI] ${AUTH_API_BASE_URL_ENV} is not set, fallback to ${UPDATE_API_BASE_URL_ENV}`);
    return updateBaseUrl;
  }

  return null;
};

const getUpdateApiOverrideUrl = (endpointName: UpdateEndpointName): string | null => {
  const baseUrl = getUpdateApiOverrideBaseUrl();
  if (!baseUrl) {
    return null;
  }
  return `${baseUrl}/openapi/get/luna/hardware/lobsterai/${getUpdateApiEnvironment()}/${endpointName}`;
};

const getAuthApiOverrideUrl = (endpointName: AuthEndpointName): string | null => {
  const baseUrl = getAuthApiOverrideBaseUrl();
  if (!baseUrl) {
    return null;
  }
  return `${baseUrl}/openapi/get/luna/hardware/lobsterai/${getUpdateApiEnvironment()}/${endpointName}`;
};

/**
 * Server API base URL — switches based on testMode.
 * Used for auth exchange/refresh, models, proxy, etc.
 */
export const getServerApiBaseUrl = (): string => (
  getAuthApiOverrideBaseUrl()
  ?? (isTestMode()
    ? 'https://lobsterai-server.inner.youdao.com'
    : 'https://lobsterai-server.youdao.com')
);

const getPortalBase = (): string => (
  isTestMode() ? PORTAL_BASE_TEST : PORTAL_BASE_PROD
);

export const getPortalLoginUrl = (): string => `${getPortalBase()}/login`;

export const getLoginOvermindUrl = (): string => (
  getAuthApiOverrideUrl(AuthEndpointName.LoginURL)
  ?? (isTestMode()
    ? 'https://api-overmind.youdao.com/openapi/get/luna/hardware/lobsterai/test/login-url'
    : 'https://api-overmind.youdao.com/openapi/get/luna/hardware/lobsterai/prod/login-url')
);

export const getMockLoginUrl = (): string | null => {
  const baseUrl = getAuthApiOverrideBaseUrl();
  if (!baseUrl) {
    return null;
  }

  return `${baseUrl}/mock-login?env=${getUpdateApiEnvironment()}`;
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
