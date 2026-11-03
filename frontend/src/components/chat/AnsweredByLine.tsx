/* eslint-disable react-refresh/only-export-components -- AnsweredByInfo 类型与 answeredBySourceLabel/answeredByCostText 工具导出供 ChatRow/useChatStream 与测试复用 */
/**
 * 消息级「由谁回答 / 为何 / 花了多少」回显（v4.15，前端）。
 *
 * chat-stream:<runID> 事件的 done 帧 / ChatSend 返回新增可选字段 answered_by
 * （旧事件/旧消息无此字段时前端静默跳过，向后兼容），形如：
 *   { engine, model, source, cost_cny, tokens_per_second? }
 *  - engine / model：实际回答的引擎与模型
 *  - source：「为何」—— feature | global | fallback（v4.15 已砍自动路由，
 *    chat 链路只会出现这三种；未知值原样兜底展示）
 *  - cost_cny：估算费用（人民币；本地引擎 = 0，不虚报费用）
 *  - tokens_per_second：生成速度 tok/s（后端按流结束块 usage 的真实
 *    completion tokens ÷ 实测流式耗时计算；缺省/0 = 不显示该段）
 */
export interface AnsweredByInfo {
  engine: string;
  model: string;
  /** feature | global | fallback（未知值原样兜底展示） */
  source: string;
  /** 估算费用，人民币；本地引擎为 0 */
  cost_cny: number;
  /** 生成速度 tok/s（可选；旧事件无此字段 → 静默跳过） */
  tokens_per_second?: number;
  /** 本次请求的 token、首字与生成耗时；字段缺省时不虚报 */
  metrics?: AnswerMetrics;
}

export interface AnswerMetrics {
  prompt_tokens?: number;
  completion_tokens?: number;
  total_tokens?: number;
  cache_hit_tokens?: number;
  cache_miss_tokens?: number;
  first_token_ms?: number;
  prefill_ms?: number;
  decode_ms?: number;
  duration_ms?: number;
  prefill_tps?: number;
  completion_tps?: number;
  tokens_per_second?: number;
}

/**
 * 「为何」标签本地映射（文案风格对齐 pages/modelcenter/utils.tsx 的
 * routeSourceLabel；zh 硬编码，不走 locales）。
 */
export function answeredBySourceLabel(source: string): string {
  switch (source) {
    case 'feature':
      return '功能绑定';
    case 'global':
      return '全局路由';
    case 'fallback':
      return '兜底';
    default:
      return source.length > 0 ? source : '未知';
  }
}

/**
 * 费用段文案：cost_cny > 0 时返回「约 ¥0.01」（toFixed(2)）；
 * <= 0（本地引擎/未知费用）返回 null —— 隐藏「约 ¥」段，不虚报费用。
 */
export function answeredByCostText(costCny: number): string | null {
  if (typeof costCny !== 'number' || !Number.isFinite(costCny) || costCny <= 0) {
    return null;
  }
  return `约 ¥${costCny.toFixed(2)}`;
}

/**
 * 生成速度段文案：tokens_per_second > 0 时返回「12.3 tok/s」（toFixed(1)）；
 * 缺省/非有限数/<= 0 返回 null —— 隐藏速度段，不虚报。
 */
export function answeredBySpeedText(tps?: number): string | null {
  if (typeof tps !== 'number' || !Number.isFinite(tps) || tps <= 0) {
    return null;
  }
  return `${tps.toFixed(1)} tok/s`;
}

function formatCount(value?: number): string | null {
  return typeof value === 'number' && Number.isFinite(value)
    ? Math.max(0, Math.round(value)).toLocaleString()
    : null;
}

function formatDuration(ms?: number): string | null {
  if (typeof ms !== 'number' || !Number.isFinite(ms) || ms < 0) return null;
  if (ms < 1000) return `${Math.round(ms)} ms`;
  return `${(ms / 1000).toFixed(ms >= 10_000 ? 1 : 2)} s`;
}

function formatRate(tps?: number): string | null {
  if (typeof tps !== 'number' || !Number.isFinite(tps) || tps <= 0) return null;
  return `${tps.toFixed(tps >= 100 ? 0 : 1)} t/s`;
}

/**
 * 消息底部一行小字（仅当 extra.answered_by 存在时由消息行组件渲染）：
 * 「由 {engine}/{model} 回答 · {sourceLabel}[ · 约 ¥{cost}][ · {tok/s}]」
 * 费用 <= 0（本地/未知）时省略费用段；速度缺省时省略速度段。
 */
function AnsweredByLine({ info }: { info: AnsweredByInfo }) {
  const label = answeredBySourceLabel(info.source);
  const costText = answeredByCostText(info.cost_cny);
  const speedText = answeredBySpeedText(info.tokens_per_second);
  const segments = [`由 ${info.engine}/${info.model} 回答`, label];
  if (costText) {
    segments.push(costText);
  }
  if (speedText) {
    segments.push(speedText);
  }
  const metrics = info.metrics;
  const metricItems: Array<[string, string | null]> = metrics ? [
    ['输入', formatCount(metrics.prompt_tokens)],
    ['输出', formatCount(metrics.completion_tokens)],
    ['总计', formatCount(metrics.total_tokens)],
    ['首字', formatDuration(metrics.first_token_ms ?? metrics.prefill_ms)],
    ['解码', formatDuration(metrics.decode_ms)],
    ['预填', formatRate(metrics.prefill_tps)],
    ['续写', formatRate(metrics.completion_tps ?? metrics.tokens_per_second)],
  ] : [];
  const visibleMetrics = metricItems.filter(([, value]) => value !== null);
  return (
    <div className="chat-answer-meta">
      <div className="chat-answer-byline">{segments.join(' · ')}</div>
      {visibleMetrics.length > 0 && (
        <div className="chat-answer-metrics" data-testid="answer-metrics" aria-label="输出指标">
          {visibleMetrics.map(([label, value]) => (
            <span key={label} className="chat-answer-metric">
              <span>{label}</span>
              <strong>{value}</strong>
            </span>
          ))}
          {formatCount(metrics?.cache_hit_tokens) && (
            <span className="chat-answer-metric is-cache">
              <span>缓存命中</span>
              <strong>{formatCount(metrics?.cache_hit_tokens)}</strong>
            </span>
          )}
        </div>
      )}
    </div>
  );
}

export default AnsweredByLine;
