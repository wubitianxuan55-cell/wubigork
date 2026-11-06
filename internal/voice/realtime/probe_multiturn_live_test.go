//go:build windows && amd64 && cgo

// probe_multiturn_live_test.go — 真语料多轮实弹探针（2026-11-07 蒸馏批0 复验钉）。
// 真 silero VAD + 真 paraformer STT 直驱管线（LLM 腿假——不在靶上），同语料喂
// 两轮：跨轮前缀清零守卫（批0 根修）若复发，轮2 转写≈轮1 文本两倍（2026-11-06
// 实弹 2→6→…→54 滚雪球的进程内复现）；修复后两轮同文本。模型缺位自动 Skip。
package realtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPipelineMultiTurnNoSnowball(t *testing.T) {
	pipe := newNativePipeline(t)
	// LLM 腿假（转写/VAD/前缀在靶）+ 打开自动生成（转写 terminal 随响应提交发布）
	pipe.legs = &fakeLegs{reply: "好"}
	pipe.cfg.CreateResponse = true
	sess := &Session{ID: "probe", out: make(chan map[string]any, 65536), done: make(chan struct{})}

	// 常驻抽干线缆（生产=前端 ws 读程恒抽干；不抽干则 Session.Send 背压级联
	// 会把 ttsCh/sttCh 顶满、VAD 停摆——那是测试伪象不是管线病）
	completed := make(chan string, 16)
	go func() {
		for {
			select {
			case ev := <-sess.out:
				if ev["type"] == "conversation.item.input_audio_transcription.completed" {
					if s, ok := ev["transcript"].(string); ok {
						completed <- s
					}
				}
			case <-sess.done:
				return
			}
		}
	}()

	raw, err := os.ReadFile(filepath.Join("..", "native", "testdata", "eot_complete_16k.wav"))
	if err != nil {
		t.Fatal(err)
	}
	pcm := raw[44:]
	const frameBytes = 320 * 2 // 320 样本 × 2B = 20ms/帧
	silence := make([]byte, frameBytes)
	// 1x 实时节奏喂入（生产=麦克风 20ms/帧恒速；微秒级灌入属非自然竞速区，
	// STT 门/重开宽限的时序语义按真实节奏设计）
	feedAll := func(b []byte) {
		for off := 0; off+frameBytes <= len(b); off += frameBytes {
			pipe.OnAudio(sess, b[off:off+frameBytes])
			time.Sleep(18 * time.Millisecond)
		}
	}

	// 轮1：语料 + 1.2s 静音（> complete 宽限 800ms）→ soft-end → STT → 响应收口
	feedAll(pcm)
	for i := 0; i < 60; i++ {
		pipe.OnAudio(sess, silence)
		time.Sleep(18 * time.Millisecond)
	}
	text1 := waitCompleted(t, completed, 1, 40*time.Second)
	time.Sleep(500 * time.Millisecond) // 响应终态/墓碑落稳再开第二轮

	// 轮2：同语料再来一轮
	feedAll(pcm)
	for i := 0; i < 60; i++ {
		pipe.OnAudio(sess, silence)
		time.Sleep(18 * time.Millisecond)
	}
	t.Logf("轮2 喂毕仪表: audioMs=%d triggered=%v audioCh=%d textCh=%d sttCh=%d promptCh=%d ttsCh=%d audioOut=%d sessOut=%d",
		pipe.vad.AudioMs(), pipe.vad.iter.triggered, len(pipe.audioCh), len(pipe.textCh),
		len(pipe.sttCh), len(pipe.promptCh), len(pipe.ttsCh), len(pipe.audioOut), len(sess.out))
	text2 := waitCompleted(t, completed, 2, 40*time.Second)

	if strings.TrimSpace(text1) == "" {
		t.Fatal("轮1 转写为空（STT 链路未通）")
	}
	// 雪球签名断言：跨轮前缀清零守卫若复发，轮2 转写≈全会话语音重转=含
	// 轮1 文本拼接/长度翻倍（2026-11-06 实弹 2→6→…→54）。ASR 对同语料存在
	// 正常变体（实测「你好」→「啊你好」），不做等值断言。
	r1, r2 := float64(len([]rune(text1))), float64(len([]rune(text2)))
	if strings.Contains(text2, text1+text1) || r2 > r1*2+6 {
		t.Fatalf("滚雪球复发（轮2 转写含轮1 拼接或长度翻倍）:\ntext1=%q\ntext2=%q", text1, text2)
	}
	t.Logf("两轮转写无累积: text1=%q(%d字) text2=%q(%d字)", text1, int(r1), text2, int(r2))
}

// waitCompleted 等第 n 条转写完成帧（抽干线程已把 completed 送入管道）。
func waitCompleted(t *testing.T, completed <-chan string, n int, d time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(d)
	count := 0
	var last string
	for time.Now().Before(deadline) {
		select {
		case s := <-completed:
			count++
			last = s
			if count >= n {
				return last
			}
		case <-time.After(50 * time.Millisecond):
		}
	}
	t.Fatalf("转写完成帧不足: got %d want %d", count, n)
	return ""
}
