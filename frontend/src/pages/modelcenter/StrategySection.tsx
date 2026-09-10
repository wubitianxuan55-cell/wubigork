import { useCallback, useEffect, useState } from 'react'
import { Button, Input, message } from 'antd'
import { ControlOutlined, ReloadOutlined, SaveOutlined } from '@ant-design/icons'
import { app } from '../../gaea/lib/bridge'
import type { SpaceProfileView } from '../../gaea/lib/types'
import { SectionHead, StatusChip } from './ui'

// 模型中心「空间策略」（长期规划阶段二·总闸画面）：办公引擎空间的装配
// profile 一张画面——本机/云端在引擎管理、功能域绑定在「功能绑定」，本区
// 显形并编辑 gaea.toml [space_profiles]（boot 装配消费覆写；权限/护栏走
// PermissionsForSpace / PlayGuardrails 既有链）。写走 GaeaSpaceProfileSet，
// 七键均当场切；生效=下次引擎重建/重启。壳层书斋/闲庭=界面导航（1B 拍板），
// 与本区引擎空间是两套——底部 meta 固定说明，防混同。

const SPACE_META: Record<string, { title: string; desc: string }> = {
  work: { title: '办公空间（work）', desc: '办公管家/造价/进度的引擎空间' },
  play: { title: '娱乐空间（play）', desc: '轻语/聊天等娱乐域分区，产品默认不弹审批卡' },
}

/** 其余功能域覆写（与 Go spaceProfileKeys 同序，不含 gaea 主控） */
const DOMAIN_KEYS = ['chat', 'whisper', 'novel', 'office', 'characterlib', 'routine'] as const
type DomainKey = (typeof DOMAIN_KEYS)[number]
type ProfileKey = 'gaea' | DomainKey

const MODEL_KEY_LABELS: Record<DomainKey, string> = {
  chat: '对话', whisper: '轻语', novel: '小说',
  office: '办公文档', characterlib: '角色库', routine: '例行',
}

function domainRef(p: SpaceProfileView, key: DomainKey): string {
  return (p.models ?? {})[key] ?? ''
}

function ModelOverride({ override, ok, resolved }: { override: string; ok: boolean; resolved: string }) {
  if (!override) {
    return <span style={{ color: 'var(--mc-muted)' }}>未配置（维持现状模型）</span>
  }
  if (ok) {
    return <span>{override} <span style={{ color: 'var(--mc-muted)' }}>→ {resolved}</span></span>
  }
  // 失败说人话：引用无法解析时 boot 告警并回退现状模型——原样显形，不让用户猜
  return (
    <span style={{ color: 'var(--mc-warn, #d46b08)' }}>
      {override}（无法解析，现状模型继续生效——检查 provider/model 拼写）
    </span>
  )
}

