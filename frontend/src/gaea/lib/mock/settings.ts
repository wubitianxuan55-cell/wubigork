// mock/settings.ts — 设置域（T6-10.1 拆分自 lib/mock.ts，方法体零改动）。
// settings 对象引用全程不变（只做属性赋值），解构快照仍实时可见。
import type { AppBindings } from "../bridge";
import type { ProviderView, SettingsView } from "../types";
import type { MakeMockState } from "./state";

type SettingsMethods = Pick<
  AppBindings,
  | "Settings" | "SetDefaultModel" | "SaveProvider" | "DeleteProvider" | "SetProviderKey"
  | "LoginProvider" | "LogoutProvider"
  | "SetPermissionMode" | "AddPermissionRule" | "RemovePermissionRule"
  | "SetSandbox" | "SetAgentParams"
  | "SetSubagentModel" | "SetSubagentModelForSkill" | "SetSubagentTemperature"
  | "SetEffort" | "SetSubagentEffort" | "SetPermLevel"
  // v4.171 批次一 legacy 直调转正（Go OfficeB.GaeaSaveSettings）：设置整体写回。
  | "SaveSettings"
>;

export function buildSettings(s: MakeMockState): SettingsMethods {
  const { settings } = s;
  return {
    async Settings() {
      return JSON.parse(JSON.stringify(settings)) as SettingsView;
    },
    async SetDefaultModel(ref: string) {
      settings.defaultModel = ref;
    },
    async SaveProvider(p: ProviderView) {
      const i = settings.providers.findIndex((x) => x.name === p.name);
      if (i >= 0) settings.providers[i] = p;
      else settings.providers.push(p);
    },
    async DeleteProvider(name: string) {
      settings.providers = settings.providers.filter((p) => p.name !== name);
    },
    async SetProviderKey(apiKeyEnv: string) {
      settings.providers.forEach((p) => {
        if (p.apiKeyEnv === apiKeyEnv) p.keySet = true;
      });
    },
    async LoginProvider(name: string) {
      const p = settings.providers.find((x) => x.name === name);
      if (p) p.oauthReady = true;
    },
    async LogoutProvider(name: string) {
      const p = settings.providers.find((x) => x.name === name);
      if (p) p.oauthReady = false;
    },
    async SetPermissionMode(mode: string) {
      settings.permissions.mode = mode;
    },
    async AddPermissionRule(list: string, rule: string) {
      const k = list as "allow" | "ask" | "deny";
      if (settings.permissions[k] && !settings.permissions[k].includes(rule)) settings.permissions[k].push(rule);
    },
    async RemovePermissionRule(list: string, rule: string) {
      const k = list as "allow" | "ask" | "deny";
      settings.permissions[k] = settings.permissions[k].filter((r) => r !== rule);
    },
    async SetSandbox(bash: string, network: boolean, workspaceRoot: string, allowWrite: string[]) {
      settings.sandbox = { bash, network, workspaceRoot, allowWrite };
    },
    async SetAgentParams(temperature: number, maxSteps: number, systemPrompt: string) {
      settings.agent = { ...settings.agent, temperature, maxSteps, systemPrompt };
    },
    async SetSubagentModel(ref: string) {
      settings.subagentModel = ref;
    },
    async SetSubagentModelForSkill(_skill: string, ref: string) {
      if (!settings.subagentModels) settings.subagentModels = {};
      settings.subagentModels[_skill] = ref;
    },
    async SetSubagentTemperature(temp: number) {
      settings.agent.subagentTemperature = temp;
    },
    async SetEffort(effort: string) {
      settings.agent.effort = effort;
    },
    async SetSubagentEffort(effort: string) {
      settings.agent.subagentEffort = effort;
    },
    async SetPermLevel(level: string) {
      settings.permLevel = level;
    },
    async SaveSettings(view: SettingsView) {
      // 契约对齐 Go GaeaSaveSettings（整体写回设置视图；v4.171 批次一转正）。
      // mock 合并进内存态 settings（同一引用，Settings() 立即可读回）：
      // agent/permissions/sandbox 与后端同构的部分更新合并，其余字段按显式赋值。
      if (view.defaultModel !== undefined) settings.defaultModel = view.defaultModel;
      if (view.subagentModel !== undefined) settings.subagentModel = view.subagentModel;
      if (view.agent) settings.agent = { ...settings.agent, ...view.agent };
      if (view.permissions) {
        settings.permissions = {
          ...settings.permissions,
          ...view.permissions,
          allow: view.permissions.allow ?? settings.permissions.allow,
          ask: view.permissions.ask ?? settings.permissions.ask,
          deny: view.permissions.deny ?? settings.permissions.deny,
        };
      }
      if (view.sandbox) settings.sandbox = { ...settings.sandbox, ...view.sandbox };
    },
  };
}
