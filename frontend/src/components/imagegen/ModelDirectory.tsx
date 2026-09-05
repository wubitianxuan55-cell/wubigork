// ModelDirectory（图像域 T1）：画室「模型目录」创作语境视图——挂在「生成模式」
// 轨道的独立 tab（与素材库/创作资产/识图试用同模式，见 ImageGenPage 接线）。
// 规划口径（docs/gaea-image-domain-longterm-plan-2026.md §T1 + T0 §5 目录协议）：
//   - 数据源唯一 = 模型中心目录：HerdsmanModelCatalog 只读绑定（ModelB 门面
//     legacy 面，经 api/image.ts 的 appFacade 三态回退消费，零新绑定），图像域
//     不另立目录事实源；
//   - 只做「创作语境」展示：能力徽标 / 状态 / 占用全部取目录实有字段；目录未
//     携带的档位与成本元数据诚实留空——成本统一「未定价」，不伪装 0（T0 红线）；
//   - 不重复模型中心管理职责：无引擎启停 / 密钥 / 下载卸载 / 模型启停入口。
// 分组口径（按目录真实字段聚类，不造档位）：capabilities/type/name 关键词归入
// 生图 / 改图 / 视频 / 识图 / 其他图像能力；模型可命中多族，卡片归入其最高能力
// 族（video > img2img > txt2img > vision > other），其余能力以徽标全量呈现；
// 无任何图像关键词的条目（纯 LLM / embedding / tts 等）不进本视图。
// 「当前使用」：生成台后端 = herdsman 且当前模型与目录条目名一致（去符号小写
// 双向包含近似——目录名可能带量化/变体后缀）。创作语境标注，非事实源。
import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { Button, Spin, Tag, Typography } from 'antd'
import { ArrowLeftOutlined, DatabaseOutlined } from '@ant-design/icons'
import {
  herdsmanModelCatalog,
  type HerdsmanCatalogModelView,
  type HerdsmanCatalogView,
} from '../../api/image'
import { useT } from '../../gaea/lib/i18n'
import type { DictKey } from '../../gaea/locales/en'

type TFn = ReturnType<typeof useT>
type Family = 'video' | 'img2img' | 'txt2img' | 'vision' | 'other'

/** 展示顺序（画室创作动线）：生图 → 改图 → 视频 → 识图 → 其他。 */
const FAMILY_ORDER: Family[] = ['txt2img', 'img2img', 'video', 'vision', 'other']
/** 落位优先级（归入最高能力族）：视频 > 改图 > 生图 > 识图 > 其他。 */
const FAMILY_PRIORITY: Family[] = ['video', 'img2img', 'txt2img', 'vision', 'other']

const GROUP_KEY: Record<Family, DictKey> = {
  txt2img: 'imagehubT1.modelDirGroupGen',
  img2img: 'imagehubT1.modelDirGroupEdit',
  video: 'imagehubT1.modelDirGroupVideo',
  vision: 'imagehubT1.modelDirGroupVision',
  other: 'imagehubT1.modelDirGroupOther',
}

/** 目录能力串 → 徽标文案（已知关键词映射；未知能力串原样呈现，诚实不意译）。 */
const CAP_LABELS: Array<[RegExp, DictKey]> = [
  [/text-to-image|image-generation|txt2img/, 'imagehubT1.modelDirCapGen'],
  [/image-to-image|img2img|inpaint|image-edit/, 'imagehubT1.modelDirCapEdit'],
  [/(text|image)-to-video|\bvideo\b|\bt2v\b|\bi2v\b/, 'imagehubT1.modelDirCapVideo'],
  [/image-understanding|vision-language|vision/, 'imagehubT1.modelDirCapVision'],
  [/\bocr\b/, 'imagehubT1.modelDirCapOcr'],
  [/document-parse/, 'imagehubT1.modelDirCapParse'],
]

function capabilityLabel(cap: string, t: TFn): string {
  const c = cap.toLowerCase()
  for (const [re, key] of CAP_LABELS) {
    if (re.test(c)) return t(key)
  }
  return cap
}

