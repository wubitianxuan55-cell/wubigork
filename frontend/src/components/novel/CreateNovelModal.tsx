import React, { useState } from 'react'
import { Typography, Space, Input, Modal, Checkbox, Radio, Button, Select, Tag, message } from 'antd'
import { C } from '../../utils/theme'
import { GENRE_OPTIONS, STYLE_OPTIONS } from './novelOptions'
import { app } from '../../gaea/lib/bridge'
import type { NovelScanResult } from '../../gaea/lib/bridge/novel'

interface CreateNovelModalProps {
  open: boolean
  onClose: () => void
  onCreate: () => Promise<void>
  title: string; onTitleChange: (v: string) => void
  genre: string[]; onGenreChange: (v: string[]) => void
  style: string[]; onStyleChange: (v: string[]) => void
  /** 成人向档位（v4.441 建档同址可选；''=非成人向。落盘走 HomePage 的 UpdateProjectMeta） */
  mature: string; onMatureChange: (v: string) => void
}

/**
 * 新建小说 Modal（标题/题材/文风是父层 HomePage 的受控 state——本组件只收
 * props 不接管）。
 * v4.421.0 收口两处体验缺陷（对齐 ImportNovelModal 的正确做法）：
 *  ① 空标题此前静默无反应（HomePage.handleCreate 直接 return）：OK 按钮按
 *     `!title.trim()` 禁用，模态内再加一道提交前兜底提示；
 *  ② `onCreate()` 在途可重复点击（antd Modal 的 onOk 不管 loading，非受控
 *     返回值不会自动置 loading）→ 本地 submitting 门闩 + confirmLoading，
 *     在途重复点击直接吞掉（confirmLoading 同时挡住 ✕/Esc 误关）。
 */
