const SUBAGENT_HUES = [
  "hsl(146 48% 50%)",
  "hsl(211 70% 60%)",
  "hsl(45 79% 51%)",
  "hsl(320 67% 63%)",
  "hsl(184 62% 44%)",
  "hsl(20 76% 57%)",
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
