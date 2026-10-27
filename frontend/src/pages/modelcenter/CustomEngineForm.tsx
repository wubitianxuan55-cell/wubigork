// CustomEngineForm.tsx — 自定义引擎添加/编辑表单（EngineSection 拆件，
// 2026-10-27 瘦身重组：537 行 17 态热点分流，表单态全部内聚本文件）。
// 价目 v1 语义随迁不变：引擎级统一价（每百万 tokens，CNY），null = 输入框
// 留空 = 清除（后端语义 0=未填=不计价），经 SaveEngine 指针三态落库；
// savedPrices 会话级记忆由父组件持有（saveEngine 成功后 engines 不刷新，
// 重开表单优先显示本会话已保存值，防回退到列表旧值）。
import { useState } from 'react'
import { Button, Input, InputNumber, message, Space } from 'antd'
import type { EngineConfig } from '../../api/engines'
import { saveEngine } from '../../api/engines'
import { isValidBaseURL } from './utils'
import { useModelCenterActions } from './context'

/** 表单校验（名称非空 + 地址 http(s) 前缀，与后端 validBaseURL 同口径双保险，
 *  防 API Key 粘进地址框）。通过返回 null，不通过返回提示文案。
 *  仅本文件两表单消费，不导出（react-refresh 组件文件禁导出函数）。 */
function customFormError(name: string, url: string): string | null {
  if (!name) return '请输入引擎名称'
  if (!isValidBaseURL(url)) return 'API 地址无效：必须以 http:// 或 https:// 开头'
  return null
}

/** 价目形状：引擎级统一价（每百万 tokens，CNY）；null = 留空 = 清除 */
export interface EnginePrices { in: number | null; out: number | null }

/** 添加自定义引擎行内表单（OpenAI 兼容）；成功后由父收起。 */
export function AddCustomEngineForm({ onClose }: { onClose: () => void }) {
  const { handleAddCustomEngine } = useModelCenterActions()
  const [name, setName] = useState('')
  const [url, setUrl] = useState('')
  const [key, setKey] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async () => {
    const invalid = customFormError(name.trim(), url.trim())
    if (invalid) { message.warning(invalid); return }
    setBusy(true)
    try {
      if (await handleAddCustomEngine(name.trim(), url.trim(), key.trim())) onClose()
    } finally { setBusy(false) }
  }

  return (
    <div
      style={{
        border: '1px solid var(--mc-border)',
        borderRadius: 'var(--mc-radius-md)',
        padding: '10px 12px',
        display: 'grid',
        gap: 8,
        maxWidth: 640,
      }}
    >
      <div style={{ fontSize: 13, fontWeight: 600 }}>添加自定义引擎（OpenAI 兼容）</div>
      <Input
        size="small"
        aria-label="自定义引擎名称"
        placeholder="名称（如：硅基流动 / OneAPI 中转）"
        value={name}
        onChange={e => setName(e.target.value)}
      />
      <Input
        size="small"
        aria-label="自定义引擎 API 地址"
        placeholder="API 地址（https://api.example.com/v1）"
        value={url}
        onChange={e => setUrl(e.target.value)}
      />
      <Input
        size="small"
        aria-label="自定义引擎 API Key"
        placeholder="API Key（可选，服务商控制台获取）"
        value={key}
        onChange={e => setKey(e.target.value)}
      />
      <Space>
        <Button size="small" type="primary" loading={busy} onClick={submit}>保存</Button>
        <Button size="small" onClick={onClose}>取消</Button>
      </Space>
    </div>
  )
}

/** 编辑自定义引擎表单（嵌在引擎卡内；Key 留空 = 不修改〔后端契约〕）。
 *  初始价目：savedPrice（本会话已保存值）优先于引擎列表（saveEngine 后未刷新的旧值）。 */