const CreateNovelModal: React.FC<CreateNovelModalProps> = ({
  open, onClose, onCreate,
  title, onTitleChange,
  genre, onGenreChange,
  style, onStyleChange,
  mature, onMatureChange,
}) => {
  const [submitting, setSubmitting] = useState(false)
  const emptyTitle = !title.trim()

  // ── 选题扫榜（oh-story story-long-scan 蒸馏；粘贴通路无爬虫）────────
  // 报告只展示不落盘（建议制红线）；「采用题材」把选题的题材组合映射进
  // GENRE_OPTIONS 勾选——作者显式动作，不静默改表单。
  const [scanOpen, setScanOpen] = useState(false)
  const [scanPlatform, setScanPlatform] = useState('')
  const [scanText, setScanText] = useState('')
  const [scanning, setScanning] = useState(false)
  const [scanRes, setScanRes] = useState<NovelScanResult | null>(null)
  const [scanErr, setScanErr] = useState('')

  const runScan = async () => {
    if (scanning) return
    setScanning(true)
    setScanErr('')
    setScanRes(null)
    try {
      setScanRes(await app.NovelScanAnalyze(scanPlatform, scanText))
    } catch (e) {
      setScanErr(e instanceof Error ? e.message : String(e))
    } finally {
      setScanning(false)
    }
  }

  const adoptGenre = (mix: string) => {
    const hits = GENRE_OPTIONS.filter((g) => mix.includes(g))
    if (hits.length > 0) onGenreChange(hits)
  }

  const handleOk = async () => {
    if (submitting) return // 门闩：在途重复点击不再起第二次创建
    if (emptyTitle) {
      // 兜底（OK 已禁用，正常点不到）：不静默、也不关窗
      message.warning('请填写书名')
      return
    }
    setSubmitting(true)
    try {
      await onCreate()
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Modal
      title={<span style={{ color: C('color-text') }}>新建小说</span>}
      open={open}
      onOk={() => void handleOk()}
      onCancel={onClose}
      okText={submitting ? '创建中…' : '创建'}
      cancelText="取消"
      confirmLoading={submitting}
      okButtonProps={{ disabled: emptyTitle || submitting }}
      width={520}
      // WebView2 冻结 rAF 时退出动画不结束会残留遮罩卡死界面：关闭即卸载。
      destroyOnHidden
      transitionName=""
      maskTransitionName=""
      styles={{
        body: { background: 'transparent' },
        header: { background: 'transparent' },
      }}
    >
      <Space direction="vertical" size={14} style={{ width: '100%' }}>
        <Input
          placeholder="小说标题（必填）"
          value={title}
          onChange={(e) => onTitleChange(e.target.value)}
          style={{ background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }}
        />

        {/* 题材 */}
        <div>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, marginBottom: 6, display: 'block' }}>
            题材（可多选）
          </Typography.Text>
          <Checkbox.Group
            options={GENRE_OPTIONS}
            value={genre}
            onChange={(v) => onGenreChange(v as string[])}
            style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 12px' }}
          />
        </div>

        {/* 文风 */}
        <div>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, marginBottom: 6, display: 'block' }}>
            文风（可多选）
          </Typography.Text>
          <Checkbox.Group
            options={STYLE_OPTIONS}
            value={style}
            onChange={(v) => onStyleChange(v as string[])}
            style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 12px' }}
          />
        </div>

        {/* 本书定位（成人向档位）——与创作页「本书定位」同口径，创建后可改 */}
        <div>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, marginBottom: 6, display: 'block' }}>
            本书定位
          </Typography.Text>
          <Radio.Group
            value={mature}
            onChange={(e) => onMatureChange(e.target.value as string)}
            options={[
              { value: '', label: '非成人向' },
              { value: 'sensual', label: '成人向 · 含蓄（张力与留白）' },
              { value: 'explicit', label: '成人向 · 直白（亲密戏正面直书）' },
            ]}
            style={{ display: 'flex', flexWrap: 'wrap', gap: '4px 12px' }}
          />
        </div>

        {/* 选题扫榜（oh-story story-long-scan 蒸馏）：粘贴榜单 → 八维观察+选题决策；只读参考不落盘 */}
        <div>
          <Typography.Link style={{ fontSize: 13 }} onClick={() => setScanOpen((v) => !v)}>
            扫榜找灵感（选题参考）{scanOpen ? '▴' : '▾'}
          </Typography.Link>
          {scanOpen && (
            <div style={{ marginTop: 8 }}>
              <Select
                value={scanPlatform || undefined}
                onChange={(v) => setScanPlatform(v || '')}
                allowClear
                placeholder="平台（留空按标题自动识别）"
                style={{ width: '100%', marginBottom: 6 }}
                options={[
                  { value: '起点', label: '起点' },
                  { value: '番茄', label: '番茄' },
                  { value: '晋江', label: '晋江（女频）' },
                  { value: '七猫', label: '七猫' },
                  { value: '刺猬猫', label: '刺猬猫' },
                  { value: '知乎盐言', label: '知乎盐言（短篇）' },
                ]}
              />
              <Input.TextArea
                rows={5}
                value={scanText}
                onChange={(e) => setScanText(e.target.value)}
                placeholder={'粘贴榜单文本（至少 50 字），首行可用「# 起点·风云榜」标注平台，每行一本书'}
                style={{ background: C('color-bg-layout'), borderColor: C('color-border'), color: C('color-text') }}
              />
              <div style={{ marginTop: 6, display: 'flex', alignItems: 'center', gap: 8 }}>
                <Button
                  size="small"
                  type="primary"
                  loading={scanning}
                  disabled={scanText.trim().length < 50}
                  onClick={() => void runScan()}
                >
                  分析选题
                </Button>
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  只读参考：不写盘、不改书稿
                </Typography.Text>
              </div>
              {scanErr && (
                <Typography.Text type="danger" style={{ fontSize: 12, display: 'block', marginTop: 6 }}>
                  {scanErr}
                </Typography.Text>
              )}
              {scanRes && (
                <div style={{ marginTop: 8 }}>
                  <Typography.Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 6 }}>
                    {scanRes.source?.disclaimer}（样本 {scanRes.source?.entry_count ?? 0} 条 · {scanRes.source?.platform}）
                  </Typography.Text>
                  {(scanRes.topics ?? []).map((t, i) => (
                    <div
                      key={i}
                      style={{ border: `1px solid ${C('color-border')}`, borderRadius: 6, padding: '8px 10px', marginBottom: 6 }}
                    >
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <Tag color={t.feasibility === '高' ? 'green' : t.feasibility === '中' ? 'orange' : 'default'}>
                          可行性{t.feasibility}
                        </Tag>
                        <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>
                          {t.direction}
                        </Typography.Text>
                      </div>
                      <div style={{ fontSize: 12, color: C('color-text-secondary'), marginTop: 4 }}>
                        {[t.genre_mix, t.selling_point, t.differentiation].filter(Boolean).join(' · ')}
                      </div>
                      {t.market_evidence && (
                        <div style={{ fontSize: 12, color: C('color-text-secondary'), marginTop: 2 }}>依据：{t.market_evidence}</div>
                      )}
                      {t.validation && <div style={{ fontSize: 12, marginTop: 2 }}>验证：{t.validation}</div>}
                      {t.feasibility_reason && (
                        <div style={{ fontSize: 12, color: C('color-text-secondary'), marginTop: 2 }}>{t.feasibility_reason}</div>
                      )}
                      {t.genre_mix && (
                        <Button size="small" style={{ marginTop: 6 }} onClick={() => adoptGenre(t.genre_mix)}>
                          采用题材
                        </Button>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </Space>
    </Modal>
  )
}

export default CreateNovelModal
