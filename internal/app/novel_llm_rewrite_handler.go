package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/novelstyle"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/util"
)

// ── 重写/去味写回 helper（N3/N4/N5 收敛点）──────────────────────────
//
// 旧实现（LLM 重写与一键去味各自内联）把写盘错误直接丢掉：
// `_ = sm.Write(sc)` / `_ = pm.WriteChapter(...)`，随后照旧回报
// `done:true, rewritten:N`——前端据此提示「已改写 N 句」，磁盘其实没变。
// 且 v4 路径写场景后没有 syncBlobFromScenes，阅读页/全文检索/导出读的整章
// blob 仍是旧文，用户以为改写没生效。收敛为单一写回入口：
//   - 写失败 → 返回中文可读 error（调用方如实报失败，不再谎报 done）；
//   - 写成功 → v4 场景路径同步整章 blob 投影。

// writeBackRewritten 把重写/去味后的正文写回磁盘，成功后同步整章 blob 投影。
//   - isScene=true（v4 场景制）：写回场景 sceneID 后 syncBlobFromScenes
//     （与 SaveScene / GenerateScene / RestoreSnapshot 同口径）；
//   - isScene=false（v3 整章）：直接写章节文件（blob 即正稿，无需同步）。
//
// blob 同步失败沿用 syncBlobFromScenes 的既有口径（warn 不阻断，下一次场景写
// 自愈），此处不额外放大为错误。
func writeBackRewritten(pm *project.Manager, chapterNum int, sceneID string, isScene bool, text string) error {
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	if isScene {
		sm := pm.SceneManager(chapterNum)
		sc, err := sm.Read(sceneID)
		if err != nil {
			return fmt.Errorf("读取场景失败（第%d章 %s）：%w", chapterNum, sceneID, err)
		}
		sc.Content = text
		if err := sm.Write(sc); err != nil {
			return fmt.Errorf("保存场景失败（第%d章 %s）：%w", chapterNum, sceneID, err)
		}
		syncBlobFromScenes(pm, chapterNum)
		return nil
	}
	if err := pm.WriteChapter(chapterNum, text); err != nil {
		return fmt.Errorf("保存章节失败（第%d章）：%w", chapterNum, err)
	}
	return nil
}

// writeBackRewrittenFn 写回执行体（测试注入写失败桩的窄 seam，验证「写失败
// 返回 error 且不再谎报 done」；生产路径恒为 writeBackRewritten，与
// diskFreeFn / wxAgentExecToolFn 同款注入手法）。
var writeBackRewrittenFn = writeBackRewritten

// ── 刀 5 再续 · LLM 受限重写（高质量句级去 AI 味）──────────────────────
//
// 确定性 DeSlopRewrite 只做词表级替换；本绑定对「打分定位到的命中句」做
// 受限 LLM 重写——只改这些句、去 AI 腔、保持原意/人物/剧情，且只有复测后
// 分数下降、篇幅变化不大才落盘（安全闸）。按 v4 逐场景 / v3 整章处理。

type rewriteSentence struct {
	text  string
	start int
	end   int
}

func (a *writingState) RewriteChapterAiTaste(chapterNum int) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	if a.client == nil {
		return nil, fmt.Errorf("AI client not ready")
	}

	units := []rewriteUnit{}
	if pm.IsV4() {
		sm := pm.SceneManager(chapterNum)
		metas, err := sm.List()
		if err != nil {
			return nil, err
		}
		for _, meta := range metas {
			sc, rerr := sm.Read(meta.ID)
			if rerr != nil {
				continue
			}
			units = append(units, rewriteUnit{id: meta.ID, isScene: true, text: sc.Content})
		}
	} else {
		content, err := pm.ReadChapter(chapterNum)
		if err != nil {
			return nil, fmt.Errorf("读取章节失败: %w", err)
		}
		units = append(units, rewriteUnit{id: "", isScene: false, text: content})
	}

	totalApplied, totalBefore, totalAfter := 0, 0, 0
	for i := range units {
		rw, before, after, applied, err := a.rewriteUnit(chapterNum, units[i].text)
		if err != nil {
			return nil, err
		}
		if applied == 0 {
			continue
		}
		units[i].text = rw
		units[i].applied = applied
		totalApplied += applied
		totalBefore += before
		totalAfter += after
	}

	// 写回：只写真正改善的单元；写失败即如实返回 error（旧实现吞错后仍回报
	// done:true + rewritten:N，前端提示「已改写 N 句」而磁盘没变）。v4 场景
	// 写回由 helper 统一同步整章 blob，阅读页/检索/导出不再读旧文。
	for i := range units {
		if units[i].applied == 0 {
			continue // 未改善的单元不落盘（写回同文本只徒增 mtime 与误覆盖风险）
		}
		if err := writeBackRewrittenFn(pm, chapterNum, units[i].id, units[i].isScene, units[i].text); err != nil {
			return nil, err
		}
	}

	if totalApplied == 0 {
		return map[string]interface{}{"chapterNum": chapterNum, "done": false, "reason": "未命中需重写的句"}, nil
	}
	return map[string]interface{}{
		"chapterNum":  chapterNum,
		"done":        true,
		"rewritten":   totalApplied,
		"beforeScore": totalBefore,
		"afterScore":  totalAfter,
	}, nil
}

