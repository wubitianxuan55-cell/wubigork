import type { WireShape } from "./shared";
import type { app as AppModels } from "../../../../wailsjs/go/models";

// WhisperMemoryView 聊天（hermes.db）记忆事实只读视图。
export type WhisperMemoryView = WireShape<AppModels.WhisperMemoryView>;

// WhisperEpisodeView 聊天（hermes.db）情节记忆只读视图（时间倒序）。
export type WhisperEpisodeView = WireShape<AppModels.WhisperEpisodeView>;

// WhisperReplayLine 记忆回放中的一行原始对话。
export interface WhisperReplayLine {
  turnIndex: number;
  role: string;
  text: string;
}

// WhisperEpisodeReplayView 情节记忆回放视图：情节元数据 + 原始对话（只读）。
export interface WhisperEpisodeReplayView {
  id: string;
  summary: string;
  dominantEmotion: string;
  emotionalIntensity: number;
  keywords: string[];
  createdAt: string;
  sourceSessionId: string;
  startTurn: number;
  endTurn: number;
  dialogue: WhisperReplayLine[];
  replayable: boolean;
}

// WhisperAnchorView 轻语时间锚点只读视图（play 空间「纪念日」）。
export interface WhisperAnchorView {
  id: string;
  anchorDate: string;
  anchorType: string;
  recurrenceRule: string;
  domain: string;
  summary: string;
  emotionalValence: number;
  emotionalIntensity: number;
  linkedFactIds: string[];
}

// WhisperAnchorReplayView 时间锚点回放视图：锚点 + 关联事实摘要 + 情节回放。
export interface WhisperAnchorReplayView {
  anchorId: string;
  anchorDate: string;
  anchorType: string;
  domain: string;
  summary: string;
  emotionalValence: number;
  emotionalIntensity: number;
  linkedFactSummaries: string[];
  episodeReplay?: WhisperEpisodeReplayView;
  replayable: boolean;
}
// ── v4.3 乐园做深（docs/gaea-v43-play-deepen-design.md）────────────────
// 轻语关系图谱子图（whisper.Subgraph：实体=三元组 Subject/Object，边=Predicate）。
export interface WhisperGraphNode {
  id: string;
  name: string;
  type: string;
  weight: number;
}
export interface WhisperGraphEdge {
  from: string;
  to: string;
  type: string;
  weight: number;
  // v4.9 图谱情绪维度：正面/负面/中性（后端按事实效价派生，空=中性）。
  emotionLabel?: string;
}
export interface WhisperSubgraph {
  nodes: WhisperGraphNode[];
  edges: WhisperGraphEdge[];
}
// WhisperProactiveNow 返回：主动关心评估结果（shouldSend=false 时不发）。
export interface WhisperProactiveResult {
  shouldSend: boolean;
  messageType?: string; // check_in/miss_you/time_aware/playful_nudge/birthday/...
  promptHint?: string;
}
// WhisperProactiveConfig 返回：主动关心定时推送配置（v4.3c 后续小步）。
export interface WhisperProactiveConfigView {
  enabled: boolean; // 定时推送总开关
  limitPerHour: number; // 每小时主动消息上限（AttentionManager 频控）
  intervalMin: number; // 评估间隔分钟（10-120）
  quietStartHour: number; // 免打扰时窗开始小时（-1=未启用）
  quietEndHour: number; // 免打扰时窗结束小时（-1=未启用）
}
// TTS 风格/情绪参数（tts.TTSParams，零值=引擎默认）。
export interface TTSParams {
  speed: number; // 倍速 0.5-2.0
  pitch: number; // 音高偏移半音 -12..12
  style: string;
  emotion: string;
}
