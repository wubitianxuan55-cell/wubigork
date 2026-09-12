// sin/SinIllustration.tsx — 故事内联插图（图文混杂的「图」那一半）。
//
// 复用面：生成走绘梦图像后端（app.SinIllustrate → mediaState.GenerateFreeImage），
// 预览走办公附件读取通道（AttachmentDataURL，与 Message.tsx 的 InlineAttachment
// 同一口径——WebView2 里本地路径不能直接 img src，必须转 data URL）。
//
// 自动生成纪律：仅在该轮流式结束（ready）且尚无产物时触发一次；失败留错并给
// 「重试」按钮，不在后台反复重试（烧算力且难解释）。

import { useCallback, useEffect, useRef, useState } from 'react'
import { Button } from 'antd'
import { app } from '../../gaea/lib/bridge'
import { cancelImageGeneration, readFileAsDataURL } from '../../api/image'
import { GenerationProgress } from '../../components/imagegen/GenerationProgress'
import { useComfyTaskProgress } from '../../components/imagegen/useComfyTaskProgress'
import {
  enqueueIllustration, illustrationQueueSnapshot, subscribeIllustrationQueue,
} from './illustrationQueue'

type Status = 'idle' | 'queued' | 'generating' | 'loading' | 'ready' | 'error'

/** SinIllustrate 返回值里的角色一致性信息（用没用参考图、为什么）。 */
interface RefInfo {
  used: boolean
  characters: string[]
  reason: string
  fallback: boolean
  anchorAdded: string[]
}

export interface SinIllustrationProps {
  storyId: string
  messageId: number
  cueKey: string
  prompt: string
  /** 已生成的插图路径（来自消息 extra；有值直接渲染，不重复生成）。 */
  path?: string
  /** 该轮流式是否已结束（未结束不自动生成，避免半截提示词出图）。 */
  ready: boolean
  /** 生成成功回调（父层写回消息状态 + 后续可导出）。 */
  onGenerated: (cueKey: string, path: string) => void
  /** 生成/读取失败时的提示（父层统一呈现）。 */
  onError?: (message: string) => void
}

