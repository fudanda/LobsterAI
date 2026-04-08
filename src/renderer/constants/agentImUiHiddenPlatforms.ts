/**
 * IM platforms hidden from Agent create/edit modals (IM channel binding tab).
 * Matches product choice to hide QQ, NetEase IM (云信), and POPO there; restore by removing ids from the set.
 */
export const AgentImUiHiddenPlatform = {
  Qq: 'qq',
  Nim: 'nim',
  Popo: 'popo',
} as const;

export const agentImUiHiddenPlatformSet: ReadonlySet<string> = new Set<string>([
  AgentImUiHiddenPlatform.Qq,
  AgentImUiHiddenPlatform.Nim,
  AgentImUiHiddenPlatform.Popo,
]);