type rewriteUnit struct {
	id      string
	isScene bool
	text    string
	// applied 本单元实际替换的句数（0 = 未改善/未命中，写回阶段跳过）。
	applied int
}

// rewriteUnit 对单个文本单元做「打分→定位命中句→LLM 批量重写→安全替换→复测」。
func (a *writingState) rewriteUnit(chapterNum int, text string) (string, int, int, int, error) {
	if strings.TrimSpace(text) == "" {
		return text, 0, 0, 0, nil
	}
	before, _ := novelstyle.ScoreTextNoRef(text)
	sentences := splitSentences(text)
	targets := pickFlaggedSentences(text, before, sentences, 10)
	if len(targets) == 0 {
		return text, 0, 0, 0, nil
	}
	rewrites, err := a.llmRewriteSentences(chapterNum, targets)
	if err != nil {
		return text, 0, 0, 0, err
	}
	rewritten := text
	applied := 0
	for _, t := range targets {
		rw, ok := rewrites[t.text]
		if !ok || strings.TrimSpace(rw) == "" {
			continue
		}
		if abs(len([]rune(rw))-len([]rune(t.text))) > len([]rune(t.text))/2 {
			continue // 篇幅漂移过大，放弃该句
		}
		if strings.Contains(rewritten, t.text) {
			rewritten = strings.Replace(rewritten, t.text, rw, 1)
			applied++
		}
	}
	if applied == 0 {
		return text, before.Score, before.Score, 0, nil
	}
	after, _ := novelstyle.ScoreTextNoRef(rewritten)
	if after.Score >= before.Score {
		return text, before.Score, after.Score, 0, nil // 安全闸：未改善不落盘
	}
	return rewritten, before.Score, after.Score, applied, nil
}

func (a *writingState) llmRewriteSentences(chapterNum int, targets []rewriteSentence) (map[string]string, error) {
	sb := strings.Builder{}
	for i, t := range targets {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, t.text))
	}
	system := "你是资深小说文字编辑。以下句子是从小说中抽出的、AI 味较重的句子。" +
		"请把每句改写得更自然、更像真人手写，消除 AI 腔（空洞比喻、情绪直述、连接词堆砌、四字格堆叠、" +
		"「眼帘/眸光/微微上扬/缓缓」类词），保持原意、人物性格、剧情、字数接近。"
	user := fmt.Sprintf("严格按 JSON 输出：{\"rewrites\":[{\"index\":0,\"text\":\"改写后句子\"}]}\n\n待改写句子：\n%s", sb.String())

	eng, model, _ := a.routeModel("novel")
	if model == "" {
		return nil, fmt.Errorf("未找到可用模型（可能离线）")
	}
	reply, err := a.client.ChatSimpleStreamWithOptions(context.Background(), model, system, user, ai.ChatSimpleOptions{
		EngineID: eng, Feature: "novel", Temperature: 0.7, MaxTokens: 2048,
	})
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Rewrites []struct {
			Index int    `json:"index"`
			Text  string `json:"text"`
		} `json:"rewrites"`
	}
	if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &parsed); err != nil {
		return nil, fmt.Errorf("解析改写结果失败: %w", err)
	}
	out := map[string]string{}
	for _, r := range parsed.Rewrites {
		if r.Text == "" || r.Index < 1 || r.Index > len(targets) {
			continue
		}
		out[targets[r.Index-1].text] = strings.TrimSpace(r.Text)
	}
	return out, nil
}

func pickFlaggedSentences(content string, score *novelstyle.TasteScore, sentences []rewriteSentence, maxN int) []rewriteSentence {
	if score == nil || len(score.Issues) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var out []rewriteSentence
	for _, iss := range score.Issues {
		for _, s := range sentences {
			if iss.Start >= s.start && iss.End <= s.end {
				if seen[s.text] {
					break
				}
				seen[s.text] = true
				out = append(out, s)
				break
			}
		}
		if len(out) >= maxN {
			break
		}
	}
	return out
}

func splitSentences(content string) []rewriteSentence {
	runes := []rune(content)
	var out []rewriteSentence
	start := 0
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '。' || r == '！' || r == '？' || r == '…' || r == '\n' {
			end := i + 1
			for end < len(runes) && (runes[end] == '”' || runes[end] == '’' || runes[end] == '"' || runes[end] == '\n') {
				end++
			}
			text := strings.TrimSpace(string(runes[start:end]))
			if text != "" {
				out = append(out, rewriteSentence{text: text, start: start, end: end})
			}
			start = end
			i = end - 1
		}
	}
	if start < len(runes) {
		text := strings.TrimSpace(string(runes[start:]))
		if text != "" {
			out = append(out, rewriteSentence{text: text, start: start, end: len(runes)})
		}
	}
	return out
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
