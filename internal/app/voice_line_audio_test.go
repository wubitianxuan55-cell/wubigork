package app

import (
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/modelengine"
)

// voice_line_audio_test.go — 台词语音文件缓存链（TTSLineAudioFor / 批量补齐 /
// 保存钩子变更判定）。合成走 ttsVoicePreviewAudioFn 桩（绝不真合成），breeze
// 探活桩（绝不 ensure 拉起真服务）。

func stubVoiceLineAudioSeams(t *testing.T, synth func(a *mediaState, text, voiceID, desc string) ([]byte, string, error), alive bool) *int {
	t.Helper()
	t.Cleanup(func() {
		ttsVoicePreviewAudioFn = (*mediaState).ttsVoicePreviewAudio
		breezeStreamProbe = func(a *mediaState) bool { return a.ttsReady() }
		breezeStreamEnsure = func(a *mediaState) { a.ensureLocalTTSService("breeze") }
		breezeStreamPollInterval = 200 * time.Millisecond
	})
	calls := 0
	ttsVoicePreviewAudioFn = func(a *mediaState, text, voiceID, desc string) ([]byte, string, error) {
		calls++
		return synth(a, text, voiceID, desc)
	}
	breezeStreamProbe = func(*mediaState) bool { return alive }
	breezeStreamEnsure = func(*mediaState) {}
	// 引擎不活用例靠轮询兜底确认（60×200ms=12s），测试里缩到 1ms 防拖慢
	breezeStreamPollInterval = 1 * time.Millisecond
	return &calls
}

func newVoiceLineTestState(t *testing.T) (*mediaState, *characterlib.Store) {
	t.Helper()
	s := characterlib.NewStore(t.TempDir())
	if s == nil {
		t.Fatalf("Store 初始化失败")
	}
	t.Cleanup(func() { _ = s.Close() })
	a := &mediaState{core: &core{engineMgr: modelengine.NewManager("", "")}}
	a.charLib = s
	return a, s
}

func TestResolveVoiceLineText(t *testing.T) {
	if got := resolveVoiceLineText("苏念", "剑来。", []string{"别挡路。"}); got != "剑来。" {
		t.Errorf("专属台词优先: %q", got)
	}
	if got := resolveVoiceLineText("苏念", " ", []string{" ", "别挡路。"}); got != "别挡路。" {
		t.Errorf("样本回退: %q", got)
	}
	if got := resolveVoiceLineText("苏念", "", nil); got != "你好，我是「苏念」，很高兴认识你。" {
		t.Errorf("问候兜底: %q", got)
	}
	if got := resolveVoiceLineText("", "", nil); got != "你好，我是「…」，很高兴认识你。" {
		t.Errorf("无名问候兜底: %q", got)
	}
	if genderDefaultVoiceDescText("male") == "" || genderDefaultVoiceDescText("female") == "" || genderDefaultVoiceDescText("neutral") != "" {
		t.Errorf("性别缺省口径不符")
	}
}

func TestVoiceFileKey_ContentAddressed(t *testing.T) {
	k1 := characterVoiceKey("剑来。", "v1", "d1")
	if k1 != characterVoiceKey("剑来。", "v1", "d1") {
		t.Fatal("同内容键必须稳定")
	}
	if k1 == characterVoiceKey("剑来。", "v1", "d2") || k1 == characterVoiceKey("改了台词", "v1", "d1") {
		t.Fatal("内容变化键必须变化")
	}
	if len(k1) != 16 || strings.ContainsAny(k1, "/\\.") {
		t.Fatalf("键应为 16 位安全 hex: %q", k1)
	}
}

// TestTTSLineAudioFor_CacheRoundTrip 未命中合成落盘（cached=false）→ 再取命中
// 回读文件（cached=true 且不再触发合成）。
func TestTTSLineAudioFor_CacheRoundTrip(t *testing.T) {
	a, _ := newVoiceLineTestState(t)
	calls := stubVoiceLineAudioSeams(t, func(*mediaState, string, string, string) ([]byte, string, error) {
		return []byte("FAKEWAV"), "audio/wav", nil
	}, true)

	ch := `{"id":"lib_1","name":"苏念","ttsLine":"剑来。","gender":"female"}`
	res, err := a.TTSLineAudioFor(ch)
	if err != nil {
		t.Fatalf("首次生成失败: %v", err)
	}
	if res["cached"] != false || res["audio"] == "" {
		t.Fatalf("首次应未命中回传音频: %v", res)
	}
	if *calls != 1 {
		t.Fatalf("首次应合成一次, calls=%d", *calls)
	}

	res2, err := a.TTSLineAudioFor(ch)
	if err != nil {
		t.Fatalf("二次取用失败: %v", err)
	}
	if res2["cached"] != true {
		t.Fatalf("二次应命中文件缓存: %v", res2)
	}
	if res2["audio"] != res["audio"] || res2["mimeType"] != "audio/wav" {
		t.Fatalf("命中应回读同一文件: %v vs %v", res2, res)
	}
	if *calls != 1 {
		t.Fatalf("命中不应再合成, calls=%d", *calls)
	}
}