export function EditCustomEngineForm({ engine, savedPrice, onSaved, onClose }: {
  engine: EngineConfig
  savedPrice?: EnginePrices
  onSaved: (id: string, prices: EnginePrices) => void
  onClose: () => void
}) {
  const { handleUpdateCustomEngine } = useModelCenterActions()
  const [name, setName] = useState(engine.name)
  const [url, setUrl] = useState(engine.base_url || '')
  const [key, setKey] = useState('')
  const [priceIn, setPriceIn] = useState<number | null>(savedPrice ? savedPrice.in : engine.user_price_in ?? null)
  const [priceOut, setPriceOut] = useState<number | null>(savedPrice ? savedPrice.out : engine.user_price_out ?? null)
  const [busy, setBusy] = useState(false)

  const submit = async () => {
    const invalid = customFormError(name.trim(), url.trim())
    if (invalid) { message.warning(invalid); return }
    if (priceIn !== null && (!Number.isFinite(priceIn) || priceIn < 0)) {
      message.warning('输入价必须是不小于 0 的数字')
      return
    }
    if (priceOut !== null && (!Number.isFinite(priceOut) || priceOut < 0)) {
      message.warning('输出价必须是不小于 0 的数字')
      return
    }
    setBusy(true)
    try {
      if (!(await handleUpdateCustomEngine(engine.id, name.trim(), url.trim(), key.trim()))) return
      // 价目 v1：经既有 SaveEngine 通道落库（数字=设置、留空=0=清除；
      // 费用折算优先消费用户价目，留空维持现状不计价）
      await saveEngine({
        id: engine.id,
        user_price_in: priceIn ?? 0,
        user_price_out: priceOut ?? 0,
      } as EngineConfig)
      onSaved(engine.id, { in: priceIn, out: priceOut })
      onClose()
    } catch {
      message.error('价目保存失败，请重试')
    } finally { setBusy(false) }
  }

  return (
    <div
      style={{
        border: '1px dashed var(--mc-border)',
        borderRadius: 'var(--mc-radius-sm)',
        padding: '8px 10px',
        display: 'grid',
        gap: 8,
        maxWidth: 640,
      }}
    >
      <div style={{ fontSize: 12, color: 'var(--mc-muted)' }}>编辑自定义引擎（Key 留空 = 不修改）</div>
      <Input
        size="small"
        aria-label="编辑引擎名称"
        placeholder="名称"
        value={name}
        onChange={e => setName(e.target.value)}
      />
      <Input
        size="small"
        aria-label="编辑引擎 API 地址"
        placeholder="API 地址（http:// 或 https:// 开头）"
        value={url}
        onChange={e => setUrl(e.target.value)}
      />
      <Input
        size="small"
        aria-label="编辑引擎 API Key"
        placeholder="API Key（留空 = 不修改）"
        value={key}
        onChange={e => setKey(e.target.value)}
      />
      {/* 价目 v1：引擎级统一价（每百万 tokens，CNY）。留空 = 清除 =
          不计价（费用统计维持现状）；仅影响费用估算，不影响调用。 */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
        <InputNumber
          size="small"
          style={{ width: '100%' }}
          min={0}
          step={0.1}
          aria-label="输入价（每百万 tokens，CNY）"
          placeholder="输入价（每百万 tokens，CNY）"
          value={priceIn}
          onChange={v => setPriceIn(typeof v === 'number' && Number.isFinite(v) ? v : null)}
        />
        <InputNumber
          size="small"
          style={{ width: '100%' }}
          min={0}
          step={0.1}
          aria-label="输出价（每百万 tokens，CNY）"
          placeholder="输出价（每百万 tokens，CNY）"
          value={priceOut}
          onChange={v => setPriceOut(typeof v === 'number' && Number.isFinite(v) ? v : null)}
        />
      </div>
      <div style={{ fontSize: 11, color: 'var(--mc-muted)' }}>
        价目为引擎级统一价（¥/百万 tokens），留空 = 不计价；填了价目的引擎，用量统计的费用估算将按此价折算
      </div>
      <Space>
        <Button size="small" type="primary" loading={busy} onClick={submit}>保存</Button>
        <Button size="small" onClick={onClose}>取消</Button>
      </Space>
    </div>
  )
}
