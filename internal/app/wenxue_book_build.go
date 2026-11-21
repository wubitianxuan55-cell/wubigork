package app

// wenxue_book_build.go — 图文整本成册（2026-11-21 用户拍板「方案模式↔长篇小说
// 结构面移植全部」三刀合一）。办公 longdoc_build 的姊妹件，事实源口径不同：
// longdoc 是显式 manifest 文件，本件是**定稿目录（定稿/final）即唯一事实源**
// （磁盘即事实，与图文族哲学一致——不另建清单文件）。
//
//   - 组装前机检闸（0 硬伤才组装）：未编号/重号/空章/占位残留/插值不闭合=硬伤；
//     跳号/无一级标题/文件名与正文标题漂移=warn（卷结构与兜底渲染是合法形态）。
//   - 配额面：target_words_total vs 全书实测（容差百分比，超差=warn 不拦）；
//     每章字数恒附结构化 JSON 行。
//   - 静态目录两遍法（docx+toc）：measureTocEntries 板间共用核心，渲染链缺席
//     如实回落 TOC 域并标注——与 longdoc_build 同款契约。

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/wspath"
)

func wenxueBookBuildTool(a *App) *wenxueTool {
	return &wenxueTool{
		name: "wenxue_book_build",
		desc: "把定稿目录（定稿/final）按章号序组装成整本交付物（docx/pdf/md）——发布/导出整本用一个动作，不逐章手拼。组装前强制机检：未编号/重号/空章/占位残留/插值不闭合=硬伤拒绝组装；跳号/无一级标题/文件名与正文标题漂移=warn 提示不拦。target_words_total 传全书目标字数即出配额偏差（超容差=warn）；每章字数恒附 JSON。docx+toc 自动静态目录两遍法（渲染实测页码，不留 F9 域；LibreOffice 缺席如实回落域目录并标注）。output_path 为必填声明写路径。",
		schema: `{"type":"object","required":["format","output_path"],"properties":{
"format":{"type":"string","enum":["docx","pdf","md"],"description":"交付格式"},
"output_path":{"type":"string","description":"声明写路径（工作区相对路径）；扩展名须与 format 一致"},
"title":{"type":"string","description":"书名（缺省=当前书目录名；封面/页眉/回执用）"},
"toc":{"type":"boolean","description":"docx 生成目录（自动静态页码两遍法）"},
"cover":{"type":"boolean","description":"生成封面"},
"template":{"type":"string","enum":["通用","公文","报告","合同","标书"],"description":"docx 模板，缺省 通用"},
"author":{"type":"string","description":"作者（封面署名/页眉缺省）"},
"subtitle":{"type":"string","description":"副题（封面）"},
"header":{"type":"string","description":"页眉"},
"footer":{"type":"string","description":"页脚，缺省 第 {page} 页"},
"target_words_total":{"type":"integer","description":"全书目标字数（大纲口径传入；配额面）"},
"tolerance_pct":{"type":"integer","description":"配额容差百分比，缺省 30"}
},"additionalProperties":false}`,
		readOnly: false,
		run: func(args json.RawMessage) (string, error) {
			var p struct {
				Format           string `json:"format"`
				OutputPath       string `json:"output_path"`
				Title            string `json:"title"`
				TOC              bool   `json:"toc"`
				Cover            bool   `json:"cover"`
				Template         string `json:"template"`
				Author           string `json:"author"`
				Subtitle         string `json:"subtitle"`
				Header           string `json:"header"`
				Footer           string `json:"footer"`
				TargetWordsTotal int    `json:"target_words_total"`
				TolerancePct     int    `json:"tolerance_pct"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return "", fmt.Errorf("wenxue_book_build 参数解析失败: %w", err)
			}
			if strings.TrimSpace(p.OutputPath) == "" {
				return "", fmt.Errorf("缺少 output_path（声明写路径）：整本交付必须声明落点")
			}
			format := strings.ToLower(strings.TrimPrefix(p.Format, "."))
			switch format {
			case "docx", "pdf", "md":
			default:
				return "", fmt.Errorf("不支持的格式 %q（docx/pdf/md）", p.Format)
			}
			root := wenxueWorkspace()
			chapters := wenxueListFinalChapters(root)
			if len(chapters) == 0 {
				return "", fmt.Errorf("定稿目录为空（定稿/final）——先把章节定稿（wenxue_final_write），成册只收发布稿不收草稿")
			}

			// ── 机检闸：0 硬伤才组装 ─────────────────────────────────────
			var hards, warns []string
			seenNo := map[int]string{}
			for _, c := range chapters {
				if c.Number == 0 {
					hards = append(hards, fmt.Sprintf("未编号：%s（前导 ASCII 数字=章号，中文序号不可排序；重命名后再成册）", c.Path))
					continue
				}
				if prev, dup := seenNo[c.Number]; dup {
					hards = append(hards, fmt.Sprintf("重号：第 %d 章 %q 与 %q", c.Number, prev, c.Path))
				} else {
					seenNo[c.Number] = c.Path
				}
				body, _, err := wenxueReadBody(root, c.Path)
				if err != nil {
					hards = append(hards, fmt.Sprintf("读取失败：%s（%v）", c.Path, err))
					continue
				}
				trimmed := strings.TrimSpace(body)
				if trimmed == "" {
					hards = append(hards, fmt.Sprintf("空章：第 %d 章 %s", c.Number, c.Path))
					continue
				}
				for _, marker := range []string{"TODO", "【待补充】", "【占位", "XXX占位"} {
					if strings.Contains(trimmed, marker) {
						hards = append(hards, fmt.Sprintf("占位残留：第 %d 章含 %q（%s）", c.Number, marker, c.Path))
						break
					}
				}
				if strings.Count(trimmed, "{{") != strings.Count(trimmed, "}}") {
					hards = append(hards, fmt.Sprintf("插值不闭合：第 %d 章 {{ %d 处 vs }} %d 处（%s）", c.Number, strings.Count(trimmed, "{{"), strings.Count(trimmed, "}}"), c.Path))
				}
				if !strings.Contains(trimmed, "\n# ") && !strings.HasPrefix(trimmed, "# ") {
					warns = append(warns, fmt.Sprintf("第 %d 章无一级标题（渲染按首行兜底）：%s", c.Number, c.Path))
				} else {
					// 文件名标题 vs 正文首行标题漂移（warn：重命名敏感，不拦发布）。
					first := ""
					for _, ln := range strings.Split(trimmed, "\n") {
						if strings.HasPrefix(strings.TrimSpace(ln), "# ") {
							first = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(ln), "# "))
							break
						}
					}
					if c.Title != "" && first != "" && c.Title != first {
						warns = append(warns, fmt.Sprintf("第 %d 章文件名标题 %q 与正文标题 %q 不一致（以正文为准渲染）", c.Number, c.Title, first))
					}
				}
			}
			// 跳号（warn：卷结构/楔子番外可能有意跳号，点名缺号不拦）。
			nums := make([]int, 0, len(chapters))
			for _, c := range chapters {
				if c.Number > 0 {
					nums = append(nums, c.Number)
				}
			}
			sort.Ints(nums)
			for i := 1; i < len(nums); i++ {
				if nums[i]-nums[i-1] > 1 {
					warns = append(warns, fmt.Sprintf("章号跳号：%d 之后直接 %d，缺 %d~%d（若为卷结构/番外有意为之可忽略）", nums[i-1], nums[i], nums[i-1]+1, nums[i]-1))
				}
			}
			if len(hards) > 0 {
				return "", fmt.Errorf("机检发现 %d 项硬伤，拒绝组装：%s（修复后重试；warn 项见组装回执）", len(hards), strings.Join(hards, "；"))
			}

			// ── 组装：定稿按章号序拼接（final 已按 Number 排序）─────────────
			var sb strings.Builder
			for i, c := range chapters {
				body, _, err := wenxueReadBody(root, c.Path)
				if err != nil {
					return "", fmt.Errorf("组装中止：第 %d 章读取失败（%s）——定稿在机检后发生漂移，重试", c.Number, c.Path)
				}
				if i > 0 {
					sb.WriteString("\n\n")
				}
				sb.WriteString(strings.TrimSpace(body))
			}

			// ── 配额面：目标 vs 实测（超容差=warn 不拦）────────────────────
			total := 0
			for _, c := range chapters {
				total += c.Words
			}
			quotaNote := ""
			if p.TargetWordsTotal > 0 {
				if p.TolerancePct <= 0 {
					p.TolerancePct = 30
				}
				dev := (total - p.TargetWordsTotal) * 100 / p.TargetWordsTotal
				if dev > p.TolerancePct || dev < -p.TolerancePct {
					quotaNote = fmt.Sprintf("；配额 warn：全书 %d 字 vs 目标 %d（偏差 %d%% 超容差 %d%%）", total, p.TargetWordsTotal, dev, p.TolerancePct)
				} else {
					quotaNote = fmt.Sprintf("；配额在容差内（%d 字 vs 目标 %d，偏差 %d%%）", total, p.TargetWordsTotal, dev)
				}
			}
			type chRow struct {
				No    int    `json:"no"`
				Title string `json:"title"`
				Path  string `json:"path"`
				Words int    `json:"words"`
			}
			rows := make([]chRow, 0, len(chapters))
			for _, c := range chapters {
				rows = append(rows, chRow{No: c.Number, Title: c.Title, Path: c.Path, Words: c.Words})
			}
			rowsJSON, _ := json.Marshal(rows)

			title := strings.TrimSpace(p.Title)
			if title == "" {
				title = filepath.Base(root)
			}

			// ── 交付链：docx+toc 走静态目录两遍法（板间共用核心）────────────
			in := ExportDeliverableInput{
				Markdown:   sb.String(),
				Format:     format,
				OutputPath: p.OutputPath,
				Title:      title,
				Template:   p.Template,
				Cover:      p.Cover,
				TOC:        p.TOC,
				Subtitle:   p.Subtitle,
				Author:     p.Author,
				Header:     p.Header,
				Footer:     p.Footer,
			}
			if in.Template == "" {
				in.Template = "通用"
			}
			tocNote := ""
			if p.TOC && format == "docx" {
				entries := make([]ExportTocEntry, 0, len(chapters))
				for _, c := range chapters {
					entries = append(entries, ExportTocEntry{Text: fmt.Sprintf("第%d章 %s", c.Number, c.Title), Level: 1, Page: "—"})
				}
				in.TocEntries = entries
				measured, merr := measureTocEntries(in, a.WenxueExportDeliverable, wenxueResolveBookPath)
				switch {
				case merr != nil:
					in.TocEntries = nil
					tocNote = fmt.Sprintf("；目录=TOC 域回落（静态页码不可用：%v；打开后 Ctrl+A→F9 刷新页码）", merr)
				default:
					in.TocEntries = measured
					tocNote = fmt.Sprintf("；目录=静态页码（%d 章渲染实测，LibreOffice 口径；Word 打印可能 ±1~2 页）", len(measured))
				}
			}
			res, err := a.WenxueExportDeliverable(in)
			if err != nil {
				return "", err
			}

			// SHA256 Go 自算（与 longdoc_build 同口径：哈希不以渲染链为准）。
			var hash string
			if abs, aerr := wenxueResolveBookPath(res.Path); aerr == nil {
				if data, rerr := os.ReadFile(abs); rerr == nil {
					sum := sha256.Sum256(data)
					hash = hex.EncodeToString(sum[:])
				}
			}
			warnNote := ""
			if len(warns) > 0 {
				warnNote = fmt.Sprintf("；warn %d 项（不拦发布）：%s", len(warns), strings.Join(warns, "；"))
			}
			out := fmt.Sprintf("整本已组装导出：%s（%d 章，%d 字%s%s%s）。\n每章字数 JSON：%s",
				res.Path, len(chapters), total, quotaNote, tocNote, warnNote, rowsJSON)
			if hash != "" {
				out = fmt.Sprintf("整本已组装导出：%s（%d 章，%d 字，sha256=%s%s%s%s）。\n每章字数 JSON：%s",
					res.Path, len(chapters), total, shortHash(hash), quotaNote, tocNote, warnNote, rowsJSON)
			}
			return out, nil
		},
	}
}

// wenxueResolveBookPath 成册链的工作区路径解析（绝对=原样清洗；相对=按图文
// 工作区 ResolveRelWithin）。
func wenxueResolveBookPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("缺少路径")
	}
	if filepath.IsAbs(raw) {
		return filepath.Clean(raw), nil
	}
	p, err := wspath.ResolveRelWithin(wenxueWorkspace(), raw)
	if err != nil {
		return "", fmt.Errorf("非法工作区相对路径: %s", raw)
	}
	return p, nil
}
