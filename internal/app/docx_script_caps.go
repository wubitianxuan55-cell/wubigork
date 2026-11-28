package app

// create_docx.py 的扩展参数能力探测（2026-11-28 黄甲反馈）。
//
// 脚本查找（findSkillScript）在工作区没播种 docx 技能时会回落到用户 skills 镜像
// （~/.codex/skills 等），那里的副本可能是旧版：实测黄甲工作区用到的是 2026-08-09
// 的镜像，不认 --toc-spec / --subtitle / --author / --bid-copy / --bid-date /
// --heading-numbering / --template-file 七个参数——封面参数让 longdoc_build 整单
// 失败（exit status 2 + argparse usage），目录两遍法的 pass-1 报错后回落成空 TOC
// 域，回执里只剩一长串 usage（实测一次硬失败、四次降级）。
//
// 做法：静态读脚本源码找 add_argument（不执行 python），不支持的扩展参数不发；
// 缺了哪些记成一句话备注，由调用方写进回执醒目位。基础参数（--spec/--title/
// --cover/--toc/--header/--footer/--font/--template）自初版就有，不探测。

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// docxExtFlag 一个需要探测的扩展参数：缺席时对应的能力用一句人话写进回执。
type docxExtFlag struct {
	flag string
	loss string
}

// docxExtFlags 需要探测的扩展参数，按引入先后排序（备注里按这个顺序列）。
var docxExtFlags = []docxExtFlag{
	{"--subtitle", "封面缺副标题"},
	{"--author", "封面缺编制单位"},
	{"--bid-copy", "封面缺正/副本标记"},
	{"--bid-date", "封面缺落款日期"},
	{"--toc-spec", "目录页码未回填（打开文档后全选再 F9 刷新）"},
	{"--heading-numbering", "标题编号联动未生效"},
	{"--template-file", "登记的格式模板未套用（退回内置模板）"},
}

type docxScriptCaps struct {
	size    int64
	modTime int64
	missing map[string]bool // nil=不裁剪（读不到，或不是 create_docx.py 形态）
}

var (
	docxCapsMu sync.Mutex
	docxCaps   = map[string]docxScriptCaps{}
)

// docxScriptMissing 返回 script 不支持的扩展参数（按引入先后排序）。读不到脚本、
// 或脚本里没有 add_argument 字样（不是 create_docx.py）时返回 nil——不做裁剪，
// 维持旧行为，由真实的 argparse 报错说话。结果按（路径、大小、修改时间）缓存。
func docxScriptMissing(script string) []string {
	if script == "" {
		return nil
	}
	fi, err := os.Stat(script)
	if err != nil || fi.IsDir() || fi.Size() > 4<<20 {
		return nil
	}
	docxCapsMu.Lock()
	rec, hit := docxCaps[script]
	docxCapsMu.Unlock()
	if !hit || rec.size != fi.Size() || rec.modTime != fi.ModTime().UnixNano() {
		rec = scanDocxScript(script, fi)
		docxCapsMu.Lock()
		docxCaps[script] = rec
		docxCapsMu.Unlock()
	}
	if rec.missing == nil {
		return nil
	}
	var out []string
	for _, f := range docxExtFlags {
		if rec.missing[f.flag] {
			out = append(out, f.flag)
		}
	}
	return out
}

// docxScriptSupports 报告 script 是否声明了 flag。
func docxScriptSupports(script, flag string) bool {
	return !containsString(docxScriptMissing(script), flag)
}

func scanDocxScript(script string, fi os.FileInfo) docxScriptCaps {
	rec := docxScriptCaps{size: fi.Size(), modTime: fi.ModTime().UnixNano()}
	raw, err := os.ReadFile(script)
	if err != nil {
		return rec
	}
	src := string(raw)
	if !strings.Contains(src, "add_argument") {
		return rec // 不是 create_docx.py 形态：不裁剪
	}
	rec.missing = map[string]bool{}
	for _, f := range docxExtFlags {
		if !strings.Contains(src, `"`+f.flag+`"`) && !strings.Contains(src, `'`+f.flag+`'`) {
			rec.missing[f.flag] = true
		}
	}
	return rec
}

// docxScriptDegradeNote 把「本次调用想用、但脚本不支持」的参数写成一句回执备注；
// 没有缺席的返回空串。
func docxScriptDegradeNote(script string, skipped []string) string {
	if len(skipped) == 0 {
		return ""
	}
	var losses []string
	for _, f := range docxExtFlags {
		if containsString(skipped, f.flag) {
			losses = append(losses, f.loss)
		}
	}
	return fmt.Sprintf("本机 create_docx.py 是旧版（%s，缺 %s）：%s。根治：把现行 docx 技能装进本工作区（.gaea/skills/docx/scripts/create_docx.py）",
		script, strings.Join(skipped, " "), strings.Join(losses, "；"))
}