/** 按目录真实字段（capabilities/type/name）归族；可命中多族，无图像关键词 = 空族。 */
function modelFamilies(m: HerdsmanCatalogModelView): Family[] {
  const hay = [m.type || '', m.name || '', ...(m.capabilities || [])].join(' ').toLowerCase()
  const out: Family[] = []
  if (/(text|image)-to-video|\bvideo\b|\bt2v\b|\bi2v\b/.test(hay)) out.push('video')
  if (/image-to-image|img2img|inpaint|image-edit/.test(hay)) out.push('img2img')
  if (/text-to-image|image-generation|txt2img/.test(hay) || m.type === 'image') out.push('txt2img')
  if (/\bocr\b|vision|image-understanding|document-parse|multimodal/.test(hay)) out.push('vision')
  if (out.length === 0 && /image/.test(hay)) out.push('other')
  return out
}

const normToken = (s: string): string => s.toLowerCase().replace(/[^a-z0-9]+/g, '')

/** 「当前使用」：herdsman 后端 + 目录名与当前模型去符号小写双向包含（变体后缀容忍）。 */
function isCurrentInStudio(backend: string, model: string, m: HerdsmanCatalogModelView): boolean {
  if (backend !== 'herdsman') return false
  const a = normToken(model || '')
  const b = normToken(m.name || '')
  if (!a || !b) return false
  return a === b || b.includes(a) || a.includes(b)
}

/** 字节 → GB 文案（0/缺失 = 目录未携带，返回空由调用方整行跳过）。 */
function formatBytes(n?: number): string {
  if (!n || n <= 0) return ''
  const gb = n / (1024 * 1024 * 1024)
  return `${gb >= 100 ? gb.toFixed(0) : gb.toFixed(1)} GB`
}

/** 参数量 → B/M 文案（0/缺失 = 目录未携带；CLI 未带单位，按满额计数格式化）。 */
function formatParams(n?: number): string {
  if (!n || n <= 0) return ''
  if (n >= 1e9) return `${Number((n / 1e9).toFixed(1))}B`
  if (n >= 1e6) return `${Number((n / 1e6).toFixed(1))}M`
  return String(n)
}

/** 详情面板单行（目录实有字段才渲染）。 */
const DetailRow: React.FC<{ label: string; value: string }> = ({ label, value }) => (
  <div style={{ display: 'flex', gap: 8 }}>
    <Typography.Text type="secondary" style={{ fontSize: 11, flexShrink: 0 }}>{label}</Typography.Text>
    <Typography.Text style={{ fontSize: 11, wordBreak: 'break-all' }}>{value}</Typography.Text>
  </div>
)

type CatalogEntry = { m: HerdsmanCatalogModelView; families: Family[] }

