package app

// T7-2 可见性收口：LocalTranslate 可取消 ctx / 长文本分段 / 单段失败重试一次 /
// 部分结果保留（不静默吞错）测试。

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/modelengine"
)

// TestSplitTranslateSegments 分段逻辑：短文本单段；无换行长文本按 rune 硬切；
// 有换行时按段落边界分组（不切断段落）。
func TestSplitTranslateSegments(t *testing.T) {
	// 短文本：单段。
	if segs := splitTranslateSegments("你好世界"); len(segs) != 1 || segs[0] != "你好世界" {
		t.Fatalf("短文本应单段: %v", segs)
	}

	// 无换行长文本：按 1500 rune 硬切。
	long := strings.Repeat("啊", 3500)
	segs := splitTranslateSegments(long)
	if len(segs) != 3 {
		t.Fatalf("3500 rune 应切 3 段, got %d", len(segs))
	}
	for _, s := range segs {
		if len([]rune(s)) > translateMaxSegmentRunes {
			t.Errorf("段超长: %d", len([]rune(s)))
		}
	}
	if strings.Join(segs, "") != long {
		t.Error("硬切拼接应还原原文")
	}

	// 有换行：段落边界分组（1600+1600 两段）。
	segs = splitTranslateSegments(strings.Repeat("你", 1600) + "\n" + strings.Repeat("好", 1600))
	if len(segs) != 2 {
		t.Fatalf("两段应切 2 段, got %d", len(segs))
	}
	if segs[0] != strings.Repeat("你", 1600) || segs[1] != strings.Repeat("好", 1600) {
		t.Errorf("段落边界切分错误: len0=%d len1=%d", len([]rune(segs[0])), len([]rune(segs[1])))
	}
}

// TestLocalTranslate_CancelledCtx 预取消的 ctx：立即返回 context.Canceled，
// 不发请求、不静默成功。
func TestLocalTranslate_CancelledCtx(t *testing.T) {
	a, _ := translateTestApp(t, []modelengine.ModelInfo{{ID: "Hy-MT2:7B"}}, func(w http.ResponseWriter, r *http.Request) {
		t.Error("取消后不应发出翻译请求")
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := a.localTranslate(ctx, LocalTranslateRequest{Text: "你好", TargetLang: "en"})
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("应返回 context.Canceled, got %v", err)
	}
	if res.Text != "" {
		t.Errorf("取消后不应有部分结果: %+v", res)
	}
}

// TestLocalTranslate_PartialResultsOnSegmentFailure 长文本第二段连续失败：
// 保留第一段译文 + Partial=true + 明确错误（不静默吞错）。
func TestLocalTranslate_PartialResultsOnSegmentFailure(t *testing.T) {
	seg2 := strings.Repeat("好", 1600)
	a, _ := translateTestApp(t, []modelengine.ModelInfo{{ID: "Hy-MT2:7B"}}, func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(400)
			return
		}
		msgs := req["messages"].([]any)
		user, _ := msgs[1].(map[string]any)["content"].(string)
		if strings.Contains(user, seg2) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"T1"}}]}`))
	})

	text := strings.Repeat("你", 1600) + "\n" + seg2
	res, err := a.localTranslate(context.Background(), LocalTranslateRequest{Text: text, TargetLang: "en"})
	if err == nil {
		t.Fatal("第二段失败应返回错误")
	}
	if !strings.Contains(err.Error(), "第 2/2 段翻译失败") {
		t.Errorf("错误应指明失败段落: %v", err)
	}
	if !res.Partial || res.Text != "T1" {
		t.Errorf("应保留第一段部分结果: %+v", res)
	}
}

// TestLocalTranslate_RetrySucceeds 单段首试失败、重试成功：整体成功，
// 请求数为 2（重试不静默吞错也不放弃）。
func TestLocalTranslate_RetrySucceeds(t *testing.T) {
	var calls int
	a, _ := translateTestApp(t, []modelengine.ModelInfo{{ID: "Hy-MT2:7B"}}, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Hello"}}]}`))
	})

	res, err := a.localTranslate(context.Background(), LocalTranslateRequest{Text: "你好", TargetLang: "en"})
	if err != nil {
		t.Fatalf("重试成功应整体成功: %v", err)
	}
	if res.Text != "Hello" || res.Partial {
		t.Errorf("结果异常: %+v", res)
	}
	if calls != 2 {
		t.Errorf("应重试一次（2 次请求）, got %d", calls)
	}
}

// TestLocalTranslate_FirstSegmentFailNoPartial 第一段就失败（含重试）：
// 无任何部分结果，返回明确错误。
func TestLocalTranslate_FirstSegmentFailNoPartial(t *testing.T) {
	a, _ := translateTestApp(t, []modelengine.ModelInfo{{ID: "Hy-MT2:7B"}}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	text := strings.Repeat("你", 1600) + "\n" + strings.Repeat("好", 1600)
	res, err := a.localTranslate(context.Background(), LocalTranslateRequest{Text: text, TargetLang: "en"})
	if err == nil {
		t.Fatal("应返回错误")
	}
	if !strings.Contains(err.Error(), "第 1/2 段翻译失败") {
		t.Errorf("错误应指明第一段: %v", err)
	}
	if res.Partial || res.Text != "" {
		t.Errorf("首段失败不应有部分结果: %+v", res)
	}
}

// TestTranslateTextTool_PartialIncludedInError 工具路径：部分结果随错误信息
// 返回给调用方，不静默丢弃。
func TestTranslateTextTool_PartialIncludedInError(t *testing.T) {
	seg2 := strings.Repeat("好", 1600)
	a, _ := translateTestApp(t, []modelengine.ModelInfo{{ID: "Hy-MT2:7B"}}, func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		msgs := req["messages"].([]any)
		user, _ := msgs[1].(map[string]any)["content"].(string)
		if strings.Contains(user, seg2) {
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"T1"}}]}`))
	})
	text := strings.Repeat("你", 1600) + "\n" + seg2
	tool := translateTextTool{a: a}
	args, err := json.Marshal(map[string]string{"text": text, "target_lang": "en"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = tool.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("应返回错误")
	}
	if !strings.Contains(err.Error(), "已翻译部分：T1") {
		t.Errorf("错误应携带已翻译部分: %v", err)
	}
}