func TestTTSLineAudioFor_Guards(t *testing.T) {
	a, _ := newVoiceLineTestState(t)
	calls := stubVoiceLineAudioSeams(t, func(*mediaState, string, string, string) ([]byte, string, error) {
		return []byte("X"), "audio/wav", nil
	}, true)

	if _, err := a.TTSLineAudioFor(`{"name":"苏念","ttsLine":"剑来。"}`); err == nil || !strings.Contains(err.Error(), "请先保存角色") {
		t.Fatalf("无 id 应拒绝: %v", err)
	}
	if _, err := a.TTSLineAudioFor(`not-json`); err == nil {
		t.Fatal("坏 JSON 应报错")
	}
	if *calls != 0 {
		t.Fatalf("守卫失败不应触达合成: %d", *calls)
	}
}

// TestBootstrapVoiceLineAudio_Batch 幂等批量：缺文件的角色补齐，已有文件的跳过；
// 引擎不活整批早退（零合成）。
func TestBootstrapVoiceLineAudio_Batch(t *testing.T) {
	a, s := newVoiceLineTestState(t)
	calls := stubVoiceLineAudioSeams(t, func(_ *mediaState, text, _, _ string) ([]byte, string, error) {
		return []byte("WAV:" + text), "audio/wav", nil
	}, true)

	_ = s.Upsert(&characterlib.Character{ID: "c1", Name: "苏念", Kind: characterlib.KindCustom, TTSLine: "剑来。"})
	_ = s.Upsert(&characterlib.Character{ID: "c2", Name: "林晚", Kind: characterlib.KindCustom, TTSLine: "夜色温柔。"})

	a.bootstrapVoiceLineAudio()
	if *calls != 2 {
		t.Fatalf("两个角色都缺文件应各合成一次, calls=%d", *calls)
	}
	// 幂等：已有文件全跳过
	a.bootstrapVoiceLineAudio()
	if *calls != 2 {
		t.Fatalf("二次批量应全跳过, calls=%d", *calls)
	}
}

func TestBootstrapVoiceLineAudio_EngineDead_Aborts(t *testing.T) {
	a, s := newVoiceLineTestState(t)
	calls := stubVoiceLineAudioSeams(t, func(*mediaState, string, string, string) ([]byte, string, error) {
		return []byte("WAV"), "audio/wav", nil
	}, false)

	_ = s.Upsert(&characterlib.Character{ID: "c1", Name: "苏念", Kind: characterlib.KindCustom})
	a.bootstrapVoiceLineAudio()
	if *calls != 0 {
		t.Fatalf("引擎不活应整批早退, calls=%d", *calls)
	}
}

// TestVoicePresetChanged 保存钩子变更判定：台词/音色/描述/样本/性别任一变→true。
func TestVoicePresetChanged(t *testing.T) {
	base := &characterlib.Character{ID: "c1", TTSLine: "剑来。", TTSVoiceID: "v", TTSDesc: "d", Gender: "female", DialogueSamples: []string{"a"}}
	if voicePresetChanged(base, &characterlib.Character{ID: "c1", TTSLine: "剑来。", TTSVoiceID: "v", TTSDesc: "d", Gender: "female", DialogueSamples: []string{"a"}}) {
		t.Fatal("全同不应判变更")
	}
	for _, mut := range []func(c *characterlib.Character){
		func(c *characterlib.Character) { c.TTSLine = "改" },
		func(c *characterlib.Character) { c.TTSVoiceID = "v2" },
		func(c *characterlib.Character) { c.TTSDesc = "d2" },
		func(c *characterlib.Character) { c.Gender = "male" },
		func(c *characterlib.Character) { c.DialogueSamples = []string{"b"} },
	} {
		cur := &characterlib.Character{ID: "c1", TTSLine: "剑来。", TTSVoiceID: "v", TTSDesc: "d", Gender: "female", DialogueSamples: []string{"a"}}
		mut(cur)
		if !voicePresetChanged(base, cur) {
			t.Fatal("任一语音面字段变化应判变更")
		}
	}
}
