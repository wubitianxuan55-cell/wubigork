package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gaea/gaea/internal/gaea/proposal"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// ── 方案库维护三工具（2026-11-24）────────────────────────────────────
//   - proposal_update：改方案元数据（标题/类型/年度/摘要/工作区），或增量修正某一小节
//     （标题/正文）——录错一个字不必整册重录。
//   - proposal_delete：删除录错的方案。
//   - proposal_export：把库内方案导回 Markdown 文件（托管图片落成文件）。
// update / delete 改动持久化数据，与 proposal_save 同走 PersistWrite（hardAsk 审批红线）。

func init() {
	tool.RegisterBuiltin(proposalUpdateTool{})
	tool.RegisterBuiltin(proposalDeleteTool{})
	tool.RegisterBuiltin(proposalExportTool{})
}

// proposalResolve 取方案；不存在时给带清单指引的错误。
func proposalResolve(st *proposal.Store, name string) (*proposal.Proposal, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("proposal（方案标题）为必填项")
	}
	p, err := st.Resolve(name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", proposal.ErrStoreUnreadable, err)
	}
	if p == nil {
		return nil, fmt.Errorf("方案库里没有《%s》。不填参数调用 proposal_search 可看方案清单", name)
	}
	return p, nil
}

// ── proposal_update：改元数据 / 增量修正小节 ─────────────────────────

type proposalUpdateTool struct{ workDir string }

func (proposalUpdateTool) Name() string { return "proposal_update" }
func (proposalUpdateTool) Description() string {
	return "修改方案库里的一份方案，不必整册重录。两类改动可同时给：①元数据 title（改名）/project_type/year/summary/workspace；②增量修正某一小节 section_id + section_title（标题路径与子节路径随之更新）和/或 body（整节替换正文）。section_id 来自 proposal_search 命中或标题树。只在你明确要改的字段；只改元数据时不要给 section_id。"
}
func (proposalUpdateTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "proposal":{"type":"string","description":"要改的方案（标题）"},
  "title":{"type":"string","description":"改方案标题（方案名随之改）"},
  "project_type":{"type":"string","description":"项目类型"},
  "year":{"type":"string","description":"年度"},
  "summary":{"type":"string","description":"一段话摘要"},
  "workspace":{"type":"string","description":"来源工作区路径（归档信息，不参与分组与检索）"},
  "section_id":{"type":"integer","description":"只修正这一小节（增量，不必整册重录）"},
  "section_title":{"type":"string","description":"小节的新标题；它的子节标题路径一并更新"},
  "body":{"type":"string","description":"整节替换的新正文（Markdown）；相对图片按当前工作区解析并托管"}
},
"required":["proposal"]}`)
}
func (proposalUpdateTool) ReadOnly() bool     { return false }
func (proposalUpdateTool) PersistWrite() bool { return true }
func (proposalUpdateTool) CompactDescription() string {
	return "改方案库方案(改标题/类型/年度/摘要;或section_id+section_title/body增量修正某一节,不必重录)"
}
func (proposalUpdateTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"proposal":{"type":"string"},"title":{"type":"string"},"project_type":{"type":"string"},"year":{"type":"string"},"summary":{"type":"string"},"workspace":{"type":"string"},"section_id":{"type":"integer"},"section_title":{"type":"string"},"body":{"type":"string"}},"required":["proposal"]}`)
}

