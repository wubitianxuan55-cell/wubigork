package app

// ── 章级变更声明：提取 → 五类门禁 → 过闸落盘（阶段八 8.5 批B）──
//
// 生成链在摘要段之后提取 ---CHAPTER_CHANGES--- JSON，经 novelgate 五类确定性
// 校验：过闸=随章节摘要落盘（ChapterSummary.DeclChanges，定稿草案 8.4 的零 LLM
// 预填充源）；拒收=不落盘+done 事件带人可读问题清单（作者可重新生成或手动修正）。
// 声明缺失/解析失败都是软降级——正文是主产物，永不因声明失效。

import (
	"encoding/json"
	"strings"

	"github.com/gaea/gaea/internal/novelgate"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// chapterChangesMarker 变更声明标记（摘要段之后；正文=第一标记之前，流式剥离
// 逻辑零改动）。
const chapterChangesMarker = "---CHAPTER_CHANGES---"

// splitDeclSegment 从摘要区全文切出变更声明段（返回摘要区文本、声明 JSON 原文）。
func splitDeclSegment(text string) (summaryPart, declPart string) {
	idx := strings.Index(text, chapterChangesMarker)
	if idx < 0 {
		return text, ""
	}
	return text[:idx], text[idx+len(chapterChangesMarker):]
}

// parseChapterChanges 解析声明 JSON（宽容剥围栏）；空段/解析失败返回 (nil, ok=false)。
func parseChapterChanges(raw string) (*types.ChapterChangeDeclaration, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, false
	}
	// 剥 ``` 围栏（模型常见输出形态；裸 JSON 直过）
	if strings.HasPrefix(s, "```") {
		if i := strings.Index(s, "\n"); i >= 0 {
			s = s[i+1:]
		}
		if j := strings.LastIndex(s, "```"); j >= 0 {
			s = s[:j]
		}
		s = strings.TrimSpace(s)
	}
	var decl types.ChapterChangeDeclaration
	if err := json.Unmarshal([]byte(s), &decl); err != nil {
		return nil, false
	}
	return &decl, true
}

// resolveChapterChanges 生成链收口：切段→解析→五类门禁。返回
// (过闸声明或 nil, 门禁报告)。nil 声明+空 issues=模型未声明（软降级）；
// issues 非空=拒收（declared 但 accepted=false，问题清单人可读）。
func resolveChapterChanges(summaryRegion, content string, chars []types.Character,
	ffs []types.Foreshadow, chapterNum int) (*types.ChapterChangeDeclaration, []novelgate.Issue) {
	_, declPart := splitDeclSegment(summaryRegion)
	decl, ok := parseChapterChanges(declPart)
	if !ok {
		return nil, nil // 未声明或不可解析：软降级（不计违规）
	}
	return decl, novelgate.ChangeDeclarationIssues(decl, chars, ffs, chapterNum, content)
}

// attachDeclToSummary 过闸声明随摘要落盘（拒绝=不写 DeclChanges，摘要照常落）。
func attachDeclToSummary(summary *types.ChapterSummary, decl *types.ChapterChangeDeclaration, issues []novelgate.Issue) bool {
	if summary == nil || decl == nil {
		return false
	}
	if len(novelgate.DeclBlockingIssues(issues)) > 0 {
		return false
	}
	summary.DeclChanges = decl
	return true
}

// declGateReport done 事件面（前端人可读）：declared/accepted/issues。
func declGateReport(decl *types.ChapterChangeDeclaration, issues []novelgate.Issue) map[string]interface{} {
	if decl == nil {
		return map[string]interface{}{"declared": false, "accepted": false, "issues": []string{}}
	}
	blocking := novelgate.DeclBlockingIssues(issues)
	msgs := make([]string, 0, len(blocking))
	for _, it := range blocking {
		msgs = append(msgs, it.Message)
	}
	return map[string]interface{}{
		"declared": true,
		"accepted": len(blocking) == 0,
		"issues":   msgs,
	}
}

// readDeclCapableChars 读角色卡（门禁与定稿草案共用；失败=空集，校验按未知实体
// 处理——诚实拒收优于静默放过）。
func readDeclCapableChars(pm *project.Manager) []types.Character {
	cf, err := pm.ReadCharacters()
	if err != nil || cf == nil {
		return nil
	}
	return cf.Characters
}

// readGateForeshadows 读伏笔登记表（门禁引用对账；失败=空集——引用校验按
// 「不在登记表」处理，诚实拒收优于静默放过）。
func readGateForeshadows(pm *project.Manager) []types.Foreshadow {
	ff, err := pm.ReadForeshadows()
	if err != nil || ff == nil {
		return nil
	}
	return ff.Items
}
