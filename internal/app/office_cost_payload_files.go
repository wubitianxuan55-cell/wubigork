package app

// office_cost_payload_files.go — 批量写库的载荷走文件（2026-11-27 AI 反馈：行级
// 载荷只能经工具参数手写转录——112 条手写清单里出了 6 个幻影码，266 行资源切
// 14 批）。
//
// import_add 的 resources_file / quotas_file / bills_file、classify 的 classify_file：
// 行由模型（或模型写的脚本，直接从资料生成）落成工作区里的 JSON 文件，程序读
// 文件。数字不再经模型转录；行数也不再卡「单次调用整理得动的量」。文件里的行
// 与直接给数组时字段逐字相同，退回、覆盖、部分修正、回读核对全走同一条链。
// 数组与文件同时给时以数组为准（文件只补没给的那一组）。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/workcost"
	"github.com/gaea/gaea/internal/gaea/wspath"
)

const (
	// clPayloadFileMaxBytes 单个载荷文件的体积上限（几千行 JSON 远用不到）。
	clPayloadFileMaxBytes = 8 << 20
	// clFileMaxRows 载荷文件单组上限。workcost.ImportMaxPerBatch（200）卡的是模型
	// 一次手写转录的量；文件不经转录，放宽到 2000。
	clFileMaxRows = 2000
)

// loadPayloadFiles 把 *_file 指向的行读进 p（只补没给数组的组）。import_add 的
// 三组行是数组，classify 是对象，分两段处理。
func (costLibraryTool) loadPayloadFiles(p *costLibraryArgs) error {
	switch p.Action {
	case "import_add":
		for _, g := range []struct {
			kind, key, label, file string
			empty                  bool
			load                   func([]json.RawMessage) error
		}{
			{"resource", "resources", clImportKindLabel["resource"], p.ResourcesFile, len(p.Resources) == 0, func(rows []json.RawMessage) error { return clDecodeRows(rows, &p.Resources) }},
			{"quota", "quotas", clImportKindLabel["quota"], p.QuotasFile, len(p.Quotas) == 0, func(rows []json.RawMessage) error { return clDecodeRows(rows, &p.Quotas) }},
			{"bill", "bills", clImportKindLabel["bill"], p.BillsFile, len(p.Bills) == 0, func(rows []json.RawMessage) error { return clDecodeRows(rows, &p.Bills) }},
		} {
			if strings.TrimSpace(g.file) == "" || !g.empty {
				continue
			}
			rows, shown, err := clReadPayloadArray(g.file, g.key, g.label)
			if err != nil {
				return err
			}
			if err := g.load(rows); err != nil {
				return fmt.Errorf("%s载荷文件 %s：%w", g.label, shown, err)
			}
			clMarkFromFile(p, g.kind)
			p.FileNote += fmt.Sprintf("%s %d 行读自 %s；", g.label, len(rows), shown)
		}
	case "classify":
		if strings.TrimSpace(p.ClassifyFile) == "" || p.Classify != nil {
			return nil
		}
		rows, shown, err := clReadPayloadObject(p.ClassifyFile)
		if err != nil {
			return err
		}
		var plan workcost.ClassifyPlan
		group := func(key string, load func([]json.RawMessage) error) error {
			raw, ok := rows[key]
			if !ok || string(raw) == "null" {
				return nil
			}
			var elems []json.RawMessage
			if err := json.Unmarshal(raw, &elems); err != nil {
				return fmt.Errorf("classify 载荷文件 %s 的 %s 不是条目数组", shown, key)
			}
			if err := load(elems); err != nil {
				return fmt.Errorf("classify 载荷文件 %s 的 %s：%w", shown, key, err)
			}
			return nil
		}
		for _, g := range []struct {
			key  string
			load func([]json.RawMessage) error
		}{
			{"projects", func(rows []json.RawMessage) error { return clDecodeRows(rows, &plan.Projects) }},
			{"bills", func(rows []json.RawMessage) error { return clDecodeRows(rows, &plan.Bills) }},
			{"quotas", func(rows []json.RawMessage) error { return clDecodeRows(rows, &plan.Quotas) }},
			{"resources", func(rows []json.RawMessage) error { return clDecodeRows(rows, &plan.Resources) }},
		} {
			if err := group(g.key, g.load); err != nil {
				return err
			}
		}
		p.Classify = &plan
		clMarkFromFile(p, "classify")
		p.FileNote = fmt.Sprintf("classify 读自 %s；", shown)
	}
	return nil
}

// clMarkFromFile 记下哪一组是文件读来的（单批上限放宽到 clFileMaxRows）。
func clMarkFromFile(p *costLibraryArgs, kind string) {
	if p.FromFile == nil {
		p.FromFile = map[string]bool{}
	}
	p.FromFile[kind] = true
}

