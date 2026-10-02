// Package voice — P0#12（审计 AP6-01）voice.Manager 回调字段并发回归锁。
//
// 背景：NewManager 注入的 emitter 与 SetWhisperChatFn/SetTTSSynthesizeFn 注入的
// 两个 seam 回调，修复前都是**裸字段写读**。写侧是启动末尾刷新 / 运行时重接线
// （internal/app/voice_handler.go:105/115/184 的 initVoice 链，VoiceRestart、
// VoiceGetSettings、VoiceStart 都会走到 initVoice），读侧是语音回合
// （handleReply/speak/WhisperReady）——两侧分处不同 goroutine 即构成 Go 内存
// 模型意义上的 data race（本机 CGO_ENABLED=0，无 gcc，跑不了 -race，故用下面
// 的交错用例做本地红/绿证据；真正的竞态判定仍以 CI 的 Actions race job 为准）。
//
// 本文件三条用例：
//  1. TestManager_ChatFnSwapMidTurnIsNotObservedMidTurn —— 确定性交错：在回合内
//     首个事件口（EmitVoiceTranscript）同步重接线。旧代码的「空值检查（原
//     :601）→ 调用点（原 :613）」是两次裸读，第二次会读到替换后的值：换成 nil
//     即本轮**丢回复**（审计描述的「语音回合丢回复且日志无线索」），换成另一个
//     回调即本轮换成别人的接线。修复后回合内只认开轮那一次原子快照。
//  2. TestManager_TTSSynthFnSwapMidTurnIsNotObservedMidTurn —— 同族第二条：
//     旧代码 TTS 回调在 :652 判定与 :755 调用也是两次裸读。
//  3. TestManager_CallbackSwapConcurrentWithVoiceTurns —— 并发 smoke：写侧
//     goroutine 持续换接线 + 读侧串行跑完整语音回合（含 TTS 播放环），断言
//     每回合恰好一条回复、每回合音频由该回合文本经某个回调变体产生、零错误
//     事件。**灵敏度声明**：本用例对竞态本身不敏感（旧代码同样能通过，见
//     TestManager_CallbackSwapConcurrentWithVoiceTurns 末尾说明），它承担的是
//     回归锁 + 在 -race 门下的判定载体；红/绿证据由前两条确定性用例提供。
package voice

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// swapEmitter 在既有 mockEmitter 之外允许在指定事件口挂**同步钩子**：钩子在
// 回合 goroutine 内被调用，因此测试能构造确定性交错，而不是靠 sleep 撞运气。
// EmitVoiceTTSAudio 的钩子在记录之后、释放 e.mu 之后调用（钩子内可安全回打
// Manager 方法，如 PlaybackDone）。
type swapEmitter struct {
	onTranscript func()
	onThinking   func(active bool)
	onTTSAudio   func()

	mu       sync.Mutex
	replies  []string
	audios   [][]byte
	errors   []error
	fallback []string
}

func (e *swapEmitter) EmitVoiceState(VoiceState) {}

func (e *swapEmitter) EmitVoiceTranscript(_ string, isFinal bool) {
	if isFinal && e.onTranscript != nil {
		e.onTranscript()
	}
}

func (e *swapEmitter) EmitVoiceReply(text string) {
	e.mu.Lock()
	e.replies = append(e.replies, text)
	e.mu.Unlock()
}

func (e *swapEmitter) EmitVoiceTTSAudio(audio []byte, _ string) {
	e.mu.Lock()
	e.audios = append(e.audios, append([]byte(nil), audio...))
	e.mu.Unlock()
	if e.onTTSAudio != nil {
		e.onTTSAudio()
	}
}

func (e *swapEmitter) EmitVoiceTTSSpeakText(text string) {
	e.mu.Lock()
	e.fallback = append(e.fallback, text)
	e.mu.Unlock()
}

func (e *swapEmitter) EmitVoiceTTSCancel()       {}
func (e *swapEmitter) EmitVoiceListening(_ bool) {}
func (e *swapEmitter) EmitVoiceThinking(a bool) {
	if e.onThinking != nil {
		e.onThinking(a)
	}
}
func (e *swapEmitter) EmitVoiceError(err error) {
	e.mu.Lock()
	e.errors = append(e.errors, err)
	e.mu.Unlock()
}

