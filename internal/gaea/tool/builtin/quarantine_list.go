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
			// 现状按盘面核（2026-11-26 造价反馈，百锦路 #1）：账本只追加、没有
			// 「已取回」态，文件被 move 走后批次仍显示 quarantined——实测模型
			// 照账本去取，source not found。每件如实报还在不在隔离区。
			present := "否（已取回或已不在）"
			if _, serr := os.Stat(filepath.Join(root, r.Stamp, filepath.FromSlash(e.RelPath))); serr == nil {
				present = "是"
			}
			fmt.Fprintf(&sb, "    %s（%s｜在隔离区：%s）\n", e.RelPath, e.Kind, present)
		}
		// 落点给到件：隔离目录下镜像原相对路径，批次目录本身不是文件所在处。
		fmt.Fprintf(&sb, "    落点：%s（每件完整路径 = 落点/原相对路径；工作区相对写法 .gaea/publish-quarantine/%s/…）\n",
			filepath.Join(root, r.Stamp), r.Stamp)
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
		"\n取回：隔离件是原件。在工作区内取回——move_file 终点须在写根内（移到工作区外会被拦）：" +
		"move_file source=<落点>/<原相对路径> destination=.gaea/work/<文件名>（过程目录，不受发布边界限制）；复核、改完后再 move_file 到正式名。", nil
}
