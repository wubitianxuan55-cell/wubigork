package proc

import (
	"os"
	"strings"
)

// 工具子进程的确定性环境（codex 蒸馏 T2）：模型可见的工具输出不应随用户终端
// 品味漂移——颜色码、分页器、本地化都会让「同一命令」产出不可比对的结果，
// 长输出还可能被 pager 卡死。注入固定值覆盖同名父变量即可根治。
//
// 转义阀门：GAEA_EXEC_ENV_RAW=1 跳过全部注入（排查「注入是否背锅」用），
// 行为与注入前的历史版本一致。

// deterministicVars 是注入给工具子进程的固定变量（顺序固定，便于比对）。
// COLORTERM 必须是空值而不是删除：下游有「存在且为空」的判定，删除会走另一条路。
// 末位 GAEA_CI=1 是可识别环境标记（机制同构 codex 的 CI 标记，名字用 gaea 自己的），
// 供自家脚本与诊断探针识别「跑在受控环境」。
var deterministicVars = []string{
	"NO_COLOR=1",
	"TERM=dumb",
	"LANG=C.UTF-8",
	"LC_CTYPE=C.UTF-8",
	"LC_ALL=C.UTF-8",
	"COLORTERM=",
	"PAGER=cat",
	"GIT_PAGER=cat",
	"GH_PAGER=cat",
	"GAEA_CI=1",
}

// rawEnvRawKey 是转义阀门开关。
const rawEnvRawKey = "GAEA_EXEC_ENV_RAW"

// DeterministicEnv 返回固定注入变量副本（不含父环境）。
func DeterministicEnv() []string {
	out := make([]string, len(deterministicVars))
	copy(out, deterministicVars)
	return out
}

// CommandEnv 组装工具子进程环境：父环境打底 → 覆盖固定变量 → 叠加 extra。
// extra（如 bash 状态锚定的探针变量）后到者优先，同名父变量一并让位避免重复项。
// 这是唯一合并点：bash 前台/后台两条路都走这里，不许各写一份。
func CommandEnv(extra ...string) []string {
	parent := os.Environ()
	if os.Getenv(rawEnvRawKey) == "1" {
		// 阀门开：与历史版本逐字节一致（含 extra，保持探针链路可用）。
		return append(parent, extra...)
	}
	// 待剔除集合 = 固定变量 ∪ extra 的键。
	drop := make(map[string]struct{}, len(deterministicVars)+len(extra))
	for _, kv := range deterministicVars {
		drop[envKey(kv)] = struct{}{}
	}
	for _, kv := range extra {
		drop[envKey(kv)] = struct{}{}
	}
	out := make([]string, 0, len(parent)+len(deterministicVars)+len(extra))
	for _, kv := range parent {
		if _, hit := drop[envKey(kv)]; !hit {
			out = append(out, kv)
		}
	}
	out = append(out, deterministicVars...)
	out = append(out, extra...)
	return out
}

// envKey 取 "K=V" 的键；无等号的整串当键（异常形态按整体剔除，保守）。
func envKey(kv string) string {
	if i := strings.IndexByte(kv, '='); i >= 0 {
		return kv[:i]
	}
	return kv
}
