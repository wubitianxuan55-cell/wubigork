// Package builtin provides Tianxuan's compile-time built-in tools. Each tool
// self-registers via init(); main blank-imports this package to wire them in.
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(quarantineList{}) }

// quarantineList 隔离账本查询（2026-11-21 反馈 P1-12 建议④：此前只能手工解析
// ledger.jsonl，且按行数统计会翻倍）。只读，读工作区 .gaea/publish-quarantine/
// ledger.jsonl，按 stamp 去重（一行=一次状态变更，同 stamp 取最后一条），批次
// 新优先；带 status 过滤。路径口径=零值实例按绝对/进程 cwd，confined 实例按
// 工作区（workspace.go Tools 列表装配）。
type quarantineList struct{ workDir string }

func (quarantineList) Name() string { return "quarantine_list" }

func (quarantineList) Description() string {
	return "列出发布边界隔离账本（只读）：按批次（stamp）去重汇总——状态（pending=终态没落上悬挂/quarantined=已隔离/partial=部分失败/swept=已清扫）、来源（foreground/background）、命令预览、被隔离文件清单与落点。找回被隔离交付物、审计拦截频次用它；注意账本一行=一次状态变更，本工具已按 stamp 去重勿再按行数计。"
}

func (quarantineList) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "status":{"type":"string","enum":["pending","quarantined","partial","swept","sweep_failed"],"description":"可选，按终态过滤"}
},
"required":[]
}`)
}

func (quarantineList) ReadOnly() bool { return true }

// SpaceTag 自声明办公域（与静态表同口径；自声明优先生效标签）。
func (quarantineList) SpaceTag() string { return spaces.SpaceWork }

func (t quarantineList) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(args, &p)
	root := filepath.Join(t.workDir, ".gaea", "publish-quarantine")
	raw, err := os.ReadFile(filepath.Join(root, "ledger.jsonl"))
	if err != nil {
		if os.IsNotExist(err) {
			return "(no quarantine records — 发布边界尚未隔离过任何文件)", nil
		}
		return "", fmt.Errorf("读取隔离账本失败: %w", err)
	}
	type rec struct {
		Stamp   string    `json:"stamp"`
		Event   string    `json:"event"`
		Status  string    `json:"status"`
		Time    time.Time `json:"time"`
		Source  string    `json:"source"`
		Command string    `json:"command"`
		Entries []struct {
			RelPath string `json:"rel"`
			Kind    string `json:"kind"`
		} `json:"entries"`
	}
	// 同 stamp 取最后一条（状态机：begin→final→swept）。
	latest := map[string]rec{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var r rec
		if json.Unmarshal([]byte(line), &r) == nil && r.Stamp != "" {
			latest[r.Stamp] = r
		}
	}
	stamps := make([]string, 0, len(latest))
	for s := range latest {
		stamps = append(stamps, s)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(stamps))) // stamp=时间戳，新优先
	var sb strings.Builder
	total := 0
	for _, s := range stamps {
		r := latest[s]
		if p.Status != "" && r.Status != p.Status {
			continue
		}
		total++
		fmt.Fprintf(&sb, "- %s [%s] %s", r.Stamp, r.Status, r.Source)
		if !r.Time.IsZero() {
			fmt.Fprintf(&sb, " %s", time.Since(r.Time).Truncate(time.Second))
		}
		sb.WriteString("\n")
		if r.Command != "" {
			fmt.Fprintf(&sb, "    cmd: %s\n", r.Command)
		}
		for i, e := range r.Entries {
			if i >= 8 {
				fmt.Fprintf(&sb, "    …等 %d 件\n", len(r.Entries)-i)
				break
			}
			fmt.Fprintf(&sb, "    %s（%s）\n", e.RelPath, e.Kind)
		}
		fmt.Fprintf(&sb, "    落点：%s\n", filepath.Join(root, r.Stamp))
	}
	if total == 0 {
		if p.Status != "" {
			return fmt.Sprintf("(无状态为 %s 的批次)", p.Status), nil
		}
		return "(no quarantine records)", nil
	}
	var head string
	if p.Status != "" {
		head = fmt.Sprintf("隔离账本（status=%s）共 %d 批次（按 stamp 去重，新优先）：\n", p.Status, total)
	} else {
		head = fmt.Sprintf("隔离账本共 %d 批次（按 stamp 去重，新优先）：\n", total)
	}
	return head + strings.TrimRight(sb.String(), "\n") +
		"\n取回：隔离件为原件，move_file 可移到工作区外暂存位复核；durable 正式落盘走结构化入口或 Temp→move_file。", nil
}