// clReadPayloadFile 读载荷文件（工作区内，拒 .. 穿越与工作区外），返回去 BOM 的
// 内容与给模型看的相对路径。
func clReadPayloadFile(rel, label string) ([]byte, string, error) {
	rel = strings.TrimSpace(rel)
	abs, err := wspath.ResolveRelWithin(gaeaCwd(), rel)
	if err != nil {
		return nil, "", fmt.Errorf("载荷文件 %s 必须在工作区内（拒 .. 穿越）", rel)
	}
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		return nil, "", fmt.Errorf("读不到%s载荷文件 %s（不存在，或在工作区之外）——先把行写成 JSON 文件（write_file，建议落 .gaea/work/）", label, rel)
	}
	if info.Size() > clPayloadFileMaxBytes {
		return nil, "", fmt.Errorf("%s载荷文件 %s 有 %.1f MB，超过上限 %d MB——按组拆成几个文件", label, rel,
			float64(info.Size())/(1<<20), clPayloadFileMaxBytes>>20)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, "", fmt.Errorf("读%s载荷文件 %s 失败：%w", label, rel, err)
	}
	return bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}), filepath.ToSlash(rel), nil
}

// clReadPayloadArray 读一个行载荷文件，拆成行（保持文件顺序）。文件可以是 JSON
// 数组 [{…},{…}]，或套了一层键的对象 {"quotas":[…]}（同一个文件里放几组时）。
func clReadPayloadArray(rel, key, label string) ([]json.RawMessage, string, error) {
	data, shown, err := clReadPayloadFile(rel, label)
	if err != nil {
		return nil, "", err
	}
	trimmed := bytes.TrimSpace(data)
	var rows []json.RawMessage
	if len(trimmed) > 0 && trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &rows); err != nil {
			return nil, "", fmt.Errorf("%s载荷文件 %s 的行数组不合法：%v", label, shown, err)
		}
	} else {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(trimmed, &obj); err != nil {
			return nil, "", fmt.Errorf("%s载荷文件 %s 应是行数组 [{…},{…}] 或 {\"%s\":[…]}，不是合法 JSON：%v", label, shown, key, err)
		}
		raw, ok := obj[key]
		if !ok {
			keys := make([]string, 0, len(obj))
			for k := range obj {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			return nil, "", fmt.Errorf("%s载荷文件 %s 是对象但没有 %s 键（文件里的键：%s）", label, shown, key, strings.Join(keys, ", "))
		}
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, "", fmt.Errorf("%s载荷文件 %s 的 %s 不是行数组", label, shown, key)
		}
	}
	for i, raw := range rows {
		t := bytes.TrimSpace(raw)
		if len(t) == 0 || t[0] != '{' {
			return nil, "", fmt.Errorf("%s载荷文件 %s 第 %d 项不是行对象：%s", label, shown, i+1, clEchoRaw(raw))
		}
	}
	if len(rows) == 0 {
		return nil, "", fmt.Errorf("%s载荷文件 %s 里没有行", label, shown)
	}
	if len(rows) > clFileMaxRows {
		return nil, "", fmt.Errorf("%s载荷文件 %s 有 %d 行，超过单文件上限 %d 行——按组拆成几个文件", label, shown, len(rows), clFileMaxRows)
	}
	return rows, shown, nil
}

// clReadPayloadObject 读一个对象载荷文件，拆成顶层键 → 原始值。
func clReadPayloadObject(rel string) (map[string]json.RawMessage, string, error) {
	data, shown, err := clReadPayloadFile(rel, "classify ")
	if err != nil {
		return nil, "", err
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(bytes.TrimSpace(data), &obj); err != nil {
		return nil, "", fmt.Errorf("classify 载荷文件 %s 不是合法的 JSON 对象：%v", shown, err)
	}
	return obj, shown, nil
}

// clDecodeRows 逐行解码，报错带文件里的行号（整批解码时 encoding/json 不说第几行）。
func clDecodeRows[T any](rows []json.RawMessage, dst *[]T) error {
	out := make([]T, 0, len(rows))
	for i, raw := range rows {
		var v T
		if err := json.Unmarshal(raw, &v); err != nil {
			return fmt.Errorf("第 %d 行：%w", i+1, err)
		}
		out = append(out, v)
	}
	*dst = out
	return nil
}

// clEchoRaw 报错时回显的原文（截断，防把整份载荷打进回执）。
func clEchoRaw(raw []byte) string {
	s := strings.TrimSpace(string(raw))
	if r := []rune(s); len(r) > 80 {
		s = string(r[:80]) + "…"
	}
	return s
}
