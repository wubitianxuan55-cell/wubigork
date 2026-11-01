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
import './voiceV2Lab.css'

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
          <span className="v2l-title">Voice V2 Lab</span>
          <span className={`v2l-phase v2l-phase-${v2.phase}`} data-testid="v2l-phase">{PHASE_LABEL[v2.phase] ?? v2.phase}</span>
        </header>

        <div className="v2l-row">
          <input className="v2l-url" value={url} onChange={(e) => setUrl(e.target.value)} spellCheck={false} />
          {v2.connected ? (
            <button className="v2l-btn" onClick={v2.disconnect}>断开</button>
          ) : (
            <button className="v2l-btn" onClick={connectLive}>连接</button>
          )}
          <button className="v2l-btn" onClick={() => (v2.micOn ? v2.stopMic() : void v2.startMic())}>{v2.micOn ? '关麦' : '开麦'}</button>
          <button className="v2l-btn" onClick={v2.cancel}>取消</button>
          <button className="v2l-btn v2l-btn-ghost" onClick={() => void v2.replayEvents(FIXTURE_TURN)}>mock 回放</button>
          {!v2.services.ready && <button className="v2l-btn v2l-btn-ghost" onClick={() => void v2.ensureServices()}>拉起依赖链</button>}
          {v2.note && <span className="v2l-note" data-testid="v2l-note">{v2.note}</span>}
        </div>

        <div className="v2l-cols">
          <section className="v2l-sec">
            <h4>用户转写</h4>
            <div className="v2l-scroll" data-testid="v2l-transcripts">
              {v2.transcripts.map((t, i) => (
                <p key={i} className="v2l-user">{t}</p>
              ))}
              {v2.partialTranscript && <p className="v2l-user v2l-partial">{v2.partialTranscript}…</p>}
            </div>
          </section>
          <section className="v2l-sec">
            <h4>助手</h4>
            <div className="v2l-scroll">
              <p className="v2l-assistant" data-testid="v2l-assistant">{v2.assistantText || '—'}</p>
            </div>
          </section>
        </div>

        <footer className="v2l-foot">
          <span className="v2l-pb" data-testid="v2l-pb">{v2.pbText || '播放队列：—'}</span>
          <div className="v2l-tail" data-testid="v2l-tail">
            {v2.tail.map((l, i) => (
              <span key={i}>{l}</span>
            ))}
          </div>
        </footer>
      </div>
    </div>
  )
}

export default VoiceV2Lab
