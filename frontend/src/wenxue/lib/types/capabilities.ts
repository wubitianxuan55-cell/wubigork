import type { WireShape } from "./shared";
import type { app as AppModels } from "../../../../wailsjs/go/models";

// MCP & Skills drawer (desktop/app.go Capabilities) — the GUI counterpart to
// /mcp + /skill: connected/failed servers and discoverable skills.
export interface ServerView {
  name: string;
  transport: string;
  status: "connected" | "failed" | "disabled";
  tools: number;
  prompts: number;
  resources: number;
  error?: string;
  toolList?: MCPToolView[];
}
export interface MCPToolView {
  name: string;
  description: string;
}
export type SkillView = WireShape<AppModels.SkillView>;
export interface CapabilitiesView {
  servers: ServerView[];
  skills: SkillView[];
}
export type MCPServerInput = WireShape<AppModels.MCPServerInput>;

export type ModelInfo = WireShape<AppModels.ModelInfo>;

// Slash sub-command / argument completion (desktop/app.go SlashArgs). Mirrors the
// CLI's arg hints so the composer can suggest e.g. /skill → list/show/new/paths.
export type SlashArgItem = WireShape<AppModels.SlashArgItem>;
export type SlashArgsResult = WireShape<AppModels.SlashArgsResult>;
