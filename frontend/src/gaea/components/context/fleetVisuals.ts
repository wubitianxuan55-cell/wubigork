const SUBAGENT_HUES = [
  "var(--gaea-chart-2)",
  "var(--gaea-chart-4)",
  "var(--gaea-chart-3)",
  "var(--gaea-chart-6)",
  "var(--gaea-chart-7)",
  "var(--gaea-chart-3)",
] as const;

function hash(value: string): number {
  let result = 0;
  for (let index = 0; index < value.length; index += 1) {
    result = (result * 31 + value.charCodeAt(index)) >>> 0;
  }
  return result;
}

export function agentHue(ref: string, index?: number): string {
  if (ref === "main" || ref === "root") {
    return "var(--fleet-violet, var(--info))";
  }
  if (index !== undefined && index >= 0) return SUBAGENT_HUES[index % SUBAGENT_HUES.length];
  return SUBAGENT_HUES[hash(ref) % SUBAGENT_HUES.length];
}

// 深度色带（dsh Tasks 页蒸馏：depth 决定卡片左缘颜色，同一层同色 → 宽图按
// 层读）。家系色保留在连线（家系=血缘维度，深度=层读维度，双维不打架）。
const DEPTH_HUES = [
  "var(--fleet-violet, var(--info))", // depth 0 root：与主 agent 家系色同源
  "var(--gaea-chart-4)",
  "var(--gaea-chart-7)",
  "var(--gaea-chart-3)",
  "var(--gaea-chart-6)",
  "var(--gaea-chart-2)",
] as const;

export function depthHue(depth: number): string {
  const index = Number.isFinite(depth) && depth > 0 ? Math.floor(depth) : 0;
  return DEPTH_HUES[index % DEPTH_HUES.length];
}
