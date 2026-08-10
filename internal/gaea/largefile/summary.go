// Package largefile implements map-reduce summarization for large documents:
// extract text (office docs via docmd, plain text directly), chunk it,
// summarize each chunk with the configured provider, then merge chunk
// summaries into one document-level summary ("摘要的摘要" when the merged
// output is still large). This mirrors the industry pattern behind 千问
// 500 页超长文 / WPS 读完整本书 / 豆包分段摘要 / M365 摘要指南.
package largefile

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/office/docmd"
)

const (
	// DefaultChunkRunes is the per-chunk summarization budget (~20k chars),
	// sized so one provider call fits comfortably in context.
	DefaultChunkRunes = 20000
	// MergeThresholdRunes: when merged chunk summaries exceed this, run one
	// final "摘要的摘要" pass over the merged text.
	MergeThresholdRunes = 20000
	// MaxExtractBytes guards against reading absurdly large raw text files.
	MaxExtractBytes = 100 << 20 // 100MB
	// summarizeChunkTimeout bounds each provider call.
	summarizeChunkTimeout = 90 * time.Second
)

// Options controls one summarization run.
type Options struct {
	// ChunkRunes is the per-chunk budget; <=0 falls back to DefaultChunkRunes.
	ChunkRunes int
	// Focus narrows what the summary should emphasize
	// (e.g. "关键数据与表格清单" or "结论与建议").
	Focus string
}

// Result is the outcome of SummarizeFile.
type Result struct {
	Path       string
	TotalPages int
	Chars      int
	Chunks     int
	Summary    string
}

// MultiResult is the outcome of SummarizeFiles.
type MultiResult struct {
	Paths   []string
	Files   int
	Chunks  int
	Summary string
}

// ExtractText pulls a file's text for summarization: office documents go
// through docmd (PDF capped at maxPDFPages, 0 = no cap); everything else is
// read as plain text with a 100MB guard.
func ExtractText(path string, maxPDFPages int) (string, int, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".docx", ".doc", ".xls", ".xlsx", ".pdf":
		md, total, _, err := docmd.ConvertLimit(path, "", maxPDFPages)
		if err != nil {
			return "", 0, err
		}
		return md, total, nil
	default:
		info, err := os.Stat(path)
		if err != nil {
			return "", 0, err
		}
		if info.Size() > MaxExtractBytes {
			return "", 0, fmt.Errorf("文件过大（%.1f MB），超过提取上限 100MB", float64(info.Size())/(1<<20))
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return "", 0, err
		}
		return string(b), 0, nil
	}
}

