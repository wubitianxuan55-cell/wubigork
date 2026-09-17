// SceneBibleDrawer.tsx — 场景圣经「视角」抽屉（刀7续 POV 视图收官，规格
// 进度计划/gaea-pov-view-20260917.md）：POV 视角掩码（povView=已知 /
// hiddenFacts=不知情且不得泄露）此前只在生成链内部消费——本抽屉把它亮给
// 作者审视：POV 头 + 已知（绿）/不知情（红）对照 + 出场角色卡 + 伏笔约束 +
// 时间锚点/主线。空区段如实空展示（编译静默降级契约），不编造。
import React, { useCallback, useEffect, useState } from 'react'
import { Alert, Drawer, Empty, Spin, Tag, Typography } from 'antd'
import { app } from '../../gaea/lib/bridge'

const softTextStyle: React.CSSProperties = { fontSize: 12, color: 'var(--v3-fg-soft, #6b7280)' }
const labelTextStyle: React.CSSProperties = { fontSize: 12, color: 'var(--v3-fg-soft, #6b7280)', marginBottom: 4, fontWeight: 600 }

/** 场景圣经视图（Go SceneBibleView 镜像；全字段可缺省防御）。 */
export interface SceneBibleView {
  pov?: string
  title?: string
  location?: string
  timeOfDay?: string
  mood?: string
  tags?: string[]
  characters?: Array<{
    name?: string; roleType?: string; status?: string; location?: string
    currentState?: string; careerMain?: string; careerSub?: string[]
    items?: string[]; knownBy?: string[]
  }>
  povView?: string
  hiddenFacts?: string[]
  foreshadows?: string[]
  memories?: string[]
  timeAnchor?: string
  style?: string
  thread?: string
}

export default function SceneBibleDrawer({ open, chapterNum, sceneID, sceneLabel, onClose }: {
  open: boolean
  chapterNum: number
  /** 场景 ID（空=整章合成视图）。 */
  sceneID: string
  sceneLabel?: string
  onClose: () => void
}) {
  const [loading, setLoading] = useState(false)
  const [view, setView] = useState<SceneBibleView | null>(null)
  const [error, setError] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    setView(null)
    try {
      setView(await app.NovelSceneBibleView(chapterNum, sceneID))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }, [chapterNum, sceneID])

  useEffect(() => {
    if (open) void load()
  }, [open, load])

  const hidden = view?.hiddenFacts ?? []

  return (
    <Drawer open={open} onClose={onClose} width={520} destroyOnHidden
      title={`场景圣经 · ${sceneLabel ?? (sceneID || `第 ${chapterNum} 章整章`)}`}>
      {loading ? (
        <div style={{ padding: '48px 0', textAlign: 'center' }}><Spin /></div>
      ) : error ? (
        <Alert type="error" showIcon message={error}
          description={<ButtonRetry onRetry={() => void load()} />} />
      ) : !view ? (
        <Empty description="没有可展示的场景圣经" />
      ) : (
        <div data-testid="scene-bible-body" style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          {/* POV 头 */}
          <div>
            <div style={{ fontSize: 15, fontWeight: 650 }}>
              {view.pov ? `通过 ${view.pov} 的眼睛` : '未设定 POV（全知视角）'}
            </div>
            <div style={{ ...softTextStyle, marginTop: 2 }}>
              {[view.title, view.location, view.timeOfDay, view.mood].filter(Boolean).join(' · ') || '—'}
              {(view.tags ?? []).map(tg => <Tag key={tg} style={{ marginLeft: 4 }}>{tg}</Tag>)}
            </div>
          </div>

          {/* 核心区：已知 vs 不知情对照 */}
          <div data-testid="scene-bible-pov-contrast" style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            <div>
              <div style={{ ...labelTextStyle, color: 'var(--color-success, var(--md-sys-color-primary, #16a34a))' }}>POV 已知的关键事实</div>
              {view.povView ? (
                <div style={{ fontSize: 12.5, lineHeight: 1.8, whiteSpace: 'pre-wrap', background: 'rgba(22,163,74,0.06)', borderRadius: 6, padding: '8px 10px' }}>{view.povView}</div>
              ) : (
                <div style={softTextStyle}>（无——该 POV 没有单独已知事实区段）</div>
              )}
            </div>
            <div>
              <div style={{ ...labelTextStyle, color: 'var(--color-error, var(--md-sys-color-error, #dc2626))' }}>POV 不知情 · 本场景不得泄露（{hidden.length}）</div>
              {hidden.length > 0 ? (
                <ul style={{ margin: 0, paddingLeft: 18, fontSize: 12.5, lineHeight: 1.9 }}>
                  {hidden.map((h, i) => <li key={i}>{h}</li>)}
                </ul>
              ) : (
                <div style={softTextStyle}>（无隐藏约束——没有对该 POV 瞒住的事实）</div>
              )}
            </div>
          </div>

          {/* 出场角色 */}
          {(view.characters ?? []).length > 0 && (
            <div>
              <div style={labelTextStyle}>出场角色（{(view.characters ?? []).length}）</div>
              {(view.characters ?? []).map((c, i) => (
                <div key={i} style={{ padding: '4px 0', fontSize: 12.5, borderBottom: '1px dashed rgba(0,0,0,0.08)' }}>
                  <b>{c.name || '—'}</b>
                  {c.roleType && <Tag style={{ marginLeft: 4 }}>{c.roleType}</Tag>}
                  {c.status && c.status !== 'Alive' && <Tag color="volcano" style={{ marginLeft: 4 }}>{c.status}</Tag>}
                  {c.location && <span style={softTextStyle}> · {c.location}</span>}
                  {c.currentState && <span style={softTextStyle}> · {c.currentState}</span>}
                  {c.careerMain && <Tag style={{ marginLeft: 4 }}>{c.careerMain}</Tag>}
                  {(c.careerSub ?? []).map(s => <Tag key={s} style={{ marginLeft: 4 }}>{s}</Tag>)}
                </div>
              ))}
            </div>
          )}

          {/* 伏笔约束 / 时间锚点 / 主线 / 文风 */}
          {(view.foreshadows ?? []).length > 0 && (
            <div>
              <div style={labelTextStyle}>未回收伏笔约束（{(view.foreshadows ?? []).length}）</div>
              <ul style={{ margin: 0, paddingLeft: 18, fontSize: 12.5, lineHeight: 1.9 }}>
                {(view.foreshadows ?? []).map((f, i) => <li key={i}>{f}</li>)}
              </ul>
            </div>
          )}
          {(view.timeAnchor || view.thread) && (
            <div style={softTextStyle}>
              {view.timeAnchor && <div>时间锚点：{view.timeAnchor}</div>}
              {view.thread && <div>主线：{view.thread}</div>}
            </div>
          )}
          <Typography.Text type="secondary" style={{ fontSize: 11 }}>
            场景圣经由确定性编译（生成链同一份），空区段如实空展示——不知道就是不知道，不编造。
          </Typography.Text>
        </div>
      )}
    </Drawer>
  )
}

function ButtonRetry({ onRetry }: { onRetry: () => void }) {
  return (
    <a onClick={onRetry} style={{ fontSize: 12 }}>重试</a>
  )
}
