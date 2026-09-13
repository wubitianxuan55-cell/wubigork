package app

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gaea/gaea/internal/novelstyle"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// ── 文风指纹参考档（并行池小说线：内核 ComputeFingerprint/Delta 的落地面）────
//
// 参考档 = 构建时点全部已写章节的风格基线（v1 口径：作者成稿即样本；
// 「只学作者手改」的结算路径留后续刀）。构建后 NovelFingerprintScore 用
// 参考档对照打分（ScoreText + Delta），与既有的一键去味/定点重写衔接成
// 「指纹 → 打分 → 重写」闭环。纯确定性、零 LLM、零网络。

// 参考档构建门槛：低于门槛的统计不稳定（句长分布/TTR 会失真），诚实拒绝。
const (
	fingerprintMinChapters = 3
	fingerprintMinChars    = 3000
)

// FingerprintSummary 文风指纹面板摘要（Fingerprint 的可显示子集，不含函数词向量）。
type FingerprintSummary struct {
	SentenceMean      float64  `json:"sentenceMean"`
	SentenceSd        float64  `json:"sentenceSd"`
	ParaMean          float64  `json:"paraMean"`
	TTR1000           float64  `json:"ttr1000"`
	DialogRatio       float64  `json:"dialogRatio"`
	FourCharRatio     float64  `json:"fourCharRatio"`
	ConnectiveDensity float64  `json:"connectiveDensity"`
	AdjAdvDensity     float64  `json:"adjAdvDensity"`
	TopBigrams        []string `json:"topBigrams"`
	TopTrigrams       []string `json:"topTrigrams"`
	AuthorSignWords   []string `json:"authorSignWords"`
}

// FingerprintStatusPayload NovelFingerprintStatus / NovelFingerprintBuild 返回。
type FingerprintStatusPayload struct {
	Exists   bool                `json:"exists"`
	BuiltAt  string              `json:"builtAt,omitempty"`
	Chapters int                 `json:"chapters,omitempty"`
	Chars    int                 `json:"chars,omitempty"`
	Summary  *FingerprintSummary `json:"summary,omitempty"`
}

// FingerprintIssueView 单条 AI 味命中（附原文摘录，面板直显免前端按 rune 切字）。
type FingerprintIssueView struct {
	Start      int    `json:"start"`
	End        int    `json:"end"`
	Reason     string `json:"reason"`
	Severity   string `json:"severity"`
	Suggestion string `json:"suggestion"`
	Excerpt    string `json:"excerpt"`
}

// FingerprintScorePayload NovelFingerprintScore 返回。
type FingerprintScorePayload struct {
	ChapterNum int                    `json:"chapterNum"`
	RefExists  bool                   `json:"refExists"`
	Score      int                    `json:"score"`
	Delta      *float64               `json:"delta,omitempty"` // 越小越像参考档；无参考档时缺省
	Issues     []FingerprintIssueView `json:"issues"`
}

// fingerprintSummaryOf 从指纹提取面板摘要（纯函数）。
func fingerprintSummaryOf(fp *novelstyle.Fingerprint) *FingerprintSummary {
	if fp == nil {
		return nil
	}
	return &FingerprintSummary{
		SentenceMean:      fp.SentenceLen.Mean,
		SentenceSd:        fp.SentenceLen.Sd,
		ParaMean:          fp.ParaLen.Mean,
		TTR1000:           fp.TTR1000,
		DialogRatio:       fp.DialogRatio,
		FourCharRatio:     fp.FourCharRatio,
		ConnectiveDensity: fp.ConnectiveDensity,
		AdjAdvDensity:     fp.AdjAdvDensity,
		TopBigrams:        fp.TopBigrams,
		TopTrigrams:       fp.TopTrigrams,
		AuthorSignWords:   fp.AuthorSignWords,
	}
}

// collectChapterSamples 逐章收集成稿文本（v4 场景工程走 Stitch 拼接，v3 整章），
// 返回样本、章数、非空白字数。遍历口径与 ForEachChapter 一致：读取失败即停。
func collectChapterSamples(getChapter func(num int) (string, error)) ([]string, int, int) {
	var samples []string
	chapters, chars := 0, 0
	for i := 1; ; i++ {
		content, err := getChapter(i)
		if err != nil {
			break
		}
		if content == "" {
			continue
		}
		samples = append(samples, content)
		chapters++
		chars += countNonSpaceRunesOf(content)
	}
	return samples, chapters, chars
}

func countNonSpaceRunesOf(text string) int {
	n := 0
	for _, r := range text {
		switch r {
		case ' ', '\t', '\n', '\r', '\v', '\f', 0x3000:
		default:
			n++
		}
	}
	return n
}

// fingerprintStatusPayload 读参考档组装状态负载；文件不存在是正常态（exists:false）。
func fingerprintStatusPayload(pm *project.Manager) (FingerprintStatusPayload, error) {
	sf, err := pm.ReadStyleFingerprint()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return FingerprintStatusPayload{Exists: false}, nil
		}
		return FingerprintStatusPayload{}, fmt.Errorf("读取文风参考档失败: %w", err)
	}
	fp, err := novelstyle.LoadFingerprint(sf.Fingerprint)
	if err != nil {
		return FingerprintStatusPayload{}, fmt.Errorf("文风参考档损坏: %w", err)
	}
	return FingerprintStatusPayload{
		Exists:   true,
		BuiltAt:  sf.BuiltAt,
		Chapters: sf.Chapters,
		Chars:    sf.Chars,
		Summary:  fingerprintSummaryOf(fp),
	}, nil
}

