package app

// gaea_schedule_file.go — 进度计划板块文件持久化绑定（v4.113.0 刀4）。
//
// 计划文件（进度计划/当前计划.gsched.json）是板块与 agent 的共享资产：
// 板块经本绑定读写（轻量 IO，不落证据卡——UI 自动保存不刷 Journal）；
// agent 经 schedule_get/apply/analyze 三工具读写（快照+证据卡在工具内）。
// 两侧校验/CPM 口径同源 internal/schedule，落盘一律 fail-closed。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gaea/gaea/internal/schedule"
)

// ScheduleLoadResult 板块装载结果：Project 为计划 JSON 串（前端再解析，
// 避免绑定面引入 schedule 结构体的 wails model 生成）。
type ScheduleLoadResult struct {
	Path    string `json:"path"`    // 工作区相对路径
	Exists  bool   `json:"exists"`  // false=尚无计划文件（前端走迁移/空态）
	Project string `json:"project"` // 计划 JSON（存在时）
}

// ScheduleSaveResult 保存回执。
type ScheduleSaveResult struct {
	Path     string `json:"path"`
	SavedAt  string `json:"savedAt"`  // ISO 时间（前端「已保存 HH:MM」显示）
	Duration int    `json:"duration"` // 保存后总工期（工作日），前端校对用
	Critical int    `json:"critical"` // 关键工作数
}

// GaeaScheduleLoad 装载默认计划文件。文件不存在返回 Exists=false（不视为错误）。
func (a *App) GaeaScheduleLoad() (ScheduleLoadResult, error) {
	rel := schedule.DefaultRelPath
	path := filepath.Join(gaeaCwd(), rel)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ScheduleLoadResult{Path: rel, Exists: false}, nil
		}
		return ScheduleLoadResult{}, fmt.Errorf("读取计划文件失败：%w", err)
	}
	// 落盘前过引擎校验（坏文件不在板块里静默炸，给出可读错误）
	if _, err := schedule.Load(path); err != nil {
		return ScheduleLoadResult{}, err
	}
	return ScheduleLoadResult{Path: rel, Exists: true, Project: string(raw)}, nil
}

// GaeaScheduleSave 保存板块计划（整量覆盖）：校验+CPM fail-closed+原子写。
func (a *App) GaeaScheduleSave(projectJSON string) (ScheduleSaveResult, error) {
	if projectJSON == "" {
		return ScheduleSaveResult{}, fmt.Errorf("计划内容为空")
	}
	var p schedule.Project
	if err := json.Unmarshal([]byte(projectJSON), &p); err != nil {
		return ScheduleSaveResult{}, fmt.Errorf("计划 JSON 解析失败：%w", err)
	}
	rel := schedule.DefaultRelPath
	path := filepath.Join(gaeaCwd(), rel)
	if err := schedule.Save(path, p); err != nil {
		return ScheduleSaveResult{}, err
	}
	cpm, a2 := p.Analyze()
	dur, crit := 0, 0
	if cpm.OK {
		dur, crit = cpm.Duration, len(a2.Critical)
	}
	return ScheduleSaveResult{
		Path:     rel,
		SavedAt:  time.Now().Format("15:04"),
		Duration: dur,
		Critical: crit,
	}, nil
}
