import type { ReactNode } from 'react'
import { Button, message, Select, Switch } from 'antd'
import {
  CommentOutlined, EditOutlined, LinkOutlined, PictureOutlined,
  RobotOutlined, SoundOutlined, ToolOutlined, UserOutlined,
} from '@ant-design/icons'
import * as App from '../../../src/wailsjsCompat'
import { SectionHead, StatusChip, type StatusTone } from './ui'
import {
  engineLabel, FEATURES, featureState, featureStateMeta,
  modelOptionsForEngine, routeSourceLabel,
} from './utils'
import { useModelCenter } from './context'

const FEATURE_ICONS: Record<string, ReactNode> = {
  chat: <CommentOutlined />,
  novel: <EditOutlined />,
  office: <ToolOutlined />,
  characterlib: <UserOutlined />,
  routine: <RobotOutlined />,
}

const FEATURE_NOTES: Record<string, string> = {
  chat: '聊天语音：TTS 与 STT 跟随语音管道，绑定的是对话大模型',
  office: '通用 + 方案 + 知识库',
  characterlib: '生成 / 补全',
  routine: '纯文本摘要/归一化/抽取/改写等无专业工具覆盖的活，由云端 agent 通过 routine_llm 兜底调用（识图/OCR/检索/转换走各自专业工具）；未绑定默认本地 herdsman',
}

const popupContainer = () => document.body

const stateTone = (color: string): StatusTone =>
  color === 'green' ? 'ok' : color === 'orange' ? 'warn' : 'neutral'

