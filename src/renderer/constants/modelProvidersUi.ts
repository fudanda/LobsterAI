import { ProviderName } from '@shared/providers';

/**
 * Model settings tab — provider list chrome. Set true to restore import/export and “add custom provider”.
 */
export const ModelProvidersUi = {
  ShowImportExport: false,
  ShowAddCustomProvider: false,
  /** When true, model sidebar lists only LZClaw (no other built-ins, no custom_* rows). */
  OnlyLzclaw: true,
} as const;

/** Single provider shown when `OnlyLzclaw` is true — must match `ProviderName`. */
export const ModelSettingsSoloProvider = ProviderName.Lzclaw;