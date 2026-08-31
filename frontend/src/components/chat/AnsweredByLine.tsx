/* eslint-disable react-refresh/only-export-components -- AnsweredByInfo 类型与 answeredBySourceLabel/answeredByCostText 工具导出供 ChatRow/useChatStream 与测试复用 */
/**
 * 消息级「由谁回答 / 为何 / 花了多少」回显（v4.15，前端）。
 *
 * chat-stream:<runID> 事件的 done 帧 / ChatSend 返回新增可选字段 answered_by
 * （旧事件/旧消息无此字段时前端静默跳过，向后兼容），形如：
 *   { engine, model, source, cost_cny }
 *  - engine / model：实际回答的引擎与模型
 *  - source：「为何」—— feature | global | fallback（v4.15 已砍自动路由，
 *    chat 链路只会出现这三种；未知值原样兜底展示）
 *  - cost_cny：估算费用（人民币；本地引擎 = 0，不虚报费用）
 */
export interface AnsweredByInfo {
  engine: string;
  model: string;
  /** feature | global | fallback（未知值原样兜底展示） */
  source: string;
  /** 估算费用，人民币；本地引擎为 0 */
  cost_cny: number;
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
 * 消息底部一行小字（仅当 extra.answered_by 存在时由消息行组件渲染）：
 * 「由 {engine}/{model} 回答 · {sourceLabel}[ · 约 ¥{cost_cny.toFixed(2)}]」
 * 费用 <= 0（本地/未知）时省略费用段。
 */
export function AnsweredByLine({ info }: { info: AnsweredByInfo }) {
  const label = answeredBySourceLabel(info.source);
  const costText = answeredByCostText(info.cost_cny);
  const segments = [`由 ${info.engine}/${info.model} 回答`, label];
  if (costText) {
    segments.push(costText);
  }
  return (
    <div
      className="text-[10px] leading-none"
      style={{ color: 'var(--md-sys-color-text-secondary)' }}
    >
      {segments.join(' · ')}
    </div>
  );
}

export default AnsweredByLine;