export function BindSection() {
  const {
    engines, featureCfg, featureDraft, featureEnabled, modelRoutes,
    chatVoiceCfg, chatVoiceDraft, chatVoiceSaving, chatVoiceOptions, chatVoiceValue,
    setVoiceCfg,
    portraitCfg, portraitDraft, portraitModelOptions, portraitSaving,
    llmModels, ttsModels,
    setFeatureDraft, setChatVoiceDraft, setPortraitDraft,
    handleSaveFeature, handleToggleFeatureEnabled,
    handleSaveChatVoice, handleClearChatVoice, handleSavePortrait,
  } = useModelCenter()

  return (
    <section className="mc-section">
      <SectionHead
        icon={<LinkOutlined />}
        title="功能模型绑定"
        desc="各功能板块独立模型，设置后持久化（重启不丢）"
      />

      <div className="mc-grid two-col">
        {FEATURES.map(f => {
          const cur = featureCfg[f.key]
          const draft = featureDraft[f.key] || { engine: '', model: '' }
          const engineModelOptions = modelOptionsForEngine(draft.engine || '', llmModels, draft.model)
          const bound = !!cur?.engine && !!cur?.model
          const enabled = featureEnabled[f.key] !== false
          const state = featureState(bound, enabled)
          const stateMeta = featureStateMeta[state]
          // 绑定/路由文本展示友好名：引擎取 engineLabel，模型经 llmModels
          // 的 modelName（Model Hub 的别名 ID → tinyrick/…:Q6_K_P）。
          const modelText = (engineId?: string, modelId?: string): string => {
            const hit = engineId && modelId
              ? llmModels.find(m => m.engineId === engineId && m.modelId === modelId)
              : undefined
            return [engineId ? engineLabel({ id: engineId }) : '', hit?.modelName || modelId || '']
              .filter(Boolean).join(' / ') || '-'
          }
          return (
            <div key={f.key} className={`mc-bind-card${bound ? ' is-bound' : ''}`}>
              <div className="mc-bind-head">
                <span className="mc-bind-title">
                  {FEATURE_ICONS[f.key] || <LinkOutlined />}
                  {f.label}
                </span>
                <StatusChip tone={stateTone(stateMeta.color)} dot>{stateMeta.label}</StatusChip>
              </div>
              {FEATURE_NOTES[f.key] && (
                <div className="mc-bind-desc">{FEATURE_NOTES[f.key]}</div>
              )}
              <div className="mc-bind-meta">
                {state === 'fallback'
                  ? '未绑定，跟随全局默认'
                  : state === 'bound-disabled'
                    ? `已停用，跟随全局默认（绑定保留：${modelText(cur!.engine, cur!.model)}）`
                    : `当前：${modelText(cur!.engine, cur!.model)}`}
              </div>
              {modelRoutes[f.key] && (
                <div className="mc-bind-meta">
                  当前生效：{modelText(modelRoutes[f.key].engine, modelRoutes[f.key].model)}
                  （{routeSourceLabel(modelRoutes[f.key].source)}）
                </div>
              )}
              {f.key === 'office' && modelRoutes['gaea'] && (
                <div className="mc-bind-meta">
                  通用办公 / 知识库路由：{modelText(modelRoutes['gaea'].engine, modelRoutes['gaea'].model)}
                  （{routeSourceLabel(modelRoutes['gaea'].source)}）
                </div>
              )}
              <div className="mc-bind-row">
                <Select
                  size="small"
                  placeholder="引擎"
                  value={draft.engine || undefined}
                  getPopupContainer={popupContainer}
                  onChange={(v: string) => setFeatureDraft(p => ({ ...p, [f.key]: { engine: v, model: '' } }))}
                  style={{ flex: 1, minWidth: 0 }}
                  options={engines.filter(e => e.enabled).map(e => ({ value: e.id, label: engineLabel(e) }))}
                />
                <Select
                  size="small"
                  placeholder="模型"
                  value={draft.model || undefined}
                  getPopupContainer={popupContainer}
                  onChange={(v: string) => setFeatureDraft(p => ({ ...p, [f.key]: { engine: p[f.key]?.engine || '', model: v } }))}
                  style={{ flex: 1, minWidth: 0 }}
                  options={engineModelOptions}
                />
              </div>
              <div className="mc-bind-switch">
                <span>功能启用（停用后回退全局模型）</span>
                <Switch
                  size="small"
                  checked={featureEnabled[f.key] !== false}
                  onChange={(v: boolean) => handleToggleFeatureEnabled(f.key, v)}
                />
              </div>
              <Button
                size="small"
                type={bound ? 'primary' : 'default'}
                block
                onClick={() => handleSaveFeature(f.key)}
              >
                {bound ? '更新绑定' : '绑定'}
              </Button>
            </div>
          )
        })}

        {/* 聊天语音：优先于全局 TTS */}
        <div className={`mc-bind-card${chatVoiceCfg.model ? ' is-bound' : ' is-soft'}`}>
          <div className="mc-bind-head">
            <span className="mc-bind-title"><SoundOutlined /> 聊天语音</span>
            <StatusChip tone={chatVoiceCfg.model ? 'ok' : 'neutral'} dot>
              {chatVoiceCfg.model ? '已绑定' : '未绑定'}
            </StatusChip>
          </div>
          <div className="mc-bind-desc">
            优先于全局 TTS；列表随引擎模型自动刷新，后续新增语音模型无需改代码
          </div>
          <div className="mc-bind-meta">
            {chatVoiceCfg.model
              ? `当前：${chatVoiceCfg.engine} / ${chatVoiceCfg.model}`
              : '未绑定：语音对话使用全局 TTS（语音模型页）'}
          </div>
          <div className="mc-bind-row">
            <Select
              size="small"
              placeholder="引擎"
              value={chatVoiceDraft.engine || undefined}
              getPopupContainer={popupContainer}
              onChange={(v: string) => setChatVoiceDraft({ engine: v, model: '' })}
              style={{ flex: 1, minWidth: 0 }}
              options={engines
                .filter(e => e.enabled && ttsModels.some(m => m.engineId === e.id))
                .map(e => ({ value: e.id, label: engineLabel(e) }))}
            />
            <Select
              size="small"
              placeholder="语音模型"
              value={chatVoiceDraft.model || undefined}
              getPopupContainer={popupContainer}
              onChange={(v: string) => setChatVoiceDraft(p => ({ ...p, model: v }))}
              style={{ flex: 1, minWidth: 0 }}
              options={modelOptionsForEngine(chatVoiceDraft.engine, ttsModels, chatVoiceDraft.model)}
            />
          </div>
          {chatVoiceDraft.engine && chatVoiceDraft.model && chatVoiceOptions.length > 0 && (
            <div className="mc-bind-row">
              <span style={{ color: 'var(--mc-muted)', fontSize: 11, whiteSpace: 'nowrap' }}>音色</span>
              <Select
                size="small"
                value={chatVoiceValue}
                placeholder="选择音色"
                getPopupContainer={popupContainer}
                onChange={async (v: string) => {
                  try {
                    await App.VoiceApplySettings?.({ ttsVoice: v })
                    message.success('音色已更新：' + v)
                  } catch (err: unknown) {
                    message.error(err instanceof Error ? err.message : '音色更新失败')
                  }
                  setVoiceCfg(p => ({ ...p, tts: { ...p.tts, voice: v } }))
                }}
                style={{ flex: 1, minWidth: 0 }}
                options={chatVoiceOptions}
              />
            </div>
          )}
          <div className="mc-bind-row">
            <Button
              size="small"
              type={chatVoiceCfg.model ? 'primary' : 'default'}
              block
              loading={chatVoiceSaving}
              onClick={handleSaveChatVoice}
            >
              {chatVoiceCfg.model ? '更新绑定' : '绑定聊天语音'}
            </Button>
            {chatVoiceCfg.model && (
              <Button size="small" danger onClick={handleClearChatVoice} loading={chatVoiceSaving}>
                清除
              </Button>
            )}
          </div>
        </div>

        {/* 绘梦：自身界面选择 */}
        <div className="mc-bind-card is-soft">
          <div className="mc-bind-head">
            <span className="mc-bind-title"><PictureOutlined /> 绘梦</span>
          </div>
          <div className="mc-bind-desc">
            图片模型在绘梦界面内选择（后端 / 模型 / ComfyUI 启停），无需在此重复设置
          </div>
        </div>

        {/* 角色库剧照：独立图片后端/模型 */}
        <div className={`mc-bind-card${portraitCfg.backend ? ' is-bound' : ' is-soft'}`}>
          <div className="mc-bind-head">
            <span className="mc-bind-title"><PictureOutlined /> 角色库剧照</span>
            <StatusChip tone={portraitCfg.backend ? 'accent' : 'neutral'} dot>
              {portraitCfg.backend ? `${portraitCfg.backend} / ${portraitCfg.model || '跟随绘梦'}` : '跟随绘梦'}
            </StatusChip>
          </div>
          <div className="mc-bind-desc">
            角色卡「生成剧照」使用独立后端；留空则跟随绘梦页当前选择
          </div>
          <div className="mc-bind-row">
            <Select
              size="small"
              placeholder="后端"
              value={portraitDraft.backend || undefined}
              getPopupContainer={popupContainer}
              onChange={(v: string) => setPortraitDraft({ backend: v, model: '' })}
              style={{ flex: 1, minWidth: 0 }}
              options={[
                { value: '', label: '跟随绘梦' },
                { value: 'xai', label: 'xAI 云端' },
                { value: 'comfyui', label: 'ComfyUI 本地' },
                { value: 'herdsman', label: 'Herdsman 本地' },
                { value: 'ollama', label: 'Ollama 本地' },
              ]}
            />
            <Select
              size="small"
              placeholder="模型"
              value={portraitDraft.model || undefined}
              getPopupContainer={popupContainer}
              onChange={(v: string) => setPortraitDraft(p => ({ ...p, model: v }))}
              style={{ flex: 1, minWidth: 0 }}
              options={portraitModelOptions}
            />
          </div>
          <Button
            size="small"
            type={portraitCfg.backend ? 'primary' : 'default'}
            block
            loading={portraitSaving}
            onClick={handleSavePortrait}
          >
            {portraitCfg.backend ? '更新剧照绑定' : '绑定剧照后端'}
          </Button>
        </div>
      </div>
    </section>
  )
}
