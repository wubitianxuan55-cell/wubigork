package builtin

// sidebar_open.go — v4.25「模型主动打开」（docs/gaea-office-upgrade-plan-2026-09.md
// A3，对标 dsh-better-sidebar 的 sidebar_open）：模型可主动把关键产物文件/目录
// 推到桌面端右面板打开，用户不用自己在文件树里找。
//
// 语义是纯 UI 动作：只校验路径落在当前工作区内并返回定位信息，**不读内容
// 不落盘**。因此 ReadOnly()=true（permission.Policy.Decide 对只读工具直接
// Allow，不走写类权限弹卡），口径与 browser_read/browser_snapshot 一致。
//
// 注册走 browser 先例：init() 注册零值实例（root 空 = 工作区未设置，Execute
// 返回结构化报错）；桌面端装配时 boot.addBuiltins → Workspace.Tools() 里绑定
// 工作区根的同名实例按名替换零值（与 read_file 等基础工具同一机制）。
// 结果统一走 envelope 结构化返回，data 形如：
//
//	{"kind":"file|directory","path_abs":"...","path_rel":"<相对工作区根>","message":"..."}

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(sidebarOpen{}) }

// sidebarOpen 把工作区内的文件/目录推到右面板打开。root 是工作区根（桌面端
// 装配时绑定为 Workspace.Dir）；零值实例（init 注册、CLI 无工作区装配）root
// 为空，Execute 直接返回结构化报错——右面板只在桌面端存在，无工作区时该
// 工具不可用。
type sidebarOpen struct{ root string }

func (sidebarOpen) Name() string { return "sidebar_open" }

func (sidebarOpen) Description() string {
	return "把工作区内的文件或目录推送到右面板打开（纯 UI 动作：不在右面板打开关键产物时主动调用，用户无需自己找文件；不读取内容、不修改文件）。path 传工作区内相对路径或绝对路径；kind 可选（file/directory），缺省按 path 实际类型推断。目录以树视图打开，文件进入预览/编辑器。"
}

func (sidebarOpen) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"要打开的工作区内路径（相对工作区根或绝对路径；目录以树视图打开，文件进入预览/编辑器）"},
  "kind":{"type":"string","enum":["file","directory"],"description":"可选：file 或 directory；缺省按 path 实际类型推断"}
},
"required":["path"]
}`)
}

// 只读档：纯 UI 动作，无宿主侧副作用 → 权限门直接 Allow（不弹写类审批卡），
// 且可与同批只读工具并行。
func (sidebarOpen) ReadOnly() bool                 { return true }
func (sidebarOpen) SpaceTag() string               { return spaces.SpaceWork }
func (sidebarOpen) CompactDescription() string     { return compactDesc["sidebar_open"] }
func (sidebarOpen) CompactSchema() json.RawMessage { return compactSchema["sidebar_open"] }

func (s sidebarOpen) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path string `json:"path"`
		Kind string `json:"kind"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &p); err != nil {
			return tool.WrapError(tool.CodeValidationError, "invalid args: "+err.Error(), nil), nil
		}
	}
	p.Path = strings.TrimSpace(p.Path)
	if p.Path == "" {
		return tool.WrapError(tool.CodeValidationError, "path 必填（工作区内相对路径或绝对路径）", nil), nil
	}
	switch p.Kind {
	case "", "file", "directory":
	default:
		return tool.WrapError(tool.CodeValidationError,
			fmt.Sprintf("kind 非法 %q（只接受 file / directory，或缺省按 path 推断）", p.Kind),
			map[string]any{"path": p.Path}), nil
	}
	if strings.TrimSpace(s.root) == "" {
		return tool.WrapError("no_workspace",
			"工作区未设置：sidebar_open 需要当前工作区（桌面端会话绑定工作区后可用）",
			map[string]any{"path": p.Path}), nil
	}
	rootAbs, err := realPath(s.root)
	if err != nil {
		return tool.WrapError(tool.CodeExecError, "resolve workspace root: "+err.Error(),
			map[string]any{"path": p.Path}), nil
	}
	// 防穿越：相对路径锚定工作区根，绝对路径原样；随后统一做 realPath（解析
	// 已存在最深祖先的符号链接）+ within 判定，与文件写类工具同款守门
	// （confine.go realPath/within）。路径必须落在工作区内。
	abs, err := realPath(resolveIn(rootAbs, p.Path))
	if err != nil {
		return tool.WrapError(tool.CodeExecError, "resolve path: "+err.Error(),
			map[string]any{"path": p.Path}), nil
	}
	if !within(rootAbs, abs) {
		return tool.WrapError(tool.CodeValidationError,
			fmt.Sprintf("path %q 落在工作区外（sidebar_open 只能打开工作区内路径；工作区根：%s）", p.Path, rootAbs),
			map[string]any{"path": p.Path}), nil
	}
	info, err := os.Stat(abs)
	if err != nil {
		return tool.WrapError(tool.CodeNotFound,
			fmt.Sprintf("path %q 不存在或不可访问（sidebar_open 只能打开已存在的文件/目录）", p.Path),
			map[string]any{"path": p.Path}), nil
	}
	// kind 缺省推断：存在且是目录 → directory，否则 file；显式 kind 与实际
	// 类型不符时报错并告知实际类型（data.kind 恒与真实目标一致）。
	kind := "file"
	if info.IsDir() {
		kind = "directory"
	}
	if p.Kind != "" && p.Kind != kind {
		return tool.WrapError(tool.CodeValidationError,
			fmt.Sprintf("kind=%s 与实际类型不符：%s 是 %s", p.Kind, p.Path, kind),
			map[string]any{"path": p.Path, "actual_kind": kind}), nil
	}
	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil {
		// within 已通过，Rel 理论上不会再失败；兜底回退绝对路径。
		rel = abs
	}
	return tool.WrapResult("ok", map[string]any{
		"kind":     kind,
		"path_abs": abs,
		"path_rel": rel,
		"message":  "已在右面板打开 " + rel,
	}), nil
}