export const ModelDirectory: React.FC<{
  onClose: () => void
  /** 生成台当前后端（创作语境标注：仅 herdsman 后端可与目录条目对上）。 */
  backend: string
  /** 生成台当前模型（创作语境标注）。 */
  model: string
}> = ({ onClose, backend, model }) => {
  const t = useT()
  const [loading, setLoading] = useState(true)
  const [errMsg, setErrMsg] = useState('')
  const [catalog, setCatalog] = useState<HerdsmanCatalogView>({})
  const [openName, setOpenName] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setErrMsg('')
    try {
      const c = await herdsmanModelCatalog()
      setCatalog(c && typeof c === 'object' ? c : {})
    } catch (e: unknown) {
      setCatalog({})
      setErrMsg((e instanceof Error && e.message) || String(e))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void load() }, [load])

  // 图像相关过滤 + 分族落位（全部由目录真实字段派生；无图像关键词的条目不进视图）。
  const { buckets, imageCount } = useMemo(() => {
    const models = Array.isArray(catalog.models) ? catalog.models : []
    const map = new Map<Family, CatalogEntry[]>()
    let count = 0
    for (const m of models) {
      const families = modelFamilies(m)
      if (families.length === 0) continue
      count += 1
      const primary = FAMILY_PRIORITY.find((f) => families.includes(f)) as Family
      const list = map.get(primary) || []
      list.push({ m, families })
      map.set(primary, list)
    }
    return { buckets: map, imageCount: count }
  }, [catalog])

  const total = typeof catalog.total === 'number' ? catalog.total : (catalog.models?.length || 0)
  const installed = typeof catalog.installed === 'number'
    ? catalog.installed
    : (catalog.models || []).filter((m) => m.installed).length
  const running = typeof catalog.running === 'number'
    ? catalog.running
    : (catalog.models || []).filter((m) => m.running).length

  const cardTagStyle = { marginInlineEnd: 0, fontSize: 10, lineHeight: '16px', borderRadius: 999 } as const

  return (
    <div className="ig-model-directory v3-zone" aria-label={t('imagehubT1.modelDirTitle')}
      style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden', padding: '10px 12px' }}>
      {/* 顶条：返回 + 标题 + 只读目录徽标 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexShrink: 0, flexWrap: 'wrap' }}>
        <Button size="small" icon={<ArrowLeftOutlined />} onClick={onClose}
          style={{ borderRadius: 999, fontSize: 12 }}>
          {t('imagehubT1.libraryBack')}
        </Button>
        <Typography.Text style={{ fontSize: 13, fontWeight: 600 }}>
          {t('imagehubT1.modelDirTitle')}
        </Typography.Text>
        <Tag icon={<DatabaseOutlined />} color="geekblue" style={cardTagStyle}>
          {t('imagehubT1.modelDirNav')}
        </Tag>
      </div>
      <Typography.Text type="secondary" style={{ fontSize: 11, marginTop: 6, flexShrink: 0 }}>
        {t('imagehubT1.modelDirSubtitle')}
      </Typography.Text>

      <div style={{ flex: 1, minHeight: 0, overflowY: 'auto', marginTop: 10, display: 'flex', flexDirection: 'column', gap: 12 }}>
        {loading && (
          <div className="ig-task-empty">
            <Spin size="small" />
            <span>{t('imagehubT1.modelDirLoading')}</span>
          </div>
        )}

        {!loading && errMsg && (
          <div className="ig-task-empty">
            <Typography.Text type="danger" style={{ fontSize: 12, fontWeight: 600 }}>
              {t('imagehubT1.modelDirLoadFail', { msg: errMsg })}
            </Typography.Text>
          </div>
        )}

        {!loading && !errMsg && (
          <>
            {/* 概要行 + 目录来源异常诚实透传（Go 目录自带 error 字段，不吞） */}
            {catalog.error && (
              <Typography.Text type="warning" style={{ fontSize: 11 }}>
                {t('imagehubT1.modelDirCatalogError', { msg: catalog.error })}
              </Typography.Text>
            )}
            <Typography.Text type="secondary" style={{ fontSize: 11 }}>
              {t('imagehubT1.modelDirCount', { n: imageCount, total, installed, running })}
            </Typography.Text>

            {imageCount === 0 && (
              <div className="ig-task-empty">
                <DatabaseOutlined style={{ fontSize: 18, opacity: 0.6 }} />
                <span>{t('imagehubT1.modelDirEmpty')}</span>
              </div>
            )}

            {FAMILY_ORDER.filter((f) => (buckets.get(f)?.length || 0) > 0).map((f) => (
              <section key={f} aria-label={t(GROUP_KEY[f])}
                style={{ display: 'flex', flexDirection: 'column', gap: 8, minWidth: 0 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <Typography.Text style={{ fontSize: 12, fontWeight: 600 }}>
                    {t(GROUP_KEY[f])}
                  </Typography.Text>
                  <Typography.Text type="secondary" style={{ fontSize: 10 }}>
                    {(buckets.get(f) || []).length}
                  </Typography.Text>
                </div>
                {(buckets.get(f) || []).map(({ m, families }) => {
                  const current = isCurrentInStudio(backend, model, m)
                  const name = m.name || ''
                  const open = openName !== '' && openName === name
                  const metaBits = [
                    formatParams(m.parameter_count) ? `${t('imagehubT1.modelDirMetaParams')} ${formatParams(m.parameter_count)}` : '',
                    m.quantization ? `${t('imagehubT1.modelDirMetaQuant')} ${m.quantization}` : '',
                    formatBytes(m.file_size) ? `${t('imagehubT1.modelDirMetaSize')} ${formatBytes(m.file_size)}` : '',
                  ].filter(Boolean)
                  return (
                    <div key={`${f}-${name}`}
                      style={{
                        border: '1px solid',
                        borderColor: current ? 'var(--color-primary)' : 'var(--border-subtle)',
                        borderRadius: 10, background: 'rgba(255,255,255,0.03)', overflow: 'hidden', minWidth: 0,
                      }}>
                      {/* 卡头（点击展开详情；当前使用的卡描边高亮） */}
                      <button type="button" aria-expanded={open}
                        onClick={() => setOpenName(open ? '' : name)}
                        style={{
                          display: 'block', width: '100%', textAlign: 'left', padding: '8px 10px',
                          background: 'transparent', border: 'none', cursor: 'pointer', color: 'inherit',
                        }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
                          <Typography.Text style={{ fontSize: 12, fontWeight: 600 }}>
                            {m.display_name || name}
                          </Typography.Text>
                          {m.display_name && name && m.display_name !== name && (
                            <Typography.Text type="secondary" style={{ fontSize: 10 }}>{name}</Typography.Text>
                          )}
                          {current && (
                            <Tag color="gold" style={cardTagStyle}>{t('imagehubT1.modelDirCurrent')}</Tag>
                          )}
                        </div>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 4, flexWrap: 'wrap', marginTop: 4 }}>
                          {m.running && <Tag color="green" style={cardTagStyle}>{t('imagehubT1.modelDirRunning')}</Tag>}
                          {m.installed
                            ? <Tag style={cardTagStyle}>{t('imagehubT1.modelDirInstalled')}</Tag>
                            : <Tag style={{ ...cardTagStyle, opacity: 0.7 }}>{t('imagehubT1.modelDirNotInstalled')}</Tag>}
                          {(m.capabilities || []).map((c) => (
                            <Tag key={c} color="blue" style={cardTagStyle}>{capabilityLabel(c, t)}</Tag>
                          ))}
                        </div>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexWrap: 'wrap', marginTop: 4 }}>
                          {metaBits.map((bit) => (
                            <Typography.Text key={bit} type="secondary" style={{ fontSize: 10 }}>{bit}</Typography.Text>
                          ))}
                          {/* 目录未携带成本元数据 → 统一诚实「未定价」，不伪装 0（T0 红线） */}
                          <Typography.Text type="secondary" style={{ fontSize: 10 }}>
                            {t('imagehubT1.metaCost')}: {t('imagehubT1.costUnset')}
                          </Typography.Text>
                        </div>
                      </button>
                      {/* 详情面板（仅目录实有字段；档位/成本未携带 → 诚实披露） */}
                      {open && (
                        <div style={{
                          padding: '8px 10px', borderTop: '1px solid var(--border-subtle)',
                          display: 'flex', flexDirection: 'column', gap: 4,
                        }}>
                          <Typography.Text type="secondary" style={{ fontSize: 10, fontWeight: 600 }}>
                            {t('imagehubT1.modelDirDetailTitle')}
                          </Typography.Text>
                          <DetailRow label={t('imagehubT1.modelDirTierLabel')} value={t('imagehubT1.modelDirTierUnset')} />
                          <DetailRow label={t('imagehubT1.metaCost')} value={t('imagehubT1.costUnset')} />
                          {m.status && <DetailRow label={t('imagehubT1.modelDirMetaStatus')} value={m.status} />}
                          {m.run_status && <DetailRow label={t('imagehubT1.modelDirMetaRunStatus')} value={m.run_status} />}
                          {m.runtime && <DetailRow label={t('imagehubT1.modelDirMetaRuntime')} value={m.runtime} />}
                          {(m.inference_engines || []).length > 0 && (
                            <DetailRow label={t('imagehubT1.modelDirMetaEngines')} value={(m.inference_engines || []).join(', ')} />
                          )}
                          {formatParams(m.active_parameters) && (
                            <DetailRow label={t('imagehubT1.modelDirActiveParams')} value={formatParams(m.active_parameters)} />
                          )}
                          {m.is_moe && <DetailRow label="MoE" value="true" />}
                          {(m.llama_cpp_variants || []).length > 0 && (
                            <DetailRow label={t('imagehubT1.modelDirMetaVariants')} value={(m.llama_cpp_variants || []).join(', ')} />
                          )}
                          {(families.length > 1) && (
                            <DetailRow
                              label={t('imagehubT1.modelDirFamilies')}
                              value={families.map((fm) => t(GROUP_KEY[fm])).join(' / ')} />
                          )}
                          {m.hint && (
                            <div style={{ background: 'rgba(255,255,255,0.04)', borderRadius: 8, padding: '6px 8px' }}>
                              <Typography.Text type="secondary" style={{ fontSize: 10 }}>
                                {t('imagehubT1.modelDirHintTitle')}
                              </Typography.Text>
                              <Typography.Paragraph style={{ marginBottom: 0, fontSize: 11, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                                {m.hint}
                              </Typography.Paragraph>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  )
                })}
              </section>
            ))}
          </>
        )}
      </div>

      {/* 底部一行：数据源与口径说明（与模型中心目录同源；管理职责在模型中心） */}
      <Typography.Text type="secondary" style={{ fontSize: 10, flexShrink: 0, marginTop: 8 }}>
        {t('imagehubT1.modelDirSourceNote')}
      </Typography.Text>
    </div>
  )
}

export default ModelDirectory
