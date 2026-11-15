// mock/office/methods_websearch.ts — 搜索主源设置三绑定（众神殿刀3 主源向导）
// 的 dev/vitest mock：内存态往返 + 固定连通性回执（不出网）。
import type { OfficeMethods } from "./types";

interface MockSearchState {
  tavilyKey: string
  braveKey: string
  searxngURL: string
}

const state: MockSearchState = { tavilyKey: "", braveKey: "", searxngURL: "" }

export const webSearchMethods: Partial<
  Pick<OfficeMethods, "WebSearchSettingsGet" | "WebSearchSettingsSet" | "WebSearchTestConnection">
> = {
  WebSearchSettingsGet: async () => ({
    local_searxng_url: state.searxngURL,
    tavily_api_key_env: state.tavilyKey ? "TAVILY_API_KEY" : "",
    brave_api_key_env: state.braveKey ? "BRAVE_API_KEY" : "",
    tavily_ready: !!state.tavilyKey,
    brave_ready: !!state.braveKey,
    searxng_ready: !!state.searxngURL,
    has_primary: !!(state.tavilyKey || state.braveKey || state.searxngURL),
    engine_order: [],
  }),
  WebSearchSettingsSet: async (tavilyKey: string, braveKey: string, searxngURL: string) => {
    state.tavilyKey = (tavilyKey || "").trim()
    state.braveKey = (braveKey || "").trim()
    state.searxngURL = (searxngURL || "").trim()
  },
  WebSearchTestConnection: async () => ({
    ok: !!state.tavilyKey,
    state: state.tavilyKey ? "ok" : "all_failed",
    degraded_reason: "",
    elapsed_ms: 1200,
    engines: state.tavilyKey
      ? [{ name: "tavily", outcome: "ok", elapsed_ms: 1200 }]
      : [{ name: "bing", outcome: "error", elapsed_ms: 0, err: "mock: 未配置主源" }],
    sample: state.tavilyKey ? ["mock 样例结果（tavily）"] : [],
    error: state.tavilyKey ? "" : "所有搜索引擎失败",
    hint: state.tavilyKey ? "" : "在上方填入 Tavily/ Brave key 或自建 SearXNG 地址后保存重试；免费兜底（Bing）仍可用。",
  }),
}
