package style

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/util"
)

// ── 风格档案 ─────────────────────────────────────────────────

// Profile 作者风格档案（.gaea/style-profile.json）。
//
// 边界声明（审计 IN1-05）：Profile=生成注入口径（ToStyleGuide 注入写作
// prompt），与 project 层 StyleFingerprint/StyleDigest（fingerprint.json /
// style_digest.json）=评分口径互为独立真相源，刻意分层，勿顺手合并；另见
// project.Manager.StyleFingerprintPath 上的对偶声明。路径与品牌兼容
// （.wubigork 回退）单源收口在 project.Manager
// （StyleProfilePath / ReadStyleProfileFile / WriteStyleProfileFile），
// 本包不再自带第二份路径/兼容分支。
type Profile struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Traits      map[string]string `json:"traits"`       // 风格特征
	RawMarkdown string            `json:"raw_markdown"` // 原始风格指导
}

// ParamClamp 生成参数钳制（S1.5-B play 内容护栏透传，platform_handler
// AnalyzeStyle 消费）：零值 = 不钳制（现状逐字节）。只降不升——低于上限的
// 现值保留，不抬高。
type ParamClamp struct {
	TemperatureMax  float64 // 生成温度上限（0 = 不钳制）
	MaxOutputTokens int     // max_tokens 上限（0 = 不钳制）
}

// Analyzer 风格分析器
type Analyzer struct {
	pm     *project.Manager
	client *ai.Client
	model  string
	clamp  ParamClamp
}

// NewAnalyzer 创建风格分析器。clamp 为护栏钳制参数（零值 = 现状）。
func NewAnalyzer(pm *project.Manager, client *ai.Client, model string, clamp ParamClamp) *Analyzer {
	return &Analyzer{pm: pm, client: client, model: model, clamp: clamp}
}

// clampParams 按 clamp 钳制生成参数（S1.5-B；零值 = 原样返回）。
func clampParams(opts ai.ChatSimpleOptions, clamp ParamClamp) ai.ChatSimpleOptions {
	if clamp.TemperatureMax > 0 && opts.Temperature > clamp.TemperatureMax {
		opts.Temperature = clamp.TemperatureMax
	}
	if clamp.MaxOutputTokens > 0 && opts.MaxTokens > clamp.MaxOutputTokens {
		opts.MaxTokens = clamp.MaxOutputTokens
	}
	return opts
}

// Analyze 分析已有章节，生成风格档案
func (a *Analyzer) Analyze() (*Profile, error) {
	// 收集已有章节内容（前3章作为样本）
	var samples []string
	for chapterNum := 1; chapterNum <= 3; chapterNum++ {
		content, err := a.pm.ReadChapter(chapterNum)
		if err != nil {
			break
		}
		// 取前500字
		runes := []rune(content)
		if len(runes) > 500 {
			content = string(runes[:500])
		}
		samples = append(samples, content)
	}

	if len(samples) == 0 {
		return nil, fmt.Errorf("没有可分析的章节")
	}

	sampleText := strings.Join(samples, "\n\n---\n\n")

	systemPrompt := `你是文学风格分析师。分析给定的小说文本，提取作者的写作风格特征。

请按以下维度分析，输出 JSON：
{
  "name": "风格名称（简洁）",
  "description": "一句话风格总结",
  "traits": {
    "narrative_voice": "叙事声音（如：冷静客观/热情奔放/冷峻克制/诗意象徵）",
    "sentence_style": "句式特点（如：短句为主/长短交错/排比丰富/简洁直白）",
    "pacing": "节奏感（如：快速推进/舒缓细腻/张弛有度）",
    "description_density": "描写密度（如：白描简约/细节丰富/感官饱满）",
    "dialogue_style": "对话风格（如：自然口语/文雅书面/犀利简洁）",
    "vocabulary_level": "词汇层次（如：通俗易懂/文学性强/专业术语多）",
    "emotional_tone": "情感基调（如：温暖治愈/冷峻压抑/热血激昂/灰色写实）",
    "chinese_style": "中文特色（如：古风韵味/现代口语/方言元素/西化句式少）"
  }
}`

	userPrompt := fmt.Sprintf("请分析以下小说的写作风格：\n\n%s", sampleText)

	reply, err := a.client.ChatSimpleStreamWithOptions(
		context.TODO(),
		a.model,
		systemPrompt,
		userPrompt,
		// S1.5-B：护栏钳制（0.3/1024 为该点现状基线；clamp 零值 = 原样）。
		clampParams(ai.ChatSimpleOptions{
			Temperature: 0.3,
			MaxTokens:   1024,
		}, a.clamp),
	)
	if err != nil {
		return nil, fmt.Errorf("风格分析失败: %w", err)
	}

	// 解析结果
	var profile Profile
	jsonStr := util.ExtractJSON(reply)
	if err := json.Unmarshal([]byte(jsonStr), &profile); err != nil {
		// 回退：使用原始回复作为风格指导
		profile = Profile{
			Name:        "自动分析风格",
			Description: "AI 从已有章节中学习",
			Traits:      map[string]string{"raw": reply},
		}
	}

	// 生成 Markdown 格式的风格指导
	profile.RawMarkdown = buildStyleGuide(profile)
	return &profile, nil
}

// SaveProfile 保存风格档案。路径构造与目录创建收口在 project 层
// （Manager.WriteStyleProfileFile），本函数只做 JSON 编码；JSON 字节
// （MarshalIndent 两空格）、落点（.gaea/style-profile.json）与权限
// （0644）同收口前逐字节不变。
func SaveProfile(pm *project.Manager, profile *Profile) error {
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	return pm.WriteStyleProfileFile(data)
}

// LoadProfile 加载风格档案。兼容回退（.wubigork）已单源收口在 project 层
// （Manager.ReadStyleProfileFile，仅 os.IsNotExist 时回退），此处不再保留
// 第二份分支；读取与 JSON 解码错误均原样透传（不包装，与收口前一致）。
func LoadProfile(pm *project.Manager) (*Profile, error) {
	data, err := pm.ReadStyleProfileFile()
	if err != nil {
		return nil, err
	}
	var profile Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

// ToStyleGuide 生成可注入 prompt 的风格指导文本

// ToStyleGuide 生成可注入 prompt 的风格指导文本
func (p *Profile) ToStyleGuide() string {
	if p.RawMarkdown != "" {
		return p.RawMarkdown
	}
	return buildStyleGuide(*p)
}

func buildStyleGuide(p Profile) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## 写作风格: %s\n", p.Name))
	sb.WriteString(fmt.Sprintf("%s\n\n", p.Description))

	labels := map[string]string{
		"narrative_voice":     "叙事声音",
		"sentence_style":      "句式特点",
		"pacing":              "节奏感",
		"description_density": "描写密度",
		"dialogue_style":      "对话风格",
		"vocabulary_level":    "词汇层次",
		"emotional_tone":      "情感基调",
		"chinese_style":       "中文特色",
	}

	for key, label := range labels {
		if val, ok := p.Traits[key]; ok && val != "" {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", label, val))
		}
	}

	return sb.String()
}

// ── 导出/导入 ────────────────────────────────────────────────

// ExportProfile 导出风格档案为 Markdown
func (p *Profile) ExportProfile() string {
	return p.ToStyleGuide()
}

// ImportProfileFromMarkdown 从 Markdown 导入风格档案
func ImportProfileFromMarkdown(md string) *Profile {
	profile := &Profile{
		Name:        "导入风格",
		Description: "从 Markdown 导入",
		Traits:      map[string]string{"custom": md},
		RawMarkdown: md,
	}
	return profile
}
