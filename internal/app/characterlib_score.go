package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/gaea/vision"
	"github.com/gaea/gaea/internal/util"
)

// characterScoreVision 视觉评分 seam（测试替换；生产恒为 visionScoreImage）。
var characterScoreVision = visionScoreImage

// scoreImageToLocalPath 把评分输入规整为本地文件路径：本地路径原样返回；
// data URL 解码落临时文件（调用方负责 remove 返回的清理函数）。
func scoreImageToLocalPath(image string) (path string, cleanup func(), err error) {
	p := strings.TrimSpace(image)
	if !strings.HasPrefix(p, "data:image/") {
		return p, func() {}, nil
	}
	idx := strings.Index(p, ",")
	if idx < 0 {
		return "", func() {}, fmt.Errorf("评分图片 data URL 格式非法（缺逗号）")
	}
	raw, err := base64.StdEncoding.DecodeString(p[idx+1:])
	if err != nil {
		return "", func() {}, fmt.Errorf("评分图片 base64 解码失败: %w", err)
	}
	tmp, err := os.CreateTemp("", "gaea-score-*.png")
	if err != nil {
		return "", func() {}, fmt.Errorf("评分图片临时文件创建失败: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup = func() { os.Remove(tmpPath) }
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		cleanup()
		return "", func() {}, fmt.Errorf("评分图片临时文件写入失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("评分图片临时文件关闭失败: %w", err)
	}
	return tmpPath, cleanup, nil
}

// visionScoreImage 对单张图发起视觉问答（data URL 先落临时文件走同一链路）。
func visionScoreImage(ctx context.Context, image, prompt string) (string, error) {
	p, cleanup, err := scoreImageToLocalPath(image)
	if err != nil {
		return "", err
	}
	defer cleanup()
	return vision.RecognizeImage(ctx, p, prompt)
}

// consistencyScoreAnchorRunes 评分提示词里外观锚点的最大长度（与设定卡同预算口径）。
const consistencyScoreAnchorRunes = 160

// buildConsistencyScorePrompt 一致性评分提示词（纯函数）：文字锚点方案 v1——
// 让视觉模型拿生成图对照角色文字设定（Appearance/Figure）打分，输出严格 JSON。
func buildConsistencyScorePrompt(c characterlib.Character) string {
	var b strings.Builder
	b.WriteString("你是角色形象一致性评审。请评估图片中的人物形象")
	anchor := strings.TrimSpace(c.Appearance)
	if figure := strings.TrimSpace(c.Figure); figure != "" {
		if anchor != "" {
			anchor += "；"
		}
		anchor += figure
	}
	if anchor != "" {
		b.WriteString("与以下文字设定的一致性。文字设定：")
		b.WriteString(truncateRunes(anchor, consistencyScoreAnchorRunes))
		b.WriteString("。")
	} else {
		b.WriteString("。")
	}
	b.WriteString("评审维度：面部特征、发型发色、体型、年龄感（服装仅当文字设定有描述时参与）。")
	b.WriteString("只输出 JSON，不要多余文字：{\"score\": 0到100的整数, \"summary\": \"一句话总评\", \"issues\": [\"不一致点，最多3条\"]}。")
	b.WriteString("评分标准：90以上高度一致；70到89基本一致但有细节出入；50到69明显偏差；低于50几乎不是同一人。")
	return b.String()
}

// consistencyScoreResult 评分结果的规范化形状（返回给前端的 JSON）。
type consistencyScoreResult struct {
	Score   int      `json:"score"`
	Summary string   `json:"summary"`
	Issues  []string `json:"issues"`
}

// parseConsistencyScoreReply 解析视觉模型回复：提取 JSON→钳位 score（0-100，
// 非法视为解析失败）→规范化 issues（字符串数组、非空项、最多 5 条防御性截断）。
func parseConsistencyScoreReply(reply string) (*consistencyScoreResult, error) {
	var raw struct {
		Score   interface{}   `json:"score"`
		Summary string        `json:"summary"`
		Issues  []interface{} `json:"issues"`
	}
	if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &raw); err != nil {
		return nil, fmt.Errorf("评分返回无法解析为 JSON")
	}
	score, ok := numOf(raw.Score)
	if !ok {
		return nil, fmt.Errorf("评分返回缺少合法的 score 数值")
	}
	s := int(score + 0.5)
	if s < 0 {
		s = 0
	}
	if s > 100 {
		s = 100
	}
	out := &consistencyScoreResult{Score: s, Summary: strings.TrimSpace(raw.Summary)}
	for _, it := range raw.Issues {
		if str, ok := it.(string); ok {
			if v := strings.TrimSpace(str); v != "" {
				out.Issues = append(out.Issues, v)
			}
		}
		if len(out.Issues) >= 5 {
			break
		}
	}
	return out, nil
}

// CharacterScoreConsistency 角色形象一致性评分（v4.404，T2 一致性 v1）：
// 文字锚点方案——视觉模型拿 image（本地路径或 data URL）对照角色文字设定
// （Appearance/Figure）打分，返回 {"score":0-100,"summary":"…","issues":[…]}
// 的 JSON 字符串。score<60 前端提示补参考。视觉模型不可用/回复不可解析如实报错。
func (a *App) CharacterScoreConsistency(chJSON, image string) (string, error) {
	var c characterlib.Character
	if err := json.Unmarshal([]byte(chJSON), &c); err != nil {
		return "", fmt.Errorf("解析角色数据失败: %w", err)
	}
	if strings.TrimSpace(c.Name) == "" {
		return "", fmt.Errorf("角色名称不能为空")
	}
	img := strings.TrimSpace(image)
	if img == "" {
		return "", fmt.Errorf("缺少待评分图片（本地路径或 data URL）")
	}
	if strings.HasPrefix(img, "http://") || strings.HasPrefix(img, "https://") {
		return "", fmt.Errorf("远端 URL 不支持评分，请使用本地路径或 data URL")
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reply, err := characterScoreVision(ctx, img, buildConsistencyScorePrompt(c))
	if err != nil {
		return "", fmt.Errorf("一致性评分失败: %w", err)
	}
	res, err := parseConsistencyScoreReply(reply)
	if err != nil {
		return "", fmt.Errorf("%w（原始返回：%.200s）", err, reply)
	}
	out, err := json.Marshal(res)
	if err != nil {
		return "", fmt.Errorf("评分结果序列化失败: %w", err)
	}
	return string(out), nil
}
