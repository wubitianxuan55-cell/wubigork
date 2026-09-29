package app

// 风格学习回灌（长篇刀5）：NovelStyleDigestBuild/Get/Clear 三个绑定面 +
// 生成注入区段 styleDigestSection（CreateChapter 与逐场景流共用；零 digest
// 零注入）。构建复用指纹链（collectChapterSamples→ComputeFingerprint→DigestOf），
// digest 与 fingerprint.json 相互独立（指纹管体检打分，digest 管生成约束）。

import (
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/novelstyle"
	"github.com/gaea/gaea/internal/types"
)

// NovelStyleDigestBuild 从成稿章节构建风格摘要档（可执行写作指令集）。
// 样本门槛与指纹一致（同一条链）；已存在则重建（覆盖）。
func (a *writingState) NovelStyleDigestBuild() (map[string]interface{}, error) {
	ensureNovelStyleWords()
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	samples, chapters, chars := collectChapterSamples(pm.ReadChapterAsStitch)
	if chapters < fingerprintMinChapters || chars < fingerprintMinChars {
		return nil, fmt.Errorf(
			"样本不足：需至少 %d 章且 %d 字（当前 %d 章 %d 字）——先多写几章（作者认可的成稿）再学习",
			fingerprintMinChapters, fingerprintMinChars, chapters, chars)
	}
	fp, err := novelstyle.ComputeFingerprint(samples)
	if err != nil {
		return nil, fmt.Errorf("计算文风指纹失败: %w", err)
	}
	instr := novelstyle.DigestOf(fp)
	if instr == "" {
		return nil, fmt.Errorf("指纹为空：无法编译写作指令")
	}
	df := &types.StyleDigestFile{
		BuiltAt:           time.Now().Format(time.RFC3339),
		Chapters:          chapters,
		Chars:             chars,
		Instructions:      instr,
		SignWords:         fp.AuthorSignWords,
		SentenceMean:      fp.SentenceLen.Mean,
		SentenceSd:        fp.SentenceLen.Sd,
		DialogRatio:       fp.DialogRatio,
		FourCharRatio:     fp.FourCharRatio,
		ConnectiveDensity: fp.ConnectiveDensity,
		AdjAdvDensity:     fp.AdjAdvDensity,
	}
	if err := pm.WriteStyleDigest(df); err != nil {
		return nil, fmt.Errorf("保存风格摘要档失败: %w", err)
	}
	return map[string]interface{}{
		"builtAt": df.BuiltAt, "chapters": chapters, "chars": chars,
		"instructions": df.Instructions, "signWords": df.SignWords,
	}, nil
}

// NovelStyleDigestGet 读风格摘要档；未构建返回 exists:false（不算错）。
func (a *writingState) NovelStyleDigestGet() (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	df, err := pm.ReadStyleDigest()
	if err != nil {
		return nil, fmt.Errorf("读取风格摘要档失败: %w", err)
	}
	if df == nil {
		return map[string]interface{}{"exists": false}, nil
	}
	return map[string]interface{}{
		"exists": true, "builtAt": df.BuiltAt, "chapters": df.Chapters, "chars": df.Chars,
		"instructions": df.Instructions, "signWords": df.SignWords,
	}, nil
}

// NovelStyleDigestClear 清除风格摘要档（生成不再注入风格约束；指纹体检档不受影响）。
func (a *writingState) NovelStyleDigestClear() error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	return pm.ClearStyleDigest()
}

// digestBody 读 digest 的指令正文（无标题；并入文风合并区段用；空档空串）。
func (a *writingState) digestBody(pm styleDigestPM) string {
	df, err := pm.ReadStyleDigest()
	if err != nil || df == nil {
		return ""
	}
	return df.Instructions
}

// joinStyleSections 文风合并区段（刀6）：style.md 显式偏好与成稿学习指令
// 并为单一区段——偏好优先、学习跟随；只有一方时保持各自完整语义（标题仍
// 单一）；双空返回空串（零注入）。合并预算 ctxStyleBudget+400（学习指令行短）。
func joinStyleSections(styleBody, digestInstr string) string {
	styleBody = strings.TrimSpace(styleBody)
	digestInstr = strings.TrimSpace(digestInstr)
	if styleBody == "" && digestInstr == "" {
		return ""
	}
	if digestInstr == "" {
		return "## 文风与表达（作者显式偏好，优先于通用默认写法）\n" +
			"以下只约束表达方式（句长/视角/标点/对话/修辞/收尾等）；事件、事实与信息边界仍以细纲为准：\n" + styleBody
	}
	if styleBody == "" {
		return "## 文风与表达（从你的成稿学到的习惯，按此口径写）\n" + digestInstr
	}
	merged := "## 文风与表达（作者显式偏好优先；成稿学习口径跟随）\n" +
		"【显式偏好】（只约束表达方式；事件与信息边界以细纲为准）\n" + styleBody +
		"\n【成稿学习】\n" + digestInstr
	return truncateCtxStyle(merged)
}

// truncateCtxStyle 文风合并区段截断（偏好优先保留：先保 styleBody，学习指令收尾截断）。
func truncateCtxStyle(merged string) string {
	const budget = 1600 + 400
	if len([]rune(merged)) <= budget {
		return merged
	}
	return string([]rune(merged)[:budget]) + "……"
}

// styleDigestSection 生成注入的作者风格约束区段（空档零注入；独立消费面保留——
// 逐场景流与 Inventory 用）。
func (a *writingState) styleDigestSection(pm styleDigestPM) string {
	df, err := pm.ReadStyleDigest()
	if err != nil || df == nil || df.Instructions == "" {
		return ""
	}
	return "## 作者风格约束（从你的成稿学到的表达习惯，按此口径写）\n" + df.Instructions
}

// styleDigestPM 读档依赖面（*project.Manager 满足）。
type styleDigestPM interface {
	ReadStyleDigest() (*types.StyleDigestFile, error)
}