func (t proposalUpdateTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Proposal     string  `json:"proposal"`
		Title        string  `json:"title"`
		ProjectType  string  `json:"project_type"`
		Year         metaStr `json:"year"`
		Summary      string  `json:"summary"`
		Workspace    string  `json:"workspace"`
		SectionID    int64   `json:"section_id"`
		SectionTitle *string `json:"section_title"`
		Body         *string `json:"body"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	st, err := proposalStore()
	if err != nil {
		return "", err
	}
	prop, err := proposalResolve(st, p.Proposal)
	if err != nil {
		return "", err
	}
	meta := proposal.UpdateInput{
		ProjectType: optString(p.ProjectType), Year: optString(string(p.Year)),
		Summary: optString(p.Summary), Workspace: optString(p.Workspace),
	}
	hasMeta := meta.ProjectType != nil || meta.Year != nil || meta.Summary != nil || meta.Workspace != nil
	if strings.TrimSpace(p.Title) != "" && strings.TrimSpace(p.Title) != prop.Title {
		meta.Title = optString(p.Title)
		hasMeta = true
	}
	edit := proposal.SectionEdit{AssetBaseDir: t.workDir}
	if p.SectionID > 0 {
		if p.SectionTitle != nil && strings.TrimSpace(*p.SectionTitle) != "" {
			edit.Title = p.SectionTitle
		}
		if p.Body != nil {
			edit.Body = p.Body
		}
		if edit.Title == nil && edit.Body == nil {
			return "", errors.New("给了 section_id 就要给 section_title 或 body（至少一项）")
		}
	}
	if !hasMeta && edit.Title == nil && edit.Body == nil {
		return "", errors.New("没有要改的内容：元数据字段（title/project_type/year/summary/workspace）或 section_id + section_title/body 至少给一项")
	}

	var b strings.Builder
	if edit.Title != nil || edit.Body != nil {
		var old proposal.Section
		if secs, err := st.Outline(prop.ID); err == nil {
			for _, s := range secs {
				if s.ID == p.SectionID {
					old = s
					break
				}
			}
		}
		affected, err := st.UpdateSection(p.SectionID, edit)
		if err != nil {
			return "", err
		}
		cur := affected[0]
		b.WriteString("✅ 已修正方案库小节\n\n")
		fmt.Fprintf(&b, "**方案**: %s\n**小节**: %d 级", prop.Title, cur.Level)
		if edit.Title != nil {
			fmt.Fprintf(&b, "，「%s」→「%s」", old.Title, cur.Title)
		} else {
			fmt.Fprintf(&b, "「%s」", cur.Title)
		}
		if edit.Body != nil {
			fmt.Fprintf(&b, "；正文已整节替换（%s）", fmtChars(len([]rune(*edit.Body))))
		}
		b.WriteString("\n")
		fmt.Fprintf(&b, "**路径**: %s\n", cur.TitlePath)
		if n := len(affected) - 1; n > 0 && edit.Title != nil {
			fmt.Fprintf(&b, "**连带**: %d 个子节的标题路径已同步\n", n)
		}
		fmt.Fprintf(&b, "**语义索引**: %s\n\n", proposalIndexStatus(ctx, st, prop.ID))
	}
	if hasMeta {
		updated, err := st.UpdateMeta(p.Proposal, meta)
		if err != nil {
			return "", err
		}
		b.WriteString("✅ 已修改方案信息\n\n")
		fmt.Fprintf(&b, "**标题**: %s", updated.Title)
		if updated.Title != prop.Title {
			fmt.Fprintf(&b, "（原名「%s」）", prop.Title)
		}
		b.WriteString("\n")
		fmt.Fprintf(&b, "**类型/年度**: %s / %s\n", orDash(updated.ProjectType), orDash(updated.Year))
		fmt.Fprintf(&b, "**落点**: 工作区 %s\n", orDash(updated.Workspace))
	}
	return tool.WrapText(b.String()), nil
}

func optString(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// ── proposal_delete：删除方案 ────────────────────────────────────────

type proposalDeleteTool struct{}

func (proposalDeleteTool) Name() string { return "proposal_delete" }
func (proposalDeleteTool) Description() string {
	return "从方案库删除一份方案（标题节点、语义索引、全文索引一并清除，不可恢复；原文件不受影响，需要时可重新录入）。只在用户明确要求删除时调用，并在调用前复述要删的是哪一份。"
}
func (proposalDeleteTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "proposal":{"type":"string","description":"要删除的方案（标题）"},
  "reason":{"type":"string","description":"删除理由（进审批摘要）"}
},
"required":["proposal"]}`)
}
func (proposalDeleteTool) ReadOnly() bool     { return false }
func (proposalDeleteTool) PersistWrite() bool { return true }
func (proposalDeleteTool) CompactDescription() string {
	return "删方案库方案(节点/语义索引一并清除,不可恢复;仅用户明确要求时)"
}
func (proposalDeleteTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"proposal":{"type":"string"},"reason":{"type":"string"}},"required":["proposal"]}`)
}

func (proposalDeleteTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Proposal string `json:"proposal"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	st, err := proposalStore()
	if err != nil {
		return "", err
	}
	prop, err := proposalResolve(st, p.Proposal)
	if err != nil {
		return "", err
	}
	secs, err := st.Outline(prop.ID)
	if err != nil {
		return "", err
	}
	_, chunks := st.PendingCount(prop.ID)
	if err := st.Delete(prop.Name); err != nil {
		return "", err
	}
	rest := 0
	if list, err := st.List(); err == nil {
		rest = len(list)
	}
	return tool.WrapText(fmt.Sprintf("🗑 已从方案库删除《%s》：%d 个标题节点、%d 个语义块一并清除。方案库现有 %d 份。原文件未动，需要时可在方案模式里重新录入。\n",
		prop.Title, len(secs), chunks, rest)), nil
}

