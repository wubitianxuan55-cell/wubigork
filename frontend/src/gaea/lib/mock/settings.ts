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
  };
}
