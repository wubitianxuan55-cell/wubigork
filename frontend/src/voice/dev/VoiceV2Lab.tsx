/**
 * voice/dev/VoiceV2Lab.tsx — Voice V2 实验页（M2 最小页面，?voicev2=1 唤出）。
 *
 * M4 起会话运行时收敛进 useVoiceChatV2（与语音台 V2 同一实现），本页只留
 * 实验面：URL 直填、mock 回放（哑线缆灌实录事件）、协议事件尾屏。
 * 不进正式导航；产品接入走 voice.backend=v2（ChatPage → VoiceConsoleV2）。
 */
import React, { useState } from 'react'
import { useVoiceChatV2 } from '../runtime/useVoiceChatV2'
import type { VoiceTransport } from '../client/VoiceClientV2'
import { FIXTURE_TURN } from '../session/fixtures'
import { VoiceV2Orb } from '../components/VoiceV2Orb'
import '../voiceV2.css'

const DEFAULT_URL = 'ws://127.0.0.1:8791/v1/realtime'

const PHASE_LABEL: Record<string, string> = {
  idle: '空闲',
  listening: '聆听',
  thinking: '思考',
  speaking: '播报',
  interrupted: '被打断',
  error: '错误',
}

/** mock 回放哑线缆：吞发送、关不动作（事件经 replayEvents 直灌客户端） */
class NoopTransport implements VoiceTransport {
  send(): void {}
  close(): void {}
  onopen: (() => void) | null = null
  onclose: (() => void) | null = null
  onerror: ((err: unknown) => void) | null = null
  onmessage: ((data: string) => void) | null = null
}

const VoiceV2Lab: React.FC = () => {
  const [url, setUrl] = useState(DEFAULT_URL)
  const v2 = useVoiceChatV2({
    url,
    makeReplayTransport: () => new NoopTransport(),
  })

  const connectLive = () => {
    void v2.connect() // 钩子实时读 opts.url，改 URL 免重渲染即连
  }

  return (
    <div className="v2l-overlay" data-testid="voice-v2-lab">
      <div className="v2l-panel">
        <header className="v2l-head">
          <div>
            <span className="vc2-eyebrow">PROTOCOL LAB</span>
            <h2 className="v2l-title">Voice V2 Lab</h2>
            <p>实时协议、播放队列与 mock 回放</p>
          </div>
          <span className={`v2l-phase v2l-phase-${v2.phase}`} data-testid="v2l-phase">
            {PHASE_LABEL[v2.phase] ?? v2.phase}
          </span>
        </header>

        <div className="v2l-toolbar">
          <label className="v2l-url-field" htmlFor="v2l-endpoint">
            <span>Realtime endpoint</span>
            <input
              id="v2l-endpoint"
              className="v2l-url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              spellCheck={false}
            />
          </label>
          <div className="v2l-actions">
            {v2.connected ? (
              <button className="v2l-btn" onClick={v2.disconnect}>断开</button>
            ) : (
              <button className="v2l-btn" onClick={connectLive} disabled={!v2.services.ready}>连接</button>
            )}
            <button className="v2l-btn" onClick={() => (v2.micOn ? v2.stopMic() : void v2.startMic())}>
              {v2.micOn ? '关麦' : '开麦'}
            </button>
            <button className="v2l-btn" onClick={v2.cancel}>取消</button>
            <button className="v2l-btn v2l-btn-ghost" onClick={() => void v2.replayEvents(FIXTURE_TURN)}>mock 回放</button>
            {!v2.services.ready && (
              <button
                className="v2l-btn v2l-btn-ghost"
                onClick={() => void v2.ensureServices().then((ready) => {
                  if (ready) void v2.connect()
                })}
              >
                启动依赖
              </button>
            )}
          </div>
          {v2.note && <p className="v2l-note" role="status" data-testid="v2l-note">{v2.note}</p>}
        </div>

        <div className="v2l-stage-grid">
          <section className="v2l-field">
            <header>
              <span className="vc2-eyebrow">VOICE FIELD</span>
              <span className={`v2l-connection${v2.connected ? ' is-live' : ''}`}>
                {v2.connected ? '已连接' : '未连接'}
              </span>
            </header>
            <div className="v2l-orb-stage">
              <VoiceV2Orb phase={v2.phase} active={v2.micOn || v2.phase === 'speaking'} connected={v2.connected} />
              <p>{v2.micOn ? '麦克风正在采集 16k PCM' : '打开麦克风后可观察真实 VAD 与转写相位'}</p>
            </div>
            <div className="v2l-signal-row" aria-hidden="true">
              {Array.from({ length: 18 }, (_, index) => (
                <i key={index} style={{ '--vc2-wave': 0.2 + ((index * 5) % 11) / 15 } as React.CSSProperties} />
              ))}
            </div>
          </section>

          <section className="v2l-stream-panel">
            <header>
              <span className="vc2-eyebrow">LIVE STREAM</span>
              <span>{v2.transcripts.length} 轮</span>
            </header>
            <div
              className="v2l-scroll"
              data-testid="v2l-transcripts"
              role="log"
              aria-live="polite"
              aria-busy={v2.phase === 'thinking'}
            >
              {v2.transcripts.map((transcript, index) => (
                <article key={index} className="v2l-user">
                  <span>你</span>
                  <p>{transcript}</p>
                </article>
              ))}
              {v2.partialTranscript && (
                <article className="v2l-user v2l-partial">
                  <span>正在听</span>
                  <p>{v2.partialTranscript}<i className="vc2-caret" aria-hidden="true" /></p>
                </article>
              )}
              {v2.transcripts.length === 0 && !v2.partialTranscript && (
                <p className="v2l-partial">{v2.micOn ? '说话后，转写会出现在这里。' : '尚未收到用户转写。'}</p>
              )}
              <article className="v2l-assistant">
                <span>助手</span>
                <p data-testid="v2l-assistant">{v2.assistantText || '等待模型回应…'}</p>
              </article>
            </div>
          </section>
        </div>

        <details className="v2l-diagnostics">
          <summary>协议事件与播放队列</summary>
          <div className="v2l-diagnostic-body">
            <span className="v2l-pb" data-testid="v2l-pb">{v2.pbText || '播放队列：—'}</span>
            <div className="v2l-tail" data-testid="v2l-tail">
              {v2.tail.length > 0
                ? v2.tail.map((line, i) => <span key={i}>{line}</span>)
                : <span>暂无协议事件</span>}
            </div>
          </div>
        </details>
      </div>
    </div>
  )
}

export default VoiceV2Lab