// ── proposal_export：导回 Markdown ──────────────────────────────────

type proposalExportTool struct{ workDir string }

var proposalAssetRefRe = regexp.MustCompile(regexp.QuoteMeta(proposal.AssetURIPrefix) + `([0-9a-fA-F]{64})`)

func (proposalExportTool) Name() string { return "proposal_export" }
func (proposalExportTool) Description() string {
	return "把方案库里的一份方案导回 Markdown 文件：按标题层级写成 # ~ ####，正文原样；正文里的托管图片落成文件，存到输出文件同名的 -assets 目录并改为相对引用。output 为 .md 文件路径。"
}
func (proposalExportTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "proposal":{"type":"string","description":"要导出的方案（标题）"},
  "output":{"type":"string","description":"导出的 Markdown 文件路径（.md）"}
},
"required":["proposal","output"]}`)
}
func (proposalExportTool) ReadOnly() bool { return false }
func (t proposalExportTool) WritePaths(args json.RawMessage) ([]string, error) {
	var p struct {
		Output string `json:"output"`
	}
	if err := json.Unmarshal(args, &p); err != nil || strings.TrimSpace(p.Output) == "" {
		return nil, err
	}
	out := resolveIn(t.workDir, p.Output)
	return []string{out, strings.TrimSuffix(out, filepath.Ext(out)) + "-assets"}, nil
}
func (proposalExportTool) CompactDescription() string {
	return "导出方案库方案为Markdown(按标题层级;托管图片落到同名-assets目录)"
}
func (proposalExportTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"proposal":{"type":"string"},"output":{"type":"string"}},"required":["proposal","output"]}`)
}

func (t proposalExportTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Proposal string `json:"proposal"`
		Output   string `json:"output"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if strings.TrimSpace(p.Output) == "" {
		return "", errors.New("output（导出的 .md 文件路径）为必填项")
	}
	out := resolveIn(t.workDir, p.Output)
	switch strings.ToLower(filepath.Ext(out)) {
	case ".md", ".markdown", ".txt":
	default:
		return "", fmt.Errorf("output 只收 .md/.markdown/.txt，收到 %s", filepath.Ext(out))
	}
	st, err := proposalStore()
	if err != nil {
		return "", err
	}
	target, err := proposalResolve(st, p.Proposal)
	if err != nil {
		return "", err
	}
	prop, secs, err := st.FullText(target.Name)
	if err != nil {
		return "", fmt.Errorf("%s: %v", proposal.ErrStoreUnreadable, err)
	}

	// 图片：托管目录 → 输出文件同名的 -assets 目录；文件缺失或写不出来的引用保留原样并计数。
	stem := strings.TrimSuffix(filepath.Base(out), filepath.Ext(out))
	assetDir := filepath.Join(filepath.Dir(out), stem+"-assets")
	relDir := stem + "-assets"
	images, missing := 0, 0
	var b strings.Builder
	fmt.Fprintf(&b, "> 方案库导出：%s（%s / %s）\n\n", prop.Title, orDash(prop.ProjectType), orDash(prop.Year))
	for _, s := range secs {
		fmt.Fprintf(&b, "%s %s\n\n", strings.Repeat("#", s.Level), s.Title)
		body := proposalAssetRefRe.ReplaceAllStringFunc(s.Body, func(ref string) string {
			src, err := st.AssetPath(ref)
			if err != nil {
				missing++
				return ref
			}
			raw, err := os.ReadFile(src)
			if err == nil {
				err = os.MkdirAll(assetDir, 0o755)
			}
			if err == nil {
				err = os.WriteFile(filepath.Join(assetDir, filepath.Base(src)), raw, 0o644)
			}
			if err != nil {
				missing++
				return ref
			}
			images++
			return relDir + "/" + filepath.Base(src)
		})
		if strings.TrimSpace(body) != "" {
			b.WriteString(strings.TrimRight(body, "\n") + "\n\n")
		}
	}
	if dir := filepath.Dir(out); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("创建输出目录失败: %w", err)
		}
	}
	if err := os.WriteFile(out, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("写入导出文件失败: %w", err)
	}
	total := 0
	for _, s := range secs {
		total += s.Chars
	}
	detail := fmt.Sprintf("已导出《%s》到 %s：%d 个标题节点、%s", prop.Title, out, len(secs), fmtChars(total))
	if images > 0 {
		detail += fmt.Sprintf("；图片 %d 张存到 %s", images, assetDir)
	}
	if missing > 0 {
		detail += fmt.Sprintf("；%d 处图片引用的托管文件缺失（保留了原引用）", missing)
	}
	return tool.WrapText(detail), nil
}
