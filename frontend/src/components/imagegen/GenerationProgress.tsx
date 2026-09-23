/* eslint-disable react-refresh/only-export-components -- 阶段名表与查表函数跟组件同源
   （节点名 → 中文阶段名），拆文件只会让两处漂移；先例：gaea/components/Transcript.tsx */
// GenerationProgress.tsx — 生成进度条（绘梦 / 原罪共用，v4.258 抽核）。
//
// 口径（与绘梦 v4.71 卡片化一致，逐字保留）：
//   - ComfyUI 有实时百分比 → 确定态进度条 + 「全部: N%」+ 「当前节点: 中文阶段名」；
//   - 其余后端/未知 → 不定态光带 + 「生成中…」（诚实：不编造百分比）。
// 类名沿用 `.ig-progress*`（绘梦既有测试与样式锚点），样式移入本组件自带
// generation-progress.css，两个消费方共享同一实现，防止两处进度视觉漂移。
import './generation-progress.css'

export interface GenerationProgressProps {
  /** 实时百分比（<0 = 未知，走不定态）。 */
  percent: number
  /** 当前执行节点（ComfyUI class_type；未知节点回退原样显示）。 */
  node?: string
  /** 无实时进度时是否显示不定态（默认 true）。 */
  indeterminateWhenUnknown?: boolean
  /** 已用时（秒；用于 meta 行右侧）。 */
  elapsed?: number
  /** 附加状态文案（如「排队中 · 前面还有 2 张」），显示在 meta 行。 */
  hint?: string
  /** 主标签覆盖（如排队中传「排队中」，避免出现「生成中… 排队中」的自相矛盾）。 */
  label?: string
}

export function GenerationProgress({
  percent, node, indeterminateWhenUnknown = true, elapsed, hint, label,
}: GenerationProgressProps) {
  const hasReal = percent >= 0
  const pct = hasReal ? Math.min(100, Math.max(0, Math.round(percent))) : -1
  const nodeLabel = node ? (COMFY_NODE_LABELS[node] || node) : ''
  return (
    <div className="ig-progress" aria-live="polite">
      <div className="ig-progress-track">
        <div
          className={hasReal ? 'ig-progress-fill' : indeterminateWhenUnknown ? 'ig-progress-fill is-indeterminate' : 'ig-progress-fill'}
          style={hasReal ? { width: `${pct}%` } : undefined}
        />
      </div>
      <div className="ig-progress-meta">
        <span>{label ?? (hasReal ? `全部: ${pct}%` : '生成中…')}</span>
        {nodeLabel && <span>当前节点: {nodeLabel}</span>}
        {typeof elapsed === 'number' && elapsed > 0 && <span>已用时 {Math.round(elapsed)}s</span>}
        {hint && <span>{hint}</span>}
      </div>
    </div>
  )
}

/** ComfyUI 节点 class_type → 中文阶段名（未知节点回退原始名）。 */
export const COMFY_NODE_LABELS: Record<string, string> = {
  queue: '排队中（前有任务）',
  CheckpointLoaderSimple: '加载模型',
  CheckpointLoader: '加载模型',
  UNETLoader: '加载模型',
  UnetLoaderGGUF: '加载模型',
  CLIPLoader: '加载模型',
  CLIPLoaderGGUF: '加载模型',
  DualCLIPLoader: '加载模型',
  LoraLoader: '加载 LoRA',
  LoraLoaderModelOnly: '加载 LoRA',
  EmptyLatentImage: '初始化画布',
  LatentFromImage: '初始化画布',
  KSampler: '采样中',
  KSamplerAdvanced: '采样中',
  SamplerCustom: '采样中',
  VAEDecode: '解码中',
  VAEDecodeTiled: '解码中',
  VAEEncode: '编码中',
  LoadImage: '读取参考图',
  SaveImage: '保存图片',
  SaveAnimatedWEBP: '保存动画',
  LTXVideo: '生成视频',
  LTXV: '生成视频',
  LTXVConditioning: '视频条件',
  ZImagePowerNodes: '图像处理',
}

