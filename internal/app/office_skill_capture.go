package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/skill"
)

// SkillCaptureInput 是一次成功对话沉淀为技能的输入。
type SkillCaptureInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Task        string `json:"task"`
	Solution    string `json:"solution"`
}

// SkillCaptureResult 是沉淀结果。
type SkillCaptureResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Reloaded    bool   `json:"reloaded"`
	Tools       int    `json:"tools"`
	Skills      int    `json:"skills"`
}

const (
	captureTaskMax     = 2000
	captureSolutionMax = 8000
)

// GaeaCaptureSkill 把一次成功的办公对话沉淀为可复用技能：写入工作区
// .gaea/skills/<name>/SKILL.md 并镜像 ~/.codex/skills/<name>/SKILL.md（供
// Codex 与 gaea 通用办公共用），随后热加载办公引擎使技能立即出现在技能
// 索引与能力抽屉。同名技能允许覆盖（用户再次沉淀的是改进版流程）。
func (a *App) GaeaCaptureSkill(in SkillCaptureInput) (SkillCaptureResult, error) {
	name := strings.TrimSpace(in.Name)
	task := strings.TrimSpace(in.Task)
	solution := strings.TrimSpace(in.Solution)
	if !skill.IsValidName(name) {
		return SkillCaptureResult{}, fmt.Errorf("技能名不合法：%q（使用字母/数字/_/-/.，1-64 字符，字母开头）", name)
	}
	if task == "" {
		return SkillCaptureResult{}, fmt.Errorf("缺少任务描述——请保留一次任务输入")
	}
	if solution == "" {
		return SkillCaptureResult{}, fmt.Errorf("缺少解决方案——请保留助手回答")
	}
	if r := []rune(task); len(r) > captureTaskMax {
		task = string(r[:captureTaskMax])
	}
	if r := []rune(solution); len(r) > captureSolutionMax {
		solution = string(r[:captureSolutionMax])
	}
	desc := strings.TrimSpace(strings.Join(strings.Fields(in.Description), " "))
	if desc == "" {
		r := []rune(task)
		if len(r) > 60 {
			r = r[:60]
		}
		desc = string(r)
	}
	if r := []rune(desc); len(r) > 120 {
		desc = string(r[:120])
	}

	body := fmt.Sprintf(`# %s 技能

## 适用场景
%s

## 操作步骤
%s

## 调用方式
- 对话中直接描述任务即可命中本技能，也可用 /%s 显式调用
- 按操作步骤执行，完成后按场景要求验证产出`, name, task, solution, name)

	content := skill.RenderSkillFile(name, desc, body)

	// 1) 工作区 .gaea/skills（gaea 项目作用域，优先加载）
	root := filepath.Join(gaeaCwd(), ".gaea", "skills")
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return SkillCaptureResult{}, fmt.Errorf("创建技能目录失败: %w", err)
	}
	path := filepath.Join(dir, skill.SkillFile)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return SkillCaptureResult{}, fmt.Errorf("写入技能失败: %w", err)
	}

	// 2) 镜像到 Codex 全局技能目录（~/.codex/skills），跨工具复用
	if home, err := os.UserHomeDir(); err == nil {
		codexDir := filepath.Join(home, ".codex", "skills", name)
		if err := os.MkdirAll(codexDir, 0o755); err == nil {
			_ = os.WriteFile(filepath.Join(codexDir, skill.SkillFile), []byte(content), 0o644)
		}
	}

	res := SkillCaptureResult{Name: name, Description: desc, Path: path}
	// 3) 热加载办公引擎（仅当引擎已初始化）：技能立刻进入索引与能力抽屉
	ga.mu.Lock()
	initialized := ga.ctrl != nil
	ga.mu.Unlock()
	if initialized {
		if r, err := a.GaeaReload(); err == nil {
			res.Reloaded = true
			res.Tools = r.Tools
			res.Skills = r.Skills
		}
	}
	return res, nil
}
