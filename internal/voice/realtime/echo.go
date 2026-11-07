// echo.go — ProtocolEcho：刀2 协议回声响应器（真管线刀5 接管前的占位 Handler）。
//
// response.create → 完整 response 事件链（created → item → content part →
// transcript → 100ms 静音音频 → done completed），用于验证前端整链状态机
// （ResponseLedger 配对、PlaybackQueue、orderingViolations）与 ws_probe 时序断言。
package realtime

// ProtocolEcho 占位管线。
type ProtocolEcho struct{}

// OnAudio 接收但不上报（真 VAD 刀3 接管）。
func (ProtocolEcho) OnAudio(*Session, []byte) {}

// OnSessionConfig 记录即走（配置在 Session 内已合并）。
func (ProtocolEcho) OnSessionConfig(*Session, SessionConfig) {}

// OnSessionEnd no-op。
func (ProtocolEcho) OnSessionEnd(*Session) {}

// OnResponseCancel 回声响应器同步完成，无可取消在途（真管线刀5 实装）。
func (ProtocolEcho) OnResponseCancel(*Session) {}

// OnPlaybackDrained 回声响应器无播放面，no-op。
func (ProtocolEcho) OnPlaybackDrained(*Session) {}

// OnResponseCreate 发一条完整回声响应链。
func (ProtocolEcho) OnResponseCreate(s *Session) {
	respID := "resp_" + randHex(6)
	itemID := "item_" + randHex(8)
	const transcript = "（语音引擎准备中——协议回声）"
	// 100ms 静音 PCM16@16k
	audio := make([]byte, 1600*2)

	s.Send(evResponseCreated(respID))
	s.Send(evOutputItemAdded(respID, itemID, 0))
	s.Send(evContentPartAdded(respID, itemID))
	s.Send(evTranscriptDelta(respID, itemID, transcript))
	s.Send(evTranscriptDone(respID, itemID, transcript))
	s.Send(evAudioDelta(respID, itemID, audio))
	s.Send(evAudioDone(respID, itemID))
	s.Send(evContentPartDone(respID, itemID))
	s.Send(evOutputItemDone(respID, itemID, 0))
	s.Send(evResponseDone(ResponseDonePayload{
		ID:     respID,
		Status: "completed",
		Usage:  map[string]any{"input_tokens": 0, "output_tokens": 0},
	}))
}