// snapshot 取当前记录快照（并发用例的断言段用，避免持锁断言）。
func (e *swapEmitter) snapshot() (replies, audios, fallback []string, errs []error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	replies = append([]string(nil), e.replies...)
	for _, a := range e.audios {
		audios = append(audios, string(a))
	}
	fallback = append([]string(nil), e.fallback...)
	errs = append([]error(nil), e.errors...)
	return
}

// chatVariant / ttsVariant 生成带标签的回调变体：变体身份写进返回值/音频字节，
// 断言即可判定「本轮到底用了哪一份接线」。
func chatVariant(tag string) WhisperChatFn {
	return func(userMsg, _ string) (string, string, [4]float64, error) {
		return tag + "|" + userMsg, "CALM_RATIONAL", [4]float64{}, nil
	}
}

func ttsVariant(tag string) TTSSynthesizeFn {
	return func(text, _ string) ([]byte, string, error) {
		return []byte(tag + ":" + text), "audio/wav", nil
	}
}

// TestManager_ChatFnSwapMidTurnIsNotObservedMidTurn 确定性交错：回合开始后、
// whisper 回调空值检查之前重接线（正是 app 层在语音回合进行中 initVoice 重接
// 线的窗口）。修复后本轮只认开轮快照，绝不在回合中途换回调/丢回复。
func TestManager_ChatFnSwapMidTurnIsNotObservedMidTurn(t *testing.T) {
	cases := []struct {
		name   string
		swapTo WhisperChatFn // 交错点注入的替换值（nil = 撤回接线）
	}{
		{"换成另一个回调", chatVariant("B")},
		{"撤回接线(nil)", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			em := &swapEmitter{}
			m := NewManager(em, DefaultVoiceConfig())
			m.SetWhisperChatFn(chatVariant("A")) // 开轮时的接线

			// 交错点：handleReply 在 whisper 回调空值检查之前同步发
			// EmitVoiceTranscript（原 :596-597 → :601/:613）。
			em.onTranscript = func() { m.SetWhisperChatFn(tc.swapTo) }

			m.runReply("喂") // 同步跑完一轮（turnMu + panic 防线同生产路径）

			replies, _, fallback, errs := em.snapshot()
			if len(errs) != 0 {
				t.Fatalf("回合内报错 %d 条（丢回复的另一种表现）：%v", len(errs), errs)
			}
			if len(replies) != 1 {
				t.Fatalf("本轮应恰好一条回复（回合开始时的接线），got %d 条 %q；fallback=%q",
					len(replies), replies, fallback)
			}
			if replies[0] != "A|喂" {
				t.Errorf("本轮回复 = %q，want %q（回合中途换接线不得影响已开始的回合）",
					replies[0], "A|喂")
			}
		})
	}
}

// TestManager_TTSSynthFnSwapMidTurnIsNotObservedMidTurn 同族第二条：TTS 合成
// 回调在回合内被重接线。旧代码 :652（是否走 TTS 的判定）与 :755（真正合成）
// 是两次裸读，第二次读到替换后的回调；修复后整轮用同一个快照。
func TestManager_TTSSynthFnSwapMidTurnIsNotObservedMidTurn(t *testing.T) {
	em := &swapEmitter{}
	m := NewManager(em, DefaultVoiceConfig()) // TTSEnabled=true
	m.SetWhisperChatFn(chatVariant("A"))
	m.SetTTSSynthesizeFn(ttsVariant("A")) // 开轮时的接线

	// 交错点：handleReply 在 TTS 回调读点之前同步发 EmitVoiceThinking(true)
	// （原 :590-591 → :652/:755）。
	em.onThinking = func(active bool) {
		if active {
			m.SetTTSSynthesizeFn(ttsVariant("B"))
		}
	}
	// 播放环解锁：等价前端「这句播完了」回打 PlaybackDone（speak 的 ack 通道）。
	em.onTTSAudio = func() { m.PlaybackDone() }

	m.runReply("喂")

	replies, audios, fallback, errs := em.snapshot()
	if len(errs) != 0 {
		t.Fatalf("回合内报错 %d 条：%v", len(errs), errs)
	}
	if len(replies) != 1 || replies[0] != "A|喂" {
		t.Fatalf("回复异常：%q（fallback=%q）", replies, fallback)
	}
	if len(audios) != 1 {
		t.Fatalf("本轮应恰好一段 TTS 音频，got %d 段 %q（fallback=%q）", len(audios), audios, fallback)
	}
	if audios[0] != "A:A|喂" {
		t.Errorf("本轮音频 = %q，want %q（回合中途换接线不得改变已开始回合的合成回调）",
			audios[0], "A:A|喂")
	}
}