// NovelFingerprintStatus 查询参考档状态（未构建返回 exists:false，不算错）。
func (a *writingState) NovelFingerprintStatus() (FingerprintStatusPayload, error) {
	pm := a.getPM()
	if pm == nil {
		return FingerprintStatusPayload{}, fmt.Errorf("请先打开项目")
	}
	return fingerprintStatusPayload(pm)
}

// NovelFingerprintBuild 用全部已写章节构建文风参考档并落盘 fingerprint.json。
func (a *writingState) NovelFingerprintBuild() (FingerprintStatusPayload, error) {
	ensureNovelStyleWords() // 词表影响分词词表，与去味/打分口径一致
	pm := a.getPM()
	if pm == nil {
		return FingerprintStatusPayload{}, fmt.Errorf("请先打开项目")
	}
	samples, chapters, chars := collectChapterSamples(pm.ReadChapterAsStitch)
	if chapters < fingerprintMinChapters || chars < fingerprintMinChars {
		return FingerprintStatusPayload{}, fmt.Errorf(
			"样本不足：需至少 %d 章且 %d 字（当前 %d 章 %d 字），先多写几章再构建",
			fingerprintMinChapters, fingerprintMinChars, chapters, chars)
	}
	fp, err := novelstyle.ComputeFingerprint(samples)
	if err != nil {
		return FingerprintStatusPayload{}, fmt.Errorf("计算文风指纹失败: %w", err)
	}
	raw, err := fp.ToJSON()
	if err != nil {
		return FingerprintStatusPayload{}, fmt.Errorf("序列化文风指纹失败: %w", err)
	}
	sf := &types.StyleFingerprintFile{
		BuiltAt:     time.Now().Format(time.RFC3339),
		Chapters:    chapters,
		Chars:       chars,
		Fingerprint: raw,
	}
	if err := pm.WriteStyleFingerprint(sf); err != nil {
		return FingerprintStatusPayload{}, fmt.Errorf("保存文风参考档失败: %w", err)
	}
	return FingerprintStatusPayload{
		Exists:   true,
		BuiltAt:  sf.BuiltAt,
		Chapters: chapters,
		Chars:    chars,
		Summary:  fingerprintSummaryOf(fp),
	}, nil
}

// excerptOf 取 [start,end) rune 区间原文，超长截断（面板摘录用）。
func excerptOf(text string, start, end int) string {
	rs := []rune(text)
	if start < 0 || end > len(rs) || start >= end {
		return ""
	}
	if end-start > 40 {
		return string(rs[start:start+40]) + "…"
	}
	return string(rs[start:end])
}

// NovelFingerprintScore 对单章做 AI 味体检：有参考档时 ScoreText + Delta 对照，
// 无参考档时按通用阈值打分（refExists:false），两条路都可用。
func (a *writingState) NovelFingerprintScore(chapterNum int) (FingerprintScorePayload, error) {
	ensureNovelStyleWords()
	pm := a.getPM()
	if pm == nil {
		return FingerprintScorePayload{}, fmt.Errorf("请先打开项目")
	}
	if chapterNum <= 0 {
		return FingerprintScorePayload{}, fmt.Errorf("章节号非法")
	}
	text, err := pm.ReadChapterAsStitch(chapterNum)
	if err != nil {
		return FingerprintScorePayload{}, fmt.Errorf("读取章节失败: %w", err)
	}
	if countNonSpaceRunesOf(text) == 0 {
		return FingerprintScorePayload{}, fmt.Errorf("第 %d 章为空，无可体检内容", chapterNum)
	}

	payload := FingerprintScorePayload{ChapterNum: chapterNum, Issues: []FingerprintIssueView{}}

	var ref *novelstyle.Fingerprint
	if sf, err := pm.ReadStyleFingerprint(); err == nil {
		if fp, lerr := novelstyle.LoadFingerprint(sf.Fingerprint); lerr == nil {
			ref = fp
			payload.RefExists = true
		} // 参考档损坏按无参考档降级，不挡体检
	}
	score, err := novelstyle.ScoreText(text, ref)
	if err != nil {
		return FingerprintScorePayload{}, fmt.Errorf("AI 味打分失败: %w", err)
	}
	payload.Score = score.Score
	if ref != nil {
		if observed, ferr := novelstyle.ComputeFingerprint([]string{text}); ferr == nil {
			d := novelstyle.Delta(observed, ref)
			payload.Delta = &d
		}
	}
	for _, iss := range score.Issues {
		payload.Issues = append(payload.Issues, FingerprintIssueView{
			Start:      iss.Start,
			End:        iss.End,
			Reason:     iss.Reason,
			Severity:   iss.Severity,
			Suggestion: iss.Suggestion,
			Excerpt:    excerptOf(text, iss.Start, iss.End),
		})
	}
	return payload, nil
}
