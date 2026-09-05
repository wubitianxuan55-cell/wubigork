// AssetStudio（图像域 T1）：画室「创作资产」面板——挂在「生成模式」轨道的独立
// tab（与 v4.99 素材库同模式，见 ImageGenPage 接线）。三槽均为只读浏览：
//   - 角色槽：CharacterList 分页拉取可聊天角色（既有绑定，零新绑定；mock 见
//     gaea/lib/mock/weixin.ts），选中带出角色数据摘要（立绘 / 性格 / 背景 /
//     外貌 / 标签）。带入生成台 = 页面级 applyRefCharacter（角色参考槽既有
//     回填路径，无参考图时由其诚实提示）；同时提供「复制设定」手动复制路径，
//     不硬拼提示词；
//   - 模板槽：绘梦既有模板库（内置 TEMPLATES + 自定义模板）复用展示，搜索 +
//     懒加载；选中走页面级 applyTemplate 回填生成表单既有字段；
//   - 近期作品：ImageHubAssets 读当前空间前 12 张（work/play 可切换，按空间
//     隔离），缩略懒读；点击跳素材库看完整溯源。
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Button, Input, Segmented, Spin, Tag, Typography, message } from 'antd'
import {
  ArrowLeftOutlined, FileImageOutlined, TeamOutlined, UserOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'
import { chatCharacters, imageHubAssets, readFileAsDataURL, type ChatCharacterView, type ImageHubAssetView } from '../../api/image'
import type { Template } from '../../data/imageTemplates'
import { useT } from '../../gaea/lib/i18n'

const CHARS_PAGE_SIZE = 12
const TPL_PAGE_SIZE = 18
const WORKS_LIMIT = 12

interface WorkThumb extends ImageHubAssetView {
  url: string
}

export const AssetStudio: React.FC<{
  onClose: () => void
  onOpenLibrary: () => void
  templates: Template[]
  customTemplates: Template[]
  onApplyTemplate: (t: Template) => void
  onApplyCharacter: (id: string) => void
}> = ({ onClose, onOpenLibrary, templates, customTemplates, onApplyTemplate, onApplyCharacter }) => {
  const t = useT()

  // ── 角色槽（CharacterList 分页追加）──
  const [chars, setChars] = useState<ChatCharacterView[]>([])
  const [charTotal, setCharTotal] = useState(0)
  const [charPage, setCharPage] = useState(1)
  const [charsLoading, setCharsLoading] = useState(false)
  const [selectedChar, setSelectedChar] = useState<ChatCharacterView | null>(null)

  const loadChars = useCallback(async (page: number) => {
    setCharsLoading(true)
    try {
      const { items, total } = await chatCharacters(page, CHARS_PAGE_SIZE)
      setCharTotal(total)
      setCharPage(page)
      setChars((prev) => page === 1 ? items : [...prev, ...items])
    } finally {
      setCharsLoading(false)
    }
  }, [])

  useEffect(() => { void loadChars(1) }, [loadChars])

  // 角色设定文本（复制路径用）：只拼真实存在的字段，不做任何填空式硬拼。
  const selectedProfileText = useMemo(() => {
    if (!selectedChar) return ''
    const lines: string[] = []
    if (selectedChar.name) lines.push(selectedChar.name)
    if (selectedChar.roleType) lines.push(selectedChar.roleType)
    if (selectedChar.personality) lines.push(`${t('imagehubT1.studioCharPersonality')}：${selectedChar.personality}`)
    if (selectedChar.background) lines.push(`${t('imagehubT1.studioCharBackground')}：${selectedChar.background}`)
    if (selectedChar.appearance) lines.push(`${t('imagehubT1.studioCharAppearance')}：${selectedChar.appearance}`)
    if (selectedChar.tags?.length) lines.push(selectedChar.tags.join(' / '))
    return lines.join('\n')
  }, [selectedChar, t])

  const copyProfile = useCallback(async () => {
    if (!selectedChar || !selectedProfileText) return
    try {
      await navigator.clipboard.writeText(selectedProfileText)
      message.success(t('imagehubT1.studioCharCopiedMsg'))
    } catch {
      message.warning(t('imagehubT1.studioCharCopyFail'))
    }
  }, [selectedChar, selectedProfileText, t])

  // ── 模板槽（绘梦既有模板库复用：内置 + 自定义，搜索 + 懒加载）──
  const allTemplates = useMemo(() => [...customTemplates, ...templates], [customTemplates, templates])
  const [tplQuery, setTplQuery] = useState('')
  const [tplVisible, setTplVisible] = useState(TPL_PAGE_SIZE)
  const filteredTemplates = useMemo(() => {
    const q = tplQuery.trim().toLowerCase()
    if (!q) return allTemplates
    return allTemplates.filter((tp) =>
      tp.label.toLowerCase().includes(q)
      || (tp.description || '').toLowerCase().includes(q)
      || tp.prompt.toLowerCase().includes(q)
      || (tp.tags || []).some((x) => x.toLowerCase().includes(q)))
  }, [allTemplates, tplQuery])
  const visibleTemplates = useMemo(
    () => filteredTemplates.slice(0, tplVisible),
    [filteredTemplates, tplVisible])

  // ── 近期作品（ImageHubAssets 前 12 张，按空间隔离，缩略懒读）──
  const [space, setSpace] = useState<'work' | 'play'>('work')
  const [works, setWorks] = useState<ImageHubAssetView[]>([])
  const [worksLoading, setWorksLoading] = useState(false)
  const [workThumbs, setWorkThumbs] = useState<WorkThumb[]>([])
  const urlCacheRef = useRef<Map<string, string>>(new Map())

  useEffect(() => {
    let cancelled = false
    setWorksLoading(true)
    void (async () => {
      const list = await imageHubAssets(space, '', WORKS_LIMIT)
      if (cancelled) return
      setWorks(list)
      setWorksLoading(false)
    })()
    return () => { cancelled = true }
  }, [space])

  useEffect(() => {
    let cancelled = false
    void (async () => {
      for (const w of works) {
        const p = w.path || ''
        if (!p || urlCacheRef.current.has(p)) continue
        const url = await readFileAsDataURL(p).catch(() => '')
        if (cancelled) return
        urlCacheRef.current.set(p, url || '')
      }
      if (cancelled) return
      setWorkThumbs(works.map((w) => ({ ...w, url: (w.path && urlCacheRef.current.get(w.path)) || '' })))
    })()
    return () => { cancelled = true }
  }, [works])

  return (
    <div className="ig-asset-studio v3-zone" aria-label={t('imagehubT1.studioTitle')}
      style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden', padding: '10px 12px' }}>
      {/* 顶条：返回 + 标题 + 去素材库出口 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexShrink: 0, flexWrap: 'wrap' }}>
        <Button size="small" icon={<ArrowLeftOutlined />} onClick={onClose}
          style={{ borderRadius: 999, fontSize: 12 }}>
          {t('imagehubT1.libraryBack')}
        </Button>
        <Typography.Text style={{ fontSize: 13, fontWeight: 600 }}>
          {t('imagehubT1.studioTitle')}
        </Typography.Text>
        <Typography.Text type="secondary" style={{ fontSize: 11 }}>
          {t('imagehubT1.studioSubtitle')}
        </Typography.Text>
        <div style={{ marginLeft: 'auto' }}>
          <Button size="small" icon={<FileImageOutlined />} onClick={onOpenLibrary}
            style={{ borderRadius: 999, fontSize: 12 }}>
            {t('imagehubT1.studioWorksOpenLib')}
          </Button>
        </div>
      </div>

      <div style={{ flex: 1, minHeight: 0, overflowY: 'auto', marginTop: 10, display: 'flex', flexDirection: 'column', gap: 14 }}>
        {/* ── 角色槽 ── */}
        <section style={{ flexShrink: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <TeamOutlined style={{ color: C('color-text-secondary') }} />
            <Typography.Text style={{ fontSize: 12, fontWeight: 600 }}>{t('imagehubT1.studioCharsTitle')}</Typography.Text>
            <Typography.Text type="secondary" style={{ fontSize: 11 }}>
              {t('imagehubT1.studioCharsTotal', { n: charTotal })}
            </Typography.Text>
            {charsLoading && <Spin size="small" />}
          </div>
          {chars.length === 0 && !charsLoading ? (
            <Typography.Text type="secondary" style={{ fontSize: 11, display: 'block', marginTop: 6 }}>
              {t('imagehubT1.studioCharsEmpty')}
            </Typography.Text>
          ) : (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(120px, 1fr))', gap: 8, marginTop: 8 }}>
              {chars.map((c) => (
                <button key={c.id || c.name} type="button" onClick={() => setSelectedChar(c)}
                  style={{
                    padding: 6, textAlign: 'left', cursor: 'pointer',
                    border: '1px solid',
                    borderColor: selectedChar?.id === c.id ? 'var(--color-primary)' : 'var(--border-subtle)',
                    borderRadius: 10, background: 'rgba(255,255,255,0.03)',
                  }}>
                  {c.portraitUrl ? (
                    <img src={c.portraitUrl} alt={c.name || ''}
                      style={{ width: '100%', aspectRatio: '1 / 1', objectFit: 'cover', borderRadius: 7, display: 'block' }} />
                  ) : (
                    <div style={{
                      width: '100%', aspectRatio: '1 / 1', borderRadius: 7, display: 'flex',
                      flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 4,
                      background: 'rgba(255,255,255,0.04)',
                    }}>
                      <UserOutlined style={{ fontSize: 18, color: C('color-text-secondary'), opacity: 0.6 }} />
                      <span style={{ fontSize: 10, color: C('color-text-secondary') }}>
                        {t('imagehubT1.studioCharNoPortrait')}
                      </span>
                    </div>
                  )}
                  <div style={{ fontSize: 12, fontWeight: 600, marginTop: 4, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                    {c.name || '-'}
                  </div>
                  {(c.tags || []).length > 0 && (
                    <div style={{ fontSize: 10, color: C('color-text-secondary'), whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                      {c.tags!.slice(0, 3).join(' · ')}
                    </div>
                  )}
                </button>
              ))}
            </div>
          )}
          {chars.length < charTotal && (
            <div style={{ marginTop: 8 }}>
              <Button size="small" onClick={() => void loadChars(charPage + 1)}
                style={{ borderRadius: 999, fontSize: 12 }}>
                {t('imagehubT1.loadMore')}
              </Button>
            </div>
          )}

          {/* 角色数据摘要（立绘 + 设定片段）+ 带入/复制两路径 */}
          {selectedChar && (
            <div style={{
              marginTop: 10, padding: '10px 12px', borderRadius: 10,
              border: '1px solid var(--border-subtle)', background: 'rgba(255,255,255,0.03)',
              display: 'flex', gap: 12,
            }}>
              {selectedChar.portraitUrl ? (
                <img src={selectedChar.portraitUrl} alt={selectedChar.name || ''}
                  style={{ width: 96, height: 96, objectFit: 'cover', borderRadius: 8, flexShrink: 0 }} />
              ) : (
                <div style={{
                  width: 96, height: 96, borderRadius: 8, flexShrink: 0, display: 'flex',
                  flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 4,
                  background: 'rgba(255,255,255,0.04)',
                }}>
                  <UserOutlined style={{ fontSize: 20, color: C('color-text-secondary'), opacity: 0.6 }} />
                  <span style={{ fontSize: 10, color: C('color-text-secondary') }}>{t('imagehubT1.studioCharNoPortrait')}</span>
                </div>
              )}
              <div style={{ minWidth: 0, display: 'flex', flexDirection: 'column', gap: 4, fontSize: 11 }}>
                <Typography.Text style={{ fontSize: 12, fontWeight: 600 }}>
                  {t('imagehubT1.studioCharSummary')} · {selectedChar.name || '-'}
                </Typography.Text>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                  {selectedChar.roleType && <Tag color="purple" style={{ marginInlineEnd: 0 }}>{selectedChar.roleType}</Tag>}
                  {(selectedChar.tags || []).map((x) => <Tag key={x} style={{ marginInlineEnd: 0 }}>{x}</Tag>)}
                </div>
                {selectedChar.personality && (
                  <span style={{ minWidth: 0 }}>
                    <Typography.Text type="secondary">{t('imagehubT1.studioCharPersonality')}: </Typography.Text>
                    <Typography.Text style={{ fontSize: 11 }}>{selectedChar.personality}</Typography.Text>
                  </span>
                )}
                {selectedChar.background && (
                  <span style={{ minWidth: 0 }}>
                    <Typography.Text type="secondary">{t('imagehubT1.studioCharBackground')}: </Typography.Text>
                    <Typography.Text style={{ fontSize: 11 }}>{selectedChar.background}</Typography.Text>
                  </span>
                )}
                {selectedChar.appearance && (
                  <span style={{ minWidth: 0 }}>
                    <Typography.Text type="secondary">{t('imagehubT1.studioCharAppearance')}: </Typography.Text>
                    <Typography.Text style={{ fontSize: 11 }}>{selectedChar.appearance}</Typography.Text>
                  </span>
                )}
                <div style={{ display: 'flex', gap: 8, marginTop: 2 }}>
                  <Button size="small" type="primary" disabled={!selectedChar.id}
                    onClick={() => selectedChar.id && onApplyCharacter(selectedChar.id)}>
                    {t('imagehubT1.studioCharApply')}
                  </Button>
                  <Button size="small" onClick={() => void copyProfile()}>
                    {t('imagehubT1.studioCharCopy')}
                  </Button>
                </div>
              </div>
            </div>
          )}
        </section>

        {/* ── 模板槽 ── */}
        <section style={{ flexShrink: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
            <Typography.Text style={{ fontSize: 12, fontWeight: 600 }}>{t('imagehubT1.studioTplTitle')}</Typography.Text>
            <Input size="small" allowClear value={tplQuery} style={{ width: 220 }}
              placeholder={t('imagehubT1.studioTplSearch')}
              onChange={(e) => { setTplQuery(e.target.value); setTplVisible(TPL_PAGE_SIZE) }} />
          </div>
          {allTemplates.length === 0 ? (
            <Typography.Text type="secondary" style={{ fontSize: 11, display: 'block', marginTop: 6 }}>
              {t('imagehubT1.studioTplEmpty')}
            </Typography.Text>
          ) : filteredTemplates.length === 0 ? (
            <Typography.Text type="secondary" style={{ fontSize: 11, display: 'block', marginTop: 6 }}>
              {t('imagehubT1.emptyFiltered')}
            </Typography.Text>
          ) : (
            <>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: 8, marginTop: 8 }}>
                {visibleTemplates.map((tp, i) => (
                  <button key={`${tp.id || 'tpl'}-${i}`} type="button" onClick={() => onApplyTemplate(tp)}
                    title={tp.prompt}
                    style={{
                      padding: '8px 10px', textAlign: 'left', cursor: 'pointer',
                      border: '1px solid var(--border-subtle)', borderRadius: 10,
                      background: 'rgba(255,255,255,0.03)', display: 'flex', flexDirection: 'column', gap: 4,
                    }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      {tp.icon && <span style={{ fontSize: 12 }}>{tp.icon}</span>}
                      <Typography.Text style={{ fontSize: 12, fontWeight: 600 }}>{tp.label}</Typography.Text>
                      {tp.size && <Tag style={{ marginInlineEnd: 0, fontSize: 10, lineHeight: '16px', borderRadius: 999 }}>{tp.size}</Tag>}
                    </div>
                    {tp.description && (
                      <Typography.Text type="secondary" style={{ fontSize: 10.5, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', display: 'block' }}>
                        {tp.description}
                      </Typography.Text>
                    )}
                    <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                      {(tp.tags || []).slice(0, 3).map((x) => (
                        <Tag key={x} style={{ marginInlineEnd: 0, fontSize: 10, lineHeight: '16px', borderRadius: 999 }}>{x}</Tag>
                      ))}
                      <span style={{ marginLeft: 'auto', fontSize: 11, color: 'var(--color-primary)' }}>
                        {t('imagehubT1.studioTplApply')}
                      </span>
                    </div>
                  </button>
                ))}
              </div>
              {filteredTemplates.length > tplVisible && (
                <div style={{ marginTop: 8 }}>
                  <Button size="small" onClick={() => setTplVisible((n) => n + TPL_PAGE_SIZE)}
                    style={{ borderRadius: 999, fontSize: 12 }}>
                    {t('imagehubT1.loadMore')}
                  </Button>
                </div>
              )}
            </>
          )}
        </section>

        {/* ── 近期作品 ── */}
        <section style={{ flexShrink: 0, paddingBottom: 6 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
            <Typography.Text style={{ fontSize: 12, fontWeight: 600 }}>{t('imagehubT1.studioWorksTitle')}</Typography.Text>
            <Typography.Text type="secondary" style={{ fontSize: 11 }}>
              {t('imagehubT1.studioWorksHint')}
            </Typography.Text>
            <div style={{ marginLeft: 'auto' }}>
              <Segmented size="small" value={space} onChange={(v) => setSpace(v as 'work' | 'play')}
                options={[
                  { label: t('imagehubT1.spaceWork'), value: 'work' },
                  { label: t('imagehubT1.spacePlay'), value: 'play' },
                ]} />
            </div>
          </div>
          {worksLoading ? (
            <div className="ig-task-empty" style={{ marginTop: 8 }}><Spin size="small" /></div>
          ) : works.length === 0 ? (
            <div className="ig-task-empty" style={{ marginTop: 8 }}>
              <FileImageOutlined style={{ fontSize: 18, color: C('color-text-secondary'), opacity: 0.6 }} />
              <span>{t('imagehubT1.empty')}</span>
            </div>
          ) : (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(120px, 1fr))', gap: 8, marginTop: 8 }}>
              {workThumbs.map((w) => (
                <button key={w.path} type="button" onClick={onOpenLibrary} title={w.path || ''}
                  style={{
                    padding: 4, textAlign: 'left', cursor: 'pointer',
                    border: '1px solid var(--border-subtle)', borderRadius: 10, background: 'rgba(255,255,255,0.03)',
                  }}>
                  {w.url ? (
                    <img src={w.url} alt={w.path || ''}
                      style={{ width: '100%', aspectRatio: '1 / 1', objectFit: 'cover', borderRadius: 7, display: 'block' }} />
                  ) : (
                    <div style={{
                      width: '100%', aspectRatio: '1 / 1', borderRadius: 7, display: 'flex',
                      alignItems: 'center', justifyContent: 'center', background: 'rgba(255,255,255,0.04)',
                    }}>
                      <FileImageOutlined style={{ fontSize: 16, color: C('color-text-secondary'), opacity: 0.6 }} />
                    </div>
                  )}
                  <div style={{ fontSize: 10, color: C('color-text-secondary'), marginTop: 4, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                    {[w.model, w.created_at].filter(Boolean).join(' · ') || w.path}
                  </div>
                </button>
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  )
}