// TestManager_CallbackSwapConcurrentWithVoiceTurns 并发 smoke：写侧（模拟启动
// 末尾刷新 / 运行时重接线）持续换两个回调 + 打读侧健康检查口，读侧串行跑 N 轮
// 完整语音回合（含 TTS 合成与播放环）。
//
// 断言的可观测异常：丢回复（条数不足）、串台（回复与本回合输入不对应）、
// 音频与回复文本不匹配（半截值/错回调）、任何错误事件（panic 防线被触发）。
//
// **灵敏度声明（如实登记）**：写侧只在两个**非 nil** 变体间切换，因此本用例在
// 修复前的旧代码下**同样能通过**——旧字段的裸读在无 -race 时不会必然暴露。
// 本用例的定位是「并发回归锁 + -race 门的判定载体」，不是本地红例；本地红例
// 由上面两条确定性交错用例提供。若后续有人把写侧改成「非 nil ↔ nil」翻转，
// 则「丢回复」在两侧都可能合法（开轮快照确实可能取到 nil），届时断言需要按
// 语义重写，不得直接放宽。
func TestManager_CallbackSwapConcurrentWithVoiceTurns(t *testing.T) {
	em := &swapEmitter{}
	m := NewManager(em, DefaultVoiceConfig())
	m.SetWhisperChatFn(chatVariant("A"))
	m.SetTTSSynthesizeFn(ttsVariant("A"))
	em.onTTSAudio = func() { m.PlaybackDone() }

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			if i%2 == 0 {
				m.SetWhisperChatFn(chatVariant("B"))
				m.SetTTSSynthesizeFn(ttsVariant("B"))
			} else {
				m.SetWhisperChatFn(chatVariant("A"))
				m.SetTTSSynthesizeFn(ttsVariant("A"))
			}
			// 同族读点一并并发踩：健康检查/就绪门/配置读写。
			_ = m.WhisperReady()
			_ = m.ASRReady()
			_ = m.HealthCheck()
			_ = m.GetConfig()
			m.ApplyConfig(DefaultVoiceConfig())
		}
	}()

	const turns = 40
	for i := 0; i < turns; i++ {
		m.runReply(fmt.Sprintf("t%d", i))
	}
	close(stop)
	wg.Wait()

	replies, audios, fallback, errs := em.snapshot()
	if len(errs) != 0 {
		t.Fatalf("并发换接线期间出现 %d 条错误事件（回合被中断）：%v", len(errs), errs)
	}
	if len(replies) != turns {
		t.Fatalf("应 %d 条回复，got %d 条（丢/重）：%q；fallback=%q", turns, len(replies), replies, fallback)
	}
	if len(audios) != turns {
		t.Fatalf("应 %d 段音频，got %d 段（丢/重）：%q", turns, len(audios), audios)
	}

	seen := make(map[string]bool, turns)
	for _, r := range replies {
		if !strings.HasPrefix(r, "A|") && !strings.HasPrefix(r, "B|") {
			t.Errorf("回复 %q 不来自任何回调变体（半截值/串台）", r)
			continue
		}
		turn := strings.TrimPrefix(strings.TrimPrefix(r, "A|"), "B|")
		if seen[turn] {
			t.Errorf("回合 %q 出现重复回复：%q", turn, replies)
		}
		seen[turn] = true

		matched := false
		for _, a := range audios {
			if a == "A:"+r || a == "B:"+r {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("回合成品 %q 找不到「同一句文本 + 某回调变体」的音频：%q", r, audios)
		}
	}
	for i := 0; i < turns; i++ {
		if !seen[fmt.Sprintf("t%d", i)] {
			t.Errorf("回合 t%d 没有产生回复（并发换接线吞了本轮）", i)
		}
	}
}
