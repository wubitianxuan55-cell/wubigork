import { useState } from 'react'
import { Button, Input } from 'antd'
import { AudioOutlined, CaretRightOutlined, SoundOutlined } from '@ant-design/icons'
import { EmptyState, ModelCard, SectionHead, StatusChip } from './ui'
import { engineLabel, filterModelsBySearch, isLocalEngine, modelAvailability, sortModelsPinnedFirst } from './utils'
import { useModelCenter } from './context'
import { usePinnedModels } from './modelPrefs'

export function VoiceSection() {
  const {
    voiceCfg, ttsModels, sttModels, handleSetVoiceModel,
    handleStartModel, engines, engineStatuses,
  } = useModelCenter()
  const [voiceSearch, setVoiceSearch] = useState('')
  const [pinned, togglePin] = usePinnedModels()
  const tts = sortModelsPinnedFirst(filterModelsBySearch(ttsModels, voiceSearch), pinned)
  const stt = sortModelsPinnedFirst(filterModelsBySearch(sttModels, voiceSearch), pinned)

  return (
    <section className="mc-section">
      <SectionHead
        icon={<SoundOutlined />}
        title="语音模型"
        desc="识别、对话、合成三段模型；点击卡片切换并自动持久化"
        extra={(
          <Input.Search
            allowClear
            placeholder="搜索语音模型"
            value={voiceSearch}
            onChange={e => setVoiceSearch(e.target.value)}
            style={{ maxWidth: 320 }}
          />
        )}
      />

      <div className="mc-panel">
        <div className="mc-panel-body">
          <div className="mc-panel-title"><SoundOutlined /> 语音管道</div>
          <div className="mc-model-chips" style={{ gap: 8 }}>
            <StatusChip tone={voiceCfg.stt.model ? 'accent' : 'neutral'} dot>
              识别 {voiceCfg.stt.model || '自动'}
            </StatusChip>
            <StatusChip tone={voiceCfg.llm.model ? 'ok' : 'neutral'} dot>
              对话 {voiceCfg.llm.model || '默认'}
            </StatusChip>
            <StatusChip tone={voiceCfg.tts.model ? 'warn' : 'neutral'} dot>
              合成 {voiceCfg.tts.model || '自动'}
            </StatusChip>
          </div>
        </div>
      </div>

      {ttsModels.length === 0 && sttModels.length === 0 ? (
        <EmptyState
          icon={<SoundOutlined />}
          title="未发现语音模型"
          hint="请先在「引擎管理」中刷新模型列表（Herdsman 本地引擎可提供 whisper / qwen3-tts 等）"
        />
      ) : (
        <>
          {ttsModels.length > 0 && (
            <div className="mc-engine-group">
              <div className="mc-group-title"><SoundOutlined /> TTS 语音合成</div>
              <div className="mc-grid">
                {tts.map(m => {
                  const active = voiceCfg.tts.engine === m.engineId && voiceCfg.tts.model === m.modelId
                  const eng = engines.find(e => e.id === m.engineId)
                  const avail = modelAvailability(m, eng?.enabled ?? true, engineStatuses[m.engineId]?.connected)
                  const blocked = avail === 'disconnected' || avail === 'disabled'
                  const canStartLocal = isLocalEngine(m.engineId) && (m.status === 'stopped' || avail === 'disconnected')
                  return (
                    <ModelCard
                      key={`${m.engineId}:${m.modelId}`}
                      name={m.modelName}
                      engineId={m.engineId}
                      engineName={engineLabel(m)}
                      kindChip={<StatusChip tone="warn">TTS</StatusChip>}
                      chips={[
                        active
                          ? <StatusChip key="active" tone="warn">语音合成中</StatusChip>
                          : null,
                        pinned.includes(m.modelId)
                          ? <StatusChip key="pin" tone="warn">置顶</StatusChip>
                          : null,
                        avail === 'disconnected'
                          ? <StatusChip key="off" tone="danger">未连接</StatusChip>
                          : null,
                      ].filter(Boolean)}
                      active={active}
                      dimmed={blocked}
                      pinned={pinned.includes(m.modelId)}
                      onTogglePin={() => togglePin(m.modelId)}
                      status={{
                        tone: active || m.status === 'running'
                          ? 'ok'
                          : avail === 'disconnected'
                            ? 'danger'
                            : m.status === 'stopped'
                              ? 'warn'
                              : 'neutral',
                        text: avail === 'disconnected'
                          ? '未连接'
                          : m.status === 'running'
                            ? '运行中'
                            : m.status === 'stopped'
                              ? '未启动'
                              : '就绪',
                      }}
                      action={(
                        <span style={{ display: 'inline-flex', gap: 6, alignItems: 'center' }}>
                          {canStartLocal && (
                            <Button
                              size="small"
                              icon={<CaretRightOutlined />}
                              onClick={() => handleStartModel(m)}
                            >
                              启动服务
                            </Button>
                          )}
                          <Button
                            size="small"
                            type={active ? 'primary' : 'default'}
                            icon={<SoundOutlined />}
                            onClick={() => handleSetVoiceModel('tts', m.engineId, m.modelId)}
                            disabled={blocked && !active}
                          >
                            {active ? '已设为语音合成' : '设为语音合成'}
                          </Button>
                        </span>
                      )}
                    />
                  )
                })}
              </div>
            </div>
          )}

          {sttModels.length > 0 && (
            <div className="mc-engine-group">
              <div className="mc-group-title"><AudioOutlined /> STT 语音识别</div>
              <div className="mc-grid">
                {stt.map(m => {
                  const active = voiceCfg.stt.engine === m.engineId && voiceCfg.stt.model === m.modelId
                  const eng = engines.find(e => e.id === m.engineId)
                  const avail = modelAvailability(m, eng?.enabled ?? true, engineStatuses[m.engineId]?.connected)
                  const blocked = avail === 'disconnected' || avail === 'disabled'
                  return (
                    <ModelCard
                      key={`${m.engineId}:${m.modelId}`}
                      name={m.modelName}
                      engineId={m.engineId}
                      engineName={engineLabel(m)}
                      kindChip={<StatusChip tone="accent">STT</StatusChip>}
                      chips={[
                        active
                          ? <StatusChip key="active" tone="accent">语音识别中</StatusChip>
                          : null,
                        pinned.includes(m.modelId)
                          ? <StatusChip key="pin" tone="warn">置顶</StatusChip>
                          : null,
                        avail === 'disconnected'
                          ? <StatusChip key="off" tone="danger">未连接</StatusChip>
                          : null,
                      ].filter(Boolean)}
                      active={active}
                      dimmed={blocked}
                      pinned={pinned.includes(m.modelId)}
                      onTogglePin={() => togglePin(m.modelId)}
                      status={{
                        tone: active || m.status === 'running'
                          ? 'ok'
                          : avail === 'disconnected'
                            ? 'danger'
                            : m.status === 'stopped'
                              ? 'warn'
                              : 'neutral',
                        text: avail === 'disconnected'
                          ? '未连接'
                          : m.status === 'running'
                            ? '运行中'
                            : m.status === 'stopped'
                              ? '未启动'
                              : '就绪',
                      }}
                      action={(
                        <Button
                          size="small"
                          type={active ? 'primary' : 'default'}
                          icon={<AudioOutlined />}
                          onClick={() => handleSetVoiceModel('asr', m.engineId, m.modelId)}
                          disabled={blocked && !active}
                        >
                          {active ? '已设为语音识别' : '设为语音识别'}
                        </Button>
                      )}
                    />
                  )
                })}
              </div>
            </div>
          )}
        </>
      )}
    </section>
  )
}
