// Package whisper — agent_policy.go
// 100% 对齐 ackem desktop-agent/policy.ts
// Agent 策略控制：操作白名单、路径安全、预算管理

package whisper

import "strings"

// ─── Action Policy ────────────────────────────────────────────

// AllowedActions 允许的操作白名单
var AllowedActions = map[string]bool{
	"read_file":      true,
	"write_file":     true,
	"list_directory": true,
	"search_files":   true,
	"web_search":     true,
	"web_fetch":      true,
	"run_command":    true,
	"move_file":      true,
	"delete_file":    true,
}

// BlockedPaths 禁止访问的路径前缀
var BlockedPaths = []string{
	"C:\\Windows\\System32",
	"C:\\Windows\\SysWOW64",
	"/etc/",
	"/System/",
	"/boot/",
}

// SensitivePaths 敏感路径模式（需要警告）
var SensitivePaths = []string{
	".ssh",
	".gnupg",
	"AppData\\Roaming",
	".config",
	"credentials",
	"password",
	"token",
	"secret",
}

// ─── Policy Check ─────────────────────────────────────────────

// PolicyCheckResult 策略检查结果
type PolicyCheckResult struct {
	Allowed          bool
	HardBlockReason  string
	SensitiveWarning string
	NormalizedPath   string
}

// CheckActionPolicy 检查动作是否合规
func CheckActionPolicy(action AgentAction) PolicyCheckResult {
	result := PolicyCheckResult{Allowed: true}

	// 1. 操作白名单
	if !AllowedActions[action.Name] {
		result.Allowed = false
		result.HardBlockReason = "操作不在白名单中：" + action.Name
		return result
	}

	// 2. 路径安全检查
	if path, ok := action.Args["path"]; ok && path != "" {
		normalized := normalizeAgentPath(path)
		result.NormalizedPath = normalized

		// 禁止系统路径
		for _, blocked := range BlockedPaths {
			if strings.HasPrefix(strings.ToLower(normalized), strings.ToLower(blocked)) {
				result.Allowed = false
				result.HardBlockReason = "禁止访问系统路径：" + normalized
				return result
			}
		}

		// 敏感路径警告
		for _, sensitive := range SensitivePaths {
			if strings.Contains(strings.ToLower(normalized), strings.ToLower(sensitive)) {
				result.SensitiveWarning = "访问包含敏感信息的路径：" + normalized
				break
			}
		}
	}

	// 3. 命令安全检查
	if action.Name == "run_command" {
		if cmd, ok := action.Args["command"]; ok {
			if containsDangerousCommand(cmd) {
				result.Allowed = false
				result.HardBlockReason = "禁止执行危险命令"
				return result
			}
		}
	}

	return result
}

func containsDangerousCommand(cmd string) bool {
	dangerous := []string{
		"rm -rf /", "del /f /s", "format", "shutdown",
		"chmod 777 /", "> /dev/sda", "mkfs",
	}
	lower := strings.ToLower(cmd)
	for _, d := range dangerous {
		if strings.Contains(lower, d) {
			return true
		}
	}
	return false
}

func normalizeAgentPath(path string) string {
	// 简单规范化
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.ReplaceAll(path, "//", "/")
	return strings.TrimSpace(path)
}

// ─── Attention Budget ─────────────────────────────────────────

// AttentionBudget 注意力预算
type AttentionBudget struct {
	MaxActionsPerTurn int     `json:"maxActionsPerTurn"`
	MaxTurnsPerTask   int     `json:"maxTurnsPerTask"`
	UsedActions       int     `json:"usedActions"`
	UsedTurns         int     `json:"usedTurns"`
}

// NewAttentionBudget 创建预算
func NewAttentionBudget() *AttentionBudget {
	return &AttentionBudget{
		MaxActionsPerTurn: 5,
		MaxTurnsPerTask:   10,
	}
}

// CanAct 是否还能执行操作
func (ab *AttentionBudget) CanAct() bool {
	return ab.UsedActions < ab.MaxActionsPerTurn && ab.UsedTurns < ab.MaxTurnsPerTask
}

// Spend 消耗预算
func (ab *AttentionBudget) Spend(actions int) {
	ab.UsedActions += actions
	ab.UsedTurns++
}

// Reset 重置预算
func (ab *AttentionBudget) Reset() {
	ab.UsedActions = 0
	ab.UsedTurns = 0
}

// BudgetExceeded 预算是否超支
func (ab *AttentionBudget) BudgetExceeded() bool {
	return ab.UsedTurns >= ab.MaxTurnsPerTask
}
