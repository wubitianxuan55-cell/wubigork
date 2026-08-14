// SessionState 是会话的中断状态 sidecar：进程存活期间 running=true，
// 回合正常结束后写回 running=false；若进程在回合中被杀/崩溃，running=true
// 残留在磁盘上，重启后即可在会话列表识别「上次未完成」的会话，并在恢复时
// 注入中断摘要。
package session

import (
	"encoding/json"
	"os"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

// SessionState 记录一个会话的运行标记与最后摘要。
type SessionState struct {
	Running   bool   `json:"running"`
	Summary   string `json:"summary,omitempty"`
	UpdatedAt int64  `json:"updatedAt"`
}

// StatePath 返回会话文件对应的状态 sidecar 路径（同目录 <base>.state.json）。
// 空路径返回空串，调用方据此跳过状态读写。
func StatePath(sessionPath string) string {
	if sessionPath == "" {
		return ""
	}
	return sessionPath + ".state.json"
}

// LoadState 读取会话状态。文件不存在时返回零值 state 且无错误，调用方把
// 「无状态」当作「未中断」处理；损坏的 JSON 同样按零值处理（防御：状态文件
// 只是辅助标记，不应阻塞会话列表或恢复）。
func LoadState(path string) (SessionState, error) {
	var st SessionState
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return st, nil
		}
		return st, err
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return SessionState{}, nil
	}
	return st, nil
}

// SaveState 原子写入会话状态（写临时文件再 rename，崩溃不残留半截文件）。
func SaveState(path string, st SessionState) error {
	if path == "" {
		return nil
	}
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return fileutil.AtomicWrite(path, b, 0o644)
}

// ClearState 删除会话状态文件；文件不存在时静默成功。
func ClearState(path string) error {
	if path == "" {
		return nil
	}
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
