package agent

import (
	"context"
	"encoding/json"
	"os"

	fileenc "github.com/gaea/gaea/internal/gaea/fileutil/encoding"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// precheckTool runs a fast deterministic check before a writer tool executes.
// Returns "" when the call is likely to succeed, or a diagnostic message when
// it is predictably going to fail — saving the model an entire API roundtrip.
//
// 缓存安全: 纯运行时判断，不修改任何进入 API 消息数组的内容。
// 阻止调用时返回的消息作为本轮新的 tool_result 追加在消息末尾，
// 不改变已有前缀。
func (a *AgentRunner) precheckTool(name string, args json.RawMessage) string {
	return a.precheckToolFor(nil, name, args)
}

// precheckToolFor 同 precheckTool，带工具实例：读文件要走工具自己的路径解析
// （相对路径 → 工作区内的绝对路径，见 staleGuardPaths）。2026-11-27 AI 反馈 #5：
// 此前按参数原始串读，正式环境里相对路径读不到，预检查整体失效。
func (a *AgentRunner) precheckToolFor(t tool.Tool, name string, args json.RawMessage) string {
	switch name {
	case "edit_file":
		return a.precheckEditFile(t, args)
	}
	return ""
}

// precheckEditFile verifies that old_string exists in the target file before
// letting edit_file run, against the on-disk text (see readFileForPrecheck).
// This catches the single most common agent failure pattern:
// the model sends an old_string that doesn't match the current file content.
//
// 判定口径与 edit_file 工具一致（anchorPresent）：只是空白漂了的锚点工具自己能
// 命中，这里不拦——闸不比工具更严。真找不到才拦，并给出最接近的位置与那里的
// 现行文本（editMissHint），模型照着改锚点直接重试，省一轮 read_file。
func (a *AgentRunner) precheckEditFile(tl tool.Tool, raw json.RawMessage) string {
	var p struct {
		Path      string `json:"path"`
		OldString string `json:"old_string"`
	}
	if err := json.Unmarshal(raw, &p); err != nil || p.Path == "" || p.OldString == "" {
		return "" // can't check — let the real Execute handle it
	}
	readPath := p.Path
	if paths := staleGuardPaths(tl, "edit_file", string(raw)); len(paths) > 0 {
		readPath = paths[0]
	}
	content, ok := a.readFileForPrecheck(readPath)
	if !ok {
		return "" // can't read — let the real Execute report the error
	}
	if anchorPresent(content, p.OldString) {
		return "" // found — let it proceed
	}
	return "precheck blocked: old_string not found in " + p.Path + ".\n" + editMissHint(content, p.OldString)
}

// readFileForPrecheck returns the on-disk text for precheck purposes.
//
// 只读磁盘，不走 toolCache（2026-11-28 黄甲反馈）：缓存槽 (path, 0, 0) 里存的是
// read_file 的**渲染输出**（行号槽「 1→」+ 尾标），不是文件原文。此前预检拿它
// 比对，模型做过一次不带 offset/limit 的整读之后，多行 old_string 必然 not found
// ——子代理改 ch04，锚点确在原文偏移 37 处，连拦 3 次；单行锚点能在带行号的
// 一行里子串命中，所以只在多行锚点上现形。也不再往那个槽里回写原文（会顶掉
// 渲染结果）。解码口径与 edit_file 一致（fileenc.Detect→Decode）：此前按原始
// 字节比，GBK 文件里的中文锚点同样会被误拦。
func (a *AgentRunner) readFileForPrecheck(path string) (string, bool) {
	if a.access != nil {
		if err := a.access.CheckPreview(context.Background(), path); err != nil {
			return "", false
		}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	enc, _ := fileenc.Detect(b)
	return string(fileenc.Decode(b, enc)), true
}