// ChunkText splits text into paragraph-aware chunks of at most chunkRunes
// runes; a single oversized paragraph is hard-split. Returns nil for empty
// input.
func ChunkText(text string, chunkRunes int) []string {
	if chunkRunes <= 0 {
		chunkRunes = DefaultChunkRunes
	}
	if strings.TrimSpace(text) == "" {
		return nil
	}
	units := strings.Split(text, "\n\n")
	var chunks []string
	var b strings.Builder
	flush := func() {
		if strings.TrimSpace(b.String()) == "" {
			return
		}
		chunks = append(chunks, b.String())
		b.Reset()
	}
	for _, u := range units {
		if runeLen(u) > chunkRunes {
			flush()
			for _, part := range hardSplit(u, chunkRunes) {
				chunks = append(chunks, part)
			}
			continue
		}
		if b.Len() > 0 && runeLen(b.String())+runeLen(u) > chunkRunes {
			flush()
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(u)
	}
	flush()
	return chunks
}

func runeLen(s string) int { return len([]rune(s)) }

func hardSplit(s string, n int) []string {
	rs := []rune(s)
	var out []string
	for i := 0; i < len(rs); i += n {
		end := i + n
		if end > len(rs) {
			end = len(rs)
		}
		out = append(out, string(rs[i:end]))
	}
	return out
}

// Summarize runs the map-reduce pipeline: chunk → per-chunk summary → merge
// (with one final pass when merged output exceeds MergeThresholdRunes).
// Returns the summary and the number of chunks processed.
func Summarize(ctx context.Context, prov provider.Provider, text string, opts Options) (summary string, chunkCount int, err error) {
	chunks := ChunkText(text, opts.ChunkRunes)
	if len(chunks) == 0 {
		return "", 0, fmt.Errorf("没有可摘要的内容")
	}
	if prov == nil {
		return "", 0, fmt.Errorf("没有可用的模型服务")
	}
	summaries := make([]string, 0, len(chunks))
	for i, c := range chunks {
		s, serr := summarizeChunk(ctx, prov, c, i+1, len(chunks), opts.Focus)
		if serr != nil {
			return "", 0, fmt.Errorf("第 %d/%d 块摘要失败: %w", i+1, len(chunks), serr)
		}
		summaries = append(summaries, s)
	}
	merged := strings.Join(summaries, "\n\n")
	if len(chunks) > 1 && runeLen(merged) > MergeThresholdRunes {
		final, ferr := summarizeChunk(ctx, prov, merged, 1, 1, opts.Focus+"（合并总览）")
		if ferr != nil {
			return "", 0, fmt.Errorf("合并摘要失败: %w", ferr)
		}
		return final, len(chunks), nil
	}
	return merged, len(chunks), nil
}

// SummarizeFile extracts and summarizes a file in one call.
func SummarizeFile(ctx context.Context, prov provider.Provider, path string, opts Options) (*Result, error) {
	text, total, err := ExtractText(path, docmd.DefaultMaxPDFPages)
	if err != nil {
		return nil, err
	}
	summary, chunks, err := Summarize(ctx, prov, text, opts)
	if err != nil {
		return nil, err
	}
	return &Result{
		Path:       path,
		TotalPages: total,
		Chars:      runeLen(text),
		Chunks:     chunks,
		Summary:    summary,
	}, nil
}

// SummarizeFiles summarizes several files (each with its own map-reduce pass),
// then merges the per-file summaries with one final "摘要的摘要" pass.
// Mirrors ChatGPT's official guidance for very long documents: summarize each
// file first, then summarize the summaries.
func SummarizeFiles(ctx context.Context, prov provider.Provider, paths []string, opts Options) (*MultiResult, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("没有可摘要的文件")
	}
	if prov == nil {
		return nil, fmt.Errorf("没有可用的模型服务")
	}
	perFile := make([]string, 0, len(paths))
	totalChunks := 0
	for i, p := range paths {
		res, err := SummarizeFile(ctx, prov, p, opts)
		if err != nil {
			return nil, fmt.Errorf("第 %d/%d 个文件 %s 摘要失败: %w", i+1, len(paths), p, err)
		}
		totalChunks += res.Chunks
		suffix := ""
		if res.TotalPages > 0 {
			suffix = fmt.Sprintf("（PDF 共 %d 页）", res.TotalPages)
		}
		perFile = append(perFile, fmt.Sprintf("## %s%s\n%s", filepath.Base(p), suffix, res.Summary))
	}
	merged := strings.Join(perFile, "\n\n")
	final, err := summarizeChunk(ctx, prov, merged, 1, 1, opts.Focus+"（多文件合并总览）")
	if err != nil {
		return nil, fmt.Errorf("多文件合并摘要失败: %w", err)
	}
	return &MultiResult{
		Paths:   append([]string(nil), paths...),
		Files:   len(paths),
		Chunks:  totalChunks,
		Summary: final,
	}, nil
}

// summarizeChunk makes one bounded provider call for a chunk.
func summarizeChunk(ctx context.Context, prov provider.Provider, text string, index, total int, focus string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, summarizeChunkTimeout)
	defer cancel()

	const sysPrompt = "你是文档摘要助手。请用中文输出该文档片段的精炼摘要，保留：主题、章节/要点列表、关键数据与结论、表格与清单（如存在）。只依据给定内容，不要添加原文没有的信息。输出为 Markdown。"
	user := fmt.Sprintf("请摘要以下文档片段（第 %d/%d 块）：", index, total)
	if focus != "" {
		user += " 重点关注：" + focus
	}
	user += "\n\n" + text

	ch, err := prov.Stream(ctx, provider.Request{
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: sysPrompt},
			{Role: provider.RoleUser, Content: user},
		},
		Temperature: 0,
	})
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case chunk, ok := <-ch:
			if !ok {
				s := strings.TrimSpace(out.String())
				if s == "" {
					return "", fmt.Errorf("summarizer returned empty output")
				}
				return s, nil
			}
			switch chunk.Type {
			case provider.ChunkText:
				out.WriteString(chunk.Text)
			case provider.ChunkError:
				return "", chunk.Err
			}
		}
	}
}