export function SinIllustration({
  storyId, messageId, cueKey, prompt, path: pathProp, ready, onGenerated, onError,
}: SinIllustrationProps) {
  const [status, setStatus] = useState<Status>(pathProp ? 'loading' : 'idle')
  const [path, setPath] = useState(pathProp ?? '')
  const [dataUrl, setDataUrl] = useState('')
  const [error, setError] = useState('')
  const [refInfo, setRefInfo] = useState<RefInfo | null>(null)
  // 自动生成只尝试一次（重试由按钮显式触发，防渲染重入导致重复烧算力）
  const autoTriedRef = useRef(false)
  const startedRef = useRef(false)
  // 队列 token（>0 = 已入队；用于算位次与取消时定位）
  const tokenRef = useRef(0)
  const [queuedAhead, setQueuedAhead] = useState(0)
  // 本地计时（后端 elapsed 只在 ComfyUI 有；其余后端用本地秒表，诚实显示等待时长）
  const [localElapsed, setLocalElapsed] = useState(0)
  const generating = status === 'generating'
  const progress = useComfyTaskProgress(generating)

  // 队列位次订阅：排队中显示「前面还有 N 张」，并在轮到自己时切换为生成中
  useEffect(() => {
    if (status !== 'queued') return
    const sync = () => {
      const snap = illustrationQueueSnapshot()
      const pos = snap.positionOf(tokenRef.current)
      setQueuedAhead(pos > 0 ? pos - 1 : 0)
    }
    sync()
    return subscribeIllustrationQueue(sync)
  }, [status])

  // 本地秒表：排队+生成阶段都走，每秒一格
  useEffect(() => {
    if (status !== 'queued' && status !== 'generating') return
    setLocalElapsed(0)
    const started = Date.now()
    const timer = setInterval(() => setLocalElapsed((Date.now() - started) / 1000), 1000)
    return () => clearInterval(timer)
  }, [status])

  // 路径 → data URL（本地文件读取失败如实显示错误，不退化成坏图）
  useEffect(() => {
    if (!path) return
    let live = true
    setStatus('loading')
    readFileAsDataURL(path)
      .then((url) => {
        if (!live) return
        setDataUrl(url)
        setStatus('ready')
      })
      .catch((err: unknown) => {
        if (!live) return
        setStatus('error')
        const msg = err instanceof Error ? err.message : '插图读取失败'
        setError(msg)
        onError?.(msg)
      })
    return () => { live = false }
  }, [path, onError])

  const generate = useCallback(async () => {
    if (startedRef.current) return
    startedRef.current = true
    setError('')
    setStatus('queued')
    const { promise, token } = enqueueIllustration(async () => {
      // 轮到自己才开始真正调用（串行：一次一张，进度才有归属）
      setStatus('generating')
      return app.SinIllustrate(storyId, messageId, cueKey, prompt, '')
    })
    tokenRef.current = token
    try {
      const res = (await promise) as {
        path?: unknown
        ref_used?: unknown
        ref_characters?: unknown
        ref_reason?: unknown
        ref_fallback?: unknown
        anchor_added?: unknown
      }
      const p = typeof res?.path === 'string' ? res.path : ''
      if (!p) throw new Error('未返回插图路径')
      const names = Array.isArray(res?.ref_characters)
        ? (res.ref_characters as unknown[]).filter((n): n is string => typeof n === 'string')
        : []
      const anchors = Array.isArray(res?.anchor_added)
        ? (res.anchor_added as unknown[]).filter((n): n is string => typeof n === 'string')
        : []
      setRefInfo({
        used: res?.ref_used === true,
        characters: names,
        reason: typeof res?.ref_reason === 'string' ? res.ref_reason : '',
        fallback: res?.ref_fallback === true,
        anchorAdded: anchors,
      })
      setPath(p)
      onGenerated(cueKey, p)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '插图生成失败'
      setStatus('error')
      setError(msg)
      onError?.(msg)
    } finally {
      startedRef.current = false
    }
  }, [cueKey, messageId, onError, onGenerated, prompt, storyId])

  // 取消：后端取消的是当前在跑的那一个任务；排队中（还没开跑）取消只标记本卡片
  const onCancel = useCallback(async () => {
    try {
      await cancelImageGeneration()
    } catch {
    // 取消失败不阻断本地状态收敛（下方仍回到可重试态）
    }
    startedRef.current = false
    // 阻断自动出图再触发（取消 = 用户明确不要这张，不该被自动重跑覆盖）
    autoTriedRef.current = true
    setStatus('idle')
    setError('')
  }, [])

  // 自动出图：流式结束后且没有产物时触发一次
  useEffect(() => {
    if (!ready || path || autoTriedRef.current) return
    autoTriedRef.current = true
    void generate()
  }, [generate, path, ready])

  return (
    <figure className="sin-figure">
      {status === 'ready' && dataUrl ? (
        <img className="sin-figure-img" src={dataUrl} alt={prompt || '故事插图'} loading="lazy" />
      ) : (
        <div className={`sin-figure-slot${status === 'error' ? ' is-error' : ''}`}>
          {status === 'loading' && (
            <>
              <span className="sin-figure-spinner" aria-hidden="true" />
              <span className="sin-figure-slot-text">正在读取插图…</span>
            </>
          )}
          {(status === 'queued' || status === 'generating') && (
            <div className="sin-figure-progress">
              <GenerationProgress
                percent={progress.percent}
                node={progress.node}
                elapsed={progress.percent >= 0 ? progress.elapsed : localElapsed}
                label={status === 'queued' ? '排队中' : undefined}
                hint={status === 'queued'
                  ? (queuedAhead > 0 ? `前面还有 ${queuedAhead} 张` : '即将开始')
                  : (progress.status === 'queued' ? '引擎排队中' : '')}
              />
              <div className="sin-figure-progress-actions">
                <Button size="small" type="text" danger onClick={() => void onCancel()}>
                  取消
                </Button>
              </div>
            </div>
          )}
          {status === 'idle' && (
            <>
              <span className="sin-figure-slot-text">插图位</span>
              <Button size="small" onClick={() => void generate()}>生成插图</Button>
            </>
          )}
          {status === 'error' && (
            <>
              <span className="sin-figure-slot-text">插图失败：{error}</span>
              <Button size="small" onClick={() => void generate()}>重试</Button>
            </>
          )}
        </div>
      )}
      <figcaption className="sin-figure-cap">
        <span className="sin-figure-cap-text" title={prompt}>{prompt}</span>
        {refInfo?.used && refInfo.characters.length > 0 && (
          <span className="sin-figure-ref is-on" title="按角色库参考图生成（人物一致性锚定）">
            角色参考：{refInfo.characters.join('、')}
          </span>
        )}
        {refInfo && !refInfo.used && (refInfo.reason || refInfo.fallback) && (
          <span className="sin-figure-ref" title={refInfo.reason}>
            {refInfo.fallback ? '参考图不可用 · 已按纯文本生成' : '未用参考图'}
          </span>
        )}
        {refInfo && refInfo.anchorAdded.length > 0 && (
          <span className="sin-figure-ref" title={`提示词已补入外观锚点：${refInfo.anchorAdded.join('、')}`}>
            已补外观锚点
          </span>
        )}
        {status === 'ready' && (
          <Button size="small" type="text" onClick={() => void generate()}>重新生成</Button>
        )}
      </figcaption>
    </figure>
  )
}
