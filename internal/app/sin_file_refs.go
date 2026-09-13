package app

// ── 原罪附件引用解析（@路径 → 本轮可读内容块）──
//
// Composer 提交时把附件统一注入为 `@绝对路径` 文本；办公线的 @引用由
// control.Controller.ResolveRefs 解析（含 MCP 资源/工作区面）。原罪按硬隔离
// 不接 Controller（sin 设计档 §8），本文件做**原罪域内的最小同构**：
//
//   - 只认「真实存在的本地文件」——正文里的 @字样（@角色、邮箱、提及）只要
//     不是存在的路径就原样保留，零误伤；
//   - 文本文件注入头部（64KB 封顶，与办公 maxFileRefBytes 同档）；二进制
//     （头 8KiB 含 NUL）只提示不倾倒；目录如实说明不注入；
//   - 图片走识图（vision.RecognizeImage，包级配置），失败回退占位——与办公
//     appendImageBlock 同语义；
//   - 引用块只进**本轮提示**，不落库：消息仍存 @路径 原文（与办公同口径，
//     历史不膨胀）。
//
// 实现独立于 control（控制面零耦合）；语义漂移靠两边各自的测试锚定。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gaea/gaea/internal/gaea/vision"
)

const (
	// sinFileRefMaxBytes 单个附件注入头部上限（与办公 maxFileRefBytes 同档：
	// "@somehuge.log" 不能撑爆上下文）。
	sinFileRefMaxBytes = 64 * 1024
	// sinFileRefMaxVisionBytes 识图文本注入上限。
	sinFileRefMaxVisionBytes = 1200
	// sinFileRefBinaryScan 二进制判定扫描窗口：头 8KiB 含 NUL 即按二进制处理。
	sinFileRefBinaryScan = 8 * 1024
)

// sinRefTokenRe @引用 token：@ + 非空白串（与办公 refTokenRe 同形）。
var sinRefTokenRe = regexp.MustCompile(`@([^\s]+)`)

// sinVisionRecognize 识图入口（可注入以便测试）。
var sinVisionRecognize = vision.RecognizeImage

// sinFileRefBlock 解析 line 里的 @文件引用，返回拼装好的内容块与逐条失败原因。
// 块为空 = 没有可解析的引用。 errs 非空的条目也已如实写入块内（模型看得见
// 失败原因），调用方无需重复上报。
func sinFileRefBlock(ctx context.Context, line string) (string, []string) {
	toks := sinRefTokens(line)
	if len(toks) == 0 {
		return "", nil
	}
	var (
		b    strings.Builder
		errs []string
	)
	for _, tok := range toks {
		path := filepath.FromSlash(tok)
		info, err := os.Stat(path)
		if err != nil {
			continue // 不存在的路径不是引用（@提及/正文 @字样零误伤，与办公同口径）
		}
		if info.IsDir() {
			errs = append(errs, "@"+tok+" — 目录不注入附件内容（请引用具体文件）")
			sinAppendRefBlock(&b, "dir", `path="`+path+`"`, "目录不注入内容。")
			continue
		}
		if sinIsImagePath(path) {
			sinAppendImageBlock(ctx, &b, path, tok)
			continue
		}
		content, err := sinReadFileRef(path)
		if err != nil {
			errs = append(errs, "@"+tok+" — "+err.Error())
			sinAppendRefBlock(&b, "file", `path="`+path+`"`, "[读取失败："+err.Error()+"]")
			continue
		}
		sinAppendRefBlock(&b, "file", `path="`+path+`"`, content)
	}
	return b.String(), errs
}

// sinRefTokens 提取去重、去尾标的 @token（与办公 parseRefTokens 同语义）。
func sinRefTokens(line string) []string {
	var toks []string
	seen := map[string]bool{}
	for _, g := range sinRefTokenRe.FindAllStringSubmatch(line, -1) {
		t := strings.TrimRight(g[1], ".,;!?)]}")
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		toks = append(toks, t)
	}
	return toks
}

// sinReadFileRef 读附件头部：截断封顶 + 二进制判定（头 8KiB 含 NUL）。
func sinReadFileRef(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	buf := make([]byte, sinFileRefMaxBytes+1)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return "", err
	}
	content := string(buf[:n])
	head := buf[:n]
	if len(head) > sinFileRefBinaryScan {
		head = head[:sinFileRefBinaryScan]
	}
	if strings.IndexByte(string(head), 0) >= 0 {
		return "[二进制文件，不注入内容]", nil
	}
	if n > sinFileRefMaxBytes {
		content = content[:sinFileRefMaxBytes] + "\n\n[已截断：仅注入前 64KB]"
	}
	return content, nil
}

// sinIsImagePath 按扩展名判断图片文件（与办公 isImagePath 同表）。
func sinIsImagePath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".tiff", ".tif":
		return true
	}
	return false
}

// sinAppendImageBlock 识图注入；失败回退占位（图片引用始终可进上下文）。
func sinAppendImageBlock(ctx context.Context, b *strings.Builder, path, raw string) {
	desc, err := sinVisionRecognize(ctx, path, "")
	switch {
	case err != nil:
		sinAppendRefBlock(b, "image", `path="`+path+`"`, "[图片附件识图失败："+err.Error()+"]")
		return
	case strings.TrimSpace(desc) == "":
		sinAppendRefBlock(b, "image", `path="`+path+`"`, "[图片附件 @"+raw+" 已就位；如需视觉理解请用识图工具]")
		return
	}
	if len(desc) > sinFileRefMaxVisionBytes {
		desc = desc[:sinFileRefMaxVisionBytes] + "…[已截断]"
	}
	sinAppendRefBlock(b, "image", `path="`+path+`"`, "【图片识别】\n"+desc)
}

// sinAppendRefBlock 块拼装（与办公 appendRefBlock 同形：<tag attr>\nbody\n</tag>）。
func sinAppendRefBlock(b *strings.Builder, tag, attr, body string) {
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	fmt.Fprintf(b, "<%s %s>\n%s\n</%s>", tag, attr, body, tag)
}
