package app

// ── 原罪工具：图文导出（sin_export）──
//
// 导出链不复制第二份：正文与插图的渲染直接复用 SinExportMarkdown（与前端
// 「另存为 .md」同一份 Markdown：正文原样、插图标记换成 ![](本地路径)），
// 本工具只多做一件事——把结果落进原罪自有目录，让模型能一口气完成
// 「写到这里，顺手导出一份」。
//
// 落点：<用户配置目录>/gaea/sin/exports/<名字>.md（与原罪 art/notes 同一数据
// 根，不写办公工作区）。同名不覆盖：自动加 -2/-3 后缀，用户的历史导出不会被
// 这一次悄悄顶掉。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

const (
	// sinExportNameMaxRunes 用户给定文件名的长度上限（rune）。
	sinExportNameMaxRunes = 60
)

func init() {
	registerSinTool(sinToolExport, func(c sinToolContext) sinTool {
		return sinExportTool{app: c.app, topicID: c.topicID}
	})
}

// sinExportsDir 原罪导出目录（与 art/notes 同一数据根）。
func sinExportsDir() string {
	return filepath.Join(sinRoot(), "exports")
}

// sinExportTool 图文导出工具。
type sinExportTool struct {
	app     *App
	topicID string
}

func (sinExportTool) Name() string { return sinToolExport }

func (sinExportTool) Description() string {
	return "把当前故事导出成一份图文 Markdown 文件（正文 + 已生成的插图，插图为本地图片引用），" +
		"落在本机原罪的导出目录里，返回文件路径。只在用户明确要「导出」「保存成文件」时用；" +
		"普通写作不要顺手导出。导出的是此刻已落库的全部内容（含用户与你的每一轮）；" +
		"注意：你当前回合正在写的正文要回合结束才落库——若用户要导出「刚写的内容」，" +
		"本回合只管写作，下一回合再调用本工具。"
}

func (sinExportTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "filename":{"type":"string","description":"可选。文件名（不含路径与 .md 扩展名），例如「唐末浮生-第一章」；缺省用故事 id + 时间戳。"}
}}`)
}

// ReadOnly=false：会在导出目录里新建文件（前端按写类工具标注）。
func (sinExportTool) ReadOnly() bool { return false }

func (t sinExportTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Filename string `json:"filename"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("参数解析失败: %w", err)
		}
	}
	if t.app == nil {
		return "", fmt.Errorf("导出工具未接入应用（app 为空）")
	}

	md, err := t.app.SinExportMarkdown(t.topicID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(md) == "" {
		return "", fmt.Errorf("故事还没有内容可导出")
	}

	name, err := sinExportFileName(p.Filename, t.topicID, time.Now())
	if err != nil {
		return "", err
	}
	dir := sinExportsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建原罪导出目录失败: %w", err)
	}
	path, err := sinExportUniquePath(dir, name, "md")
	if err != nil {
		return "", err
	}
	if err := sinWriteFileAtomic(path, []byte(md)); err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "已导出图文 Markdown：%s\n", path)
	fmt.Fprintf(&b, "共 %d 字（插图以本地图片路径内嵌）。", len([]rune(md)))
	return b.String(), nil
}

// sinExportFileName 计算导出文件名（不含扩展名）。用户给了名字就净化成一个
// 纯文件名（Windows 保留字符/路径分隔/上跳一律剔除，fail-closed），没给就用
// 故事 id + 本地时间戳。
func sinExportFileName(raw, topicID string, now time.Time) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		id := strings.TrimSpace(topicID)
		if !sinNotesIDOK(id) {
			id = "sin"
		}
		return fmt.Sprintf("%s-%s", id, now.Format("20060102-150405")), nil
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' ||
			r == '<' || r == '>' || r == '|':
			// 路径分隔与 Windows 保留字符：直接剔除，不转义成看似合法的别的名字。
		case unicode.IsControl(r):
		default:
			b.WriteRune(r)
		}
	}
	cleaned := strings.Trim(strings.TrimSpace(b.String()), ". ")
	// 去扩展名（用户可能已经写了 .md，别导出成 xxx.md.md）
	cleaned = strings.TrimSuffix(cleaned, ".md")
	cleaned = strings.Trim(strings.TrimSpace(cleaned), ". ")
	if cleaned == "" {
		return "", fmt.Errorf("文件名不合法（净化后为空），换一个名字或留空用默认名")
	}
	return truncateRunes(cleaned, sinExportNameMaxRunes), nil
}

// sinExportUniquePath 同名不覆盖：base.ext → base-2.ext → base-3.ext（上限 99）。
func sinExportUniquePath(dir, base, ext string) (string, error) {
	first := filepath.Join(dir, base+"."+ext)
	if _, err := os.Stat(first); os.IsNotExist(err) {
		return first, nil
	}
	for i := 2; i <= 99; i++ {
		p := filepath.Join(dir, fmt.Sprintf("%s-%d.%s", base, i, ext))
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return p, nil
		}
	}
	return "", fmt.Errorf("导出目录里同名文件太多（%s-2 ~ %s-99 已存在），换一个名字", base, base)
}

// sinWriteFileAtomic 原子写文件（临时文件 + rename，与原罪便签同一纪律）。
func sinWriteFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("写入导出文件失败: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("保存导出文件失败: %w", err)
	}
	return nil
}