export function StrategySection() {
  const [profiles, setProfiles] = useState<SpaceProfileView[] | null>(null)
  const [activeSpace, setActiveSpace] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  // 覆写编辑态（总闸当场切；生效=下次引擎重建/重启，与激活空间同口径）
  const [editing, setEditing] = useState<{ space: string; key: ProfileKey } | null>(null)
  const [draft, setDraft] = useState('')
  const [saving, setSaving] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [p, a] = await Promise.all([
        app.GaeaSpaceProfiles(),
        app.GaeaSpaceActive().catch(() => null), // 当前空间读取失败只降级高亮，不拖垮整区
      ])
      setProfiles(p)
      setActiveSpace(a?.space ?? null)
    } catch (e: unknown) {
      setProfiles(null)
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void load() }, [load])

  const startEdit = (space: string, key: ProfileKey, current: string) => {
    setEditing({ space, key })
    setDraft(current)
  }

  const saveEdit = async (space: string, key: ProfileKey) => {
    setSaving(true)
    try {
      const views = await app.GaeaSpaceProfileSet(space, key, draft)
      setProfiles(views)
      setEditing(null)
      message.success(draft.trim() ? '已写入，下次引擎重建/重启生效' : '已清除，回退现状模型（下次引擎重建/重启生效）')
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : '写入失败')
    } finally {
      setSaving(false)
    }
  }

  const isEditing = (space: string, key: ProfileKey) =>
    editing?.space === space && editing.key === key

  const modeOn = profiles?.[0]?.modeOn ?? true

  return (
    <section className="mc-section" data-testid="mc-strategy">
      <SectionHead
        icon={<ControlOutlined />}
        title="空间策略"
        desc="办公引擎空间的装配 profile：模型覆写、权限模式、内容护栏——读自 gaea.toml，改后下次引擎重建/重启生效"
      />

      {!modeOn && profiles && (
        <div className="mc-bind-row" style={{ marginBottom: 12 }}>
          <StatusChip tone="warn" dot>space.mode=off：空间维度整体关闭，以下策略不生效，全域回退 work 现状</StatusChip>
        </div>
      )}

      {error && (
        <div className="mc-bind-card" data-testid="mc-strategy-error" style={{ marginBottom: 12 }}>
          <div className="mc-bind-head">
            <span className="mc-bind-title">空间策略读取失败</span>
            <Button size="small" icon={<ReloadOutlined />} loading={loading} onClick={() => void load()}>重试</Button>
          </div>
          <div className="mc-bind-desc">{error}</div>
          <div className="mc-bind-meta">旧版后端无 GaeaSpaceProfiles 绑定时也会这样——升级后端后重试</div>
        </div>
      )}

      <div className="mc-grid two-col">
        {(profiles ?? []).map((p) => (
          <div className="mc-bind-card" key={p.space} data-testid={`mc-strategy-${p.space}`}>
            <div className="mc-bind-head">
              <span className="mc-bind-title">{SPACE_META[p.space]?.title ?? p.space}</span>
              {activeSpace === p.space
                ? <StatusChip tone="ok" dot>当前生效</StatusChip>
                : <StatusChip tone="neutral" dot>未生效</StatusChip>}
            </div>
            <div className="mc-bind-desc">{SPACE_META[p.space]?.desc ?? ''}</div>
            <div className="mc-bind-row">
              <span className="mc-bind-title" style={{ fontSize: 12 }}>办公 Agent 模型</span>
              {isEditing(p.space, 'gaea') ? (
                <span style={{ display: 'inline-flex', gap: 6, alignItems: 'center' }}>
                  <Input
                    size="small"
                    value={draft}
                    onChange={(e) => setDraft(e.target.value)}
                    placeholder="provider/model（留空=维持现状）"
                    style={{ width: 260 }}
                    data-testid={`mc-strategy-${p.space}-input`}
                    disabled={saving}
                  />
                  <Button size="small" type="primary" icon={<SaveOutlined />}
                    loading={saving} onClick={() => void saveEdit(p.space, 'gaea')}
                    data-testid={`mc-strategy-${p.space}-save`}>保存</Button>
                  <Button size="small" disabled={saving} onClick={() => setEditing(null)}>取消</Button>
                </span>
              ) : (
                <Button size="small" type="text" onClick={() => startEdit(p.space, 'gaea', p.gaea)}
                  data-testid={`mc-strategy-${p.space}-edit`}>编辑</Button>
              )}
            </div>
            <div className="mc-bind-row" data-testid={`mc-strategy-${p.space}-gaea`}>
              <ModelOverride override={p.gaea} ok={p.gaeaOk} resolved={p.gaeaResolved} />
            </div>
            <div className="mc-bind-title" style={{ fontSize: 12, marginTop: 4 }}>其余功能域</div>
            {DOMAIN_KEYS.map((k) => {
              const val = domainRef(p, k)
              return (
                <div className="mc-bind-row" key={k} data-testid={`mc-strategy-${p.space}-${k}`} style={{ fontSize: 12 }}>
                  <span className="mc-bind-title" style={{ fontSize: 12, minWidth: 64 }}>{MODEL_KEY_LABELS[k]}</span>
                  {isEditing(p.space, k) ? (
                    <span style={{ display: 'inline-flex', gap: 6, alignItems: 'center', flex: 1 }}>
                      <Input
                        size="small"
                        value={draft}
                        onChange={(e) => setDraft(e.target.value)}
                        placeholder="provider/model（留空=维持现状）"
                        style={{ width: 220 }}
                        data-testid={`mc-strategy-${p.space}-${k}-input`}
                        disabled={saving}
                      />
                      <Button size="small" type="primary" icon={<SaveOutlined />}
                        loading={saving} onClick={() => void saveEdit(p.space, k)}
                        data-testid={`mc-strategy-${p.space}-${k}-save`}>保存</Button>
                      <Button size="small" disabled={saving} onClick={() => setEditing(null)}>取消</Button>
                    </span>
                  ) : (
                    <>
                      <span style={{ flex: 1, color: val ? undefined : 'var(--mc-muted)' }}>
                        {val || '未配置（维持现状）'}
                      </span>
                      <Button size="small" type="text" onClick={() => startEdit(p.space, k, val)}
                        data-testid={`mc-strategy-${p.space}-${k}-edit`}>编辑</Button>
                    </>
                  )}
                </div>
              )
            })}
            <div className="mc-bind-row">
              <span className="mc-bind-title" style={{ fontSize: 12 }}>权限</span>
              <span>
                生效模式 {p.permMode || '（现状）'}；强制审批{' '}
                {p.permHardAskBySpace
                  ? `按空间配置（${p.permHardAskCount} 项）`
                  : '未按空间配置（包级默认集）'}
              </span>
            </div>
            {p.space === 'play' && (
              <div className="mc-bind-row">
                <span className="mc-bind-title" style={{ fontSize: 12 }}>内容护栏</span>
                <span>{p.guardrailsOn ? '生效（钳制中）' : '未配置（零钳制）'}</span>
              </div>
            )}
          </div>
        ))}
      </div>

      <div className="mc-bind-meta" style={{ marginTop: 12 }}>
        壳层书斋/闲庭切换是界面导航，不打扰在跑的活（1B 拍板）；本区是办公引擎空间策略，两套是定局。
        本机/云端引擎见「引擎管理」，功能域绑定见「功能绑定」。
      </div>
    </section>
  )
}
